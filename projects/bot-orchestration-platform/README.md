# Bot Orchestration Platform (BOP)

Go core of the flagship project described in [FULLSTACK-INFRA-ROADMAP.md](../../FULLSTACK-INFRA-ROADMAP.md). This is the "core/control plane" that owns bot execution, job state, and log output — the Next.js dashboard ([projects/bop-dashboard/](../bop-dashboard/README.md), Phase 1) is a client of the HTTP API defined here, nothing more.

## สถานะปัจจุบัน

Phase 0 (`Golang: สร้าง Bot Execution Engine`) ทำแล้ว: `Bot` interface, worker pool ที่รันพร้อมกันได้หลาย job แบบมี timeout/cancel, in-memory repository + logger (persist + live-subscribe design พร้อมสลับเป็น PostgreSQL/OpenSearch ทีหลังโดยไม่แก้ caller), bot ตัวอย่างที่ implement interface จริง, และ HTTP API ที่ dashboard ใน Phase 1 จะเรียกใช้ได้ทันที — ทั้งหมด build/vet/test -race ผ่าน และลองยิงจริงทั้ง CLI กับ HTTP server แล้ว

**Phase 2.5 (Temporal migration) ทำเสร็จแล้วก่อนกำหนด**: `internal/workflow` เดิมเป็น engine เขียนเองแบบเบา (ไม่มี durable execution, ไม่มี retry policy ต่อ step, scheduler เขียนเอง) — ตอนนี้ย้ายไปใช้ **Temporal** จริงจังทั้งหมดแล้ว: `internal/workflow` มี `Execute` (Temporal workflow function) + `Activities.RunStep` (Temporal Activity adapter ที่เรียก `bot.Bot.Run`) แทนที่ `Runner`/`RunRepository` เดิม, `internal/scheduler` ถูกลบทิ้งทั้ง package เพราะใช้ **Temporal Schedules** แทน ดูรายละเอียดที่หัวข้อ "Workflow และ Scheduler" ด้านล่าง และ [TEMPORAL-MIGRATION-TODO.md](TEMPORAL-MIGRATION-TODO.md) สำหรับ decision log ของ Open Question ที่ตัดสินใจไว้ระหว่างย้าย

**Backend prerequisite ของ Phase 1 (Next.js dashboard) ทำเสร็จแล้วเช่นกัน**: [projects/bop-dashboard/](../bop-dashboard/README.md) (แยก repo/deploy จาก backend ตั้งใจ) ต้องการ endpoint/กลไกเพิ่มที่ backend ยังไม่มี — เพิ่มครบแล้วทั้งหมด: `GET /bots` (list bot + config schema ผ่าน `bot.Bot.ConfigSchema()` ที่เพิ่มเข้า interface ใหม่), CORS middleware (ดู `dashboardOrigin` ใน `internal/api/http/handler.go`), `GET /jobs/{id}/logs/stream` (SSE — real-time log สำหรับ job เดี่ยวๆ ที่ไม่ผ่าน workflow), `POST /login`/`POST /logout`/`GET /me` (JWT แบบเขียนเอง ดู `internal/auth/jwt.go`), และ normalize JSON ทั้ง API ให้เป็น snake_case สม่ำเสมอกัน (`job.Job`/`logger.Line` เดิมเป็น PascalCase default ของ Go) ดู decision log เต็มๆ ที่ [projects/bop-dashboard/PHASE1-DASHBOARD-TODO.md](../bop-dashboard/PHASE1-DASHBOARD-TODO.md)

ยังไม่ทำ (ของ Phase 0 ที่เหลือ) — ดู checklist เต็มที่ [roadmap Phase 0](../../FULLSTACK-INFRA-ROADMAP.md#phase-0--golang-สร้าง-bot-execution-engine):

- [ ] Bot ตัวที่สองที่ใช้ `chromedp` (headless browser) พิสูจน์ว่า interface รองรับ implementation ที่ซับซ้อนกว่า HTTP GET ธรรมดา
- [ ] แยก error type retryable/non-retryable ด้วย `errors.Is`/`errors.As` (ตอนนี้ error จาก bot แค่ถูกเก็บเป็น string ธรรมดา)
- [ ] PostgreSQL implementation ของ `job.Repository` (ตอนนี้มีแค่ in-memory)
- [ ] Table-driven test ของ logic parse ข้อมูล + `httptest.Server` mock (มี test ของ worker pool/workflow runner/scheduler แล้ว แต่ยังไม่มีของ logic parse ข้อมูลจริงจัง เพราะยังไม่มี bot ที่ parse อะไรซับซ้อน)

## สถาปัตยกรรม

```
cmd/bop/main.go          HTTP API server — ประกอบส่วนที่เกี่ยวกับ job เดี่ยวๆ + Temporal client (อ่านอย่างเดียว)
cmd/bop-worker/main.go   Temporal Worker process แยกต่างหาก — ประกอบ Activities + Temporal Schedule
   │
   ├── internal/bot        Bot interface (Name/ConfigSchema/Run) — ทุก bot implement อันนี้
   │                       ไม่รู้จัก HTTP/DB/logger เลย ConfigSchema() ประกาศ config field
   │                       ให้ dashboard สร้าง dynamic form ได้ (roadmap Phase 1.3)
   ├── internal/job        Job entity + Repository interface (json tag snake_case)
   ├── internal/logger     Logger interface (persist + live-subscribe ในตัวเดียว, json tag
   │                       snake_case) — GET /jobs/{id}/logs/stream (SSE) ใช้ Subscribe()
   ├── internal/auth       JWT (HS256) เขียนเอง — ใช้กับ POST /login (roadmap Phase 1.8)
   ├── internal/worker     Pool — concurrency core สำหรับ job เดี่ยวๆ (POST /bots/{name}/jobs)
   │                       รัน bot ผ่าน semaphore-bounded goroutine (Submit = fire-and-forget,
   │                       SubmitAndWait = รอผลก่อนไปต่อ) — ไม่เกี่ยวกับ workflow อีกต่อไป
   │
   ├── internal/workflow   Workflow/Step (input schema) + Execute (Temporal workflow function)
   │                       + Activities.RunStep (Temporal Activity ที่เรียก bot.Bot.Run) —
   │                       แทนที่ Runner/RunRepository/internal-scheduler เดิมทั้งหมด ดู
   │                       TEMPORAL-MIGRATION-TODO.md สำหรับเหตุผลการย้าย
   │
   ├── internal/bots/httpstatus     Bot ตัวอย่าง (zero external dependency)
   ├── internal/repository/memory   Repository implementation (in-memory) — เก็บ Job เดี่ยวๆ
   ├── internal/logger/memory       Logger implementation (in-memory)
   └── internal/api/http             HTTP handler — สิ่งเดียวที่ dashboard จะคุยด้วย (query
                                      workflow run ผ่าน Temporal client โดยตรง ไม่มี
                                      repository ของตัวเองอีกต่อไป)
```

ทุก dependency ชี้เข้าหา interface ที่ `internal/bot`, `internal/job`, `internal/logger` ไม่ใช่ implementation ตรงๆ — สลับ `internal/repository/memory` เป็น `internal/repository/postgres` ทีหลัง (roadmap Phase 0.4) แก้แค่บรรทัดเดียวใน `cmd/bop/main.go` โค้ดส่วนอื่นไม่ต้องแตะเลย ส่วนการรัน workflow ทั้งหมด (durable execution, retry, scheduling) เป็นหน้าที่ของ Temporal server เอง ไม่ใช่โค้ด BOP อีกต่อไป — BOP เหลือแค่ "ประกอบ" (register bot, ประกาศ workflow function) กับ Temporal เท่านั้น

**หมายเหตุเรื่อง zero external dependency**: ตั้งแต่ Phase 2.5 เป็นต้นไป `go.mod` มี `go.temporal.io/sdk` เป็น dependency ภายนอกตัวแรกของโปรเจกต์นี้ — เป็นข้อยกเว้นที่ตั้งใจ (ดูเหตุผลใน TEMPORAL-MIGRATION-TODO.md ข้อ 2) เพราะ durable workflow execution คือสิ่งที่ reinvent เองไม่คุ้มอีกต่อไป ไม่ใช่ความผิดพลาดของ design เดิม

## รันยังไง

```bash
go build ./...
go vet ./...
go test -race ./...

# 1) เปิด Temporal server + UI (ต้องรันก่อนเสมอ ทั้ง cmd/bop และ cmd/bop-worker ต่อ
#    เข้า localhost:7233) — ดู docker-compose.yml สำหรับ service ที่รัน (Temporal server
#    + PostgreSQL ของ Temporal เอง + Temporal UI ที่ http://localhost:8233)
docker compose up -d

# 2) CLI mode — สั่งรัน bot ตัวเดียวจบแล้ว exit (ไม่ต้องมี server/UI, ไม่ต้องพึ่ง Temporal เลย)
go run ./cmd/bop run --bot=http-status-checker --url=https://example.com

# 3) รัน 2 process คู่กัน (ต้องรันทั้งคู่ให้ workflow ทำงานจริง — ดูเหตุผลการแยก process
#    ที่หัวไฟล์ cmd/bop-worker/main.go)
go run ./cmd/bop-worker   # Temporal Worker — รัน workflow/activity จริง + สร้าง Schedule
go run ./cmd/bop          # HTTP API ที่ :8080 — ให้ dashboard เรียก, อ่านสถานะ workflow ผ่าน Temporal
```

ตัวอย่างเรียก API ด้วย curl (response ทั้งหมดเป็น JSON แบบ snake_case สม่ำเสมอกันทั้ง API):

```bash
curl localhost:8080/bots                                                       # list bot + config schema (ใช้ทำ dynamic form)
curl -X POST localhost:8080/bots/http-status-checker/jobs -d '{"url":"https://example.com"}'
curl localhost:8080/jobs
curl localhost:8080/jobs/<id>
curl localhost:8080/jobs/<id>/logs                # snapshot log ทั้งหมด ณ ตอนนี้ (REST)
curl -N localhost:8080/jobs/<id>/logs/stream      # log แบบ real-time (SSE — ส่ง backfill มาก่อนแล้วค่อย stream ต่อ, -N กัน curl buffer)
curl -X DELETE localhost:8080/jobs/<id>           # cancel ถ้า job ยังรันอยู่

curl localhost:8080/workflow-runs                # ดูว่า scheduled workflow รันไปแล้วกี่รอบ (query ตรงจาก Temporal)
curl localhost:8080/workflow-runs/<workflow-id>   # ดูผลลัพธ์ทีละ step ของ run เดียว (workflow-id ดูได้จาก Temporal UI หรือ response ของ list ด้านบน)

curl -c cookies.txt -X POST localhost:8080/login -d '{"username":"admin","password":"changeme"}'  # ได้ httpOnly cookie กลับมา (เปลี่ยน credential ก่อน deploy จริงเสมอ)
curl -b cookies.txt localhost:8080/me             # เช็คว่า session ยัง valid อยู่ไหม
curl -b cookies.txt -X POST localhost:8080/logout
```

## Workflow และ Scheduler (Temporal)

`cmd/bop-worker/main.go` สร้าง Temporal Schedule ให้ workflow ตัวอย่าง (`check-example-com`) ที่รัน `http-status-checker` เป็น step เดียว ทุก 1 นาที ผ่าน `client.ScheduleClient().Create` — เปิด `docker compose up -d` แล้วรัน `go run ./cmd/bop-worker` ทิ้งไว้สัก 1-2 นาทีจะเห็น run ใหม่โผล่ใน `GET /workflow-runs` เอง (หรือดูตรงๆ ผ่าน Temporal UI ที่ http://localhost:8233) โดยไม่ต้องสั่งอะไรเพิ่ม

เพิ่ม step ที่ 2, 3 เข้าไปได้ทันทีโดยไม่ต้องแก้ `internal/workflow/temporal.go` เลย (แก้แค่ `wf` ที่ประกาศไว้ใน `ensureExampleSchedule`):

```go
wf := workflow.Workflow{
    Name: "check-multiple-sites",
    Steps: []workflow.Step{
        {BotName: "http-status-checker", Config: bot.Config{"url": "https://example.com"}},
        {BotName: "http-status-checker", Config: bot.Config{"url": "https://another-site.com"}},
    },
}
```

`workflow.Execute` (Temporal workflow function) จะรัน step ที่ 2 ก็ต่อเมื่อ step ที่ 1 สำเร็จเท่านั้น — ถ้า step ไหนล้มเหลว workflow ทั้งก้อนหยุดทันที (ดู `internal/workflow/temporal.go`)

**สิ่งที่ Temporal แก้ให้จากเวอร์ชันเบาก่อนหน้า** (ดู [TEMPORAL-MIGRATION-TODO.md](TEMPORAL-MIGRATION-TODO.md) สำหรับรายละเอียดการย้ายและ Open Question ที่ตัดสินใจไว้):
- **Durable execution** — worker process ตายกลางทาง Temporal replay history แล้ว resume ต่อจาก step ที่ยังไม่เสร็จได้เอง ไม่ต้องเริ่มใหม่ทั้ง workflow
- **Retry policy ต่อ step** — ตั้งผ่าน `workflow.ActivityOptions.RetryPolicy` พร้อม `NonRetryableErrorTypes` แยกแยะ error ที่ retry ไปก็ไม่มีประโยชน์ (เช่น `bot.ConfigError` จาก config ที่ user กรอกผิด) ออกจาก error ที่ retry แล้วอาจสำเร็จ (network timeout ชั่วคราว)
- **Temporal Schedules** แทนที่ `internal/scheduler` ที่เขียนเอง — ยังไม่กันเรื่อง overlapping runs แบบละเอียด (Control-M "resource pool"/"skip if running") อยู่ดี แต่ตอนนี้เป็นเรื่องของ `Overlap` policy ที่ Temporal จัดการให้ ไม่ใช่โค้ดที่ BOP ต้องเขียนเอง

**ข้อจำกัดที่ยังเหลืออยู่ (ตั้งใจไม่ทำตอนนี้)**:
- **Compensation logic (Saga)** — ยังไม่มี เพราะ `http-status-checker` ไม่มี side effect ที่ต้อง rollback เลย รอจนมี bot ที่มี side effect จริงจังค่อยออกแบบ (ดู TEMPORAL-MIGRATION-TODO.md ข้อ 4)
- **Log ของ step ในระหว่าง workflow ดึงผ่าน `GET /jobs/{id}/logs` ไม่ได้** — Activity log เข้า in-memory logger ของ `cmd/bop-worker` (คนละ process กับ `cmd/bop` ที่ HTTP handler อ่าน log อยู่) จนกว่าจะมี logger backend ที่ทั้งสอง process แชร์กันได้จริง (PostgreSQL/OpenSearch, roadmap Phase 2.2) — ดูหมายเหตุเต็มที่ `internal/workflow/activity.go`

## บทเรียนที่เจอจริงระหว่างสร้าง (คุ้มค่าเก็บไว้เล่าตอนสัมภาษณ์)

`go test -race` เจอ data race จริงในดราฟต์แรก: `worker.Pool.run()` เดิมรับ `*job.Job` แล้วแก้ field ตรงๆ บน pointer เดียวกับที่ HTTP handler (และ repository) ยังถืออยู่ — สอง goroutine แตะ struct เดียวกันพร้อมกันโดยไม่มี synchronization คือ race แท้ๆ ตามนิยาม ไม่ใช่แค่ theoretical

วิธีแก้ (ดู commit นี้เทียบกับ [books/04-golang/02-concurrency-and-context.md](../../books/04-golang/02-concurrency-and-context.md)): เปลี่ยนให้ **repository เก็บ/คืนค่าเป็น copy เสมอ** (ไม่ปล่อย pointer เข้าไปใน map ที่ share กันออกไปให้ใครถือ) และ **`Pool.Submit` copy `job.Job` เป็น value ก่อนส่งเข้า goroutine** ทำให้ `run()` เป็นเจ้าของ copy ของตัวเองแต่ผู้เดียวตลอดอายุของ job นั้น ไม่มีใครอื่นแตะ struct เดียวกันได้อีก — สื่อสารกับโลกภายนอกผ่าน `repo.Update()` เท่านั้น ซึ่งก็ copy อีกชั้นตอนเก็บ

หลักการที่ได้: **อย่าปล่อย pointer ของ mutable struct ให้ข้าม goroutine boundary โดยไม่มีแผน synchronization ชัดเจน** — ถ้าจะแชร์ state ข้าม goroutine ให้ผ่าน channel/mutex/repository ที่ควบคุม access เดียว ไม่ใช่แชร์ pointer ตรงๆ แล้วหวังว่าจะไม่ชนกัน

## Next Steps

ตามลำดับใน [roadmap](../../FULLSTACK-INFRA-ROADMAP.md):
1. เก็บ Phase 0 ที่เหลือให้ครบ (chromedp bot, error types, PostgreSQL repository, table-driven test)
2. **Phase 2.5 (Temporal migration) ทำเสร็จแล้ว** — `internal/workflow` ย้ายไปใช้ Temporal จริงจังทั้งหมด (`internal/scheduler` ถูกลบทิ้ง) ดู decision log เต็มๆ ที่ [TEMPORAL-MIGRATION-TODO.md](TEMPORAL-MIGRATION-TODO.md) — สิ่งที่เหลือค้างจากงานนี้: (1) ยังไม่เคยรัน end-to-end จริงกับ Temporal server (build/vet/test -race ผ่านหมด แต่ยังไม่ได้ยิง `docker compose up -d` แล้วดู schedule trigger จริงในเครื่องนี้ เพราะติด Docker permission ตอนที่เขียนโค้ดนี้) (2) compensation logic ยังไม่มี รอ bot ที่มี side effect จริงจังก่อน (3) log ของ step ใน workflow ยังดึงผ่าน HTTP ไม่ได้จนกว่าจะมี logger backend ที่แชร์ข้าม process ได้ (PostgreSQL/OpenSearch, Phase 2.2)
3. **Phase 1 (Next.js dashboard) ทำแล้วเช่นกัน** — ทั้ง backend prerequisite (endpoint ใหม่ครบตามที่ระบุด้านบน) และตัว dashboard เองที่ [projects/bop-dashboard/](../bop-dashboard/README.md) (implement ครบ 1.1-1.9 ตาม [PHASE1-DASHBOARD-TODO.md](../bop-dashboard/PHASE1-DASHBOARD-TODO.md)) — `go test -race ./...` ผ่านหมดรวม `internal/api/http` (smoke test เต็มผ่าน `httptest` ครอบคลุม bots/jobs/auth/CORS โดยไม่ต้องพึ่ง Docker/Temporal เลย) ฝั่ง dashboard เองก็ typecheck/lint/test/build ผ่านหมดเช่นกัน — **ที่ยังไม่ verify คือ end-to-end จริงกับ Temporal server** (เหตุผลเดียวกับข้อ 2: ติด Docker permission ในเครื่องที่เขียนโค้ดนี้)
