// Package logger กำหนดวิธีที่ระบบเก็บและส่งต่อ log message ของแต่ละ job — design
// (เก็บถาวรก่อนเสมอ แล้วค่อยกระจายให้คนที่กำลังดูอยู่แบบ real-time) ถูกกำหนดตายตัว
// ไว้ที่ interface นี้ ส่วนที่เปลี่ยนได้คือ "จะเก็บที่ไหน" เท่านั้น: ตอนนี้เก็บใน memory
// ก่อน, พอ dashboard ต้องการดู log แบบ real-time จริงจัง (roadmap Phase 1.5) จะสลับไป
// เก็บใน PostgreSQL, และสุดท้ายพอ log มีปริมาณเยอะขึ้นจนต้องการค้นหาแบบ full-text
// (roadmap Phase 2.2) จะสลับไปเก็บที่ OpenSearch ผ่าน Fluent Bit — โค้ดที่เรียกใช้
// interface นี้ (worker pool, API handler) ไม่ต้องแก้อะไรเลยไม่ว่าจะสลับ backend กี่รอบ
package logger

import (
	"context"
	"time"
)

// Level คือระดับความสำคัญของ log แต่ละบรรทัด
type Level string

const (
	LevelInfo  Level = "info"  // ข้อมูลทั่วไป (เช่น "เริ่มทำงานแล้ว", "กำลังโหลดหน้า...")
	LevelWarn  Level = "warn"  // สิ่งผิดปกติแต่ยังไม่ถึงกับทำให้ job ล้มเหลว
	LevelError Level = "error" // ข้อผิดพลาดที่ทำให้ job ทำงานต่อไม่ได้
)

// Line คือ log หนึ่งบรรทัดที่ถูกเก็บ/ส่งต่อ ผูกกับ job หนึ่งรายการเสมอผ่าน JobID
type Line struct {
	JobID   string    // job ที่ log บรรทัดนี้เป็นของ
	Time    time.Time // เวลาที่เกิด log บรรทัดนี้
	Level   Level     // ระดับความสำคัญ
	Message string    // ข้อความ log
}

// Logger คือ interface หลักที่ worker pool ใช้ส่ง log ออกมาระหว่างที่ job กำลังทำงาน
// ทุก implementation ต้องทำ 2 อย่างเสมอ: (1) เก็บ log บรรทัดนั้นไว้ถาวร (2) ส่งต่อให้
// ใครก็ตามที่กำลัง subscribe ฟัง log ของ job นั้นอยู่แบบ real-time ด้วย
type Logger interface {
	// Log บันทึก log หนึ่งบรรทัดของ job ที่ระบุ พร้อมส่งต่อให้ subscriber ที่กำลังฟังอยู่ทันที
	Log(ctx context.Context, jobID string, level Level, message string)

	// History คืน log ทั้งหมดที่เคยบันทึกไว้ของ job นั้น เรียงตามลำดับเวลา — ใช้ตอนโหลด
	// หน้า job detail ครั้งแรก (ก่อนจะเปิด live connection ต่อ) เพื่อไม่ให้พลาด log
	// ที่เกิดขึ้นไปแล้วก่อนหน้านี้
	History(ctx context.Context, jobID string) ([]Line, error)

	// Subscribe เปิดช่องทางรับ log บรรทัดใหม่ๆ ของ job ที่ระบุแบบ real-time — คืนค่าเป็น
	// channel ที่จะมีข้อมูลไหลเข้ามาเรื่อยๆ ทุกครั้งที่มีการเรียก Log ของ job นั้น และคืน
	// unsubscribe function ที่ต้องเรียกเมื่อเลิกฟัง (เช่น user ปิดหน้าเว็บ) เพื่อล้างทรัพยากร
	// ไม่ให้ค้างอยู่ใน memory ตลอดไป — จะถูกใช้จริงตอนเขียน WebSocket handler ใน
	// roadmap Phase 1.5
	Subscribe(jobID string) (lines <-chan Line, unsubscribe func())
}
