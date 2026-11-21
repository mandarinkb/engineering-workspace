package http

import "net/http"

// handleMe คืนชื่อ user ของ cookie ปัจจุบัน — dashboard เรียก endpoint นี้ตอนโหลดหน้าแรก
// เพื่อรู้ว่า user ยัง login ค้างอยู่ไหม (แทนที่จะพยายามอ่าน/ถอดรหัส JWT เองฝั่ง client
// ซึ่งทำไม่ได้อยู่แล้วเพราะ cookie เป็น httpOnly) การตรวจ cookie (signature ถูกต้อง +
// ยังไม่หมดอายุ) เกิดใน requireSession ก่อนถึง handler นี้แล้ว ถ้าไม่ผ่านจะได้ 401 ไปเลย
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"username": usernameFrom(r.Context())})
}
