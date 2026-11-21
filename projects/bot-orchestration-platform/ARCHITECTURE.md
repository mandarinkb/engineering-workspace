# BOP Architecture

เอกสารนี้อธิบายโครงสร้างโค้ด Go ของ BOP หลัง refactor เป็น clean architecture ว่าแต่ละโฟลเดอร์มีหน้าที่อะไร import อะไรได้บ้าง และโค้ดใหม่ควรไปอยู่ตรงไหน

## ภาพรวม

```
                 ┌────────────────────────────────────────────┐
  ขาเข้า          │  cmd/bop, cmd/bop-worker  (composition root) │
                 └───────┬───────────────────────────┬────────┘
                         │ ประกอบ                     │ ประกอบ
                         ▼                           ▼
  internal/adapter/http ──► internal/usecase ──► internal/domain ◄── internal/external
                                                     ▲              internal/repository
                                                     │              internal/pkg/logger
                                                     │              internal/adapter/worker
                                     (implement port ที่ประกาศไว้ใน domain)
```

กฎข้อเดียวที่ต้องจำ: **dependency ชี้เข้าหา `domain` เสมอ** ชั้นในห้าม import ชั้นนอก

## โครงสร้างโฟลเดอร์

```
cmd/
  bop/main.go              composition root ของ HTTP API server
  bop-worker/main.go       composition root ของ Temporal Worker process

internal/
  domain/                  entity + port (interface) — import ได้แค่ stdlib
    apperr/                ประเภท error กลาง: ErrInvalid, ErrNotFound, ErrConflict
    bot/                   Bot interface, Config, Result, ConfigError
    job/                   Job entity, Repository port, Executor port
    logger/                Logger port, Pruner port, Line, Level
    workflow/              Workflow, Step, StepResult, RunSummary/RunDetail, Engine port
    schedule/              Definition, Repository port, Spec, Overlap, Scheduler port

  usecase/                 application logic — import ได้แค่ domain และ pkg (jwt, idgen)
    job/                   สร้าง/ดู/ยกเลิก job เดี่ยว + log
    workflowrun/           สั่งรัน workflow แบบ ad-hoc + ดูสถานะ run + readiness
    schedule/              validate + ลำดับการเรียก scheduler/repository + compensation
    auth/                  ตรวจ credential + ออก/ตรวจ token
    logretention/          งานลบ log ที่เก่ากว่า 30 วัน วันละครั้ง (รันใน bop-api)

  adapter/
    http/                  ขาเข้า: HTTP handler แปลง JSON <-> usecase เท่านั้น (1 ไฟล์ต่อ 1 endpoint)
    worker/                ขาออก: Pool รัน bot ใน process เดียวกัน (implement domain/job.Executor)

  external/                ขาออก: implement port ด้วยระบบภายนอก
    temporal/              Engine, Scheduler, Execute (workflow function), Activities.RunStep
    k8s/                   JobActivities.RunKubernetesJob
    bots/                  httpstatus, externaljob (implement bot.Bot)

  repository/              ขาออก: implement Repository port (storage)
    memory/                job + schedule repository แบบ in-memory
    postgres/              job repository + logger บน PostgreSQL + Connect/Migrate (migration ฝังใน binary)

  pkg/                     library ที่ใช้ร่วมกัน
    jwt/                   JWT HS256 เขียนเอง (ไม่รู้จัก domain)
    idgen/                 สุ่ม ID ด้วย crypto/rand (ไม่รู้จัก domain)
    logger/                Logger แบบ in-memory (implement domain/logger.Logger)
```

## หน้าที่ของแต่ละชั้น

### `domain`

- เก็บ entity, value object, กฎที่ไม่ขึ้นกับเทคโนโลยี (เช่น `schedule.Spec.Validate`, `schedule.ParseOverlap`, การเปลี่ยนสถานะ `job.Job.Start/Succeed/Fail/Cancel`) และ **port** คือ interface ที่ชั้นนอกต้อง implement
- import ได้แค่ stdlib และ `domain` ด้วยกันเอง
- json tag บน `job.Job`, `logger.Line`, `workflow.Step`, `workflow.StepResult` และ `bot.ConfigField` ตั้งใจเก็บไว้ที่ domain เพราะเป็นทั้ง wire format ของ API และ payload ที่ Temporal serialize ไว้ (ดูหัวข้อ "ข้อห้ามที่ทำให้ระบบเดิมพัง")

| Port | ประกาศที่ | Implementation ปัจจุบัน |
|---|---|---|
| `job.Repository` | `domain/job` | `repository/memory.JobRepository`, `repository/postgres.JobRepository` (เลือกตาม `BOP_DATABASE_URL` ใน `cmd/bop`) |
| `job.Executor` | `domain/job` | `adapter/worker.Pool` |
| `logger.Logger` | `domain/logger` | `pkg/logger.Logger`, `repository/postgres.Logger` (เลือกตาม `BOP_DATABASE_URL` ใน `cmd/bop`) |
| `logger.Pruner` | `domain/logger` | `repository/postgres.Logger` (ใช้โดย `usecase/logretention`) |
| `schedule.Repository` | `domain/schedule` | `repository/memory.ScheduleRepository` |
| `schedule.Scheduler` | `domain/schedule` | `external/temporal.Scheduler` |
| `workflow.Engine` | `domain/workflow` | `external/temporal.Engine` |
| `bot.Bot` | `domain/bot` | `external/bots/httpstatus`, `external/bots/externaljob` |

### `usecase`

- ทำงานผ่าน port เท่านั้น ห้าม import Temporal SDK, `net/http`, K8s client หรือ implementation ตัวจริง
- รับผิดชอบการ validate input, ลำดับขั้นตอน และ compensation เช่น `schedule.Service.Create` สร้างที่ scheduler ก่อนแล้วค่อยบันทึก definition ถ้าบันทึกพลาดให้ลบ schedule ที่เพิ่งสร้างทิ้ง
- คืน error เป็นประเภทใน `domain/apperr` เพื่อให้ชั้น HTTP เลือก status code ได้โดยไม่ต้องรู้ว่าข้างหลังเป็นอะไร

### `adapter/http`

- แปลง request เป็นการเรียก usecase แล้วแปลงผลลัพธ์เป็น JSON
- 1 ไฟล์ต่อ 1 endpoint ตั้งชื่อตาม action + resource (`create_job.go`, `pause_schedule.go`) request/response DTO ที่ใช้ endpoint เดียวอยู่ในไฟล์ของ endpoint นั้น ส่วนที่ใช้ร่วมกันอยู่ใน `*_common.go` (`scheduleSummary`, `workflowRunSummary`, `writeScheduleError`) route ทั้งหมดลงทะเบียนที่ `server.go` และ `writeJSON` อยู่ที่ `response.go`
- เก็บเรื่องที่เป็นของ HTTP ล้วน ๆ: CORS, cookie, SSE, การ parse query parameter
- ไม่มี business logic และไม่ import Temporal SDK

### `adapter/worker`

- `Pool` implement `job.Executor` ด้วย goroutine pool ใน process เดียวกับ HTTP server (จำกัด concurrency, timeout ต่อ job, ยกเลิกกลางทาง)
- เปลี่ยนสถานะ job ผ่าน method ของ entity (`Start`, `Succeed`, `Fail`, `Cancel`) เท่านั้น ห้ามตั้ง `Status` เอง เพราะกฎการเปลี่ยนสถานะอยู่ใน `domain/job`
- `Pool.Shutdown(ctx)` หยุดรับ job ใหม่แล้วรอ job ที่ค้างอยู่ให้จบ ถ้า ctx หมดก่อนจะยกเลิก job ที่เหลือ (สถานะ cancelled) `cmd/bop` เรียกหลังปิด HTTP server ตอนได้รับ SIGINT/SIGTERM
- copy job ด้วย `job.Job.Clone()` ทุกครั้งที่ส่งข้าม goroutine เพราะ `*j` ไม่ได้ copy map `Config`

### `external`

- implement port ด้วยระบบภายนอก และแปลง error ของระบบนั้นเป็น `apperr` เช่น `serviceerror.NotFound` ของ Temporal เป็น `apperr.ErrNotFound` และ `ErrScheduleAlreadyRunning` เป็น `apperr.ErrConflict`
- `external/temporal` มีทั้งฝั่งเรียกออก (Engine, Scheduler) และฝั่งที่ Temporal เรียกเข้ามา (Execute, Activities) เพราะทั้งหมดผูกกับ Temporal SDK ตัวเดียวกัน

### `repository`

- implement `Repository` port ของ domain ทุกตัวที่เกี่ยวกับการเก็บข้อมูล
- implementation ทุกตัวต้อง copy ค่าเข้า-ออกเสมอ ห้ามคืน pointer ที่ map ภายในถืออยู่ (ดูเหตุผลใน `repository/memory/job_repository.go`)
- PostgreSQL implementation อยู่ที่ `repository/postgres/` (`Connect` เปิด pool ที่ทุก repository/logger ใช้ร่วมกัน, `Migrate` รันเฉพาะใน bop-api) แล้วสลับในขั้นประกอบ dependency ของ `cmd/bop/main.go`
- contract test ของ `job.Repository` อยู่ที่ `repository/job_repository_test.go` และของ `logger.Logger` อยู่ที่ `repository/logger_test.go` รันชุดเดียวกันกับทุก implementation (ส่วน PostgreSQL รันเมื่อตั้ง `BOP_TEST_DATABASE_URL`)
- ตาราง `log_lines` มีเจ้าของได้ 2 แบบตาม spec: Job (`job_id`) หรือ Step (`workflow_id` + `step_index` + `attempt`) บังคับด้วย CHECK ให้มีแบบเดียวพอดี

### `pkg`

- `jwt` และ `idgen` เป็น library ทั่วไปที่ไม่รู้จัก domain
- `logger` เป็นข้อยกเว้นที่ import `domain` อยู่ เพราะ implement port ของ domain — implementation บน PostgreSQL อยู่ที่ `repository/postgres` (ถ้าวันหนึ่งย้ายไป OpenSearch ก็สร้างที่ `repository/` เช่นกัน)

### `cmd` (composition root)

- เป็นที่เดียวที่เลือก implementation จริงแล้วประกอบเข้ากับ usecase
- อ่าน env var ทั้งหมดที่นี่ ยกเว้น `BOP_DASHBOARD_ORIGIN` ซึ่งเป็น config ของชั้น HTTP เอง

## ตารางการ import

ตารางนี้นับเฉพาะโค้ดหลัก ไฟล์ `_test.go` import ชั้นอื่นเพื่อประกอบ test ได้ (เช่น `adapter/http/main_test.go` ใช้ `external/temporal` และ `repository/memory`)

| จาก \ ไป | domain | usecase | adapter | external | repository | pkg |
|---|:-:|:-:|:-:|:-:|:-:|:-:|
| domain | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| usecase | ✅ | ❌ | ❌ | ❌ | ❌ | ✅ jwt, idgen |
| adapter/http | ✅ | ✅ | — | ❌ | ❌ | ❌ |
| adapter/worker | ✅ | ❌ | — | ❌ | ❌ | ❌ |
| external | ✅ | ❌ | ❌ | — | ❌ | ❌ |
| repository | ✅ | ❌ | ❌ | ❌ | — | ❌ |
| pkg | ✅ logger | ❌ | ❌ | ❌ | ❌ | ✅ |
| cmd | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

`golangci-lint run ./...` บังคับตารางนี้อัตโนมัติผ่าน `depguard` (กฎอยู่ใน `.golangci.yml` และรันใน CI) ถ้าแก้ตารางนี้ต้องแก้ `.golangci.yml` ให้ตรงกันด้วย หรือตรวจด้วยมือได้ด้วย:

```bash
go list -f '{{.ImportPath}}: {{join .Imports " "}}' ./internal/domain/... ./internal/usecase/... \
  | grep -E "adapter|external|repository|temporal|k8s" || echo "layers clean"
```

## ตัวอย่าง flow: `POST /schedules`

1. `adapter/http.handleCreateSchedule` decode JSON เป็น `createScheduleRequest` แล้วเรียก `usecase/schedule.Service.Create`
2. usecase ตรวจ `id` → `steps` → `Spec.Validate()` → `ParseOverlap()` ถ้าผิดคืน `apperr.ErrInvalid`
3. usecase เรียก `schedule.Scheduler.Create` ซึ่งตัวจริงคือ `external/temporal.Scheduler` แปลงเป็น `client.ScheduleOptions` แล้วเรียก Temporal
4. usecase บันทึก `schedule.Definition` ผ่าน `schedule.Repository` ถ้าพลาดให้เรียก `Scheduler.Delete` เพื่อย้อนกลับ
5. usecase เรียก `Get` เพื่อรวมสถานะจาก Temporal กับ steps จาก repository
6. handler แปลง `schedule.Detail` เป็น JSON และแปลง error เป็น status code ผ่าน `writeScheduleError`

## การแปลง error เป็น HTTP status

| error จาก usecase | Status |
|---|---|
| `apperr.ErrInvalid` | 400 |
| `apperr.ErrNotFound` | 404 |
| `apperr.ErrConflict` | 409 |
| อื่น ๆ | 500 |

wrapper ทุกตัวใน `apperr` คืนข้อความเดิมของ error ข้างใน เพราะ response body ส่ง `err.Error()` ให้ client ตรง ๆ ส่วน `apperr.Plain` ใช้ตัดประเภทออกในขั้นที่ API เดิมตอบ 500 เสมอ เช่น describe หลังสร้าง schedule สำเร็จแล้ว

## ข้อห้ามที่ทำให้ระบบเดิมพัง

- **ห้ามเปลี่ยนชื่อฟังก์ชัน `Execute`** ใน `external/temporal/workflow.go` เพราะ Temporal ใช้ชื่อฟังก์ชันเป็น workflow type ของ schedule ที่มีอยู่แล้ว
- **ห้ามเปลี่ยนค่า `ActivityName` ("RunStep") และ `K8sJobActivityName` ("RunKubernetesJob")** เพราะ remote worker ใน `projects/remote-job-worker` ลงทะเบียนด้วยชื่อเดียวกัน
- **ห้ามเปลี่ยน json tag ของ `workflow.Step` และ `workflow.K8sJobSpec`** เพราะ schedule ที่มีอยู่ใน Temporal เก็บ payload ตาม tag เดิม ถ้าเปลี่ยนจะ decode ไม่ได้
- **ห้ามเปลี่ยน json tag ของ `job.Job`, `logger.Line` และ DTO ใน `adapter/http`** เพราะ dashboard (`projects/bop-dashboard`) อ่านอยู่

## เพิ่มของใหม่ต้องไปไว้ที่ไหน

| อยากเพิ่ม | ที่อยู่ | ต้องแก้ที่ไหนอีก |
|---|---|---|
| bot ชนิดใหม่ | `internal/external/bots/<name>/` implement `bot.Bot` | register ทั้ง `cmd/bop` (`pool.Register`) และ `cmd/bop-worker` (`activities.Register`) |
| endpoint ใหม่ | ไฟล์ใหม่ใน `adapter/http` (เช่น `get_bot.go`) + route ใน `server.go` + method ใน usecase ที่เกี่ยวข้อง | ถ้าต้องใช้ระบบภายนอกใหม่ ให้ประกาศ port ใน `domain` ก่อน |
| storage ใหม่ (PostgreSQL) | `internal/repository/postgres/` implement `job.Repository` / `schedule.Repository` | สลับใน `cmd/bop/main.go` แล้วรัน test ชุดเดียวกันกับ in-memory |
| ระบบภายนอกใหม่ (เช่น ส่งแจ้งเตือน) | port ใน `domain/<concept>` + implementation ใน `internal/external/<system>/` | usecase เรียกผ่าน port แล้วประกอบใน `cmd` |
| กฎทางธุรกิจที่ไม่ขึ้นกับเทคโนโลยี | method บน type ใน `domain` (แบบ `Spec.Validate`) | usecase เป็นคนเรียก |
| ลำดับขั้นตอนหลาย port | `usecase` | — |

## ชื่อ import alias ที่ใช้ร่วมกัน

| Alias | Package | เหตุผล |
|---|---|---|
| `apihttp` | `internal/adapter/http` | ชนกับ `net/http` |
| `memlogger` | `internal/pkg/logger` | ชนกับ `internal/domain/logger` |
| `jobuc`, `scheduleuc`, `workflowrunuc`, `authuc` | `internal/usecase/*` | ชนกับ package ใน domain ที่ชื่อเดียวกัน |
| `wfdomain` | `internal/domain/workflow` (ใน `external/temporal`) | ชนกับ `go.temporal.io/sdk/workflow` |
| `sdktemporal` | `go.temporal.io/sdk/temporal` | ชนกับ `internal/external/temporal` |

## ประวัติ

- เดิมโค้ดอยู่ที่ `internal/{bot,job,logger,workflow,schedule,worker,bots,auth,api/http,repository/memory,logger/memory}` โดย HTTP handler เรียก Temporal SDK และ repository ตรง ๆ
- refactor เป็นโครงสร้างนี้บน branch `refactor/clean-architecture` (2026-10-10) ย้ายไฟล์ด้วย `git mv` จึงตามประวัติเดิมได้ด้วย `git log --follow`
- path ในเอกสาร `*-TODO.md` เป็นโครงสร้างก่อน refactor เพราะเป็นบันทึกการตัดสินใจตามเวลาที่เขียน
