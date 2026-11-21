package http_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apihttp "bop/internal/adapter/http"
	"bop/internal/adapter/worker"
	"bop/internal/domain/apperr"
	"bop/internal/domain/logger"
	"bop/internal/domain/workflow"
	"bop/internal/external/temporal"
	memlogger "bop/internal/pkg/logger"
	"bop/internal/repository/memory"
	authuc "bop/internal/usecase/auth"
	jobuc "bop/internal/usecase/job"
	scheduleuc "bop/internal/usecase/schedule"
	workflowrunuc "bop/internal/usecase/workflowrun"
)

// twoStepEngine จำลอง engine ที่รู้จัก Workflow Run เดียว ("wf-two") ซึ่งมี 2 Step
type twoStepEngine struct {
	noStepsEngine
}

func (twoStepEngine) GetRunWorkflow(_ context.Context, id string) (*workflow.Workflow, error) {
	if id != "wf-two" {
		return nil, apperr.WrapNotFound(fmt.Errorf("workflow run %s not found", id))
	}
	return &workflow.Workflow{
		Name:  "two",
		Steps: []workflow.Step{{BotName: "ok-bot"}, {BotName: "fail-bot"}},
	}, nil
}

func newStepLogsTestServer(t *testing.T, stepLogs logger.StepLogger) *httptest.Server {
	t.Helper()
	repo := memory.NewJobRepository()
	log := memlogger.New()
	auth := authuc.NewService([]byte(testSecret), testUsername, testPassword)
	srv := apihttp.NewServer(
		jobuc.NewService(repo, worker.NewPool(1, time.Second, repo, log), log),
		workflowrunuc.NewService(twoStepEngine{}, stepLogs),
		scheduleuc.NewService(temporal.NewScheduler(nil)),
		auth,
	)
	ts := httptest.NewServer(withSession(t, auth, srv))
	t.Cleanup(ts.Close)
	return ts
}

// TestGetWorkflowRunStepLogs_ReturnsEveryAttemptInOrder ยืนยันว่า log ของ Step อ่านได้ด้วย workflow ID
// + ลำดับ Step ครบทุก Attempt เรียงตามเวลา และทุกบรรทัดบอก attempt ของตัวเอง
func TestGetWorkflowRunStepLogs_ReturnsEveryAttemptInOrder(t *testing.T) {
	stepLogs := memlogger.New()
	ctx := context.Background()
	for attempt := 1; attempt <= 3; attempt++ {
		owner := logger.StepAttempt{WorkflowID: "wf-two", StepIndex: 1, Attempt: attempt}
		stepLogs.LogStep(ctx, owner, logger.LevelInfo, "requesting")
		stepLogs.LogStep(ctx, owner, logger.LevelError, "connection refused")
	}
	stepLogs.LogStep(ctx, logger.StepAttempt{WorkflowID: "wf-two", StepIndex: 0, Attempt: 1},
		logger.LevelInfo, "other step")
	ts := newStepLogsTestServer(t, stepLogs)

	resp, err := http.Get(ts.URL + "/workflow-runs/wf-two/steps/1/logs")
	if err != nil {
		t.Fatalf("GET step logs: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var lines []struct {
		Attempt int    `json:"attempt"`
		Level   string `json:"level"`
		Message string `json:"message"`
		Time    string `json:"time"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&lines); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(lines) != 6 {
		t.Fatalf("got %d lines, want 6 (2 per Attempt): %+v", len(lines), lines)
	}
	for i, l := range lines {
		wantAttempt := i/2 + 1
		wantMessage := "requesting"
		if i%2 == 1 {
			wantMessage = "connection refused"
		}
		if l.Attempt != wantAttempt || l.Message != wantMessage || l.Time == "" {
			t.Errorf("line %d = %+v, want attempt %d %q with time", i, l, wantAttempt, wantMessage)
		}
	}
}

// TestGetWorkflowRunStepLogs_StepWithoutLogsIsEmptyList ยืนยันว่า Step ที่มีจริงแต่ยังไม่ได้รัน
// (เช่น Step หลัง Failed Step) ตอบ [] ไม่ใช่ 404
func TestGetWorkflowRunStepLogs_StepWithoutLogsIsEmptyList(t *testing.T) {
	ts := newStepLogsTestServer(t, memlogger.New())

	resp, err := http.Get(ts.URL + "/workflow-runs/wf-two/steps/0/logs")
	if err != nil {
		t.Fatalf("GET step logs: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if string(body) != "[]" {
		t.Errorf("body = %s, want []", body)
	}
}

// TestGetWorkflowRunStepLogs_Errors ครอบลำดับ Step ที่ไม่มีจริง (404), Workflow Run ที่ไม่มีจริง (404)
// และลำดับ Step ที่ไม่ใช่ตัวเลข (400)
func TestGetWorkflowRunStepLogs_Errors(t *testing.T) {
	tests := []struct {
		name       string
		givePath   string
		wantStatus int
	}{
		{name: "index past the last Step", givePath: "/workflow-runs/wf-two/steps/2/logs", wantStatus: http.StatusNotFound},
		{name: "negative index", givePath: "/workflow-runs/wf-two/steps/-1/logs", wantStatus: http.StatusNotFound},
		{name: "unknown Workflow Run", givePath: "/workflow-runs/wf-missing/steps/0/logs", wantStatus: http.StatusNotFound},
		{name: "index is not a number", givePath: "/workflow-runs/wf-two/steps/first/logs", wantStatus: http.StatusBadRequest},
	}
	ts := newStepLogsTestServer(t, memlogger.New())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Get(ts.URL + tt.givePath)
			if err != nil {
				t.Fatalf("GET %s: %v", tt.givePath, err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}
		})
	}
}
