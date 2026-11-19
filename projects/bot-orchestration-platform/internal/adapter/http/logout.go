package http

import "net/http"

// handleLogout ล้าง cookie ทิ้งด้วยการตั้ง MaxAge เป็นค่าติดลบ (สั่งเบราว์เซอร์ให้ลบ
// cookie นี้ทันที) — ไม่ต้องเช็ค token เดิมเลยก่อนล้าง เพราะการล้าง cookie ปลอดภัยเสมอ
// ไม่ว่า token เดิมจะยังใช้ได้อยู่หรือหมดอายุไปแล้วก็ตาม
func (s *Server) handleLogout(w http.ResponseWriter, _ *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	w.WriteHeader(http.StatusNoContent)
}
