package schedule

import (
	"context"
	"errors"
	"time"

	"bop/internal/domain/workflow"
)

// Spec คือ "เมื่อไหร่ควรรัน" ของ schedule — รองรับ 2 แบบ mutually exclusive:
// IntervalSeconds (ง่ายที่สุด "ทุก N วินาที") หรือ CronExpression (มาตรฐาน cron string
// เช่น "0 9 * * MON-FRI") เลือกใช้ cron string แทนที่จะออกแบบ Calendar spec เอง (Day/
// Month/Hour/Minute แยก field) เพราะ Temporal แปลง cron expression เป็น calendar spec ให้
// เองภายในอยู่แล้ว
//
// ตอนอ่านกลับจาก engine ทั้งสอง field เป็น nil ได้ ถ้า schedule ถูกสร้างด้วย Calendar spec
// ตรงๆ ผ่านช่องทางอื่น (เช่น tctl) ที่ BOP ไม่รองรับสร้างเอง
type Spec struct {
	IntervalSeconds *int64
	CronExpression  *string
}

// Validate คืน error ทันทีถ้าไม่ได้ระบุมาเลยสักแบบ หรือระบุมาพร้อมกันทั้งสองแบบ (ความ
// กำกวมที่ต้องปฏิเสธตั้งแต่ต้น ไม่ใช่เดาเอาว่าจะใช้อันไหน)
func (s Spec) Validate() error {
	switch {
	case s.IntervalSeconds != nil && s.CronExpression != nil:
		return errors.New("ระบุได้แค่ interval_seconds หรือ cron_expression อย่างใดอย่างหนึ่งเท่านั้น")
	case s.IntervalSeconds != nil:
		if *s.IntervalSeconds <= 0 {
			return errors.New("interval_seconds ต้องมากกว่า 0")
		}
		return nil
	case s.CronExpression != nil:
		if *s.CronExpression == "" {
			return errors.New("cron_expression ห้ามว่างเปล่า")
		}
		return nil
	default:
		return errors.New("ต้องระบุ interval_seconds หรือ cron_expression อย่างใดอย่างหนึ่ง")
	}
}

// Overlap คือ overlap policy แบบอ่านง่าย (ใช้ใน JSON request/response) — ตอบคำถาม "ถ้า
// รอบเก่ายังไม่จบ แล้วถึงเวลารอบใหม่พอดี จะทำยังไง":
//
//	skip            = ข้ามรอบใหม่ไปเลย (default)
//	buffer_one      = รอคิวไว้ 1 รอบ พอรอบเก่าจบค่อยรันรอบที่รอไว้
//	buffer_all      = รอคิวได้ไม่จำกัดจำนวนรอบ
//	cancel_other    = ยกเลิก (cancel) รอบเก่าอย่างนุ่มนวลแล้วเริ่มรอบใหม่ทันที
//	terminate_other = terminate รอบเก่าทันที (แรงกว่า cancel ไม่รอ cleanup logic ใดๆ)
//	allow_all       = รันซ้อนกันได้ไม่จำกัด
//
// ค่าว่าง ("") หมายถึง "ไม่ระบุ" — ตอนสร้าง/แก้ schedule usecase แปลงเป็น skip ส่วนตอน
// trigger/backfill ส่งต่อเป็นค่าว่างให้ engine ใช้ policy default ของ schedule นั้นเอง
type Overlap string

const (
	OverlapUnspecified    Overlap = ""
	OverlapSkip           Overlap = "skip"
	OverlapBufferOne      Overlap = "buffer_one"
	OverlapBufferAll      Overlap = "buffer_all"
	OverlapCancelOther    Overlap = "cancel_other"
	OverlapTerminateOther Overlap = "terminate_other"
	OverlapAllowAll       Overlap = "allow_all"
)

// ParseOverlap ตรวจว่าชื่อ policy ที่ client ส่งมาถูกต้อง — ค่าว่างถือว่าถูกต้อง (ไม่ระบุ)
func ParseOverlap(name string) (Overlap, error) {
	switch o := Overlap(name); o {
	case OverlapUnspecified, OverlapSkip, OverlapBufferOne, OverlapBufferAll,
		OverlapCancelOther, OverlapTerminateOther, OverlapAllowAll:
		return o, nil
	default:
		return "", errors.New("overlap policy ไม่รู้จัก: ต้องเป็นหนึ่งใน skip, buffer_one, buffer_all, cancel_other, terminate_other, allow_all")
	}
}

// Summary คือข้อมูล schedule หนึ่งตัวเท่าที่ engine ให้มาได้จากการ list (ไม่ต้อง describe
// ทีละตัว)
type Summary struct {
	ID              string
	WorkflowName    string
	Paused          bool
	Note            string
	Spec            Spec
	NextActionTimes []time.Time
}

// Action คือการรันหนึ่งครั้งในประวัติล่าสุดของ schedule
type Action struct {
	ScheduleTime time.Time
	ActualTime   time.Time
	WorkflowID   string
}

// Detail ต่อยอดจาก Summary เพิ่มสิ่งที่ต้อง describe ถึงจะได้ บวกกับ Steps ที่มาจาก
// Repository ของ BOP เอง (ไม่ใช่จาก engine — engine ไม่เติม field นี้ให้)
type Detail struct {
	Summary
	Overlap       Overlap
	Steps         []workflow.Step
	RecentActions []Action
	CreatedAt     time.Time
}

// Scheduler คือ port ของ "ตัวจัดการตารางเวลา" ที่ usecase ใช้ — ตอนนี้ implement โดย
// Temporal Schedules (internal/external/temporal) ซึ่งเป็น source of truth ตัวจริงของ
// เวลา/สถานะทั้งหมด (next run time, paused หรือไม่, ประวัติการรัน)
//
// error ที่คืนจาก implementation: schedule ไม่มีอยู่ต้องตรงกับ apperr.ErrNotFound และ
// สร้างซ้ำต้องตรงกับ apperr.ErrConflict
type Scheduler interface {
	Create(ctx context.Context, id string, spec Spec, overlap Overlap, wf workflow.Workflow) error
	List(ctx context.Context) ([]Summary, error)
	Describe(ctx context.Context, id string) (*Detail, error)
	Update(ctx context.Context, id string, spec Spec, overlap Overlap, wf workflow.Workflow) error
	Delete(ctx context.Context, id string) error
	Pause(ctx context.Context, id, note string) error
	Unpause(ctx context.Context, id, note string) error
	Trigger(ctx context.Context, id string, overlap Overlap) error
	Backfill(ctx context.Context, id string, start, end time.Time, overlap Overlap) error
}
