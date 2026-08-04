# TODO: ย้าย Workflow/Scheduler ไปใช้ Temporal (Roadmap Phase 2.5)

> **สถานะ: implement เสร็จแล้ว** (checklist ด้านล่างติ๊กครบยกเว้นข้อที่บอกไว้ชัดว่าตั้งใจข้าม)
> เหลือแค่ **ยังไม่เคยรัน end-to-end จริงกับ Temporal server ในเครื่องนี้** — โค้ด build/vet/`test -race` ผ่านหมดแล้ว (รวม test ที่ใช้ `testsuite.WorkflowTestSuite`/`TestActivityEnvironment` mock กลไก workflow/activity) แต่ session ที่ implement ติด Docker permission (`permission denied` ที่ `/var/run/docker.sock`, user ไม่ได้อยู่ใน `docker` group ที่ระดับระบบ ไม่มี passwordless sudo ให้แก้เอง) จึงไม่ได้ `docker compose up -d` แล้วยิง `go run ./cmd/bop-worker` + `go run ./cmd/bop` ทดสอบจริงสักครั้ง — **สิ่งแรกที่ session ถัดไปควรทำคือรัน end-to-end ให้ครบ** (ดู README.md หัวข้อ "รันยังไง") ก่อนเชื่อว่าใช้งานได้จริง

เอกสารนี้เขียนไว้ให้ session อื่นหยิบไป implement ต่อได้เลยโดยไม่ต้องมีบริบทจากการคุยครั้งนี้ — อ่านหัวข้อ "บริบท" ด้านล่างก่อนเริ่มลงมือ

## บริบท

โปรเจกต์นี้คือ **Bot Orchestration Platform (BOP)** ดูภาพรวมที่ [README.md](README.md) — ตอนนี้มี workflow/scheduler แบบ **เวอร์ชันเบา** ที่สร้างไว้ก่อนกำหนด (ปกติวางแผนไว้เป็น Phase 2.5) เพื่อพิสูจน์ pattern และมี baseline ไว้เทียบกับ Temporal:

- `internal/workflow/` — `Workflow`/`Step`/`Run` + `Runner.Trigger` ที่รัน step ตามลำดับผ่าน `worker.Pool.SubmitAndWait`
- `internal/scheduler/` — `Schedule` interface + `Interval` implementation ที่คอย trigger workflow ตามเวลา (เทียบเท่า cron)

**งานนี้คือการย้ายทั้งสอง package นี้ไปใช้ Temporal จริงจัง** ตามที่วางแผนไว้ใน [FULLSTACK-INFRA-ROADMAP.md Phase 2.5](../../FULLSTACK-INFRA-ROADMAP.md#25-queue-และ-workflow-orchestration-จุดที่-domain-นี้-mapping-ตรงที่สุด)

## อ่านก่อนเริ่ม

1. [README.md](README.md) หัวข้อ **"Workflow และ Scheduler"** — list ข้อจำกัดของเวอร์ชันปัจจุบันไว้ชัดเจนแล้ว (ไม่มี durable execution, ไม่มี retry policy ต่อ step, ไม่มี compensation, scheduler ไม่กัน overlapping run) **ข้อจำกัดเหล่านี้คือ requirement ที่เวอร์ชัน Temporal ต้องแก้ให้ได้** ไม่ใช่แค่ background
2. โค้ดปัจจุบันที่เกี่ยวข้องโดยตรง: `internal/workflow/workflow.go`, `internal/workflow/runner.go`, `internal/scheduler/scheduler.go`, `internal/worker/pool.go` (`SubmitAndWait`), `internal/api/http/handler.go` (endpoint `/workflow-runs`), `cmd/bop/main.go` (จุด wiring)
3. ทฤษฎี Temporal เต็มๆ: [books/13-temporal/01-workflow-and-activity.md](../../books/13-temporal/01-workflow-and-activity.md), [books/13-temporal/02-retry-and-compensation.md](../../books/13-temporal/02-retry-and-compensation.md)
4. Saga pattern: [books/11-microservices/04-saga-and-resilience.md](../../books/11-microservices/04-saga-and-resilience.md)

## เป้าหมาย

ย้ายจาก custom workflow engine (ที่เขียนเอง, ไม่มี durable execution) ไปใช้ Temporal (durable execution จริง, retry policy ประกาศได้ต่อ step, compensation logic เขียนได้เป็นธรรมชาติในโค้ด workflow, ไม่ต้องกังวลเรื่อง process restart กลางทาง) — โดยที่ HTTP API (`/workflow-runs`) และหน้า dashboard (Phase 1) ยังใช้งานได้เหมือนเดิมหรือดีขึ้น ไม่ใช่แย่ลง

## TODO Checklist

### 1. Setup Temporal Server (local dev)

- [x] เพิ่ม Temporal server + Temporal UI + PostgreSQL เข้า `docker-compose.yml` — แยก instance กับ PostgreSQL ที่ BOP จะใช้เก็บ job history ใน Phase 0.4 (ตัดสินใจแยกไปเลย ไม่แชร์ database) และตั้งใจให้ scope ของ compose file นี้แคบมาก (แค่ Temporal เอง) ไม่ได้ containerize ตัว BOP app ด้วย — นั่นคือ Phase 2.1 เต็มรูปแบบที่ยังไม่เริ่ม
- [ ] **ยังไม่ยืนยัน** ว่า Temporal server รันได้จริง เปิด Temporal UI (port 8233) เห็น namespace `default` — session ที่เขียน docker-compose.yml ติด Docker permission ในเครื่อง (ดูหมายเหตุสถานะที่หัวไฟล์) ยังไม่เคยรัน `docker compose up -d` สำเร็จเลยสักครั้ง **ต้องทำเป็นอย่างแรกก่อนเชื่อว่า Phase นี้ใช้งานได้จริง**

### 2. เพิ่ม Temporal Go SDK

- [x] `go get go.temporal.io/sdk` (v1.47.0) — จุดแรกที่ BOP มี external dependency ตามที่คาดไว้ (Go toolchain ของ go.mod ต้องขยับตามเป็น go 1.25.4 ด้วย เพราะ SDK เองต้องการ Go version นั้น ไม่ใช่การ bump เกินจำเป็น)
- [x] ตัดสินใจ Open Question ข้อ 4: **แยก Temporal Worker เป็น process ใหม่ `cmd/bop-worker/`** (เจ้าของ repo เลือกเอง หลังจากเห็น tradeoff เรื่อง production realism vs local dev ที่ต้องรัน 2 process)

### 3. ออกแบบ Workflow ด้วย Temporal

- [x] ตัดสินใจ Open Question ข้อ 1: **`workflow.Step`/`workflow.Workflow` เก็บไว้ต่อ** เป็น input schema — ตัวรัน workflow จริงย้ายจาก `Runner.Trigger` (ลบแล้ว) ไปเป็น `workflow.Execute` (`internal/workflow/temporal.go`) ที่ loop ผ่าน `Steps` เรียก `workflow.ExecuteActivity` ทีละตัว
- [x] แปลง `bot.Bot.Run` ให้ถูกเรียกจาก Temporal Activity — `internal/workflow/activity.go` (`Activities.RunStep`) เป็น adapter บางๆ ตามที่ตั้งใจไว้ ไม่ได้แก้ bot logic เดิมเลย (ยกเว้นเพิ่ม `bot.NewConfigError(...)` 1 บรรทัดใน `httpstatus.go` — ดูข้อถัดไป)
- [x] Retry Policy + `NonRetryableErrorTypes` — เพิ่ม `bot.ConfigError` (type ใหม่ใน `internal/bot/errors.go`) ให้ `httpstatus` คืนกลับแทน error ธรรมดา แล้ว `Activities.RunStep` ห่อเป็น `temporal.NewApplicationErrorWithCause(..., "ConfigError", ...)` ให้ตรงกับ `nonRetryableErrorTypes` ใน `temporal.go` — มี test ยืนยันพฤติกรรมนี้ตรงๆ ที่ `internal/workflow/activity_test.go` (`TestRunStep_ConfigErrorIsNonRetryable`)

### 4. Compensation Logic (Saga)

- [ ] **ยังไม่ทำตามแผนเดิม** — bot ตัวอย่าง (`http-status-checker`) ยังไม่มี side effect ที่ต้อง rollback รอจนมี bot ที่มี side effect จริงจังก่อนค่อยออกแบบ (คำแนะนำเดิมในเอกสารนี้ยังใช้ได้ ไม่มีอะไรเปลี่ยน)
- [ ] เมื่อมี use case แล้ว ทำตามตัวอย่างใน [books/13-temporal/02-retry-and-compensation.md](../../books/13-temporal/02-retry-and-compensation.md)

### 5. Scheduling ผ่าน Temporal

- [x] ตัดสินใจ Open Question ข้อ 2: **Temporal Schedules ครอบคลุม use case ปัจจุบันทั้งหมด** (มีแค่ `Interval` ธรรมดา ยังไม่มีกฎปฏิทินซับซ้อนจริงๆ ในระบบ) — **ลบ `internal/scheduler` ทิ้งทั้ง package แล้ว** (`scheduler.go`, `scheduler_test.go`) ตัว Schedule จริงสร้างผ่าน `client.ScheduleClient().Create` ใน `cmd/bop-worker/main.go` (`ensureExampleSchedule`, idempotent ผ่าน `temporal.ErrScheduleAlreadyRunning`)
- [ ] ถ้าในอนาคตต้องการกฎปฏิทินซับซ้อนกว่านี้ (ข้ามวันหยุด ฯลฯ) ที่ Temporal Schedules ไม่รองรับตรงๆ — ยังไม่เจอ use case จริง ปล่อยไว้ก่อนตาม YAGNI

### 6. Job History / Dashboard Integration

- [x] ตัดสินใจ Open Question ข้อ 3: **`job.Repository` ยังจำเป็นอยู่** แต่ตอนนี้เก็บไว้เฉพาะ job เดี่ยวๆ ที่สั่งผ่าน `POST /bots/{name}/jobs` เท่านั้น (ไม่ผ่าน Temporal เลย) — ส่วน workflow run เปลี่ยนไปอ่านจาก Temporal Visibility API ล้วนๆ, `internal/repository/memory/run_repository.go` (`workflow.RunRepository` impl เดิม) ถูกลบทิ้งแล้ว
- [x] `GET /workflow-runs` ใช้ `client.ListWorkflow` และ `GET /workflow-runs/{id}` ใช้ `DescribeWorkflowExecution` + `QueryWorkflow(..., "steps")` (query handler ที่ประกาศไว้ใน `workflow.Execute`) — ดู `internal/api/http/handler.go`

### 7. Live Log Streaming (เชื่อมกับ roadmap Phase 1.5)

- [x] `Activities.RunStep` เรียก `logger.Logger.Log` เหมือนเดิมทุกประการ ผูกกับ log ID รูปแบบ `<WorkflowID>-<ActivityID>` (ดู `internal/workflow/activity.go`) — **แต่มีข้อจำกัดใหม่ที่ต้องรู้**: logger instance ของ `cmd/bop-worker` เป็นคนละ process (คนละ in-memory map) กับที่ `cmd/bop` ใช้ตอบ `GET /jobs/{id}/logs` ดังนั้น log ของ step ใน workflow **บันทึกจริงแต่ดึงผ่าน HTTP ไม่ได้** จนกว่าจะสลับไปใช้ backend ที่แชร์ข้าม process ได้จริง (PostgreSQL/OpenSearch, Phase 2.2) — นี่คือ consequence ของการแยก process ตาม Open Question ข้อ 4 ที่ตอนถามคำถามให้เจ้าของ repo เลือกยังไม่ได้พูดถึงจุดนี้ชัดๆ บันทึกไว้ตรงนี้กันลืม
- [ ] `activity.RecordHeartbeat` — ยังไม่ implement เพราะ `http-status-checker` เป็น HTTP request เดียวจบเร็ว ไม่ใช่งานยาวที่ต้องการ heartbeat จริงๆ รอ bot ที่ใช้เวลานานค่อยเพิ่ม

### 8. Testing

- [x] `internal/workflow/temporal_test.go` ใช้ `testsuite.WorkflowTestSuite` (mock activity ผ่าน `OnActivity`) ทดสอบ orchestration logic (ทุก step สำเร็จ / หยุดที่ step แรกที่ล้มเหลว) และ `internal/workflow/activity_test.go` ใช้ `TestActivityEnvironment` ทดสอบ `Activities.RunStep` เองตรงๆ (รวม ConfigError → non-retryable) — แทนที่ `runner_test.go` เดิมที่ลบไปแล้วครบถ้วน
- [x] `internal/scheduler/scheduler_test.go` ถูกลบไปพร้อมกับทั้ง package ตามข้อ 5

### 9. Deployment

- [ ] เพิ่ม Temporal server เข้า K8s manifest — **ยังไม่ทำตามแผนเดิม** รอถึง [roadmap Phase 2.1](../../FULLSTACK-INFRA-ROADMAP.md#21-containerize--deploy-พื้นฐาน) จริงจัง (ยังไม่มี Dockerfile/K8s manifest ของ BOP เองเลยตอนนี้)

## Open Questions — ตัดสินใจแล้วทั้งหมด (เก็บไว้เป็น decision log)

1. **`internal/workflow` (Step/Workflow types)** — **เก็บไว้** เป็น input schema ให้ `workflow.Execute` อ่าน ไม่ได้ออกแบบใหม่ (ดูข้อ 3 ในตาราง checklist ด้านบน)
2. **`internal/scheduler`** — **แทนที่ด้วย Temporal Schedules ทั้งหมด** ไม่ทำ hybrid เพราะยังไม่มีกฎปฏิทินซับซ้อนจริงในระบบตอนนี้ (YAGNI) — ลบ package ทิ้งแล้ว
3. **`job.Repository`** — **แยกกันคนละ concept** ไม่รวมกับ Temporal Visibility API: `job.Repository` เก็บเฉพาะ job เดี่ยวๆ ที่สั่งผ่าน `POST /bots/{name}/jobs` (ไม่ผ่าน workflow) ส่วน workflow run ทั้งหมดอ่านจาก Temporal โดยตรง
4. **Temporal Worker process** — **แยกเป็น process ใหม่ `cmd/bop-worker`** เจ้าของ repo เลือกเอง หลังชั่งน้ำหนัก production realism (Temporal ใช้จริงแยก worker/API เกือบทุกที่ เพื่อ scale/deploy อิสระจากกัน) กับความซับซ้อนที่ต้องรัน 2 process ตอน local dev — **ผลข้างเคียงที่เพิ่งเจอตอน implement จริง** (ไม่ได้คุยกันตอนถามคำถามนี้): logger.Logger เป็น in-memory ล้วนๆ อยู่คนละ process ทำให้ log ของ step ใน workflow ดึงผ่าน `GET /jobs/{id}/logs` ไม่ได้จนกว่าจะมี logger backend ที่แชร์ข้าม process ได้จริง (ดูข้อ 7 ในตาราง checklist ด้านบน)

## Reference

- [books/13-temporal/](../../books/13-temporal/README.md) — ทฤษฎี Temporal เต็ม
- [books/11-microservices/04-saga-and-resilience.md](../../books/11-microservices/04-saga-and-resilience.md) — Saga pattern
- [FULLSTACK-INFRA-ROADMAP.md](../../FULLSTACK-INFRA-ROADMAP.md) — roadmap เต็มของ BOP
- Temporal Go SDK — เอกสารทางการที่ docs.temporal.io
- โค้ดจริงหลัง implement: `internal/workflow/temporal.go` (workflow function), `internal/workflow/activity.go` (Activity adapter), `internal/workflow/temporal_test.go` + `activity_test.go` (test), `cmd/bop-worker/main.go` (Worker process + Schedule), `internal/api/http/handler.go` (endpoint ที่ query Temporal โดยตรง), `docker-compose.yml` (Temporal server local dev)
