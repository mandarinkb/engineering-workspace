package http_test

import (
	"bytes"
	"io"
	"net/http"
	"strings"
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
		{"bad overlap policy", `{"id":"s1","workflow_name":"wf","steps":[{"bot_name":"stub-bot","config":{}}],"spec":{"interval_seconds":60},"overlap":"not-a-policy"}`},
		{"catch-up window shorter than 10 seconds", `{"id":"s1","workflow_name":"wf","steps":[{"bot_name":"stub-bot","config":{}}],"spec":{"interval_seconds":60},"catchup_window_seconds":5}`},
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

// TestUpdateSchedule_RejectsShortCatchupWindowBeforeTouchingTemporal ทดสอบว่า PUT
// ปฏิเสธ Catch-up Window ที่สั้นกว่า 10 วินาทีพร้อมข้อความที่บอกชื่อ field ก่อนแตะ
// Temporal client (เป็น nil ใน newTestServer)
func TestUpdateSchedule_RejectsShortCatchupWindowBeforeTouchingTemporal(t *testing.T) {
	ts := newTestServer(t)
	body := `{"workflow_name":"wf","steps":[{"bot_name":"stub-bot","config":{}}],"spec":{"interval_seconds":60},"catchup_window_seconds":5}`

	req, err := http.NewRequest(http.MethodPut, ts.URL+"/schedules/s1", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT /schedules/s1: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	msg, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(msg), "catchup_window_seconds") {
		t.Errorf("expected error message to name catchup_window_seconds, got %q", msg)
	}
}
