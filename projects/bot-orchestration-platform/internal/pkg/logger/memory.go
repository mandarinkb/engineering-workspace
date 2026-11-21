// Package logger คือ implementation ของ logger.Logger (internal/domain/logger) แบบเก็บ log ไว้ใน memory ล้วนๆ
// เหมาะกับช่วง dev เท่านั้น (log จะหายหมดถ้า restart โปรแกรม) — แต่ก็ implement ครบ
// ทั้งสองหน้าที่ตามที่ logger.Logger กำหนดไว้แล้วคือ (1) เก็บถาวร (2) กระจายแบบ real-time
package logger

import (
	"context"
	"sync"
	"time"

	"bop/internal/domain/logger"
)

// Logger เก็บ log แยกเป็น 2 ส่วน:
//   - logs: เก็บ log ทั้งหมดของแต่ละ job แบบ append-only list (ไว้ตอบ History)
//   - subs: เก็บรายชื่อ channel ของทุกคนที่กำลัง "ฟัง" log ของแต่ละ job อยู่แบบ real-time
//     (ไว้ตอบ Subscribe และกระจายข้อความใหม่ให้ทุกคนที่ฟังอยู่พร้อมกัน)
//
// และ log ของ Step แยกอีกส่วน (stepLogs) — ใน memory อ่านได้เฉพาะ process ที่เขียนเท่านั้น
// bop-worker ที่ไม่ได้ตั้ง BOP_DATABASE_URL จึงเห็น log ของ Step แค่ใน process ตัวเอง
type Logger struct {
	mu       sync.Mutex                    // ป้องกัน logs, subs และ stepLogs ด้านล่างจากการเข้าถึงพร้อมกัน
	logs     map[string][]logger.Line      // job ID -> log ทั้งหมดของ job นั้น (เรียงตามเวลา)
	subs     map[string][]chan logger.Line // job ID -> รายการ channel ของคนที่กำลังฟัง log ของ job นั้นแบบสด
	stepLogs map[stepKey][]logger.StepLine // Step -> log ทุก Attempt ของ Step นั้น (เรียงตามเวลา)
}

// stepKey คือ Step หนึ่งใน Workflow Run (ทุก Attempt ใช้ key เดียวกัน)
type stepKey struct {
	workflowID string
	stepIndex  int
}

var (
	_ logger.Logger     = (*Logger)(nil)
	_ logger.StepLogger = (*Logger)(nil)
)

// New สร้าง logger เปล่าๆ พร้อมใช้งาน
func New() *Logger {
	return &Logger{
		logs:     make(map[string][]logger.Line),
		subs:     make(map[string][]chan logger.Line),
		stepLogs: make(map[stepKey][]logger.StepLine),
	}
}

// LogStep บันทึก log หนึ่งบรรทัดของ Attempt หนึ่งของ Step
func (l *Logger) LogStep(_ context.Context, owner logger.StepAttempt, level logger.Level, message string) {
	line := logger.StepLine{
		WorkflowID: owner.WorkflowID,
		StepIndex:  owner.StepIndex,
		Attempt:    owner.Attempt,
		Time:       time.Now(),
		Level:      level,
		Message:    message,
	}
	key := stepKey{workflowID: owner.WorkflowID, stepIndex: owner.StepIndex}

	l.mu.Lock()
	defer l.mu.Unlock()
	l.stepLogs[key] = append(l.stepLogs[key], line)
}

// StepHistory คืน log ทุก Attempt ของ Step นั้น (copy ออกมาก่อนคืนเหมือน History)
func (l *Logger) StepHistory(_ context.Context, workflowID string, stepIndex int) ([]logger.StepLine, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	lines := l.stepLogs[stepKey{workflowID: workflowID, stepIndex: stepIndex}]
	out := make([]logger.StepLine, len(lines))
	copy(out, lines)
	return out, nil
}

// Log บันทึก log หนึ่งบรรทัด แล้วส่งต่อให้ทุก subscriber ของ job นั้นทันที
// ขั้นตอนการทำงาน:
//  1. สร้าง Line ใหม่พร้อม timestamp ปัจจุบัน
//  2. lock แล้วเก็บ Line นี้ต่อท้าย slice log ของ job นั้น (สำหรับ History ในอนาคต)
//  3. ส่งข้อความไปยังทุก subscriber ของ job นั้น "ขณะที่ยังถือ lock อยู่" — ถ้า channel
//     ไหนเต็ม (subscriber อ่านช้าเกินไป) จะ "ทิ้ง" ข้อความนั้นไปเลยแทนที่จะรอ (ผ่าน
//     select + default) การส่งจึงไม่มีทางบล็อค และถือ lock ไว้แค่ช่วงสั้นๆ
//
// ต้องส่งภายใต้ lock เดียวกับที่ unsubscribe ใช้ปิด channel: เดิมเคยคัดลอกรายชื่อ subscriber
// ออกมาแล้ว unlock ก่อนส่ง ทำให้ unsubscribe แทรกเข้ามา close channel ได้ระหว่างนั้น แล้ว
// การส่งเข้า channel ที่ปิดแล้วจะ panic ทันที (select + default กันไม่ได้) ทั้ง process ล่ม
// (ดู TestLogger_UnsubscribeWhileLogging)
func (l *Logger) Log(_ context.Context, jobID string, level logger.Level, message string) {
	line := logger.Line{JobID: jobID, Time: time.Now(), Level: level, Message: message}

	l.mu.Lock()
	defer l.mu.Unlock()
	l.logs[jobID] = append(l.logs[jobID], line)
	for _, ch := range l.subs[jobID] {
		select {
		case ch <- line:
		default: // subscriber อ่านไม่ทัน: ทิ้งข้อความนี้ไปแทนที่จะบล็อครอ (ไม่ให้กระทบ bot ที่กำลังรัน)
		}
	}
}

// History คืน log ทั้งหมดที่เคยบันทึกไว้ของ job นั้น (copy ออกมาก่อนคืนค่า เพื่อไม่ให้
// ผู้เรียกไปแก้ไข slice ต้นฉบับที่เก็บไว้ภายในโดยไม่ตั้งใจ)
func (l *Logger) History(_ context.Context, jobID string) ([]logger.Line, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]logger.Line, len(l.logs[jobID]))
	copy(out, l.logs[jobID])
	return out, nil
}

// Subscribe เปิด channel ใหม่สำหรับรับ log สดของ job ที่ระบุ
// ขั้นตอนการทำงาน:
//  1. สร้าง channel ใหม่ที่มี buffer ขนาด 16 บรรทัด (กันไม่ให้ต้องบล็อคทันทีถ้ามีข้อความ
//     เข้ามาถี่ๆ ก่อนที่ผู้ฟังจะอ่านทัน)
//  2. เก็บ channel นี้ไว้ในรายการ subscriber ของ job นั้น เพื่อให้ Log() ข้างบนส่งข้อความมาถึง
//  3. คืน channel (ฝั่งอ่านอย่างเดียว) พร้อมฟังก์ชัน unsubscribe ที่ผู้เรียกต้องเรียกเมื่อ
//     เลิกฟัง (เช่น ปิดหน้าเว็บ) — unsubscribe จะลบ channel ออกจากรายการ subscriber
//     และปิด channel เพื่อคืนทรัพยากร ป้องกัน memory leak ถ้ามีคนเปิด-ปิดหน้าดู log
//     ซ้ำๆ จำนวนมาก
func (l *Logger) Subscribe(jobID string) (<-chan logger.Line, func()) {
	ch := make(chan logger.Line, 16)

	l.mu.Lock()
	l.subs[jobID] = append(l.subs[jobID], ch)
	l.mu.Unlock()

	unsubscribe := func() {
		l.mu.Lock()
		defer l.mu.Unlock()
		subs := l.subs[jobID]
		for i, c := range subs {
			if c == ch {
				l.subs[jobID] = append(subs[:i], subs[i+1:]...) // ลบ channel นี้ออกจาก slice
				close(ch)
				break
			}
		}
	}
	return ch, unsubscribe
}
