# Bot Orchestration Platform (BOP)

Go core of the flagship project described in [FULLSTACK-INFRA-ROADMAP.md](../../FULLSTACK-INFRA-ROADMAP.md). This is the "core/control plane" that owns bot execution, job state, and log output — the Next.js dashboard ([projects/bop-dashboard/](../bop-dashboard/README.md), Phase 1) is a client of the HTTP API defined here, nothing more.

> **อยากรู้ภาพรวม feature ทั้งหมดตอนนี้แบบเร็วๆ**: ดู [FEATURES.md](FEATURES.md) (สรุป feature ปัจจุบัน + คำแนะนำสิ่งที่ควรทำต่อ) — README ไฟล์นี้เน้น "รันยังไง/สถาปัตยกรรม/API reference" ส่วน FEATURES.md เน้น "มีอะไรแล้วบ้าง/ควรทำอะไรต่อ" ไม่ซ้ำเนื้อหากันตั้งใจ

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
   ├── internal/schedule   Definition + Repository — เก็บ "workflow นี้ทำอะไร" (ชื่อ+steps)
   │                       ของแต่ละ Temporal Schedule ไว้เอง (Temporal เก็บแค่ "เมื่อไหร่/
   │                       สถานะ" ให้แล้ว) ใช้กับ schedule CRUD API ด้านล่าง
   │
   ├── internal/bots/httpstatus     Bot ตัวอย่าง (zero external dependency)
   ├── internal/bots/externaljob    Bot ที่ shell out ไปเรียก Go binary ภายนอก (jobcode) ผ่าน os/exec
   ├── internal/repository/memory   Repository implementation (in-memory) — เก็บ Job เดี่ยวๆ
   │                                 + Schedule definition เดี่ยวๆ
   ├── internal/logger/memory       Logger implementation (in-memory)
   └── internal/api/http             HTTP handler — สิ่งเดียวที่ dashboard จะคุยด้วย (query
                                      workflow run + จัดการ Schedule ผ่าน Temporal client
                                      โดยตรง ไม่มี repository ของตัวเองสำหรับ workflow run)
```

**3 ทางเลือกในการรัน job ที่ไม่ใช่ local Go bot** (ดู [FEATURES.md](FEATURES.md) สำหรับตารางเทียบสั้นๆ):
- `internal/bots/externaljob` — shell out ในเครื่อง/container เดียวกับ BOP ([EXTERNAL-JOB-BOT-TODO.md](EXTERNAL-JOB-BOT-TODO.md))
- `workflow.Step.TaskQueue` — route ไป Temporal worker คนละ repo/process ([REMOTE-JOB-WORKER-TODO.md](REMOTE-JOB-WORKER-TODO.md), reference implementation ที่ [projects/remote-job-worker/](../remote-job-worker/README.md))
- `internal/workflow/k8sjob.go` (`workflow.Step.K8sJob`) — สั่งสร้าง Kubernetes Job ต่อการรันหนึ่งครั้ง ([K8S-JOB-HYBRID-EXECUTION-TODO.md](K8S-JOB-HYBRID-EXECUTION-TODO.md))

ทุก dependency ชี้เข้าหา interface ที่ `internal/bot`, `internal/job`, `internal/logger` ไม่ใช่ implementation ตรงๆ — สลับ `internal/repository/memory` เป็น `internal/repository/postgres` ทีหลัง (roadmap Phase 0.4) แก้แค่บรรทัดเดียวใน `cmd/bop/main.go` โค้ดส่วนอื่นไม่ต้องแตะเลย ส่วนการรัน workflow ทั้งหมด (durable execution, retry, scheduling) เป็นหน้าที่ของ Temporal server เอง ไม่ใช่โค้ด BOP อีกต่อไป — BOP เหลือแค่ "ประกอบ" (register bot, ประกาศ workflow function) กับ Temporal เท่านั้น

**หมายเหตุเรื่อง zero external dependency**: ตั้งแต่ Phase 2.5 เป็นต้นไป `go.mod` มี `go.temporal.io/sdk` เป็น dependency ภายนอกตัวแรกของโปรเจกต์นี้ — เป็นข้อยกเว้นที่ตั้งใจ (ดูเหตุผลใน TEMPORAL-MIGRATION-TODO.md ข้อ 2) เพราะ durable workflow execution คือสิ่งที่ reinvent เองไม่คุ้มอีกต่อไป ไม่ใช่ความผิดพลาดของ design เดิม — ต่อมาเพิ่ม `k8s.io/client-go` (+ `k8s.io/api`, `k8s.io/apimachinery` transitively) เข้ามาอีกตัวสำหรับ K8s Job hybrid execution (`internal/workflow/k8sjob.go`) ด้วยเหตุผลเดียวกัน: เขียน Kubernetes API client เองไม่คุ้มเทียบกับ SDK ที่เป็นมาตรฐานอยู่แล้ว

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
curl -X POST 'localhost:8080/bots/external-job/jobs?timeout_seconds=120' -d '{"jobcode":"SEND_REPORT","extra_args":"--dry-run"}'  # timeout_seconds เป็น query param (optional) — ดูหัวข้อ "Bot: external-job" ด้านล่าง
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

# Schedule CRUD (ดูหัวข้อ "จัดการ Schedule" ด้านล่างสำหรับรายละเอียดเต็ม)
curl -X POST localhost:8080/schedules -d '{
  "id": "check-example-com-v2",
  "workflow_name": "check-example-com-v2",
  "steps": [{"bot_name": "http-status-checker", "config": {"url": "https://example.com"}}],
  "spec": {"interval_seconds": 60}
}'
curl localhost:8080/schedules                              # list schedule ทั้งหมด
curl localhost:8080/schedules/check-example-com-v2          # ดูรายละเอียด (spec, paused, recent runs, steps)
curl -X PUT localhost:8080/schedules/check-example-com-v2 -d '{...}'   # แทนที่ทั้งก้อน (full replace)
curl -X POST localhost:8080/schedules/check-example-com-v2/pause
curl -X POST localhost:8080/schedules/check-example-com-v2/unpause
curl -X POST localhost:8080/schedules/check-example-com-v2/trigger     # สั่งรันทันที นอกตารางเวลาปกติ
curl -X POST localhost:8080/schedules/check-example-com-v2/backfill -d '{"start_time":"2026-01-01T00:00:00Z","end_time":"2026-01-02T00:00:00Z"}'
curl -X DELETE localhost:8080/schedules/check-example-com-v2
```

## Bot: external-job (shell out ไปเรียก binary ภายนอก)

`internal/bots/externaljob` คือ bot ที่ห่อ Go binary แยกต่างหาก (compile ไว้แล้ว ไม่ใช่ source code ส่วนหนึ่งของ repo นี้) ที่มี "jobcode" หลายตัวรวมอยู่ในไฟล์เดียว (`./your-binary --jobcode=SEND_REPORT`) ให้สั่งรัน/schedule ผ่าน BOP ได้เหมือน bot อื่นๆ **โดยไม่แก้โค้ดของ binary เดิมเลยแม้แต่บรรทัดเดียว** — ดู decision log เต็มๆ ที่ [EXTERNAL-JOB-BOT-TODO.md](EXTERNAL-JOB-BOT-TODO.md)

- **Generic bot เดียว ไม่ใช่ 1 bot ต่อ 1 jobcode** — `ConfigSchema()` มี `jobcode` (required, free text) และ `extra_args` (optional, free text คั่นด้วยช่องว่าง ต่อท้าย `--jobcode=<code>` เป็น argument เพิ่ม) เพิ่ม jobcode ใหม่ในไฟล์ binary เดิมทีหลังไม่ต้อง build/deploy BOP ใหม่เลย
- **Path ของ binary** — hardcode เป็น `defaultBinaryPath` ใน `internal/bots/externaljob/externaljob.go` (ปัจจุบันเป็น placeholder เพราะยังไม่มี binary จริงให้ทดสอบ) — ต้องแก้ค่านี้ตรงๆ ให้ตรง path จริงบนเครื่อง/container ที่รัน `cmd/bop` และ `cmd/bop-worker` ก่อนใช้งานจริง และต้อง copy binary เข้า Dockerfile ของทั้งสอง process ตอน containerize (roadmap Phase 2.1 — ยังไม่ทำ)
- **Exit code semantics** — exit code `0` = สำเร็จ, อื่นๆ = ล้มเหลวด้วย error ธรรมดา (retryable — Temporal/worker pool จะ retry ได้ตาม RetryPolicy ปกติ เพราะถือว่า exit code ที่ไม่ใช่ 0 อาจเกิดจากปัญหาชั่วคราวของ binary เอง) ต่างจาก jobcode ที่ไม่ได้ระบุมาเลยซึ่งเป็น `bot.ConfigError` (non-retryable)
- **Log streaming** — stdout/stderr ของ process ลูกถูกอ่านคนละ goroutine พร้อมกัน (กัน deadlock ตอน pipe เต็ม) แล้วส่งเข้า `log()` แบบ real-time ทันที บรรทัดจาก stderr มี prefix `[stderr] ` กำกับ
- **Cancellation ฆ่าทั้ง process group** — ตั้ง `Setpgid: true` แล้ว kill ทั้งกลุ่ม (`syscall.Kill(-pid, ...)`) ไม่ใช่แค่ process ลูกโดยตรง เพราะถ้า binary เปิด subprocess ของตัวเองต่ออีกที (เช่น shell script ที่เรียก `sleep`/`curl`) การ kill แค่ตัวแรกจะทิ้ง grandchild เป็น orphan ที่ยังถือ stdout/stderr pipe ค้างไว้ ทำให้ log streaming ค้างรอ EOF ไม่มีวันมาถึง (พิสูจน์จริงด้วย test `TestRun_ContextCancelKillsProcess`)

### Timeout ต่อ job/step (per-job/per-step timeout)

เดิมทั้งระบบใช้ timeout เดียวตายตัว 30 วินาที (`worker.NewPool` ใน `cmd/bop/main.go`, `ActivityOptions.StartToCloseTimeout` ใน `internal/workflow/temporal.go`) ทุก bot เหมือนกันหมด — ไม่พอสำหรับ `external-job` ที่ jobcode บางตัวอาจใช้เวลานานกว่านั้นมาก จึงเพิ่มความสามารถกำหนด timeout ได้ต่อ job/step แต่ละครั้งแทน (ยังคง 30 วินาทีเป็นค่า default ถ้าไม่ได้ระบุมา — bot เดิมอย่าง `http-status-checker` ไม่กระทบเลย):

- **Job เดี่ยว** (`POST /bots/{name}/jobs`) — ระบุผ่าน query parameter `timeout_seconds` (จำนวนเต็มบวก หน่วยวินาที) เช่น `POST /bots/external-job/jobs?timeout_seconds=120` — เลือกใช้ query param แทนที่จะห่อ JSON body ใหม่ เพราะ body เดิมคือ `bot.Config` ล้วนๆ (flat map) อยู่แล้ว การเปลี่ยนรูปแบบ body จะทำลาย wire format ที่ dashboard เรียกอยู่
- **Workflow step / schedule** — `workflow.Step` มี field `timeout_seconds` (optional) เพิ่มมาแล้ว ใช้ได้ทันทีใน `POST /schedules` เช่น `{"bot_name": "external-job", "config": {...}, "timeout_seconds": 300}`
- ทั้งสองทางเก็บค่าเป็น `int` (`job.Job.TimeoutSeconds`/`workflow.Step.TimeoutSeconds`) — `0` (ไม่ได้ระบุ) แปลว่า "ใช้ default ของระบบ" เสมอ

## รัน step ผ่าน remote Temporal worker (task queue แยกกัน คนละ repo)

ทางเลือกที่สอง (คู่กับ `internal/bots/externaljob` ด้านบน) สำหรับ job/binary ที่อยู่ **คนละ repository และ deploy คนละจังหวะกับ BOP จริงๆ** — แทนที่จะ shell out ในเครื่อง/container เดียวกัน ให้ repo นั้นรัน Temporal Worker ของตัวเอง (embed `go.temporal.io/sdk` เอง — repo นั้นแก้โค้ดได้ ต่างจาก `externaljob` ที่ตั้งใจไม่แตะ binary เดิม) แล้ว BOP route ไปหา task queue นั้นแทนที่จะรันผ่าน local `bot.Bot` registry เลย รายละเอียด contract เต็มๆ ที่ repo ภายนอกต้อง implement อยู่ที่ [REMOTE-JOB-WORKER-TODO.md](REMOTE-JOB-WORKER-TODO.md)

- **`workflow.Step.TaskQueue`** (optional) — ค่าว่าง (default) ใช้ local `Activities.RunStep` ตามเดิมทุกประการ (lookup `bot.Bot` จาก registry ของ `cmd/bop-worker`) ถ้าระบุ Temporal จะส่ง Activity call ไปยัง worker process ที่ poll task queue ชื่อนั้นแทน (คนละ repo/binary/แม้แต่คนละภาษาก็ได้) — กลไกนี้เป็นของ Temporal เองล้วนๆ (`workflow.ActivityOptions.TaskQueue`) ไม่ได้เขียน routing logic เพิ่มเองเลยสักบรรทัด
- **`POST /workflow-runs`** — endpoint ใหม่ รัน step (1 หรือหลาย step) ทันทีแบบ ad-hoc โดยไม่ต้องสร้าง schedule ก่อน ใช้ได้ทั้งกับ step local และ step ที่มี `task_queue` (เพราะ routing ตัดสินใจต่อ step อยู่แล้ว) — คือช่องทางที่ทำให้ "job เดี่ยว" route ไป remote task queue ได้ (ต่างจาก `POST /bots/{name}/jobs` ที่ผ่าน `worker.Pool` ล้วนๆ ไม่มี Temporal เกี่ยวข้องเลย จึง route ข้าม task queue ไม่ได้)

```bash
curl -X POST localhost:8080/workflow-runs -d '{
  "workflow_name": "send-report-remote",
  "steps": [{"bot_name": "external-report-job", "config": {"jobcode": "SEND_REPORT"}, "task_queue": "external-job-queue"}]
}'
# ได้ {"workflow_id": "...", "run_id": "..."} กลับมาทันที (fire-and-forget เหมือน POST /bots/{name}/jobs)
curl localhost:8080/workflow-runs/<workflow_id>   # poll สถานะ/ผลลัพธ์ผ่าน endpoint เดิมที่มีอยู่แล้ว ไม่ต้องเพิ่มอะไรใหม่
```

**สถานะ**: `POST /workflow-runs` ทดสอบ end-to-end จริงแล้วกับ Temporal server จริง (แก้ปัญหา Docker permission ได้แล้ว — เพิ่ม user เข้า `docker` group) — สร้าง ad-hoc run, เห็น step รันจริงและคืนผลลัพธ์ถูกต้อง (`GET /workflow-runs/{id}` เห็น `steps[]` ครบ) แถมยังเจอ transient network error จริงระหว่างทดสอบ (`i/o timeout` ตอนต่อ `example.com`) แล้ว Temporal retry อัตโนมัติจนสำเร็จ — พิสูจน์ RetryPolicy ทำงานถูกต้องกับ failure จริง ไม่ใช่แค่ทฤษฎี — **ส่วนที่ยังไม่ได้ verify**: การ route activity ข้าม task queue ไปยัง worker อีกตัว (`Step.TaskQueue`) ต้องมี worker ตัวที่สอง poll queue คนละชื่อกันจริง (เช่นรัน `projects/remote-job-worker/` คู่กัน) ยังไม่ได้ทดสอบชุดนี้

## รัน step ผ่าน Kubernetes Job (ephemeral container ต่อการรันหนึ่งครั้ง)

ทางเลือกที่สาม (คู่กับ `externaljob` และ remote Temporal worker ด้านบน) — ดู design doc เต็มๆ พร้อมตารางเทียบทั้ง 3 ทางเลือกที่ [K8S-JOB-HYBRID-EXECUTION-TODO.md](K8S-JOB-HYBRID-EXECUTION-TODO.md) แทนที่จะมี worker รันค้างไว้ตลอด (poll task queue) ให้ **Temporal Activity ของ `cmd/bop-worker` เองสั่งสร้าง Kubernetes `Job` (ephemeral container) ต่อการรันหนึ่งครั้ง** แล้วรอดูผล — container image เป็นอะไรก็ได้ ไม่ต้อง embed Temporal SDK เข้าไปในโค้ดของ job เองเลยด้วยซ้ำ (ต่างจาก remote worker) แลกกับ cold-start latency ที่สูงกว่า

- **`workflow.Step.K8sJob *K8sJobSpec`** (optional) — nil = รันผ่าน `bot.Bot` registry/remote task queue ตามเดิมทุกประการ ถ้าไม่ nil จะข้ามทั้งสองทางนั้นไปเลย แล้วเรียก activity คนละตัว/คนละ registry (`K8sJobActivityName = "RunKubernetesJob"`) แทน — `K8sJobSpec` มี `image` (required), `command`/`args`/`env` (optional), `namespace` (optional, default `bop-jobs`)
- **`internal/workflow/k8sjob.go`** (`K8sJobActivities.RunKubernetesJob`) — สร้าง `batchv1.Job` ด้วย **`BackoffLimit: 0` เสมอ** (สำคัญมาก — ให้ Temporal `RetryPolicy` เป็นเจ้าของ retry decision แต่ผู้เดียว ไม่ให้ K8s retry ซ้อนอีกชั้น) poll สถานะเป็นระยะพร้อม `activity.RecordHeartbeat`, stream log ของ Pod เข้า BOP logger ทันทีที่ Job จบ (ก่อน TTL controller ลบ Job ทิ้ง), และลบ Job ทิ้งทันทีถ้า activity ถูกยกเลิก/timeout (กัน orphan Job ทำงานต่อในเบื้องหลัง)
- **Register แบบ best-effort ใน `cmd/bop-worker/main.go`** — ลอง `rest.InClusterConfig()` ก่อน ถ้าล้มเหลว (เช่น รันบนเครื่อง dev ธรรมดา ไม่ได้อยู่ใน K8s Pod) แค่ log คำเตือนแล้วข้ามไป ไม่ `log.Fatalf` ปิด process ทั้งตัว — bot local/remote task queue อื่นๆ ยังใช้งานได้ปกติ

```json
// ตัวอย่าง step ที่มี K8sJob (ใช้ใน POST /workflow-runs หรือ POST /schedules ก็ได้)
{"bot_name": "generate-report", "k8s_job": {"image": "registry.example.com/report-job:v1", "args": ["--jobcode=SEND_REPORT"]}}
```

**สถานะ**: unit test ผ่านหมด (`internal/workflow/k8sjob_test.go`, mock ด้วย `k8s.io/client-go/kubernetes/fake`) แต่**ยังไม่เคยทดสอบกับ K8s cluster จริงเลย** — รอ Phase 2.1 (containerize BOP เข้า K8s) เสร็จก่อน ดู Open Questions ที่ยังไม่ได้ตัดสินใจ (image registry, RBAC manifest, resource limit, timeout ของ image pull) ที่ K8S-JOB-HYBRID-EXECUTION-TODO.md

## จัดการ Schedule (CRUD + manual trigger — เทียบเท่า Control-M job management)

`internal/api/http/schedule_handler.go` ห่อ `client.ScheduleClient` ของ Temporal SDK เป็น HTTP endpoint ทั้งชุด — ไม่ได้เขียน scheduling logic เองเลยสักบรรทัด (Temporal จัดการ cron parsing, calendar rule, overlap policy, catchup window ให้หมดแล้ว) BOP เก็บเพิ่มแค่ "workflow definition" (ชื่อ + steps) ไว้ใน `internal/schedule` (in-memory, เหตุผลที่แยกเก็บเองแทนที่จะ decode จาก Temporal ตรงๆ อยู่ใน comment ของไฟล์นั้น)

| Endpoint | เทียบ Control-M | หมายเหตุ |
|---|---|---|
| `POST /schedules` | New Job Definition | body: `id`, `workflow_name`, `steps[]`, `spec` (`interval_seconds` หรือ `cron_expression` เลือกอย่างใดอย่างหนึ่ง), `overlap` (optional) |
| `GET /schedules` | Job list | เร็ว เพราะไม่ Describe ทีละตัว |
| `GET /schedules/{id}` | View Job | รวม overlap policy, ประวัติการรันล่าสุด, next run time, steps |
| `PUT /schedules/{id}` | Edit Job | full replace ทั้งก้อน (ไม่ใช่ partial patch) |
| `DELETE /schedules/{id}` | Delete Job | ลบถาวรทั้งฝั่ง Temporal และ definition ของ BOP |
| `POST /schedules/{id}/pause` | Hold | หยุดชั่วคราว ไม่ลบ |
| `POST /schedules/{id}/unpause` | Free | เริ่มใหม่หลัง pause |
| `POST /schedules/{id}/trigger` | **Order Now / Run Now** | สั่งรันทันที นอกตารางเวลาปกติ (รอบถัดไปตามตารางยังมาตามเดิม) |
| `POST /schedules/{id}/backfill` | Rerun | รันย้อนหลังตามช่วงเวลาที่ระบุ ราวกับเวลานั้นเพิ่งผ่านไปตอนนี้ |

**ข้อจำกัดที่ต้องรู้**: `internal/schedule` เป็น in-memory ล้วนๆ (เหมือน `job.Repository` ก่อนต่อ PostgreSQL จริงใน Phase 0.4) — restart `cmd/bop` แล้ว field `steps` ใน `GET /schedules/{id}` จะหายไป (Temporal เองยังรัน schedule ต่อปกติ ไม่กระทบการทำงานจริง แค่ BOP ลืมว่า workflow นั้น "ทำอะไร") เช่นเดียวกับ `check-example-com-schedule` ที่ `cmd/bop-worker` สร้างตรงผ่าน SDK (ไม่ผ่าน API นี้) จะไม่มี steps ให้ดูเลยตั้งแต่ต้น — ดู comment เต็มที่ `internal/schedule/schedule.go`

**สถานะ**: ทดสอบ end-to-end จริงแล้วครบทั้ง lifecycle (create → get → pause → **trigger** → unpause → delete) กับ Temporal server จริง — `trigger` ยืนยันแล้วว่าสั่งรันจริงได้แม้ schedule กำลัง paused อยู่ (เห็น workflow run ใหม่โผล่ใน `GET /workflow-runs` ทันที) ยังไม่ได้ทดสอบ `backfill` โดยเฉพาะ (path เดียวกับ trigger แค่ระบุช่วงเวลาย้อนหลัง คาดว่าทำงานเหมือนกัน แต่ยังไม่ได้ยืนยันจริง)

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

สิ่งที่ทำเสร็จแล้วสรุปสั้นๆ (รายละเอียดเต็มดู [FEATURES.md](FEATURES.md)): Phase 0 core engine, Phase 2.5 Temporal migration, Phase 1 backend prerequisite + dashboard, Schedule CRUD เต็มชุด, 3 ทางเลือกรัน external job (`externaljob`/remote task queue/K8s Job hybrid), `POST /workflow-runs` (ad-hoc run), per-job/per-step timeout — ดูหัวข้อต่างๆ ด้านบนของไฟล์นี้สำหรับรายละเอียดแต่ละอัน

**ช่องว่างที่สำคัญที่สุดตอนนี้ (เรียงตาม priority ใน [FEATURES.md](FEATURES.md))**:
1. **PostgreSQL persistence** — `job.Repository`/`schedule.Repository` ยังเป็น in-memory ทั้งคู่ (Phase 0.4) เป็น prerequisite ของแทบทุกอย่างถัดจากนี้ รวมถึง Phase 3 enterprise feature ด้านล่าง
2. **Job Dependency (DAG)** และ **Observability พื้นฐาน** (Phase 2.2) — ยังไม่มีเลยทั้งคู่
3. **Temporal server จริง verify แล้ว** (แก้ปัญหา Docker permission ได้แล้ว — เพิ่ม user เข้า `docker` group) ครอบคลุม: job เดี่ยว + SSE log stream, auth เต็ม flow, cancel job, Schedule CRUD เต็ม lifecycle, `POST /workflow-runs` (รวมเจอ+เห็น retry ของ transient network error จริง) — **ที่ยังไม่ verify**: route ข้าม task queue จริง (ต้องมี worker ตัวที่สอง), และ **K8s cluster ยังไม่มีเลย** (K8s Job hybrid execution ยังผ่านแค่ unit test ด้วย fake clientset — ดู [KUBERNETES-DEPLOYMENT-TODO.md](KUBERNETES-DEPLOYMENT-TODO.md))

**Design doc ที่เขียนไว้ล่วงหน้า รอ implement**:
- [PHASE3-ENTERPRISE-FEATURES-TODO.md](PHASE3-ENTERPRISE-FEATURES-TODO.md) — Condition-Based Triggering, Resource Pools, Flow Control, Approval Gate, Gantt View, Alert Triage, Job Versioning (ฝั่ง backend เต็มรูปแบบ พร้อมลำดับที่แนะนำให้ทำ — **ควรรอ PostgreSQL persistence เสร็จก่อน**)
- [projects/bop-dashboard/PHASE3-DASHBOARD-TODO.md](../bop-dashboard/PHASE3-DASHBOARD-TODO.md) — เอกสารคู่กันฝั่ง frontend
