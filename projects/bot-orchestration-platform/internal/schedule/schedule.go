// Package schedule เก็บ "definition" ของ schedule แต่ละตัวที่สร้างผ่าน BOP HTTP API
// (ดู internal/api/http/schedule_handler.go) — คือ workflow name + steps ที่ schedule
// นั้นสั่งรันตามเวลา
//
// หมายเหตุสำคัญเรื่องขอบเขตความรับผิดชอบ (อ่านก่อนแก้ไข อย่าสับสน): package นี้ "ไม่ใช่"
// แหล่งข้อมูลจริง (source of truth) ของสถานะ schedule เลย — Temporal เองเป็นคนเก็บและ
// ควบคุมทุกอย่างที่เกี่ยวกับ "เวลา/สถานะ" จริงๆ (next run time, paused หรือไม่, ประวัติ
// การรันย้อนหลัง) ผ่าน client.ScheduleClient ต่างหาก (ดู
// internal/api/http/schedule_handler.go: buildScheduleSpec, handleGetSchedule ฯลฯ)
// สิ่งเดียวที่ package นี้เก็บคือ "workflow นี้ทำอะไรบ้าง" (ชื่อ + steps) ซึ่ง Temporal
// เก็บไว้เหมือนกันแต่เข้ารหัสเป็น payload ที่ decode กลับมายากกว่าเก็บเองตรงๆ (ดูเหตุผล
// เต็มๆ ที่ comment ของ Repository ด้านล่าง)
package schedule

import (
	"context"
	"time"

	"bop/internal/workflow"
)

// Definition คือ workflow definition ของ schedule หนึ่งตัว — ID ต้องตรงกับ Temporal
// Schedule ID เป๊ะ (ตัวเดียวกับที่ใช้ตอน client.ScheduleClient().Create) เพื่อให้ join
// ข้อมูลจากสองแหล่ง (Temporal + Definition นี้) เข้าด้วยกันได้ตอนตอบ HTTP response
type Definition struct {
	ID           string
	WorkflowName string
	Steps        []workflow.Step
	CreatedAt    time.Time
}

// Repository คือ interface สำหรับเก็บ/ค้นหา Definition — ใช้หลักการเดียวกับ
// job.Repository ทุกประการ (ทุก implementation ต้อง copy ค่าเข้า-ออกเสมอ) เหตุผลที่มี
// package นี้แยกต่างหากแทนที่จะ decode เอาจาก Temporal ตรงๆ ทุกครั้งที่ query: Temporal
// เก็บ workflow args เป็น payload ที่ต้องผ่าน DataConverter ถึงจะอ่านกลับเป็น
// workflow.Workflow ได้ (internal representation ของ SDK ที่ไม่ได้ประกาศเป็น public
// contract ชัดเจนพอจะ depend ได้อย่างมั่นใจในระยะยาว) เก็บเองแบบตรงไปตรงมาแบบนี้
// ปลอดภัยกว่าและ debug ง่ายกว่ามาก แลกกับข้อจำกัดที่ต้องรู้: **ข้อมูลใน repository นี้
// เป็น in-memory ล้วนๆ หายทุกครั้งที่ restart cmd/bop** (เหมือนกับ job.Repository เดิม
// ก่อนต่อ PostgreSQL จริงใน roadmap Phase 0.4) ส่วน schedule ใน Temporal เองยังอยู่ครบ
// ไม่หายไปไหน (Temporal persist เองฝั่ง server) แค่ GET /schedules/{id} จะไม่มี steps
// ให้ดูจนกว่าจะสร้าง definition ใหม่ผ่าน API อีกครั้ง (หรือย้ายไปใช้ PostgreSQL
// implementation ในอนาคต)
type Repository interface {
	Create(ctx context.Context, d *Definition) error
	Update(ctx context.Context, d *Definition) error
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (*Definition, error)
	List(ctx context.Context) ([]*Definition, error)
}
