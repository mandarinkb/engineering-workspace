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

	workflowID := fmt.Sprintf("%s-%s", workflowName, idgen.New())
	runID, err := s.engine.Start(ctx, workflowID, workflow.Workflow{Name: workflowName, Steps: steps})
	if err != nil {
		return nil, err
	}
	return &StartResult{WorkflowID: workflowID, RunID: runID}, nil
}

// List คืน workflow run ทั้งหมด (ทั้งที่กำลังรันอยู่และจบไปแล้ว)
func (s *Service) List(ctx context.Context) ([]workflow.RunSummary, error) {
	return s.engine.ListRuns(ctx)
}

// Get คืนรายละเอียด run เดียวรวมผลลัพธ์ทีละ step — id คือ workflow ID
func (s *Service) Get(ctx context.Context, id string) (*workflow.RunDetail, error) {
	return s.engine.GetRun(ctx, id)
}

// Ready คืน error ถ้าต่อ engine ไม่ได้ (ใช้กับ readiness probe)
func (s *Service) Ready(ctx context.Context) error {
	return s.engine.CheckHealth(ctx)
}
