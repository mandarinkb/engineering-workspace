package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// TestHandleCreateJob_AndReadBack ทดสอบเส้นทางเต็มของ job เดี่ยว: สร้างผ่าน
// POST /bots/{name}/jobs แล้วอ่านกลับผ่าน GET /jobs/{id} และ GET /jobs/{id}/logs ต้อง
// เห็นผลลัพธ์สำเร็จในที่สุด (poll สั้นๆ เพราะ Submit เป็น async)
func TestHandleCreateJob_AndReadBack(t *testing.T) {
	ts := newTestServer(t)

	resp, err := http.Post(ts.URL+"/bots/stub-bot/jobs", "application/json", bytes.NewBufferString(`{"url":"https://example.com"}`))
	if err != nil {
		t.Fatalf("POST job: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.StatusCode)
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		r, err := http.Get(ts.URL + "/jobs/" + created.ID)
		if err != nil {
			t.Fatalf("GET job: %v", err)
		}
		var j struct {
			Status string `json:"status"`
		}
		_ = json.NewDecoder(r.Body).Decode(&j)
		r.Body.Close()
		if j.Status == "succeeded" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("job did not succeed in time, last status: %q", j.Status)
		}
		time.Sleep(20 * time.Millisecond)
	}

	logsResp, err := http.Get(ts.URL + "/jobs/" + created.ID + "/logs")
	if err != nil {
		t.Fatalf("GET logs: %v", err)
	}
	defer logsResp.Body.Close()
	var lines []struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(logsResp.Body).Decode(&lines); err != nil {
		t.Fatalf("decode logs: %v", err)
	}
	if len(lines) == 0 {
		t.Error("expected at least one log line from the stub bot")
	}
}
