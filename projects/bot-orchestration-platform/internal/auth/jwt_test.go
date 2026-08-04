package auth_test

import (
	"strings"
	"testing"
	"time"

	"bop/internal/auth"
)

var testSecret = []byte("test-secret")

// TestIssueVerify_RoundTrip ทดสอบเส้นทางปกติ: Issue แล้ว Verify ด้วย secret เดียวกันต้อง
// ได้ Claims กลับมาตรงกับตอนสร้าง (Subject ถูกต้อง ยังไม่หมดอายุ)
func TestIssueVerify_RoundTrip(t *testing.T) {
	token, err := auth.Issue(testSecret, "admin", time.Hour)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	claims, err := auth.Verify(testSecret, token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if claims.Subject != "admin" {
		t.Errorf("expected subject %q, got %q", "admin", claims.Subject)
	}
}

// TestVerify_TamperedPayloadRejected ทดสอบว่าถ้ามีคนแก้ payload (เช่นพยายามเปลี่ยน
// Subject เป็นชื่ออื่นโดยไม่มี secret) Verify ต้องปฏิเสธทันที เพราะ signature เดิมคำนวณ
// จาก payload ตัวเก่า ไม่ตรงกับ payload ที่ถูกแก้ไขแล้ว — นี่คือคุณสมบัติหลักที่ JWT ต้อง
// มีเพื่อป้องกันการปลอมแปลง token
func TestVerify_TamperedPayloadRejected(t *testing.T) {
	token, err := auth.Issue(testSecret, "admin", time.Hour)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 segments, got %d", len(parts))
	}
	// สลับ segment payload กับ signature มั่วๆ เพื่อจำลอง token ที่ถูกแก้ไข
	tampered := parts[0] + ".tampered-payload." + parts[2]

	if _, err := auth.Verify(testSecret, tampered); err == nil {
		t.Fatal("expected tampered token to be rejected")
	}
}

// TestVerify_ExpiredTokenRejected ทดสอบว่า token ที่หมดอายุแล้ว (ttl ติดลบ) ต้องถูก
// ปฏิเสธ แม้ signature จะถูกต้องสมบูรณ์ก็ตาม
func TestVerify_ExpiredTokenRejected(t *testing.T) {
	token, err := auth.Issue(testSecret, "admin", -time.Hour) // หมดอายุไปแล้ว 1 ชม.
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	if _, err := auth.Verify(testSecret, token); err == nil {
		t.Fatal("expected expired token to be rejected")
	}
}

// TestVerify_WrongSecretRejected ทดสอบว่า secret คนละตัวกับตอน Issue ต้อง verify ไม่ผ่าน
// (จำลองสถานการณ์ที่ token ถูกปลอมโดยคนที่ไม่รู้ secret จริงของ server)
func TestVerify_WrongSecretRejected(t *testing.T) {
	token, err := auth.Issue(testSecret, "admin", time.Hour)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	if _, err := auth.Verify([]byte("wrong-secret"), token); err == nil {
		t.Fatal("expected token signed with a different secret to be rejected")
	}
}
