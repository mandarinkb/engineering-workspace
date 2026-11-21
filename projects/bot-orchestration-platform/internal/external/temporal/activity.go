package temporal

import (
	"context"
	"errors"
	"fmt"

	"go.temporal.io/sdk/activity"
	sdktemporal "go.temporal.io/sdk/temporal"

	"bop/internal/domain/bot"
	"bop/internal/domain/logger"
	wfdomain "bop/internal/domain/workflow"
)

// Activities คือตัวห่อ (adapter) บางๆ ที่ทำให้ bot.Bot ทุกตัวที่ register ไว้ถูกเรียกจาก
// Temporal Activity ได้ — ไม่ได้เขียน bot logic ใหม่เลยแม้แต่บรรทัดเดียวตามที่
// TEMPORAL-MIGRATION-TODO.md ระบุไว้ (ข้อ 3) แค่แปลง signature ให้ตรงกับที่ Temporal SDK
// ต้องการ (activity function รับ context.Context ธรรมดา ไม่ใช่ workflow.Context) แล้ว
// เรียก bot.Run ต่ออีกทีเท่านั้น
//
// ต่างจาก worker.Pool ตรงที่ Activities "ไม่" ทำ concurrency limiting/timeout ของตัวเอง
// (ไม่มี semaphore แบบ worker.Pool.sem) เพราะ Temporal Worker เอง (ดู cmd/bop-worker)
// มีกลไกจำกัดจำนวน activity ที่รันพร้อมกันอยู่แล้วผ่าน worker.Options
// (MaxConcurrentActivityExecutionSize) และ timeout ต่อ activity ก็ตั้งผ่าน
// workflow.ActivityOptions.StartToCloseTimeout แทน (ดู workflow.go) — worker.Pool เดิม
// ยังถูกใช้อยู่ แต่เฉพาะกับ job เดี่ยวๆ ที่สั่งตรงผ่าน POST /bots/{name}/jobs เท่านั้น
// (ดู internal/adapter/http/create_job.go) ไม่เกี่ยวกับ workflow ที่รันผ่าน Temporal อีกต่อไป
type Activities struct {
	bots map[string]bot.Bot
	log  logger.StepLogger
}

// NewActivities สร้าง Activities เปล่าๆ พร้อม logger ที่จะใช้บันทึก log ความคืบหน้าของ
// แต่ละ step — ต้องเรียก Register ให้ครบทุก bot ที่ workflow อาจสั่งใช้ก่อนเริ่ม Worker
// (ดู cmd/bop-worker/main.go) เหมือนกับที่ cmd/bop เดิมต้อง pool.Register ให้ครบก่อน serve
func NewActivities(log logger.StepLogger) *Activities {
	return &Activities{bots: make(map[string]bot.Bot), log: log}
}

// Register เพิ่ม bot ชนิดหนึ่งเข้าไปให้ Activities รู้จัก — หลักการเดียวกับ
// worker.Pool.Register ทุกประการ แค่คนละ registry กัน (และอยู่คนละ process กันจริงๆ ด้วย
// เพราะแยก cmd/bop-worker ออกมาต่างหากตามที่ตัดสินใจไว้ — เพิ่ม bot ชนิดใหม่ต้อง register
// ทั้งสองที่: pool.Register ใน cmd/bop สำหรับ job เดี่ยว และ activities.Register ใน
// cmd/bop-worker สำหรับ step ใน workflow)
func (a *Activities) Register(b bot.Bot) {
	a.bots[b.Name()] = b
}

// RunStep คือ Temporal Activity function ตัวเดียวที่ Execute (workflow.go) เรียกสำหรับ
// ทุก step ในทุก workflow — รับ Step (BotName + Config) เป็น input คืนข้อความสรุป
// ผลลัพธ์ (bot.Result.Summary) เป็น output
//
// ขั้นตอนการทำงาน:
//  1. หา bot ที่ตรงกับ step.BotName จาก registry — ถ้าไม่รู้จัก ห่อเป็น
//     temporal.NewApplicationError ชนิด "UnknownBot" (non-retryable เหมือน ConfigError
//     ด้านล่าง ดูเหตุผลที่ nonRetryableErrorTypes ใน workflow.go) **เดิม (ก่อนแก้ไขนี้)
//     ปล่อยเป็น error ธรรมดาตามหลัก YAGNI เพราะตอนนั้นยังไม่มี use case จริงที่ workflow
//     อ้างชื่อ bot ผิด — แก้เป็น non-retryable ตอนนี้เพราะเจอปัญหาจริงแล้ว: เวอร์ชันเดิม
//     ทำให้ schedule ที่มี step.BotName ว่างเปล่า (เช่น decode payload เก่าที่ serialize
//     ไว้ก่อนเพิ่ม json tag ให้ Step ไม่ตรงกับ struct เวอร์ชันใหม่) retry ไม่มีที่สิ้นสุด
//     (ไม่มี MaximumAttempts กำกับ) เสีย worker resource ไปเรื่อยๆ ทั้งที่ retry กี่ครั้ง
//     ก็ได้ผลลัพธ์เดิมแน่นอน (ชื่อ bot ที่ผิดไม่มีทางเปลี่ยนเองระหว่าง retry)
//  2. เรียก b.Run พร้อม log function ที่ผูกเจ้าของ log ไว้แล้ว: workflow ID + ลำดับ Step
//     (stepIndex ที่ Execute ส่งมา) + หมายเลข Attempt จาก activity info — ทุก Attempt ของ Step
//     เดียวกันจึงอ่านรวมกันได้ทีหลังผ่าน GET /workflow-runs/{id}/steps/{index}/logs
//  3. ถ้า error เป็น *bot.ConfigError (เช็คด้วย errors.As ซึ่งมองทะลุผ่าน wrapper ไปหา
//     type จริงข้างในได้ ดู bot.ConfigError.Unwrap) ห่อกลับเป็น
//     temporal.NewApplicationErrorWithCause พร้อมตั้ง Type เป็น "ConfigError" ให้ตรงกับ
//     nonRetryableErrorTypes ใน workflow.go เป๊ะ — Temporal เช็ค Type นี้เพื่อตัดสินใจว่า
//     จะ retry activity นี้ต่อไหม ถ้าไม่ห่อแบบนี้ Temporal จะมองเป็น error ทั่วไปแล้ว retry
//     จนครบจำนวน Attempt สูงสุด (ทั้งที่ config ผิดเหมือนเดิมทุกครั้ง — เสียเวลาไปฟรีๆ) ส่วน
//     error ชนิดอื่นปล่อยเป็น error ธรรมดา ให้ Temporal retry ตาม stepRetryPolicy ใน
//     workflow.go (เผื่อเป็นปัญหาชั่วคราว เช่น network timeout)
//
// ทุก Attempt ที่ล้มเหลว (รวม UnknownBot) log error เป็นบรรทัดสุดท้ายของ Attempt นั้น เพื่อให้
// operator เห็นว่า Attempt ก่อนหน้าล้มเหลวเพราะอะไรโดยไม่ต้องเปิด Temporal UI
//
// stepIndex เป็น argument ที่สองแทนการแปะลงใน Step เพราะ Step คือ payload ที่ผู้ใช้ส่งมาและ
// Schedule เก็บไว้ — remote worker (projects/remote-job-worker) ที่ register RunStep แบบรับ Step
// อย่างเดียวยังทำงานได้ เพราะ SDK ทิ้ง argument ที่เกินจำนวน parameter ไปเอง ส่วน Attempt ที่
// ถูก schedule ไว้ก่อน deploy เวอร์ชันนี้จะได้ stepIndex เป็น 0
//
// log ของ Step อ่านผ่าน bop-api ได้เมื่อ bop-worker ตั้ง BOP_DATABASE_URL (logger บน PostgreSQL
// ตัวเดียวกับ bop-api) ถ้าไม่ตั้งจะอยู่ใน memory ของ bop-worker และอ่านจาก API ไม่ได้
func (a *Activities) RunStep(ctx context.Context, step wfdomain.Step, stepIndex int) (string, error) {
	owner := stepAttempt(ctx, stepIndex)
	log := func(msg string) { a.log.LogStep(ctx, owner, logger.LevelInfo, msg) }
	fail := func(err error) error {
		a.log.LogStep(ctx, owner, logger.LevelError, err.Error())
		return err
	}

	b, ok := a.bots[step.BotName]
	if !ok {
		err := fmt.Errorf("workflow: unknown bot %q", step.BotName)
		return "", fail(sdktemporal.NewApplicationErrorWithCause(err.Error(), "UnknownBot", err))
	}

	result, err := b.Run(ctx, log, step.Config)
	if err != nil {
		var cfgErr *bot.ConfigError
		if errors.As(err, &cfgErr) {
			return "", fail(sdktemporal.NewApplicationErrorWithCause(err.Error(), "ConfigError", err))
		}
		return "", fail(err)
	}

	return result.Summary, nil
}

// stepAttempt คือเจ้าของ log ของ Attempt ที่กำลังรันอยู่ (Temporal นับ Attempt เริ่มที่ 1)
func stepAttempt(ctx context.Context, stepIndex int) logger.StepAttempt {
	info := activity.GetInfo(ctx)
	return logger.StepAttempt{
		WorkflowID: info.WorkflowExecution.ID,
		StepIndex:  stepIndex,
		Attempt:    int(info.Attempt),
	}
}
