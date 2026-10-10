package temporal

import (
	"context"

	workflowpb "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"

	wfdomain "bop/internal/domain/workflow"
)

// Engine implement wfdomain.Engine ด้วย Temporal client — ย้ายมาจาก HTTP handler เดิม
// (GET/POST /workflow-runs, GET /readyz) ที่เคยเรียก client.Client ตรงๆ
type Engine struct {
	c client.Client
}

// NewEngine ห่อ Temporal client ที่ Dial ไว้แล้วใน cmd/bop/main.go
func NewEngine(c client.Client) *Engine {
	return &Engine{c: c}
}

// Start เริ่ม workflow.Execute ด้วย workflow ID ที่ usecase กำหนด บน task queue ของตัว
// "workflow" เอง (cmd/bop-worker) — คนละเรื่องกับ Step.TaskQueue ของแต่ละ activity ภายใน
func (e *Engine) Start(ctx context.Context, workflowID string, wf wfdomain.Workflow) (string, error) {
	run, err := e.c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: TaskQueueName,
	}, Execute, wf)
	if err != nil {
		return "", err
	}
	return run.GetRunID(), nil
}

// ListRuns query ตรงไปที่ Temporal Visibility API — เรียงจากล่าสุดไปเก่าสุดตาม default
// ordering ของ Temporal เอง (Temporal เก็บ event history + visibility record ให้ทั้งหมด)
func (e *Engine) ListRuns(ctx context.Context) ([]wfdomain.RunSummary, error) {
	resp, err := e.c.ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
		Namespace: Namespace,
	})
	if err != nil {
		return nil, err
	}

	runs := make([]wfdomain.RunSummary, 0, len(resp.Executions))
	for _, info := range resp.Executions {
		runs = append(runs, toRunSummary(info))
	}
	return runs, nil
}

// GetRun ดึงสถานะโดยรวมผ่าน DescribeWorkflowExecution และดึงผลลัพธ์ทีละ step ผ่าน
// QueryWorkflow เรียก query handler ชื่อ "steps" ที่ประกาศไว้ใน Execute (workflow.go) —
// ใช้ได้ทั้งตอน workflow ยังรันอยู่และจบไปแล้ว
func (e *Engine) GetRun(ctx context.Context, workflowID string) (*wfdomain.RunDetail, error) {
	desc, err := e.c.DescribeWorkflowExecution(ctx, workflowID, "")
	if err != nil {
		return nil, err
	}

	var steps []wfdomain.StepResult
	// QueryWorkflow อาจ error ได้หลายกรณี (เช่น เพิ่งเริ่ม workflow ยังไม่ทัน register
	// query handler, หรือ history ถูกลบไปตาม retention แล้ว) — ไม่ถือเป็น fatal error
	// ปล่อยให้ steps เป็น nil แทนที่จะ error ทั้งหมด
	if val, qerr := e.c.QueryWorkflow(ctx, workflowID, "", "steps"); qerr == nil {
		_ = val.Get(&steps)
	}

	return &wfdomain.RunDetail{
		RunSummary: toRunSummary(desc.WorkflowExecutionInfo),
		Steps:      steps,
	}, nil
}

// CheckHealth ใช้กับ readiness probe — error ถ้าต่อ Temporal server ไม่ได้
func (e *Engine) CheckHealth(ctx context.Context) error {
	_, err := e.c.CheckHealth(ctx, &client.CheckHealthRequest{})
	return err
}

// toRunSummary แปลง WorkflowExecutionInfo ที่ได้จาก Temporal (ทั้งจาก ListWorkflow และ
// DescribeWorkflowExecution) ให้เป็น RunSummary — StartTime เป็น *timestamppb.Timestamp
// เสมอ (ไม่มีทางเป็น nil สำหรับ execution ที่มีอยู่จริง) ส่วน CloseTime เป็น nil ตราบใดที่
// workflow ยังไม่จบ จึงต้องเช็คก่อนแปลง
func toRunSummary(info *workflowpb.WorkflowExecutionInfo) wfdomain.RunSummary {
	summary := wfdomain.RunSummary{
		WorkflowID:   info.GetExecution().GetWorkflowId(),
		RunID:        info.GetExecution().GetRunId(),
		WorkflowName: info.GetType().GetName(),
		Status:       info.GetStatus().String(),
		StartedAt:    info.GetStartTime().AsTime(),
	}
	if info.GetCloseTime() != nil {
		ended := info.GetCloseTime().AsTime()
		summary.EndedAt = &ended
	}
	return summary
}
