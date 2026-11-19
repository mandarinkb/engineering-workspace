package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"bop/internal/domain/apperr"
	"bop/internal/domain/workflow"
)

// createWorkflowRunRequest คือ JSON body ของ POST /workflow-runs — เหมือน steps/workflow_name
// ของ createScheduleRequest แต่ไม่มี spec/overlap เพราะ endpoint นี้ไม่ได้สร้าง schedule
// (การรันซ้ำตามตารางเวลา) แค่รัน step ที่ระบุ "ทันทีครั้งเดียว" (ad-hoc) เท่านั้น
type createWorkflowRunRequest struct {
	WorkflowName string          `json:"workflow_name"`
	Steps        []workflow.Step `json:"steps"`
}

// createWorkflowRunResponse คือ JSON response — ให้แค่ id ที่พอเอาไป poll สถานะต่อผ่าน
// GET /workflow-runs/{id} ที่มีอยู่แล้ว (ไม่ต้องเพิ่ม endpoint หรือ repository ใหม่เลย)
type createWorkflowRunResponse struct {
	WorkflowID string `json:"workflow_id"`
	RunID      string `json:"run_id"`
}

// handleCreateWorkflowRun สั่งรัน workflow (1 หรือหลาย step เรียงกัน) ทันทีแบบ ad-hoc —
// ดูเหตุผลและกฎการตั้ง workflow ID ที่ workflowrun.Service.Start
//
// เหมือนกับ handleCreateJob endpoint นี้เป็น fire-and-forget (เริ่ม workflow แล้ว return
// ทันที ไม่รอผลลัพธ์) — client ต้อง poll สถานะทีหลังเองผ่าน GET /workflow-runs/{workflow_id}
func (s *Server) handleCreateWorkflowRun(w http.ResponseWriter, r *http.Request) {
	var req createWorkflowRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	run, err := s.runs.Start(r.Context(), req.WorkflowName, req.Steps)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, apperr.ErrInvalid) {
			status = http.StatusBadRequest
		}
		http.Error(w, err.Error(), status)
		return
	}

	writeJSON(w, http.StatusAccepted, createWorkflowRunResponse{
		WorkflowID: run.WorkflowID,
		RunID:      run.RunID,
	})
}
