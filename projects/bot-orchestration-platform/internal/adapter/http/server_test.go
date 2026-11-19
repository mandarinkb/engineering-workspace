package http_test

import (
	"net/http"
	"testing"
)

// TestCORSHeaders ทดสอบว่าทุก response มี CORS header ที่ dashboard (origin คนละ port)
// ต้องใช้ และ OPTIONS preflight ตอบ 204 ทันทีโดยไม่ต้องผ่าน route จริง
func TestCORSHeaders(t *testing.T) {
	ts := newTestServer(t)

	req, err := http.NewRequest(http.MethodOptions, ts.URL+"/jobs", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("OPTIONS /jobs: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for preflight, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("expected Access-Control-Allow-Credentials=true, got %q", got)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got == "" {
		t.Error("expected Access-Control-Allow-Origin to be set")
	}
}
