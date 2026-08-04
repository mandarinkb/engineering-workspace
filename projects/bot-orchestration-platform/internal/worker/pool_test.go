package worker_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"bop/internal/bot"
	"bop/internal/job"
	memlogger "bop/internal/logger/memory"
	memrepo "bop/internal/repository/memory"
	"bop/internal/worker"
)

// stubBot คือ bot ปลอมที่สร้างขึ้นมาเฉพาะไว้ทดสอบพฤติกรรมของ worker pool (concurrency
// และการยกเลิก) เท่านั้น ไม่ได้ยิง network จริงเลย เพื่อให้ test รันเร็วและไม่พึ่งพา
// internet หรือ service ภายนอกใดๆ
type stubBot struct{ delay time.Duration } // delay คือเวลาที่ bot ปลอมนี้จะ "แกล้งทำงาน" นานแค่ไหน

func (b *stubBot) Name() string { return "stub" }

// Run แกล้งทำงานเป็นเวลา delay แล้วคืนค่าสำเร็จ — แต่ถ้า ctx ถูกยกเลิกก่อนครบเวลา
// (เช่นถูกเรียก Cancel หรือ timeout) จะหยุดทำงานทันทีและคืน error ของ ctx ออกไปแทน
// (พฤติกรรมนี้จำลอง bot จริงที่ควรเช็ค ctx.Done() ระหว่างทำงานเสมอ)
func (b *stubBot) Run(ctx context.Context, log bot.LogFunc, _ bot.Config) (bot.Result, error) {
	log("starting")
	select {
	case <-time.After(b.delay):
	case <-ctx.Done():
		return bot.Result{}, ctx.Err()
	}
	return bot.Result{Summary: "done"}, nil
}

// TestPool_RunsJobsConcurrentlyWithoutRace ทดสอบว่า pool รัน job หลายตัวพร้อมกันได้จริง
// (มากกว่าจำนวน concurrency ที่ตั้งไว้ ต้องรอคิวถูกต้อง) และทุก job จบด้วยสถานะสำเร็จ
// ในที่สุด — สำคัญที่สุดคือต้องรันด้วยคำสั่ง `go test -race ./...` เพื่อให้ Go ตรวจจับ
// data race (การเข้าถึงข้อมูลเดียวกันพร้อมกันจากหลาย goroutine โดยไม่มีการป้องกัน) ที่
// อาจเกิดขึ้นระหว่าง worker pool กับโค้ดที่เรียกใช้งานมันอยู่ (ดู FULLSTACK-INFRA-ROADMAP.md
// Phase 0.2 และคำอธิบายบทเรียนที่เจอจริงใน README ของโปรเจกต์นี้)
func TestPool_RunsJobsConcurrentlyWithoutRace(t *testing.T) {
	repo := memrepo.NewJobRepository()
	log := memlogger.New()
	pool := worker.NewPool(3, time.Second, repo, log) // จำกัดพร้อมกันได้แค่ 3 แต่จะยิง 10 job
	pool.Register(&stubBot{delay: 20 * time.Millisecond})

	const n = 10
	var wg sync.WaitGroup // ใช้รอให้ทุก goroutine ที่ตรวจสอบผลลัพธ์ทำงานเสร็จก่อนจบ test
	for i := 0; i < n; i++ {
		j := &job.Job{
			ID:        fmt.Sprintf("job-%d", i),
			BotName:   "stub",
			Status:    job.StatusPending,
			CreatedAt: time.Now(),
		}
		if err := repo.Create(context.Background(), j); err != nil {
			t.Fatalf("create job: %v", err)
		}
		if err := pool.Submit(j); err != nil {
			t.Fatalf("submit job: %v", err)
		}

		// เปิด goroutine แยกไว้คอยเช็คว่า job นี้จบสำเร็จภายในเวลาที่กำหนดหรือไม่
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			deadline := time.Now().Add(2 * time.Second)
			for time.Now().Before(deadline) {
				current, err := repo.Get(context.Background(), id)
				if err != nil {
					t.Errorf("get job %s: %v", id, err)
					return
				}
				if current.Status == job.StatusSucceeded {
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
			t.Errorf("job %s did not finish in time", id)
		}(j.ID)
	}
	wg.Wait() // รอให้ทุก goroutine ตรวจสอบผลเสร็จก่อนจบ test
}

// TestPool_CancelStopsAJobInFlight ทดสอบว่า Cancel() สามารถหยุด job ที่กำลังทำงาน
// อยู่จริงๆ ได้ ไม่ใช่แค่ปล่อยให้ job ทำงานจนจบตามปกติแล้วเปลี่ยน status ทีหลังเฉยๆ
func TestPool_CancelStopsAJobInFlight(t *testing.T) {
	repo := memrepo.NewJobRepository()
	log := memlogger.New()
	pool := worker.NewPool(1, 5*time.Second, repo, log)
	pool.Register(&stubBot{delay: 5 * time.Second}) // ตั้งให้ทำงานนานๆ เพื่อให้มีเวลายิง Cancel ทัน

	j := &job.Job{ID: "cancel-me", BotName: "stub", Status: job.StatusPending, CreatedAt: time.Now()}
	if err := repo.Create(context.Background(), j); err != nil {
		t.Fatalf("create job: %v", err)
	}
	if err := pool.Submit(j); err != nil {
		t.Fatalf("submit job: %v", err)
	}

	// รอสักครู่ให้แน่ใจว่า job เริ่มทำงานจริงแล้ว (ผ่านสถานะ pending ไปเป็น running แล้ว)
	// ก่อนจะสั่งยกเลิก เพื่อทดสอบสถานการณ์ "ยกเลิก job ที่กำลังทำงานอยู่จริง" ไม่ใช่
	// "ยกเลิก job ที่ยังไม่ทันเริ่ม"
	time.Sleep(50 * time.Millisecond)
	if !pool.Cancel(j.ID) {
		t.Fatal("expected Cancel to find a running job")
	}

	// เช็คซ้ำๆ ว่าสถานะเปลี่ยนเป็น cancelled ภายในเวลาที่กำหนดหรือไม่
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		current, err := repo.Get(context.Background(), j.ID)
		if err != nil {
			t.Fatalf("get job: %v", err)
		}
		if current.Status == job.StatusCancelled {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("job was not marked cancelled in time")
}
