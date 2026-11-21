package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apihttp "bop/internal/adapter/http"
	"bop/internal/adapter/worker"
	"bop/internal/domain/bot"
	"bop/internal/domain/workflow"
	"bop/internal/external/temporal"
	memlogger "bop/internal/pkg/logger"
	"bop/internal/repository/memory"
	authuc "bop/internal/usecase/auth"
	jobuc "bop/internal/usecase/job"
	scheduleuc "bop/internal/usecase/schedule"
	workflowrunuc "bop/internal/usecase/workflowrun"
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

// newTestServer คือ server ที่ "login แล้ว" — แนบ cookie bop_session ที่ valid ให้ทุก
// request ที่ยังไม่มี cookie นี้เอง เพราะทุก endpoint ข้อมูลต้องมี session (ticket 12)
// test ของ endpoint ทั่วไปจึงเรียก http.Get/http.Post ตรง ๆ ได้โดยไม่ต้องจัดการ login เอง
// test ที่อยากเห็นพฤติกรรมตอนไม่มี session ให้ใช้ newAnonymousTestServer แทน
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv, auth := newAPIServer(t)
	ts := httptest.NewServer(withSession(t, auth, srv))
	t.Cleanup(ts.Close)
	return ts
}

// withSession ห่อ handler ให้แนบ cookie bop_session ที่ valid (ออกโดย auth) ให้ทุก request
// ที่ยังไม่มี cookie นี้ — test ที่ประกอบ Server เอง (เช่นใส่ engine ปลอม) ใช้ห่อ srv ก่อนส่ง
// เข้า httptest.NewServer เพื่อให้ผ่านการบังคับ login
func withSession(t *testing.T, auth *authuc.Service, srv http.Handler) http.Handler {
	t.Helper()
	token, err := auth.Login(testUsername, testPassword)
	if err != nil {
		t.Fatalf("issue test session: %v", err)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := r.Cookie(sessionCookieName); err != nil {
			r.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
		}
		srv.ServeHTTP(w, r)
	})
}

// newAnonymousTestServer คือ server ตัวเดียวกับ cmd/bop/main.go โดยไม่แนบ session ให้ —
// client ต้อง login เองถ้าต้องการ (ใช้ทดสอบการบังคับ login)
func newAnonymousTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv, _ := newAPIServer(t)
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	return ts
}

// healthyEngine ห่อ engine ที่ Temporal client เป็น nil ให้ CheckHealth ผ่านเสมอ —
// /readyz จึงเรียกได้ใน test โดยไม่ panic (method อื่นยังเป็นของ engine เดิมทั้งหมด)
type healthyEngine struct {
	workflow.Engine
}

func (healthyEngine) CheckHealth(context.Context) error { return nil }

const (
	testUsername      = "admin"
	testPassword      = "changeme"
	testSecret        = "test-secret"
	sessionCookieName = "bop_session"
)

// newAPIServer ประกอบ Server เต็มรูปแบบเหมือน cmd/bop/main.go ทุกประการ ยกเว้น
// Temporal client ที่ปล่อยเป็น nil ไปเลย (ห่ออยู่ใน temporal.Engine/temporal.Scheduler) —
// ปลอดภัยเพราะ test ใน package นี้แตะ endpoint กลุ่ม /workflow-runs และ /schedules เฉพาะ path ที่
// usecase validate fail ก่อนเรียก engine เท่านั้น (path ที่ต้องมี Temporal server รันอยู่จริง
// ทดสอบแยกต่างหากแบบ end-to-end ไม่ใช่ unit test ระดับนี้)
func newAPIServer(t *testing.T) (*apihttp.Server, *authuc.Service) {
	t.Helper()
	repo := memory.NewJobRepository()
	log := memlogger.New()
	pool := worker.NewPool(2, 5*time.Second, repo, log)
	pool.Register(stubBot{})

	auth := authuc.NewService([]byte(testSecret), testUsername, testPassword)
	srv := apihttp.NewServer(
		jobuc.NewService(repo, pool, log),
		workflowrunuc.NewService(healthyEngine{Engine: temporal.NewEngine(nil)}),
		scheduleuc.NewService(temporal.NewScheduler(nil)),
		auth,
	)
	return srv, auth
}
