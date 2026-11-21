package http

import (
	"errors"
	"net/http"
	"strconv"

	"bop/internal/domain/apperr"
)

// handleGetWorkflowRunStepLogs คืน log ทุก Attempt ของ Step หนึ่งใน Workflow Run เรียงตามเวลา —
// id คือ Temporal Workflow ID และ index คือลำดับ Step เริ่มที่ 0 (ตรงกับ step_index ใน
// GET /workflow-runs/{id}) แต่ละบรรทัดมี attempt บอกว่ามาจาก Attempt ไหน
//
// index ที่ไม่ใช่ตัวเลขตอบ 400 ส่วนลำดับ Step หรือ Workflow Run ที่ไม่มีจริงตอบ 404 เพื่อให้พิมพ์ผิด
// เห็นได้ทันที Step ที่มีจริงแต่ยังไม่ได้รันตอบ []
func (s *Server) handleGetWorkflowRunStepLogs(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "step index must be an integer", http.StatusBadRequest)
		return
	}

	lines, err := s.runs.StepLogs(r.Context(), r.PathValue("id"), index)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, apperr.ErrNotFound) {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}
	writeJSON(w, http.StatusOK, lines)
}
