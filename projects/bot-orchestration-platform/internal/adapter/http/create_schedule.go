package http

import (
	"encoding/json"
	"net/http"

	"bop/internal/domain/workflow"
	scheduleuc "bop/internal/usecase/schedule"
)

// createScheduleRequest/updateScheduleRequest คือ JSON body ของ POST/PUT /schedules —
// แยก type กันแม้หน้าตาจะคล้ายกันมาก เพราะ ID ของ create มาจาก body (ต้องระบุเอง) แต่
// ID ของ update มาจาก URL path (ห้ามเปลี่ยน ID ผ่าน update ได้ — ถ้าอยากเปลี่ยนชื่อต้อง
// ลบตัวเก่าสร้างตัวใหม่)
type createScheduleRequest struct {
	ID           string              `json:"id"`
	WorkflowName string              `json:"workflow_name"`
	Steps        []workflow.Step     `json:"steps"`
	Spec         scheduleSpecRequest `json:"spec"`
	Overlap      string              `json:"overlap,omitempty"`
	// CatchupWindowSeconds เป็น pointer เพื่อแยก "ไม่ส่ง" (ได้ค่า default) ออกจาก "ส่ง 0"
	// (ต้องถูกปฏิเสธ)
	CatchupWindowSeconds *int64 `json:"catchup_window_seconds,omitempty"`
}

// handleCreateSchedule สร้าง schedule ใหม่ตาม spec ที่ระบุ — ดูลำดับ "สร้างที่ scheduler
// ก่อนแล้วค่อยเก็บ definition" และ compensation ที่ scheduleuc.Service.Create
func (s *Server) handleCreateSchedule(w http.ResponseWriter, r *http.Request) {
	var req createScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	detail, err := s.schedules.Create(r.Context(), req.ID, scheduleuc.Input{
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
	writeJSON(w, http.StatusCreated, toScheduleDetail(detail))
}
