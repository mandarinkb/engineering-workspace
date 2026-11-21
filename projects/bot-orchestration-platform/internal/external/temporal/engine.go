package temporal

import (
	"context"
	"encoding/base64"
	"errors"

	"go.temporal.io/api/serviceerror"
	workflowpb "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"

	"bop/internal/domain/apperr"
	wfdomain "bop/internal/domain/workflow"
)

// statusQueries คือ visibility query ของแต่ละกลุ่มสถานะ — failed รวม Terminated และ
// TimedOut ตามนิยามของ RunStatusFailed ใช้ OR แทน IN เพื่อไม่ผูกกับ visibility store
// ที่รองรับ syntax ครบเท่านั้น
var statusQueries = map[wfdomain.RunStatusFilter]string{
	wfdomain.RunStatusRunning:   "ExecutionStatus = 'Running'",
	wfdomain.RunStatusCompleted: "ExecutionStatus = 'Completed'",
	wfdomain.RunStatusFailed: "(ExecutionStatus = 'Failed' OR ExecutionStatus = 'Terminated'" +
		" OR ExecutionStatus = 'TimedOut')",
	wfdomain.RunStatusCanceled: "ExecutionStatus = 'Canceled'",
}

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

// ListRuns query ตรงไปที่ Temporal Visibility API ทีละหน้า — เรียงจากล่าสุดไปเก่าสุดตาม
// default ordering ของ Temporal เอง NextPageToken ของ Temporal เป็น []byte จึงห่อเป็น
// base64 (URL-safe) ให้ส่งผ่าน query string ได้ ฝั่ง client ถือเป็น opaque string
func (e *Engine) ListRuns(ctx context.Context, q wfdomain.RunQuery) (wfdomain.RunPage, error) {
	token, err := base64.RawURLEncoding.DecodeString(q.PageToken)
	if err != nil {
		return wfdomain.RunPage{}, apperr.Invalid("page_token is malformed")
	}

	resp, err := e.c.ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
		Namespace:     Namespace,
		PageSize:      int32(q.PageSize), // usecase ปัดเพดานไว้ที่ 200 แล้ว ไม่ล้น int32
		NextPageToken: token,
		Query:         statusQueries[q.Status],
	})
	if err != nil {
		// token ที่ decode ได้แต่ Temporal อ่านไม่ออก (เช่นแก้มือ) ตอบ InvalidArgument
		var invalid *serviceerror.InvalidArgument
		if errors.As(err, &invalid) {
			return wfdomain.RunPage{}, apperr.WrapInvalid(err)
		}
		return wfdomain.RunPage{}, err
	}

	runs := make([]wfdomain.RunSummary, 0, len(resp.Executions))
	for _, info := range resp.Executions {
		runs = append(runs, toRunSummary(info))
	}
	return wfdomain.RunPage{
		Runs:          runs,
		NextPageToken: base64.RawURLEncoding.EncodeToString(resp.GetNextPageToken()),
	}, nil
}

// CountRuns นับผ่าน CountWorkflowExecutions ของ Visibility API ด้วย query สถานะชุดเดียวกับ
// ListRuns (statusQueries) ตัวเลขจึงตรงกับจำนวนที่ได้จากการเดินหน้า ListRuns ด้วย filter
// เดียวกันจนหมด
func (e *Engine) CountRuns(ctx context.Context, status wfdomain.RunStatusFilter) (int64, error) {
	resp, err := e.c.CountWorkflow(ctx, &workflowservice.CountWorkflowExecutionsRequest{
		Namespace: Namespace,
		Query:     statusQueries[status],
	})
	if err != nil {
		return 0, err
	}
	return resp.GetCount(), nil
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
