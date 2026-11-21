package logretention_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"bop/internal/usecase/logretention"
)

// fakePruner จำ cutoff ทุกครั้งที่ถูกเรียก และส่งสัญญาณออก channel ให้ test รอได้
type fakePruner struct {
	mu      sync.Mutex
	cutoffs []time.Time
	called  chan struct{}
	err     error
}

func newFakePruner() *fakePruner {
	return &fakePruner{called: make(chan struct{}, 1)}
}

func (p *fakePruner) DeleteBefore(_ context.Context, cutoff time.Time) (int64, error) {
	p.mu.Lock()
	p.cutoffs = append(p.cutoffs, cutoff)
	p.mu.Unlock()
	select {
	case p.called <- struct{}{}:
	default:
	}
	return 3, p.err
}

func (p *fakePruner) calls() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.cutoffs)
}

func TestPruneOnce_DeletesLinesOlderThan30Days(t *testing.T) {
	now := time.Date(2026, 11, 21, 3, 0, 0, 0, time.UTC)
	p := newFakePruner()
	s := logretention.NewService(p, logretention.WithClock(func() time.Time { return now }))

	deleted, err := s.PruneOnce(context.Background())
	if err != nil {
		t.Fatalf("PruneOnce: %v", err)
	}
	if deleted != 3 {
		t.Errorf("deleted = %d, want 3", deleted)
	}
	want := time.Date(2026, 10, 22, 3, 0, 0, 0, time.UTC) // 30 วันก่อน now
	if len(p.cutoffs) != 1 || !p.cutoffs[0].Equal(want) {
		t.Errorf("cutoffs = %v, want [%v]", p.cutoffs, want)
	}
}

func TestPruneOnce_ReturnsPrunerError(t *testing.T) {
	p := newFakePruner()
	p.err = errors.New("db down")
	if _, err := logretention.NewService(p).PruneOnce(context.Background()); err == nil {
		t.Error("PruneOnce: want error, got nil")
	}
}

// TestRun_PrunesAtStartAndOnEveryInterval: bop-api restart บ่อยตอน deploy จึงลบทันทีตอน start
// แล้วลบซ้ำทุก interval (วันละครั้งในของจริง) จน ctx ถูกยกเลิก — error ไม่หยุดลูป
func TestRun_PrunesAtStartAndOnEveryInterval(t *testing.T) {
	p := newFakePruner()
	p.err = errors.New("db down")
	s := logretention.NewService(p, logretention.WithInterval(10*time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.Run(ctx)
	}()

	deadline := time.After(2 * time.Second)
	for p.calls() < 3 {
		select {
		case <-p.called:
		case <-deadline:
			t.Fatalf("pruned %d times, want at least 3", p.calls())
		}
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after ctx was cancelled")
	}
}

func TestDefaults(t *testing.T) {
	if logretention.MaxAge != 30*24*time.Hour {
		t.Errorf("MaxAge = %v, want 30 days", logretention.MaxAge)
	}
	if logretention.Interval != 24*time.Hour {
		t.Errorf("Interval = %v, want 24h", logretention.Interval)
	}
}
