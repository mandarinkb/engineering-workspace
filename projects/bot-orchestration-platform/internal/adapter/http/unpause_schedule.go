package http

import (
	"encoding/json"
	"net/http"
)

// handleUnpauseSchedule เริ่ม schedule ที่ pause ไว้ใหม่ (เทียบ "Free" ของ Control-M) — ดู
// handlePauseSchedule
func (s *Server) handleUnpauseSchedule(w http.ResponseWriter, r *http.Request) {
	var req pauseScheduleRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	if err := s.schedules.Unpause(r.Context(), r.PathValue("id"), req.Note); err != nil {
		writeScheduleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
