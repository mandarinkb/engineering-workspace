package workflow_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bop/internal/bot"
	"bop/internal/job"
	memlogger "bop/internal/logger/memory"
	memrepo "bop/internal/repository/memory"
	"bop/internal/worker"
	"bop/internal/workflow"
)

// alwaysSucceedBot คือ bot ปลอมสำหรับทดสอบที่ทำงานสำเร็จเสมอ ไม่ทำ side effect จริงจัง
type alwaysSucceedBot struct{ name string }

func (b *alwaysSucceedBot) Name() string { return b.name }
func (b *alwaysSucceedBot) Run(_ context.Context, log bot.LogFunc, _ bot.Config) (bot.Result, error) {
	log("ok")
	return bot.Result{Summary: "ok"}, nil
}

// alwaysFailBot คือ bot ปลอมสำหรับทดสอบที่ล้มเหลวเสมอ — ใช้พิสูจน์ว่า Runner หยุด
// workflow ทันทีเมื่อเจอ step ที่พัง ไม่ใช่พยายามรัน step ที่เหลือต่อ
type alwaysFailBot struct{ name string }

func (b *alwaysFailBot) Name() string { return b.name }
func (b *alwaysFailBot) Run(_ context.Context, _ bot.LogFunc, _ bot.Config) (bot.Result, error) {
	return bot.Result{}, fmt.Errorf("intentional failure")
}

// newTestRunner ประกอบ Runner พร้อม bot ปลอมทั้งสองตัวไว้ให้ test แต่ละอันเรียกใช้ซ้ำได้
func newTestRunner(t *testing.T) *workflow.Runner {
	t.Helper()
	jobRepo := memrepo.NewJobRepository()
	log := memlogger.New()
	pool := worker.NewPool(4, 5*time.Second, jobRepo, log)
	pool.Register(&alwaysSucceedBot{name: "ok-bot"})
	pool.Register(&alwaysFailBot{name: "fail-bot"})

	runRepo := memrepo.NewRunRepository()
	return workflow.NewRunner(pool, runRepo)
}

// TestRunner_AllStepsSucceed ทดสอบว่า workflow ที่ทุก step สำเร็จ จบด้วยสถานะ succeeded
// และมี StepResult ครบตามจำนวน step ที่กำหนดไว้
func TestRunner_AllStepsSucceed(t *testing.T) {
	runner := newTestRunner(t)

	wf := workflow.Workflow{
		Name: "two-ok-steps",
		Steps: []workflow.Step{
			{BotName: "ok-bot"},
			{BotName: "ok-bot"},
		},
	}

	run, err := runner.Trigger(context.Background(), wf)
	if err != nil {
		t.Fatalf("Trigger returned error: %v", err)
	}
	if run.Status != workflow.RunSucceeded {
		t.Fatalf("expected status succeeded, got %s", run.Status)
	}
	if len(run.Steps) != 2 {
		t.Fatalf("expected 2 step results, got %d", len(run.Steps))
	}
	for _, s := range run.Steps {
		if s.Status != job.StatusSucceeded {
			t.Errorf("step %s: expected succeeded, got %s", s.JobID, s.Status)
		}
	}
}

// TestRunner_StopsAtFirstFailedStep ทดสอบว่า workflow ที่ step กลางล้มเหลว จะหยุดทันที
// ไม่รัน step ถัดไปต่อ และสถานะสุดท้ายต้องเป็น failed
func TestRunner_StopsAtFirstFailedStep(t *testing.T) {
	runner := newTestRunner(t)

	wf := workflow.Workflow{
		Name: "fail-in-the-middle",
		Steps: []workflow.Step{
			{BotName: "ok-bot"},
			{BotName: "fail-bot"},
			{BotName: "ok-bot"}, // step นี้ไม่ควรถูกรันเลย
		},
	}

	run, err := runner.Trigger(context.Background(), wf)
	if err == nil {
		t.Fatal("expected Trigger to return an error when a step fails")
	}
	if run.Status != workflow.RunFailed {
		t.Fatalf("expected status failed, got %s", run.Status)
	}
	// ต้องมีผลลัพธ์แค่ 2 step (ok-bot ที่สำเร็จ + fail-bot ที่ล้มเหลว) ไม่ใช่ 3 —
	// พิสูจน์ว่า step ที่ 3 ไม่ถูกรันเลยจริงๆ
	if len(run.Steps) != 2 {
		t.Fatalf("expected exactly 2 step results (workflow should stop after failure), got %d", len(run.Steps))
	}
	if run.Steps[1].Status != job.StatusFailed {
		t.Errorf("expected second step to be failed, got %s", run.Steps[1].Status)
	}
}
