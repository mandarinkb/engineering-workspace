package http

import "net/http"

// handleGetJobLogs คืน log ทั้งหมดที่บันทึกไว้ของ job นั้น (ยังไม่ใช่ real-time
// ตอนนี้ — เป็นแค่ snapshot ของ log ณ ตอนที่เรียก endpoint นี้)
func (s *Server) handleGetJobLogs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	lines, err := s.jobs.Logs(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, lines)
}
