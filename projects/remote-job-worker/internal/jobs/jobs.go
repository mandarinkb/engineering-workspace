// Package jobs คือ reference implementation ของฝั่ง "remote worker" ตาม contract ที่
// REMOTE-JOB-WORKER-TODO.md (ใน projects/bot-orchestration-platform/) กำหนดไว้ — เป้าหมาย
// ของไฟล์นี้คือพิสูจน์ว่า contract ทำงานได้จริง (register activity ชื่อ "RunStep" บน
// task queue ของตัวเอง แล้ว BOP (Bot Orchestration Platform) ยิงงานมาหาได้ผ่าน Temporal
// task queue routing) ไม่ใช่ implementation ของ jobcode จริงของเจ้าของ repo (ยังไม่มีให้
// ทดสอบตอนเขียนโค้ดนี้ ใช้ placeholder jobcode ไปก่อนเหมือนที่ทำกับ path ของ binary ใน
// internal/bots/externaljob ฝั่ง BOP)
//
// package นี้ตั้งใจไม่ import อะไรจาก repo ของ BOP เลยแม้แต่บรรทัดเดียว — พิสูจน์หลักการ
// สำคัญที่สุดของ contract นี้: remote worker ไม่จำเป็นต้องรู้จักโค้ดของ BOP เลย รู้แค่
// "JSON shape ของ input" กับ "ชื่อ activity/task queue ที่ตกลงกันไว้" ก็พอ (แม้แต่คนละ
// ภาษาโปรแกรมก็ยังทำได้ เพราะ Temporal คุยกันด้วย JSON ผ่าน default DataConverter)
package jobs

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
)

// Step คือ input ที่ RunStep รับ — จงใจประกาศ struct นี้ขึ้นมาเองแทนที่จะ import
// package "bop/internal/workflow" จาก repo ของ BOP ตรงๆ (แม้จะทำได้ถ้าอยาก type-safe
// กว่านี้) เพื่อสาธิตว่า decouple กันจริงตาม contract — แค่ json tag ตรงกับ
// workflow.Step ของฝั่ง BOP ก็พอ ฟิลด์ที่ไม่ได้ใช้ (เช่น TaskQueue ที่ BOP ส่งมาด้วยเสมอ
// เพราะเป็นส่วนหนึ่งของ struct เดียวกัน) ไม่ต้องประกาศเลย — encoding/json (ซึ่ง Temporal
// ใช้เป็น default DataConverter) จะข้าม field ที่ struct นี้ไม่รู้จักไปเงียบๆ ตอน decode
type Step struct {
	BotName        string            `json:"bot_name"`
	Config         map[string]string `json:"config"`
	TimeoutSeconds int               `json:"timeout_seconds,omitempty"`
}

// heartbeatInterval คือความถี่ที่ RunStep เรียก activity.RecordHeartbeat ระหว่างทำงาน
// jobcode ที่ใช้เวลานาน (ดู case "LONG_TASK" ด้านล่าง) — ตั้งสั้นกว่า StartToCloseTimeout
// ที่ BOP ตั้งไว้มากพอสมควร เพื่อให้ Temporal เห็นสัญญาณ "ยังทำงานอยู่จริง" บ่อยพอ ไม่เข้าใจ
// ผิดว่า worker ค้าง/ตายไปแล้วทั้งที่ยังทำงานอยู่ (ตาม REMOTE-JOB-WORKER-TODO.md ข้อ 6)
const heartbeatInterval = 2 * time.Second

// longTaskDuration คือเวลาที่จำลองให้ jobcode "LONG_TASK" ใช้ทำงาน
const longTaskDuration = 10 * time.Second

// RunStep คือ Temporal Activity function ตัวเดียวที่ cmd/worker/main.go register ไว้ภายใต้
// ชื่อ "RunStep" (ต้องตรงกับ workflow.ActivityName ของฝั่ง BOP เป๊ะ — ดู
// REMOTE-JOB-WORKER-TODO.md ข้อ 3) — dispatch ตาม step.Config["jobcode"] เหมือนที่ binary
// เดิมของเจ้าของ repo dispatch ผ่าน flag --jobcode ทุกประการ ต่างกันแค่ entry point
// (จาก os.Args/flag.Parse() เปลี่ยนมาเป็น activity function ที่ Temporal เรียกให้แทน)
//
// jobcode ที่ implement ไว้เป็นตัวอย่าง (placeholder เพราะยังไม่รู้ jobcode จริง):
//   - "PING"       — สำเร็จทันที ใช้ smoke test ว่า wiring ทั้งหมดถูกต้อง (เทียบเท่า
//     http-status-checker ฝั่ง BOP ที่เป็น bot ที่ง่ายที่สุด)
//   - "LONG_TASK"  — จำลองงานที่ใช้เวลานาน เรียก activity.RecordHeartbeat เป็นระยะ และ
//     เช็ค ctx.Done() เพื่อหยุดทันทีถ้าถูกยกเลิก/timeout (ตรงกับ contract ของ Temporal
//     Activity ที่ดีทุกตัว — เทียบเท่าที่ bot.Bot.Run ฝั่ง BOP ต้องเช็ค ctx.Done() เหมือนกัน)
//   - "FAIL_TRANSIENT" — จำลอง error ชั่วคราว (เช่น dependency ภายนอกของ job เองล่ม
//     ชั่วคราว) คืน error ธรรมดา (retryable) ให้ Temporal retry ตาม RetryPolicy default
//     ของฝั่ง BOP ต่อไป
//   - jobcode ว่างเปล่า หรือไม่ตรงกับ case ไหนเลย — ถือเป็นปัญหาที่ retry ไปก็ผลลัพธ์เดิม
//     แน่นอน (jobcode ผิด ไม่ได้เปลี่ยนเองระหว่าง retry) จึงห่อเป็น
//     temporal.NewApplicationErrorWithCause ชนิด "ConfigError" ให้ตรงกับ
//     nonRetryableErrorTypes ที่ฝั่ง BOP ประกาศไว้ (internal/workflow/temporal.go) —
//     Temporal จะไม่ retry error ชนิดนี้เลยแม้แต่ครั้งเดียว
func RunStep(ctx context.Context, step Step) (string, error) {
	jobcode := step.Config["jobcode"]

	switch jobcode {
	case "PING":
		return "pong", nil

	case "LONG_TASK":
		return runLongTask(ctx, longTaskDuration, heartbeatInterval)

	case "FAIL_TRANSIENT":
		return "", fmt.Errorf("remote-job-worker: jobcode %q failed due to a transient dependency error (retry should be attempted)", jobcode)

	default:
		err := fmt.Errorf("remote-job-worker: unknown or missing jobcode %q", jobcode)
		return "", temporal.NewApplicationErrorWithCause(err.Error(), "ConfigError", err)
	}
}

// runLongTask จำลอง job ที่ใช้เวลานาน โดยแบ่งเป็นช่วงสั้นๆ (heartbeatEvery ต่อรอบ) แล้ว
// เช็ค 2 อย่างทุกรอบ: (1) ctx.Done() — ถ้า BOP สั่งยกเลิกหรือ timeout ครบตามที่ตั้งไว้ ต้อง
// หยุดทำงานทันที ไม่ทำงานต่อไปเรื่อยๆ โดยไม่สนใจ (เหมือน contract ของ bot.Bot.Run ฝั่ง BOP)
// และ (2) ส่ง heartbeat เป็นระยะผ่าน activity.RecordHeartbeat เพื่อบอก Temporal ว่า "ยัง
// ทำงานอยู่จริง ไม่ได้ค้าง" — สำคัญมากสำหรับ activity ที่ใช้เวลานานใกล้ๆ
// StartToCloseTimeout เพราะ Temporal server เองตรวจจับ worker process ตายไม่ได้โดยตรง
// ต้องอาศัย timeout/heartbeat ในการตัดสินใจ — รับ total/heartbeatEvery เป็น parameter
// (แทนที่จะ hardcode ในฟังก์ชันตรงๆ) เพื่อให้ unit test เรียกด้วยค่าสั้นๆ ได้โดยไม่ต้องรอ
// จริง 10 วินาทีทุกครั้งที่รัน test (ดู jobs_test.go)
func runLongTask(ctx context.Context, total, heartbeatEvery time.Duration) (string, error) {
	ticker := time.NewTicker(heartbeatEvery)
	defer ticker.Stop()

	deadline := time.Now().Add(total)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			activity.RecordHeartbeat(ctx, "still working")
		}
	}

	return "long_task completed", nil
}
