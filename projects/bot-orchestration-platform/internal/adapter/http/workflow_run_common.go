package http

import (
	"time"

	"bop/internal/domain/workflow"
)

// workflowRunSummary คือรูปแบบ JSON ที่คืนให้ dashboard ดู — แปลงมาจาก
// workflow.RunSummary ของ domain (ซึ่ง adapter ฝั่ง Temporal แปลงมาจาก protobuf type
// อีกทอดหนึ่ง) เก็บไว้แค่เท่าที่ dashboard ต้องใช้จริง
type workflowRunSummary struct {
	WorkflowID   string     `json:"workflow_id"`
	RunID        string     `json:"run_id"`
	WorkflowName string     `json:"workflow_name"`
	Status       string     `json:"status"`
	StartedAt    time.Time  `json:"started_at"`
	EndedAt      *time.Time `json:"ended_at,omitempty"`
	// ScheduleID ส่งเป็น "" เสมอสำหรับ run ที่สั่งเอง (ไม่ละ field) ให้ client เช็คค่าเดียวพอ
	ScheduleID string `json:"schedule_id"`
}

// toRunSummary แปลง workflow.RunSummary ของ domain เป็นรูปแบบ JSON ของ API
func toRunSummary(run workflow.RunSummary) workflowRunSummary {
	return workflowRunSummary{
		WorkflowID:   run.WorkflowID,
		RunID:        run.RunID,
		WorkflowName: run.WorkflowName,
		Status:       run.Status,
		StartedAt:    run.StartedAt,
		EndedAt:      run.EndedAt,
		ScheduleID:   run.ScheduleID,
	}
}
