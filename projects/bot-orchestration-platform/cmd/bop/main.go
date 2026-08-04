// Command bop คือจุดเริ่มต้นของโปรแกรม ทำหน้าที่ "ประกอบ" (wire) ทุกส่วนของระบบเข้า
// ด้วยกัน: สร้าง repository, logger, worker pool, register bot ที่รู้จักทั้งหมด, ตั้ง
// workflow/scheduler สำหรับรัน bot อัตโนมัติตามเวลา แล้วเปิดใช้งานได้ 2 แบบ: (1) HTTP
// API server (ค่า default) สำหรับให้ dashboard เรียกใช้ (2) CLI mode สำหรับสั่งรัน bot
// ตัวเดียวจบแล้วออกจากโปรแกรม เหมาะกับตอนทดสอบ core engine ก่อนที่จะมี UI จริง
// (ดู FULLSTACK-INFRA-ROADMAP.md Phase 0.4)
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	apihttp "bop/internal/api/http" // ตั้งชื่อ alias เพราะ package นี้ก็ชื่อ "http" เหมือน net/http
	"bop/internal/bot"
	"bop/internal/bots/httpstatus"
	"bop/internal/job"
	"bop/internal/logger"
	memlogger "bop/internal/logger/memory"   // alias เพราะซ้ำชื่อกับ repository/memory ด้านล่าง
	memrepo "bop/internal/repository/memory" // alias เพราะซ้ำชื่อกับ logger/memory ด้านบน
	"bop/internal/scheduler"
	"bop/internal/worker"
	"bop/internal/workflow"
)

func main() {
	// ขั้นตอนที่ 1: สร้าง dependency พื้นฐานทั้งหมด — ตอนนี้ใช้ in-memory implementation
	// ทั้งคู่ (repository และ logger) เพราะยังไม่ได้ทำ PostgreSQL/OpenSearch จริง
	// (ดู roadmap Phase 0.4 และ Phase 2.2) การสลับไปใช้ implementation จริงในอนาคต
	// จะแก้แค่ 2 บรรทัดนี้เท่านั้น โค้ดส่วนอื่นทั้งหมดไม่ต้องแตะเลย เพราะทุกที่อ้างอิง
	// ผ่าน interface (job.Repository, logger.Logger) ไม่เคยอ้างอิง concrete type ตรงๆ
	repo := memrepo.NewJobRepository()
	appLogger := memlogger.New()

	// ขั้นตอนที่ 2: สร้าง worker pool — จำกัดให้รันได้พร้อมกันสูงสุด 4 job และแต่ละ job
	// มีเวลาไม่เกิน 30 วินาที ก่อนจะถูกตัดจบอัตโนมัติ (ป้องกัน bot ที่ค้างไม่รู้จบ)
	pool := worker.NewPool(4, 30*time.Second, repo, appLogger)

	// ขั้นตอนที่ 3: register bot ทุกชนิดที่ระบบรู้จักไว้ที่นี่ที่เดียว — การเพิ่ม bot
	// ชนิดใหม่ในอนาคตแค่เพิ่มบรรทัด pool.Register(...) ตรงนี้ ไม่ต้องแก้โค้ดที่อื่นเลย
	pool.Register(httpstatus.New())

	// ขั้นตอนที่ 4: ตั้ง workflow + scheduler — เวอร์ชันเบาก่อนย้ายไปใช้ Temporal จริงจัง
	// ใน roadmap Phase 2.5 (ดูข้อจำกัดที่อธิบายไว้ที่หัวไฟล์ internal/workflow/workflow.go)
	// runRepo เก็บประวัติการรันของแต่ละ workflow, runner รันแต่ละ step ตามลำดับผ่าน pool
	// เดิมที่ประกาศไว้ข้างบน (ไม่ได้สร้าง execution engine แยกต่างหาก), scheduler คอย
	// เช็คว่าถึงเวลาที่ workflow ไหนควรถูกสั่งรันอัตโนมัติหรือยัง
	runRepo := memrepo.NewRunRepository()
	runner := workflow.NewRunner(pool, runRepo)
	sched := scheduler.New(runner, time.Second) // เช็ค schedule ทุก 1 วินาที

	// ตัวอย่าง workflow: เช็คสถานะ example.com ทุก 1 นาที (มีแค่ 1 step ตอนนี้ เพื่อสาธิต
	// ว่า scheduler+runner ทำงานจริง — เพิ่ม step อื่นต่อท้ายได้ทันทีโดยไม่ต้องแก้โค้ดส่วนอื่น
	// เพราะ Runner รัน Steps ทีละตัวตามลำดับให้อัตโนมัติอยู่แล้ว)
	sched.Register(workflow.Workflow{
		Name: "check-example-com",
		Steps: []workflow.Step{
			{BotName: "http-status-checker", Config: bot.Config{"url": "https://example.com"}},
		},
	}, scheduler.Interval{Every: time.Minute})

	// รัน scheduler ใน goroutine แยก ให้ทำงานอยู่เบื้องหลังตลอดอายุโปรแกรม — ใช้
	// context.Background() ไปก่อน (ยังไม่มี graceful shutdown ในเวอร์ชันนี้ โปรแกรม
	// ถูกปิดยังไง scheduler ก็ถูกปิดไปพร้อมกันเพราะเป็น process เดียวกัน)
	go sched.Start(context.Background())

	// ขั้นตอนที่ 5: เช็คว่าถูกเรียกแบบ CLI mode หรือไม่ (มี argument แรกเป็นคำว่า "run")
	// ถ้าใช่ ให้ทำงานแบบ CLI แล้วจบโปรแกรมทันทีหลังจากนั้น ไม่เปิด HTTP server ต่อ
	if len(os.Args) > 1 && os.Args[1] == "run" {
		runCLI(repo, pool, os.Args[2:])
		return
	}

	// ถ้าไม่ใช่ CLI mode ให้เปิด HTTP API server ตามปกติ (ค่า default ของโปรแกรมนี้)
	serve(repo, appLogger, pool, runRepo)
}

// runCLI คือโหมดสำหรับทดสอบ engine โดยไม่ต้องเปิด server หรือมี UI เลย ใช้แบบนี้:
//
//	go run ./cmd/bop run --bot=http-status-checker --url=https://example.com
//
// การทำงาน:
//  1. อ่านค่า --bot และ --url จาก command line argument ด้วย flag package
//  2. สร้าง Job ใหม่เอง (แทนที่จะผ่าน HTTP API) แล้วสั่ง repo.Create + pool.Submit
//     เหมือนกับที่ HTTP handler ทำทุกประการ (พิสูจน์ว่า core engine ใช้งานได้จริง
//     โดยไม่ต้องพึ่ง HTTP layer เลย)
//  3. เข้าลูป "polling" เช็คสถานะของ job ทุก 200 มิลลิวินาที จนกว่าจะเจอสถานะที่ไม่ใช่
//     pending/running อีกต่อไป (คือ job จบแล้วไม่ว่าจะสำเร็จ/ล้มเหลว/ถูกยกเลิก) แล้ว
//     print ผลลัพธ์สุดท้ายออกมาก่อนจบโปรแกรม — เหตุผลที่ต้อง poll แบบนี้เพราะ
//     pool.Submit ทำงานแบบ asynchronous (ไม่รอผล) จึงต้องมีวิธีรอผลลัพธ์เอาเองฝั่ง CLI
func runCLI(repo job.Repository, pool *worker.Pool, args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	botName := fs.String("bot", "", "bot name to run")
	url := fs.String("url", "", "url config value passed to the bot")
	_ = fs.Parse(args)

	if *botName == "" {
		log.Fatal("usage: bop run --bot=<name> --url=<url>")
	}

	j := &job.Job{
		ID:        fmt.Sprintf("cli-%d", time.Now().UnixNano()), // ใช้ timestamp เป็น ID พอสำหรับ CLI mode (ไม่ต้องสุ่มแบบ HTTP handler)
		BotName:   *botName,
		Config:    map[string]string{"url": *url},
		Status:    job.StatusPending,
		CreatedAt: time.Now(),
	}
	if err := repo.Create(context.Background(), j); err != nil {
		log.Fatal(err)
	}
	if err := pool.Submit(j); err != nil {
		log.Fatal(err)
	}

	// รอจน job จบด้วยการเช็คสถานะซ้ำๆ (polling) — วิธีนี้ง่ายที่สุดสำหรับ CLI เล็กๆ
	// ถ้าเป็นระบบจริงจังกว่านี้อาจใช้ channel หรือ logger.Subscribe แทนการ poll ตรงๆ
	for {
		time.Sleep(200 * time.Millisecond)
		current, err := repo.Get(context.Background(), j.ID)
		if err != nil {
			log.Fatal(err)
		}
		if current.Status != job.StatusPending && current.Status != job.StatusRunning {
			fmt.Printf("job %s finished: status=%s summary=%q error=%q\n",
				current.ID, current.Status, current.Summary, current.Error)
			return
		}
	}
}

// serve เปิด HTTP API server ที่ port 8080 — ผูก dependency ทั้งหมดเข้ากับ Server
// ที่ประกาศไว้ใน internal/api/http แล้วปล่อยให้ http.ListenAndServe รับ request เข้ามาเรื่อยๆ
// จนกว่าโปรแกรมจะถูกปิด (Ctrl+C หรือ signal อื่น)
func serve(repo job.Repository, l logger.Logger, pool *worker.Pool, runRepo workflow.RunRepository) {
	srv := apihttp.NewServer(repo, l, pool, runRepo)
	addr := ":8080"
	fmt.Printf("bop api listening on %s\n", addr)
	if err := http.ListenAndServe(addr, srv); err != nil {
		log.Fatal(err)
	}
}
