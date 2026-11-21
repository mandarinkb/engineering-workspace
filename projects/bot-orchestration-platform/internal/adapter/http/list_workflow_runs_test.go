package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apihttp "bop/internal/adapter/http"
	"bop/internal/adapter/worker"
	"bop/internal/domain/workflow"
	"bop/internal/external/temporal"
	memlogger "bop/internal/pkg/logger"
	"bop/internal/repository/memory"
	authuc "bop/internal/usecase/auth"
	jobuc "bop/internal/usecase/job"
	scheduleuc "bop/internal/usecase/schedule"
	workflowrunuc "bop/internal/usecase/workflowrun"
)

// TestListWorkflowRuns_RejectsBadQueryBeforeTouchingTemporal พิสูจน์ว่า query param ที่ผิด
// ของ GET /workflow-runs ตอบ 400 ก่อนแตะ Temporal เสมอ — newTestServer ไม่มี Temporal
// client จริง (nil) ถ้า handler หลุดไปเรียก engine จะ panic แทนที่จะได้ 400 ส่วนการเดินหน้า
// และ filter สถานะจริงทดสอบผ่าน seam A (Bruno) กับ Temporal ที่รันอยู่จริง
func TestListWorkflowRuns_RejectsBadQueryBeforeTouchingTemporal(t *testing.T) {
	ts := newTestServer(t)

	tests := []struct {
		name      string
		giveQuery string
	}{
		{"unknown status", "status=Running"},
		{"page_size not a number", "page_size=abc"},
		{"page_size zero", "page_size=0"},
		{"page_size negative", "page_size=-1"},
		{"page_token not base64", "page_token=%25%25%25"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Get(ts.URL + "/workflow-runs?" + tt.giveQuery)
			if err != nil {
				t.Fatalf("GET /workflow-runs: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", resp.StatusCode)
			}
		})
	}
}

// scheduledRunsEngine จำลอง engine ที่คืน Workflow Run จาก Schedule หนึ่งตัวกับ run ที่สั่งเอง
// หนึ่งตัว และจำ query ล่าสุดไว้ให้ test ตรวจว่า filter ถูกส่งต่อมาถึง engine ครบ
type scheduledRunsEngine struct {
	noStepsEngine

	got *workflow.RunQuery
}

func (e scheduledRunsEngine) ListRuns(
	_ context.Context, q workflow.RunQuery,
) (workflow.RunPage, error) {
	*e.got = q
	return workflow.RunPage{Runs: []workflow.RunSummary{
		{WorkflowID: "nightly-wf-2026-11-21T00:00:00Z", ScheduleID: "nightly", Status: "Completed"},
		{WorkflowID: "adhoc-1", Status: "Completed"},
	}}, nil
}

// TestListWorkflowRuns_FiltersBySchedule ยืนยันว่า ?schedule_id= ถูกส่งไปพร้อม status และ
// การแบ่งหน้า และแต่ละ run บอก schedule_id ของ Schedule ที่สั่งมัน (ว่างสำหรับ run ที่สั่งเอง)
func TestListWorkflowRuns_FiltersBySchedule(t *testing.T) {
	var got workflow.RunQuery
	repo := memory.NewJobRepository()
	log := memlogger.New()
	auth := authuc.NewService([]byte(testSecret), testUsername, testPassword)
	srv := apihttp.NewServer(
		jobuc.NewService(repo, worker.NewPool(1, time.Second, repo, log), log),
		workflowrunuc.NewService(scheduledRunsEngine{got: &got}),
		scheduleuc.NewService(temporal.NewScheduler(nil)),
		auth,
	)
	ts := httptest.NewServer(withSession(t, auth, srv))
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/workflow-runs?schedule_id=nightly&status=completed&page_size=10")
	if err != nil {
		t.Fatalf("GET /workflow-runs: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	want := workflow.RunQuery{
		Filter:   workflow.RunFilter{Status: workflow.RunStatusCompleted, ScheduleID: "nightly"},
		PageSize: 10,
	}
	if got != want {
		t.Errorf("engine got query %+v, want %+v", got, want)
	}

	var body struct {
		Runs []map[string]json.RawMessage `json:"runs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(body.Runs) != 2 {
		t.Fatalf("expected 2 runs, got %d", len(body.Runs))
	}
	if got := string(body.Runs[0]["schedule_id"]); got != `"nightly"` {
		t.Errorf(`scheduled run: expected "schedule_id": "nightly", got %s`, got)
	}
	if got := string(body.Runs[1]["schedule_id"]); got != `""` {
		t.Errorf(`ad-hoc run: expected "schedule_id": "", got %s`, got)
	}
}
