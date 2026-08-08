// Package worker คือหัวใจด้าน concurrency ของทั้งระบบ: bounded pool ที่รัน bot job
// หลายตัวพร้อมกันได้ (จำกัดจำนวนสูงสุด) พร้อมรองรับ timeout ต่อ job และการยกเลิกกลางทาง
// ดู FULLSTACK-INFRA-ROADMAP.md Phase 0.2 สำหรับ checklist ที่เกี่ยวข้อง
package worker

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"bop/internal/bot"
	"bop/internal/job"
	"bop/internal/logger"
)

// Pool รัน job หลายตัวพร้อมกันได้ แต่จำกัดจำนวนสูงสุดที่รันพร้อมกันได้จริง (concurrency)
type Pool struct {
	bots    map[string]bot.Bot // bot ที่ register ไว้แล้ว (ชื่อ -> implementation)
	repo    job.Repository     // ที่เก็บสถานะ job (อัปเดตทุกครั้งที่สถานะเปลี่ยน)
	log     logger.Logger      // ที่ส่ง log ระหว่าง job ทำงาน
	sem     chan struct{}      // "semaphore" จำกัดจำนวน goroutine ที่ทำงานพร้อมกัน (อธิบายด้านล่าง)
	timeout time.Duration      // เวลาสูงสุดที่ยอมให้แต่ละ job รันได้ก่อนถูกตัดจบอัตโนมัติ

	mu      sync.Mutex                    // ป้องกัน cancels ด้านล่างจากการเข้าถึงพร้อมกัน
	cancels map[string]context.CancelFunc // job ID -> ฟังก์ชันสำหรับยกเลิก job นั้น (ถ้ากำลังรันอยู่)
}

// NewPool สร้าง pool ใหม่
//   - concurrency: จำนวน job สูงสุดที่ยอมให้รันพร้อมกันได้จริงในเวลาเดียวกัน
//   - jobTimeout: เวลาสูงสุดต่อ 1 job ก่อนจะถูกยกเลิกอัตโนมัติ (กัน bot ที่ค้างไม่รู้จบ)
func NewPool(concurrency int, jobTimeout time.Duration, repo job.Repository, log logger.Logger) *Pool {
	return &Pool{
		bots:    make(map[string]bot.Bot),
		repo:    repo,
		log:     log,
		sem:     make(chan struct{}, concurrency), // buffered channel ขนาด concurrency คือตัว semaphore
		timeout: jobTimeout,
		cancels: make(map[string]context.CancelFunc),
	}
}

// Register เพิ่ม bot ชนิดหนึ่งเข้าไปให้ pool รู้จักและเรียกใช้ได้ — ต้องเรียกตอน
// เริ่มโปรแกรม (ก่อน Submit) ครั้งเดียวต่อ bot หนึ่งชนิด
func (p *Pool) Register(b bot.Bot) {
	p.bots[b.Name()] = b
}

// List คืน bot ทั้งหมดที่ register ไว้ เรียงตามชื่อ (a-z) — ใช้ให้ HTTP handler ตอบ
// GET /bots (ดู internal/api/http/handler.go) โดย handler ไม่ต้องรู้จัก concrete
// bot.Bot ใดๆ เลย แค่เอา Name()/ConfigSchema() ของแต่ละตัวไปแปลงเป็น JSON ต่อ
//
// หมายเหตุเรื่อง concurrency: อ่าน p.bots ตรงๆ โดยไม่ล็อค เพราะ p.bots ถูกเขียนแค่ตอน
// Register (เรียกครั้งเดียวตอน wiring ใน cmd/bop/main.go ก่อนเปิด HTTP server เสมอ ยังไม่มี
// goroutine อื่นใดเริ่มทำงานพร้อมกัน) หลังจากนั้น map นี้จะถูก "อ่านอย่างเดียว" ตลอดอายุ
// โปรแกรม ไม่มีการเขียนซ้อนเข้ามาอีก — ถ้าในอนาคตมี dynamic bot registration ระหว่างที่
// โปรแกรมรันอยู่แล้ว (ไม่ใช่แค่ตอน startup) ต้องกลับมาใส่ mutex ป้องกันตรงนี้ด้วย
func (p *Pool) List() []bot.Bot {
	out := make([]bot.Bot, 0, len(p.bots))
	for _, b := range p.bots {
		out = append(out, b)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}

// Submit สั่งเริ่ม job แบบไม่รอผล (asynchronous / "fire-and-forget") — return ทันที
// โดยไม่รอให้ job ทำงานเสร็จ สถานะของ job จะถูกอัปเดตเข้า repository เรื่อยๆ ระหว่างที่
// ทำงาน ผู้เรียกต้องไปเช็คสถานะเองผ่าน repo.Get ทีหลัง — ใช้กรณีที่ผู้เรียก (เช่น HTTP
// handler ตอนสร้าง job เดี่ยวๆ ผ่าน POST /bots/{name}/jobs) ไม่ต้องการรอผลลัพธ์ตรงนั้น
//
// จุดสำคัญ: Submit จะ copy ค่าของ *j เป็นตัวแปรใหม่ก่อนส่งเข้า goroutine ที่รันจริง แล้ว
// ไม่แตะ pointer j ที่ผู้เรียกส่งเข้ามาอีกเลย — นี่คือสิ่งที่ป้องกันไม่ให้ผู้เรียก (เช่น
// HTTP handler ที่กำลังจะเอา j ไปแปลงเป็น JSON ตอบกลับ client) เกิดการเข้าถึงข้อมูล
// พร้อมกัน (data race) กับ goroutine ด้านล่างที่กำลังจะแก้ไขค่าต่างๆ ของ job นี้ระหว่างทำงาน
func (p *Pool) Submit(j *job.Job) error {
	b, ok := p.bots[j.BotName]
	if !ok {
		return fmt.Errorf("worker: unknown bot %q", j.BotName)
	}
	jobCopy := *j // คัดลอกค่าทั้งหมดออกมาเป็น value ใหม่ ไม่ใช่แค่คัดลอก pointer
	p.startRun(b, jobCopy)
	return nil
}

// SubmitAndWait เหมือน Submit ทุกประการ แต่จะ "รอ" จนกว่า job จะทำงานจบจริงๆ ก่อนจะ
// return (ไม่ว่าจะจบด้วยผลสำเร็จ ล้มเหลว หรือถูกยกเลิก) แล้วคืนสถานะสุดท้ายของ job นั้น
// กลับมาให้เลย — จำเป็นมากสำหรับ workflow ที่มีหลาย step เรียงกัน (ดู internal/workflow)
// เพราะ step ที่ 2 ต้องรอผลของ step ที่ 1 ก่อนถึงจะรู้ว่าควรเริ่มหรือไม่ (ถ้า step ที่ 1
// ล้มเหลว step ที่ 2 ก็ไม่ควรเริ่มเลย)
//
// ถ้า ctx ถูกยกเลิกก่อนที่ job จะทำงานจบ (เช่น timeout ของ workflow โดยรวม) จะ return
// error ของ ctx ทันทีโดยไม่รอ job ต่อ — แต่ตัว job เองจะยังทำงานต่อไปในเบื้องหลังจนกว่า
// job timeout ของตัวเองจะถึง (เพราะ job มี context ของตัวเองแยกจาก ctx ที่ส่งเข้ามาตรงนี้)
func (p *Pool) SubmitAndWait(ctx context.Context, j *job.Job) (*job.Job, error) {
	b, ok := p.bots[j.BotName]
	if !ok {
		return nil, fmt.Errorf("worker: unknown bot %q", j.BotName)
	}
	jobCopy := *j
	done := p.startRun(b, jobCopy)

	select {
	case final := <-done:
		return final, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// startRun คือ helper กลางที่ Submit และ SubmitAndWait ใช้ร่วมกัน — เปิด goroutine ใหม่
// ให้รัน job แล้วคืน channel ที่จะได้รับค่า job สุดท้ายทันทีที่ job นั้นทำงานจบ (Submit
// ไม่สนใจ channel นี้เลย ปล่อยทิ้งไว้เฉยๆ; SubmitAndWait รอค่าจาก channel นี้)
//
// channel มี buffer ขนาด 1 เสมอ เพื่อให้ p.run() ส่งค่าเข้าไปได้ทันทีโดยไม่ต้องรอใครมารับ
// — กันกรณี SubmitAndWait ยกเลิกการรอไปแล้วเพราะ ctx หมดเวลา แต่ job ยังทำงานต่อจนจบอยู่ดี
// ถ้า channel ไม่มี buffer, p.run() จะค้างรอส่งค่าตลอดไปเพราะไม่มีใครมารับอีกแล้ว (goroutine leak)
func (p *Pool) startRun(b bot.Bot, j job.Job) <-chan *job.Job {
	done := make(chan *job.Job, 1)
	go p.run(b, j, done)
	return done
}

// Cancel สั่งยกเลิก job ที่กำลังรันอยู่ (ถ้ามี) — คืนค่า false ถ้าหา job นั้นไม่เจอในรายการ
// ที่กำลังรันอยู่ (อาจจะจบไปแล้ว หรือ ID ไม่มีอยู่จริง)
func (p *Pool) Cancel(jobID string) bool {
	p.mu.Lock()
	cancel, ok := p.cancels[jobID]
	p.mu.Unlock()
	if !ok {
		return false
	}
	cancel() // เรียก cancel function ของ context ที่ผูกกับ job นี้ — ทำให้ ctx.Done() ของมัน "ปิด"
	return true
}

// run คือฟังก์ชันที่ทำงานจริงในแต่ละ goroutine ที่ถูกสร้างจาก startRun
// j เป็น value (ไม่ใช่ pointer) จึง "เป็นเจ้าของ" แต่เพียงผู้เดียวตลอดชีวิตของ goroutine นี้
// การแก้ไข field ของ j ด้านล่างจะไม่มีทางถูก goroutine อื่นมองเห็นโดยตรงเลย — วิธีเดียว
// ที่โลกภายนอกจะเห็นการเปลี่ยนแปลงคือผ่าน p.repo.Update (ซึ่งตัว repository เองก็ copy
// ค่าอีกชั้นตอนเก็บ — ดู internal/repository/memory) หรือผ่าน done channel ตอนจบ ทำให้
// ไม่มีจุดไหนเลยที่ struct เดียวกันถูกแก้ไขจากสอง goroutine พร้อมกัน เมื่อทำงานจบแล้ว
// จะส่งค่าสุดท้ายเข้า done เสมอ ไม่ว่าจะจบแบบไหน (สำเร็จ/ล้มเหลว/ถูกยกเลิก)
func (p *Pool) run(b bot.Bot, j job.Job, done chan<- *job.Job) {
	// ขั้นตอนที่ 1: รอจนกว่าจะมี worker slot ว่าง (ถ้า pool เต็มอยู่แล้ว จะค้างรอตรงนี้)
	// กลไกนี้เรียกว่า "semaphore pattern": p.sem คือ buffered channel ขนาดเท่ากับ
	// concurrency ที่ตั้งไว้ การส่งค่าเข้า channel (p.sem <- struct{}{}) จะสำเร็จทันที
	// ถ้ายังมีที่ว่างใน buffer แต่ถ้า buffer เต็มแล้ว (มี goroutine อื่นทำงานอยู่ครบจำนวน
	// concurrency แล้ว) บรรทัดนี้จะ "บล็อค" รอจนกว่าจะมี goroutine อื่นทำงานเสร็จแล้ว
	// อ่านค่าออกจาก channel (ปลดปล่อยที่ว่าง) ก่อน — struct{}{} คือค่าว่างเปล่าที่ไม่กิน memory
	// ใช้แค่เป็น "signal" ว่ามีคนจองที่ไว้เท่านั้น ไม่ได้สนใจค่าจริงๆ ที่ส่งเข้าไป
	p.sem <- struct{}{}
	defer func() { <-p.sem }() // เมื่อ job นี้จบ (ไม่ว่าสำเร็จ/ล้มเหลว) คืนที่ว่างกลับให้ job อื่นใช้ต่อ

	// ขั้นตอนที่ 2: สร้าง context ที่มี timeout ผูกกับ job นี้โดยเฉพาะ และเก็บ cancel
	// function ไว้ใน map เพื่อให้ Cancel() ข้างบนสามารถหามาเรียกได้ทีหลัง — ถ้า job ระบุ
	// TimeoutSeconds ของตัวเองมา (เช่น bot ที่ทำงานนานผิดปกติอย่าง externaljob) ใช้ค่านั้น
	// แทน p.timeout กลางของ pool ทั้งก้อน มิฉะนั้น (ค่าเริ่มต้น 0) ใช้ p.timeout ตามเดิม
	timeout := p.timeout
	if j.TimeoutSeconds > 0 {
		timeout = time.Duration(j.TimeoutSeconds) * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	p.mu.Lock()
	p.cancels[j.ID] = cancel
	p.mu.Unlock()
	defer func() {
		cancel() // ปลดปล่อย resource ของ context เสมอเมื่อ job จบ (ไม่ว่าจะจบด้วยเหตุผลอะไร)
		p.mu.Lock()
		delete(p.cancels, j.ID) // เอา entry ออกจาก map เพราะ job นี้ไม่ได้ "กำลังรัน" อีกต่อไปแล้ว
		p.mu.Unlock()
	}()

	// ขั้นตอนที่ 3: บันทึกว่า job เริ่มรันแล้ว พร้อมเวลาเริ่ม แล้วเปลี่ยนสถานะเป็น running
	now := time.Now()
	j.StartedAt = &now
	j.Status = job.StatusRunning
	_ = p.repo.Update(ctx, &j)
	p.log.Log(ctx, j.ID, logger.LevelInfo, fmt.Sprintf("job started: bot=%s", j.BotName))

	// ขั้นตอนที่ 4: เรียกให้ bot ทำงานจริง — สร้างฟังก์ชัน log แบบง่ายที่ผูก jobID ไว้ให้แล้ว
	// เพื่อส่งให้ bot เรียกใช้รายงานความคืบหน้าโดยไม่ต้องรู้จัก jobID หรือ logger.Logger เอง
	// (ใช้ context.Background() แทน ctx ตรงนี้เพราะอยากให้ log บันทึกได้เสมอแม้ ctx
	// ของ job จะหมดเวลา/ถูกยกเลิกไปแล้วก็ตาม)
	log := func(msg string) { p.log.Log(context.Background(), j.ID, logger.LevelInfo, msg) }
	result, err := b.Run(ctx, log, j.Config)

	// ขั้นตอนที่ 5: บันทึกผลลัพธ์สุดท้าย แยกเป็น 3 กรณี: ถูกยกเลิก / ล้มเหลว / สำเร็จ
	end := time.Now()
	j.EndedAt = &end
	switch {
	case err != nil && ctx.Err() == context.Canceled:
		// ctx.Err() == context.Canceled หมายความว่า error ที่ bot คืนมาเกิดจากการที่
		// เราสั่งยกเลิกเอง (ผ่าน Cancel()) ไม่ใช่ bot ทำงานผิดพลาดเอง จึงแยกสถานะเป็น
		// "cancelled" ไม่ใช่ "failed" เพื่อให้ dashboard แสดงผลถูกต้องตามความเป็นจริง
		j.Status = job.StatusCancelled
		j.Error = "cancelled by user"
		p.log.Log(context.Background(), j.ID, logger.LevelWarn, "job cancelled")
	case err != nil:
		j.Status = job.StatusFailed
		j.Error = err.Error()
		p.log.Log(context.Background(), j.ID, logger.LevelError, fmt.Sprintf("job failed: %v", err))
	default:
		j.Status = job.StatusSucceeded
		j.Summary = result.Summary
		p.log.Log(context.Background(), j.ID, logger.LevelInfo, fmt.Sprintf("job succeeded: %s", result.Summary))
	}
	// ใช้ context ใหม่ (ไม่ใช่ ctx เดิม) สำหรับการบันทึกครั้งสุดท้ายนี้ เพราะ ctx เดิมอาจ
	// หมดเวลาหรือถูกยกเลิกไปแล้ว แต่เราต้องการให้การบันทึกสถานะสุดท้ายสำเร็จเสมอไม่ว่ากรณีใด
	_ = p.repo.Update(context.Background(), &j)

	// ส่งค่าสุดท้ายเข้า done channel เสมอ (ดูคำอธิบาย buffer ขนาด 1 ที่ startRun ว่าทำไม
	// ส่งตรงนี้ไม่มีทางค้าง/บล็อค แม้จะไม่มีใครฟังอยู่แล้วก็ตาม)
	final := j
	done <- &final
}
