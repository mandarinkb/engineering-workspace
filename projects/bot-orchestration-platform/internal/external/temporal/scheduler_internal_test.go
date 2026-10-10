// package temporal (ไม่ใช่ temporal_test) เพราะ test ไฟล์นี้ทดสอบ unexported function
// (toScheduleSpec, fromScheduleSpec, overlapPolicyByName/overlapPolicyName) ตรงๆ โดยไม่
// ต้องพึ่ง Temporal client จริง — ย้ายมาจาก internal/api/http/schedule_internal_test.go เดิม
// ส่วนที่ทดสอบการ validate spec/overlap ย้ายไปอยู่ที่ internal/domain/schedule แล้ว
package temporal

import (
	"testing"
	"time"

	enumspb "go.temporal.io/api/enums/v1"

	"bop/internal/domain/schedule"
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
