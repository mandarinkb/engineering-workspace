package http

import (
	"net/http"

	"bop/internal/domain/workflow"
)

// workflowRunDetail ต่อยอดจาก workflowRunSummary เพิ่มผลลัพธ์ทีละ step เข้ามา
type workflowRunDetail struct {
	workflowRunSummary
	Steps []workflow.StepResult `json:"steps"`
}

// handleGetWorkflowRun คืนรายละเอียด workflow run เดียว รวมผลลัพธ์ทีละ step ที่รันไปแล้ว
// — id ในที่นี้คือ Temporal Workflow ID — ใช้ได้ทั้งตอน workflow ยังรันอยู่และจบไปแล้ว
// (ดูวิธีดึงผลลัพธ์ทีละ step ที่ internal/external/temporal/engine.go)
func (s *Server) handleGetWorkflowRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	run, err := s.runs.Get(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, workflowRunDetail{
		workflowRunSummary: toRunSummary(run.RunSummary),
		Steps:              run.Steps,
	})
}
