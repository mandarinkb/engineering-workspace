package http_test

import (
	"bytes"
	"net/http"
	"testing"
)

// TestScheduleValidation_RejectsBadRequestsBeforeTouchingTemporal ทดสอบ path ที่
// create_schedule.go ต้อง validate request ก่อนแตะ Temporal client เสมอ — สำคัญเพราะ
// newTestServer (main_test.go) ไม่มี Temporal client จริงให้ต่อ (temporal เป็น nil) ทำให้
// ทดสอบ "สร้าง schedule สำเร็จจริง" ในนี้ไม่ได้ (ต้องมี Temporal server รันอยู่จริง —
// ทดสอบแยกแบบ end-to-end เท่านั้น ดู README) แต่ยังทดสอบได้ว่า handler ปฏิเสธ
// request ที่ผิดตั้งแต่ต้นโดยไม่ panic เพราะพยายามเรียก method บน nil client
func TestScheduleValidation_RejectsBadRequestsBeforeTouchingTemporal(t *testing.T) {
	ts := newTestServer(t)

	cases := []struct {
		name string
		body string
	}{
		{"missing id", `{"workflow_name":"wf","steps":[{"bot_name":"stub-bot","config":{}}],"spec":{"interval_seconds":60}}`},
		{"missing steps", `{"id":"s1","workflow_name":"wf","steps":[],"spec":{"interval_seconds":60}}`},
		{"missing spec", `{"id":"s1","workflow_name":"wf","steps":[{"bot_name":"stub-bot","config":{}}]}`},
		{"both interval and cron", `{"id":"s1","workflow_name":"wf","steps":[{"bot_name":"stub-bot","config":{}}],"spec":{"interval_seconds":60,"cron_expression":"* * * * *"}}`},
		{"max_attempts zero", `{"id":"s1","workflow_name":"wf","steps":[{"bot_name":"stub-bot","config":{},"max_attempts":0}],"spec":{"interval_seconds":60}}`},
		{"max_attempts negative", `{"id":"s1","workflow_name":"wf","steps":[{"bot_name":"stub-bot","config":{},"max_attempts":-2}],"spec":{"interval_seconds":60}}`},
		{"bad overlap policy", `{"id":"s1","workflow_name":"wf","steps":[{"bot_name":"stub-bot","config":{}}],"spec":{"interval_seconds":60},"overlap":"not-a-policy"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Post(ts.URL+"/schedules", "application/json", bytes.NewBufferString(tc.body))
			if err != nil {
				t.Fatalf("POST /schedules: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", resp.StatusCode)
			}
		})
	}
}
