// Package schedule คือ usecase ของการจัดการ schedule (สร้าง/แก้ไข/ลบ/pause/trigger ทันที/
// backfill) — scheduling logic จริงทั้งหมด (cron parsing, calendar rule, overlap policy,
// catchup window) อยู่ที่ schedule.Scheduler (ตอนนี้คือ Temporal Schedules) สิ่งที่ usecase
// นี้ทำเพิ่มคือ validate input — ข้อมูลของ Schedule ทั้งหมดรวม steps อ่านจาก scheduler ที่
// เดียว BOP ไม่เก็บซ้ำเอง (ADR 0001)
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
}

// NewService ผูก use case เข้ากับ port ที่ส่งเข้ามา (ประกอบใน cmd/bop/main.go)
func NewService(scheduler schedule.Scheduler) *Service {
	return &Service{scheduler: scheduler}
}

// Input คือข้อมูลที่ต้องใช้ตอนสร้างหรือแทนที่ schedule ทั้งก้อน
type Input struct {
	WorkflowName string
	Steps        []workflow.Step
	Spec         schedule.Spec
	Overlap      string // ชื่อ policy จาก client ค่าว่าง = skip
	// CatchupWindowSeconds คือ Catch-up Window เป็นวินาที nil = schedule.DefaultCatchupWindow
	// ใช้ทั้งตอนสร้างและแก้ไข Schedule เดิมที่ไม่เคยตั้งจึงได้ค่า default เมื่อถูกแก้ไข
	CatchupWindowSeconds *int64
}

// validate ตรวจ input ตามลำดับเดิมของ API (steps -> spec -> overlap -> catch-up window)
// แล้วคืน policy ที่เติมค่า default แล้ว (overlap ว่าง = skip, Catch-up Window ว่าง = 1
// ชั่วโมง) — error ทุกตัวเป็น apperr.ErrInvalid
func (in Input) validate() (schedule.Policy, error) {
	if len(in.Steps) == 0 {
		return schedule.Policy{}, apperr.Invalid("steps must not be empty")
	}
	if err := in.Spec.Validate(); err != nil {
		return schedule.Policy{}, apperr.WrapInvalid(err)
	}
	overlap, err := schedule.ParseOverlap(in.Overlap)
	if err != nil {
		return schedule.Policy{}, apperr.WrapInvalid(err)
	}
	if overlap == schedule.OverlapUnspecified {
		overlap = schedule.OverlapSkip
	}
	catchupWindow, err := schedule.ParseCatchupWindow(in.CatchupWindowSeconds)
	if err != nil {
		return schedule.Policy{}, apperr.WrapInvalid(err)
	}
	return schedule.Policy{Overlap: overlap, CatchupWindow: catchupWindow}, nil
}

// Create สร้าง schedule ใหม่ที่ scheduler แล้วอ่านรายละเอียดกลับมาตอบ
//
// error จาก describe หลังสร้างเสร็จห่อด้วย apperr.Plain เพราะ API เดิมตอบ 500 ในขั้นนี้
// เสมอไม่ว่า error ข้างในจะเป็นประเภทไหน
func (s *Service) Create(ctx context.Context, id string, in Input) (*schedule.Detail, error) {
	if id == "" {
		return nil, apperr.Invalid("id is required")
	}
	policy, err := in.validate()
	if err != nil {
		return nil, err
	}

	wf := workflow.Workflow{Name: in.WorkflowName, Steps: in.Steps}
	if err := s.scheduler.Create(ctx, id, in.Spec, policy, wf); err != nil {
		if errors.Is(err, apperr.ErrConflict) {
			return nil, err
		}
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

// Get คืนรายละเอียดของ schedule จาก scheduler ที่เดียว รวม steps ที่จะรันจริง — ใช้ได้กับ
// Schedule ที่สร้างนอก API (เช่น check-example-com-schedule ของ cmd/bop-worker) และไม่
// เปลี่ยนหลัง cmd/bop restart
func (s *Service) Get(ctx context.Context, id string) (*schedule.Detail, error) {
	return s.scheduler.Describe(ctx, id)
}

// Update แทนที่ schedule ทั้งก้อนด้วยค่าใหม่ (full replace ไม่ใช่ partial patch)
func (s *Service) Update(ctx context.Context, id string, in Input) (*schedule.Detail, error) {
	policy, err := in.validate()
	if err != nil {
		return nil, err
	}

	wf := workflow.Workflow{Name: in.WorkflowName, Steps: in.Steps}
	if err := s.scheduler.Update(ctx, id, in.Spec, policy, wf); err != nil {
		return nil, err
	}

	detail, err := s.Get(ctx, id)
	return detail, apperr.Plain(err)
}

// Delete ลบ schedule ทิ้งถาวรที่ scheduler
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.scheduler.Delete(ctx, id)
}

// Pause หยุด schedule ชั่วคราวโดยไม่ลบทิ้ง — เทียบเท่า "Hold" ของ Control-M
func (s *Service) Pause(ctx context.Context, id, note string) error {
	return s.scheduler.Pause(ctx, id, note)
}

// Unpause เริ่ม schedule ที่ pause ไว้ใหม่ — เทียบเท่า "Free" ของ Control-M
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
