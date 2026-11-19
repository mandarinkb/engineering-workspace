package http

import "net/http"

// handleListSchedules คืนรายการ schedule ทั้งหมด
func (s *Server) handleListSchedules(w http.ResponseWriter, r *http.Request) {
	list, err := s.schedules.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	out := make([]scheduleSummary, 0, len(list))
	for _, item := range list {
		out = append(out, toScheduleSummary(item))
	}
	writeJSON(w, http.StatusOK, out)
}
