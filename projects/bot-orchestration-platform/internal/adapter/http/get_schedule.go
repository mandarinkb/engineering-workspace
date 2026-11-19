package http

import "net/http"

// handleGetSchedule คืนรายละเอียดเต็มของ schedule เดียว (รวม overlap policy, ประวัติ
// การรันล่าสุด, และ steps จาก schedule.Repository)
func (s *Server) handleGetSchedule(w http.ResponseWriter, r *http.Request) {
	detail, err := s.schedules.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeScheduleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toScheduleDetail(detail))
}
