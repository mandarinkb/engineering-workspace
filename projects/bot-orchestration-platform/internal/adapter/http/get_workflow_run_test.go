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

// noStepsEngine จำลอง engine ที่ตอบ GetRun ด้วย Steps เป็น nil — เหมือน Temporal ตอน
// Workflow Run เพิ่งเริ่มและยังไม่มี Step ไหนจบ (หรือ query ผลลัพธ์ไม่ได้)
type noStepsEngine struct{}

func (noStepsEngine) Start(context.Context, string, workflow.Workflow) (string, error) {
	return "", nil
}

func (noStepsEngine) ListRuns(context.Context, workflow.RunQuery) (workflow.RunPage, error) {
	return workflow.RunPage{}, nil
}

func (noStepsEngine) GetRun(_ context.Context, id string) (*workflow.RunDetail, error) {
	return &workflow.RunDetail{
		RunSummary: workflow.RunSummary{WorkflowID: id, Status: "Running", StartedAt: time.Now()},
	}, nil
}

func (noStepsEngine) GetRunWorkflow(context.Context, string) (*workflow.Workflow, error) {
	return &workflow.Workflow{}, nil
}

func (noStepsEngine) CheckHealth(context.Context) error { return nil }

// TestGetWorkflowRun_NoFinishedStepReturnsEmptyList ยืนยันว่า Workflow Run ที่ยังไม่มี Step
// ไหนจบตอบ "steps": [] ไม่ใช่ null เพื่อให้ API client วนอ่าน steps ได้โดยไม่ต้องเช็ค null
func TestGetWorkflowRun_NoFinishedStepReturnsEmptyList(t *testing.T) {
	repo := memory.NewJobRepository()
	log := memlogger.New()
	auth := authuc.NewService([]byte(testSecret), testUsername, testPassword)
	srv := apihttp.NewServer(
		jobuc.NewService(repo, worker.NewPool(1, time.Second, repo, log), log),
		workflowrunuc.NewService(noStepsEngine{}, log),
		scheduleuc.NewService(temporal.NewScheduler(nil)),
		auth,
	)
	ts := httptest.NewServer(withSession(t, auth, srv))
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/workflow-runs/wf-1")
	if err != nil {
		t.Fatalf("GET /workflow-runs/wf-1: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got := string(body["steps"]); got != "[]" {
		t.Fatalf(`expected "steps": [], got %s`, got)
	}
}
