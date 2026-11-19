package http

import "net/http"

// handleListJobs คืนรายการ job ทั้งหมดในระบบ เรียงจากล่าสุดไปเก่าสุด
func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := s.jobs.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, jobs)
}
