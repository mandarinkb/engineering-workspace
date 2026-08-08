// Command worker คือจุดเริ่มต้นของ remote job worker — process แยกต่างหากจาก Bot
// Orchestration Platform (BOP) โดยสมบูรณ์ (คนละ repo, คนละ Go module, deploy คนละจังหวะ
// กันได้จริง) ที่ทำหน้าที่ "รับงาน" ที่ BOP ส่งมาผ่าน Temporal task queue ของตัวเอง แทนที่
// จะให้ BOP shell out มาเรียก binary นี้ตรงๆ ในเครื่องเดียวกัน (แบบ
// internal/bots/externaljob ของฝั่ง BOP) — ดู contract เต็มๆ ที่
// REMOTE-JOB-WORKER-TODO.md ใน projects/bot-orchestration-platform/
//
// process นี้ต่อเข้า Temporal server เดียวกับที่ BOP ใช้ (namespace/host:port ต้องตรงกัน
// เสมอ) แล้ว poll task queue ชื่อของตัวเอง (taskQueueName ด้านล่าง — ต้องตรงกับค่าที่ BOP
// ระบุใน workflow.Step.TaskQueue ตอนสั่งงานมา) รอรับ Activity call ชื่อ "RunStep" ซึ่งเป็น
// ชื่อเดียวกับที่ BOP ใช้เรียก local bot ของตัวเองด้วย — Temporal เป็นคนจับคู่ให้เองผ่าน
// task queue routing ไม่ต้องเขียน logic ตัดสินใจ "งานนี้ควรไปที่ไหน" เองเลยสักบรรทัด
package main

import (
	"log"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"remote-job-worker/internal/jobs"
)

// namespace ต้องตรงกับ workflow.Namespace ของฝั่ง BOP (internal/workflow/temporal.go) เสมอ
// — ทั้งคู่ต้องต่อ Temporal server namespace เดียวกัน ไม่งั้นจะไม่มีวันเจองานของกันและกันเลย
const namespace = "default"

// taskQueueName คือชื่อ task queue ของ worker ตัวนี้เอง — ต้องไม่ชนกับ "bop-workflows"
// (ชื่อ task queue ของฝั่ง BOP เอง ดู workflow.TaskQueueName) และต้องเป็นชื่อเดียวกับที่
// ระบุใน workflow.Step.TaskQueue ตอนฝั่ง BOP สั่งงานมา (ผ่าน POST /workflow-runs เช่น
// {"task_queue": "external-job-queue"}) — ถ้าตั้งชื่อไม่ตรงกัน Temporal จะเก็บงานนั้นค้าง
// ไว้ใน queue เฉยๆ ไม่มี worker ตัวไหนมา poll เจอเลย
const taskQueueName = "external-job-queue"

// activityName ต้องตรงกับ workflow.ActivityName ของฝั่ง BOP ("RunStep") เป๊ะทุกตัวอักษร —
// Temporal จับคู่ ExecuteActivity(ctx, "RunStep", step) ของฝั่ง BOP เข้ากับ activity ที่
// register ไว้ภายใต้ชื่อนี้ที่นี่ ผ่านชื่อ (string) ล้วนๆ ไม่ได้ผ่าน function reference ตรงๆ
// ข้ามกระบวนการ/repo กัน (เหตุผลเดียวกับที่ workflow.ActivityName ฝั่ง BOP อธิบายไว้)
const activityName = "RunStep"

func main() {
	// ขั้นตอนที่ 1: ต่อ Temporal client — host:port นี้คือค่า default ตอน dev local
	// (ดู docker-compose.yml ของ projects/bot-orchestration-platform/) production ต้อง
	// เปลี่ยนให้ตรงกับ Temporal server จริงที่ BOP ต่ออยู่ (hardcode ไปก่อนตาม pattern
	// เดียวกับค่าคงที่อื่นๆ ในทั้งสอง repo เช่น ":8080" ของ cmd/bop, ไม่เพิ่ม flag/env var
	// จนกว่าจะมีความจำเป็นจริง)
	c, err := client.Dial(client.Options{
		HostPort:  "localhost:7233",
		Namespace: namespace,
	})
	if err != nil {
		log.Fatalf("connect to temporal: %v", err)
	}
	defer c.Close()

	// ขั้นตอนที่ 2: สร้าง Worker ที่ poll task queue ของตัวเอง (ไม่ใช่ "bop-workflows")
	// แล้ว register activity function เดียว (jobs.RunStep) ภายใต้ชื่อ "RunStep" ให้ตรงกับ
	// ฝั่ง BOP เป๊ะ — ตัว worker นี้ "ไม่" register workflow ใดๆ เลย (ต่างจาก
	// cmd/bop-worker ของฝั่ง BOP ที่ register ทั้ง workflow.Execute และ activity) เพราะ
	// worker ตัวนี้มีหน้าที่แค่รับ-ทำ Activity เท่านั้น ตัว workflow orchestration
	// (ลำดับ step, retry policy, durable execution) ยังคงเป็นหน้าที่ของฝั่ง BOP ทั้งหมด
	w := worker.New(c, taskQueueName, worker.Options{})
	w.RegisterActivityWithOptions(jobs.RunStep, activity.RegisterOptions{Name: activityName})

	// ขั้นตอนที่ 3: รัน Worker ค้างอยู่ (poll งานเรื่อยๆ) จนกว่าจะได้รับ interrupt signal
	// (Ctrl+C) — worker.InterruptCh() คือ helper ของ SDK เอง ดักจับ signal มาตรฐานให้แล้ว
	// (pattern เดียวกับ cmd/bop-worker/main.go ของฝั่ง BOP ทุกประการ)
	log.Printf("remote-job-worker listening on task queue %q (namespace %q)", taskQueueName, namespace)
	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatalf("worker stopped with error: %v", err)
	}
}
