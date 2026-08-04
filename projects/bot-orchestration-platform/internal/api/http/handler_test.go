package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
	"time"

	apihttp "bop/internal/api/http"
	"bop/internal/bot"
	memlogger "bop/internal/logger/memory"
	memrepo "bop/internal/repository/memory"
	"bop/internal/worker"
)

// stubBot คือ bot ปลอมสำหรับทดสอบ HTTP layer โดยเฉพาะ — ประกาศ ConfigSchema จริงจัง
// เพื่อพิสูจน์ว่า GET /bots ส่งต่อ schema นั้นออกไปให้ครบ ไม่ใช่แค่ชื่อเฉยๆ
type stubBot struct{}

func (stubBot) Name() string { return "stub-bot" }
func (stubBot) ConfigSchema() []bot.ConfigField {
	return []bot.ConfigField{{Key: "url", Label: "URL", Type: bot.ConfigFieldURL, Required: true}}
}
func (stubBot) Run(_ context.Context, log bot.LogFunc, _ bot.Config) (bot.Result, error) {
	log("running")
	return bot.Result{Summary: "ok"}, nil
}

// newTestServer ประกอบ Server เต็มรูปแบบเหมือน cmd/bop/main.go ทุกประการ ยกเว้น
// temporalClient ที่ปล่อยเป็น nil ไปเลย — ปลอดภัยเพราะ test ในไฟล์นี้ไม่แตะ endpoint
// กลุ่ม /workflow-runs ที่ต้องใช้ temporal.Client จริงเลย (นั่นต้องมี Temporal server
// รันอยู่จริง ทดสอบแยกต่างหากแบบ end-to-end ไม่ใช่ unit test ระดับนี้)
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	repo := memrepo.NewJobRepository()
	log := memlogger.New()
	pool := worker.NewPool(2, 5*time.Second, repo, log)
	pool.Register(stubBot{})
	scheduleRepo := memrepo.NewScheduleRepository()

	srv := apihttp.NewServer(repo, log, pool, nil, scheduleRepo)
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	return ts
}

// TestHandleListBots ทดสอบว่า GET /bots คืนชื่อ + config schema ของ bot ที่ register ไว้
func TestHandleListBots(t *testing.T) {
	ts := newTestServer(t)

	resp, err := http.Get(ts.URL + "/bots")
	if err != nil {
		t.Fatalf("GET /bots: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var bots []struct {
		Name         string `json:"name"`
		ConfigSchema []struct {
			Key      string `json:"key"`
			Required bool   `json:"required"`
		} `json:"config_schema"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&bots); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(bots) != 1 || bots[0].Name != "stub-bot" {
		t.Fatalf("expected exactly [stub-bot], got %+v", bots)
	}
	if len(bots[0].ConfigSchema) != 1 || bots[0].ConfigSchema[0].Key != "url" || !bots[0].ConfigSchema[0].Required {
		t.Errorf("expected config schema with required \"url\" field, got %+v", bots[0].ConfigSchema)
	}
}

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

// TestAuthFlow ทดสอบเส้นทางเต็มของ login: username/password ผิด -> 401, ถูก -> ได้
// cookie กลับมาแล้ว GET /me ต้องเห็น username เดียวกัน, logout แล้ว /me ต้อง 401 อีกครั้ง
func TestAuthFlow(t *testing.T) {
	ts := newTestServer(t)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar: %v", err)
	}
	client := &http.Client{Jar: jar}

	// 1) credential ผิดต้องโดน 401 และไม่ได้ cookie ติดมือกลับมา
	badResp, err := client.Post(ts.URL+"/login", "application/json", bytes.NewBufferString(`{"username":"admin","password":"wrong"}`))
	if err != nil {
		t.Fatalf("POST /login (bad): %v", err)
	}
	badResp.Body.Close()
	if badResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for bad credentials, got %d", badResp.StatusCode)
	}

	meResp, err := client.Get(ts.URL + "/me")
	if err != nil {
		t.Fatalf("GET /me (before login): %v", err)
	}
	meResp.Body.Close()
	if meResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 before login, got %d", meResp.StatusCode)
	}

	// 2) credential ถูกต้องต้องได้ cookie แล้ว /me ต้องเห็น username ตรงกัน
	goodResp, err := client.Post(ts.URL+"/login", "application/json", bytes.NewBufferString(`{"username":"admin","password":"changeme"}`))
	if err != nil {
		t.Fatalf("POST /login (good): %v", err)
	}
	goodResp.Body.Close()
	if goodResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for good credentials, got %d", goodResp.StatusCode)
	}

	meResp2, err := client.Get(ts.URL + "/me")
	if err != nil {
		t.Fatalf("GET /me (after login): %v", err)
	}
	defer meResp2.Body.Close()
	if meResp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 after login, got %d", meResp2.StatusCode)
	}
	var me struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(meResp2.Body).Decode(&me); err != nil {
		t.Fatalf("decode /me: %v", err)
	}
	if me.Username != "admin" {
		t.Errorf("expected username %q, got %q", "admin", me.Username)
	}

	// 3) logout แล้ว /me ต้อง 401 อีกครั้ง
	logoutResp, err := client.Post(ts.URL+"/logout", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /logout: %v", err)
	}
	logoutResp.Body.Close()

	meResp3, err := client.Get(ts.URL + "/me")
	if err != nil {
		t.Fatalf("GET /me (after logout): %v", err)
	}
	meResp3.Body.Close()
	if meResp3.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 after logout, got %d", meResp3.StatusCode)
	}
}

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

// TestScheduleValidation_RejectsBadRequestsBeforeTouchingTemporal ทดสอบ path ที่
// schedule_handler.go ต้อง validate request ก่อนแตะ Temporal client เสมอ — สำคัญเพราะ
// newTestServer ในไฟล์นี้ไม่มี Temporal client จริงให้ต่อ (temporal เป็น nil) ทำให้
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
