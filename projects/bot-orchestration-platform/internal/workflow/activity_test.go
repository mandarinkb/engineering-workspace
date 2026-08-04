package workflow_test

import (
	"context"
	"errors"
	"testing"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"

	"bop/internal/bot"
	memlogger "bop/internal/logger/memory"
	"bop/internal/workflow"
)

// stubBot คือ bot ปลอมสำหรับทดสอบ Activities.RunStep โดยเฉพาะ (ต่างจาก
// alwaysSucceedBot/alwaysFailBot เดิมของ runner_test.go ที่ถูกลบไปแล้วตรงที่ stubBot นี้
// กำหนด error ที่จะคืนได้อิสระ ทำให้ทดสอบทั้งกรณี error ธรรมดาและ bot.ConfigError ได้
// ด้วย struct เดียว)
type stubBot struct {
	name   string
	result bot.Result
	err    error
}

func (b *stubBot) Name() string { return b.name }
func (b *stubBot) Run(_ context.Context, log bot.LogFunc, _ bot.Config) (bot.Result, error) {
	log("stub bot running")
	return b.result, b.err
}

// newActivityTestEnv สร้าง TestActivityEnvironment พร้อม register RunStep ของ activities
// ที่ส่งเข้ามาไว้แล้ว ภายใต้ชื่อ workflow.ActivityName เดียวกับที่ cmd/bop-worker ใช้จริง
func newActivityTestEnv(t *testing.T, activities *workflow.Activities) *testsuite.TestActivityEnvironment {
	t.Helper()
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestActivityEnvironment()
	env.RegisterActivityWithOptions(activities.RunStep, activity.RegisterOptions{Name: workflow.ActivityName})
	return env
}

// TestRunStep_Success ทดสอบเส้นทางปกติ: bot ทำงานสำเร็จ RunStep ต้องคืน Summary ของ
// bot.Result นั้นตรงๆ (พิสูจน์ว่า Activities เป็นแค่ adapter บางๆ ไม่ได้แก้ไขผลลัพธ์ของ
// bot เลย ตามที่ตั้งใจไว้)
func TestRunStep_Success(t *testing.T) {
	activities := workflow.NewActivities(memlogger.New())
	activities.Register(&stubBot{name: "ok-bot", result: bot.Result{Summary: "done"}})
	env := newActivityTestEnv(t, activities)

	val, err := env.ExecuteActivity(activities.RunStep, workflow.Step{BotName: "ok-bot"})
	if err != nil {
		t.Fatalf("ExecuteActivity: %v", err)
	}
	var summary string
	if err := val.Get(&summary); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if summary != "done" {
		t.Errorf("expected summary %q, got %q", "done", summary)
	}
}

// TestRunStep_ConfigErrorIsNonRetryable คือ test สำคัญที่สุดของไฟล์นี้ — พิสูจน์ว่าเมื่อ
// bot คืน bot.ConfigError กลับมา RunStep ต้องห่อมันเป็น temporal.ApplicationError ที่
// Type() == "ConfigError" จริง (ตรงกับ nonRetryableErrorTypes ใน temporal.go เป๊ะ) ถ้า
// test นี้ไม่ผ่าน แปลว่า Temporal จะ retry error ที่ retry ไปเท่าไหร่ก็ไม่มีทางสำเร็จ
// ซ้ำไปเรื่อยๆ โดยไม่ตั้งใจ — ดูคำอธิบายเหตุผลเต็มๆ ที่ activity.go
func TestRunStep_ConfigErrorIsNonRetryable(t *testing.T) {
	activities := workflow.NewActivities(memlogger.New())
	activities.Register(&stubBot{name: "bad-config-bot", err: bot.NewConfigError(errors.New("missing url"))})
	env := newActivityTestEnv(t, activities)

	_, err := env.ExecuteActivity(activities.RunStep, workflow.Step{BotName: "bad-config-bot"})
	if err == nil {
		t.Fatal("expected error")
	}

	var appErr *temporal.ApplicationError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected error chain to contain *temporal.ApplicationError, got %T: %v", err, err)
	}
	if appErr.Type() != "ConfigError" {
		t.Errorf("expected ApplicationError Type %q, got %q", "ConfigError", appErr.Type())
	}
}

// TestRunStep_UnknownBot ทดสอบว่า step ที่อ้างชื่อ bot ที่ไม่ได้ register ไว้เลย คืน error
// ทันที (ไม่ panic ไม่ค้าง) — ไม่ใช่ ConfigError เพราะปัญหานี้ไม่เกี่ยวกับ config ของ user
func TestRunStep_UnknownBot(t *testing.T) {
	activities := workflow.NewActivities(memlogger.New())
	env := newActivityTestEnv(t, activities)

	_, err := env.ExecuteActivity(activities.RunStep, workflow.Step{BotName: "does-not-exist"})
	if err == nil {
		t.Fatal("expected error for unknown bot")
	}
}
