// Command bop คือจุดเริ่มต้นของ HTTP API server + CLI ของ BOP ทำหน้าที่ "ประกอบ" (wire)
// ส่วนที่เกี่ยวกับ job เดี่ยวๆ ทั้งหมดเข้าด้วยกัน: repository, logger, worker pool,
// register bot ที่รู้จักทั้งหมด, ต่อ Temporal client (สำหรับอ่านสถานะ workflow run ผ่าน
// HTTP เท่านั้น) แล้วเปิดใช้งานได้ 2 แบบ: (1) HTTP API server (ค่า default) สำหรับให้
// dashboard เรียกใช้ (2) CLI mode สำหรับสั่งรัน bot ตัวเดียวจบแล้วออกจากโปรแกรม เหมาะกับ
// ตอนทดสอบ core engine ก่อนที่จะมี UI จริง (ดู FULLSTACK-INFRA-ROADMAP.md Phase 0.4)
//
// process นี้ "ไม่ได้" รัน workflow ใดๆ เอง (ไม่มี Temporal Worker อยู่ในนี้) — การรัน
// workflow ทั้งแบบ scheduled และการ register Activity ทั้งหมดอยู่ที่ cmd/bop-worker
// ซึ่งเป็นคนละ process แยกกันโดยตั้งใจ (ดูเหตุผลที่หัวไฟล์ cmd/bop-worker/main.go) —
// ต้องรันทั้งสอง process คู่กันเสมอถ้าต้องการให้ workflow ทำงานจริง (ดู README.md)
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"go.temporal.io/sdk/client"

	apihttp "bop/internal/api/http" // ตั้งชื่อ alias เพราะ package นี้ก็ชื่อ "http" เหมือน net/http
	"bop/internal/bots/httpstatus"
	"bop/internal/job"
	"bop/internal/logger"
	memlogger "bop/internal/logger/memory"   // alias เพราะซ้ำชื่อกับ repository/memory ด้านล่าง
	memrepo "bop/internal/repository/memory" // alias เพราะซ้ำชื่อกับ logger/memory ด้านบน
	"bop/internal/schedule"
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

	// ขั้นตอนที่ 4: ต่อ Temporal client — ใช้แค่ "อ่าน" สถานะ workflow run ผ่าน
	// GET /workflow-runs (ดู internal/api/http/handler.go) เท่านั้น ไม่ได้ใช้สั่งรัน
	// workflow เอง (การสั่งรันอัตโนมัติตามตารางเวลาย้ายไปอยู่ที่ cmd/bop-worker ทั้งหมด
	// แล้ว ผ่าน Temporal Schedule — ดูเหตุผลการแยก process ที่หัวไฟล์ cmd/bop-worker/main.go)
	// ต้องรัน `docker compose up -d` (ดู docker-compose.yml) ให้ Temporal server พร้อมก่อน
	// ไม่งั้น Dial จะ error ทันที
	temporalClient, err := client.Dial(client.Options{
		HostPort:  "localhost:7233",
		Namespace: workflow.Namespace,
	})
	if err != nil {
		log.Fatalf("connect to temporal: %v", err)
	}
	defer temporalClient.Close()

	// ขั้นตอนที่ 5: bookkeeping ของ schedule definition (ชื่อ workflow + steps ต่อ
	// schedule หนึ่งตัว) — ดูเหตุผลที่แยกเก็บเองแทนที่จะ decode จาก Temporal ตรงๆ ที่
	// internal/schedule/schedule.go
	scheduleRepo := memrepo.NewScheduleRepository()

	// ขั้นตอนที่ 6: เช็คว่าถูกเรียกแบบ CLI mode หรือไม่ (มี argument แรกเป็นคำว่า "run")
	// ถ้าใช่ ให้ทำงานแบบ CLI แล้วจบโปรแกรมทันทีหลังจากนั้น ไม่เปิด HTTP server ต่อ
	if len(os.Args) > 1 && os.Args[1] == "run" {
		runCLI(repo, pool, os.Args[2:])
		return
	}

	// ถ้าไม่ใช่ CLI mode ให้เปิด HTTP API server ตามปกติ (ค่า default ของโปรแกรมนี้)
	serve(repo, appLogger, pool, temporalClient, scheduleRepo)
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
func serve(repo job.Repository, l logger.Logger, pool *worker.Pool, temporalClient client.Client, scheduleRepo schedule.Repository) {
	srv := apihttp.NewServer(repo, l, pool, temporalClient, scheduleRepo)
	addr := ":8080"
	fmt.Printf("bop api listening on %s\n", addr)
	if err := http.ListenAndServe(addr, srv); err != nil {
		log.Fatal(err)
	}
}
