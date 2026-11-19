package http

import "net/http"

// handleHealthz ตอบ GET /healthz — liveness: process ยังตอบสนองอยู่ไหม (ไม่เช็ค dependency
// ภายนอกเลยโดยตั้งใจ — liveness fail = K8s restart Pod ทันที ไม่ควร restart แค่เพราะ
// Temporal server ช้าชั่วคราว นั่นเป็นหน้าที่ของ readiness ไม่ใช่ liveness)
func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
