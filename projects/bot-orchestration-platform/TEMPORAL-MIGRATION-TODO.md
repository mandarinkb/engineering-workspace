# TODO: ย้าย Workflow/Scheduler ไปใช้ Temporal (Roadmap Phase 2.5)

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

- [ ] เพิ่ม Temporal server + Temporal UI + PostgreSQL (persistence backend ของ Temporal เอง แยกจาก PostgreSQL ที่ BOP จะใช้เก็บ job history ใน Phase 0.4 — จะแชร์ instance เดียวกันคนละ database หรือแยก instance เลยก็ได้ ตัดสินใจตอน implement) เข้า `docker-compose.yml` สำหรับ local dev (เชื่อมกับ [roadmap Phase 2.1](../../FULLSTACK-INFRA-ROADMAP.md#21-containerize--deploy-พื้นฐาน))
- [ ] ยืนยันว่า Temporal server รันได้ เปิด Temporal UI (default port 8233) เห็น namespace `default`

### 2. เพิ่ม Temporal Go SDK

- [ ] `go get go.temporal.io/sdk` — **หมายเหตุ**: BOP ตั้งใจ zero external dependency มาตลอด (ดู `go.mod`) จุดนี้คือจุดแรกที่ควรมี dependency ภายนอก เพราะ workflow orchestration ระดับ durable execution คือสิ่งที่ reinvent เองไม่คุ้มอีกต่อไป — ไม่ใช่ความผิดพลาดของ design เดิม
- [ ] ตัดสินใจ: แยก Temporal Worker เป็น process ใหม่ (เช่น `cmd/bop-worker/`) หรือรวมไว้ใน `cmd/bop` เดิม (ดู Open Question ข้อ 4 ด้านล่าง)

### 3. ออกแบบ Workflow ด้วย Temporal

- [ ] ตัดสินใจว่า `workflow.Step`/`workflow.Workflow` (type เดิมใน `internal/workflow/workflow.go`) ยังใช้เป็น "input data" ให้ Temporal workflow function อ่านได้ไหม (แนะนำ: ใช้ต่อได้ แค่เปลี่ยนตัวที่ execute จาก `Runner.Trigger` เป็น Temporal workflow function ที่ loop ผ่าน `Steps` แล้วเรียก `workflow.ExecuteActivity` ทีละตัวแทน)
- [ ] แปลง `bot.Bot.Run` ให้ถูกเรียกจาก Temporal Activity function — เก็บ `Bot` interface เดิมไว้ได้ Activity เป็นแค่ adapter บางๆ ที่เรียก `bot.Run` อีกที ไม่ต้องเขียน bot logic ใหม่
- [ ] ใส่ Retry Policy ต่อ Activity ตามที่อธิบายไว้ใน [books/13-temporal/02-retry-and-compensation.md](../../books/13-temporal/02-retry-and-compensation.md) — กำหนด `NonRetryableErrorTypes` ให้ error ประเภท "missing required config key" (เช่นจาก `internal/bots/httpstatus`) ไม่ต้อง retry เลย เพราะ retry กี่ครั้งก็ได้ผลลัพธ์เดิม

### 4. Compensation Logic (Saga)

- [ ] **อย่าออกแบบ compensation ล่วงหน้าแบบไม่มี use case จริง** — bot ตัวอย่างตอนนี้ (`http-status-checker`) ไม่มี side effect ที่ต้อง rollback เลย รอจนมี bot ที่มี side effect จริงจัง (เขียนลง database, เรียก external API ที่ mutate state) ค่อยออกแบบ compensating action ให้ตรงกับ use case นั้น
- [ ] เมื่อมี use case แล้ว ทำตามตัวอย่างใน [books/13-temporal/02-retry-and-compensation.md](../../books/13-temporal/02-retry-and-compensation.md) (order+payment+inventory) เขียน compensation ที่ rollback ย้อนลำดับกับ step ที่สำเร็จไปแล้ว

### 5. Scheduling ผ่าน Temporal

- [ ] ประเมิน **Temporal Schedules** (ฟีเจอร์ built-in สำหรับรัน workflow ตามเวลา คล้าย cron) ว่าครอบคลุม use case ปัจจุบัน (`scheduler.Interval`) ไหม — ถ้าครอบคลุม ลบ `internal/scheduler` ทิ้งได้เลย ใช้ Temporal Schedules แทนทั้งหมด
- [ ] ถ้า Temporal Schedules ไม่รองรับกฎปฏิทินซับซ้อนที่ตั้งใจไว้ใน roadmap เดิม (เช่น "ข้ามวันหยุด", "เฉพาะวันทำการสุดท้ายของเดือน") ให้ดู Open Question ข้อ 2 ด้านล่าง — ไม่ต้องตัดสินใจตอนนี้

### 6. Job History / Dashboard Integration

- [ ] ดู Open Question ข้อ 3 — ตัดสินใจว่า `job.Repository` ยังจำเป็นคู่กับ Temporal Visibility API (`ListWorkflowExecutions`, `DescribeWorkflowExecution`) ไหม
- [ ] อัปเดต `GET /workflow-runs` และ `GET /workflow-runs/{id}` (ปัจจุบัน query จาก in-memory `workflow.RunRepository`) ให้ query จาก Temporal client แทน หรือลบทิ้งถ้าตัดสินใจให้ dashboard คุยกับ Temporal API ตรงๆ

### 7. Live Log Streaming (เชื่อมกับ roadmap Phase 1.5)

- [ ] `logger.Logger`/`bot.LogFunc` ที่มีอยู่แล้ว (ดู [internal/logger/logger.go](internal/logger/logger.go)) ควรใช้ต่อได้เหมือนเดิมจากภายใน Activity — แค่ต้องตรวจสอบว่า Activity context ส่ง job/run ID ที่ถูกต้องเข้าไปด้วย
- [ ] พิจารณาใช้ `activity.RecordHeartbeat` ควบคู่กับ Logger เดิมสำหรับ Activity ที่ใช้เวลานาน (ป้องกัน worker ตายเงียบๆ ตามที่อธิบายไว้ใน [books/13-temporal/01-workflow-and-activity.md](../../books/13-temporal/01-workflow-and-activity.md))

### 8. Testing

- [ ] ใช้ `go.temporal.io/sdk/testsuite` (`testsuite.WorkflowTestSuite`) เขียน test ของ workflow function โดยไม่ต้องมี Temporal server จริง — แทนที่ `internal/workflow/runner_test.go` เดิม
- [ ] ตัดสินใจว่า `internal/scheduler/scheduler_test.go` เดิมยังมีประโยชน์ไหม (ถ้า `internal/scheduler` ถูกลบทั้งหมดตามข้อ 5 ก็ลบ test ไปด้วย)

### 9. Deployment

- [ ] เพิ่ม Temporal server เข้า K8s manifest ตอนถึง [roadmap Phase 2.1](../../FULLSTACK-INFRA-ROADMAP.md#21-containerize--deploy-พื้นฐาน) จริงจัง

## Open Questions (ต้องตัดสินใจก่อนเริ่มโค้ดจริง)

รวบรวมจาก checklist ข้างบนมาไว้ที่เดียว:

1. **`internal/workflow` (Step/Workflow types)** — เก็บไว้เป็น input schema ให้ Temporal workflow function อ่าน หรือออกแบบใหม่ทั้งหมด?
2. **`internal/scheduler`** — แทนที่ด้วย Temporal Schedules ทั้งหมด หรือแบบ hybrid (Temporal Schedules สำหรับ simple interval + custom logic คำนวณวันที่เองสำหรับกฎปฏิทินซับซ้อน แล้วค่อยยิง `StartWorkflow` ผ่าน Temporal client ตอนถึงเวลาที่คำนวณได้)?
3. **`job.Repository`** — ยังจำเป็นคู่กับ Temporal Visibility API ไหม หรือรวมเป็น concept เดียวกัน? (ต้องคิดถึง bot เดี่ยวๆ ที่สั่งผ่าน `POST /bots/{name}/jobs` โดยไม่ผ่าน workflow ด้วย — bot พวกนี้ไม่มี Temporal workflow ครอบ จึงยังต้องมี job history ของตัวเองอยู่ดีหรือเปล่า)
4. **Temporal Worker process** — แยกจาก HTTP API server (`cmd/bop`) เป็น process ใหม่ (`cmd/bop-worker`) หรือรวมกันในตัวเดียว?

## Reference

- [books/13-temporal/](../../books/13-temporal/README.md) — ทฤษฎี Temporal เต็ม
- [books/11-microservices/04-saga-and-resilience.md](../../books/11-microservices/04-saga-and-resilience.md) — Saga pattern
- [FULLSTACK-INFRA-ROADMAP.md](../../FULLSTACK-INFRA-ROADMAP.md) — roadmap เต็มของ BOP
- Temporal Go SDK — เอกสารทางการที่ docs.temporal.io
