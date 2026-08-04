# Bot Orchestration Platform (BOP)

Go core of the flagship project described in [FULLSTACK-INFRA-ROADMAP.md](../../FULLSTACK-INFRA-ROADMAP.md). This is the "core/control plane" that owns bot execution, job state, and log output — the Next.js dashboard (Phase 1, not started yet) will be a client of the HTTP API defined here, nothing more.

## สถานะปัจจุบัน

Phase 0 (`Golang: สร้าง Bot Execution Engine`) ทำแล้ว: `Bot` interface, worker pool ที่รันพร้อมกันได้หลาย job แบบมี timeout/cancel, in-memory repository + logger (persist + live-subscribe design พร้อมสลับเป็น PostgreSQL/OpenSearch ทีหลังโดยไม่แก้ caller), bot ตัวอย่างที่ implement interface จริง, และ HTTP API ที่ dashboard ใน Phase 1 จะเรียกใช้ได้ทันที — ทั้งหมด build/vet/test -race ผ่าน และลองยิงจริงทั้ง CLI กับ HTTP server แล้ว

**เพิ่มมาก่อนกำหนด (ปกติวางแผนไว้เป็น Phase 2.5)**: `internal/workflow` + `internal/scheduler` — รัน bot หลายตัวเรียงเป็น workflow เดียว (step ถัดไปเริ่มได้ก็ต่อเมื่อ step ก่อนหน้าสำเร็จ) พร้อม scheduler ที่ trigger workflow ตามเวลาอัตโนมัติ (เทียบเท่า cron ของระบบนี้) ดูรายละเอียดที่หัวข้อ "Workflow และ Scheduler" ด้านล่าง — **นี่คือเวอร์ชันเบาก่อนย้ายไป Temporal จริงจัง ไม่ใช่ของทดแทนถาวร** (ไม่มี durable execution, ไม่มี retry policy ต่อ step, ความคืบหน้าหายถ้า process restart กลางทาง — ข้อจำกัดพวกนี้มีคอมเมนต์อธิบายไว้ในโค้ดและเป็นเหตุผลที่ Temporal มีค่าจริงตอนไปถึง Phase 2.5)

ยังไม่ทำ (ของ Phase 0 ที่เหลือ) — ดู checklist เต็มที่ [roadmap Phase 0](../../FULLSTACK-INFRA-ROADMAP.md#phase-0--golang-สร้าง-bot-execution-engine):

- [ ] Bot ตัวที่สองที่ใช้ `chromedp` (headless browser) พิสูจน์ว่า interface รองรับ implementation ที่ซับซ้อนกว่า HTTP GET ธรรมดา
- [ ] แยก error type retryable/non-retryable ด้วย `errors.Is`/`errors.As` (ตอนนี้ error จาก bot แค่ถูกเก็บเป็น string ธรรมดา)
- [ ] PostgreSQL implementation ของ `job.Repository` (ตอนนี้มีแค่ in-memory)
- [ ] Table-driven test ของ logic parse ข้อมูล + `httptest.Server` mock (มี test ของ worker pool/workflow runner/scheduler แล้ว แต่ยังไม่มีของ logic parse ข้อมูลจริงจัง เพราะยังไม่มี bot ที่ parse อะไรซับซ้อน)

## สถาปัตยกรรม

```
cmd/bop/main.go          ประกอบทุกอย่างเข้าด้วยกัน (dependency injection แบบ manual)
   │
   ├── internal/bot        Bot interface — ทุก bot implement อันนี้ ไม่รู้จัก HTTP/DB/logger เลย
   ├── internal/job        Job entity + Repository interface
   ├── internal/logger     Logger interface (persist + live-subscribe ในตัวเดียว)
   ├── internal/worker     Pool — concurrency core, รัน bot ผ่าน semaphore-bounded goroutine
   │                       (Submit = fire-and-forget, SubmitAndWait = รอผลก่อนไปต่อ)
   │
   ├── internal/workflow   Workflow/Step/Run + Runner — รัน step หลายตัวเรียงลำดับผ่าน
   │                       worker.Pool.SubmitAndWait, หยุดทันทีถ้า step ไหนล้มเหลว
   ├── internal/scheduler  Schedule interface + Interval + Scheduler — คอย trigger
   │                       workflow ตามเวลา (เทียบเท่า cron ของระบบนี้)
   │
   ├── internal/bots/httpstatus     Bot ตัวอย่าง (zero external dependency)
   ├── internal/repository/memory   Repository implementation (in-memory) — เก็บทั้ง Job และ Run
   ├── internal/logger/memory       Logger implementation (in-memory)
   └── internal/api/http             HTTP handler — สิ่งเดียวที่ dashboard จะคุยด้วย
```

ทุก dependency ชี้เข้าหา interface ที่ `internal/bot`, `internal/job`, `internal/logger`, `internal/workflow` (RunRepository), `internal/scheduler` (Schedule) ไม่ใช่ implementation ตรงๆ — สลับ `internal/repository/memory` เป็น `internal/repository/postgres` ทีหลัง (roadmap Phase 0.4) แก้แค่บรรทัดเดียวใน `cmd/bop/main.go` โค้ดส่วนอื่นไม่ต้องแตะเลย เช่นเดียวกับที่จะสลับ `scheduler.Interval` เป็น calendar-based schedule ที่ซับซ้อนกว่าในอนาคต (roadmap Phase 2.5) โดยไม่ต้องแก้ `Scheduler` เลย

## รันยังไง

```bash
# ตั้ง Go module dependency ให้เรียบร้อยก่อน (ตอนนี้ไม่มี external dependency เลย)
go build ./...
go vet ./...
go test -race ./...

# CLI mode — สั่งรัน bot ตัวเดียวจบแล้ว exit (ไม่ต้องมี server/UI)
go run ./cmd/bop run --bot=http-status-checker --url=https://example.com

# Server mode (default) — เปิด HTTP API ที่ :8080
go run ./cmd/bop
```

ตัวอย่างเรียก API ด้วย curl:

```bash
curl -X POST localhost:8080/bots/http-status-checker/jobs -d '{"url":"https://example.com"}'
curl localhost:8080/jobs
curl localhost:8080/jobs/<id>
curl localhost:8080/jobs/<id>/logs
curl -X DELETE localhost:8080/jobs/<id>   # cancel ถ้า job ยังรันอยู่

curl localhost:8080/workflow-runs         # ดูว่า scheduled workflow รันไปแล้วกี่รอบ
curl localhost:8080/workflow-runs/<id>    # ดูผลลัพธ์ทีละ step ของ run เดียว
```

## Workflow และ Scheduler

`cmd/bop/main.go` register ตัวอย่าง workflow ไว้ 1 ตัว (`check-example-com`) ที่รัน `http-status-checker` เป็น step เดียว ทุก 1 นาที ผ่าน `scheduler.Interval{Every: time.Minute}` — เปิด server แล้วรอสัก 1-2 นาทีจะเห็น run ใหม่โผล่ใน `GET /workflow-runs` เอง โดยไม่ต้องสั่งอะไรเพิ่ม

เพิ่ม step ที่ 2, 3 เข้าไปได้ทันทีโดยไม่ต้องแก้โค้ดส่วนอื่นเลย:

```go
sched.Register(workflow.Workflow{
    Name: "check-multiple-sites",
    Steps: []workflow.Step{
        {BotName: "http-status-checker", Config: bot.Config{"url": "https://example.com"}},
        {BotName: "http-status-checker", Config: bot.Config{"url": "https://another-site.com"}},
    },
}, scheduler.Interval{Every: 5 * time.Minute})
```

Runner จะรัน step ที่ 2 ก็ต่อเมื่อ step ที่ 1 จบด้วยสถานะ `succeeded` เท่านั้น — ถ้า step ไหนล้มเหลว workflow ทั้งก้อนหยุดทันทีและถูกบันทึกเป็น `failed` พร้อมระบุว่า step ไหนพังเพราะอะไร (ดู `run.Error`)

**ข้อจำกัดที่ควรรู้ก่อนใช้จริงจัง** (มีคอมเมนต์อธิบายไว้ในโค้ดทุกจุด ไม่ใช่แค่ตรงนี้):
- ไม่มี durable execution — process restart กลางทางที่ workflow กำลังรันอยู่ = ความคืบหน้าหายหมด (ต่างจาก Temporal ที่ resume จาก event history ได้ ดู [books/13-temporal/01-workflow-and-activity.md](../../books/13-temporal/01-workflow-and-activity.md))
- ไม่มี retry policy ต่อ step และไม่มี compensation logic อัตโนมัติ — step ล้มเหลวคือจบเลย ไม่ลองใหม่ ไม่ rollback อะไรให้
- Scheduler ยังไม่กันเรื่อง overlapping runs (workflow รอบก่อนยังไม่จบ แต่ถึงเวลา trigger รอบใหม่แล้ว จะรันซ้อนกันได้) — ของจริงแบบ Control-M มี "resource pool"/"skip if running" ป้องกันเรื่องนี้ วางแผนไว้เป็น Phase 3

**ทำไมถึงเพิ่มตอนนี้ก่อน Phase 2.5 ทั้งที่ Temporal ทำเรื่องนี้ได้ดีกว่า**: เพื่อพิสูจน์ pattern "sequential step ผ่าน SubmitAndWait" และมี baseline เปรียบเทียบไว้ตอนย้ายไป Temporal จริง — จะเห็นชัดเจนว่า Temporal แก้ปัญหาอะไรที่เวอร์ชันนี้แก้ไม่ได้ (durable execution, retry, compensation) แทนที่จะอ่านแค่ทฤษฎีเฉยๆ

## บทเรียนที่เจอจริงระหว่างสร้าง (คุ้มค่าเก็บไว้เล่าตอนสัมภาษณ์)

`go test -race` เจอ data race จริงในดราฟต์แรก: `worker.Pool.run()` เดิมรับ `*job.Job` แล้วแก้ field ตรงๆ บน pointer เดียวกับที่ HTTP handler (และ repository) ยังถืออยู่ — สอง goroutine แตะ struct เดียวกันพร้อมกันโดยไม่มี synchronization คือ race แท้ๆ ตามนิยาม ไม่ใช่แค่ theoretical

วิธีแก้ (ดู commit นี้เทียบกับ [books/04-golang/02-concurrency-and-context.md](../../books/04-golang/02-concurrency-and-context.md)): เปลี่ยนให้ **repository เก็บ/คืนค่าเป็น copy เสมอ** (ไม่ปล่อย pointer เข้าไปใน map ที่ share กันออกไปให้ใครถือ) และ **`Pool.Submit` copy `job.Job` เป็น value ก่อนส่งเข้า goroutine** ทำให้ `run()` เป็นเจ้าของ copy ของตัวเองแต่ผู้เดียวตลอดอายุของ job นั้น ไม่มีใครอื่นแตะ struct เดียวกันได้อีก — สื่อสารกับโลกภายนอกผ่าน `repo.Update()` เท่านั้น ซึ่งก็ copy อีกชั้นตอนเก็บ

หลักการที่ได้: **อย่าปล่อย pointer ของ mutable struct ให้ข้าม goroutine boundary โดยไม่มีแผน synchronization ชัดเจน** — ถ้าจะแชร์ state ข้าม goroutine ให้ผ่าน channel/mutex/repository ที่ควบคุม access เดียว ไม่ใช่แชร์ pointer ตรงๆ แล้วหวังว่าจะไม่ชนกัน

## Next Steps

ตามลำดับใน [roadmap](../../FULLSTACK-INFRA-ROADMAP.md):
1. เก็บ Phase 0 ที่เหลือให้ครบ (chromedp bot, error types, PostgreSQL repository, table-driven test)
2. เริ่ม Phase 1 — Next.js dashboard เรียก API ชุดนี้ตรงๆ ได้เลยตั้งแต่วันแรก เพราะ endpoint ที่จำเป็น (`POST /bots/{name}/jobs`, `GET /jobs`, `GET /jobs/{id}`, `GET /jobs/{id}/logs`, `GET /workflow-runs`, `GET /workflow-runs/{id}`) มีครบแล้ว
3. ตอนถึง Phase 2.5 จริงจัง: ย้าย `internal/workflow`/`internal/scheduler` ไปใช้ Temporal แทน — โค้ดปัจจุบันเป็น baseline ที่ดีสำหรับเทียบว่า Temporal แก้ปัญหาอะไรเพิ่มบ้าง (ดูหัวข้อข้อจำกัดด้านบน) มี TODO checklist ละเอียดพร้อม implement แล้วที่ [TEMPORAL-MIGRATION-TODO.md](TEMPORAL-MIGRATION-TODO.md)
