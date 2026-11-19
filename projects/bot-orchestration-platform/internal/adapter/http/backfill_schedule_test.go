package http_test

import (
	"bytes"
	"net/http"
	"testing"
)

// TestScheduleBackfill_RejectsInvalidTimeRange ทดสอบว่า start_time >= end_time โดน
// ปฏิเสธก่อนแตะ Temporal client เหมือนกัน
func TestScheduleBackfill_RejectsInvalidTimeRange(t *testing.T) {
	ts := newTestServer(t)

	body := `{"start_time":"2026-01-02T00:00:00Z","end_time":"2026-01-01T00:00:00Z"}`
	resp, err := http.Post(ts.URL+"/schedules/does-not-matter/backfill", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("POST backfill: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
