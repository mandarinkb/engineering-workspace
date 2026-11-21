// Package workflowrun คือ usecase ของ workflow run — สั่งรัน workflow แบบ ad-hoc และอ่าน
// สถานะ run ผ่าน workflow.Engine (ตอนนี้คือ Temporal) โดยไม่ import Temporal SDK เลย
package workflowrun

import (
	"context"
	"fmt"

	"bop/internal/domain/apperr"
	"bop/internal/domain/workflow"
	"bop/internal/pkg/idgen"
)

// Service รวม use case ทั้งหมดของ workflow run
type Service struct {
	engine workflow.Engine
}

// NewService ผูก use case เข้ากับ engine ที่ส่งเข้ามา (ประกอบใน cmd/bop/main.go)
func NewService(engine workflow.Engine) *Service {
	return &Service{engine: engine}
}

// StartResult คือผลของการสั่งรัน — มีแค่ ID ที่พอเอาไป poll สถานะต่อผ่าน Get
type StartResult struct {
	WorkflowID string
	RunID      string
}

// Start สั่งรัน workflow (1 หรือหลาย step เรียงกัน) ทันทีแบบ ad-hoc — ต่างจาก schedule
// ตรงที่ไม่สร้างอะไรที่ "รันซ้ำตามตารางเวลา" เลย แค่เริ่ม 1 run แล้วจบ เหมาะกับทั้ง (1)
// อยากรัน step ชุดหนึ่งครั้งเดียวโดยไม่ต้องสร้าง/ลบ schedule และ (2) "job เดี่ยว" ที่ต้อง
// route ไปยัง Temporal Task Queue อื่น (ผ่าน Step.TaskQueue) ซึ่ง job.Executor ทำไม่ได้
//
// ผูก workflow ID กับ workflowName + ID สุ่ม เพื่อกัน ID ชนกันเวลาสั่งซ้ำหลายครั้งด้วยชื่อ
// เดียวกัน — ต่างจาก schedule ที่ใช้ ID ตายตัวจาก client เพราะที่นั่น ID ต้อง stable ข้าม
// request แต่ที่นี่แต่ละครั้งคือ run ใหม่ที่ไม่เกี่ยวข้องกับ run ก่อนหน้าเลย
func (s *Service) Start(ctx context.Context, workflowName string, steps []workflow.Step) (*StartResult, error) {
	if workflowName == "" {
		return nil, apperr.Invalid("workflow_name is required")
	}
	if len(steps) == 0 {
		return nil, apperr.Invalid("steps must not be empty")
	}
	if err := workflow.ValidateSteps(steps); err != nil {
		return nil, apperr.WrapInvalid(err)
	}

	workflowID := fmt.Sprintf("%s-%s", workflowName, idgen.New())
	runID, err := s.engine.Start(ctx, workflowID, workflow.Workflow{Name: workflowName, Steps: steps})
	if err != nil {
		return nil, err
	}
	return &StartResult{WorkflowID: workflowID, RunID: runID}, nil
}

// ขนาดหน้าของรายการ Workflow Run — schedule รายนาทีสร้าง run วันละ 1,440 ตัว จึงต้อง
// แบ่งหน้าเสมอ ขอเกิน maxPageSize ถูกปัดลงแทนที่จะ error เพื่อให้ client ขอก้อนใหญ่ได้
// โดยไม่ต้องรู้เพดาน
const (
	defaultPageSize = 50
	maxPageSize     = 200
)

// List คืน Workflow Run หนึ่งหน้า (ทั้งที่กำลังรันอยู่และจบไปแล้ว) — PageSize 0 แปลว่า
// ไม่ได้ระบุ ใช้ค่าเริ่มต้น 50, ติดลบหรือสถานะที่ไม่รู้จักเป็น apperr.ErrInvalid
func (s *Service) List(ctx context.Context, q workflow.RunQuery) (workflow.RunPage, error) {
	if err := validateFilter(q.Filter); err != nil {
		return workflow.RunPage{}, err
	}
	switch {
	case q.PageSize < 0:
		return workflow.RunPage{}, apperr.Invalid("page_size must be a positive integer")
	case q.PageSize == 0:
		q.PageSize = defaultPageSize
	case q.PageSize > maxPageSize:
		q.PageSize = maxPageSize
	}
	return s.engine.ListRuns(ctx, q)
}

// Get คืนรายละเอียด run เดียวรวมผลลัพธ์ทีละ step — id คือ workflow ID
func (s *Service) Get(ctx context.Context, id string) (*workflow.RunDetail, error) {
	return s.engine.GetRun(ctx, id)
}

// Ready คืน error ถ้าต่อ engine ไม่ได้ (ใช้กับ readiness probe)
func (s *Service) Ready(ctx context.Context) error {
	return s.engine.CheckHealth(ctx)
}

// validateFilter ตรวจเงื่อนไขกรองที่ใช้ร่วมกันทุก use case ที่อ่านรายการ Workflow Run —
// ScheduleID รับได้ทุกค่าเพราะ Schedule ID ไม่มีรูปแบบบังคับ (engine เป็นคน escape เอง)
func validateFilter(f workflow.RunFilter) error {
	if !f.Status.Valid() {
		return apperr.Invalid("status must be one of running, completed, failed, canceled")
	}
	return nil
}
