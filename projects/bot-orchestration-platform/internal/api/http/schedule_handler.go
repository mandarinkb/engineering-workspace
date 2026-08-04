// ไฟล์นี้รวม HTTP handler ทั้งหมดที่เกี่ยวกับ "การจัดการ schedule" (สร้าง/แก้ไข/ลบ/
// pause/trigger ทันที/backfill) — แยกออกจาก handler.go เพราะเนื้อหาเยอะพอสมควรและเป็น
// concern ที่ชัดเจนแยกจาก job/bot/auth เดิม
//
// แนวคิดหลักที่ต้องเข้าใจก่อนอ่านโค้ดด้านล่าง: endpoint กลุ่มนี้เป็นแค่ "ตัวห่อบางๆ"
// (thin wrapper) รอบ client.ScheduleClient ของ Temporal SDK เกือบทั้งหมด — ไม่ได้เขียน
// scheduling logic เองเลยสักบรรทัด (Temporal ทำให้หมดแล้ว: cron parsing, calendar rule,
// overlap policy, catchup window ฯลฯ) สิ่งเดียวที่ BOP เก็บเพิ่มเองคือ "workflow
// definition" (ชื่อ + steps) ผ่าน schedule.Repository เพราะ Temporal เก็บส่วนนี้เป็น
// payload ที่ decode กลับยาก (ดูเหตุผลเต็มๆ ที่ internal/schedule/schedule.go)
package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"

	"bop/internal/schedule"
	"bop/internal/workflow"
)

// scheduleSpecRequest คือ JSON shape ของ "เมื่อไหร่ควรรัน" ที่ client ส่งมา — รองรับ 2
// แบบ mutually exclusive: interval_seconds (ง่ายที่สุด "ทุก N วินาที") หรือ
// cron_expression (มาตรฐาน cron string เช่น "0 9 * * MON-FRI") — เลือกใช้ cron string
// แทนที่จะออกแบบ JSON schema ของ Calendar spec เอง (Day/Month/Hour/Minute แยก field)
// เพราะ Temporal SDK แปลง cron expression เป็น calendar spec ให้เองภายในอยู่แล้ว ได้
// ความสามารถ "เฉพาะวันทำการ", "ข้ามวันหยุด" แบบเดียวกับที่ roadmap Phase 2.5 พูดถึงไว้
// โดยไม่ต้องออกแบบ/ทดสอบ schema เองเลย
type scheduleSpecRequest struct {
	IntervalSeconds *int64  `json:"interval_seconds,omitempty"`
	CronExpression  *string `json:"cron_expression,omitempty"`
}

// buildScheduleSpec แปลง scheduleSpecRequest เป็น client.ScheduleSpec ที่ SDK ต้องการ —
// คืน error ทันทีถ้าไม่ได้ระบุมาเลยสักแบบ หรือระบุมาพร้อมกันทั้งสองแบบ (ความกำกวมที่ต้อง
// ปฏิเสธตั้งแต่ต้น ไม่ใช่เดาเอาว่าจะใช้อันไหน)
func buildScheduleSpec(req scheduleSpecRequest) (client.ScheduleSpec, error) {
	switch {
	case req.IntervalSeconds != nil && req.CronExpression != nil:
		return client.ScheduleSpec{}, errors.New("ระบุได้แค่ interval_seconds หรือ cron_expression อย่างใดอย่างหนึ่งเท่านั้น")
	case req.IntervalSeconds != nil:
		if *req.IntervalSeconds <= 0 {
			return client.ScheduleSpec{}, errors.New("interval_seconds ต้องมากกว่า 0")
		}
		return client.ScheduleSpec{
			Intervals: []client.ScheduleIntervalSpec{{Every: time.Duration(*req.IntervalSeconds) * time.Second}},
		}, nil
	case req.CronExpression != nil:
		if *req.CronExpression == "" {
			return client.ScheduleSpec{}, errors.New("cron_expression ห้ามว่างเปล่า")
		}
		return client.ScheduleSpec{CronExpressions: []string{*req.CronExpression}}, nil
	default:
		return client.ScheduleSpec{}, errors.New("ต้องระบุ interval_seconds หรือ cron_expression อย่างใดอย่างหนึ่ง")
	}
}

// specToResponse คือทิศทางย้อนกลับของ buildScheduleSpec — แปลง client.ScheduleSpec ที่
// อ่านมาจาก Temporal กลับเป็น field ที่ตอบผ่าน JSON ได้ตรงกับที่ client เคยส่งมาตอนสร้าง
// (เฉพาะ 2 รูปแบบที่ BOP รองรับสร้างเองเท่านั้น — ถ้า schedule ถูกสร้างด้วย Calendar spec
// ตรงๆ ผ่านช่องทางอื่น (เช่น tctl) จะไม่มีทั้งคู่ ปล่อยเป็น nil ทั้งสอง field)
func specToResponse(spec *client.ScheduleSpec) (intervalSeconds *int64, cronExpression *string) {
	if spec == nil {
		return nil, nil
	}
	if len(spec.Intervals) > 0 {
		seconds := int64(spec.Intervals[0].Every / time.Second)
		return &seconds, nil
	}
	if len(spec.CronExpressions) > 0 {
		expr := spec.CronExpressions[0]
		return nil, &expr
	}
	return nil, nil
}

// overlapPolicyByName/overlapPolicyName คือ mapping ระหว่างชื่อ overlap policy แบบอ่าน
// ง่าย (ใช้ใน JSON request/response) กับ enum ตัวจริงของ Temporal — ความหมายสั้นๆ ของ
// แต่ละ policy (ตอบคำถาม "ถ้ารอบเก่ายังไม่จบ แล้วถึงเวลารอบใหม่พอดี จะทำยังไง"):
//
//	skip            = ข้ามรอบใหม่ไปเลย (default — ตรงกับพฤติกรรมเดิมของ internal/scheduler
//	                   เวอร์ชันก่อน Temporal ที่ "รันซ้อนกันได้แบบไม่ตั้งใจ" ต่างกันตรงที่
//	                   Temporal ให้เลือก "ไม่ซ้อน" ได้ ไม่ใช่ปล่อยซ้อนกันโดยไม่ได้ตั้งใจ)
//	buffer_one      = รอคิวไว้ 1 รอบ พอรอบเก่าจบค่อยรันรอบที่รอไว้
//	buffer_all      = รอคิวได้ไม่จำกัดจำนวนรอบ
//	cancel_other    = ยกเลิก (cancel) รอบเก่าอย่างนุ่มนวลแล้วเริ่มรอบใหม่ทันที
//	terminate_other = terminate รอบเก่าทันที (แรงกว่า cancel ไม่รอ cleanup logic ใดๆ)
//	allow_all       = รันซ้อนกันได้ไม่จำกัด (พฤติกรรมเดิมของ internal/scheduler ก่อน Temporal)
var overlapPolicyByName = map[string]enumspb.ScheduleOverlapPolicy{
	"":                enumspb.SCHEDULE_OVERLAP_POLICY_SKIP,
	"skip":            enumspb.SCHEDULE_OVERLAP_POLICY_SKIP,
	"buffer_one":      enumspb.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE,
	"buffer_all":      enumspb.SCHEDULE_OVERLAP_POLICY_BUFFER_ALL,
	"cancel_other":    enumspb.SCHEDULE_OVERLAP_POLICY_CANCEL_OTHER,
	"terminate_other": enumspb.SCHEDULE_OVERLAP_POLICY_TERMINATE_OTHER,
	"allow_all":       enumspb.SCHEDULE_OVERLAP_POLICY_ALLOW_ALL,
}

var overlapPolicyName = map[enumspb.ScheduleOverlapPolicy]string{
	enumspb.SCHEDULE_OVERLAP_POLICY_UNSPECIFIED:     "skip", // unspecified มาจาก schedule เก่าที่สร้างก่อนมี field นี้ — Temporal ปฏิบัติเหมือน skip
	enumspb.SCHEDULE_OVERLAP_POLICY_SKIP:            "skip",
	enumspb.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE:      "buffer_one",
	enumspb.SCHEDULE_OVERLAP_POLICY_BUFFER_ALL:      "buffer_all",
	enumspb.SCHEDULE_OVERLAP_POLICY_CANCEL_OTHER:    "cancel_other",
	enumspb.SCHEDULE_OVERLAP_POLICY_TERMINATE_OTHER: "terminate_other",
	enumspb.SCHEDULE_OVERLAP_POLICY_ALLOW_ALL:       "allow_all",
}

func parseOverlapPolicy(name string) (enumspb.ScheduleOverlapPolicy, error) {
	policy, ok := overlapPolicyByName[name]
	if !ok {
		return 0, errors.New("overlap policy ไม่รู้จัก: ต้องเป็นหนึ่งใน skip, buffer_one, buffer_all, cancel_other, terminate_other, allow_all")
	}
	return policy, nil
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

// scheduleSummary คือรูปแบบ JSON ของ schedule หนึ่งตัวในหน้า list — เท่าที่ Temporal
// ListSchedules คืนมาให้โดยไม่ต้อง Describe() เพิ่ม (เร็วกว่าเพราะไม่ยิง RPC ทีละตัว)
type scheduleSummary struct {
	ID              string      `json:"id"`
	WorkflowName    string      `json:"workflow_name"`
	Paused          bool        `json:"paused"`
	Note            string      `json:"note,omitempty"`
	IntervalSeconds *int64      `json:"interval_seconds,omitempty"`
	CronExpression  *string     `json:"cron_expression,omitempty"`
	NextActionTimes []time.Time `json:"next_action_times,omitempty"`
}

// scheduleDetail ต่อยอดจาก scheduleSummary เพิ่มรายละเอียดที่ต้อง Describe() เท่านั้นถึง
// จะได้ (overlap policy, ประวัติการรันล่าสุด, เวลาสร้าง) บวกกับ Steps ที่มาจาก
// schedule.Repository ของ BOP เอง (ไม่ใช่จาก Temporal โดยตรง — อาจว่างเปล่าได้ถ้า
// schedule นี้ไม่ได้สร้างผ่าน API ชุดนี้ ดูหมายเหตุที่ internal/schedule/schedule.go)
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

func toScheduleSummary(id string, spec *client.ScheduleSpec, paused bool, note, workflowName string, nextActionTimes []time.Time) scheduleSummary {
	interval, cron := specToResponse(spec)
	return scheduleSummary{
		ID:              id,
		WorkflowName:    workflowName,
		Paused:          paused,
		Note:            note,
		IntervalSeconds: interval,
		CronExpression:  cron,
		NextActionTimes: nextActionTimes,
	}
}

// handleCreateSchedule สร้าง Temporal Schedule ใหม่ตาม spec ที่ระบุ แล้วเก็บ workflow
// definition ไว้ใน schedule.Repository คู่กัน — ลำดับสำคัญ: สร้างที่ Temporal ก่อนเสมอ
// (เป็น source of truth ตัวจริง) แล้วค่อยบันทึก definition ของเราทีหลัง ถ้าขั้นหลังพลาด
// (ไม่ควรเกิดในทางปกติ) จะ compensate ด้วยการลบ schedule ที่เพิ่งสร้างใน Temporal ทิ้ง
// ไม่ปล่อยให้เหลือ schedule กำพร้าที่ไม่มี definition คู่กันเลย (Saga-style compensation
// เดียวกับแนวคิดที่พูดถึงใน FULLSTACK-INFRA-ROADMAP.md Phase 3)
func (s *Server) handleCreateSchedule(w http.ResponseWriter, r *http.Request) {
	var req createScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.ID == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	if len(req.Steps) == 0 {
		http.Error(w, "steps must not be empty", http.StatusBadRequest)
		return
	}

	spec, err := buildScheduleSpec(req.Spec)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	overlap, err := parseOverlapPolicy(req.Overlap)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	wf := workflow.Workflow{Name: req.WorkflowName, Steps: req.Steps}
	_, err = s.temporal.ScheduleClient().Create(r.Context(), client.ScheduleOptions{
		ID:      req.ID,
		Spec:    spec,
		Overlap: overlap,
		Action: &client.ScheduleWorkflowAction{
			ID:        req.WorkflowName,
			Workflow:  workflow.Execute,
			Args:      []interface{}{wf},
			TaskQueue: workflow.TaskQueueName,
		},
	})
	if err != nil {
		if errors.Is(err, temporal.ErrScheduleAlreadyRunning) {
			http.Error(w, "schedule already exists", http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	def := &schedule.Definition{ID: req.ID, WorkflowName: req.WorkflowName, Steps: req.Steps, CreatedAt: time.Now()}
	if err := s.scheduleRepo.Create(r.Context(), def); err != nil {
		_ = s.temporal.ScheduleClient().GetHandle(r.Context(), req.ID).Delete(r.Context())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	detail, err := s.describeSchedule(r.Context(), req.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, detail)
}

// handleListSchedules คืนรายการ schedule ทั้งหมด — ใช้ List() ของ Temporal ตรงๆ (ไม่ยิง
// Describe ทีละตัวเพิ่ม เพราะ ListSchedules ให้ข้อมูลพอสำหรับหน้า list อยู่แล้ว เร็วกว่า
// มากถ้ามี schedule จำนวนมาก)
func (s *Server) handleListSchedules(w http.ResponseWriter, r *http.Request) {
	iter, err := s.temporal.ScheduleClient().List(r.Context(), client.ScheduleListOptions{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	out := make([]scheduleSummary, 0)
	for iter.HasNext() {
		entry, err := iter.Next()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out = append(out, toScheduleSummary(entry.ID, entry.Spec, entry.Paused, entry.Note, entry.WorkflowType.Name, entry.NextActionTimes))
	}
	writeJSON(w, http.StatusOK, out)
}

// handleGetSchedule คืนรายละเอียดเต็มของ schedule เดียว (รวม overlap policy, ประวัติ
// การรันล่าสุด, และ steps จาก schedule.Repository)
func (s *Server) handleGetSchedule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	detail, err := s.describeSchedule(r.Context(), id)
	if err != nil {
		s.writeTemporalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

// describeSchedule รวมข้อมูลจาก 2 แหล่งเข้าด้วยกัน: Temporal (สถานะ/เวลาที่แท้จริง ผ่าน
// Describe) และ schedule.Repository ของ BOP เอง (workflow steps) — ใช้ทั้งใน
// handleCreateSchedule (ตอบ response หลังสร้างเสร็จ), handleUpdateSchedule, และ
// handleGetSchedule
func (s *Server) describeSchedule(ctx context.Context, id string) (*scheduleDetail, error) {
	desc, err := s.temporal.ScheduleClient().GetHandle(ctx, id).Describe(ctx)
	if err != nil {
		return nil, err
	}

	// หลัง Describe เสมอ Schedule.Action จะเป็น *client.ScheduleWorkflowAction ที่มี
	// field Workflow ถูก resolve เป็นชื่อ workflow type (string) แล้ว ไม่ใช่ function
	// reference แบบตอนสร้าง (ดู comment ของ ScheduleWorkflowAction.Workflow ใน SDK)
	workflowName := ""
	if action, ok := desc.Schedule.Action.(*client.ScheduleWorkflowAction); ok {
		if name, ok := action.Workflow.(string); ok {
			workflowName = name
		}
	}

	overlap := "skip"
	if desc.Schedule.Policy != nil {
		overlap = overlapPolicyName[desc.Schedule.Policy.Overlap]
	}

	paused, note := false, ""
	if desc.Schedule.State != nil {
		paused = desc.Schedule.State.Paused
		note = desc.Schedule.State.Note
	}

	actions := make([]scheduleAction, 0, len(desc.Info.RecentActions))
	for _, a := range desc.Info.RecentActions {
		action := scheduleAction{ScheduleTime: a.ScheduleTime, ActualTime: a.ActualTime}
		if a.StartWorkflowResult != nil {
			action.WorkflowID = a.StartWorkflowResult.WorkflowID
		}
		actions = append(actions, action)
	}

	detail := &scheduleDetail{
		scheduleSummary: toScheduleSummary(id, desc.Schedule.Spec, paused, note, workflowName, desc.Info.NextActionTimes),
		Overlap:         overlap,
		RecentActions:   actions,
		CreatedAt:       desc.Info.CreatedAt,
	}

	// เติม Steps จาก schedule.Repository ของ BOP เอง — ถ้าไม่เจอ (schedule นี้สร้าง
	// ก่อนมี API ชุดนี้ เช่น check-example-com-schedule ที่ cmd/bop-worker สร้างตรงผ่าน
	// SDK, หรือ cmd/bop เพิ่ง restart ไปเพราะ repository เป็น in-memory) ปล่อย Steps
	// เป็น nil เงียบๆ ไม่ใช่ fatal error ของทั้ง request — schedule ยังใช้งานได้ปกติ
	// แค่ frontend จะไม่รู้ว่า steps ข้างในคืออะไรเท่านั้น
	if def, err := s.scheduleRepo.Get(ctx, id); err == nil {
		detail.Steps = def.Steps
	}

	return detail, nil
}

// handleUpdateSchedule แทนที่ schedule ทั้งก้อนด้วยค่าใหม่ (PUT = full replace ตาม REST
// convention ปกติ ไม่ใช่ partial patch) — DoUpdate callback ของ Temporal SDK ส่ง
// input.Description (สถานะปัจจุบัน) มาให้ แต่ handler นี้ไม่อ่านมันเลยโดยตั้งใจ เพราะ
// full replace ไม่จำเป็นต้องรู้ค่าเดิม
func (s *Server) handleUpdateSchedule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req updateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if len(req.Steps) == 0 {
		http.Error(w, "steps must not be empty", http.StatusBadRequest)
		return
	}
	spec, err := buildScheduleSpec(req.Spec)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	overlap, err := parseOverlapPolicy(req.Overlap)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	wf := workflow.Workflow{Name: req.WorkflowName, Steps: req.Steps}
	err = s.temporal.ScheduleClient().GetHandle(r.Context(), id).Update(r.Context(), client.ScheduleUpdateOptions{
		DoUpdate: func(client.ScheduleUpdateInput) (*client.ScheduleUpdate, error) {
			return &client.ScheduleUpdate{
				Schedule: &client.Schedule{
					Action: &client.ScheduleWorkflowAction{
						ID:        req.WorkflowName,
						Workflow:  workflow.Execute,
						Args:      []interface{}{wf},
						TaskQueue: workflow.TaskQueueName,
					},
					Spec:   &spec,
					Policy: &client.SchedulePolicies{Overlap: overlap},
				},
			}, nil
		},
	})
	if err != nil {
		s.writeTemporalError(w, err)
		return
	}

	def := &schedule.Definition{ID: id, WorkflowName: req.WorkflowName, Steps: req.Steps, CreatedAt: time.Now()}
	// ใช้ Update() ก่อน ถ้าไม่เจอ definition เดิม (schedule นี้สร้างก่อนมี API ชุดนี้)
	// ให้ Create() แทน — ไม่ปล่อยให้ PUT ที่สำเร็จฝั่ง Temporal แล้วมาล้มเหลวเพราะ
	// bookkeeping ของเราเองยังไม่เคยมี record นี้มาก่อน
	if err := s.scheduleRepo.Update(r.Context(), def); err != nil {
		_ = s.scheduleRepo.Create(r.Context(), def)
	}

	detail, err := s.describeSchedule(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

// handleDeleteSchedule ลบ schedule ทิ้งทั้งฝั่ง Temporal และ definition ของเราเอง —
// ลบที่ Temporal ก่อนเสมอ (source of truth) ถ้าสำเร็จค่อยลบ definition local (ไม่ fatal
// ถ้าไม่เจอ — schedule อาจไม่เคยมี definition ใน repository นี้อยู่แล้วตั้งแต่ต้น)
func (s *Server) handleDeleteSchedule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.temporal.ScheduleClient().GetHandle(r.Context(), id).Delete(r.Context()); err != nil {
		s.writeTemporalError(w, err)
		return
	}
	_ = s.scheduleRepo.Delete(r.Context(), id)
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
	id := r.PathValue("id")
	var req noteRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // body ว่างได้ ไม่ error ถ้า decode ไม่ผ่าน

	if err := s.temporal.ScheduleClient().GetHandle(r.Context(), id).Pause(r.Context(), client.SchedulePauseOptions{Note: req.Note}); err != nil {
		s.writeTemporalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleUnpauseSchedule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req noteRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	if err := s.temporal.ScheduleClient().GetHandle(r.Context(), id).Unpause(r.Context(), client.ScheduleUnpauseOptions{Note: req.Note}); err != nil {
		s.writeTemporalError(w, err)
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
// "Order Now"/"Run Now" ของ Control-M ตรงๆ คือ manual trigger ที่ต้องการ ไม่กระทบตาราง
// เวลาปกติของ schedule เลย (รอบถัดไปตามตารางยังมาตามเวลาเดิม)
func (s *Server) handleTriggerSchedule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req triggerRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	overlap := enumspb.SCHEDULE_OVERLAP_POLICY_UNSPECIFIED // unspecified = ใช้ policy default ของ schedule เอง
	if req.Overlap != "" {
		var err error
		overlap, err = parseOverlapPolicy(req.Overlap)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	if err := s.temporal.ScheduleClient().GetHandle(r.Context(), id).Trigger(r.Context(), client.ScheduleTriggerOptions{Overlap: overlap}); err != nil {
		s.writeTemporalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// backfillRequest คือ JSON body ของ POST /schedules/{id}/backfill — เทียบเท่า "Rerun"
// ของ Control-M: สั่งให้ Temporal ประเมิน schedule ย้อนหลังในช่วงเวลาที่ระบุ แล้วรัน
// Action ทุกครั้งที่ควรจะเกิดขึ้นในช่วงนั้น ราวกับเวลานั้นเพิ่งผ่านไปตอนนี้
type backfillRequest struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Overlap   string    `json:"overlap,omitempty"`
}

func (s *Server) handleBackfillSchedule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req backfillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if !req.StartTime.Before(req.EndTime) {
		http.Error(w, "start_time must be before end_time", http.StatusBadRequest)
		return
	}

	overlap := enumspb.SCHEDULE_OVERLAP_POLICY_UNSPECIFIED
	if req.Overlap != "" {
		var err error
		overlap, err = parseOverlapPolicy(req.Overlap)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	err := s.temporal.ScheduleClient().GetHandle(r.Context(), id).Backfill(r.Context(), client.ScheduleBackfillOptions{
		Backfill: []client.ScheduleBackfill{{Start: req.StartTime, End: req.EndTime, Overlap: overlap}},
	})
	if err != nil {
		s.writeTemporalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// writeTemporalError แปลง error จาก Temporal SDK เป็น HTTP status code ที่เหมาะสม —
// แยก "ไม่เจอ schedule นี้" (404) ออกจาก error อื่นๆ ทั้งหมด (500) เพื่อให้ client
// (dashboard) แยกแยะได้ว่าควรแสดง "ไม่พบ" หรือ "เกิดข้อผิดพลาด" ให้ user เห็น
func (s *Server) writeTemporalError(w http.ResponseWriter, err error) {
	var notFound *serviceerror.NotFound
	if errors.As(err, &notFound) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
