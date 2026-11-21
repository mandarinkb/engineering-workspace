package main

import (
	"bytes"
	"context"
	"log"
	"os"
	"strings"
	"testing"

	"bop/internal/domain/logger"
	memlogger "bop/internal/pkg/logger"
	"bop/internal/repository/postgres"
)

// TestNewStepLogger_WithoutURLWarnsAndUsesMemory ยืนยันว่า bop-worker ที่ไม่ได้ตั้ง BOP_DATABASE_URL
// ยัง start ได้ด้วย logger แบบ memory และเตือนว่า log ของ Step จะอ่านผ่าน API ไม่ได้
func TestNewStepLogger_WithoutURLWarnsAndUsesMemory(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	l, closeDB, err := newStepLogger(context.Background(), "")
	if err != nil {
		t.Fatalf("newStepLogger: %v", err)
	}
	defer closeDB()
	if _, ok := l.(*memlogger.Logger); !ok {
		t.Errorf("Step logger without database = %T, want the memory implementation", l)
	}
	if !strings.Contains(buf.String(), "BOP_DATABASE_URL") {
		t.Errorf("log = %q, want a warning that mentions BOP_DATABASE_URL", buf.String())
	}
}

// TestNewStepLogger_WithURLStoresInPostgres ยืนยันว่า log ของ Step ที่ bop-worker เขียนลง database
// ที่ตั้งไว้ — worker ไม่รัน migration เอง (เป็นหน้าที่ของ bop-api) test จึง migrate ให้ก่อน
// รันเฉพาะเมื่อตั้ง BOP_TEST_DATABASE_URL
func TestNewStepLogger_WithURLStoresInPostgres(t *testing.T) {
	url := os.Getenv("BOP_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("BOP_TEST_DATABASE_URL not set: skipping PostgreSQL test")
	}
	if err := postgres.Migrate(url); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	ctx := context.Background()

	l, closeDB, err := newStepLogger(ctx, url)
	if err != nil {
		t.Fatalf("newStepLogger: %v", err)
	}
	defer closeDB()

	owner := logger.StepAttempt{WorkflowID: "worker-main-test", StepIndex: 0, Attempt: 1}
	l.LogStep(ctx, owner, logger.LevelInfo, "stored")
	lines, err := l.StepHistory(ctx, owner.WorkflowID, owner.StepIndex)
	if err != nil {
		t.Fatalf("StepHistory: %v", err)
	}
	if len(lines) == 0 || lines[len(lines)-1].Message != "stored" {
		t.Errorf("StepHistory = %+v, want the stored line", lines)
	}
}
