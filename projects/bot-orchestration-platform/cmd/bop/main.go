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
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.temporal.io/sdk/client"

	apihttp "bop/internal/adapter/http" // ตั้งชื่อ alias เพราะ package นี้ก็ชื่อ "http" เหมือน net/http
	"bop/internal/adapter/worker"
	"bop/internal/domain/job"
	"bop/internal/domain/logger"
	"bop/internal/external/bots/externaljob"
	"bop/internal/external/bots/httpstatus"
	"bop/internal/external/temporal"
	memlogger "bop/internal/pkg/logger"
	"bop/internal/repository/memory"
	"bop/internal/repository/postgres"
	authuc "bop/internal/usecase/auth"
	jobuc "bop/internal/usecase/job"
	"bop/internal/usecase/logretention"
	scheduleuc "bop/internal/usecase/schedule"
	workflowrunuc "bop/internal/usecase/workflowrun"
)

// shutdownTimeout คือเวลาที่ยอมรอให้ HTTP request และ job ที่ค้างอยู่จบเองตอนปิดโปรแกรม
// ก่อนจะยกเลิกทิ้ง (เท่ากับ job timeout กลางของ pool ด้านล่าง)
const shutdownTimeout = 30 * time.Second

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run ประกอบ dependency ทั้งหมดแล้วเริ่มทำงานจนกว่าจะได้รับ SIGINT/SIGTERM — คืน error
// แทนการเรียก log.Fatal เอง เพื่อให้ main เป็นจุดเดียวที่ exit และ defer ทุกตัวในนี้
// (เช่นปิด Temporal client) ได้ทำงานเสมอ
func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// ขั้นตอนที่ 1: สร้าง dependency พื้นฐานทั้งหมด — ตั้ง BOP_DATABASE_URL = เก็บ Job และ log
	// ของ Job ใน PostgreSQL (database `bop`) ให้อยู่รอดข้าม restart ไม่ตั้ง = in-memory แบบเดิม
	// พร้อมคำเตือน (ดู openDatabase) — ทุกที่อ้างอิงผ่าน interface (job.Repository, logger.Logger)
	// จึงสลับ implementation ได้ที่นี่ที่เดียว
	db, err := openDatabase(ctx, os.Getenv("BOP_DATABASE_URL"))
	if err != nil {
		return err
	}
	if db != nil {
		// defer ตัวนี้รันหลัง pool.Shutdown ด้านล่าง เพราะ job ที่ถูกยกเลิกตอนปิดยังต้องบันทึกสถานะ
		defer db.Close()
	}
	repo := newJobRepository(db)
	appLogger := newLogger(db)

	// ขั้นตอนที่ 2: สร้าง worker pool — จำกัดให้รันได้พร้อมกันสูงสุด 4 job และแต่ละ job
	// มีเวลาไม่เกิน 30 วินาที ก่อนจะถูกตัดจบอัตโนมัติ (ป้องกัน bot ที่ค้างไม่รู้จบ)
	pool := worker.NewPool(4, 30*time.Second, repo, appLogger)

	// ขั้นตอนที่ 3: register bot ทุกชนิดที่ระบบรู้จักไว้ที่นี่ที่เดียว — การเพิ่ม bot
	// ชนิดใหม่ในอนาคตแค่เพิ่มบรรทัด pool.Register(...) ตรงนี้ ไม่ต้องแก้โค้ดที่อื่นเลย
	pool.Register(httpstatus.New())
	pool.Register(externaljob.New())

	// ขั้นตอนที่ 4: ต่อ Temporal client — ใช้แค่ "อ่าน" สถานะ workflow run ผ่าน
	// GET /workflow-runs (ดู internal/adapter/http/list_workflow_runs.go) เท่านั้น ไม่ได้ใช้สั่งรัน
	// workflow เอง (การสั่งรันอัตโนมัติตามตารางเวลาย้ายไปอยู่ที่ cmd/bop-worker ทั้งหมด
	// แล้ว ผ่าน Temporal Schedule — ดูเหตุผลการแยก process ที่หัวไฟล์ cmd/bop-worker/main.go)
	// ต้องรัน `docker compose up -d` (ดู docker-compose.yml) ให้ Temporal server พร้อมก่อน
	// ไม่งั้น Dial จะ error ทันที — อ่านจาก env var TEMPORAL_HOST_PORT ก่อนเสมอ (ตอน deploy
	// จริงบน K8s ค่านี้จะชี้ไปที่ Service ของ Temporal แทน localhost) fallback เป็นค่าเดิม
	// ถ้าไม่ได้ตั้ง env var ไว้ (เช่นตอน local dev)
	temporalClient, err := client.Dial(client.Options{
		HostPort:  getenv("TEMPORAL_HOST_PORT", "localhost:7233"),
		Namespace: temporal.Namespace,
	})
	if err != nil {
		return fmt.Errorf("connect to temporal: %w", err)
	}
	defer temporalClient.Close()

	// ขั้นตอนที่ 5: bookkeeping ของ schedule definition (ชื่อ workflow + steps ต่อ
	// schedule หนึ่งตัว) — ดูเหตุผลที่แยกเก็บเองแทนที่จะ decode จาก Temporal ตรงๆ ที่
	// internal/domain/schedule/schedule.go
	scheduleRepo := memory.NewScheduleRepository()

	// ขั้นตอนที่ 6: เช็คว่าถูกเรียกแบบ CLI mode หรือไม่ (มี argument แรกเป็นคำว่า "run")
	// ถ้าใช่ ให้ทำงานแบบ CLI แล้วจบโปรแกรมทันทีหลังจากนั้น ไม่เปิด HTTP server ต่อ
	if len(os.Args) > 1 && os.Args[1] == "run" {
		return runCLI(ctx, repo, pool, os.Args[2:])
	}

	// ขั้นตอนที่ 7: ประกอบ usecase จาก port ทั้งหมด (clean architecture: HTTP adapter รู้จัก
	// แค่ usecase, usecase รู้จักแค่ interface ใน internal/domain ส่วน implementation จริง
	// ทั้งหมดถูกเลือกที่นี่ที่เดียว) — secret/credential ของ auth อ่านจาก env var ที่นี่
	// **ห้ามใช้ค่า fallback ใน production เด็ดขาด** BOP_JWT_SECRET ต้องสุ่มค่าใหม่ด้วย
	// `openssl rand -base64 32` แล้วตั้งผ่าน K8s Secret ก่อน deploy จริง (ดู
	// KUBERNETES-DEPLOYMENT-TODO.md หัวข้อ 3.4) credential admin เป็นตัวเดียวที่ระบบรู้จัก
	// ตอนนี้เพราะยังไม่มี user database (BOP เป็น single-operator tool)
	jobs := jobuc.NewService(repo, pool, appLogger)
	runs := workflowrunuc.NewService(temporal.NewEngine(temporalClient))
	schedules := scheduleuc.NewService(temporal.NewScheduler(temporalClient), scheduleRepo)
	auth := authuc.NewService(
		[]byte(getenv("BOP_JWT_SECRET", "bop-dev-secret-change-me-before-deploying")),
		getenv("BOP_ADMIN_USERNAME", "admin"),
		getenv("BOP_ADMIN_PASSWORD", "changeme"),
	)

	// งานลบ log ที่เก่ากว่า 30 วันรันเฉพาะใน HTTP API server (process ที่อยู่ยาว) และเฉพาะเมื่อ
	// log อยู่ใน PostgreSQL — stopRetention ต้องรันก่อน db.Close ด้านบน (defer ทำงานย้อนลำดับ)
	stopRetention := startLogRetention(ctx, appLogger)
	defer stopRetention()

	// ถ้าไม่ใช่ CLI mode ให้เปิด HTTP API server ตามปกติ (ค่า default ของโปรแกรมนี้)
	if err := serve(ctx, apihttp.NewServer(jobs, runs, schedules, auth)); err != nil {
		return err
	}

	// HTTP server ปิดแล้วจึงไม่มี job ใหม่เข้ามาอีก — รอ job ที่ค้างอยู่ให้จบ ถ้าเกิน
	// shutdownTimeout pool จะยกเลิก job ที่เหลือและบันทึกสถานะเป็น cancelled ให้
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := pool.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown worker pool: %w", err)
	}
	return nil
}

// openDatabase เตรียม PostgreSQL ของ BOP ตอน start: รัน migration ก่อน (bop-api เป็น process
// เดียวที่รัน migration — bop-worker แค่ postgres.Connect เพื่อไม่ให้สอง process แข่งกันแก้
// schema) แล้วเปิด connection pool ที่ repository/logger ทุกตัวใช้ร่วมกัน
//
// databaseURL ว่าง = คืน pool เป็น nil พร้อม log เตือน ให้ `go run` แบบไม่มี database ยังใช้ได้
// (ข้อมูลหายเมื่อ restart) ผู้เรียกต้อง Close pool ที่ได้เอง
func openDatabase(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	if databaseURL == "" {
		log.Printf("warning: BOP_DATABASE_URL not set, using in-memory storage (jobs and logs are lost on restart)")
		return nil, nil
	}
	if err := postgres.Migrate(databaseURL); err != nil {
		return nil, err
	}
	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	log.Printf("using PostgreSQL job and log storage")
	return db, nil
}

// newJobRepository เลือก implementation ของ job.Repository ตาม pool ที่ openDatabase คืนมา
func newJobRepository(db *pgxpool.Pool) job.Repository {
	if db == nil {
		return memory.NewJobRepository()
	}
	return postgres.NewJobRepository(db)
}

// newLogger เลือก implementation ของ logger.Logger ตาม pool ที่ openDatabase คืนมา
func newLogger(db *pgxpool.Pool) logger.Logger {
	if db == nil {
		return memlogger.New()
	}
	return postgres.NewLogger(db)
}

// startLogRetention เริ่มงานลบ log ตามอายุใน goroutine ถ้า l ลบ log ได้ (PostgreSQL) แล้วคืน
// ฟังก์ชันที่หยุดงานและรอจน goroutine จบ — logger แบบ memory ไม่มีอะไรให้ลบ คืน no-op
func startLogRetention(ctx context.Context, l logger.Logger) (stop func()) {
	pruner, ok := l.(logger.Pruner)
	if !ok {
		return func() {}
	}
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		logretention.NewService(pruner).Run(ctx)
	}()
	return func() {
		cancel()
		<-done
	}
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
//
// ถ้ากด Ctrl+C ระหว่างรอ (ctx ถูกยกเลิก) จะเลิกรอแล้วคืน ctx.Err()
func runCLI(ctx context.Context, repo job.Repository, pool *worker.Pool, args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	botName := fs.String("bot", "", "bot name to run")
	url := fs.String("url", "", "url config value passed to the bot")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil // ผู้ใช้ขอดู -h: flag พิมพ์ usage ให้แล้ว ไม่ใช่ error
		}
		return err
	}

	if *botName == "" {
		return errors.New("usage: bop run --bot=<name> --url=<url>")
	}

	j := &job.Job{
		ID:        fmt.Sprintf("cli-%d", time.Now().UnixNano()), // ใช้ timestamp เป็น ID พอสำหรับ CLI mode (ไม่ต้องสุ่มแบบ HTTP handler)
		BotName:   *botName,
		Config:    map[string]string{"url": *url},
		Status:    job.StatusPending,
		CreatedAt: time.Now(),
	}
	if err := repo.Create(ctx, j); err != nil {
		return fmt.Errorf("create job: %w", err)
	}
	if err := pool.Submit(j); err != nil {
		return fmt.Errorf("submit job: %w", err)
	}

	// รอจน job จบด้วยการเช็คสถานะซ้ำๆ (polling) — วิธีนี้ง่ายที่สุดสำหรับ CLI เล็กๆ
	// ถ้าเป็นระบบจริงจังกว่านี้อาจใช้ channel หรือ logger.Subscribe แทนการ poll ตรงๆ
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		current, err := repo.Get(ctx, j.ID)
		if err != nil {
			return fmt.Errorf("get job: %w", err)
		}
		if current.Status.IsFinal() {
			fmt.Printf("job %s finished: status=%s summary=%q error=%q\n",
				current.ID, current.Status, current.Summary, current.Error)
			return nil
		}
	}
}

// serve เปิด HTTP API server จนกว่า ctx จะถูกยกเลิก (Ctrl+C หรือ SIGTERM) แล้วปิดแบบ
// graceful: หยุดรับ connection ใหม่และรอ request ที่ค้างอยู่ภายใน shutdownTimeout —
// addr อ่านจาก env var BOP_HTTP_ADDR ก่อนเสมอ fallback เป็น ":8080" (local dev)
//
// BaseContext ผูก context ของทุก request เข้ากับ ctx เพราะ SSE stream (GET
// /jobs/{id}/logs/stream) เปิดค้างไว้จนกว่า r.Context() จะถูกยกเลิก ถ้าไม่ผูกไว้
// srv.Shutdown จะรอ stream ที่เปิดอยู่จนหมด shutdownTimeout ทุกครั้งที่มีคนเปิดหน้า live log
func serve(ctx context.Context, handler http.Handler) error {
	srv := &http.Server{
		Addr:        getenv("BOP_HTTP_ADDR", ":8080"),
		Handler:     handler,
		BaseContext: func(net.Listener) context.Context { return ctx },
	}

	errCh := make(chan error, 1)
	go func() {
		fmt.Printf("bop api listening on %s\n", srv.Addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("http server: %w", err) // เปิดไม่สำเร็จตั้งแต่ต้น เช่น port ถูกใช้อยู่
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}
	if err := <-errCh; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("http server: %w", err)
	}
	return nil
}

// getenv คืนค่า environment variable ชื่อ key ถ้าตั้งไว้ (และไม่ใช่ค่าว่าง) ไม่งั้นคืนค่า
// fallback แทน — ใช้รวมจุดเดียวสำหรับ pattern "อ่านจาก env ก่อน ไม่มีค่อย fallback เป็นค่า
// hardcode เดิม" ที่ใช้ซ้ำหลายจุดในไฟล์นี้ (BOP_HTTP_ADDR, TEMPORAL_HOST_PORT)
func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
