package http

import "net/http"

// handleGetJob คืนข้อมูล job เดียวตาม ID ที่ระบุใน path (สถานะล่าสุด ณ ตอนที่เรียก)
func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	j, err := s.jobs.Get(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, j)
}
