package http_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	apihttp "bop/internal/adapter/http"
	"bop/internal/adapter/worker"
	"bop/internal/domain/bot"
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

// newTestServer ประกอบ Server เต็มรูปแบบเหมือน cmd/bop/main.go ทุกประการ ยกเว้น
// Temporal client ที่ปล่อยเป็น nil ไปเลย (ห่ออยู่ใน temporal.Engine/temporal.Scheduler) —
// ปลอดภัยเพราะ test ใน package นี้แตะ endpoint กลุ่ม /workflow-runs และ /schedules เฉพาะ path ที่
// usecase validate fail ก่อนเรียก engine เท่านั้น (path ที่ต้องมี Temporal server รันอยู่จริง
// ทดสอบแยกต่างหากแบบ end-to-end ไม่ใช่ unit test ระดับนี้)
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	repo := memory.NewJobRepository()
	log := memlogger.New()
	pool := worker.NewPool(2, 5*time.Second, repo, log)
	pool.Register(stubBot{})
	scheduleRepo := memory.NewScheduleRepository()

	srv := apihttp.NewServer(
		jobuc.NewService(repo, pool, log),
		workflowrunuc.NewService(temporal.NewEngine(nil)),
		scheduleuc.NewService(temporal.NewScheduler(nil), scheduleRepo),
		authuc.NewService([]byte("test-secret"), "admin", "changeme"),
	)
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	return ts
}
