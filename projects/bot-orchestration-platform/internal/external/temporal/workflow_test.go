package temporal_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/activity"
	sdktemporal "go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"

	"bop/internal/domain/workflow"
	"bop/internal/external/temporal"
	memlogger "bop/internal/pkg/logger"
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
	env.RegisterWorkflow(temporal.Execute)

	activities := temporal.NewActivities(memlogger.New())
	env.RegisterActivityWithOptions(activities.RunStep, activity.RegisterOptions{Name: temporal.ActivityName})
	return env
}

// TestExecute_AllStepsSucceed ทดสอบว่า workflow ที่ทุก step สำเร็จ จบโดยไม่มี error
// คืนผลลัพธ์ทีละ step ครบตามจำนวน และ query "steps" ให้ผลลัพธ์เดียวกับที่ได้จาก
// GetWorkflowResult (พิสูจน์ว่า query handler เห็นผลลัพธ์เดียวกับที่ workflow คืนจริง —
// แทนที่พฤติกรรมของ workflow.RunRepository เดิมที่เก็บ Run.Steps ไว้ให้ query ทีหลังได้)
func TestExecute_AllStepsSucceed(t *testing.T) {
	env := newTestEnv(t)
	env.OnActivity(temporal.ActivityName, mock.Anything, mock.Anything).Return("ok", nil)

	wf := workflow.Workflow{
		Name: "two-ok-steps",
		Steps: []workflow.Step{
			{BotName: "ok-bot"},
			{BotName: "ok-bot"},
		},
	}
	env.ExecuteWorkflow(temporal.Execute, wf)

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
	env.OnActivity(temporal.ActivityName, mock.Anything, workflow.Step{BotName: "ok-bot"}).
		Return("ok", nil).Times(1)
	env.OnActivity(temporal.ActivityName, mock.Anything, workflow.Step{BotName: "fail-bot"}).
		Return("", errors.New("intentional failure")).Times(3) // ครบ 3 Attempt ตามค่าเริ่มต้น

	wf := workflow.Workflow{
		Name: "fail-in-the-middle",
		Steps: []workflow.Step{
			{BotName: "ok-bot"},
			{BotName: "fail-bot"},
			{BotName: "ok-bot"}, // ไม่ควรถูกเรียกเลย
		},
	}
	env.ExecuteWorkflow(temporal.Execute, wf)

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
	env.OnActivity(temporal.ActivityName, mock.Anything, mock.Anything).Return("ok", nil)

	wf := workflow.Workflow{
		Name: "step-with-remote-task-queue",
		Steps: []workflow.Step{
			{BotName: "remote-bot", TaskQueue: "external-job-queue"},
		},
	}
	env.ExecuteWorkflow(temporal.Execute, wf)

	if !env.IsWorkflowCompleted() {
		t.Fatal("expected workflow to complete")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// intPtr ช่วยสร้างค่า Step.MaxAttempts (เป็น pointer เพื่อแยก "ไม่ระบุ" ออกจาก 0)
func intPtr(n int) *int { return &n }

// TestExecute_StepFailsAfterDefaultThreeAttempts ทดสอบว่า Step ที่ error ทุก Attempt และไม่ได้
// ตั้ง max_attempts กลายเป็น Failed Step หลังครบ 3 Attempt พอดี — ก่อนหน้านี้ RetryPolicy ไม่มี
// MaximumAttempts ทำให้ Step ถูกลองใหม่ไม่มีที่สิ้นสุดจน Workflow Run ค้าง running
func TestExecute_StepFailsAfterDefaultThreeAttempts(t *testing.T) {
	env := newTestEnv(t)
	env.OnActivity(temporal.ActivityName, mock.Anything, mock.Anything).
		Return("", errors.New("connection refused")).Times(3)

	wf := workflow.Workflow{
		Name:  "always-fails",
		Steps: []workflow.Step{{BotName: "fail-bot"}},
	}
	env.ExecuteWorkflow(temporal.Execute, wf)

	if !env.IsWorkflowCompleted() {
		t.Fatal("expected workflow to complete (with error)")
	}
	if err := env.GetWorkflowError(); err == nil {
		t.Fatal("expected workflow to fail once the step runs out of attempts")
	}
	env.AssertExpectations(t)
}

// TestExecute_StepFailsAfterMaxAttempts ทดสอบว่า max_attempts ที่ตั้งต่อ Step ใช้แทนค่าเริ่มต้น 3
// ได้ทั้งน้อยกว่าและมากกว่า
func TestExecute_StepFailsAfterMaxAttempts(t *testing.T) {
	tests := []struct {
		name         string
		giveAttempts int
	}{
		{name: "fewer than default", giveAttempts: 1},
		{name: "more than default", giveAttempts: 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			env.OnActivity(temporal.ActivityName, mock.Anything, mock.Anything).
				Return("", errors.New("connection refused")).Times(tt.giveAttempts)

			wf := workflow.Workflow{
				Name: "always-fails",
				Steps: []workflow.Step{
					{BotName: "fail-bot", MaxAttempts: intPtr(tt.giveAttempts)},
				},
			}
			env.ExecuteWorkflow(temporal.Execute, wf)

			if err := env.GetWorkflowError(); err == nil {
				t.Fatal("expected workflow to fail once the step runs out of attempts")
			}
			env.AssertExpectations(t)
		})
	}
}

// TestExecute_NonRetryableErrorFailsOnFirstAttempt ทดสอบว่า ConfigError และ UnknownBot
// ล้มเหลวใน Attempt แรกเสมอ แม้ max_attempts จะตั้งไว้สูง เพราะลองใหม่กี่ครั้งก็ได้ผลเดิม
func TestExecute_NonRetryableErrorFailsOnFirstAttempt(t *testing.T) {
	for _, errType := range []string{"ConfigError", "UnknownBot"} {
		t.Run(errType, func(t *testing.T) {
			env := newTestEnv(t)
			env.OnActivity(temporal.ActivityName, mock.Anything, mock.Anything).
				Return("", sdktemporal.NewApplicationError("broken", errType)).Times(1)

			wf := workflow.Workflow{
				Name:  "non-retryable",
				Steps: []workflow.Step{{BotName: "bad-bot", MaxAttempts: intPtr(5)}},
			}
			env.ExecuteWorkflow(temporal.Execute, wf)

			if err := env.GetWorkflowError(); err == nil {
				t.Fatal("expected workflow to fail")
			}
			env.AssertExpectations(t)
		})
	}
}

// TestExecute_StepAfterFailedStepDoesNotRun ทดสอบว่าเมื่อ Step แรกกลายเป็น Failed Step หลัง
// หมด Attempt แล้ว Step ถัดไปไม่ถูกรันเลย และ Workflow Run ล้มเหลว
func TestExecute_StepAfterFailedStepDoesNotRun(t *testing.T) {
	env := newTestEnv(t)
	env.OnActivity(temporal.ActivityName, mock.Anything, workflow.Step{BotName: "fail-bot"}).
		Return("", errors.New("connection refused")).Times(3)

	wf := workflow.Workflow{
		Name: "fail-first",
		Steps: []workflow.Step{
			{BotName: "fail-bot"},
			{BotName: "ok-bot"}, // ไม่ควรถูกเรียกเลย (ไม่มี mock ถ้าถูกเรียก testsuite จะ panic)
		},
	}
	env.ExecuteWorkflow(temporal.Execute, wf)

	if err := env.GetWorkflowError(); err == nil {
		t.Fatal("expected workflow to fail")
	}
	var queried []workflow.StepResult
	val, err := env.QueryWorkflow("steps")
	if err != nil {
		t.Fatalf("QueryWorkflow: %v", err)
	}
	if err := val.Get(&queried); err != nil {
		t.Fatalf("decode query result: %v", err)
	}
	if len(queried) != 1 || queried[0].Error == "" {
		t.Fatalf("expected exactly 1 failed step result, got %+v", queried)
	}
	env.AssertExpectations(t)
}

// TestExecute_KubernetesJobStepFailsAfterMaxAttempts ทดสอบว่าจำนวน Attempt สูงสุดใช้กับ activity
// ของ Kubernetes Job ด้วย ไม่ใช่แค่ activity ของ Bot ปกติ — register function แทนไว้ใต้ชื่อ
// K8sJobActivityName เพื่อให้ OnActivity แบบ string mock ได้โดยไม่ต้องมี cluster
func TestExecute_KubernetesJobStepFailsAfterMaxAttempts(t *testing.T) {
	tests := []struct {
		name         string
		giveAttempts *int
		wantCalls    int
	}{
		{name: "default", giveAttempts: nil, wantCalls: 3},
		{name: "configured", giveAttempts: intPtr(2), wantCalls: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			env.RegisterActivityWithOptions(
				func(context.Context, workflow.K8sJobSpec) (string, error) { return "", nil },
				activity.RegisterOptions{Name: temporal.K8sJobActivityName},
			)
			env.OnActivity(temporal.K8sJobActivityName, mock.Anything, mock.Anything).
				Return("", errors.New("pod failed")).Times(tt.wantCalls)

			wf := workflow.Workflow{
				Name: "k8s-fails",
				Steps: []workflow.Step{{
					K8sJob:      &workflow.K8sJobSpec{Image: "busybox"},
					MaxAttempts: tt.giveAttempts,
				}},
			}
			env.ExecuteWorkflow(temporal.Execute, wf)

			if err := env.GetWorkflowError(); err == nil {
				t.Fatal("expected workflow to fail")
			}
			env.AssertExpectations(t)
		})
	}
}
