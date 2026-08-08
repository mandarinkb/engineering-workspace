package workflow_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"

	memlogger "bop/internal/logger/memory"
	"bop/internal/workflow"
)

// newTestEnv สร้าง TestWorkflowEnvironment พร้อม register ActivityName ไว้แล้ว (ต้อง
// register ก่อนถึงจะ OnActivity(ชื่อแบบ string, ...) ได้ ดูคำอธิบายใน
// go.temporal.io/sdk/internal/workflow_testsuite.go — OnActivity แบบ string เช็คว่า
// ชื่อนั้น register ไว้แล้วจริง) — ทุก test ในไฟล์นี้ mock พฤติกรรมของ Activity ผ่าน
// OnActivity แทนที่จะปล่อยให้ Activities.RunStep รันจริง เพราะสิ่งที่ต้องการทดสอบตรงนี้
// คือ "workflow orchestration logic" (Execute เรียก step ตามลำดับถูกไหม หยุดถูกจังหวะไหม)
// ไม่ใช่ตัว bot จริง — การแปลง bot.ConfigError เป็น non-retryable ApplicationError ของ
// Activities.RunStep เองมี test แยกต่างหากที่ activity_test.go
func newTestEnv(t *testing.T) *testsuite.TestWorkflowEnvironment {
	t.Helper()
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(workflow.Execute)

	activities := workflow.NewActivities(memlogger.New())
	env.RegisterActivityWithOptions(activities.RunStep, activity.RegisterOptions{Name: workflow.ActivityName})
	return env
}

// TestExecute_AllStepsSucceed ทดสอบว่า workflow ที่ทุก step สำเร็จ จบโดยไม่มี error
// คืนผลลัพธ์ทีละ step ครบตามจำนวน และ query "steps" ให้ผลลัพธ์เดียวกับที่ได้จาก
// GetWorkflowResult (พิสูจน์ว่า query handler เห็นผลลัพธ์เดียวกับที่ workflow คืนจริง —
// แทนที่พฤติกรรมของ workflow.RunRepository เดิมที่เก็บ Run.Steps ไว้ให้ query ทีหลังได้)
func TestExecute_AllStepsSucceed(t *testing.T) {
	env := newTestEnv(t)
	env.OnActivity(workflow.ActivityName, mock.Anything, mock.Anything).Return("ok", nil)

	wf := workflow.Workflow{
		Name: "two-ok-steps",
		Steps: []workflow.Step{
			{BotName: "ok-bot"},
			{BotName: "ok-bot"},
		},
	}
	env.ExecuteWorkflow(workflow.Execute, wf)

	if !env.IsWorkflowCompleted() {
		t.Fatal("expected workflow to complete")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var results []workflow.StepResult
	if err := env.GetWorkflowResult(&results); err != nil {
		t.Fatalf("GetWorkflowResult: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 step results, got %d", len(results))
	}
	for _, r := range results {
		if r.Summary != "ok" || r.Error != "" {
			t.Errorf("step %d: unexpected result %+v", r.StepIndex, r)
		}
	}

	var queried []workflow.StepResult
	val, err := env.QueryWorkflow("steps")
	if err != nil {
		t.Fatalf("QueryWorkflow: %v", err)
	}
	if err := val.Get(&queried); err != nil {
		t.Fatalf("decode query result: %v", err)
	}
	if len(queried) != len(results) {
		t.Fatalf("query \"steps\" returned %d results, want %d matching GetWorkflowResult", len(queried), len(results))
	}
}

// TestExecute_StopsAtFirstFailedStep ทดสอบว่า workflow ที่ step กลางล้มเหลว หยุดทันที
// ไม่รัน step ที่เหลือต่อ — เช็คผ่าน mock.Times(1) ต่อ step ว่า step ที่ 3 (ok-bot ตัวที่
// สอง) ไม่เคยถูกเรียกเลย ถ้าถูกเรียกจริง mock ของมันจะถูกเรียกเกิน Times(1) ที่ตั้งไว้
// ให้ mock ของ ok-bot ตัวแรก แล้ว testify จะ panic ทันที ไม่ต้องเดาจากจำนวนผลลัพธ์เฉยๆ
func TestExecute_StopsAtFirstFailedStep(t *testing.T) {
	env := newTestEnv(t)
	env.OnActivity(workflow.ActivityName, mock.Anything, workflow.Step{BotName: "ok-bot"}).
		Return("ok", nil).Times(1)
	env.OnActivity(workflow.ActivityName, mock.Anything, workflow.Step{BotName: "fail-bot"}).
		Return("", errors.New("intentional failure")).Times(1)

	wf := workflow.Workflow{
		Name: "fail-in-the-middle",
		Steps: []workflow.Step{
			{BotName: "ok-bot"},
			{BotName: "fail-bot"},
			{BotName: "ok-bot"}, // ไม่ควรถูกเรียกเลย
		},
	}
	env.ExecuteWorkflow(workflow.Execute, wf)

	if !env.IsWorkflowCompleted() {
		t.Fatal("expected workflow to complete (with error)")
	}
	if err := env.GetWorkflowError(); err == nil {
		t.Fatal("expected workflow to return an error when a step fails")
	}

	var queried []workflow.StepResult
	val, err := env.QueryWorkflow("steps")
	if err != nil {
		t.Fatalf("QueryWorkflow: %v", err)
	}
	if err := val.Get(&queried); err != nil {
		t.Fatalf("decode query result: %v", err)
	}
	if len(queried) != 2 {
		t.Fatalf("expected exactly 2 step results (workflow should stop after failure), got %d", len(queried))
	}
	if queried[1].Error == "" {
		t.Errorf("expected second step result to record the failure, got %+v", queried[1])
	}
}

// TestExecute_StepWithTaskQueueStillExecutes เป็น regression test สำหรับ Step.TaskQueue —
// พิสูจน์แค่ว่า Execute ไม่ error/panic เมื่อ step ระบุ TaskQueue มา (ค่าอะไรก็ได้) ไม่ได้
// พิสูจน์ว่า activity ถูก route ไปยัง task queue นั้นจริงๆ เพราะ TestWorkflowEnvironment
// ของ Temporal SDK รัน activity ทุกตัวแบบ in-process เสมอ ไม่ได้ simulate การมี worker
// หลายตัวคนละ task queue จริง — การพิสูจน์ routing ข้าม task queue จริงต้องทดสอบแบบ
// end-to-end กับ Temporal server จริงและ worker อย่างน้อย 2 ตัว (ดู README.md หัวข้อ
// "รัน step ผ่าน remote Temporal worker")
func TestExecute_StepWithTaskQueueStillExecutes(t *testing.T) {
	env := newTestEnv(t)
	env.OnActivity(workflow.ActivityName, mock.Anything, mock.Anything).Return("ok", nil)

	wf := workflow.Workflow{
		Name: "step-with-remote-task-queue",
		Steps: []workflow.Step{
			{BotName: "remote-bot", TaskQueue: "external-job-queue"},
		},
	}
	env.ExecuteWorkflow(workflow.Execute, wf)

	if !env.IsWorkflowCompleted() {
		t.Fatal("expected workflow to complete")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
