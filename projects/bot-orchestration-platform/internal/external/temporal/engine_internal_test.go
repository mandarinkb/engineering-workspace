// package temporal (ไม่ใช่ temporal_test) เพราะทดสอบ unexported function (visibilityQuery,
// toRunSummary) ตรงๆ โดยไม่ต้องมี Temporal server — การกรองจริงกับ visibility store ทดสอบที่
// seam A (Bruno)
package temporal

import (
	"testing"

	commonpb "go.temporal.io/api/common/v1"
	workflowpb "go.temporal.io/api/workflow/v1"
	"go.temporal.io/sdk/converter"

	wfdomain "bop/internal/domain/workflow"
)

func TestVisibilityQuery(t *testing.T) {
	tests := []struct {
		name string
		give wfdomain.RunFilter
		want string
	}{
		{"no filter", wfdomain.RunFilter{}, ""},
		{
			"status only",
			wfdomain.RunFilter{Status: wfdomain.RunStatusRunning},
			"ExecutionStatus = 'Running'",
		},
		{
			"schedule only",
			wfdomain.RunFilter{ScheduleID: "nightly"},
			"TemporalScheduledById = 'nightly'",
		},
		{
			"status and schedule",
			wfdomain.RunFilter{Status: wfdomain.RunStatusCompleted, ScheduleID: "nightly"},
			"ExecutionStatus = 'Completed' AND TemporalScheduledById = 'nightly'",
		},
		{
			"schedule ID with quote and backslash is escaped",
			wfdomain.RunFilter{ScheduleID: `it's\x`},
			`TemporalScheduledById = 'it\'s\\x'`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := visibilityQuery(tt.give); got != tt.want {
				t.Errorf("visibilityQuery(%+v) = %q, want %q", tt.give, got, tt.want)
			}
		})
	}
}

func TestToRunSummary_ScheduleIDFromSearchAttribute(t *testing.T) {
	payload, err := converter.GetDefaultDataConverter().ToPayload("nightly")
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}
	info := &workflowpb.WorkflowExecutionInfo{
		SearchAttributes: &commonpb.SearchAttributes{
			IndexedFields: map[string]*commonpb.Payload{"TemporalScheduledById": payload},
		},
	}

	if got := toRunSummary(info).ScheduleID; got != "nightly" {
		t.Errorf("ScheduleID = %q, want %q", got, "nightly")
	}
}

func TestToRunSummary_AdHocRunHasNoScheduleID(t *testing.T) {
	if got := toRunSummary(&workflowpb.WorkflowExecutionInfo{}).ScheduleID; got != "" {
		t.Errorf("ScheduleID = %q, want empty", got)
	}
}
