// ไฟล์นี้รวม type และ helper ที่ endpoint กลุ่ม /schedules ใช้ร่วมกัน (handler แต่ละตัวอยู่
// คนละไฟล์: create_schedule.go, list_schedules.go, ...)
//
// handler ของ schedule ทำแค่แปลง JSON <-> schedule usecase (internal/usecase/schedule) — การ
// validate, ลำดับการเรียก scheduler กับ repository, และ compensation อยู่ที่ usecase ส่วน
// การเรียก Temporal Schedules จริงอยู่ที่ internal/external/temporal/scheduler.go

package http

import (
	"errors"
	"net/http"
	"time"

	"bop/internal/domain/apperr"
	"bop/internal/domain/schedule"
	"bop/internal/domain/workflow"
)

// scheduleSpecRequest คือ JSON shape ของ "เมื่อไหร่ควรรัน" ที่ client ส่งมา — รองรับ 2
// แบบ mutually exclusive: interval_seconds หรือ cron_expression (ดูกฎที่ schedule.Spec)
type scheduleSpecRequest struct {
	IntervalSeconds *int64  `json:"interval_seconds,omitempty"`
	CronExpression  *string `json:"cron_expression,omitempty"`
}

func (r scheduleSpecRequest) toDomain() schedule.Spec {
	return schedule.Spec{IntervalSeconds: r.IntervalSeconds, CronExpression: r.CronExpression}
}

// scheduleSummary คือรูปแบบ JSON ของ schedule หนึ่งตัวในหน้า list
type scheduleSummary struct {
	ID              string      `json:"id"`
	WorkflowName    string      `json:"workflow_name"`
	Paused          bool        `json:"paused"`
	Note            string      `json:"note,omitempty"`
	IntervalSeconds *int64      `json:"interval_seconds,omitempty"`
	CronExpression  *string     `json:"cron_expression,omitempty"`
	NextActionTimes []time.Time `json:"next_action_times,omitempty"`
}

// scheduleDetail ต่อยอดจาก scheduleSummary เพิ่มรายละเอียดที่ต้อง describe เท่านั้นถึง
// จะได้ (overlap policy, ประวัติการรันล่าสุด, เวลาสร้าง) บวกกับ Steps ที่มาจาก
// schedule.Repository ของ BOP เอง (อาจว่างเปล่าได้ ดูหมายเหตุที่ scheduleuc.Service.Get)
// CatchupWindowSeconds คือ Catch-up Window ที่ตั้งอยู่จริงใน Temporal Schedule
type scheduleDetail struct {
	scheduleSummary
	Overlap              string           `json:"overlap"`
	CatchupWindowSeconds int64            `json:"catchup_window_seconds"`
	Steps                []workflow.Step  `json:"steps,omitempty"`
	RecentActions        []scheduleAction `json:"recent_actions,omitempty"`
	CreatedAt            time.Time        `json:"created_at,omitempty"`
}

type scheduleAction struct {
	ScheduleTime time.Time `json:"schedule_time"`
	ActualTime   time.Time `json:"actual_time"`
	WorkflowID   string    `json:"workflow_id,omitempty"`
}

func toScheduleSummary(s schedule.Summary) scheduleSummary {
	return scheduleSummary{
		ID:              s.ID,
		WorkflowName:    s.WorkflowName,
		Paused:          s.Paused,
		Note:            s.Note,
		IntervalSeconds: s.Spec.IntervalSeconds,
		CronExpression:  s.Spec.CronExpression,
		NextActionTimes: s.NextActionTimes,
	}
}

func toScheduleDetail(d *schedule.Detail) *scheduleDetail {
	actions := make([]scheduleAction, 0, len(d.RecentActions))
	for _, a := range d.RecentActions {
		actions = append(actions, scheduleAction{ScheduleTime: a.ScheduleTime, ActualTime: a.ActualTime, WorkflowID: a.WorkflowID})
	}
	return &scheduleDetail{
		scheduleSummary:      toScheduleSummary(d.Summary),
		Overlap:              string(d.Overlap),
		CatchupWindowSeconds: int64(d.CatchupWindow / time.Second),
		Steps:                d.Steps,
		RecentActions:        actions,
		CreatedAt:            d.CreatedAt,
	}
}

// writeScheduleError แปลงประเภท error จาก usecase เป็น HTTP status code — input ผิด (400),
// ไม่เจอ schedule (404), สร้างซ้ำ (409), อื่นๆ ทั้งหมด (500) เพื่อให้ client (dashboard)
// แยกแยะได้ว่าควรแสดง "ข้อมูลไม่ถูกต้อง", "ไม่พบ" หรือ "เกิดข้อผิดพลาด" ให้ user เห็น
func writeScheduleError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, apperr.ErrInvalid):
		status = http.StatusBadRequest
	case errors.Is(err, apperr.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, apperr.ErrConflict):
		status = http.StatusConflict
	}
	http.Error(w, err.Error(), status)
}
