package temporal

import (
	"context"
	"errors"
	"fmt"
	"time"

	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
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

// Create สร้าง Temporal Schedule ใหม่ คืน apperr.ErrConflict ถ้า id นี้มีอยู่แล้ว
func (s *Scheduler) Create(ctx context.Context, id string, spec schedule.Spec, policy schedule.Policy, wf wfdomain.Workflow) error {
	_, err := s.c.ScheduleClient().Create(ctx, client.ScheduleOptions{
		ID:            id,
		Spec:          toScheduleSpec(spec),
		Overlap:       overlapPolicyByName[policy.Overlap],
		CatchupWindow: policy.CatchupWindow,
		Action:        workflowAction(wf),
	})
	if errors.Is(err, sdktemporal.ErrScheduleAlreadyRunning) {
		return apperr.Conflict("schedule already exists")
	}
	return err
}

// List ใช้ List() ของ Temporal ตรงๆ (ไม่ยิง Describe ทีละตัวเพิ่ม เร็วกว่ามากถ้ามี schedule
// จำนวนมาก) ScheduleListEntry ไม่มี action args มีแค่ WorkflowType จึงได้ชื่อ workflow type
// ("Execute") ใน workflow_name — ชื่อที่ผู้ใช้ตั้งมีเฉพาะใน Describe (GET /schedules/{id})
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

// Describe คืนข้อมูลทั้งหมดของ Schedule จาก Temporal ที่เดียว (ADR 0001) รวม steps ที่
// ถอดรหัสจาก action args — จึงใช้ได้กับ Schedule ที่สร้างนอก API ด้วย และไม่หายหลัง restart
func (s *Scheduler) Describe(ctx context.Context, id string) (*schedule.Detail, error) {
	desc, err := s.c.ScheduleClient().GetHandle(ctx, id).Describe(ctx)
	if err != nil {
		return nil, translate(err)
	}

	steps, err := scheduledSteps(desc.Schedule.Action)
	if err != nil {
		return nil, fmt.Errorf("schedule %s: %w", id, err)
	}

	workflowName := scheduledWorkflowName(desc.Schedule.Action)

	overlap := schedule.OverlapSkip
	var catchupWindow time.Duration
	if desc.Schedule.Policy != nil {
		overlap = overlapPolicyName[desc.Schedule.Policy.Overlap]
		catchupWindow = desc.Schedule.Policy.CatchupWindow
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
		CatchupWindow: catchupWindow,
		Steps:         steps,
		RecentActions: actions,
		CreatedAt:     desc.Info.CreatedAt,
	}, nil
}

// Update แทนที่ workflow (steps), ตารางเวลา, overlap policy และ Catch-up Window ของ
// schedule — ส่วนอื่นคงค่าเดิมที่ Temporal ส่งมาใน input.Description ดู replaceSchedule
func (s *Scheduler) Update(ctx context.Context, id string, spec schedule.Spec, policy schedule.Policy, wf wfdomain.Workflow) error {
	sdkSpec := toScheduleSpec(spec)
	err := s.c.ScheduleClient().GetHandle(ctx, id).Update(ctx, client.ScheduleUpdateOptions{
		DoUpdate: func(in client.ScheduleUpdateInput) (*client.ScheduleUpdate, error) {
			updated := replaceSchedule(in.Description.Schedule, sdkSpec, policy, workflowAction(wf))
			return &client.ScheduleUpdate{Schedule: &updated}, nil
		},
	})
	return translate(err)
}

// Delete ลบ Temporal Schedule ทิ้งถาวร
func (s *Scheduler) Delete(ctx context.Context, id string) error {
	return translate(s.c.ScheduleClient().GetHandle(ctx, id).Delete(ctx))
}

// Pause หยุด schedule ชั่วคราว พร้อม note บอกเหตุผล
func (s *Scheduler) Pause(ctx context.Context, id, note string) error {
	return translate(s.c.ScheduleClient().GetHandle(ctx, id).Pause(ctx, client.SchedulePauseOptions{Note: note}))
}

// Unpause เริ่ม schedule ที่ pause ไว้ใหม่
func (s *Scheduler) Unpause(ctx context.Context, id, note string) error {
	return translate(s.c.ScheduleClient().GetHandle(ctx, id).Unpause(ctx, client.ScheduleUnpauseOptions{Note: note}))
}

// Trigger สั่งรัน schedule ทันทีหนึ่งครั้ง นอกตารางเวลาปกติ
func (s *Scheduler) Trigger(ctx context.Context, id string, overlap schedule.Overlap) error {
	return translate(s.c.ScheduleClient().GetHandle(ctx, id).Trigger(ctx, client.ScheduleTriggerOptions{
		Overlap: overlapPolicyByName[overlap],
	}))
}

// Backfill สั่งรันย้อนหลังตามช่วงเวลา start ถึง end
func (s *Scheduler) Backfill(ctx context.Context, id string, start, end time.Time, overlap schedule.Overlap) error {
	return translate(s.c.ScheduleClient().GetHandle(ctx, id).Backfill(ctx, client.ScheduleBackfillOptions{
		Backfill: []client.ScheduleBackfill{{Start: start, End: end, Overlap: overlapPolicyByName[overlap]}},
	}))
}

// executeWorkflowType คือชื่อ workflow type ที่ Temporal ตั้งจากชื่อฟังก์ชัน Execute — action
// ของ Schedule ที่ใช้ type อื่นไม่ใช่ Workflow ของ BOP จึงไม่มี steps ให้ถอดรหัส
const executeWorkflowType = "Execute"

// scheduledSteps ถอดรหัส steps จาก action args ของ Schedule ที่ได้จาก Describe — SDK คืน
// Args เป็น *commonpb.Payload (ตาม comment ของ ScheduleWorkflowAction.Args) จึงต้องแปลง
// กลับด้วย DataConverter ตัวเดียวกับตอนสร้าง (default ทั้ง cmd/bop และ cmd/bop-worker)
//
// Schedule ที่ไม่ได้สั่งรัน Execute (สร้างโดยระบบอื่นใน namespace เดียวกัน) ได้รายการว่าง
// แต่ถ้าเป็น Execute แล้วถอดรหัสไม่ได้คือ payload ไม่ตรงกับ workflow.Workflow ซึ่งต้องรู้
// ทันที ไม่ใช่แสดง steps ว่างให้เข้าใจผิดว่า Schedule ไม่ทำอะไร
func scheduledSteps(action client.ScheduleAction) ([]wfdomain.Step, error) {
	wfAction, ok := action.(*client.ScheduleWorkflowAction)
	if !ok || wfAction.Workflow != executeWorkflowType || len(wfAction.Args) == 0 {
		return []wfdomain.Step{}, nil
	}
	payload, ok := wfAction.Args[0].(*commonpb.Payload)
	if !ok {
		return nil, fmt.Errorf("unexpected action arg type %T", wfAction.Args[0])
	}
	var wf wfdomain.Workflow
	if err := converter.GetDefaultDataConverter().FromPayload(payload, &wf); err != nil {
		return nil, fmt.Errorf("decode workflow steps: %w", err)
	}
	if wf.Steps == nil {
		return []wfdomain.Step{}, nil
	}
	return wf.Steps, nil
}

// scheduledWorkflowName คืนชื่อ Workflow ที่ผู้ใช้ตั้งตอนสร้าง (Workflow.Name ใน action args)
// ไม่ใช่ชื่อ workflow type ซึ่งเป็น "Execute" เสมอ — ถ้าคืน type name ฟอร์มแก้ไขจะได้ "Execute"
// แล้ว PUT กลับไปเปลี่ยน Action ID ของ Schedule ทุกครั้งที่บันทึก
//
// Schedule ที่ไม่ใช่รูปแบบของ BOP (type อื่น, ไม่มี args, ถอดรหัสไม่ได้ หรือชื่อว่าง) ย้อนกลับ
// ไปใช้ชื่อ workflow type เหมือนเดิม ส่วน Execute ที่ถอดรหัสไม่ได้ scheduledSteps รายงาน
// error อยู่แล้ว จึงไม่ต้องซ้ำที่นี่
func scheduledWorkflowName(action client.ScheduleAction) string {
	wfAction, ok := action.(*client.ScheduleWorkflowAction)
	if !ok {
		return ""
	}
	// หลัง Describe Workflow ถูก resolve เป็นชื่อ type (string) แล้ว ไม่ใช่ function reference
	// แบบตอนสร้าง (ดู comment ของ ScheduleWorkflowAction.Workflow ใน SDK)
	typeName, _ := wfAction.Workflow.(string)
	if typeName != executeWorkflowType || len(wfAction.Args) == 0 {
		return typeName
	}
	payload, ok := wfAction.Args[0].(*commonpb.Payload)
	if !ok {
		return typeName
	}
	var wf wfdomain.Workflow
	if err := converter.GetDefaultDataConverter().FromPayload(payload, &wf); err != nil || wf.Name == "" {
		return typeName
	}
	return wf.Name
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

// replaceSchedule สร้าง schedule ชุดใหม่สำหรับ Update จาก schedule ปัจจุบัน โดยแทนที่แค่
// action, spec, overlap policy และ Catch-up Window (usecase ส่งค่า default มาเสมอถ้า client
// ไม่ระบุ Schedule เดิมที่ไม่เคยตั้งจึงได้ค่า default ตอนถูกแก้ไขครั้งถัดไป)
//
// ต้องเริ่มจากค่าปัจจุบัน ไม่ใช่ client.Schedule เปล่า เพราะ:
//   - SDK อ่าน schedule.State.Note ตรงๆ ตอนแปลงเป็น protobuf (sdk-go v1.47.0
//     internal_schedule_client.go) ถ้า State เป็น nil จะ panic — เดิม PUT /schedules ทำ
//     bop-api panic ทุกครั้งด้วยเหตุนี้ (เจอจาก Bruno collection)
//   - การแก้ steps หรือตารางเวลาไม่ควรล้างค่าที่ผู้ใช้ไม่ได้แตะ เช่นสถานะ pause, note และ
//     policy อื่น เช่น PauseOnFailure
func replaceSchedule(
	current client.Schedule,
	spec client.ScheduleSpec,
	policy schedule.Policy,
	action client.ScheduleAction,
) client.Schedule {
	updated := current
	updated.Action = action
	updated.Spec = &spec

	var policies client.SchedulePolicies
	if current.Policy != nil {
		policies = *current.Policy
	}
	policies.Overlap = overlapPolicyByName[policy.Overlap]
	policies.CatchupWindow = policy.CatchupWindow
	updated.Policy = &policies

	var state client.ScheduleState
	if current.State != nil {
		state = *current.State
	}
	updated.State = &state

	return updated
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
