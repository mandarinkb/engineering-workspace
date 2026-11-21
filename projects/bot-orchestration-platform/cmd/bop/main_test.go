package main

import (
	"bytes"
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"bop/internal/domain/job"
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
