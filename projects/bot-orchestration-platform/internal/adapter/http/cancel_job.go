package http

import "net/http"

// handleCancelJob สั่งยกเลิก job ที่กำลังรันอยู่ คืน 409 Conflict ถ้า job นั้นไม่ได้
// กำลังรันอยู่แล้ว (จบไปแล้ว หรือ ID ไม่มีอยู่จริง)
func (s *Server) handleCancelJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.jobs.Cancel(id) {
		http.Error(w, "job not running", http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
