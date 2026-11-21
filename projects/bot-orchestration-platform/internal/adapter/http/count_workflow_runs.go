package http

import (
	"errors"
	"net/http"

	"bop/internal/domain/apperr"
)

// workflowRunCount คือ JSON response ของ GET /workflow-runs/count
type workflowRunCount struct {
	Count int64 `json:"count"`
}

// handleCountWorkflowRuns นับ Workflow Run ตาม status (running | completed | failed | canceled,
// ไม่ส่ง = ทุกสถานะ) และ schedule_id (filter เดียวกับ GET /workflow-runs) ฝั่ง server — Overview ของ dashboard ใช้ตัวเลขนี้แทนการนับจากหน้าแรก
// ของ GET /workflow-runs ซึ่งผิดทันทีที่มี run เกินหนึ่งหน้า
func (s *Server) handleCountWorkflowRuns(w http.ResponseWriter, r *http.Request) {
	count, err := s.runs.Count(r.Context(), runFilterFromQuery(r.URL.Query()))
	if err != nil {
		code := http.StatusInternalServerError
		if errors.Is(err, apperr.ErrInvalid) {
			code = http.StatusBadRequest
		}
		http.Error(w, err.Error(), code)
		return
	}
	writeJSON(w, http.StatusOK, workflowRunCount{Count: count})
}
