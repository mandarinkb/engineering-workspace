// Package auth ให้กลไก JWT (JSON Web Token) แบบ HS256 (HMAC-SHA256) ขั้นต่ำสุดที่ BOP
// ต้องใช้สำหรับ roadmap Phase 1.8 (login + protected route ของ dashboard) — implement
// เอง (ไม่พึ่ง library ภายนอกอย่าง golang-jwt/jwt) เพราะ JWT ที่ต้องการตรงนี้เรียบง่ายมาก
// (มีแค่ signing algorithm เดียว, claim ไม่กี่ตัว, ไม่ต้องรองรับ RSA/algorithm อื่น) เขียน
// เองแค่ไม่ถึง 100 บรรทัดคุ้มกว่าเพิ่ม dependency ใหม่อีกตัว — ต่างจาก go.temporal.io/sdk
// ที่ reinvent เองไม่คุ้มเพราะ durable execution ซับซ้อนมาก (ดู
// projects/bot-orchestration-platform/TEMPORAL-MIGRATION-TODO.md ข้อ 2) และยังเป็นโอกาส
// เรียนรู้กลไกจริงของ JWT ไปในตัว (ตรงกับเป้าหมายการเรียนรู้ของ repo นี้)
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// header คือส่วนหัวของ JWT บอกว่าใช้ signing algorithm อะไร — ตาม JWT spec (RFC 7519)
// ต้องมี field "alg" กับ "typ" เสมอ แม้ระบบนี้จะรองรับแค่ HS256 อย่างเดียวก็ตาม
type header struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

// Claims คือข้อมูลที่เข้ารหัสอยู่ใน JWT — เก็บแค่เท่าที่ระบบนี้ต้องใช้จริงตอนนี้ (ยังไม่มี
// user database จริงจัง Subject จึงเป็นแค่ username ธรรมดา ไม่ใช่ user ID จาก DB — ดู
// handleLogin ที่ internal/api/http/handler.go)
type Claims struct {
	Subject   string `json:"sub"` // ใครเป็นเจ้าของ token นี้
	ExpiresAt int64  `json:"exp"` // เวลาหมดอายุ เป็น Unix timestamp (วินาที) ตาม JWT spec
}

// Issue สร้าง JWT ใหม่แบบ HS256 ให้ subject ที่ระบุ หมดอายุใน ttl จากนี้ไป
//
// โครงสร้าง JWT คือ 3 ส่วนคั่นด้วย "." เสมอ:
//
//	base64url(header) + "." + base64url(payload) + "." + base64url(signature)
//
// signature คือผลลัพธ์ของ HMAC-SHA256 ที่คำนวณจาก "base64url(header).base64url(payload)"
// ทั้งก้อน โดยใช้ secret ที่มีแค่ server รู้ (ไม่ใช่ secret ของ user แต่ละคน) — จุดสำคัญ
// คือ header/payload เป็นแค่ base64 (encode ธรรมดา ไม่ใช่ encrypt) ใครก็ตามที่ได้ token
// ไปสามารถ decode มาอ่าน payload ได้เลยโดยไม่ต้องมี secret เลย (ห้ามใส่ข้อมูลลับใน
// Claims) สิ่งที่ secret ป้องกันคือ "การปลอมแปลง" ไม่ใช่ "การอ่าน" — ถ้ามีคนแก้ไข payload
// เอง (เช่น เปลี่ยน Subject เป็นชื่อ admin) แล้วส่ง token กลับมา signature จะไม่ตรงกับ
// payload ที่ถูกแก้แล้วอีกต่อไป Verify ด้านล่างจะจับได้ทันที
func Issue(secret []byte, subject string, ttl time.Duration) (string, error) {
	h, err := encodeSegment(header{Alg: "HS256", Typ: "JWT"})
	if err != nil {
		return "", fmt.Errorf("auth: encode header: %w", err)
	}
	c, err := encodeSegment(Claims{Subject: subject, ExpiresAt: time.Now().Add(ttl).Unix()})
	if err != nil {
		return "", fmt.Errorf("auth: encode claims: %w", err)
	}

	signingInput := h + "." + c
	return signingInput + "." + sign(secret, signingInput), nil
}

// Verify ตรวจสอบ JWT ว่า (1) signature ถูกต้องจริง คำนวณจาก secret เดียวกันกับตอน Issue
// และ payload ไม่ถูกแก้ไขระหว่างทาง และ (2) ยังไม่หมดอายุ — คืน Claims ถ้าผ่านทั้งสอง
// เงื่อนไข ไม่งั้นคืน error พร้อมบอกว่าพังเพราะอะไร (แต่ caller ควรตอบ client แค่
// "unauthorized" เฉยๆ ไม่ควร forward รายละเอียด error นี้ตรงๆ ไปให้ client เห็น เพื่อไม่ให้
// เป็นข้อมูลที่ช่วยผู้โจมตีปลอมแปลง token ได้ง่ายขึ้น)
func Verify(secret []byte, token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("auth: malformed token")
	}
	signingInput := parts[0] + "." + parts[1]

	// hmac.Equal เปรียบเทียบแบบ constant-time (ใช้เวลาเท่ากันเสมอไม่ว่า byte ไหนต่างกัน
	// ก่อน) แทนที่จะเทียบ string ตรงๆ ด้วย == ธรรมดา — ถ้าเทียบแบบธรรมดา ผู้โจมตีอาจวัด
	// เวลาที่ใช้เทียบเพื่อเดา signature ที่ถูกต้องทีละ byte ได้ (timing attack) เป็น best
	// practice มาตรฐานเวลาต้องเทียบค่าที่เกี่ยวกับ cryptographic signature เสมอ
	expected := sign(secret, signingInput)
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return nil, errors.New("auth: invalid signature")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("auth: decode payload: %w", err)
	}
	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("auth: decode claims: %w", err)
	}
	if time.Now().Unix() > claims.ExpiresAt {
		return nil, errors.New("auth: token expired")
	}
	return &claims, nil
}

// encodeSegment แปลง value เป็น JSON แล้ว base64url-encode แบบไม่มี padding (RawURLEncoding)
// ตามที่ JWT spec กำหนด (URL-safe alphabet, ไม่มี "=" ต่อท้าย เพราะ "=" มีความหมายพิเศษ
// ใน query string ถ้า token ถูกส่งผ่าน URL)
func encodeSegment(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// sign คำนวณ HMAC-SHA256 ของ signingInput ด้วย secret แล้ว encode เป็น base64url —
// ใช้ร่วมกันทั้ง Issue (สร้าง signature ใหม่) และ Verify (คำนวณ signature ที่ "ควรจะเป็น"
// มาเทียบกับที่แนบมาใน token)
func sign(secret []byte, signingInput string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(signingInput))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
