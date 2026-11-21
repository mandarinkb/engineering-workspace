package http_test

import (
	"net/http"
	"testing"
)

// TestListWorkflowRuns_RejectsBadQueryBeforeTouchingTemporal พิสูจน์ว่า query param ที่ผิด
// ของ GET /workflow-runs ตอบ 400 ก่อนแตะ Temporal เสมอ — newTestServer ไม่มี Temporal
// client จริง (nil) ถ้า handler หลุดไปเรียก engine จะ panic แทนที่จะได้ 400 ส่วนการเดินหน้า
// และ filter สถานะจริงทดสอบผ่าน seam A (Bruno) กับ Temporal ที่รันอยู่จริง
func TestListWorkflowRuns_RejectsBadQueryBeforeTouchingTemporal(t *testing.T) {
	ts := newTestServer(t)

	tests := []struct {
		name      string
		giveQuery string
	}{
		{"unknown status", "status=Running"},
		{"page_size not a number", "page_size=abc"},
		{"page_size zero", "page_size=0"},
		{"page_size negative", "page_size=-1"},
		{"page_token not base64", "page_token=%25%25%25"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Get(ts.URL + "/workflow-runs?" + tt.giveQuery)
			if err != nil {
				t.Fatalf("GET /workflow-runs: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", resp.StatusCode)
			}
		})
	}
}
