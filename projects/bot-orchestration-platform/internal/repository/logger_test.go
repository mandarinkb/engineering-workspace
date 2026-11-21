package repository_test

import (
	"context"
	"testing"
	"time"

	"bop/internal/domain/job"
	"bop/internal/domain/logger"
	memlogger "bop/internal/pkg/logger"
	"bop/internal/repository/postgres"
	"bop/internal/usecase/logretention"
)

// contract ของ logger.Logger: memory กับ PostgreSQL ต้องตอบเหมือนกันทุกข้อ เพื่อให้ cmd/bop
// สลับตาม BOP_DATABASE_URL ได้โดย worker pool และ handler REST/SSE ไม่ต้องแก้

func TestLoggerContract_Memory(t *testing.T) {
	runLoggerContract(t, func(*testing.T) logger.Logger { return memlogger.New() })
}

func TestLoggerContract_Postgres(t *testing.T) {
	url := testDatabaseURL(t)
	runLoggerContract(t, func(t *testing.T) logger.Logger {
		return postgres.NewLogger(openCleanDatabase(t, url))
	})
}

func runLoggerContract(t *testing.T, newLogger func(*testing.T) logger.Logger) {
	t.Run("History returns the Job's lines in the order they were logged", func(t *testing.T) {
		l := newLogger(t)
		ctx := context.Background()
		l.Log(ctx, "job-a", logger.LevelInfo, "first")
		l.Log(ctx, "job-b", logger.LevelInfo, "other job")
		l.Log(ctx, "job-a", logger.LevelWarn, "second")
		l.Log(ctx, "job-a", logger.LevelError, "third")

		lines, err := l.History(ctx, "job-a")
		if err != nil {
			t.Fatalf("History: %v", err)
		}
		want := []struct {
			level   logger.Level
			message string
		}{{logger.LevelInfo, "first"}, {logger.LevelWarn, "second"}, {logger.LevelError, "third"}}
		if len(lines) != len(want) {
			t.Fatalf("History = %+v, want %d lines", lines, len(want))
		}
		for i, w := range want {
			if lines[i].JobID != "job-a" || lines[i].Level != w.level || lines[i].Message != w.message {
				t.Errorf("line %d = %+v, want job-a %s %q", i, lines[i], w.level, w.message)
			}
			if lines[i].Time.IsZero() {
				t.Errorf("line %d has no time", i)
			}
		}
	})

	t.Run("History of a Job without logs is an empty slice, not nil", func(t *testing.T) {
		// GET /jobs/{id}/logs ส่งผลเป็น JSON ตรงๆ — nil กลายเป็น null
		lines, err := newLogger(t).History(context.Background(), "no-logs")
		if err != nil {
			t.Fatalf("History: %v", err)
		}
		if lines == nil || len(lines) != 0 {
			t.Errorf("History = %#v, want empty non-nil slice", lines)
		}
	})

	t.Run("a subscriber receives the same line that History returns", func(t *testing.T) {
		l := newLogger(t)
		ctx := context.Background()
		ch, unsubscribe := l.Subscribe("job-a")
		defer unsubscribe()

		l.Log(ctx, "job-a", logger.LevelInfo, "live")
		var live logger.Line
		select {
		case live = <-ch:
		case <-time.After(2 * time.Second):
			t.Fatal("subscriber received nothing")
		}
		lines, err := l.History(ctx, "job-a")
		if err != nil {
			t.Fatalf("History: %v", err)
		}
		if len(lines) != 1 {
			t.Fatalf("History = %+v, want 1 line", lines)
		}
		// dashboard รวม backfill กับ live ด้วยเวลา — เวลาที่ stream ส่งต้องเท่ากับที่อ่านกลับได้
		if live.Message != "live" || !live.Time.Equal(lines[0].Time) {
			t.Errorf("live = %+v, history = %+v, want same message and time", live, lines[0])
		}
	})

	t.Run("unsubscribe closes the channel and is safe to call twice", func(t *testing.T) {
		ch, unsubscribe := newLogger(t).Subscribe("job-a")
		unsubscribe()
		if _, ok := <-ch; ok {
			t.Fatal("channel still open after unsubscribe")
		}
		unsubscribe()
	})

	t.Run("a line logged with a cancelled context is still stored", func(t *testing.T) {
		// worker pool log "job cancelled" ด้วย context ของ Job ที่เพิ่งถูกยกเลิก
		l := newLogger(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		l.Log(ctx, "job-a", logger.LevelWarn, "job cancelled")

		lines, err := l.History(context.Background(), "job-a")
		if err != nil {
			t.Fatalf("History: %v", err)
		}
		if len(lines) != 1 || lines[0].Message != "job cancelled" {
			t.Errorf("History = %+v, want the cancelled line", lines)
		}
	})
}

// TestPostgresLogger_SurvivesReconnect จำลอง restart ของ bop-api: log ผ่าน pool ชุดหนึ่ง ปิดทิ้ง
// แล้วเปิดชุดใหม่ (รัน migration ซ้ำแบบตอน start) ต้องยังอ่าน log เดิมได้ครบ
func TestPostgresLogger_SurvivesReconnect(t *testing.T) {
	url := testDatabaseURL(t)
	ctx := context.Background()

	before := openCleanDatabase(t, url)
	l := postgres.NewLogger(before)
	l.Log(ctx, "j-restart", logger.LevelInfo, "job started")
	l.Log(ctx, "j-restart", logger.LevelInfo, "job succeeded: ok")
	before.Close()

	if err := postgres.Migrate(url); err != nil {
		t.Fatalf("Migrate again: %v", err)
	}
	after, err := postgres.Connect(ctx, url)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(after.Close)

	lines, err := postgres.NewLogger(after).History(ctx, "j-restart")
	if err != nil {
		t.Fatalf("History after reconnect: %v", err)
	}
	if len(lines) != 2 || lines[0].Message != "job started" || lines[1].Message != "job succeeded: ok" {
		t.Errorf("History after reconnect = %+v, want both lines in order", lines)
	}
}

// TestLogRetention_DeletesOnlyLinesOlderThan30Days รันงานลบของ bop-api กับ PostgreSQL จริง:
// log line ที่เก่ากว่า 30 วันหาย log ที่ใหม่กว่ายังอยู่ และ Job เจ้าของ log ยังอ่านสถานะกับ
// summary ได้
func TestLogRetention_DeletesOnlyLinesOlderThan30Days(t *testing.T) {
	url := testDatabaseURL(t)
	ctx := context.Background()
	db := openCleanDatabase(t, url)
	now := time.Date(2026, 11, 21, 3, 0, 0, 0, time.UTC)

	created := now.Add(-40 * 24 * time.Hour)
	j := &job.Job{ID: "j-old", BotName: "b", Status: job.StatusPending, CreatedAt: created}
	jobs := postgres.NewJobRepository(db)
	if err := jobs.Create(ctx, j); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := j.Start(created); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := j.Succeed("ok", created.Add(time.Second)); err != nil {
		t.Fatalf("Succeed: %v", err)
	}
	if err := jobs.Update(ctx, j); err != nil {
		t.Fatalf("Update: %v", err)
	}

	for _, c := range []struct {
		message string
		age     time.Duration
	}{
		{"31 days old", 31 * 24 * time.Hour},
		{"30 days and a minute old", 30*24*time.Hour + time.Minute},
		{"29 days old", 29 * 24 * time.Hour},
		{"just now", 0},
	} {
		at := now.Add(-c.age)
		postgres.NewLogger(db, postgres.WithClock(func() time.Time { return at })).
			Log(ctx, "j-old", logger.LevelInfo, c.message)
	}

	l := postgres.NewLogger(db)
	retention := logretention.NewService(l, logretention.WithClock(func() time.Time { return now }))
	deleted, err := retention.PruneOnce(ctx)
	if err != nil {
		t.Fatalf("PruneOnce: %v", err)
	}
	if deleted != 2 {
		t.Errorf("deleted = %d, want 2", deleted)
	}

	lines, err := l.History(ctx, "j-old")
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	var got []string
	for _, line := range lines {
		got = append(got, line.Message)
	}
	if len(got) != 2 || got[0] != "29 days old" || got[1] != "just now" {
		t.Errorf("remaining lines = %v, want [29 days old, just now]", got)
	}

	kept, err := jobs.Get(ctx, "j-old")
	if err != nil {
		t.Fatalf("Get Job after retention: %v", err)
	}
	if kept.Status != job.StatusSucceeded || kept.Summary != "ok" {
		t.Errorf("Job after retention: status=%q summary=%q, want succeeded/ok", kept.Status, kept.Summary)
	}
}

// contract ของ logger.StepLogger: log ของ Step ที่ bop-worker เขียนต้องอ่านกลับได้ตาม Workflow Run
// และลำดับ Step ครบทุก Attempt ไม่ว่าจะเก็บใน memory หรือ PostgreSQL

func TestStepLoggerContract_Memory(t *testing.T) {
	runStepLoggerContract(t, func(*testing.T) logger.StepLogger { return memlogger.New() })
}

func TestStepLoggerContract_Postgres(t *testing.T) {
	url := testDatabaseURL(t)
	runStepLoggerContract(t, func(t *testing.T) logger.StepLogger {
		return postgres.NewLogger(openCleanDatabase(t, url))
	})
}

func runStepLoggerContract(t *testing.T, newLogger func(*testing.T) logger.StepLogger) {
	t.Run("StepHistory returns every Attempt of the Step in time order", func(t *testing.T) {
		l := newLogger(t)
		ctx := context.Background()
		step := func(attempt int) logger.StepAttempt {
			return logger.StepAttempt{WorkflowID: "wf-a", StepIndex: 0, Attempt: attempt}
		}
		l.LogStep(ctx, step(1), logger.LevelInfo, "requesting")
		l.LogStep(ctx, step(1), logger.LevelError, "connection refused")
		l.LogStep(ctx, logger.StepAttempt{WorkflowID: "wf-a", StepIndex: 1, Attempt: 1},
			logger.LevelInfo, "other step")
		l.LogStep(ctx, logger.StepAttempt{WorkflowID: "wf-b", StepIndex: 0, Attempt: 1},
			logger.LevelInfo, "other run")
		l.LogStep(ctx, step(2), logger.LevelInfo, "requesting again")

		lines, err := l.StepHistory(ctx, "wf-a", 0)
		if err != nil {
			t.Fatalf("StepHistory: %v", err)
		}
		want := []struct {
			attempt int
			level   logger.Level
			message string
		}{
			{1, logger.LevelInfo, "requesting"},
			{1, logger.LevelError, "connection refused"},
			{2, logger.LevelInfo, "requesting again"},
		}
		if len(lines) != len(want) {
			t.Fatalf("StepHistory = %+v, want %d lines", lines, len(want))
		}
		for i, w := range want {
			got := lines[i]
			if got.WorkflowID != "wf-a" || got.StepIndex != 0 || got.Attempt != w.attempt ||
				got.Level != w.level || got.Message != w.message {
				t.Errorf("line %d = %+v, want wf-a step 0 attempt %d %s %q",
					i, got, w.attempt, w.level, w.message)
			}
			if got.Time.IsZero() {
				t.Errorf("line %d has no time", i)
			}
		}
	})

	t.Run("StepHistory of a Step without logs is an empty slice, not nil", func(t *testing.T) {
		// GET /workflow-runs/{id}/steps/{index}/logs ส่งผลเป็น JSON ตรงๆ — nil กลายเป็น null
		lines, err := newLogger(t).StepHistory(context.Background(), "wf-none", 0)
		if err != nil {
			t.Fatalf("StepHistory: %v", err)
		}
		if lines == nil || len(lines) != 0 {
			t.Errorf("StepHistory = %#v, want empty non-nil slice", lines)
		}
	})

	t.Run("a Step line logged with a cancelled context is still stored", func(t *testing.T) {
		// Attempt ที่หมดเวลา (StartToCloseTimeout) log error ด้วย ctx ที่ถูกยกเลิกไปแล้ว
		l := newLogger(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		l.LogStep(ctx, logger.StepAttempt{WorkflowID: "wf-a", StepIndex: 0, Attempt: 1},
			logger.LevelError, "context deadline exceeded")

		lines, err := l.StepHistory(context.Background(), "wf-a", 0)
		if err != nil {
			t.Fatalf("StepHistory: %v", err)
		}
		if len(lines) != 1 || lines[0].Message != "context deadline exceeded" {
			t.Errorf("StepHistory = %+v, want the cancelled line", lines)
		}
	})
}

// TestLogRetention_DeletesOldStepLines ยืนยันว่า log ของ Step อยู่ใต้กฎลบเมื่อเกิน 30 วันเดียวกับ
// log ของ Job
func TestLogRetention_DeletesOldStepLines(t *testing.T) {
	url := testDatabaseURL(t)
	ctx := context.Background()
	db := openCleanDatabase(t, url)
	now := time.Date(2026, 11, 21, 3, 0, 0, 0, time.UTC)

	for _, c := range []struct {
		message string
		age     time.Duration
	}{
		{"31 days old", 31 * 24 * time.Hour},
		{"29 days old", 29 * 24 * time.Hour},
	} {
		at := now.Add(-c.age)
		postgres.NewLogger(db, postgres.WithClock(func() time.Time { return at })).
			LogStep(ctx, logger.StepAttempt{WorkflowID: "wf-old", StepIndex: 0, Attempt: 1},
				logger.LevelInfo, c.message)
	}

	l := postgres.NewLogger(db)
	retention := logretention.NewService(l, logretention.WithClock(func() time.Time { return now }))
	deleted, err := retention.PruneOnce(ctx)
	if err != nil {
		t.Fatalf("PruneOnce: %v", err)
	}
	if deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted)
	}

	lines, err := l.StepHistory(ctx, "wf-old", 0)
	if err != nil {
		t.Fatalf("StepHistory: %v", err)
	}
	if len(lines) != 1 || lines[0].Message != "29 days old" {
		t.Errorf("remaining Step lines = %+v, want only the 29-day-old line", lines)
	}
}

// ข้อความที่มี NUL หรือ UTF-8 เสียทำให้ INSERT ลง text column พลาด — Postgres logger ต้องล้างก่อน
// เก็บ ไม่งั้นบรรทัดหายเงียบๆ (รวมบรรทัด error สุดท้ายของ Failed Step) เลยไม่อยู่ใน contract กลาง
// เพราะ memory logger ไม่มีข้อจำกัดนี้
func TestLoggerPostgres_SanitisesNULAndInvalidUTF8(t *testing.T) {
	url := testDatabaseURL(t)
	l := postgres.NewLogger(openCleanDatabase(t, url))
	ctx := context.Background()
	const dirty = "before\x00mid\xff\xfeafter"
	const clean = "beforemid�after"

	t.Run("Job line", func(t *testing.T) {
		l.Log(ctx, "job-a", logger.LevelError, dirty)
		lines, err := l.History(ctx, "job-a")
		if err != nil {
			t.Fatalf("History: %v", err)
		}
		if len(lines) != 1 || lines[0].Message != clean {
			t.Fatalf("History = %+v, want one line %q", lines, clean)
		}
	})

	t.Run("Step line", func(t *testing.T) {
		step := logger.StepAttempt{WorkflowID: "wf-a", StepIndex: 0, Attempt: 1}
		l.LogStep(ctx, step, logger.LevelError, dirty)
		lines, err := l.StepHistory(ctx, "wf-a", 0)
		if err != nil {
			t.Fatalf("StepHistory: %v", err)
		}
		if len(lines) != 1 || lines[0].Message != clean {
			t.Fatalf("StepHistory = %+v, want one line %q", lines, clean)
		}
	})
}
