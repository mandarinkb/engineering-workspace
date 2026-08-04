package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"bop/internal/workflow"
)

// RunRepository คือ in-memory implementation ของ workflow.RunRepository — อยู่ package
// memory เดียวกับ JobRepository ใน job_repository.go เพราะเป็นความกังวลเดียวกัน (เก็บ
// ข้อมูลใน memory ล้วนๆ) แค่คนละ entity เท่านั้น
type RunRepository struct {
	mu   sync.RWMutex
	runs map[string]*workflow.Run
}

// NewRunRepository สร้าง repository เปล่าๆ พร้อมใช้งาน
func NewRunRepository() *RunRepository {
	return &RunRepository{runs: make(map[string]*workflow.Run)}
}

func (r *RunRepository) Create(_ context.Context, run *workflow.Run) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.runs[run.ID]; exists {
		return fmt.Errorf("memory: run %s already exists", run.ID)
	}
	r.runs[run.ID] = cloneRun(run)
	return nil
}

func (r *RunRepository) Update(_ context.Context, run *workflow.Run) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.runs[run.ID] = cloneRun(run)
	return nil
}

func (r *RunRepository) Get(_ context.Context, id string) (*workflow.Run, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	run, ok := r.runs[id]
	if !ok {
		return nil, fmt.Errorf("memory: run %s not found", id)
	}
	return cloneRun(run), nil
}

func (r *RunRepository) List(_ context.Context) ([]*workflow.Run, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*workflow.Run, 0, len(r.runs))
	for _, run := range r.runs {
		out = append(out, cloneRun(run))
	}
	sort.Slice(out, func(i, k int) bool { return out[i].CreatedAt.After(out[k].CreatedAt) })
	return out, nil
}

// cloneRun คัดลอก Run แบบลึกพอที่จะไม่แชร์ backing array ของ slice Steps กับต้นฉบับ —
// ต่างจาก job.Job ตรงที่ workflow.Run มี field ที่เป็น slice (Steps) ซึ่ง workflow.Runner
// จะ append เข้าไปเรื่อยๆ ระหว่างที่ workflow กำลังทำงาน (ดู internal/workflow/runner.go)
// ถ้า copy แบบตื้น (`cp := *run`) จะได้แค่คัดลอก slice header (pointer+len+cap) ออกมา
// ซึ่งยังชี้ไปที่ backing array เดิมอยู่ — แม้จะพิสูจน์ได้ว่ากรณีนี้ไม่ทำให้ข้อมูลเสีย
// (เพราะ index ที่ถูกอ่านกับ index ที่ถูกเขียนไม่ทับกัน) แต่เพื่อไม่ให้ต้องมานั่งพิสูจน์
// ทุกครั้งว่าปลอดภัยจริง จึงเลือก copy เนื้อหาของ slice ออกมาเป็น backing array ใหม่ทั้งหมด
// ไปเลย ให้เหมือนกับหลักการเดียวกับ JobRepository (ป้องกัน data race แบบไม่ต้องคิดเยอะ)
func cloneRun(run *workflow.Run) *workflow.Run {
	cp := *run
	cp.Steps = append([]workflow.StepResult(nil), run.Steps...)
	return &cp
}
