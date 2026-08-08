# BOP — สรุป Feature ปัจจุบัน + คำแนะนำ Feature ที่น่าสนใจต่อไป

> สรุป ณ วันที่เขียน — อ้างอิงจาก [README.md](README.md) และ decision doc ต่างๆ ในโฟลเดอร์นี้ ถ้าโค้ดเปลี่ยนไปหลังจากนี้ให้เชื่อ README.md/โค้ดจริงมากกว่าไฟล์นี้ (ไฟล์นี้เป็น snapshot ไม่ใช่ living doc ที่อัปเดตอัตโนมัติ)

## สถานะโดยรวม

`go build ./...`, `go vet ./...`, `go test -race ./...` ผ่านหมดทั้ง `bot-orchestration-platform` และ `remote-job-worker` (2 Go module แยกกัน) — **แต่ยังไม่เคยทดสอบ end-to-end จริงกับ Temporal server/K8s cluster เลย** (ติด Docker permission ตอน implement, ยังไม่มี K8s cluster ให้ทดสอบ) มีแค่ unit test level (`TestActivityEnvironment`/`TestWorkflowEnvironment`/fake K8s clientset) — นี่คือช่องว่างที่สำคัญที่สุดที่ควรรู้ก่อนพูดว่า "production-ready"

## Feature ที่มีอยู่แล้ว

### 1. Bot Execution Engine (แกนกลาง)

- `bot.Bot` interface (`Name`/`ConfigSchema`/`Run`) — pluggable, เพิ่ม bot ชนิดใหม่ไม่ต้องแก้ core เลย
- `worker.Pool` — semaphore-bounded concurrency, per-job timeout, cancellation ผ่าน context
- Bot ที่ implement แล้ว 2 ตัว: `http-status-checker` (ตัวอย่างง่ายที่สุด, zero dependency), `external-job` (shell out เรียก binary ภายนอกผ่าน `os/exec`)

### 2. สามทางเลือกในการรัน job ที่ไม่ใช่ Go/ไม่อยู่ใน process เดียวกับ BOP

| ทางเลือก | เอกสาร | สถานะ |
|---|---|---|
| Local exec (`internal/bots/externaljob`) | [EXTERNAL-JOB-BOT-TODO.md](EXTERNAL-JOB-BOT-TODO.md) | โค้ด+test เสร็จ, binary path ยัง placeholder |
| Remote Temporal task queue (`projects/remote-job-worker/`) | [REMOTE-JOB-WORKER-TODO.md](REMOTE-JOB-WORKER-TODO.md) | โค้ด+reference worker+test เสร็จ, ยังไม่ทดสอบกับ Temporal server จริง |
| K8s Job hybrid (`internal/workflow/k8sjob.go`) | [K8S-JOB-HYBRID-EXECUTION-TODO.md](K8S-JOB-HYBRID-EXECUTION-TODO.md) | โค้ด+test (fake clientset) เสร็จ, รอ K8s cluster จริง + RBAC manifest |

ทั้งสามทางเลือก **cancellation ทำงานถูกต้องจริง** (kill process group / delete K8s Job / ไม่ทิ้ง orphan) — พิสูจน์ด้วย test ทุกตัว

### 3. Workflow Orchestration (Temporal)

- `internal/workflow.Execute` — Temporal workflow function, รัน step ตามลำดับ, durable execution (worker ตายกลางทาง resume ต่อได้เอง)
- Retry policy แยก non-retryable (`ConfigError`, `UnknownBot`) ออกจาก retryable โดยอัตโนมัติ
- **Per-step timeout** (`Step.TimeoutSeconds`) และ **per-step task queue routing** (`Step.TaskQueue`) — ไม่ใช้ timeout/queue เดียวตายตัวทั้งระบบอีกต่อไป
- Query handler `"steps"` — ดูผลลัพธ์ทีละ step ได้ทั้งตอน workflow กำลังรันอยู่และจบไปแล้ว

### 4. Schedule Management (เทียบเท่า Control-M job management)

CRUD เต็มชุดผ่าน `client.ScheduleClient` ของ Temporal: create/list/get/update/delete/pause/unpause/**trigger** (Order Now)/**backfill** (Rerun) — รองรับทั้ง interval และ cron expression (รวม "เฉพาะวันทำการ" ผ่าน cron syntax), overlap policy 6 แบบ

### 5. Ad-hoc Workflow Run

`POST /workflow-runs` — รัน step ทันทีโดยไม่ต้องสร้าง schedule ก่อน ใช้ได้ทั้ง step ที่รันผ่าน bot local, remote task queue, หรือ K8s Job (routing ตัดสินใจต่อ step)

### 6. Job เดี่ยว + Logging

- `POST /bots/{name}/jobs` — สร้าง+รัน job เดี่ยวทันที (ไม่ผ่าน Temporal, ผ่าน `worker.Pool` ตรงๆ) พร้อม per-job timeout (`?timeout_seconds=`)
- Cancel job ที่กำลังรันอยู่ (`DELETE /jobs/{id}`)
- Log แบบ real-time ผ่าน SSE (`GET /jobs/{id}/logs/stream`) + snapshot (`GET /jobs/{id}/logs`) — **เก็บใน memory เท่านั้น หายเมื่อ restart**

### 7. Auth

JWT (HS256, เขียนเอง) + httpOnly cookie — **แต่เป็น single hardcoded user** (`admin`/`changeme`) ยังไม่มี user table/RBAC จริง (ตั้งใจ ตาม "BOP เป็น single-operator tool" ไม่ใช่ multi-tenant SaaS ตอนนี้)

### 8. HTTP API (สรุปทุก endpoint)

`GET/POST /bots`, `/jobs`, `/jobs/{id}`, `/jobs/{id}/logs[/stream]`, `DELETE /jobs/{id}`, `GET/POST /workflow-runs`, `GET /workflow-runs/{id}`, `POST /login|logout`, `GET /me`, CRUD เต็มของ `/schedules` (+ pause/unpause/trigger/backfill)

### 9. Dashboard (Next.js, แยก repo)

[projects/bop-dashboard/](../bop-dashboard/README.md) — bot list, dynamic config form, job history, live job monitor (SSE), auth — ทำครบ Phase 1 ตาม roadmap แต่**ยังไม่มี UI สำหรับเลือก dispatch mode ใหม่** (remote task queue / K8s Job) ที่เพิ่งเพิ่มเข้า backend — ใช้ได้แค่ local bot ผ่านฟอร์มเดิม

## ข้อจำกัดที่สำคัญที่สุด (พูดตรงๆ)

1. **Job/Schedule definition ยังเป็น in-memory ทั้งหมด** — restart แล้วประวัติ job/schedule metadata หายหมด (Temporal เองยังรัน schedule ต่อได้ปกติ แต่ BOP "ลืม" ว่า schedule นั้นทำอะไร)
2. **ไม่มี PostgreSQL/OpenSearch จริงเลยสักตัว** — ตามแผน Phase 0.4/2.2 แต่ยังไม่ได้ทำ
3. **ยังไม่เคย containerize/deploy K8s จริง** — ไม่มี Dockerfile ด้วยซ้ำตอนนี้
4. **ไม่มี metric/tracing/alert เลย** — ไม่รู้ว่า bot ไหนพังเพราะอะไรจากภายนอก ต้องอ่าน log เอง
5. **ไม่มี job dependency (DAG)** — ทุก step ต้องอยู่ใน workflow เดียวกันเรียงลำดับตายตัว ยังทำ "job B รอ job A จาก schedule คนละตัว" ไม่ได้
6. **RBAC ยังไม่มี** — auth มีแค่ login/logout, ทุกคนที่ login ได้สิทธิ์เท่ากันหมด

## คำแนะนำ Feature ที่น่าสนใจ (เรียงตาม impact/effort)

1. **PostgreSQL persistence ให้ `job.Repository`/`schedule.Repository`** — ทำก่อนสุด เพราะเป็นรากของปัญหาข้อ 1-2 ด้านบน และเป็น prerequisite ที่ทุกอย่างอื่นในตารางนี้ต้องพึ่งไม่ทางใดก็ทางหนึ่ง (metric/audit trail ก็ต้องมีที่เก็บถาวรเหมือนกัน) — interface `job.Repository`/`schedule.Repository` ออกแบบไว้พร้อมสลับอยู่แล้ว ใช้เวลาไม่มากเทียบกับ value ที่ได้

2. **Job Dependency (DAG) ข้าม schedule** — เป็นฟีเจอร์ที่ทำให้ BOP ต่างจาก "แค่ cron ที่ retry ได้" อย่างชัดเจนที่สุด และเป็น talking point ที่ดีมากตอนสัมภาษณ์ (child workflow / `workflow.Await` / signal ระหว่าง workflow) — ตรงกับ positioning "Control-M ตัวเล็ก" ที่สุดในบรรดา feature ที่ยังไม่ทำ

3. **Observability ขั้นต่ำ (Prometheus metric อย่างเดียวก่อนพอ ไม่ต้องรอครบ Fluent Bit/Jaeger)** — แค่ job success/fail count + duration histogram ต่อ bot ก็ตอบคำถามสัมภาษณ์ "รู้ได้ยังไงว่า prod พัง" ได้แล้ว ทำแยกจาก log/tracing ได้ ไม่ต้องรอ Phase 2.2 ทั้งก้อน

4. **SLA monitoring + alert แบบง่าย** (job ใช้เวลาเกิน N เท่าของค่าเฉลี่ยเดิม → แจ้งเตือน) — ต่อยอดจาก metric ข้อ 3 ได้ทันที ไม่ต้องมี infra ใหม่ และเป็นฟีเจอร์ระดับ Control-M จริงๆ ที่ demo ได้ชัดเจน

5. **เติม UI ของ dashboard ให้เลือก dispatch mode ได้** (local / remote task queue / K8s Job) — backend รองรับครบแล้วแต่ dashboard ยังตามไม่ทัน เป็นช่องว่างที่เห็นชัดถ้ามีคนมาทดลองใช้งานจริงผ่านหน้าเว็บ ใช้เวลาไม่นานเพราะ schema (`ConfigField`) ออกแบบไว้ให้ dynamic form อยู่แล้ว

ฟีเจอร์ที่ตั้งใจ**ไม่แนะนำให้รีบทำตอนนี้** แม้จะอยู่ใน roadmap: Approval Gate/Resource Pool/Flow Control (Phase 3) — สนุกและ "ดูล้ำ" แต่ยังไม่มีประโยชน์จริงถ้า persistence/observability พื้นฐาน (ข้อ 1-3) ยังไม่มี จะกลายเป็น "ฟีเจอร์ enterprise บนฐานที่ไม่มั่นคง" ซึ่งเป็นจุดอ่อนที่คนสัมภาษณ์สาย senior จะจับได้ทันที — design doc เต็มของ Phase 3 ทั้ง 7 ฟีเจอร์ (data model, Temporal pattern, Open Questions, ลำดับที่แนะนำ) เขียนไว้ล่วงหน้าแล้วที่ [PHASE3-ENTERPRISE-FEATURES-TODO.md](PHASE3-ENTERPRISE-FEATURES-TODO.md) พร้อมหยิบไป implement ทีหลังตอน persistence เสร็จ
