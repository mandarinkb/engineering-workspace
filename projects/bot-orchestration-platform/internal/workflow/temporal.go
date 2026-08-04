package workflow

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// Namespace คือ Temporal namespace เดียวที่ BOP ใช้ตอนนี้ — "default" คือ namespace ที่
// temporalio/auto-setup (ดู docker-compose.yml) สร้างให้อัตโนมัติตั้งแต่ container
// เริ่มครั้งแรก ไม่ต้องสร้างเพิ่มเอง ทั้ง cmd/bop (ตอน query ผ่าน HTTP handler) และ
// cmd/bop-worker (ตอน register worker + สร้าง Schedule) ต้องใช้ค่าเดียวกันนี้
const Namespace = "default"

// TaskQueueName คือชื่อ Task Queue เดียวที่ทั้งฝั่งสั่งงาน (cmd/bop-worker ตอนสร้าง
// Schedule ที่จะ StartWorkflow) และฝั่งรับงาน (cmd/bop-worker ตอน register Worker เอง)
// ต้องใช้ตรงกัน — Temporal จับคู่ workflow/activity ที่รอถูกรันเข้ากับ worker ผ่านชื่อนี้
// เท่านั้น ถ้าตั้งไม่ตรงกัน worker จะไม่มีวันเห็นงานที่ค้างอยู่เลย (งานจะค้างอยู่ใน task
// queue เฉยๆ จนกว่าจะมี worker ที่ poll queue ชื่อเดียวกันมาหยิบไปทำ)
const TaskQueueName = "bop-workflows"

// ActivityName คือชื่อที่ใช้ผูก Activity ตอน register (worker.RegisterActivityWithOptions
// ใน cmd/bop-worker) เข้ากับตอนสั่งเรียกจาก workflow (ExecuteActivity ด้านล่าง) — ใช้
// string คงที่แทนการอ้างอิง method ของ Activities ตรงๆ เพื่อให้ไฟล์นี้ (โค้ดของ workflow
// function) ไม่ต้อง import Activities struct จาก activity.go เลย ตามหลักการที่
// books/13-temporal/01-workflow-and-activity.md อธิบายไว้ว่า workflow function ไม่ควร
// ผูกกับ activity implementation ตรงๆ แค่ต้องรู้ "ชื่อ" กับ "input/output type" เท่านั้น
// (export ไว้เพราะ cmd/bop-worker ต้องใช้ชื่อเดียวกันนี้ตอน RegisterActivityWithOptions)
const ActivityName = "RunStep"

// nonRetryableErrorTypes คือรายชื่อ error type (ต้องตรงกับ Type ที่ตั้งไว้ตอนสร้าง
// temporal.NewApplicationErrorWithCause ใน activity.go) ที่ Temporal จะไม่ retry ให้เลย
// แม้แต่ครั้งเดียว — ทั้งคู่เป็น error ที่ retry กี่ครั้งก็ได้ผลลัพธ์เดิมเป๊ะเสมอ:
//   - "ConfigError" — user กรอก config ของ bot ผิด/ขาด (ดู bot.ConfigError, ตัวอย่างที่
//     internal/bots/httpstatus)
//   - "UnknownBot" — step.BotName ไม่ตรงกับ bot ที่ register ไว้ใน Activities เลย (เพิ่ม
//     เข้ามาหลังเจอปัญหาจริง: schedule ที่มี Args เก่าซึ่ง serialize ไว้ก่อนเพิ่ม json tag
//     ให้ Step จะ decode BotName เป็นค่าว่างเปล่า แล้ว retry ไม่มีที่สิ้นสุดโดยไม่จำเป็น —
//     ดูรายละเอียดที่ activity.go)
var nonRetryableErrorTypes = []string{"ConfigError", "UnknownBot"}

// Execute คือ Temporal workflow function ของ BOP — แทนที่ Runner.Trigger เวอร์ชันก่อน
// Temporal ทั้งหมด รับ Workflow (ลำดับของ Step) มาเป็น input แล้ววน ExecuteActivity
// ทีละ step ตามลำดับเหมือน Runner เดิมทุกประการ (step ถัดไปเริ่มได้ก็ต่อเมื่อ step ก่อน
// หน้าสำเร็จ) — สิ่งที่ต่างไปจริงๆ คือตอนนี้ Temporal เป็นคนบันทึก "event history" ของ
// ทุก step ที่รันไปแล้วให้เองอัตโนมัติ (durable execution): ถ้า worker process (cmd/bop-worker)
// ตายกลางทางที่ workflow กำลังรันอยู่ Temporal จะ "replay" function นี้ใหม่จาก history
// ตอน worker กลับมาออนไลน์ — step ที่มีผลลัพธ์อยู่ใน history อยู่แล้วจะไม่ถูกเรียกซ้ำ
// (ExecuteActivity คืนผลลัพธ์เดิมจาก history ทันที) จึง "resume" ต่อจากจุดที่ค้างได้เอง
// โดยอัตโนมัติ ต่างจากเวอร์ชันก่อนหน้าที่ความคืบหน้าทั้งหมดหายไปทันทีถ้า process ตาย
//
// ข้อกำหนดสำคัญที่ต้องรู้ก่อนแก้ไขไฟล์นี้: Temporal workflow function ต้อง "deterministic"
// เสมอ (ห้ามเรียก time.Now()/math.rand ตรงๆ, ห้ามเปิด goroutine ปกติ, ห้ามทำ I/O ตรงๆ
// เช่น เรียก HTTP/DB เอง) เพราะ Temporal ต้อง replay function นี้ซ้ำได้ผลลัพธ์เดิมทุกครั้ง
// ตอน resume ถ้า logic เปลี่ยนผลลัพธ์ไปตามเวลาจริงหรือ random จะ replay ไม่ตรงกับ history
// เดิมและพัง (Temporal SDK มี workflow.Now()/workflow.SideEffect ให้ใช้แทนถ้าจำเป็นจริงๆ
// แต่ workflow นี้ยังไม่ต้องใช้เพราะ I/O ทั้งหมดถูกย้ายไปทำใน Activity ที่ activity.go แทน)
func Execute(ctx workflow.Context, wf Workflow) ([]StepResult, error) {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			NonRetryableErrorTypes: nonRetryableErrorTypes,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var results []StepResult

	// SetQueryHandler เปิดให้ฝั่งนอก (internal/api/http ผ่าน client.QueryWorkflow) เรียก
	// เข้ามาถามว่า "ตอนนี้รันไปถึง step ไหนแล้ว ผลลัพธ์เป็นยังไง" ได้ ไม่ว่า workflow จะ
	// กำลังรันอยู่หรือจบไปแล้วก็ตาม (Temporal ตอบ query ของ workflow ที่จบแล้วได้ด้วยการ
	// replay history มาหาผลลัพธ์ ตราบใดที่ยังอยู่ในช่วง retention ของ namespace) — นี่คือ
	// สิ่งที่แทนที่ workflow.RunRepository เดิมที่เก็บ Run.Steps ไว้ใน in-memory map ของ
	// BOP เอง ข้อสำคัญ: closure ที่ SetQueryHandler รับต้อง "อ่านอย่างเดียว" ห้ามแก้ state
	// ใดๆ เพราะ Temporal อาจเรียก closure นี้กี่ครั้งก็ได้โดยไม่ใช่ส่วนหนึ่งของ event history
	err := workflow.SetQueryHandler(ctx, "steps", func() ([]StepResult, error) {
		return results, nil
	})
	if err != nil {
		return nil, err
	}

	for i, step := range wf.Steps {
		res := StepResult{StepIndex: i, BotName: step.BotName}

		activityErr := workflow.ExecuteActivity(ctx, ActivityName, step).Get(ctx, &res.Summary)
		if activityErr != nil {
			// step ล้มเหลว (หมดจำนวน retry ตาม RetryPolicy แล้ว หรือเป็น error ที่ตั้งไว้
			// ว่าห้าม retry เลย) — บันทึกผลลัพธ์ของ step นี้ไว้ใน results ก่อน แล้วหยุด
			// ทันที ไม่รัน step ที่เหลือต่อ เหมือน Runner เดิมทุกประการ
			res.Error = activityErr.Error()
			results = append(results, res)
			return results, activityErr
		}

		results = append(results, res)
	}

	return results, nil
}
