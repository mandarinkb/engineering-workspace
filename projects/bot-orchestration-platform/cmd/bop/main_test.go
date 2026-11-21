package main

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"bop/internal/domain/job"
	"bop/internal/domain/logger"
	memlogger "bop/internal/pkg/logger"
	"bop/internal/repository/memory"
)

// TestOpenDatabase_WithoutURLWarnsAndUsesMemory ยืนยันว่า `go run` แบบไม่ตั้ง BOP_DATABASE_URL
// ยังทำงานได้ (ไม่มี pool = ใช้ repository แบบ memory) และมีคำเตือนใน log ตอน start
func TestOpenDatabase_WithoutURLWarnsAndUsesMemory(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	db, err := openDatabase(context.Background(), "")
	if err != nil {
		t.Fatalf("openDatabase: %v", err)
	}
	if db != nil {
		t.Fatal("openDatabase without URL returned a pool, want nil (memory mode)")
	}
	if _, ok := newJobRepository(db).(*memory.JobRepository); !ok {
		t.Error("job repository without database is not the memory implementation")
	}
	if _, ok := newLogger(db).(*memlogger.Logger); !ok {
		t.Error("logger without database is not the memory implementation")
	}
	if !strings.Contains(buf.String(), "BOP_DATABASE_URL") {
		t.Errorf("log = %q, want a warning that mentions BOP_DATABASE_URL", buf.String())
	}
}

// TestOpenDatabase_MigratesOnStart ยืนยันว่า bop-api สร้าง schema เองตอน start: database ใหม่
// ที่ไม่มีตารางเลยใช้เก็บ Job ได้ทันทีหลัง openDatabase (รันเฉพาะเมื่อตั้ง BOP_TEST_DATABASE_URL)
func TestOpenDatabase_MigratesOnStart(t *testing.T) {
	url := os.Getenv("BOP_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("BOP_TEST_DATABASE_URL not set: skipping PostgreSQL test")
	}
	ctx := context.Background()
	db, err := openDatabase(ctx, url)
	if err != nil {
		t.Fatalf("openDatabase: %v", err)
	}
	t.Cleanup(db.Close)

	repo := newJobRepository(db)
	j := &job.Job{
		ID:        "main-" + time.Now().Format(time.RFC3339Nano),
		BotName:   "b",
		Status:    job.StatusPending,
		CreatedAt: time.Now(),
	}
	if err := repo.Create(ctx, j); err != nil {
		t.Fatalf("Create after openDatabase: %v", err)
	}
	if _, err := repo.Get(ctx, j.ID); err != nil {
		t.Fatalf("Get after openDatabase: %v", err)
	}

	// log ของ Job ต้องเก็บลง database เดียวกันได้ทันทีเช่นกัน
	l := newLogger(db)
	l.Log(ctx, j.ID, logger.LevelInfo, "stored")
	lines, err := l.History(ctx, j.ID)
	if err != nil {
		t.Fatalf("History after openDatabase: %v", err)
	}
	if len(lines) != 1 || lines[0].Message != "stored" {
		t.Errorf("History = %+v, want the stored line", lines)
	}
	if _, ok := l.(logger.Pruner); !ok {
		t.Error("PostgreSQL logger cannot be pruned by the log retention job")
	}
}

// TestServe_ShutsDownPromptlyWithOpenStream ทดสอบว่า serve ปิดตัวได้ทันทีเมื่อ ctx ถูก
// ยกเลิก แม้จะมี request แบบ stream (เหมือน SSE live log) ที่รอ r.Context() ค้างอยู่ —
// ถ้าไม่ผูก BaseContext ไว้ serve จะค้างรอจนหมด shutdownTimeout (30 วินาที)
func TestServe_ShutsDownPromptlyWithOpenStream(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("find free port: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close()
	t.Setenv("BOP_HTTP_ADDR", addr)

	streaming := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		close(streaming)
		<-r.Context().Done()
	})

	ctx, cancel := context.WithCancel(context.Background())
	served := make(chan error, 1)
	go func() { served <- serve(ctx, handler) }()

	var resp *http.Response
	for range 50 { // รอให้ server เปิด port ก่อน
		if resp, err = http.Get("http://" + addr + "/stream"); err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("GET /stream: %v", err)
	}
	defer resp.Body.Close()
	<-streaming

	cancel()
	select {
	case err := <-served:
		if err != nil {
			t.Fatalf("serve returned error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("serve did not shut down while a stream was open")
	}
}

// fakeJobRepository นับการเรียก FailInterrupted เพื่อตรวจว่า reconcileInterruptedJobs เรียก
// ตามค่า BOP_RECONCILE_INTERRUPTED_JOBS
type fakeJobRepository struct {
	job.Repository
	calls  int
	reason string
	now    time.Time
	n      int
	err    error
}

func (f *fakeJobRepository) FailInterrupted(_ context.Context, reason string, now time.Time) (int, error) {
	f.calls++
	f.reason, f.now = reason, now
	return f.n, f.err
}

func TestReconcileInterruptedJobs(t *testing.T) {
	fixed := time.Date(2026, 11, 21, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		give      string
		giveErr   error
		wantCalls int
		wantErr   bool
	}{
		{name: "unset means enabled", give: "", wantCalls: 1},
		{name: "true", give: "true", wantCalls: 1},
		{name: "false disables", give: "false", wantCalls: 0},
		{name: "unknown value is an error", give: "maybe", wantCalls: 0, wantErr: true},
		{name: "repository error is returned", give: "", giveErr: errors.New("db down"), wantCalls: 1, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			log.SetOutput(&buf)
			t.Cleanup(func() { log.SetOutput(os.Stderr) })
			repo := &fakeJobRepository{n: 3, err: tt.giveErr}

			err := reconcileInterruptedJobs(context.Background(), repo, tt.give, func() time.Time { return fixed })

			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if repo.calls != tt.wantCalls {
				t.Errorf("FailInterrupted calls = %d, want %d", repo.calls, tt.wantCalls)
			}
			if tt.wantCalls == 1 && (repo.reason != interruptedReason || !repo.now.Equal(fixed)) {
				t.Errorf("called with (%q, %v), want (%q, %v)", repo.reason, repo.now, interruptedReason, fixed)
			}
			if tt.wantCalls == 1 && tt.giveErr == nil && !strings.Contains(buf.String(), "marked 3 job(s)") {
				t.Errorf("log = %q, want the swept count", buf.String())
			}
		})
	}
}
