package http

import (
	"encoding/json"
	"net/http"
)

// pauseScheduleRequest คือ JSON body ของ POST /schedules/{id}/pause (unpause ใช้ type
// เดียวกันเพราะรับแค่ note เหมือนกัน) — note เป็น optional: Temporal มี default ให้เองถ้า
// ไม่ระบุมา เช่น "Paused via Go SDK"
type pauseScheduleRequest struct {
	Note string `json:"note,omitempty"`
}

// handlePauseSchedule/handleUnpauseSchedule เทียบเท่า "Hold"/"Free" ของ Control-M —
// หยุดชั่วคราวโดยไม่ลบ schedule ทิ้ง (ต่างจาก DELETE ที่ลบถาวร)
func (s *Server) handlePauseSchedule(w http.ResponseWriter, r *http.Request) {
	var req pauseScheduleRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // body ว่างได้ ไม่ error ถ้า decode ไม่ผ่าน

	if err := s.schedules.Pause(r.Context(), r.PathValue("id"), req.Note); err != nil {
		writeScheduleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
