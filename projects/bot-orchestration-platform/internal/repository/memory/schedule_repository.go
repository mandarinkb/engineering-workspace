package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"bop/internal/schedule"
	"bop/internal/workflow"
)

// ScheduleRepository คือ in-memory implementation ของ schedule.Repository — โครงสร้าง
// และเหตุผลการ copy เข้า-ออกทุกครั้งเหมือนกับ JobRepository/RunRepository เดิมทุกประการ
// (ดูคำอธิบายละเอียดที่ job_repository.go) ต่างกันแค่ entity ที่เก็บ
type ScheduleRepository struct {
	mu    sync.RWMutex
	items map[string]*schedule.Definition
}

// NewScheduleRepository สร้าง repository เปล่าๆ พร้อมใช้งาน
func NewScheduleRepository() *ScheduleRepository {
	return &ScheduleRepository{items: make(map[string]*schedule.Definition)}
}

func (r *ScheduleRepository) Create(_ context.Context, d *schedule.Definition) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.items[d.ID]; exists {
		return fmt.Errorf("memory: schedule %s already exists", d.ID)
	}
	r.items[d.ID] = cloneDefinition(d)
	return nil
}

func (r *ScheduleRepository) Update(_ context.Context, d *schedule.Definition) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.items[d.ID]; !exists {
		return fmt.Errorf("memory: schedule %s not found", d.ID)
	}
	r.items[d.ID] = cloneDefinition(d)
	return nil
}

func (r *ScheduleRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.items[id]; !exists {
		return fmt.Errorf("memory: schedule %s not found", id)
	}
	delete(r.items, id)
	return nil
}

func (r *ScheduleRepository) Get(_ context.Context, id string) (*schedule.Definition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.items[id]
	if !ok {
		return nil, fmt.Errorf("memory: schedule %s not found", id)
	}
	return cloneDefinition(d), nil
}

func (r *ScheduleRepository) List(_ context.Context) ([]*schedule.Definition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*schedule.Definition, 0, len(r.items))
	for _, d := range r.items {
		out = append(out, cloneDefinition(d))
	}
	sort.Slice(out, func(i, k int) bool { return out[i].ID < out[k].ID })
	return out, nil
}

// cloneDefinition คัดลอก Definition แบบลึกพอที่จะไม่แชร์ backing array ของ slice Steps
// กับต้นฉบับ — เหตุผลเดียวกับ cloneRun เดิมใน run_repository.go (ที่ถูกลบไปแล้วตอนย้าย
// ไป Temporal): ผู้เรียก Get/List ไม่ควรมีทาง mutate ข้อมูลที่ repository เก็บไว้ภายใน
// โดยไม่ตั้งใจผ่าน slice ที่ยังชี้ไป backing array เดิม
func cloneDefinition(d *schedule.Definition) *schedule.Definition {
	cp := *d
	cp.Steps = append([]workflow.Step(nil), d.Steps...)
	return &cp
}
