package http

import (
	"errors"
	"net/http"
	"strconv"

	"bop/internal/domain/apperr"
	"bop/internal/domain/workflow"
)

// workflowRunPage คือ JSON response ของ GET /workflow-runs — next_page_token ว่างแปลว่า
// ไม่มีหน้าถัดไป (ส่งเป็น "" เสมอ ไม่ละ field ทิ้ง ให้ client เช็คค่าเดียวพอ)
type workflowRunPage struct {
	Runs          []workflowRunSummary `json:"runs"`
	NextPageToken string               `json:"next_page_token"`
}

// handleListWorkflowRuns คืน Workflow Run ทีละหน้า (ทั้งที่กำลังรันอยู่และจบไปแล้ว)
// เรียงจากล่าสุดไปเก่าสุดตาม default ordering ของ Temporal Visibility API — รับ page_size
// (default 50 สูงสุด 200), page_token จากหน้าก่อน และ status (running | completed | failed |
// canceled) ที่กรองฝั่ง server แทนให้ dashboard โหลดทุกตัวมากรองเอง (engine เป็นคนเก็บ
// event history + visibility record ให้ทั้งหมด ดู internal/external/temporal/engine.go)
func (s *Server) handleListWorkflowRuns(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	q := workflow.RunQuery{
		PageToken: query.Get("page_token"),
		Status:    workflow.RunStatusFilter(query.Get("status")),
	}
	// page_size 0 ใน RunQuery แปลว่า "ไม่ได้ระบุ" (usecase ใส่ค่าเริ่มต้นให้) ดังนั้นค่าที่
	// client ส่งมาเองต้องเป็นจำนวนเต็มบวกเท่านั้น ไม่งั้นจะแยกไม่ออกจากการไม่ส่ง
	if raw := query.Get("page_size"); raw != "" {
		size, err := strconv.Atoi(raw)
		if err != nil || size <= 0 {
			http.Error(w, "page_size must be a positive integer", http.StatusBadRequest)
			return
		}
		q.PageSize = size
	}

	page, err := s.runs.List(r.Context(), q)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, apperr.ErrInvalid) {
			status = http.StatusBadRequest
		}
		http.Error(w, err.Error(), status)
		return
	}

	runs := make([]workflowRunSummary, 0, len(page.Runs))
	for _, run := range page.Runs {
		runs = append(runs, toRunSummary(run))
	}
	writeJSON(w, http.StatusOK, workflowRunPage{Runs: runs, NextPageToken: page.NextPageToken})
}
