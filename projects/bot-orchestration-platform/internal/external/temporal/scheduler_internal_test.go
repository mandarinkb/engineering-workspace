// package temporal (ไม่ใช่ temporal_test) เพราะ test ไฟล์นี้ทดสอบ unexported function
// (toScheduleSpec, fromScheduleSpec, overlapPolicyByName/overlapPolicyName, scheduledSteps) ตรงๆ โดยไม่
// ต้องพึ่ง Temporal client จริง — ย้ายมาจาก internal/api/http/schedule_internal_test.go เดิม
// ส่วนที่ทดสอบการ validate spec/overlap ย้ายไปอยู่ที่ internal/domain/schedule แล้ว
package temporal

import (
	"reflect"
	"testing"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"

	"bop/internal/domain/bot"
	"bop/internal/domain/schedule"
	wfdomain "bop/internal/domain/workflow"
)

func TestToScheduleSpec_Interval(t *testing.T) {
	seconds := int64(60)
	spec := toScheduleSpec(schedule.Spec{IntervalSeconds: &seconds})
	if len(spec.Intervals) != 1 || spec.Intervals[0].Every != time.Minute {
		t.Fatalf("expected 1 interval of 1 minute, got %+v", spec.Intervals)
	}
}

func TestToScheduleSpec_Cron(t *testing.T) {
	cron := "0 9 * * MON-FRI"
	spec := toScheduleSpec(schedule.Spec{CronExpression: &cron})
	if len(spec.CronExpressions) != 1 || spec.CronExpressions[0] != cron {
		t.Fatalf("expected cron expression %q, got %+v", cron, spec.CronExpressions)
	}
}

func TestFromScheduleSpec_RoundTrip(t *testing.T) {
	seconds := int64(120)
	sdkSpec := toScheduleSpec(schedule.Spec{IntervalSeconds: &seconds})
	got := fromScheduleSpec(&sdkSpec)
	if got.CronExpression != nil {
		t.Errorf("expected nil cron expression, got %v", *got.CronExpression)
	}
	if got.IntervalSeconds == nil || *got.IntervalSeconds != seconds {
		t.Fatalf("expected interval_seconds %d, got %v", seconds, got.IntervalSeconds)
	}
}

// TestOverlapPolicyByName ทดสอบว่าทุกชื่อ policy ที่ schedule.ParseOverlap ยอมรับ map ไป
// เป็น enum ของ Temporal ได้ถูกตัว — "" (ไม่ระบุ) เป็น UNSPECIFIED เพราะ usecase แปลงเป็น
// skip ให้เองตอนสร้าง/แก้ schedule ส่วนตอน trigger/backfill ต้องส่ง UNSPECIFIED ต่อไป
func TestOverlapPolicyByName(t *testing.T) {
	cases := []struct {
		name schedule.Overlap
		want enumspb.ScheduleOverlapPolicy
	}{
		{schedule.OverlapUnspecified, enumspb.SCHEDULE_OVERLAP_POLICY_UNSPECIFIED},
		{schedule.OverlapSkip, enumspb.SCHEDULE_OVERLAP_POLICY_SKIP},
		{schedule.OverlapBufferOne, enumspb.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE},
		{schedule.OverlapBufferAll, enumspb.SCHEDULE_OVERLAP_POLICY_BUFFER_ALL},
		{schedule.OverlapCancelOther, enumspb.SCHEDULE_OVERLAP_POLICY_CANCEL_OTHER},
		{schedule.OverlapTerminateOther, enumspb.SCHEDULE_OVERLAP_POLICY_TERMINATE_OTHER},
		{schedule.OverlapAllowAll, enumspb.SCHEDULE_OVERLAP_POLICY_ALLOW_ALL},
	}
	for _, tc := range cases {
		got, ok := overlapPolicyByName[tc.name]
		if !ok {
			t.Errorf("overlapPolicyByName missing entry for %q", tc.name)
			continue
		}
		if got != tc.want {
			t.Errorf("overlapPolicyByName[%q] = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestOverlapPolicyName_CoversAllKnownValues ทดสอบว่า overlapPolicyName (ใช้ตอบ JSON
// response ของ GET /schedules/{id}) ครอบคลุมทุกค่าที่ส่งเข้าไปได้ — กัน "ตกหล่น" ตอนแปลง
// กลับ (ส่ง policy เข้าไปได้ แต่ตอบกลับมาไม่ได้)
func TestOverlapPolicyName_CoversAllKnownValues(t *testing.T) {
	for name, policy := range overlapPolicyByName {
		if _, ok := overlapPolicyName[policy]; !ok {
			t.Errorf("overlapPolicyName missing entry for policy %v (name %q)", policy, name)
		}
	}
}

func TestReplaceSchedule_NilStateAndPolicy(t *testing.T) {
	// schedule ที่ Temporal ส่งมาอาจไม่มี State/Policy — ผลลัพธ์ต้องมีทั้งคู่เสมอ เพราะ SDK
	// อ่าน State.Note ตรงๆ (เดิม PUT /schedules panic ด้วยเหตุนี้)
	policy := schedule.Policy{Overlap: schedule.OverlapSkip, CatchupWindow: time.Hour}
	got := replaceSchedule(client.Schedule{}, client.ScheduleSpec{}, policy, &client.ScheduleWorkflowAction{ID: "wf"})
	if got.State == nil {
		t.Fatal("expected non-nil State")
	}
	if got.Policy == nil || got.Policy.Overlap != enumspb.SCHEDULE_OVERLAP_POLICY_SKIP {
		t.Fatalf("expected Policy with overlap SKIP, got %+v", got.Policy)
	}
	if got.Policy.CatchupWindow != time.Hour {
		t.Errorf("expected catch-up window 1h, got %v", got.Policy.CatchupWindow)
	}
}

func TestReplaceSchedule_KeepsUntouchedFields(t *testing.T) {
	current := client.Schedule{
		Action: &client.ScheduleWorkflowAction{ID: "old"},
		Spec:   &client.ScheduleSpec{CronExpressions: []string{"0 9 * * *"}},
		Policy: &client.SchedulePolicies{
			Overlap:        enumspb.SCHEDULE_OVERLAP_POLICY_SKIP,
			CatchupWindow:  time.Hour,
			PauseOnFailure: true,
		},
		State: &client.ScheduleState{Paused: true, Note: "maintenance"},
	}
	spec := client.ScheduleSpec{Intervals: []client.ScheduleIntervalSpec{{Every: 2 * time.Hour}}}

	policy := schedule.Policy{Overlap: schedule.OverlapBufferOne, CatchupWindow: 2 * time.Minute}
	got := replaceSchedule(current, spec, policy, &client.ScheduleWorkflowAction{ID: "new"})

	if action, ok := got.Action.(*client.ScheduleWorkflowAction); !ok || action.ID != "new" {
		t.Errorf("expected new action, got %+v", got.Action)
	}
	if len(got.Spec.Intervals) != 1 || len(got.Spec.CronExpressions) != 0 {
		t.Errorf("expected spec replaced by the 2h interval, got %+v", got.Spec)
	}
	if got.Policy.Overlap != enumspb.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE {
		t.Errorf("expected overlap BUFFER_ONE, got %v", got.Policy.Overlap)
	}
	if got.Policy.CatchupWindow != 2*time.Minute {
		t.Errorf("expected catch-up window replaced by 2m, got %v", got.Policy.CatchupWindow)
	}
	if !got.Policy.PauseOnFailure {
		t.Error("expected PauseOnFailure kept")
	}
	if !got.State.Paused || got.State.Note != "maintenance" {
		t.Errorf("expected paused state and note kept, got %+v", got.State)
	}
	if current.Policy.Overlap != enumspb.SCHEDULE_OVERLAP_POLICY_SKIP {
		t.Error("expected the current schedule's policy not to be mutated")
	}
}

// scheduledAction สร้าง action แบบที่ Describe คืนมา: Workflow เป็นชื่อ workflow type และ
// Args เป็น *commonpb.Payload ที่ encode ด้วย DataConverter ตัวเดียวกับตอนสร้าง Schedule
func scheduledAction(t *testing.T, workflowType string, args ...interface{}) *client.ScheduleWorkflowAction {
	t.Helper()
	payloads := make([]interface{}, 0, len(args))
	for _, arg := range args {
		p, err := converter.GetDefaultDataConverter().ToPayload(arg)
		if err != nil {
			t.Fatalf("encode arg: %v", err)
		}
		payloads = append(payloads, p)
	}
	return &client.ScheduleWorkflowAction{ID: "wf", Workflow: workflowType, Args: payloads}
}

func TestScheduledSteps_DecodesStepsFromActionArgs(t *testing.T) {
	want := []wfdomain.Step{
		{BotName: "http-status-checker", Config: bot.Config{"url": "https://example.com"}, TimeoutSeconds: 5},
		{K8sJob: &wfdomain.K8sJobSpec{Image: "busybox", Args: []string{"echo", "hi"}}},
	}
	action := scheduledAction(t, "Execute", wfdomain.Workflow{Name: "check", Steps: want})

	got, err := scheduledSteps(action)
	if err != nil {
		t.Fatalf("scheduledSteps: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("steps = %+v, want %+v", got, want)
	}
}

func TestScheduledSteps_EmptyForOtherWorkflowTypes(t *testing.T) {
	got, err := scheduledSteps(scheduledAction(t, "SomeoneElsesWorkflow", "not a bop workflow"))
	if err != nil {
		t.Fatalf("scheduledSteps: %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Errorf("steps = %#v, want empty non-nil list", got)
	}
}

func TestScheduledSteps_ErrorWhenArgsCannotBeDecoded(t *testing.T) {
	if _, err := scheduledSteps(scheduledAction(t, "Execute", "not a workflow")); err == nil {
		t.Error("expected an error for an Execute action whose args are not a workflow")
	}
}

func TestScheduledWorkflowName(t *testing.T) {
	tests := []struct {
		name   string
		action client.ScheduleAction
		want   string
	}{
		{"name from args", scheduledAction(t, "Execute", wfdomain.Workflow{Name: "check-example-com"}), "check-example-com"},
		{"empty name falls back to type", scheduledAction(t, "Execute", wfdomain.Workflow{}), "Execute"},
		{"args not a workflow falls back to type", scheduledAction(t, "Execute", "not a workflow"), "Execute"},
		{"no args falls back to type", scheduledAction(t, "Execute"), "Execute"},
		{"other workflow type keeps its type", scheduledAction(t, "SomeoneElsesWorkflow", wfdomain.Workflow{Name: "x"}), "SomeoneElsesWorkflow"},
		{"not a workflow action", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := scheduledWorkflowName(tt.action); got != tt.want {
				t.Errorf("scheduledWorkflowName = %q, want %q", got, tt.want)
			}
		})
	}
}

// round trip: ชื่อที่อ่านจาก Describe แล้วส่งกลับใน PUT ต้องสร้าง action ที่มี Action ID และ
// ชื่อใน args เท่าเดิม
func TestScheduledWorkflowName_RoundTripKeepsActionID(t *testing.T) {
	original := wfdomain.Workflow{Name: "uiwalk", Steps: []wfdomain.Step{{BotName: "b"}}}
	payload, err := converter.GetDefaultDataConverter().ToPayload(original)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	described := &client.ScheduleWorkflowAction{ID: "uiwalk", Workflow: "Execute", Args: []interface{}{payload}}

	updated := workflowAction(wfdomain.Workflow{Name: scheduledWorkflowName(described), Steps: original.Steps})

	if updated.ID != "uiwalk" {
		t.Errorf("action ID = %q, want uiwalk", updated.ID)
	}
	if got := updated.Args[0].(wfdomain.Workflow).Name; got != "uiwalk" {
		t.Errorf("name in args = %q, want uiwalk", got)
	}
}
