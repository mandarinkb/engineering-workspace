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

// Step คือขั้นตอนหนึ่งใน workflow — สั่งรัน bot ตัวหนึ่งด้วย config ที่กำหนด
type Step struct {
	BotName string
	Config  bot.Config
}

// Workflow คือลำดับของ Step ที่ต้องรันเรียงกันตามลำดับ (step แรกสุดในลิสต์รันก่อนเสมอ)
// — Name ถูกใช้เป็นส่วนหนึ่งของ Temporal Workflow ID ตอนสร้าง Schedule (ดู
// cmd/bop-worker/main.go) เพื่อให้เปิด Temporal UI แล้วรู้ทันทีว่า run ไหนเป็นของ
// workflow ที่ชื่ออะไร
type Workflow struct {
	Name  string
	Steps []Step
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
