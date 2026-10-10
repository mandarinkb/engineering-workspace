// Package auth คือ usecase ของการ login — ตรวจ credential แล้วออก token และตรวจ token ที่
// client ส่งกลับมา ส่วนการเก็บ token ใน httpOnly cookie เป็นเรื่องของชั้น HTTP
// (internal/adapter/http) ไม่ใช่ของ usecase นี้
package auth

import (
	"errors"
	"time"

	"bop/internal/pkg/jwt"
)

// TokenTTL คืออายุของ token ที่ออกให้ตอน login
const TokenTTL = 24 * time.Hour

// ErrInvalidCredentials คืนจาก Login เมื่อ username/password ไม่ตรง
var ErrInvalidCredentials = errors.New("invalid credentials")

// Service ถือ secret ที่ใช้เซ็น token กับ credential เดียวที่ระบบรู้จักตอนนี้ — ยังไม่มี
// user database เลย (BOP เป็น single-operator tool ควบคุม bot ของตัวเอง ไม่ใช่ multi-tenant
// SaaS) ค่าทั้งหมดอ่านจาก env var ใน cmd/bop/main.go
type Service struct {
	secret   []byte
	username string
	password string
}

// NewService สร้าง Service จาก secret และ credential ที่ประกอบมาจาก composition root
func NewService(secret []byte, username, password string) *Service {
	return &Service{secret: secret, username: username, password: password}
}

// Login ตรวจ username/password ถ้าถูกต้องคืน token ที่มีอายุ TokenTTL
func (s *Service) Login(username, password string) (string, error) {
	if username != s.username || password != s.password {
		return "", ErrInvalidCredentials
	}
	return jwt.Issue(s.secret, username, TokenTTL)
}

// Verify ตรวจ token (signature ถูกต้อง + ยังไม่หมดอายุ) แล้วคืนชื่อ user
func (s *Service) Verify(token string) (string, error) {
	claims, err := jwt.Verify(s.secret, token)
	if err != nil {
		return "", err
	}
	return claims.Subject, nil
}
