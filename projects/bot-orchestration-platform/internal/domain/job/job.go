// Package job กำหนด entity หลักของระบบคือ "Job" (หนึ่งครั้งที่มีการสั่งรัน bot) และ
// interface สำหรับเก็บ/ค้นหาข้อมูล Job เหล่านี้ — Job คือ record ที่หน้า job history
// table และหน้า live monitor ของ dashboard (roadmap Phase 1.4/1.5) เอาไปแสดงผล
package job

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"time"

	"bop/internal/domain/bot"
)

// Status คือสถานะของ job ในแต่ละช่วงเวลา เปลี่ยนแปลงตามลำดับ:
// pending (รอคิว) -> running (กำลังรัน) -> succeeded/failed/cancelled (จบแล้ว)
type Status string

// สถานะทั้งหมดของ job (ดูลำดับการเปลี่ยนที่ Status)
const (
	StatusPending   Status = "pending"   // สร้าง job แล้วแต่ยังไม่ได้เริ่มรัน (รอ worker ว่าง)
	StatusRunning   Status = "running"   // worker กำลังรัน bot อยู่ตอนนี้
	StatusSucceeded Status = "succeeded" // bot ทำงานจบโดยไม่มี error
	StatusFailed    Status = "failed"    // bot ทำงานจบแต่มี error เกิดขึ้น
	StatusCancelled Status = "cancelled" // ถูกสั่งยกเลิกกลางทางก่อนที่จะทำงานเสร็จ
)

// Job คือ record หนึ่งรายการของการรัน bot หนึ่งครั้ง เก็บทั้งข้อมูล config ที่ใช้รัน,
// สถานะปัจจุบัน, และผลลัพธ์ (ถ้ามี) — struct นี้ถูกแก้ไข field ต่างๆ ระหว่างที่ job
// กำลังทำงานอยู่ (ดู internal/adapter/worker) จึงต้องระวังเรื่อง concurrent access เป็นพิเศษ
// (ดูคำอธิบายละเอียดใน internal/repository/memory ว่าทำไมถึงต้อง copy struct นี้
// ทุกครั้งที่ส่งเข้า-ออกจาก repository)
// json tag ทุก field เป็น snake_case ตั้งใจให้ตรงกับ endpoint กลุ่ม /workflow-runs
// (ดู internal/adapter/http/workflow_run_common.go) เพื่อให้ JSON ที่ dashboard (projects/bop-dashboard/)
// ได้รับมี casing สม่ำเสมอกันทั้ง API ไม่ใช่ผสมกันระหว่าง PascalCase (ค่า default ของ Go
// ตอนไม่มี tag) กับ snake_case แบบที่เคยเป็นมาก่อนหน้านี้
type Job struct {
	ID      string            `json:"id"`       // รหัสเฉพาะของ job นี้ (สุ่มสร้างตอนสร้าง job ใหม่)
	BotName string            `json:"bot_name"` // ชื่อ bot ที่ใช้รัน job นี้ (ตรงกับ Bot.Name())
	Config  map[string]string `json:"config"`   // ค่า config ที่ user กรอกมาตอนสร้าง job
	// TimeoutSeconds คือ timeout เฉพาะของ job นี้ (หน่วยวินาที) — ถ้าเป็น 0 (ไม่ได้ระบุมา)
	// worker.Pool จะใช้ timeout กลางของทั้งระบบแทน (ค่าที่ตั้งไว้ตอนสร้าง worker.NewPool)
	// เพิ่ม field นี้เพราะ bot บางตัว (เช่น externaljob ที่ shell out ไปเรียก binary
	// ภายนอกที่ใช้เวลาทำงานไม่แน่นอน) ต้องการเวลามากกว่า/น้อยกว่า timeout กลาง 30 วินาที
	// ที่ bot อื่นอย่าง http-status-checker ใช้พอดีอยู่แล้ว — ให้ผู้เรียก API ระบุต่อ job
	// แต่ละครั้งได้แทนที่จะบังคับใช้ค่าเดียวกันทุก bot ในระบบ (ดู EXTERNAL-JOB-BOT-TODO.md)
	TimeoutSeconds int        `json:"timeout_seconds,omitempty"`
	Status         Status     `json:"status"`               // สถานะปัจจุบันของ job (ดู const ด้านบน)
	Summary        string     `json:"summary,omitempty"`    // ข้อความสรุปผลลัพธ์ (มีค่าก็ต่อเมื่อ Status = succeeded)
	Error          string     `json:"error,omitempty"`      // ข้อความ error (มีค่าก็ต่อเมื่อ Status = failed หรือ cancelled)
	CreatedAt      time.Time  `json:"created_at"`           // เวลาที่สร้าง job นี้ (ก่อนเริ่มรันจริง)
	StartedAt      *time.Time `json:"started_at,omitempty"` // เวลาที่ worker เริ่มรันจริง (เป็น pointer เพราะยังไม่มีค่าตอน pending)
	EndedAt        *time.Time `json:"ended_at,omitempty"`   // เวลาที่ job จบ ไม่ว่าจะสำเร็จ/ล้มเหลว/ถูกยกเลิก (เป็น pointer ด้วยเหตุผลเดียวกัน)
}

// ErrInvalidTransition คือ error ที่ method เปลี่ยนสถานะด้านล่างคืน เมื่อสถานะปัจจุบัน
// ของ job ไม่อนุญาตให้เปลี่ยนไปเป็นสถานะที่ขอ (เช่นสั่ง Start job ที่จบไปแล้ว)
var ErrInvalidTransition = errors.New("job: invalid status transition")

// Clone คืนสำเนาแบบลึกของ job: Config (map) และเวลาที่เป็น pointer ถูกคัดลอกใหม่ทั้งหมด
// ใช้ทุกครั้งที่ job ข้ามขอบเขตระหว่าง goroutine หรือเข้า-ออก repository เพราะ `cp := *j`
// คัดลอกแค่ตัว map header ผู้ถือสำเนาทั้งสองฝั่งจะยังแก้ Config ก้อนเดียวกันอยู่
func (j *Job) Clone() *Job {
	cp := *j
	cp.Config = maps.Clone(j.Config)
	if j.StartedAt != nil {
		t := *j.StartedAt
		cp.StartedAt = &t
	}
	if j.EndedAt != nil {
		t := *j.EndedAt
		cp.EndedAt = &t
	}
	return &cp
}

// IsFinal บอกว่าสถานะนี้คือสถานะจบแล้ว (succeeded/failed/cancelled) ที่จะไม่เปลี่ยนอีก
func (s Status) IsFinal() bool {
	return s == StatusSucceeded || s == StatusFailed || s == StatusCancelled
}

// Start เปลี่ยน job จาก pending เป็น running และบันทึกเวลาเริ่ม — ทำได้เฉพาะ job ที่ยัง pending
// method เปลี่ยนสถานะทุกตัวรับเวลาเข้ามาแทนการเรียก time.Now() เอง เพื่อให้ test กำหนดเวลาได้
func (j *Job) Start(now time.Time) error {
	if j.Status != StatusPending {
		return fmt.Errorf("%w: start from %q", ErrInvalidTransition, j.Status)
	}
	j.Status = StatusRunning
	j.StartedAt = &now
	return nil
}

// Succeed จบ job ที่กำลัง running ด้วยผลสำเร็จ พร้อมข้อความสรุปผล
func (j *Job) Succeed(summary string, now time.Time) error {
	if err := j.finish(StatusSucceeded, now); err != nil {
		return err
	}
	j.Summary = summary
	return nil
}

// Fail จบ job ที่กำลัง running ด้วยความล้มเหลว พร้อมข้อความ error
func (j *Job) Fail(reason string, now time.Time) error {
	if err := j.finish(StatusFailed, now); err != nil {
		return err
	}
	j.Error = reason
	return nil
}

// Cancel จบ job ที่กำลัง running เพราะถูกสั่งยกเลิกกลางทาง พร้อมเหตุผล
func (j *Job) Cancel(reason string, now time.Time) error {
	if err := j.finish(StatusCancelled, now); err != nil {
		return err
	}
	j.Error = reason
	return nil
}

// finish คือกฎร่วมของการจบ job ทุกแบบ: ต้องอยู่ในสถานะ running เท่านั้น ถ้าผิดกฎจะ
// คืน error โดยไม่แตะ field ใดๆ ของ job เลย
func (j *Job) finish(to Status, now time.Time) error {
	if j.Status != StatusRunning {
		return fmt.Errorf("%w: %q to %q", ErrInvalidTransition, j.Status, to)
	}
	j.Status = to
	j.EndedAt = &now
	return nil
}

// Repository คือ interface สำหรับเก็บและค้นหา Job — มี 2 implementation: in-memory
// (internal/repository/memory ข้อมูลหายเมื่อ restart) และ PostgreSQL
// (internal/repository/postgres อยู่รอดข้าม restart) cmd/bop/main.go เลือกตาม
// BOP_DATABASE_URL — โค้ดส่วนอื่นอ้างอิงผ่าน interface นี้เท่านั้น จึงไม่รู้ว่าใช้ตัวไหน
type Repository interface {
	// Create สร้าง job ใหม่ในระบบ ต้อง error ถ้ามี ID นี้อยู่แล้ว
	Create(ctx context.Context, j *Job) error
	// Update บันทึกทับข้อมูล job ที่มีอยู่แล้วด้วยค่าล่าสุด (ใช้ตอนสถานะเปลี่ยน)
	Update(ctx context.Context, j *Job) error
	// Get ค้นหา job ตัวเดียวด้วย ID ต้อง error ถ้าไม่พบ
	Get(ctx context.Context, id string) (*Job, error)
	// List คืนรายการ job ทั้งหมดในระบบ
	List(ctx context.Context) ([]*Job, error)
	// FailInterrupted ตั้ง Job ที่ยัง pending/running ทุกตัวเป็น failed พร้อม error = reason และ
	// ended_at = now แล้วคืนจำนวนที่แก้ — เรียกตอน bop-api start เท่านั้น (Job ที่ค้างมาจาก process
	// ก่อนหน้าที่ตายกลางทาง ไม่มี goroutine ไหนเขียนสถานะสุดท้ายให้แล้ว) ถือว่า bop-api รันตัวเดียว
	// (ดู ARCHITECTURE.md) Job ที่จบแล้วไม่ถูกแตะ memory implementation คืน 0 เพราะข้อมูลหายตอน
	// restart อยู่แล้ว
	FailInterrupted(ctx context.Context, reason string, now time.Time) (int, error)
}

// Executor คือ port ของ "ตัวรัน job เดี่ยว" ที่ usecase ใช้ — ตอนนี้ implement โดย
// worker.Pool (internal/adapter/worker) ที่รัน bot ใน process เดียวกับ HTTP server
// usecase รู้จักแค่ interface นี้ ไม่รู้ว่าข้างหลังเป็น goroutine pool หรืออย่างอื่น
type Executor interface {
	// Submit ส่ง job เข้าคิวรันแบบไม่รอผล — error ถ้าไม่รู้จัก bot ตาม j.BotName
	Submit(j *Job) error
	// Cancel สั่งยกเลิก job ที่กำลังรันอยู่ — false ถ้า job นั้นไม่ได้กำลังรัน
	Cancel(jobID string) bool
	// List คืน bot ทั้งหมดที่ register ไว้ เรียงตามชื่อ
	List() []bot.Bot
}
