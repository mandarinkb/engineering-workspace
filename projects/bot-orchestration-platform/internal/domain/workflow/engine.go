package workflow

import (
	"context"
	"time"
)

// RunSummary คือข้อมูลสรุปของ workflow run หนึ่งครั้ง (ทั้งที่ schedule สั่งอัตโนมัติและที่
// สั่งแบบ ad-hoc) — เป็น read model ของ domain ที่ไม่ผูกกับ protobuf type ของ Temporal
// (adapter ฝั่ง Temporal เป็นคนแปลง WorkflowExecutionInfo มาเป็น type นี้)
type RunSummary struct {
	WorkflowID   string
	RunID        string
	WorkflowName string
	Status       string
	StartedAt    time.Time
	EndedAt      *time.Time // nil ตราบใดที่ run ยังไม่จบ
}

// RunDetail ต่อยอดจาก RunSummary เพิ่มผลลัพธ์ทีละ step — Steps เป็น nil ได้ถ้า engine
// ตอบ query ผลลัพธ์ไม่ได้ (เช่น run เพิ่งเริ่ม หรือ history หมด retention ไปแล้ว)
type RunDetail struct {
	RunSummary
	Steps []StepResult
}

// Engine คือ port ของ "ตัวรัน workflow" ที่ usecase ใช้ — ตอนนี้ implement โดย Temporal
// (internal/external/temporal) usecase จึงไม่ต้อง import Temporal SDK เลย
type Engine interface {
	// Start เริ่ม workflow run ใหม่ด้วย workflow ID ที่ระบุ แล้ว return ทันทีโดยไม่รอผล
	Start(ctx context.Context, workflowID string, wf Workflow) (runID string, err error)
	// ListRuns คืน workflow run ทั้งหมด เรียงจากล่าสุดไปเก่าสุด
	ListRuns(ctx context.Context) ([]RunSummary, error)
	// GetRun คืนรายละเอียด run เดียวตาม workflow ID
	GetRun(ctx context.Context, workflowID string) (*RunDetail, error)
	// CheckHealth คืน error ถ้าต่อ engine ไม่ได้ (ใช้กับ readiness probe)
	CheckHealth(ctx context.Context) error
}
