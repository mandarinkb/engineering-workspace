// Package idgen สุ่มสร้างรหัสเฉพาะ (ID) ให้ job และ workflow run — แยกออกมาจาก HTTP
// handler เดิมเพราะตอนนี้ usecase เป็นคนสร้าง ID ไม่ใช่ชั้น HTTP
package idgen

import (
	"crypto/rand"
	"encoding/hex"
)

// New สุ่มสร้างรหัสเฉพาะโดยใช้ crypto/rand (สุ่มแบบปลอดภัยทาง cryptography) สร้าง byte
// สุ่ม 16 ตัวแล้วแปลงเป็น string hex — ไม่ใช้ library ภายนอกอย่าง uuid เพื่อให้โปรเจกต์นี้
// ยังคง zero external dependency อยู่
func New() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
