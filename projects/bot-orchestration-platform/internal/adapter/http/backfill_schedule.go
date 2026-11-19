package http

import (
	"encoding/json"
	"net/http"
	"time"
)

// backfillScheduleRequest คือ JSON body ของ POST /schedules/{id}/backfill — เทียบเท่า "Rerun"
// ของ Control-M
type backfillScheduleRequest struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Overlap   string    `json:"overlap,omitempty"`
}

func (s *Server) handleBackfillSchedule(w http.ResponseWriter, r *http.Request) {
	var req backfillScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if err := s.schedules.Backfill(r.Context(), r.PathValue("id"), req.StartTime, req.EndTime, req.Overlap); err != nil {
		writeScheduleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
