// test การ validate spec/overlap — ย้ายมาจาก internal/api/http/schedule_internal_test.go
// เดิม (buildScheduleSpec/parseOverlapPolicy) หลังกฎพวกนี้ย้ายมาอยู่ที่ domain
package schedule_test

import (
	"testing"

	"bop/internal/domain/schedule"
)

func TestSpecValidate_Interval(t *testing.T) {
	seconds := int64(60)
	if err := (schedule.Spec{IntervalSeconds: &seconds}).Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSpecValidate_Cron(t *testing.T) {
	cron := "0 9 * * MON-FRI"
	if err := (schedule.Spec{CronExpression: &cron}).Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSpecValidate_RejectsBothIntervalAndCron(t *testing.T) {
	seconds := int64(60)
	cron := "0 9 * * MON-FRI"
	if err := (schedule.Spec{IntervalSeconds: &seconds, CronExpression: &cron}).Validate(); err == nil {
		t.Fatal("expected error when both interval_seconds and cron_expression are set")
	}
}

func TestSpecValidate_RejectsNeither(t *testing.T) {
	if err := (schedule.Spec{}).Validate(); err == nil {
		t.Fatal("expected error when neither interval_seconds nor cron_expression is set")
	}
}

func TestSpecValidate_RejectsNonPositiveInterval(t *testing.T) {
	zero := int64(0)
	if err := (schedule.Spec{IntervalSeconds: &zero}).Validate(); err == nil {
		t.Fatal("expected error for interval_seconds <= 0")
	}
}

func TestParseOverlap(t *testing.T) {
	cases := []struct {
		name    string
		want    schedule.Overlap
		wantErr bool
	}{
		{"", schedule.OverlapUnspecified, false},
		{"skip", schedule.OverlapSkip, false},
		{"buffer_one", schedule.OverlapBufferOne, false},
		{"buffer_all", schedule.OverlapBufferAll, false},
		{"cancel_other", schedule.OverlapCancelOther, false},
		{"terminate_other", schedule.OverlapTerminateOther, false},
		{"allow_all", schedule.OverlapAllowAll, false},
		{"not-a-real-policy", "", true},
	}
	for _, tc := range cases {
		got, err := schedule.ParseOverlap(tc.name)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParseOverlap(%q): expected error, got none", tc.name)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseOverlap(%q): unexpected error: %v", tc.name, err)
		}
		if got != tc.want {
			t.Errorf("ParseOverlap(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
}
