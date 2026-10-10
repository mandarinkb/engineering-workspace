package temporal

import (
	"context"
	"errors"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	sdktemporal "go.temporal.io/sdk/temporal"

	"bop/internal/domain/apperr"
	"bop/internal/domain/schedule"
	wfdomain "bop/internal/domain/workflow"
)

// Scheduler implement schedule.Scheduler ด้วย client.ScheduleClient ของ Temporal SDK —
// เป็นแค่ "ตัวห่อบางๆ" ไม่ได้เขียน scheduling logic เองเลย (Temporal ทำให้หมด: cron
// parsing, calendar rule, overlap policy, catchup window ฯลฯ) ย้ายมาจาก
// schedule_handler.go เดิมที่เคยเรียก SDK ตรงจากชั้น HTTP
type Scheduler struct {
	c client.Client
}

// NewScheduler ห่อ Temporal client ที่ Dial ไว้แล้วใน cmd/bop/main.go
func NewScheduler(c client.Client) *Scheduler {
	return &Scheduler{c: c}
}

// workflowAction สร้าง action ที่ schedule สั่งรันทุกรอบ — workflow ID ของแต่ละรอบใช้
// ชื่อ workflow (Temporal เติม timestamp ต่อท้ายให้เองทุกรอบ)
func workflowAction(wf wfdomain.Workflow) *client.ScheduleWorkflowAction {
	return &client.ScheduleWorkflowAction{
		ID:        wf.Name,
		Workflow:  Execute,
		Args:      []interface{}{wf},
		TaskQueue: TaskQueueName,
	}
}

func (s *Scheduler) Create(ctx context.Context, id string, spec schedule.Spec, overlap schedule.Overlap, wf wfdomain.Workflow) error {
	_, err := s.c.ScheduleClient().Create(ctx, client.ScheduleOptions{
		ID:      id,
		Spec:    toScheduleSpec(spec),
		Overlap: overlapPolicyByName[overlap],
		Action:  workflowAction(wf),
	})
	if errors.Is(err, sdktemporal.ErrScheduleAlreadyRunning) {
		return apperr.Conflict("schedule already exists")
	}
	return err
}

// List ใช้ List() ของ Temporal ตรงๆ (ไม่ยิง Describe ทีละตัวเพิ่ม เพราะ ListSchedules ให้
// ข้อมูลพอสำหรับหน้า list อยู่แล้ว เร็วกว่ามากถ้ามี schedule จำนวนมาก)
func (s *Scheduler) List(ctx context.Context) ([]schedule.Summary, error) {
	iter, err := s.c.ScheduleClient().List(ctx, client.ScheduleListOptions{})
	if err != nil {
		return nil, err
	}

	out := make([]schedule.Summary, 0)
	for iter.HasNext() {
		entry, err := iter.Next()
		if err != nil {
			return nil, err
		}
		out = append(out, schedule.Summary{
			ID:              entry.ID,
			WorkflowName:    entry.WorkflowType.Name,
			Paused:          entry.Paused,
			Note:            entry.Note,
			Spec:            fromScheduleSpec(entry.Spec),
			NextActionTimes: entry.NextActionTimes,
		})
	}
	return out, nil
}

// Describe คืนสถานะ/เวลาที่แท้จริงจาก Temporal — ไม่เติม Steps (usecase เติมจาก
// repository ของ BOP เอง)
func (s *Scheduler) Describe(ctx context.Context, id string) (*schedule.Detail, error) {
	desc, err := s.c.ScheduleClient().GetHandle(ctx, id).Describe(ctx)
	if err != nil {
		return nil, translate(err)
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

	overlap := schedule.OverlapSkip
	if desc.Schedule.Policy != nil {
		overlap = overlapPolicyName[desc.Schedule.Policy.Overlap]
	}

	paused, note := false, ""
	if desc.Schedule.State != nil {
		paused = desc.Schedule.State.Paused
		note = desc.Schedule.State.Note
	}

	actions := make([]schedule.Action, 0, len(desc.Info.RecentActions))
	for _, a := range desc.Info.RecentActions {
		action := schedule.Action{ScheduleTime: a.ScheduleTime, ActualTime: a.ActualTime}
		if a.StartWorkflowResult != nil {
			action.WorkflowID = a.StartWorkflowResult.WorkflowID
		}
		actions = append(actions, action)
	}

	return &schedule.Detail{
		Summary: schedule.Summary{
			ID:              id,
			WorkflowName:    workflowName,
			Paused:          paused,
			Note:            note,
			Spec:            fromScheduleSpec(desc.Schedule.Spec),
			NextActionTimes: desc.Info.NextActionTimes,
		},
		Overlap:       overlap,
		RecentActions: actions,
		CreatedAt:     desc.Info.CreatedAt,
	}, nil
}

// Update แทนที่ schedule ทั้งก้อน — DoUpdate callback ของ SDK ส่ง input.Description
// (สถานะปัจจุบัน) มาให้ แต่ไม่อ่านมันเลยโดยตั้งใจ เพราะ full replace ไม่จำเป็นต้องรู้ค่าเดิม
func (s *Scheduler) Update(ctx context.Context, id string, spec schedule.Spec, overlap schedule.Overlap, wf wfdomain.Workflow) error {
	sdkSpec := toScheduleSpec(spec)
	err := s.c.ScheduleClient().GetHandle(ctx, id).Update(ctx, client.ScheduleUpdateOptions{
		DoUpdate: func(client.ScheduleUpdateInput) (*client.ScheduleUpdate, error) {
			return &client.ScheduleUpdate{
				Schedule: &client.Schedule{
					Action: workflowAction(wf),
					Spec:   &sdkSpec,
					Policy: &client.SchedulePolicies{Overlap: overlapPolicyByName[overlap]},
				},
			}, nil
		},
	})
	return translate(err)
}

func (s *Scheduler) Delete(ctx context.Context, id string) error {
	return translate(s.c.ScheduleClient().GetHandle(ctx, id).Delete(ctx))
}

func (s *Scheduler) Pause(ctx context.Context, id, note string) error {
	return translate(s.c.ScheduleClient().GetHandle(ctx, id).Pause(ctx, client.SchedulePauseOptions{Note: note}))
}

func (s *Scheduler) Unpause(ctx context.Context, id, note string) error {
	return translate(s.c.ScheduleClient().GetHandle(ctx, id).Unpause(ctx, client.ScheduleUnpauseOptions{Note: note}))
}

func (s *Scheduler) Trigger(ctx context.Context, id string, overlap schedule.Overlap) error {
	return translate(s.c.ScheduleClient().GetHandle(ctx, id).Trigger(ctx, client.ScheduleTriggerOptions{
		Overlap: overlapPolicyByName[overlap],
	}))
}

func (s *Scheduler) Backfill(ctx context.Context, id string, start, end time.Time, overlap schedule.Overlap) error {
	return translate(s.c.ScheduleClient().GetHandle(ctx, id).Backfill(ctx, client.ScheduleBackfillOptions{
		Backfill: []client.ScheduleBackfill{{Start: start, End: end, Overlap: overlapPolicyByName[overlap]}},
	}))
}

// translate แปลง "ไม่เจอ schedule นี้" ของ Temporal เป็น apperr.ErrNotFound (ข้อความเดิม)
// เพื่อให้ HTTP แยก 404 ออกจาก error อื่นๆ ได้โดยไม่ต้องรู้จัก serviceerror ของ Temporal
func translate(err error) error {
	var notFound *serviceerror.NotFound
	if errors.As(err, &notFound) {
		return apperr.WrapNotFound(err)
	}
	return err
}

// toScheduleSpec แปลง schedule.Spec (validate แล้วโดย usecase) เป็น client.ScheduleSpec
func toScheduleSpec(spec schedule.Spec) client.ScheduleSpec {
	switch {
	case spec.IntervalSeconds != nil:
		return client.ScheduleSpec{
			Intervals: []client.ScheduleIntervalSpec{{Every: time.Duration(*spec.IntervalSeconds) * time.Second}},
		}
	case spec.CronExpression != nil:
		return client.ScheduleSpec{CronExpressions: []string{*spec.CronExpression}}
	default:
		return client.ScheduleSpec{}
	}
}

// fromScheduleSpec คือทิศทางย้อนกลับของ toScheduleSpec — เฉพาะ 2 รูปแบบที่ BOP รองรับสร้าง
// เองเท่านั้น ถ้า schedule ถูกสร้างด้วย Calendar spec ตรงๆ ผ่านช่องทางอื่น (เช่น tctl) จะไม่มี
// ทั้งคู่ ปล่อยเป็น nil ทั้งสอง field
func fromScheduleSpec(spec *client.ScheduleSpec) schedule.Spec {
	if spec == nil {
		return schedule.Spec{}
	}
	if len(spec.Intervals) > 0 {
		seconds := int64(spec.Intervals[0].Every / time.Second)
		return schedule.Spec{IntervalSeconds: &seconds}
	}
	if len(spec.CronExpressions) > 0 {
		expr := spec.CronExpressions[0]
		return schedule.Spec{CronExpression: &expr}
	}
	return schedule.Spec{}
}

// overlapPolicyByName/overlapPolicyName คือ mapping ระหว่าง schedule.Overlap กับ enum ตัว
// จริงของ Temporal — ค่าว่าง (ไม่ระบุ) map เป็น UNSPECIFIED ซึ่ง Temporal ตีความว่า "ใช้
// policy default ของ schedule เอง" ตอน trigger/backfill (usecase แปลงค่าว่างเป็น skip ให้
// แล้วตอนสร้าง/แก้ schedule)
var overlapPolicyByName = map[schedule.Overlap]enumspb.ScheduleOverlapPolicy{
	schedule.OverlapUnspecified:    enumspb.SCHEDULE_OVERLAP_POLICY_UNSPECIFIED,
	schedule.OverlapSkip:           enumspb.SCHEDULE_OVERLAP_POLICY_SKIP,
	schedule.OverlapBufferOne:      enumspb.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE,
	schedule.OverlapBufferAll:      enumspb.SCHEDULE_OVERLAP_POLICY_BUFFER_ALL,
	schedule.OverlapCancelOther:    enumspb.SCHEDULE_OVERLAP_POLICY_CANCEL_OTHER,
	schedule.OverlapTerminateOther: enumspb.SCHEDULE_OVERLAP_POLICY_TERMINATE_OTHER,
	schedule.OverlapAllowAll:       enumspb.SCHEDULE_OVERLAP_POLICY_ALLOW_ALL,
}

var overlapPolicyName = map[enumspb.ScheduleOverlapPolicy]schedule.Overlap{
	enumspb.SCHEDULE_OVERLAP_POLICY_UNSPECIFIED:     schedule.OverlapSkip, // unspecified มาจาก schedule เก่าที่สร้างก่อนมี field นี้ — Temporal ปฏิบัติเหมือน skip
	enumspb.SCHEDULE_OVERLAP_POLICY_SKIP:            schedule.OverlapSkip,
	enumspb.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE:      schedule.OverlapBufferOne,
	enumspb.SCHEDULE_OVERLAP_POLICY_BUFFER_ALL:      schedule.OverlapBufferAll,
	enumspb.SCHEDULE_OVERLAP_POLICY_CANCEL_OTHER:    schedule.OverlapCancelOther,
	enumspb.SCHEDULE_OVERLAP_POLICY_TERMINATE_OTHER: schedule.OverlapTerminateOther,
	enumspb.SCHEDULE_OVERLAP_POLICY_ALLOW_ALL:       schedule.OverlapAllowAll,
}
