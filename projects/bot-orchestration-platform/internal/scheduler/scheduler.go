// Package scheduler คอยดูว่าถึงเวลาที่ workflow ไหนควรถูกสั่งรันอัตโนมัติหรือยัง แล้ว
// สั่ง workflow.Runner ให้ trigger ให้ — เทียบเท่ากับ "cron" ของระบบนี้ แต่ต่างจาก cron
// ทั่วไปตรงที่ตัว schedule เองเป็น interface (ดู Schedule ด้านล่าง) ทำให้ในอนาคตแทนที่
// จะรองรับแค่ "ทุก N เวลา" (Interval) จะเพิ่มกฎซับซ้อนกว่านั้นได้ (เช่น "เฉพาะวันทำการ
// สุดท้ายของเดือน" ตาม FULLSTACK-INFRA-ROADMAP.md Phase 2.5 — Calendar-based Scheduling)
// โดยไม่ต้องแก้ Scheduler เลยแม้แต่บรรทัดเดียว แค่เขียน Schedule implementation ใหม่
package scheduler

import (
	"context"
	"sync"
	"time"

	"bop/internal/workflow"
)

// Schedule ตัดสินใจว่า "ครั้งถัดไป" ควรรันเมื่อไหร่ โดยดูจากเวลาที่รันครั้งล่าสุด (หรือ
// เวลาปัจจุบันตอน register ถ้ายังไม่เคยรันเลย) — แยกเป็น interface เพื่อให้เพิ่มกฎ
// การตั้งเวลาแบบใหม่ๆ ได้ในอนาคตโดยไม่ต้องแก้ Scheduler
type Schedule interface {
	// Next คืนเวลาที่ควรรันครั้งถัดไป โดยคำนวณจาก after (เวลาที่รันครั้งล่าสุด/เวลาปัจจุบัน)
	Next(after time.Time) time.Time
}

// Interval คือ Schedule แบบง่ายที่สุด: รันซ้ำทุกๆ ระยะเวลาคงที่ (เช่น ทุก 1 ชั่วโมง)
// เทียบเท่า cron expression ธรรมดาแบบ "ทุก N นาที/ชั่วโมง" — ยังไม่รองรับกฎปฏิทินซับซ้อน
// (เช่น "ข้ามวันหยุด", "เฉพาะวันทำการสุดท้ายของเดือน") ดูหมายเหตุที่หัว package
type Interval struct {
	Every time.Duration
}

// Next ของ Interval ง่ายมาก: บวก Every เข้ากับเวลาที่รันครั้งล่าสุดตรงๆ ไม่มี logic ซับซ้อน
func (i Interval) Next(after time.Time) time.Time {
	return after.Add(i.Every)
}

// entry คือ workflow หนึ่งตัวที่ถูก register เข้า Scheduler พร้อม schedule ของมัน และ
// เวลาที่ควรรันครั้งถัดไป (คำนวณไว้ล่วงหน้าเสมอ ไม่ต้องคำนวณใหม่ทุกครั้งที่เช็ค)
type entry struct {
	workflow workflow.Workflow
	schedule Schedule
	nextRun  time.Time
}

// Scheduler คอยเช็คเป็นระยะๆ (ตาม tick ที่กำหนดตอนสร้าง) ว่ามี workflow ไหนถึงเวลาที่
// ควรรันบ้างหรือยัง แล้วสั่ง Runner.Trigger ให้ในเบื้องหลัง (ไม่รอผลลัพธ์ ไม่บล็อคการ
// เช็ครอบถัดไป)
type Scheduler struct {
	mu      sync.Mutex
	entries []*entry
	runner  *workflow.Runner
	tick    time.Duration
}

// New สร้าง Scheduler ใหม่
//   - runner: ตัวที่จะถูกสั่งให้ trigger workflow จริงๆ เมื่อถึงเวลา
//   - tick: ความถี่ที่ scheduler จะตื่นมาเช็คว่ามี workflow ไหนถึงเวลาหรือยัง (ยิ่งถี่ยิ่ง
//     แม่นยำเรื่องเวลา แต่ก็ใช้ CPU เช็คบ่อยขึ้นเล็กน้อย — ปกติตั้งเป็นระดับวินาทีก็พอ)
func New(runner *workflow.Runner, tick time.Duration) *Scheduler {
	return &Scheduler{runner: runner, tick: tick}
}

// Register เพิ่ม workflow เข้า scheduler พร้อมกำหนดว่าจะให้รันตาม schedule แบบไหน —
// คำนวณเวลาที่ควรรันครั้งแรกทันที (นับจากตอนนี้) ไว้เลย
func (s *Scheduler) Register(wf workflow.Workflow, sched Schedule) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, &entry{
		workflow: wf,
		schedule: sched,
		nextRun:  sched.Next(time.Now()),
	})
}

// Start เริ่มลูปหลักของ scheduler — ทำงานค้างอยู่ (ไม่ return) จนกว่า ctx จะถูกยกเลิก
// ควรเรียกใน goroutine แยกจาก main flow ของโปรแกรม (ดูตัวอย่างใน cmd/bop/main.go)
func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(s.tick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.checkAndTrigger(now)
		}
	}
}

// checkAndTrigger ไล่เช็คทุก entry ว่าถึงเวลาที่ควรรัน (now ไม่ก่อนกว่า nextRun) หรือยัง
// ตัวไหนถึงเวลาแล้ว จะคำนวณ nextRun รอบถัดไปทันที (กันไม่ให้ trigger ซ้ำที่ tick ถัดไป)
// แล้วค่อยสั่ง trigger จริงแบบ asynchronous (คนละ goroutine ต่อ workflow ที่ถึงเวลา) เพื่อ
// ไม่ให้ workflow ตัวหนึ่งที่ใช้เวลานานไปบล็อคการเช็ค/trigger workflow ตัวอื่นที่ถึงเวลาพร้อมกัน
func (s *Scheduler) checkAndTrigger(now time.Time) {
	s.mu.Lock()
	var due []*entry
	for _, e := range s.entries {
		if !now.Before(e.nextRun) { // now >= nextRun แปลว่าถึงเวลาแล้ว
			due = append(due, e)
			e.nextRun = e.schedule.Next(now)
		}
	}
	s.mu.Unlock()

	for _, e := range due {
		// รันแต่ละ workflow ที่ถึงเวลาใน goroutine แยกกัน ไม่ต้องรอผลตรงนี้ (ผลลัพธ์/error
		// ถูกบันทึกลง Run repository อยู่แล้วภายใน Trigger เอง คนที่อยากรู้ผลไปดูผ่าน
		// workflow.RunRepository หรือ endpoint GET /workflow-runs ได้)
		//
		// หมายเหตุข้อจำกัด (v1): ถ้า workflow ตัวเดิมยังรันจาก tick ก่อนหน้าไม่จบ แล้วถึง
		// เวลา trigger รอบใหม่อีกครั้ง จะรันซ้อนกันได้ (ไม่มีการเช็คว่า "ตัวเดิมยังรันอยู่
		// ไหม") — ของจริงแบบ Control-M มีกลไก "resource pool"/"skip if running" ป้องกัน
		// เรื่องนี้ ซึ่งวางแผนไว้เป็นฟีเจอร์ขั้นสูงใน Phase 3 ของ roadmap
		go s.runner.Trigger(context.Background(), e.workflow)
	}
}
