// Command bop-worker คือ Temporal Worker process ของ BOP — แยกออกมาจาก cmd/bop (HTTP
// API server) ตามที่ตัดสินใจไว้ใน Open Question ข้อ 4 ของ TEMPORAL-MIGRATION-TODO.md
// เหตุผลที่แยก: Temporal ใช้งานจริงใน production แทบทุกที่แยก worker process ออกจาก
// API server เสมอ เพราะ scale/deploy กันคนละจังหวะได้อิสระ (worker ต้อง scale ตามปริมาณ
// workflow/activity ที่รันอยู่ ส่วน API server scale ตามปริมาณ HTTP request) — แลกกับ
// ความซับซ้อนที่ต้องรัน 2 process ตอน local dev แทนที่จะเป็น 1 (ดู README.md "รันยังไง")
//
// process นี้ทำ 2 อย่าง: (1) เปิด Temporal Worker ที่ poll งานจาก Task Queue เดียวกับที่
// cmd/bop-worker ประกาศ workflow.Execute + workflow.Activities.RunStep ไว้ (2) สร้าง
// Temporal Schedule ให้ workflow ตัวอย่าง "check-example-com" รันทุก 1 นาที (แทนที่
// internal/scheduler เดิมที่ถูกลบไปแล้วทั้งหมด — Temporal Schedules ทำหน้าที่นี้แทนได้
// ครบตาม Open Question ข้อ 2 เพราะยังไม่มีกฎปฏิทินซับซ้อนอะไรที่ Temporal Schedules ไม่รองรับ)
package main

import (
	"context"
	"errors"
	"log"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	sdktemporal "go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"bop/internal/bot"
	"bop/internal/bots/externaljob"
	"bop/internal/bots/httpstatus"
	memlogger "bop/internal/logger/memory"
	"bop/internal/workflow"
)

// k8sJobDefaultNamespace คือ namespace ที่ K8s Job hybrid execution (ดู
// K8S-JOB-HYBRID-EXECUTION-TODO.md) จะสร้าง Job ถ้า workflow.Step.K8sJob.Namespace ไม่ได้
// ระบุมา — ตั้งใจแยกจาก namespace ที่ BOP เอง/Temporal worker รันอยู่ (blast radius,
// resource quota แยกกันชัดเจน ดู Open Question ข้อ 3 ของเอกสารนั้น) hardcode ไปก่อนตาม
// pattern เดียวกับค่าคงที่อื่นๆ ในโปรเจกต์นี้ (เช่น ":8080" ใน cmd/bop/main.go)
const k8sJobDefaultNamespace = "bop-jobs"

func main() {
	// ขั้นตอนที่ 1: ต่อ Temporal client เข้ากับ Temporal server (ดู docker-compose.yml
	// ที่ root ของโปรเจกต์ — ต้อง `docker compose up -d` ให้ server พร้อมก่อนรัน process นี้)
	// ที่อยู่ "localhost:7233" คือ gRPC endpoint default ของ Temporal server (แค่ hardcode
	// ไปก่อนเหมือนที่ cmd/bop hardcode addr ":8080" ไว้ ไม่เพิ่ม flag/env var จนกว่าจะมี
	// ความจำเป็นจริง)
	c, err := client.Dial(client.Options{
		HostPort:  "localhost:7233",
		Namespace: workflow.Namespace,
	})
	if err != nil {
		log.Fatalf("connect to temporal: %v", err)
	}
	defer c.Close()

	// ขั้นตอนที่ 2: เตรียม Activities — เป็น registry ของ bot ที่ workflow เรียกใช้ได้
	// (คนละ registry กับ worker.Pool ใน cmd/bop เพราะเป็นคนละ process กัน ดูหมายเหตุยาว
	// ที่ internal/workflow/activity.go เรื่อง logger แยก process) เพิ่ม bot ชนิดใหม่ต้อง
	// register ที่นี่ด้วยเสมอ ไม่งั้น workflow ที่อ้างถึง bot นั้นจะ fail ด้วย "unknown bot"
	appLogger := memlogger.New()
	activities := workflow.NewActivities(appLogger)
	activities.Register(httpstatus.New())
	activities.Register(externaljob.New())

	// ขั้นตอนที่ 3: สร้างและเริ่ม Temporal Worker — ตัวนี้คือ process ที่ poll งานจาก
	// Task Queue จริงๆ (RegisterWorkflow ผูก "ชื่อ workflow type" เข้ากับ workflow.Execute,
	// RegisterActivity ผูก "RunStep" ตามชื่อคงที่ที่ internal/workflow/temporal.go กำหนดไว้
	// เข้ากับ method จริงของ activities — ต้องตั้งชื่อผ่าน activity.RegisterOptions ให้ตรง
	// กับ string "RunStep" เป๊ะ เพราะ workflow.ExecuteActivity เรียกด้วยชื่อ ไม่ใช่ reference
	// ฟังก์ชันตรงๆ ดูเหตุผลที่ temporal.go)
	w := worker.New(c, workflow.TaskQueueName, worker.Options{})
	w.RegisterWorkflow(workflow.Execute)
	w.RegisterActivityWithOptions(activities.RunStep, activity.RegisterOptions{Name: workflow.ActivityName})

	// ขั้นตอนที่ 3.5: register K8s Job hybrid execution (workflow.Step.K8sJob) แบบ
	// "best-effort" — ต่างจาก activities.Register ด้านบนที่ fail แล้ว log.Fatalf ได้เลย
	// (เพราะ Temporal connection คือ dependency ที่ต้องมีเสมอ) แต่ rest.InClusterConfig()
	// ใช้ไม่ได้เลยตอนรัน cmd/bop-worker บนเครื่อง dev ธรรมดา (ยังไม่ได้ containerize เข้า
	// K8s ตาม Phase 2.1 — ดู K8S-JOB-HYBRID-EXECUTION-TODO.md) เพราะฟังก์ชันนี้ใช้ได้เฉพาะ
	// ตอนโปรเซสรันอยู่ "ข้างใน" Pod ของ K8s จริงๆ เท่านั้น (อ่าน ServiceAccount token ที่
	// K8s mount ให้อัตโนมัติ) — ถ้า error แค่ log คำเตือนแล้วข้ามไป ปล่อยให้ worker เริ่ม
	// ทำงานต่อได้ปกติ (bot local + remote task queue ยังใช้งานได้เต็มที่ แค่ step ที่มี
	// K8sJob จะ fail ตอนถูกเรียกจริงเพราะไม่มี activity ไหน register ไว้รับ) แทนที่จะ
	// log.Fatalf ปิด process ทั้งตัวเพราะฟีเจอร์ที่ยังไม่มี cluster ให้ใช้เลย
	if cfg, err := rest.InClusterConfig(); err != nil {
		log.Printf("k8s in-cluster config unavailable, skipping K8s Job hybrid execution: %v", err)
	} else if clientset, err := kubernetes.NewForConfig(cfg); err != nil {
		log.Printf("k8s clientset init failed, skipping K8s Job hybrid execution: %v", err)
	} else {
		k8sActivities := workflow.NewK8sJobActivities(clientset, k8sJobDefaultNamespace, appLogger)
		w.RegisterActivityWithOptions(k8sActivities.RunKubernetesJob, activity.RegisterOptions{Name: workflow.K8sJobActivityName})
		log.Printf("k8s job hybrid execution enabled (default namespace %q)", k8sJobDefaultNamespace)
	}

	// ขั้นตอนที่ 4: สร้าง Temporal Schedule ให้ workflow ตัวอย่างรันทุก 1 นาที (ค่าเดียวกับ
	// scheduler.Interval{Every: time.Minute} ในเวอร์ชันก่อน Temporal) — เรียกตอน startup
	// ทุกครั้ง แต่ idempotent: ถ้า Schedule ID นี้มีอยู่แล้ว (เช่น worker ถูก restart)
	// Temporal จะคืน ErrScheduleAlreadyRunning ซึ่งไม่ใช่ปัญหา ข้ามไปเฉยๆ ได้เลย
	ensureExampleSchedule(context.Background(), c)

	// ขั้นตอนที่ 5: รัน Worker ค้างอยู่ (poll งานเรื่อยๆ) จนกว่าจะได้รับ interrupt signal
	// (Ctrl+C) — worker.InterruptCh() คือ helper ของ SDK เอง ดักจับ signal มาตรฐานให้แล้ว
	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatalf("worker stopped with error: %v", err)
	}
}

// ensureExampleSchedule สร้าง Temporal Schedule ชื่อ "check-example-com" ที่สั่ง
// workflow.Execute รันด้วย workflow.Workflow ตัวอย่าง (เช็คสถานะ example.com) ทุก 1 นาที
// — ตัวอย่างเดียวกับที่ cmd/bop/main.go เวอร์ชันก่อน Temporal เคย sched.Register ไว้ตรงๆ
// ในโค้ด ตอนนี้ย้ายมาไว้ที่นี่เพราะ cmd/bop-worker คือ process ที่ "เป็นเจ้าของ" workflow
// ที่รันอัตโนมัติตามตาราง ส่วน cmd/bop มีหน้าที่แค่ตอบ HTTP request เท่านั้น
//
// หมายเหตุ (หลังเพิ่ม schedule CRUD API ใน internal/api/http/schedule_handler.go):
// schedule ตัวนี้สร้างตรงผ่าน Temporal SDK ในนี้ ไม่ได้ผ่าน POST /schedules เลย จึงไม่มี
// entry คู่กันใน schedule.Repository ของ cmd/bop (คนละ process กัน ดู
// internal/schedule/schedule.go) — GET /schedules/check-example-com-schedule จะเห็น
// spec/สถานะถูกต้องปกติ (มาจาก Temporal ตรงๆ) แต่ field "steps" จะว่างเปล่า เพราะ BOP
// ไม่มี bookkeeping ของ workflow definition ตัวนี้เก็บไว้เลย — ไม่ใช่ bug เป็นผลตามมาที่
// รู้อยู่แล้วจากการมี 2 ช่องทางสร้าง schedule (โค้ดตรงนี้ + HTTP API)
func ensureExampleSchedule(ctx context.Context, c client.Client) {
	wf := workflow.Workflow{
		Name: "check-example-com",
		Steps: []workflow.Step{
			{BotName: "http-status-checker", Config: bot.Config{"url": "https://example.com"}},
		},
	}

	_, err := c.ScheduleClient().Create(ctx, client.ScheduleOptions{
		ID: "check-example-com-schedule",
		Spec: client.ScheduleSpec{
			Intervals: []client.ScheduleIntervalSpec{{Every: time.Minute}},
		},
		Action: &client.ScheduleWorkflowAction{
			ID:        "check-example-com",
			Workflow:  workflow.Execute,
			Args:      []interface{}{wf},
			TaskQueue: workflow.TaskQueueName,
		},
	})
	if err != nil {
		if errors.Is(err, sdktemporal.ErrScheduleAlreadyRunning) {
			log.Println("schedule check-example-com-schedule already exists, skipping creation")
			return
		}
		log.Fatalf("create schedule: %v", err)
	}
	log.Println("created schedule check-example-com-schedule")
}
