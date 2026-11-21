package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"bop/internal/pkg/jwt"
)

// dataEndpoints คือ endpoint ที่อ่านหรือเปลี่ยนข้อมูลทุกตัว ณ ตอนเขียน test — body ตั้งใจให้
// ผิดหรือชี้ไปที่ id ที่ไม่มีอยู่ ถ้าวันหนึ่งการบังคับ login หลุด request ก็จะไม่สร้างหรือลบ
// ข้อมูลจริง
var dataEndpoints = []struct {
	method string
	path   string
}{
	{http.MethodGet, "/me"},
	{http.MethodGet, "/bots"},
	{http.MethodPost, "/bots/stub-bot/jobs"},
	{http.MethodGet, "/jobs"},
	{http.MethodGet, "/jobs/missing"},
	{http.MethodGet, "/jobs/missing/logs"},
	{http.MethodGet, "/jobs/missing/logs/stream"},
	{http.MethodDelete, "/jobs/missing"},
	{http.MethodGet, "/workflow-runs"},
	{http.MethodGet, "/workflow-runs/missing"},
	{http.MethodGet, "/workflow-runs/missing/steps/0/logs"},
	{http.MethodPost, "/workflow-runs"},
	{http.MethodPost, "/schedules"},
	{http.MethodGet, "/schedules"},
	{http.MethodGet, "/schedules/missing"},
	{http.MethodPut, "/schedules/missing"},
	{http.MethodDelete, "/schedules/missing"},
	{http.MethodPost, "/schedules/missing/pause"},
	{http.MethodPost, "/schedules/missing/unpause"},
	{http.MethodPost, "/schedules/missing/trigger"},
	{http.MethodPost, "/schedules/missing/backfill"},
	// route ที่ยังไม่มีใครลงทะเบียนก็ต้อง 401 ก่อน — พิสูจน์ว่า route ใหม่ถูกบังคับ login
	// เป็นค่าเริ่มต้น ไม่ต้อง opt-in ทีละ route
	{http.MethodGet, "/workflow-runs/count"},
}

func doRequest(t *testing.T, method, url, cookie string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, url, bytes.NewBufferString(`{}`))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func TestRequireSession_RejectsDataEndpointsWithoutValidSession(t *testing.T) {
	ts := newAnonymousTestServer(t)

	expired, err := jwt.Issue([]byte(testSecret), testUsername, -time.Minute)
	if err != nil {
		t.Fatalf("issue expired token: %v", err)
	}
	forged, err := jwt.Issue([]byte("other-secret"), testUsername, time.Hour)
	if err != nil {
		t.Fatalf("issue forged token: %v", err)
	}

	sessions := []struct {
		name   string
		cookie string
	}{
		{"no cookie", ""},
		{"malformed token", sessionCookieName + "=not-a-jwt"},
		{"expired token", sessionCookieName + "=" + expired},
		{"wrong signature", sessionCookieName + "=" + forged},
	}

	for _, s := range sessions {
		for _, e := range dataEndpoints {
			t.Run(s.name+"/"+e.method+" "+e.path, func(t *testing.T) {
				resp := doRequest(t, e.method, ts.URL+e.path, s.cookie)
				if resp.StatusCode != http.StatusUnauthorized {
					t.Fatalf("expected 401, got %d", resp.StatusCode)
				}
			})
		}
	}
}

func TestRequireSession_AllowsDataEndpointsWithValidSession(t *testing.T) {
	ts := newAnonymousTestServer(t)
	cookie := login(t, ts.URL)

	resp := doRequest(t, http.MethodGet, ts.URL+"/jobs", cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /jobs: expected 200, got %d", resp.StatusCode)
	}

	createReq, err := http.NewRequest(http.MethodPost, ts.URL+"/bots/stub-bot/jobs",
		bytes.NewBufferString(`{"url":"https://example.com"}`))
	if err != nil {
		t.Fatalf("build create request: %v", err)
	}
	createReq.Header.Set("Cookie", cookie)
	createResp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		t.Fatalf("POST job: %v", err)
	}
	defer createResp.Body.Close()
	if createResp.StatusCode != http.StatusAccepted {
		t.Fatalf("POST job: expected 202, got %d", createResp.StatusCode)
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("decode job: %v", err)
	}

	// SSE ใช้ cookie เดียวกัน (EventSource ส่ง cookie ด้วย withCredentials)
	streamReq, err := http.NewRequest(http.MethodGet, ts.URL+"/jobs/"+created.ID+"/logs/stream", nil)
	if err != nil {
		t.Fatalf("build stream request: %v", err)
	}
	streamReq.Header.Set("Cookie", cookie)
	client := &http.Client{Timeout: 2 * time.Second}
	streamResp, err := client.Do(streamReq)
	if err != nil {
		t.Fatalf("GET stream: %v", err)
	}
	defer streamResp.Body.Close()
	if streamResp.StatusCode != http.StatusOK {
		t.Fatalf("GET stream: expected 200, got %d", streamResp.StatusCode)
	}
	if ct := streamResp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("GET stream: expected text/event-stream, got %q", ct)
	}
}

func TestRequireSession_PublicEndpointsWorkWithoutSession(t *testing.T) {
	ts := newAnonymousTestServer(t)

	tests := []struct {
		method string
		path   string
		want   int
	}{
		{http.MethodGet, "/healthz", http.StatusOK},
		{http.MethodGet, "/readyz", http.StatusOK},
		{http.MethodPost, "/logout", http.StatusNoContent},
		{http.MethodOptions, "/jobs", http.StatusNoContent},
		{http.MethodOptions, "/schedules/missing", http.StatusNoContent},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			resp := doRequest(t, tt.method, ts.URL+tt.path, "")
			if resp.StatusCode != tt.want {
				t.Fatalf("expected %d, got %d", tt.want, resp.StatusCode)
			}
		})
	}

	// /login ต้องเรียกได้โดยไม่มี session — credential ผิดได้ 401 จาก handler เอง ส่วน
	// credential ถูกได้ 204 (ถ้าโดนบังคับ login ก็จะไม่มีทางได้ 204)
	if c := login(t, ts.URL); c == "" {
		t.Fatal("expected a session cookie from POST /login")
	}
}

// login เรียก POST /login แล้วคืนค่า header Cookie ที่ใช้ส่งต่อได้
func login(t *testing.T, baseURL string) string {
	t.Helper()
	body := `{"username":"` + testUsername + `","password":"` + testPassword + `"}`
	resp, err := http.Post(baseURL+"/login", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("POST /login: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("POST /login: expected 204, got %d", resp.StatusCode)
	}
	for _, c := range resp.Cookies() {
		if c.Name == sessionCookieName {
			return c.Name + "=" + c.Value
		}
	}
	return ""
}
