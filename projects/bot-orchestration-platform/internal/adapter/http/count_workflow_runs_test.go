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

// countingEngine จำลอง engine ที่นับ Workflow Run ได้ตามกลุ่มสถานะ — จำนวนตั้งให้มากกว่า
// หน้าเดียว (สูงสุด 200) เพื่อยืนยันว่าตัวเลขมาจากการนับฝั่ง engine ไม่ใช่ความยาวของหน้าแรก
type countingEngine struct {
	noStepsEngine

	counts map[workflow.RunStatusFilter]int64
}

func (e countingEngine) CountRuns(_ context.Context, status workflow.RunStatusFilter) (int64, error) {
	return e.counts[status], nil
}

func newCountingTestServer(t *testing.T, counts map[workflow.RunStatusFilter]int64) *httptest.Server {
	t.Helper()
	repo := memory.NewJobRepository()
	log := memlogger.New()
	auth := authuc.NewService([]byte(testSecret), testUsername, testPassword)
	srv := apihttp.NewServer(
		jobuc.NewService(repo, worker.NewPool(1, time.Second, repo, log), log),
		workflowrunuc.NewService(countingEngine{counts: counts}),
		scheduleuc.NewService(temporal.NewScheduler(nil)),
		auth,
	)
	ts := httptest.NewServer(withSession(t, auth, srv))
	t.Cleanup(ts.Close)
	return ts
}

// TestCountWorkflowRuns_ReturnsEngineCountPerStatus ยืนยันว่า GET /workflow-runs/count ตอบ
// { "count": n } ตามกลุ่มสถานะที่ขอ (ไม่ส่ง status = นับทุกสถานะ)
func TestCountWorkflowRuns_ReturnsEngineCountPerStatus(t *testing.T) {
	ts := newCountingTestServer(t, map[workflow.RunStatusFilter]int64{
		workflow.RunStatusAny:     1500,
		workflow.RunStatusRunning: 237,
		workflow.RunStatusFailed:  3,
	})

	tests := []struct {
		name      string
		giveQuery string
		want      string
	}{
		{"no status counts every run", "", `{"count":1500}`},
		{"running", "?status=running", `{"count":237}`},
		{"failed", "?status=failed", `{"count":3}`},
		{"canceled with none", "?status=canceled", `{"count":0}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Get(ts.URL + "/workflow-runs/count" + tt.giveQuery)
			if err != nil {
				t.Fatalf("GET /workflow-runs/count: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected 200, got %d", resp.StatusCode)
			}
			var body json.RawMessage
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if got := string(body); got != tt.want {
				t.Fatalf("expected %s, got %s", tt.want, got)
			}
		})
	}
}

// TestCountWorkflowRuns_RejectsUnknownStatusBeforeTouchingTemporal ใช้ newTestServer ที่
// Temporal client เป็น nil — ถ้า handler หลุดไปเรียก engine จะ panic แทนที่จะได้ 400
func TestCountWorkflowRuns_RejectsUnknownStatusBeforeTouchingTemporal(t *testing.T) {
	ts := newTestServer(t)

	resp, err := http.Get(ts.URL + "/workflow-runs/count?status=Running")
	if err != nil {
		t.Fatalf("GET /workflow-runs/count: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
