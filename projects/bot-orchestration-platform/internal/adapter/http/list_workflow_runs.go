package http

import "net/http"

// handleListWorkflowRuns คืนรายการ workflow run ทั้งหมด (ทั้งที่กำลังรันอยู่และจบไปแล้ว)
// เรียงจากล่าสุดไปเก่าสุดตาม default ordering ของ Temporal Visibility API — ใช้ดูว่า
// scheduled workflow ทำงานไปแล้วกี่รอบ สำเร็จ/ล้มเหลวรอบไหนบ้าง (engine เป็นคนเก็บ event
// history + visibility record ให้ทั้งหมด ดู internal/external/temporal/engine.go)
func (s *Server) handleListWorkflowRuns(w http.ResponseWriter, r *http.Request) {
	list, err := s.runs.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	runs := make([]workflowRunSummary, 0, len(list))
	for _, run := range list {
		runs = append(runs, toRunSummary(run))
	}
	writeJSON(w, http.StatusOK, runs)
}
