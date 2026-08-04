// Package job กำหนด entity หลักของระบบคือ "Job" (หนึ่งครั้งที่มีการสั่งรัน bot) และ
// interface สำหรับเก็บ/ค้นหาข้อมูล Job เหล่านี้ — Job คือ record ที่หน้า job history
// table และหน้า live monitor ของ dashboard (roadmap Phase 1.4/1.5) เอาไปแสดงผล
package job

import (
	"context"
	"time"
)

// Status คือสถานะของ job ในแต่ละช่วงเวลา เปลี่ยนแปลงตามลำดับ:
// pending (รอคิว) -> running (กำลังรัน) -> succeeded/failed/cancelled (จบแล้ว)
type Status string

const (
	StatusPending   Status = "pending"   // สร้าง job แล้วแต่ยังไม่ได้เริ่มรัน (รอ worker ว่าง)
	StatusRunning   Status = "running"   // worker กำลังรัน bot อยู่ตอนนี้
	StatusSucceeded Status = "succeeded" // bot ทำงานจบโดยไม่มี error
	StatusFailed    Status = "failed"    // bot ทำงานจบแต่มี error เกิดขึ้น
	StatusCancelled Status = "cancelled" // ถูกสั่งยกเลิกกลางทางก่อนที่จะทำงานเสร็จ
)

// Job คือ record หนึ่งรายการของการรัน bot หนึ่งครั้ง เก็บทั้งข้อมูล config ที่ใช้รัน,
// สถานะปัจจุบัน, และผลลัพธ์ (ถ้ามี) — struct นี้ถูกแก้ไข field ต่างๆ ระหว่างที่ job
// กำลังทำงานอยู่ (ดู internal/worker) จึงต้องระวังเรื่อง concurrent access เป็นพิเศษ
// (ดูคำอธิบายละเอียดใน internal/repository/memory ว่าทำไมถึงต้อง copy struct นี้
// ทุกครั้งที่ส่งเข้า-ออกจาก repository)
type Job struct {
	ID        string            // รหัสเฉพาะของ job นี้ (สุ่มสร้างตอนสร้าง job ใหม่)
	BotName   string            // ชื่อ bot ที่ใช้รัน job นี้ (ตรงกับ Bot.Name())
	Config    map[string]string // ค่า config ที่ user กรอกมาตอนสร้าง job
	Status    Status            // สถานะปัจจุบันของ job (ดู const ด้านบน)
	Summary   string            // ข้อความสรุปผลลัพธ์ (มีค่าก็ต่อเมื่อ Status = succeeded)
	Error     string            // ข้อความ error (มีค่าก็ต่อเมื่อ Status = failed หรือ cancelled)
	CreatedAt time.Time         // เวลาที่สร้าง job นี้ (ก่อนเริ่มรันจริง)
	StartedAt *time.Time        // เวลาที่ worker เริ่มรันจริง (เป็น pointer เพราะยังไม่มีค่าตอน pending)
	EndedAt   *time.Time        // เวลาที่ job จบ ไม่ว่าจะสำเร็จ/ล้มเหลว/ถูกยกเลิก (เป็น pointer ด้วยเหตุผลเดียวกัน)
}

// Repository คือ interface สำหรับเก็บและค้นหา Job — ตอนนี้มี implementation เดียวคือ
// in-memory (internal/repository/memory) ซึ่งข้อมูลจะหายไปทุกครั้งที่ restart โปรแกรม
// เมื่อไหร่ที่ต้องการให้ job history อยู่ถาวรข้าม restart (ดู roadmap Phase 0.4)
// ก็แค่เขียน implementation ใหม่ที่ implement interface นี้ (เช่นต่อ PostgreSQL จริง)
// แล้วสลับตอนประกอบ dependency ใน cmd/bop/main.go — โค้ดส่วนอื่นทั้งหมดไม่ต้องแก้เลย
// เพราะทุกที่อ้างอิงผ่าน interface นี้ ไม่เคยอ้างอิง concrete implementation ตรงๆ
type Repository interface {
	// Create สร้าง job ใหม่ในระบบ ต้อง error ถ้ามี ID นี้อยู่แล้ว
	Create(ctx context.Context, j *Job) error
	// Update บันทึกทับข้อมูล job ที่มีอยู่แล้วด้วยค่าล่าสุด (ใช้ตอนสถานะเปลี่ยน)
	Update(ctx context.Context, j *Job) error
	// Get ค้นหา job ตัวเดียวด้วย ID ต้อง error ถ้าไม่พบ
	Get(ctx context.Context, id string) (*Job, error)
	// List คืนรายการ job ทั้งหมดในระบบ
	List(ctx context.Context) ([]*Job, error)
}
