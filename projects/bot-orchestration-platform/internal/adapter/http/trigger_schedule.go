package http

import (
	"encoding/json"
	"net/http"
)

// triggerScheduleRequest คือ JSON body ของ POST /schedules/{id}/trigger — Overlap ปล่อยว่างได้
// (ใช้ policy default ของ schedule เอง)
type triggerScheduleRequest struct {
	Overlap string `json:"overlap,omitempty"`
}

// handleTriggerSchedule สั่งรัน schedule นี้ "ทันที นอกตารางเวลาปกติ" — เทียบเท่า
// "Order Now"/"Run Now" ของ Control-M
func (s *Server) handleTriggerSchedule(w http.ResponseWriter, r *http.Request) {
	var req triggerScheduleRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	if err := s.schedules.Trigger(r.Context(), r.PathValue("id"), req.Overlap); err != nil {
		writeScheduleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
