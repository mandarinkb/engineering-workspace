// ไฟล์นี้รวม HTTP handler ทั้งหมดที่เกี่ยวกับ "การจัดการ schedule" (สร้าง/แก้ไข/ลบ/
// pause/trigger ทันที/backfill) — แยกออกจาก handler.go เพราะเนื้อหาเยอะพอสมควรและเป็น
// concern ที่ชัดเจนแยกจาก job/bot/auth เดิม
//
// handler ในไฟล์นี้ทำแค่แปลง JSON <-> schedule usecase (internal/usecase/schedule) — การ
// validate, ลำดับการเรียก scheduler กับ repository, และ compensation อยู่ที่ usecase ส่วน
// การเรียก Temporal Schedules จริงอยู่ที่ internal/external/temporal/scheduler.go
package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"bop/internal/domain/apperr"
	"bop/internal/domain/schedule"
	"bop/internal/domain/workflow"
	scheduleuc "bop/internal/usecase/schedule"
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

// createScheduleRequest/updateScheduleRequest คือ JSON body ของ POST/PUT /schedules —
// แยก type กันแม้หน้าตาจะคล้ายกันมาก เพราะ ID ของ create มาจาก body (ต้องระบุเอง) แต่
// ID ของ update มาจาก URL path (ห้ามเปลี่ยน ID ผ่าน update ได้ — ถ้าอยากเปลี่ยนชื่อต้อง
// ลบตัวเก่าสร้างตัวใหม่)
type createScheduleRequest struct {
	ID           string              `json:"id"`
	WorkflowName string              `json:"workflow_name"`
	Steps        []workflow.Step     `json:"steps"`
	Spec         scheduleSpecRequest `json:"spec"`
	Overlap      string              `json:"overlap,omitempty"`
}

type updateScheduleRequest struct {
	WorkflowName string              `json:"workflow_name"`
	Steps        []workflow.Step     `json:"steps"`
	Spec         scheduleSpecRequest `json:"spec"`
	Overlap      string              `json:"overlap,omitempty"`
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
type scheduleDetail struct {
	scheduleSummary
	Overlap       string           `json:"overlap"`
	Steps         []workflow.Step  `json:"steps,omitempty"`
	RecentActions []scheduleAction `json:"recent_actions,omitempty"`
	CreatedAt     time.Time        `json:"created_at,omitempty"`
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
		scheduleSummary: toScheduleSummary(d.Summary),
		Overlap:         string(d.Overlap),
		Steps:           d.Steps,
		RecentActions:   actions,
		CreatedAt:       d.CreatedAt,
	}
}

// handleCreateSchedule สร้าง schedule ใหม่ตาม spec ที่ระบุ — ดูลำดับ "สร้างที่ scheduler
// ก่อนแล้วค่อยเก็บ definition" และ compensation ที่ scheduleuc.Service.Create
func (s *Server) handleCreateSchedule(w http.ResponseWriter, r *http.Request) {
	var req createScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	detail, err := s.schedules.Create(r.Context(), req.ID, scheduleuc.Input{
		WorkflowName: req.WorkflowName,
		Steps:        req.Steps,
		Spec:         req.Spec.toDomain(),
		Overlap:      req.Overlap,
	})
	if err != nil {
		writeScheduleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toScheduleDetail(detail))
}

// handleListSchedules คืนรายการ schedule ทั้งหมด
func (s *Server) handleListSchedules(w http.ResponseWriter, r *http.Request) {
	list, err := s.schedules.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	out := make([]scheduleSummary, 0, len(list))
	for _, item := range list {
		out = append(out, toScheduleSummary(item))
	}
	writeJSON(w, http.StatusOK, out)
}

// handleGetSchedule คืนรายละเอียดเต็มของ schedule เดียว (รวม overlap policy, ประวัติ
// การรันล่าสุด, และ steps จาก schedule.Repository)
func (s *Server) handleGetSchedule(w http.ResponseWriter, r *http.Request) {
	detail, err := s.schedules.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeScheduleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toScheduleDetail(detail))
}

// handleUpdateSchedule แทนที่ schedule ทั้งก้อนด้วยค่าใหม่ (PUT = full replace ตาม REST
// convention ปกติ ไม่ใช่ partial patch)
func (s *Server) handleUpdateSchedule(w http.ResponseWriter, r *http.Request) {
	var req updateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	detail, err := s.schedules.Update(r.Context(), r.PathValue("id"), scheduleuc.Input{
		WorkflowName: req.WorkflowName,
		Steps:        req.Steps,
		Spec:         req.Spec.toDomain(),
		Overlap:      req.Overlap,
	})
	if err != nil {
		writeScheduleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toScheduleDetail(detail))
}

// handleDeleteSchedule ลบ schedule ทิ้งทั้งฝั่ง scheduler และ definition ของเราเอง
func (s *Server) handleDeleteSchedule(w http.ResponseWriter, r *http.Request) {
	if err := s.schedules.Delete(r.Context(), r.PathValue("id")); err != nil {
		writeScheduleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// noteRequest คือ JSON body ที่ pause/unpause รับ (optional ทั้งคู่ — Temporal มี default
// note ให้เองถ้าไม่ระบุมา เช่น "Paused via Go SDK")
type noteRequest struct {
	Note string `json:"note,omitempty"`
}

// handlePauseSchedule/handleUnpauseSchedule เทียบเท่า "Hold"/"Free" ของ Control-M —
// หยุดชั่วคราวโดยไม่ลบ schedule ทิ้ง (ต่างจาก DELETE ที่ลบถาวร)
func (s *Server) handlePauseSchedule(w http.ResponseWriter, r *http.Request) {
	var req noteRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // body ว่างได้ ไม่ error ถ้า decode ไม่ผ่าน

	if err := s.schedules.Pause(r.Context(), r.PathValue("id"), req.Note); err != nil {
		writeScheduleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleUnpauseSchedule(w http.ResponseWriter, r *http.Request) {
	var req noteRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	if err := s.schedules.Unpause(r.Context(), r.PathValue("id"), req.Note); err != nil {
		writeScheduleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// triggerRequest คือ JSON body ของ POST /schedules/{id}/trigger — Overlap ปล่อยว่างได้
// (ใช้ policy default ของ schedule เอง)
type triggerRequest struct {
	Overlap string `json:"overlap,omitempty"`
}

// handleTriggerSchedule สั่งรัน schedule นี้ "ทันที นอกตารางเวลาปกติ" — เทียบเท่า
// "Order Now"/"Run Now" ของ Control-M
func (s *Server) handleTriggerSchedule(w http.ResponseWriter, r *http.Request) {
	var req triggerRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	if err := s.schedules.Trigger(r.Context(), r.PathValue("id"), req.Overlap); err != nil {
		writeScheduleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// backfillRequest คือ JSON body ของ POST /schedules/{id}/backfill — เทียบเท่า "Rerun"
// ของ Control-M
type backfillRequest struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Overlap   string    `json:"overlap,omitempty"`
}

func (s *Server) handleBackfillSchedule(w http.ResponseWriter, r *http.Request) {
	var req backfillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if err := s.schedules.Backfill(r.Context(), r.PathValue("id"), req.StartTime, req.EndTime, req.Overlap); err != nil {
		writeScheduleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
