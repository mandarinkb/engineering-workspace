package http

import "net/http"

// handleDeleteSchedule ลบ schedule ทิ้งทั้งฝั่ง scheduler และ definition ของเราเอง
func (s *Server) handleDeleteSchedule(w http.ResponseWriter, r *http.Request) {
	if err := s.schedules.Delete(r.Context(), r.PathValue("id")); err != nil {
		writeScheduleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
