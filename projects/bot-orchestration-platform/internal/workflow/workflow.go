// Package workflow ให้ความสามารถรัน bot หลายตัวเรียงลำดับกันเป็น "workflow" เดียว
// (step ที่ 2 เริ่มได้ก็ต่อเมื่อ step ที่ 1 สำเร็จแล้วเท่านั้น) — เป็น "เวอร์ชันเบา" ก่อนที่
// จะย้ายไปใช้ Temporal จริงจังตาม FULLSTACK-INFRA-ROADMAP.md Phase 2.5
//
// ข้อจำกัดที่ควรรู้ (เทียบกับ Temporal ใน books/13-temporal/): เวอร์ชันนี้ไม่มี durable
// execution — ถ้าโปรแกรมถูก restart กลางทางที่ workflow กำลังรันอยู่ ความคืบหน้าทั้งหมด
// จะหายไปทันที (เพราะ RunRepository เป็น in-memory และ Runner.Trigger ทำงานอยู่ใน
// goroutine เดียวที่ไม่มีการบันทึก "จุดที่ทำถึง" แบบที่ Temporal ทำผ่าน event history)
// ก็ไม่มี retry policy ต่อ step หรือ compensation logic อัตโนมัติเหมือน Temporal ด้วย —
// ความเจ็บปวดจากข้อจำกัดพวกนี้แหละคือเหตุผลที่ Temporal มีค่าในโลกจริง เก็บไว้เป็นบทเรียน
// เปรียบเทียบตอนย้ายไป Temporal จริงใน Phase 2.5
package workflow

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"bop/internal/bot"
	"bop/internal/job"
)

// Step คือขั้นตอนหนึ่งใน workflow — สั่งรัน bot ตัวหนึ่งด้วย config ที่กำหนด
type Step struct {
	BotName string
	Config  bot.Config
}

// Workflow คือลำดับของ Step ที่ต้องรันเรียงกันตามลำดับ (step แรกสุดในลิสต์รันก่อนเสมอ)
type Workflow struct {
	Name  string
	Steps []Step
}

// RunStatus คือสถานะของการรัน workflow หนึ่งรอบ (หนึ่ง Run)
type RunStatus string

const (
	RunPending   RunStatus = "pending"   // สร้าง Run แล้วแต่ยังไม่เริ่ม step แรก
	RunRunning   RunStatus = "running"   // กำลังรัน step ใดก็ตามอยู่
	RunSucceeded RunStatus = "succeeded" // ทุก step ผ่านหมดแล้ว
	RunFailed    RunStatus = "failed"    // มี step ใด step หนึ่งล้มเหลว จึงหยุดทั้ง workflow
)

// StepResult คือผลลัพธ์ของ step หนึ่งที่รันไปแล้ว เก็บไว้ใน Run.Steps ตามลำดับที่รันจริง
type StepResult struct {
	JobID  string     // ID ของ job.Job ที่ step นี้สร้างขึ้นมา (ไปดูรายละเอียด/log เต็มๆ ได้ที่ job นั้นผ่าน GET /jobs/{id})
	Status job.Status // สถานะสุดท้ายของ step นี้
}

// Run คือ record หนึ่งรอบของการรัน Workflow หนึ่งครั้ง (เหมือน job.Job แต่เป็นระดับ
// workflow ทั้งก้อน ไม่ใช่ bot เดี่ยวๆ) — เก็บผลลัพธ์ทีละ step ที่รันไปแล้วเรื่อยๆ ระหว่าง
// ที่ workflow กำลังทำงาน ทำให้ดูความคืบหน้าได้แม้ workflow ยังไม่จบ
type Run struct {
	ID           string
	WorkflowName string
	Status       RunStatus
	Steps        []StepResult // เพิ่มเข้ามาทีละตัวหลัง step นั้นๆ รันจบ ไม่ใช่ใส่มาครบตั้งแต่แรก
	Error        string       // มีค่าก็ต่อเมื่อ Status = failed (บอกว่า step ไหนล้มเหลวและทำไม)
	CreatedAt    time.Time
	EndedAt      *time.Time
}

// RunRepository คือ interface สำหรับเก็บและค้นหา Run — ใช้หลักการเดียวกับ job.Repository
// ทุกประการ (ทุก implementation ต้อง copy ค่าเข้า-ออกเสมอ ห้ามปล่อย pointer ตัวเดิมออกไป
// เพราะ Runner แก้ไข Run ระหว่างที่ workflow กำลังทำงานอยู่เหมือนที่ worker.Pool แก้ไข
// job.Job — ดูคำอธิบายละเอียดที่ internal/repository/memory)
type RunRepository interface {
	Create(ctx context.Context, r *Run) error
	Update(ctx context.Context, r *Run) error
	Get(ctx context.Context, id string) (*Run, error)
	List(ctx context.Context) ([]*Run, error)
}

// newRunID สุ่มสร้างรหัสเฉพาะสำหรับ Run ใหม่ วิธีเดียวกับ newID ใน internal/api/http —
// คัดลอกมาไว้ตรงนี้แทนที่จะแชร์ function เดียวกัน เพราะเป็นโค้ดสั้นๆ ไม่กี่บรรทัด ไม่คุ้มที่
// จะสร้าง package ใหม่หรือผูก dependency ข้าม package แค่เพื่อฟังก์ชันเดียว
func newRunID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
