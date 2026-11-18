// test กฎการเปลี่ยนสถานะของ Job: pending -> running -> succeeded/failed/cancelled
// สถานะที่จบแล้วเปลี่ยนต่อไม่ได้ และข้ามขั้นจาก pending ไปจบเลยไม่ได้
package job_test

import (
	"errors"
	"testing"
	"time"

	"bop/internal/domain/job"
)

var now = time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC)

func pendingJob() *job.Job {
	return &job.Job{ID: "j1", BotName: "stub", Status: job.StatusPending, CreatedAt: now}
}

func runningJob(t *testing.T) *job.Job {
	t.Helper()
	j := pendingJob()
	if err := j.Start(now); err != nil {
		t.Fatalf("Start: %v", err)
	}
	return j
}

func TestStart_FromPending(t *testing.T) {
	j := pendingJob()
	if err := j.Start(now); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if j.Status != job.StatusRunning {
		t.Errorf("status = %q, want running", j.Status)
	}
	if j.StartedAt == nil || !j.StartedAt.Equal(now) {
		t.Errorf("StartedAt = %v, want %v", j.StartedAt, now)
	}
}

func TestStart_RejectsWhenNotPending(t *testing.T) {
	j := runningJob(t)
	if err := j.Start(now); !errors.Is(err, job.ErrInvalidTransition) {
		t.Fatalf("err = %v, want ErrInvalidTransition", err)
	}
}

func TestSucceed_FromRunning(t *testing.T) {
	j := runningJob(t)
	end := now.Add(time.Second)
	if err := j.Succeed("ok", end); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if j.Status != job.StatusSucceeded || j.Summary != "ok" {
		t.Errorf("got status=%q summary=%q", j.Status, j.Summary)
	}
	if j.EndedAt == nil || !j.EndedAt.Equal(end) {
		t.Errorf("EndedAt = %v, want %v", j.EndedAt, end)
	}
}

func TestFail_FromRunning(t *testing.T) {
	j := runningJob(t)
	if err := j.Fail("boom", now); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if j.Status != job.StatusFailed || j.Error != "boom" || j.EndedAt == nil {
		t.Errorf("got status=%q error=%q endedAt=%v", j.Status, j.Error, j.EndedAt)
	}
}

func TestCancel_FromRunning(t *testing.T) {
	j := runningJob(t)
	if err := j.Cancel("cancelled by user", now); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if j.Status != job.StatusCancelled || j.Error != "cancelled by user" || j.EndedAt == nil {
		t.Errorf("got status=%q error=%q endedAt=%v", j.Status, j.Error, j.EndedAt)
	}
}

// การจบงานทุกแบบต้องเริ่มจาก running เท่านั้น — job ที่ยัง pending หรือจบไปแล้วต้องถูกปฏิเสธ
// และต้องไม่แตะ field ใดๆ ของ job เลย
func TestFinish_RejectsWhenNotRunning(t *testing.T) {
	finishers := map[string]func(*job.Job) error{
		"Succeed": func(j *job.Job) error { return j.Succeed("ok", now) },
		"Fail":    func(j *job.Job) error { return j.Fail("boom", now) },
		"Cancel":  func(j *job.Job) error { return j.Cancel("stop", now) },
	}
	starts := map[string]func(t *testing.T) *job.Job{
		"pending":   func(*testing.T) *job.Job { return pendingJob() },
		"succeeded": func(t *testing.T) *job.Job { j := runningJob(t); _ = j.Succeed("done", now); return j },
		"failed":    func(t *testing.T) *job.Job { j := runningJob(t); _ = j.Fail("err", now); return j },
		"cancelled": func(t *testing.T) *job.Job { j := runningJob(t); _ = j.Cancel("x", now); return j },
	}
	for fname, finish := range finishers {
		for sname, start := range starts {
			t.Run(fname+"_from_"+sname, func(t *testing.T) {
				j := start(t)
				before := *j
				if err := finish(j); !errors.Is(err, job.ErrInvalidTransition) {
					t.Fatalf("err = %v, want ErrInvalidTransition", err)
				}
				if j.Status != before.Status || j.Summary != before.Summary ||
					j.Error != before.Error || j.EndedAt != before.EndedAt {
					t.Errorf("job changed after rejected transition: before=%+v after=%+v", before, *j)
				}
			})
		}
	}
}

func TestStatusIsFinal(t *testing.T) {
	cases := map[job.Status]bool{
		job.StatusPending:   false,
		job.StatusRunning:   false,
		job.StatusSucceeded: true,
		job.StatusFailed:    true,
		job.StatusCancelled: true,
	}
	for s, want := range cases {
		if got := s.IsFinal(); got != want {
			t.Errorf("%q.IsFinal() = %v, want %v", s, got, want)
		}
	}
}
