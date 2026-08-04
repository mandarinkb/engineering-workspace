package workflow

import (
	"context"
	"errors"
	"fmt"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"

	"bop/internal/bot"
	"bop/internal/logger"
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
// workflow.ActivityOptions.StartToCloseTimeout แทน (ดู temporal.go) — worker.Pool เดิม
// ยังถูกใช้อยู่ แต่เฉพาะกับ job เดี่ยวๆ ที่สั่งตรงผ่าน POST /bots/{name}/jobs เท่านั้น
// (ดู internal/api/http/handler.go) ไม่เกี่ยวกับ workflow ที่รันผ่าน Temporal อีกต่อไป
type Activities struct {
	bots map[string]bot.Bot
	log  logger.Logger
}

// NewActivities สร้าง Activities เปล่าๆ พร้อม logger ที่จะใช้บันทึก log ความคืบหน้าของ
// แต่ละ step — ต้องเรียก Register ให้ครบทุก bot ที่ workflow อาจสั่งใช้ก่อนเริ่ม Worker
// (ดู cmd/bop-worker/main.go) เหมือนกับที่ cmd/bop เดิมต้อง pool.Register ให้ครบก่อน serve
func NewActivities(log logger.Logger) *Activities {
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

// RunStep คือ Temporal Activity function ตัวเดียวที่ Execute (temporal.go) เรียกสำหรับ
// ทุก step ในทุก workflow — รับ Step (BotName + Config) เป็น input คืนข้อความสรุป
// ผลลัพธ์ (bot.Result.Summary) เป็น output
//
// ขั้นตอนการทำงาน:
//  1. หา bot ที่ตรงกับ step.BotName จาก registry — ถ้าไม่รู้จัก ห่อเป็น
//     temporal.NewApplicationError ชนิด "UnknownBot" (non-retryable เหมือน ConfigError
//     ด้านล่าง ดูเหตุผลที่ nonRetryableErrorTypes ใน temporal.go) **เดิม (ก่อนแก้ไขนี้)
//     ปล่อยเป็น error ธรรมดาตามหลัก YAGNI เพราะตอนนั้นยังไม่มี use case จริงที่ workflow
//     อ้างชื่อ bot ผิด — แก้เป็น non-retryable ตอนนี้เพราะเจอปัญหาจริงแล้ว: เวอร์ชันเดิม
//     ทำให้ schedule ที่มี step.BotName ว่างเปล่า (เช่น decode payload เก่าที่ serialize
//     ไว้ก่อนเพิ่ม json tag ให้ Step ไม่ตรงกับ struct เวอร์ชันใหม่) retry ไม่มีที่สิ้นสุด
//     (ไม่มี MaximumAttempts กำกับ) เสีย worker resource ไปเรื่อยๆ ทั้งที่ retry กี่ครั้ง
//     ก็ได้ผลลัพธ์เดิมแน่นอน (ชื่อ bot ที่ผิดไม่มีทางเปลี่ยนเองระหว่าง retry)
//  2. เรียก b.Run พร้อม log function ที่ผูก "log ID" ของ step นี้ไว้แล้ว (ดูหมายเหตุยาว
//     ด้านล่างเรื่อง log ID)
//  3. ถ้า error เป็น *bot.ConfigError (เช็คด้วย errors.As ซึ่งมองทะลุผ่าน wrapper ไปหา
//     type จริงข้างในได้ ดู bot.ConfigError.Unwrap) ห่อกลับเป็น
//     temporal.NewApplicationErrorWithCause พร้อมตั้ง Type เป็น "ConfigError" ให้ตรงกับ
//     nonRetryableErrorTypes ใน temporal.go เป๊ะ — Temporal เช็ค Type นี้เพื่อตัดสินใจว่า
//     จะ retry activity นี้ต่อไหม ถ้าไม่ห่อแบบนี้ Temporal จะมองเป็น error ทั่วไปแล้ว retry
//     ตาม RetryPolicy default (ซึ่งจะ retry ซ้ำไปเรื่อยๆ ทั้งที่ config ผิดเหมือนเดิมทุกครั้ง
//     — เสียเวลาและ resource ไปฟรีๆ) ส่วน error ชนิดอื่นปล่อยเป็น error ธรรมดา ให้ Temporal
//     retry ตาม RetryPolicy default ไป (เผื่อเป็นปัญหาชั่วคราว เช่น network timeout)
//
// หมายเหตุเรื่อง log ID (สำคัญ — อ่านก่อนแก้): เวอร์ชันก่อน Temporal (Runner เดิม) สร้าง
// job.Job จริงลง job.Repository ให้ทุก step (ID รูปแบบ "<runID>-step<N>") ทำให้
// GET /jobs/{id}/logs เดิมใช้ดู log ของ step ใน workflow ได้ด้วย — เวอร์ชันนี้ไม่สร้าง
// job.Repository entry ให้ step ของ workflow อีกต่อไป (job.Repository เก็บไว้เฉพาะ job
// เดี่ยวที่ไม่ผ่าน workflow แล้ว ตาม Open Question ข้อ 3 ที่ตัดสินใจไว้ใน
// TEMPORAL-MIGRATION-TODO.md) จึงผูก log ไว้กับ "<Temporal WorkflowID>-<ActivityID>" แทน
//
// ข้อจำกัดที่ต้องรู้ตอนนี้: NewActivities ถูกสร้างขึ้นใน cmd/bop-worker ซึ่งเป็นคนละ
// process กับ cmd/bop (ที่ HTTP handler `GET /jobs/{id}/logs` อ่าน log อยู่) — logger.Logger
// ตัวที่ใช้ตอนนี้เป็น in-memory ล้วนๆ (ยังไม่มี PostgreSQL ตาม roadmap Phase 2.2) ข้อมูล
// จึงอยู่คนละ memory space กัน ทำให้ log ของ step ใน workflow "บันทึกจริง" (เห็นใน stdout
// ของ cmd/bop-worker ผ่าน log.Log) แต่ "ดึงผ่าน HTTP ของ cmd/bop ไม่ได้" จนกว่าจะสลับไปใช้
// backend ที่ทั้งสอง process แชร์กันได้จริง (PostgreSQL/OpenSearch) — เขียน interface
// เดียวกันไว้รอเปลี่ยนแค่ 2 บรรทัดตอนนั้นได้เลยเหมือนกับ job.Repository เดิม
func (a *Activities) RunStep(ctx context.Context, step Step) (string, error) {
	b, ok := a.bots[step.BotName]
	if !ok {
		err := fmt.Errorf("workflow: unknown bot %q", step.BotName)
		return "", temporal.NewApplicationErrorWithCause(err.Error(), "UnknownBot", err)
	}

	info := activity.GetInfo(ctx)
	logID := fmt.Sprintf("%s-%s", info.WorkflowExecution.ID, info.ActivityID)

	log := func(msg string) { a.log.Log(ctx, logID, logger.LevelInfo, msg) }
	result, err := b.Run(ctx, log, step.Config)
	if err != nil {
		var cfgErr *bot.ConfigError
		if errors.As(err, &cfgErr) {
			return "", temporal.NewApplicationErrorWithCause(err.Error(), "ConfigError", err)
		}
		return "", err
	}

	return result.Summary, nil
}
