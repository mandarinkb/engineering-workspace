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

// RunStatusFilter คือกลุ่มสถานะที่ใช้กรองรายการ Workflow Run — ค่าว่างคือไม่กรอง
// ส่วน RunStatusFailed รวม failed, terminated และ timed out ไว้ด้วยกัน เพราะคนดูสนใจว่า
// "พังไหม" มากกว่าพังแบบไหน (ตรงกับที่ dashboard ใช้สีแดงกับทั้งสามสถานะ)
type RunStatusFilter string

// กลุ่มสถานะที่ GET /workflow-runs?status= รับได้
const (
	RunStatusAny       RunStatusFilter = ""
	RunStatusRunning   RunStatusFilter = "running"
	RunStatusCompleted RunStatusFilter = "completed"
	RunStatusFailed    RunStatusFilter = "failed"
	RunStatusCanceled  RunStatusFilter = "canceled"
)

// Valid บอกว่าเป็นกลุ่มสถานะที่รู้จักหรือไม่ (ค่าว่าง = ไม่กรอง ถือว่าถูกต้อง)
func (f RunStatusFilter) Valid() bool {
	switch f {
	case RunStatusAny, RunStatusRunning, RunStatusCompleted, RunStatusFailed, RunStatusCanceled:
		return true
	}
	return false
}

// RunQuery คือเงื่อนไขขอรายการ Workflow Run ทีละหน้า — PageToken เป็น opaque string ที่ได้
// จาก RunPage.NextPageToken ของหน้าก่อน (ว่าง = หน้าแรก) usecase ต้อง normalize PageSize
// มาแล้วเสมอ engine ไม่ตั้งค่าเริ่มต้นเอง
type RunQuery struct {
	PageSize  int
	PageToken string
	Status    RunStatusFilter
}

// RunPage คือ Workflow Run หนึ่งหน้า เรียงจากล่าสุดไปเก่าสุด — NextPageToken ว่างแปลว่า
// ไม่มีหน้าถัดไปแล้ว
type RunPage struct {
	Runs          []RunSummary
	NextPageToken string
}

// Engine คือ port ของ "ตัวรัน workflow" ที่ usecase ใช้ — ตอนนี้ implement โดย Temporal
// (internal/external/temporal) usecase จึงไม่ต้อง import Temporal SDK เลย
type Engine interface {
	// Start เริ่ม workflow run ใหม่ด้วย workflow ID ที่ระบุ แล้ว return ทันทีโดยไม่รอผล
	Start(ctx context.Context, workflowID string, wf Workflow) (runID string, err error)
	// ListRuns คืน Workflow Run หนึ่งหน้าตาม query เรียงจากล่าสุดไปเก่าสุด — PageToken ที่
	// อ่านไม่ออกคืน error ประเภท apperr.ErrInvalid
	ListRuns(ctx context.Context, q RunQuery) (RunPage, error)
	// GetRun คืนรายละเอียด run เดียวตาม workflow ID
	GetRun(ctx context.Context, workflowID string) (*RunDetail, error)
	// CheckHealth คืน error ถ้าต่อ engine ไม่ได้ (ใช้กับ readiness probe)
	CheckHealth(ctx context.Context) error
}
