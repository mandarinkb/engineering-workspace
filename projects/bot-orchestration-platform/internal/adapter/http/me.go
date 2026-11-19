package http

import "net/http"

// handleMe ตรวจ cookie ปัจจุบันว่ายัง valid อยู่ไหม (signature ถูกต้อง + ยังไม่หมดอายุ)
// คืนชื่อ user ถ้าผ่าน — dashboard เรียก endpoint นี้ตอนโหลดหน้าแรกเพื่อรู้ว่า user
// ยัง login ค้างอยู่ไหม (แทนที่จะพยายามอ่าน/ถอดรหัส JWT เองฝั่ง client ซึ่งทำไม่ได้อยู่
// แล้วเพราะ cookie เป็น httpOnly)
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		http.Error(w, "not authenticated", http.StatusUnauthorized)
		return
	}
	username, err := s.auth.Verify(cookie.Value)
	if err != nil {
		http.Error(w, "not authenticated", http.StatusUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"username": username})
}
