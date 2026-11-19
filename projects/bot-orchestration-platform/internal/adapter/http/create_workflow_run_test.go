package http_test

import (
	"bytes"
	"net/http"
	"testing"
)

// TestCreateWorkflowRun_RejectsBadRequestsBeforeTouchingTemporal ทดสอบ path เดียวกับ
// TestScheduleValidation_RejectsBadRequestsBeforeTouchingTemporal (create_schedule_test.go) แต่สำหรับ
// POST /workflow-runs — newTestServer ไม่มี Temporal client จริง (nil) จึงทดสอบ "สร้าง
// workflow run สำเร็จจริง" ในนี้ไม่ได้ (ต้องมี Temporal server จริง ทดสอบแบบ end-to-end
// แยกต่างหาก) แต่ยังพิสูจน์ได้ว่า handler validate request ผิดก่อนแตะ s.temporal เสมอ
// (ไม่ panic เพราะเรียก method บน nil client)
func TestCreateWorkflowRun_RejectsBadRequestsBeforeTouchingTemporal(t *testing.T) {
	ts := newTestServer(t)

	cases := []struct {
		name string
		body string
	}{
		{"missing workflow_name", `{"steps":[{"bot_name":"stub-bot","config":{}}]}`},
		{"missing steps", `{"workflow_name":"wf","steps":[]}`},
		{"invalid json", `not json`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Post(ts.URL+"/workflow-runs", "application/json", bytes.NewBufferString(tc.body))
			if err != nil {
				t.Fatalf("POST /workflow-runs: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", resp.StatusCode)
			}
		})
	}
}
