// Package schedule คือ usecase ของการจัดการ schedule (สร้าง/แก้ไข/ลบ/pause/trigger ทันที/
// backfill) — scheduling logic จริงทั้งหมด (cron parsing, calendar rule, overlap policy,
// catchup window) อยู่ที่ schedule.Scheduler (ตอนนี้คือ Temporal Schedules) สิ่งที่ usecase
// นี้ทำเพิ่มคือ validate input และเก็บ "workflow definition" (ชื่อ + steps) คู่กันผ่าน
// schedule.Repository (ดูเหตุผลที่ internal/domain/schedule/schedule.go)
package schedule

import (
	"context"
	"errors"
	"time"

	"bop/internal/domain/apperr"
	"bop/internal/domain/schedule"
	"bop/internal/domain/workflow"
)

// Service รวม use case ทั้งหมดของ schedule
type Service struct {
	scheduler schedule.Scheduler
	repo      schedule.Repository
}

// NewService ผูก use case เข้ากับ port ที่ส่งเข้ามา (ประกอบใน cmd/bop/main.go)
func NewService(scheduler schedule.Scheduler, repo schedule.Repository) *Service {
	return &Service{scheduler: scheduler, repo: repo}
}

// Input คือข้อมูลที่ต้องใช้ตอนสร้างหรือแทนที่ schedule ทั้งก้อน
type Input struct {
	WorkflowName string
	Steps        []workflow.Step
	Spec         schedule.Spec
	Overlap      string // ชื่อ policy จาก client ค่าว่าง = skip
}

// validate ตรวจ input ตามลำดับเดิมของ API (steps -> spec -> overlap) แล้วคืน overlap ที่
// แปลงค่าว่างเป็น skip แล้ว — error ทุกตัวเป็น apperr.ErrInvalid
func (in Input) validate() (schedule.Overlap, error) {
	if len(in.Steps) == 0 {
		return "", apperr.Invalid("steps must not be empty")
	}
	if err := in.Spec.Validate(); err != nil {
		return "", apperr.WrapInvalid(err)
	}
	overlap, err := schedule.ParseOverlap(in.Overlap)
	if err != nil {
		return "", apperr.WrapInvalid(err)
	}
	if overlap == schedule.OverlapUnspecified {
		overlap = schedule.OverlapSkip
	}
	return overlap, nil
}

// Create สร้าง schedule ใหม่ที่ scheduler แล้วเก็บ workflow definition ไว้ใน repository
// คู่กัน — ลำดับสำคัญ: สร้างที่ scheduler ก่อนเสมอ (เป็น source of truth ตัวจริง) แล้วค่อย
// บันทึก definition ของเราทีหลัง ถ้าขั้นหลังพลาด (ไม่ควรเกิดในทางปกติ) จะ compensate ด้วย
// การลบ schedule ที่เพิ่งสร้างทิ้ง ไม่ปล่อยให้เหลือ schedule กำพร้าที่ไม่มี definition คู่กัน
// (Saga-style compensation เดียวกับแนวคิดใน FULLSTACK-INFRA-ROADMAP.md Phase 3)
//
// error จาก repository และจาก describe หลังสร้างเสร็จห่อด้วย apperr.Plain เพราะ API เดิม
// ตอบ 500 ในสองขั้นตอนนี้เสมอไม่ว่า error ข้างในจะเป็นประเภทไหน
func (s *Service) Create(ctx context.Context, id string, in Input) (*schedule.Detail, error) {
	if id == "" {
		return nil, apperr.Invalid("id is required")
	}
	overlap, err := in.validate()
	if err != nil {
		return nil, err
	}

	wf := workflow.Workflow{Name: in.WorkflowName, Steps: in.Steps}
	if err := s.scheduler.Create(ctx, id, in.Spec, overlap, wf); err != nil {
		if errors.Is(err, apperr.ErrConflict) {
			return nil, err
		}
		return nil, apperr.Plain(err)
	}

	def := &schedule.Definition{ID: id, WorkflowName: in.WorkflowName, Steps: in.Steps, CreatedAt: time.Now()}
	if err := s.repo.Create(ctx, def); err != nil {
		_ = s.scheduler.Delete(ctx, id)
		return nil, apperr.Plain(err)
	}

	detail, err := s.Get(ctx, id)
	return detail, apperr.Plain(err)
}

// List คืนรายการ schedule ทั้งหมดเท่าที่ scheduler list ให้ได้ (ไม่ describe ทีละตัว เร็ว
// กว่ามากถ้ามี schedule จำนวนมาก)
func (s *Service) List(ctx context.Context) ([]schedule.Summary, error) {
	return s.scheduler.List(ctx)
}

// Get รวมข้อมูลจาก 2 แหล่งเข้าด้วยกัน: scheduler (สถานะ/เวลาที่แท้จริง) และ repository ของ
// BOP เอง (workflow steps) — ถ้าไม่เจอ definition (schedule นี้สร้างก่อนมี API ชุดนี้ เช่น
// check-example-com-schedule ที่ cmd/bop-worker สร้างตรงผ่าน SDK, หรือ cmd/bop เพิ่ง restart
// เพราะ repository เป็น in-memory) ปล่อย Steps เป็น nil เงียบๆ ไม่ใช่ fatal error — schedule
// ยังใช้งานได้ปกติ แค่ frontend จะไม่รู้ว่า steps ข้างในคืออะไรเท่านั้น
func (s *Service) Get(ctx context.Context, id string) (*schedule.Detail, error) {
	detail, err := s.scheduler.Describe(ctx, id)
	if err != nil {
		return nil, err
	}
	if def, err := s.repo.Get(ctx, id); err == nil {
		detail.Steps = def.Steps
	}
	return detail, nil
}

// Update แทนที่ schedule ทั้งก้อนด้วยค่าใหม่ (full replace ไม่ใช่ partial patch)
func (s *Service) Update(ctx context.Context, id string, in Input) (*schedule.Detail, error) {
	overlap, err := in.validate()
	if err != nil {
		return nil, err
	}

	wf := workflow.Workflow{Name: in.WorkflowName, Steps: in.Steps}
	if err := s.scheduler.Update(ctx, id, in.Spec, overlap, wf); err != nil {
		return nil, err
	}

	def := &schedule.Definition{ID: id, WorkflowName: in.WorkflowName, Steps: in.Steps, CreatedAt: time.Now()}
	// ใช้ Update() ก่อน ถ้าไม่เจอ definition เดิม (schedule นี้สร้างก่อนมี API ชุดนี้) ให้
	// Create() แทน — ไม่ปล่อยให้การแก้ที่สำเร็จฝั่ง scheduler แล้วมาล้มเหลวเพราะ bookkeeping
	// ของเราเองยังไม่เคยมี record นี้มาก่อน
	if err := s.repo.Update(ctx, def); err != nil {
		_ = s.repo.Create(ctx, def)
	}

	detail, err := s.Get(ctx, id)
	return detail, apperr.Plain(err)
}

// Delete ลบ schedule ทิ้งทั้งฝั่ง scheduler และ definition ของเราเอง — ลบที่ scheduler ก่อน
// เสมอ (source of truth) ถ้าสำเร็จค่อยลบ definition (ไม่ fatal ถ้าไม่เจอ — schedule อาจไม่
// เคยมี definition ใน repository นี้อยู่แล้วตั้งแต่ต้น)
func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.scheduler.Delete(ctx, id); err != nil {
		return err
	}
	_ = s.repo.Delete(ctx, id)
	return nil
}

// Pause/Unpause เทียบเท่า "Hold"/"Free" ของ Control-M — หยุดชั่วคราวโดยไม่ลบ schedule ทิ้ง
func (s *Service) Pause(ctx context.Context, id, note string) error {
	return s.scheduler.Pause(ctx, id, note)
}

func (s *Service) Unpause(ctx context.Context, id, note string) error {
	return s.scheduler.Unpause(ctx, id, note)
}

// Trigger สั่งรันทันที นอกตารางเวลาปกติ — เทียบเท่า "Order Now" ของ Control-M ไม่กระทบ
// ตารางเวลาปกติ (รอบถัดไปยังมาตามเวลาเดิม) overlap ว่าง = ใช้ policy default ของ schedule
func (s *Service) Trigger(ctx context.Context, id, overlapName string) error {
	overlap, err := schedule.ParseOverlap(overlapName)
	if err != nil {
		return apperr.WrapInvalid(err)
	}
	return s.scheduler.Trigger(ctx, id, overlap)
}

// Backfill เทียบเท่า "Rerun" ของ Control-M: ประเมิน schedule ย้อนหลังในช่วงเวลาที่ระบุ แล้ว
// รัน action ทุกครั้งที่ควรจะเกิดขึ้นในช่วงนั้น ราวกับเวลานั้นเพิ่งผ่านไปตอนนี้
func (s *Service) Backfill(ctx context.Context, id string, start, end time.Time, overlapName string) error {
	if !start.Before(end) {
		return apperr.Invalid("start_time must be before end_time")
	}
	overlap, err := schedule.ParseOverlap(overlapName)
	if err != nil {
		return apperr.WrapInvalid(err)
	}
	return s.scheduler.Backfill(ctx, id, start, end, overlap)
}
