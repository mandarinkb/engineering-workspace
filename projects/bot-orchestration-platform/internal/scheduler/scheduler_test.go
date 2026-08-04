package scheduler_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"bop/internal/bot"
	memlogger "bop/internal/logger/memory"
	memrepo "bop/internal/repository/memory"
	"bop/internal/scheduler"
	"bop/internal/worker"
	"bop/internal/workflow"
)

// countingBot นับจำนวนครั้งที่ถูกเรียก Run แบบปลอดภัยข้าม goroutine ด้วย atomic —
// ใช้พิสูจน์ว่า Scheduler เรียก workflow ซ้ำๆ ตาม schedule จริง ไม่ใช่แค่ครั้งเดียว
type countingBot struct{ count *int64 }

func (b *countingBot) Name() string { return "counting-bot" }
func (b *countingBot) Run(_ context.Context, _ bot.LogFunc, _ bot.Config) (bot.Result, error) {
	atomic.AddInt64(b.count, 1)
	return bot.Result{Summary: "tick"}, nil
}

// TestScheduler_TriggersWorkflowRepeatedlyWhenDue ทดสอบว่า Scheduler เรียก workflow
// ซ้ำๆ ตาม Interval ที่กำหนดจริง (ไม่ใช่แค่ครั้งเดียวตอน register) — ใช้ tick และ
// interval สั้นมาก (หลัก 10-30 มิลลิวินาที) เพื่อให้ test รันเสร็จเร็ว ไม่ต้องรอนาน
func TestScheduler_TriggersWorkflowRepeatedlyWhenDue(t *testing.T) {
	var count int64

	jobRepo := memrepo.NewJobRepository()
	log := memlogger.New()
	pool := worker.NewPool(4, time.Second, jobRepo, log)
	pool.Register(&countingBot{count: &count})

	runRepo := memrepo.NewRunRepository()
	runner := workflow.NewRunner(pool, runRepo)

	sched := scheduler.New(runner, 10*time.Millisecond) // เช็คถี่มากเพื่อให้ test เร็ว
	sched.Register(workflow.Workflow{
		Name:  "tick-workflow",
		Steps: []workflow.Step{{BotName: "counting-bot"}},
	}, scheduler.Interval{Every: 30 * time.Millisecond})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go sched.Start(ctx)

	<-ctx.Done()
	time.Sleep(50 * time.Millisecond) // ให้เวลา trigger รอบสุดท้ายทำงานจบก่อนเช็คผล

	got := atomic.LoadInt64(&count)
	if got < 3 {
		t.Fatalf("expected at least 3 triggers in ~200ms with a 30ms interval, got %d", got)
	}
}
