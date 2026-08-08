// Package workflow ให้ definition ของ Workflow ที่ BOP รองรับ: ลำดับของ Step (สั่งรัน
// bot ทีละตัวตามลำดับ, step ถัดไปเริ่มได้ก็ต่อเมื่อ step ก่อนหน้าสำเร็จ) — ตัว "engine"
// ที่รัน Workflow นี้จริงๆ คือ Temporal (ดู temporal.go: Execute คือ Temporal workflow
// function, activity.go: Activities.RunStep คือ Temporal Activity ที่เรียก bot.Run จริง)
//
// เดิม package นี้เขียน execution engine เองทั้งหมด (Runner.Trigger รันแบบ synchronous
// ผ่าน worker.Pool.SubmitAndWait, RunRepository เก็บผลแบบ in-memory) ก่อนย้ายมาใช้
// Temporal จริงจังตาม FULLSTACK-INFRA-ROADMAP.md Phase 2.5 — ดูเหตุผลและ Open Question
// ที่ตัดสินใจไว้ที่ TEMPORAL-MIGRATION-TODO.md ที่ root ของโปรเจกต์นี้ Step/Workflow
// (type ด้านล่าง) เป็นส่วนเดียวที่เก็บไว้ต่อจากเวอร์ชันเดิม ใช้เป็น "input schema" ให้
// Execute อ่านแทนที่จะออกแบบ type ใหม่ทั้งหมด (Open Question ข้อ 1 ในเอกสารนั้น)
package workflow

import "bop/internal/bot"

// Step คือขั้นตอนหนึ่งใน workflow — สั่งรัน bot ตัวหนึ่งด้วย config ที่กำหนด — json tag
// เป็น snake_case เพราะตอนนี้ type นี้ถูกใช้เป็น request/response body ตรงๆ ของ
// POST /schedules (ดู internal/api/http/schedule_handler.go) ไม่ใช่แค่ argument ภายใน
// ที่ส่งผ่าน Temporal payload เหมือนเดิมอย่างเดียวแล้ว
type Step struct {
	BotName string     `json:"bot_name"`
	Config  bot.Config `json:"config"`
	// TimeoutSeconds คือ StartToCloseTimeout เฉพาะของ step นี้ (หน่วยวินาที) — ถ้าเป็น 0
	// (ไม่ได้ระบุมา) Execute (temporal.go) จะใช้ defaultActivityTimeout กลางแทน เพิ่ม field
	// นี้ด้วยเหตุผลเดียวกับ job.Job.TimeoutSeconds: bot บางตัว (เช่น externaljob) ใช้เวลา
	// ทำงานนานกว่า 30 วินาทีที่เป็นค่าเริ่มต้นของทั้งระบบมาก ต้องกำหนดได้ต่อ step แทนที่จะ
	// บังคับใช้ค่าเดียวกันทุก step ในทุก workflow (ดู EXTERNAL-JOB-BOT-TODO.md)
	TimeoutSeconds int `json:"timeout_seconds,omitempty"`
	// TaskQueue คือชื่อ Temporal Task Queue ที่ Activity ของ step นี้จะถูกส่งไปให้รัน — ถ้า
	// เป็นค่าว่าง (ไม่ได้ระบุมา) ใช้ task queue เดียวกับตัว workflow เอง (TaskQueueName คงที่
	// "bop-workflows") ซึ่งมีแค่ cmd/bop-worker เป็นคน poll อยู่ ทำให้ Activities.RunStep
	// (activity.go) เป็นคนรับงานไปหา bot.Bot จาก local registry ตามปกติ (พฤติกรรมเดิมก่อน
	// เพิ่ม field นี้ ไม่มีอะไรเปลี่ยน)
	//
	// ถ้าระบุ TaskQueue เป็นชื่ออื่น: Temporal จะส่ง Activity call ไปยัง worker process
	// ที่ poll task queue ชื่อนั้นแทน — worker นั้นไม่จำเป็นต้องเป็น cmd/bop-worker เลยด้วยซ้ำ
	// (อยู่คนละ repo/binary/แม้แต่คนละภาษาก็ได้ ขอแค่ register activity function ภายใต้ชื่อ
	// "RunStep" ตรงกับ ActivityName ด้านล่างเป๊ะ และรับ/คืนค่าที่ serialize เข้ากันได้กับ
	// Step/string ผ่าน JSON default DataConverter ของ Temporal) — ใช้กรณี job/bot อยู่คนละ
	// repository และ deploy คนละจังหวะกับ BOP จริงๆ จนไม่คุ้มทำเป็น bot.Bot ภายใน BOP เอง
	// (ดู REMOTE-JOB-WORKER-TODO.md สำหรับ contract เต็มที่ remote worker ต้อง implement)
	TaskQueue string `json:"task_queue,omitempty"`
}

// Workflow คือลำดับของ Step ที่ต้องรันเรียงกันตามลำดับ (step แรกสุดในลิสต์รันก่อนเสมอ)
// — Name ถูกใช้เป็นส่วนหนึ่งของ Temporal Workflow ID ตอนสร้าง Schedule (ดู
// cmd/bop-worker/main.go, internal/api/http/schedule_handler.go) เพื่อให้เปิด Temporal
// UI แล้วรู้ทันทีว่า run ไหนเป็นของ workflow ที่ชื่ออะไร
type Workflow struct {
	Name  string `json:"name"`
	Steps []Step `json:"steps"`
}

// StepResult คือผลลัพธ์ของ step หนึ่งที่รันไปแล้ว เก็บไว้ใน slice ที่ Execute คืนกลับและ
// เปิดให้ query ผ่าน Temporal Query handler "steps" (ดู temporal.go) — ต่างจากเวอร์ชันก่อน
// Temporal ตรงที่ไม่มี JobID ผูกกับ job.Repository อีกต่อไป เพราะ job.Repository ตอนนี้
// เก็บไว้เฉพาะ job เดี่ยวๆ ที่สั่งตรงผ่าน POST /bots/{name}/jobs เท่านั้น (Open Question
// ข้อ 3 ใน TEMPORAL-MIGRATION-TODO.md — ดูหมายเหตุเรื่อง log ID ใน activity.go ประกอบ)
type StepResult struct {
	StepIndex int    `json:"step_index"`
	BotName   string `json:"bot_name"`
	Summary   string `json:"summary,omitempty"` // มีค่าก็ต่อเมื่อ step นี้สำเร็จ
	Error     string `json:"error,omitempty"`   // มีค่าก็ต่อเมื่อ step นี้ล้มเหลว
}
