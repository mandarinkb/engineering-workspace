package http

import (
	"encoding/json"
	"net/http"

	"bop/internal/domain/workflow"
	scheduleuc "bop/internal/usecase/schedule"
)

// updateScheduleRequest คือ JSON body ของ PUT /schedules/{id} — ดูเหตุผลที่แยก type จาก
// createScheduleRequest ที่ create_schedule.go
type updateScheduleRequest struct {
	WorkflowName string              `json:"workflow_name"`
	Steps        []workflow.Step     `json:"steps"`
	Spec         scheduleSpecRequest `json:"spec"`
	Overlap      string              `json:"overlap,omitempty"`
	// CatchupWindowSeconds เป็น pointer เพื่อแยก "ไม่ส่ง" (ได้ค่า default) ออกจาก "ส่ง 0"
	// (ต้องถูกปฏิเสธ)
	CatchupWindowSeconds *int64 `json:"catchup_window_seconds,omitempty"`
}

// handleUpdateSchedule แทนที่ schedule ทั้งก้อนด้วยค่าใหม่ (PUT = full replace ตาม REST
// convention ปกติ ไม่ใช่ partial patch)
func (s *Server) handleUpdateSchedule(w http.ResponseWriter, r *http.Request) {
	var req updateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	detail, err := s.schedules.Update(r.Context(), r.PathValue("id"), scheduleuc.Input{
		WorkflowName:         req.WorkflowName,
		Steps:                req.Steps,
		Spec:                 req.Spec.toDomain(),
		Overlap:              req.Overlap,
		CatchupWindowSeconds: req.CatchupWindowSeconds,
	})
	if err != nil {
		writeScheduleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toScheduleDetail(detail))
}
