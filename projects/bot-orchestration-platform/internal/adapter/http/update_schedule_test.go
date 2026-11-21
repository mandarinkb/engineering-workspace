package http_test

import (
	"bytes"
	"net/http"
	"testing"
)

// TestUpdateSchedule_RejectsInvalidMaxAttempts ทดสอบว่า PUT /schedules/{id} ใช้กฎ max_attempts
// เดียวกับตอนสร้าง และตอบ 400 ก่อนแตะ Temporal client (ซึ่งเป็น nil ใน newTestServer)
func TestUpdateSchedule_RejectsInvalidMaxAttempts(t *testing.T) {
	ts := newTestServer(t)

	body := `{"workflow_name":"wf","steps":[{"bot_name":"stub-bot","config":{},"max_attempts":0}],"spec":{"interval_seconds":60}}`
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
}
