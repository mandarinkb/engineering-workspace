package workflow

import (
	"context"
	"fmt"
	"time"

	"bop/internal/job"
	"bop/internal/worker"
)

// Runner รัน Workflow ทีละ step ตามลำดับ โดยใช้ pool.SubmitAndWait รอให้แต่ละ step
// จบก่อนจะเริ่ม step ถัดไป — ถ้า step ไหนไม่สำเร็จ (failed หรือ cancelled) จะหยุดทันที
// ไม่รัน step ที่เหลือต่อ (ยังไม่มี retry หรือ compensation logic อัตโนมัติในเวอร์ชันนี้
// ดูหมายเหตุข้อจำกัดที่หัวไฟล์ workflow.go)
type Runner struct {
	pool *worker.Pool
	repo RunRepository
}

// NewRunner สร้าง Runner ที่ผูกกับ worker pool และที่เก็บ Run ที่กำหนด
func NewRunner(pool *worker.Pool, repo RunRepository) *Runner {
	return &Runner{pool: pool, repo: repo}
}

// Trigger เริ่มรัน Workflow หนึ่งรอบทันที (แบบ synchronous — บล็อครอจนกว่า workflow
// ทั้งก้อนจะจบ ไม่ว่าจะสำเร็จหรือล้มเหลว) เหมาะสำหรับให้ Scheduler เรียกใน goroutine
// แยกของตัวเอง (ดู internal/scheduler) เพื่อไม่ให้ workflow หนึ่งตัวที่รันนานไปบล็อค
// การเช็ค schedule ของ workflow ตัวอื่น
//
// ขั้นตอนการทำงาน:
//  1. สร้าง Run record ใหม่ สถานะ running บันทึกลง repository ทันที
//  2. วนลูปตาม Steps ทีละตัวตามลำดับ สร้าง job.Job ให้แต่ละ step (ID ผูกกับ Run.ID
//     และลำดับ step เพื่อให้อ่านแล้วรู้ทันทีว่า job ไหนเป็นของ workflow run ไหน step ที่เท่าไหร่)
//  3. เรียก pool.SubmitAndWait รอผลของ step นั้นก่อนไปต่อ
//  4. ถ้า step ไหนล้มเหลว (error จาก SubmitAndWait เอง เช่น ctx ของ Trigger หมดเวลา
//     หรือ job จบด้วยสถานะที่ไม่ใช่ succeeded) ให้บันทึก Run เป็น failed พร้อมระบุว่า
//     step ไหนที่พัง แล้วหยุดทันที ไม่รัน step ที่เหลือ
//  5. ถ้าทุก step ผ่านหมด บันทึก Run เป็น succeeded
func (r *Runner) Trigger(ctx context.Context, wf Workflow) (*Run, error) {
	run := &Run{
		ID:           newRunID(),
		WorkflowName: wf.Name,
		Status:       RunRunning,
		CreatedAt:    time.Now(),
	}
	_ = r.repo.Create(ctx, run)

	for i, step := range wf.Steps {
		j := &job.Job{
			ID:        fmt.Sprintf("%s-step%d", run.ID, i),
			BotName:   step.BotName,
			Config:    step.Config,
			Status:    job.StatusPending,
			CreatedAt: time.Now(),
		}

		finished, err := r.pool.SubmitAndWait(ctx, j)
		if err != nil {
			// err ตรงนี้คือ ctx ของ Trigger เองถูกยกเลิก/timeout ไม่ใช่ error จากตัว bot
			// (error จากตัว bot สะท้อนออกมาผ่าน finished.Status/finished.Error แทน ซึ่งเช็คด้านล่าง)
			return r.fail(ctx, run, i, step, err.Error())
		}

		run.Steps = append(run.Steps, StepResult{JobID: finished.ID, Status: finished.Status})
		_ = r.repo.Update(ctx, run) // อัปเดตความคืบหน้าให้เห็นทันทีหลังจบแต่ละ step ไม่ต้องรอจนจบทั้ง workflow

		if finished.Status != job.StatusSucceeded {
			return r.fail(ctx, run, i, step, finished.Error)
		}
	}

	run.Status = RunSucceeded
	end := time.Now()
	run.EndedAt = &end
	_ = r.repo.Update(ctx, run)
	return run, nil
}

// fail คือ helper บันทึก Run เป็นสถานะ failed พร้อมข้อความบอกว่า step ไหนพังเพราะอะไร
// แล้วคืน error กลับไปให้ผู้เรียก Trigger รู้ด้วย (ไม่ใช่แค่เงียบๆ อัปเดตสถานะเฉยๆ)
func (r *Runner) fail(ctx context.Context, run *Run, stepIndex int, step Step, reason string) (*Run, error) {
	run.Status = RunFailed
	run.Error = fmt.Sprintf("step %d (%s): %s", stepIndex, step.BotName, reason)
	end := time.Now()
	run.EndedAt = &end
	_ = r.repo.Update(ctx, run)
	return run, fmt.Errorf("workflow %s failed at step %d (%s): %s", run.WorkflowName, stepIndex, step.BotName, reason)
}
