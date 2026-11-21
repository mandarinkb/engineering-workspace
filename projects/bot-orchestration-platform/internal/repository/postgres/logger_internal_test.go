// package postgres (ไม่ใช่ postgres_test) เพราะทดสอบ sanitizeMessage ที่ไม่ export ตรงๆ
package postgres

import "testing"

func TestSanitizeMessage(t *testing.T) {
	tests := []struct {
		name string
		give string
		want string
	}{
		{"plain text unchanged", "hello เทสต์", "hello เทสต์"},
		{"NUL removed", "a\x00b\x00", "ab"},
		// byte เสียที่ติดกันถูกแทนด้วย U+FFFD ตัวเดียวต่อหนึ่งช่วง (พฤติกรรมของ strings.ToValidUTF8)
		{"invalid UTF-8 becomes U+FFFD", "bad\xff\xfebytes", "bad�bytes"},
		{"both", "x\x00\xc3(y", "x�(y"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeMessage(tt.give); got != tt.want {
				t.Errorf("sanitizeMessage(%q) = %q, want %q", tt.give, got, tt.want)
			}
		})
	}
}
