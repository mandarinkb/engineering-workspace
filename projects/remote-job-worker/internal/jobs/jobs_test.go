// package jobs (ไม่ใช่ jobs_test) เพราะ TestRunLongTask_StopsWhenContextCancelled ต้อง
// เรียก runLongTask ที่ไม่ export ตรงๆ โดยไม่ผ่าน Temporal test environment เลย (ดู
// คอมเมนต์ที่ test นั้น) — pattern เดียวกับที่ internal/api/http/schedule_internal_test.go
// ของฝั่ง BOP ใช้ทดสอบ unexported function ตรงๆ
package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

// newTestEnv สร้าง TestActivityEnvironment พร้อม register activity function ที่จะทดสอบ
// ไว้ให้แล้ว (จำเป็นแม้จะเรียก ExecuteActivity ด้วย function reference ตรงๆ ก็ตาม — SDK
// ยัง require ให้ register ก่อนเสมอ ไม่งั้น panic "no activity is registered for
// taskqueue") — ต่างจาก TestWorkflowEnvironment ของฝั่ง BOP
// (internal/workflow/temporal_test.go) ตรงที่ตัวนี้ทดสอบ activity function เดี่ยวๆ
// โดยตรง ไม่ต้องมี workflow ห่ออยู่รอบนอกเลย
func newTestEnv(t *testing.T, activityFn interface{}) *testsuite.TestActivityEnvironment {
	t.Helper()
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestActivityEnvironment()
	env.RegisterActivity(activityFn)
	return env
}

// TestRunStep_Ping ทดสอบเส้นทางที่ง่ายที่สุด: jobcode "PING" ต้องสำเร็จทันที คืน "pong"
// — ใช้ยืนยันว่า wiring พื้นฐาน (RunStep รับ Step แล้ว dispatch ถูก case) ทำงานถูกต้อง
func TestRunStep_Ping(t *testing.T) {
	env := newTestEnv(t, RunStep)

	val, err := env.ExecuteActivity(RunStep, Step{Config: map[string]string{"jobcode": "PING"}})
	if err != nil {
		t.Fatalf("ExecuteActivity: %v", err)
	}
	var summary string
	if err := val.Get(&summary); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if summary != "pong" {
		t.Errorf("expected summary %q, got %q", "pong", summary)
	}
}

// TestRunStep_MissingOrUnknownJobcode ทดสอบว่า jobcode ที่ว่างเปล่าหรือไม่ตรงกับ case ไหน
// เลย ต้องคืน error ที่ห่อเป็น temporal.ApplicationError ชนิด "ConfigError" (non-retryable
// ตาม contract ที่ REMOTE-JOB-WORKER-TODO.md กำหนดไว้ — ต้องตรงกับ nonRetryableErrorTypes
// ที่ฝั่ง BOP ประกาศไว้ที่ internal/workflow/temporal.go เป๊ะ ไม่งั้น Temporal จะ retry
// jobcode ที่ผิดซ้ำไปเรื่อยๆ โดยไม่มีประโยชน์ เพราะ jobcode ผิดไม่มีทางเปลี่ยนเองระหว่าง retry)
func TestRunStep_MissingOrUnknownJobcode(t *testing.T) {
	cases := []struct {
		name    string
		jobcode string
	}{
		{"empty jobcode", ""},
		{"unknown jobcode", "NO_SUCH_JOBCODE"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t, RunStep)
			_, err := env.ExecuteActivity(RunStep, Step{Config: map[string]string{"jobcode": tc.jobcode}})
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
		})
	}
}

// TestRunStep_FailTransient ทดสอบว่า jobcode "FAIL_TRANSIENT" คืน error ธรรมดา (ไม่ใช่
// ApplicationError ชนิด ConfigError) — ต้อง retryable ตาม RetryPolicy default ของฝั่ง BOP
// เพราะจำลองปัญหาชั่วคราว ไม่ใช่ config ที่ผิดถาวร
func TestRunStep_FailTransient(t *testing.T) {
	env := newTestEnv(t, RunStep)

	_, err := env.ExecuteActivity(RunStep, Step{Config: map[string]string{"jobcode": "FAIL_TRANSIENT"}})
	if err == nil {
		t.Fatal("expected error")
	}
	var appErr *temporal.ApplicationError
	if errors.As(err, &appErr) && appErr.Type() == "ConfigError" {
		t.Fatalf("expected an ordinary (retryable) error, got non-retryable ConfigError: %v", err)
	}
}

// TestRunLongTask_CompletesAndSendsHeartbeats ทดสอบว่า runLongTask (jobcode "LONG_TASK"
// เรียกฟังก์ชันนี้อยู่ข้างใน) ทำงานจนจบและเรียก activity.RecordHeartbeat เป็นระยะจริง —
// เรียกผ่าน env.ExecuteActivity ตรงๆ (ไม่ผ่าน RunStep) พร้อมกำหนด total/heartbeatEvery
// สั้นๆ เอง เพื่อไม่ต้องรอ longTaskDuration (10 วินาที) จริงทุกครั้งที่รัน test
func TestRunLongTask_CompletesAndSendsHeartbeats(t *testing.T) {
	env := newTestEnv(t, runLongTask)

	var heartbeats int
	env.SetOnActivityHeartbeatListener(func(_ *activity.Info, _ converter.EncodedValues) {
		heartbeats++
	})

	val, err := env.ExecuteActivity(runLongTask, 350*time.Millisecond, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("ExecuteActivity: %v", err)
	}
	var summary string
	if err := val.Get(&summary); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if summary != "long_task completed" {
		t.Errorf("expected summary %q, got %q", "long_task completed", summary)
	}
	if heartbeats == 0 {
		t.Error("expected at least one heartbeat to be recorded")
	}
}

// TestRunLongTask_StopsWhenContextCancelled ทดสอบว่า runLongTask เช็ค ctx.Done() จริง
// และหยุดทันทีถ้า context ถูกยกเลิกไปแล้วก่อนเริ่มด้วยซ้ำ — เรียกฟังก์ชันตรงๆ แบบ plain Go
// (ไม่ผ่าน TestActivityEnvironment เลย) เพราะ ctx ที่ยกเลิกไปแล้วทำให้ select เลือก
// ctx.Done() case ทันทีตั้งแต่รอบแรก ไม่มีทางไปถึงบรรทัดที่เรียก activity.RecordHeartbeat
// เลย (ซึ่งต้องพึ่ง activity execution context ของจริงถึงจะเรียกได้โดยไม่ panic) — พิสูจน์
// ว่า long-running job หยุดทำงานทันทีถ้าถูกยกเลิก ไม่ทำงานต่อไปเรื่อยๆ โดยไม่สนใจ (contract
// เดียวกับที่ bot.Bot.Run ต้องทำฝั่ง BOP)
func TestRunLongTask_StopsWhenContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	_, err := runLongTask(ctx, 10*time.Second, 2*time.Second)
	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if elapsed > time.Second {
		t.Fatalf("expected runLongTask to return promptly after cancellation, took %s", elapsed)
	}
}
