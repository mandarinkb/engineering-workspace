package http_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

// downEngine คือ engine ที่ต่อ Temporal ไม่ได้ (CheckHealth ล้มเหลว)
type downEngine struct {
	workflow.Engine
}

func (downEngine) CheckHealth(context.Context) error { return errors.New("temporal down") }

// stubPinger จำลอง database: err = ping ล้มเหลว, block = ค้างจนกว่า ctx จะหมดเวลา
type stubPinger struct {
	err   error
	block bool
}

func (p stubPinger) Ping(ctx context.Context) error {
	if p.block {
		<-ctx.Done()
		return ctx.Err()
	}
	return p.err
}

func newReadyzServer(t *testing.T, engine workflow.Engine, opts ...apihttp.ServerOption) *httptest.Server {
	t.Helper()
	repo := memory.NewJobRepository()
	log := memlogger.New()
	auth := authuc.NewService([]byte(testSecret), testUsername, testPassword)
	srv := apihttp.NewServer(
		jobuc.NewService(repo, worker.NewPool(1, time.Second, repo, log), log),
		workflowrunuc.NewService(engine, log),
		scheduleuc.NewService(temporal.NewScheduler(nil)),
		auth,
		opts...,
	)
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	return ts
}

// TestReadyz ยืนยันว่า /readyz เช็คทั้ง Temporal และ database (เมื่อมี) และบอกสาเหตุที่ต่างกัน
// ในตัว body — ไม่มี database (โหมด memory) ผลเหมือนเดิมทุกประการ
func TestReadyz(t *testing.T) {
	healthy := healthyEngine{Engine: temporal.NewEngine(nil)}
	down := downEngine{Engine: temporal.NewEngine(nil)}
	tests := []struct {
		name       string
		giveEngine workflow.Engine
		giveOpts   []apihttp.ServerOption
		wantStatus int
		wantBody   string
	}{
		{"no database configured, temporal ok", healthy, nil, http.StatusOK, ""},
		{"no database configured, temporal down", down, nil, http.StatusServiceUnavailable, "temporal unreachable"},
		{"database ok, temporal ok", healthy, []apihttp.ServerOption{apihttp.WithDatabase(stubPinger{})}, http.StatusOK, ""},
		{
			"database down", healthy,
			[]apihttp.ServerOption{apihttp.WithDatabase(stubPinger{err: errors.New("refused")})},
			http.StatusServiceUnavailable, "database unreachable",
		},
		{
			"database hangs past the timeout", healthy,
			[]apihttp.ServerOption{apihttp.WithDatabase(stubPinger{block: true})},
			http.StatusServiceUnavailable, "database unreachable",
		},
		{
			"temporal down is reported even when database is ok", down,
			[]apihttp.ServerOption{apihttp.WithDatabase(stubPinger{})},
			http.StatusServiceUnavailable, "temporal unreachable",
		},
		{
			"both down reports temporal first", down,
			[]apihttp.ServerOption{apihttp.WithDatabase(stubPinger{err: errors.New("refused")})},
			http.StatusServiceUnavailable, "temporal unreachable",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newReadyzServer(t, tt.giveEngine, tt.giveOpts...)

			resp, err := http.Get(ts.URL + "/readyz")
			if err != nil {
				t.Fatalf("GET /readyz: %v", err)
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)

			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}
			if got := strings.TrimSpace(string(body)); got != tt.wantBody {
				t.Errorf("body = %q, want %q", got, tt.wantBody)
			}
		})
	}
}
