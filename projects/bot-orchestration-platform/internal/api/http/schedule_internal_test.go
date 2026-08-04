// package http (ไม่ใช่ http_test) เพราะ test ไฟล์นี้ทดสอบ unexported function
// (buildScheduleSpec, parseOverlapPolicy, specToResponse) ตรงๆ โดยไม่ผ่าน HTTP layer
// เลย — เหมาะกับ pure business logic ของ schedule_handler.go ที่ไม่ต้องพึ่ง
// Temporal client จริงในการทดสอบ (ต่างจาก handler_test.go ที่ทดสอบผ่าน HTTP request
// จริงแต่ทำได้แค่ path ที่ validate fail ก่อนแตะ Temporal client เพราะ test setup ไม่มี
// Temporal server จริงให้ต่อ)
package http

import (
	"testing"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
)

func TestBuildScheduleSpec_Interval(t *testing.T) {
	seconds := int64(60)
	spec, err := buildScheduleSpec(scheduleSpecRequest{IntervalSeconds: &seconds})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(spec.Intervals) != 1 || spec.Intervals[0].Every != time.Minute {
		t.Fatalf("expected 1 interval of 1 minute, got %+v", spec.Intervals)
	}
}

func TestBuildScheduleSpec_Cron(t *testing.T) {
	cron := "0 9 * * MON-FRI"
	spec, err := buildScheduleSpec(scheduleSpecRequest{CronExpression: &cron})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(spec.CronExpressions) != 1 || spec.CronExpressions[0] != cron {
		t.Fatalf("expected cron expression %q, got %+v", cron, spec.CronExpressions)
	}
}

func TestBuildScheduleSpec_RejectsBothIntervalAndCron(t *testing.T) {
	seconds := int64(60)
	cron := "0 9 * * MON-FRI"
	if _, err := buildScheduleSpec(scheduleSpecRequest{IntervalSeconds: &seconds, CronExpression: &cron}); err == nil {
		t.Fatal("expected error when both interval_seconds and cron_expression are set")
	}
}

func TestBuildScheduleSpec_RejectsNeither(t *testing.T) {
	if _, err := buildScheduleSpec(scheduleSpecRequest{}); err == nil {
		t.Fatal("expected error when neither interval_seconds nor cron_expression is set")
	}
}

func TestBuildScheduleSpec_RejectsNonPositiveInterval(t *testing.T) {
	zero := int64(0)
	if _, err := buildScheduleSpec(scheduleSpecRequest{IntervalSeconds: &zero}); err == nil {
		t.Fatal("expected error for interval_seconds <= 0")
	}
}

// TestSpecToResponse_RoundTrip ทดสอบว่า buildScheduleSpec แล้วแปลงกลับด้วย
// specToResponse ได้ค่าเดิม (round-trip) — สำคัญเพราะ GET /schedules/{id} ต้องแสดง
// spec ที่ตรงกับตอน POST /schedules ส่งมา ไม่ใช่ค่าที่เพี้ยนไปจากการแปลงไป-กลับ
func TestSpecToResponse_RoundTrip(t *testing.T) {
	seconds := int64(120)
	spec, err := buildScheduleSpec(scheduleSpecRequest{IntervalSeconds: &seconds})
	if err != nil {
		t.Fatalf("buildScheduleSpec: %v", err)
	}
	gotInterval, gotCron := specToResponse(&spec)
	if gotCron != nil {
		t.Errorf("expected nil cron expression, got %v", *gotCron)
	}
	if gotInterval == nil || *gotInterval != seconds {
		t.Fatalf("expected interval_seconds %d, got %v", seconds, gotInterval)
	}
}

func TestParseOverlapPolicy(t *testing.T) {
	cases := []struct {
		name    string
		want    enumspb.ScheduleOverlapPolicy
		wantErr bool
	}{
		{"", enumspb.SCHEDULE_OVERLAP_POLICY_SKIP, false},
		{"skip", enumspb.SCHEDULE_OVERLAP_POLICY_SKIP, false},
		{"buffer_one", enumspb.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE, false},
		{"buffer_all", enumspb.SCHEDULE_OVERLAP_POLICY_BUFFER_ALL, false},
		{"cancel_other", enumspb.SCHEDULE_OVERLAP_POLICY_CANCEL_OTHER, false},
		{"terminate_other", enumspb.SCHEDULE_OVERLAP_POLICY_TERMINATE_OTHER, false},
		{"allow_all", enumspb.SCHEDULE_OVERLAP_POLICY_ALLOW_ALL, false},
		{"not-a-real-policy", 0, true},
	}
	for _, tc := range cases {
		got, err := parseOverlapPolicy(tc.name)
		if tc.wantErr {
			if err == nil {
				t.Errorf("parseOverlapPolicy(%q): expected error, got none", tc.name)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseOverlapPolicy(%q): unexpected error: %v", tc.name, err)
		}
		if got != tc.want {
			t.Errorf("parseOverlapPolicy(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestOverlapPolicyName_CoversAllKnownValues ทดสอบว่า overlapPolicyName (ใช้ตอบ JSON
// response ของ GET /schedules/{id}) ครอบคลุมทุกค่าที่ parseOverlapPolicy ยอมรับเป็น
// input ได้ — กัน "ตกหล่น" ตอนแปลงกลับ (ส่ง policy เข้าไปได้ แต่ตอบกลับมาไม่ได้)
func TestOverlapPolicyName_CoversAllKnownValues(t *testing.T) {
	for name, policy := range overlapPolicyByName {
		if name == "" {
			continue // "" คือ alias ของ "skip" ไม่ใช่ค่าที่ควรถูกคืนกลับมาตรงๆ
		}
		if _, ok := overlapPolicyName[policy]; !ok {
			t.Errorf("overlapPolicyName missing entry for policy %v (name %q)", policy, name)
		}
	}
}
