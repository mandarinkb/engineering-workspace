package http

import "net/http"

// handleReadyz ตอบ GET /readyz — readiness: process พร้อมรับ traffic จริงไหม (เช็คว่าต่อ
// Temporal ได้อยู่) K8s หยุดส่ง traffic มาที่ Pod นี้ชั่วคราว (ไม่ restart) ถ้า readiness fail
// — ต่างจาก liveness ตรงนี้แหละ: Temporal ล่มชั่วคราวไม่ควรทำให้ Pod ถูก restart วนไปเรื่อยๆ
// (CrashLoopBackOff) แค่หยุดรับ traffic จนกว่า Temporal จะกลับมา
func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	if err := s.runs.Ready(r.Context()); err != nil {
		http.Error(w, "temporal unreachable", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}
