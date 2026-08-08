# TODO: Remote Job Worker (รัน jobcode จาก repo/binary อื่นผ่าน Temporal task queue แยกกัน)

> **สถานะ**: ฝั่ง BOP (repo นี้) ทำเสร็จแล้ว — `workflow.Step.TaskQueue`, endpoint `POST /workflow-runs` พร้อมใช้งาน, `go build`/`go vet`/`go test -race ./...` ผ่านหมด (ดู [README.md](README.md) หัวข้อ "รัน step ผ่าน remote Temporal worker") **ฝั่ง remote worker ก็ทำ reference implementation เสร็จแล้วเช่นกัน** ที่ [projects/remote-job-worker/](../remote-job-worker/README.md) (Go module แยกต่างหากจาก BOP โดยสมบูรณ์ ไม่ import โค้ดของ BOP เลย) — เป็น**ตัวอย่าง**ที่ยัง dispatch แค่ placeholder jobcode (`PING`/`LONG_TASK`/`FAIL_TRANSIENT`) เพราะยังไม่รู้ jobcode จริงของเจ้าของ repo (เหมือน path ของ binary ใน `internal/bots/externaljob` ที่ยังเป็น placeholder เหตุผลเดียวกัน) — สลับ logic จริงใน `internal/jobs/jobs.go` ของ module นั้นได้เมื่อรู้ jobcode จริงแล้ว โครง `cmd/worker/main.go` ไม่ต้องแก้เลย

## บริบท

โปรเจกต์หลักคือ **Bot Orchestration Platform (BOP)** — ดูภาพรวมทั้งหมดที่ [FULLSTACK-INFRA-ROADMAP.md](../../FULLSTACK-INFRA-ROADMAP.md) และ [README.md](README.md)

เอกสารนี้เป็น**ทางเลือกที่สอง**คู่กับ [EXTERNAL-JOB-BOT-TODO.md](EXTERNAL-JOB-BOT-TODO.md) (`internal/bots/externaljob` — shell out ไปเรียก binary ในเครื่อง/container เดียวกัน) สำหรับกรณีที่ job/binary อยู่**คนละ repository และอยากแยก deploy คนละจังหวะกับ BOP จริงๆ** (ไม่ใช่แค่คนละไฟล์ในเครื่องเดียวกัน) — แลกกับที่ repo นั้นต้องแก้โค้ดเพื่อ embed Temporal Go SDK เอง (ต่างจาก `externaljob` ที่ตั้งใจไม่แตะ binary เดิมเลย)

มี**ทางเลือกที่สาม**ด้วย (ยังเป็นแค่ design doc ยังไม่ implement) ที่ [K8S-JOB-HYBRID-EXECUTION-TODO.md](K8S-JOB-HYBRID-EXECUTION-TODO.md) — แทนที่จะมี worker รันค้างไว้ตลอด (แบบเอกสารนี้) ให้ Temporal Activity ของ BOP เองสั่งสร้าง Kubernetes Job (ephemeral container) ต่อการรันหนึ่งครั้งแทน scale-to-zero ได้จริงแต่แลกกับ cold-start latency — ดูตารางเทียบ 3 ทางเลือกที่เอกสารนั้น

**หลักการที่ใช้**: Temporal มีกลไก **task queue routing** ในตัวอยู่แล้ว — worker process ไหนก็ได้ (คนละ repo, คนละ binary, แม้แต่คนละภาษาโปรแกรมก็ได้ เพราะ Temporal ไม่สนใจ implementation ข้างใน) ที่ connect เข้า Temporal server เดียวกันและ poll task queue ชื่อที่ตกลงกันไว้ จะได้รับงานที่ BOP ส่งไปที่ queue นั้นโดยอัตโนมัติ — BOP (ฝั่งนี้) แค่ต้องรู้ชื่อ task queue เท่านั้น ไม่ต้องรู้จักหรือ import โค้ดของ remote worker เลยแม้แต่บรรทัดเดียว

## สิ่งที่ BOP ทำให้แล้ว (อ่านเพื่อเข้าใจ contract)

1. [internal/workflow/workflow.go](internal/workflow/workflow.go) — `Step.TaskQueue` (optional, `json:"task_queue,omitempty"`) — ถ้าระบุ Temporal จะ route activity call ของ step นั้นไปยัง worker ที่ poll queue ชื่อนี้แทน local `Activities.RunStep`
2. [internal/workflow/temporal.go](internal/workflow/temporal.go) — `ActivityName = "RunStep"` (constant), `TaskQueueName = "bop-workflows"` (task queue ของตัว workflow เอง — คนละเรื่องกับ `Step.TaskQueue` ของ activity แต่ละ step ข้างใน), `Namespace = "default"` — `Execute()` เรียก `workflow.ExecuteActivity(ctx, "RunStep", step)` พร้อม `ActivityOptions.TaskQueue = step.TaskQueue`
3. [internal/api/http/handler.go](internal/api/http/handler.go) — `POST /workflow-runs` รัน step ทันทีแบบ ad-hoc (ไม่ต้องสร้าง schedule) — ใช้ trigger job ที่ route ไป remote task queue ได้โดยไม่ต้องผ่าน `POST /bots/{name}/jobs` (ที่ไม่มี Temporal เกี่ยวข้องเลย route ข้าม queue ไม่ได้)

## Contract ที่ remote worker (repo ภายนอก) ต้อง implement

### 1. Connect Temporal server/namespace เดียวกับ BOP

Dev local: `localhost:7233`, namespace `"default"` (ดู `docker-compose.yml` ของ repo นี้ — ต้อง `docker compose up -d` ให้ Temporal server พร้อมก่อน) — production ต้องใช้ host:port/namespace เดียวกับที่ `cmd/bop`/`cmd/bop-worker` ต่ออยู่จริง (ดู roadmap Phase 2 เรื่อง containerize/deploy)

```go
c, err := client.Dial(client.Options{
    HostPort:  "localhost:7233",
    Namespace: "default",
})
```

### 2. เปิด Worker บน task queue ของตัวเอง (ชื่ออะไรก็ได้ ไม่ชนกับ `"bop-workflows"`)

```go
w := worker.New(c, "external-job-queue", worker.Options{}) // ชื่อ queue ต้องตรงกับที่ระบุใน Step.TaskQueue ตอนเรียกจากฝั่ง BOP
```

### 3. Register activity function ภายใต้ชื่อ **"RunStep"** ให้ตรงกับ `workflow.ActivityName` ของ BOP เป๊ะ

```go
w.RegisterActivityWithOptions(RunStep, activity.RegisterOptions{Name: "RunStep"}) // ชื่อ "RunStep" ต้องตรงตัวอักษรเป๊ะ ไม่งั้น Temporal ไม่รู้จักว่าให้เรียก activity ไหน
```

### 4. Input/output ของ activity function

**ไม่ต้อง import package `bop/internal/workflow` เลยก็ได้** — Temporal ใช้ JSON เป็น default DataConverter แค่ประกาศ struct ที่มี JSON shape ตรงกับ `workflow.Step` พอ (field ที่ไม่ใช้ไม่ต้องประกาศ `encoding/json` จะ ignore field ที่ไม่รู้จักตอน decode ให้เอง):

```go
type Step struct {
    BotName        string            `json:"bot_name"`
    Config         map[string]string `json:"config"`
    TimeoutSeconds int               `json:"timeout_seconds,omitempty"` // อาจไม่ต้องสนใจก็ได้ถ้า activity ฝั่งนี้ไม่ได้ยาวเกิน ActivityOptions.StartToCloseTimeout ที่ BOP ตั้งไว้
}

func RunStep(ctx context.Context, step Step) (string, error) {
    // step.Config["jobcode"], step.Config["extra_args"] ฯลฯ ตามที่ BOP ส่งมา (เหมือน bot.Config ของ internal/bots/externaljob)
    // ทำงานจริง แล้วคืน summary string สั้นๆ (จะไปโผล่ใน StepResult.Summary ที่ GET /workflow-runs/{id} คืนกลับ) กับ error (nil ถ้าสำเร็จ)
    ...
}
```

### 5. Error convention (non-retryable vs retryable)

BOP กำหนด `nonRetryableErrorTypes = []string{"ConfigError", "UnknownBot"}` ไว้ที่ [internal/workflow/temporal.go](internal/workflow/temporal.go) — ถ้า remote worker เจอ error ที่ retry กี่ครั้งก็ไม่มีทางสำเร็จ (เช่น config ที่ BOP ส่งมาผิด/ขาด) ควรห่อด้วย:

```go
return "", temporal.NewApplicationErrorWithCause("missing jobcode", "ConfigError", err)
```

ไม่งั้น Temporal จะถือเป็น error ธรรมดาแล้ว retry ตาม `RetryPolicy` default ไปเรื่อยๆ (เหมาะกับปัญหาชั่วคราว เช่น dependency ภายนอกของ job เองล่มชั่วคราว)

### 6. Optional: `activity.RecordHeartbeat` ถ้า job ใช้เวลานาน

ถ้า job ของ remote worker ใช้เวลานานใกล้ๆ หรือเกิน `Step.TimeoutSeconds` ที่ BOP ตั้งไว้ ควรเรียก `activity.RecordHeartbeat(ctx, ...)` เป็นระยะระหว่างทำงาน กัน Temporal เข้าใจผิดว่า worker ค้าง/ตายทั้งที่ยังทำงานอยู่จริง (pattern เดียวกับที่ยังค้างเป็น TODO ฝั่ง BOP เองใน [TEMPORAL-MIGRATION-TODO.md](TEMPORAL-MIGRATION-TODO.md) ข้อ 7)

## วิธี trigger จากฝั่ง BOP (ทดสอบตอน remote worker พร้อมแล้ว)

```bash
curl -X POST localhost:8080/workflow-runs -d '{
  "workflow_name": "send-report-remote",
  "steps": [{"bot_name": "external-report-job", "config": {"jobcode": "SEND_REPORT"}, "task_queue": "external-job-queue"}]
}'
curl localhost:8080/workflow-runs/<workflow_id>   # poll ผลลัพธ์
```

`bot_name` ในที่นี้แค่เป็น label ที่โผล่ใน `StepResult.BotName` เท่านั้น — **ไม่ได้ใช้ lookup อะไรฝั่ง BOP เลย** เมื่อ `task_queue` ถูกระบุมา (ต่างจาก step ที่ไม่มี `task_queue` ที่ `bot_name` ต้องตรงกับ bot ที่ `activities.Register` ไว้ใน `cmd/bop-worker` จริงๆ)

## TODO Checklist (ฝั่ง remote worker repo)

- [x] ตั้งชื่อ task queue ของตัวเอง (ไม่ชนกับ `"bop-workflows"`) — ใช้ `"external-job-queue"` (constant `taskQueueName` ใน [projects/remote-job-worker/cmd/worker/main.go](../remote-job-worker/cmd/worker/main.go)) เปลี่ยนได้ถ้าอยากได้ชื่ออื่น
- [x] Implement `worker.New` + `RegisterActivityWithOptions(..., Name: "RunStep")` ตามข้อ 1-3 ด้านบน — [projects/remote-job-worker/cmd/worker/main.go](../remote-job-worker/cmd/worker/main.go)
- [x] Implement activity function ตามข้อ 4 — [projects/remote-job-worker/internal/jobs/jobs.go](../remote-job-worker/internal/jobs/jobs.go) `RunStep` — **ยังเป็น placeholder jobcode** (`PING`/`LONG_TASK`/`FAIL_TRANSIENT`) เพราะยังไม่รู้ jobcode จริง เหลือแค่สลับ logic ข้างในเป็นของจริง ไม่ต้องแก้โครงรอบนอกเลย
- [x] ตัดสินใจ error convention ต่อ jobcode ตามข้อ 5 — จำลองไว้ครบทั้ง 2 แบบใน reference implementation: `ConfigError` (non-retryable, jobcode ผิด/ขาด) และ error ธรรมดา (retryable, `FAIL_TRANSIENT`)
- [ ] ทดสอบ trigger จาก BOP ผ่าน `POST /workflow-runs` ตามตัวอย่างด้านบน ยืนยันเห็นผลผ่าน `GET /workflow-runs/{id}` — **ยังไม่เคยรัน end-to-end จริงกับ Temporal server** (มีแค่ unit test ระดับ `TestActivityEnvironment` ที่ [projects/remote-job-worker/internal/jobs/jobs_test.go](../remote-job-worker/internal/jobs/jobs_test.go) ผ่านหมด — เหตุผลเดียวกับที่ endpoint กลุ่ม `/schedules`/`/workflow-runs` ของ BOP เองก็ยังไม่เคยทดสอบ end-to-end จริง)
- [x] พิจารณา `activity.RecordHeartbeat` ถ้า job ใช้เวลานาน (ข้อ 6) — สาธิตไว้ใน jobcode `LONG_TASK` (`runLongTask` เรียกทุก `heartbeatInterval` ระหว่างทำงาน พร้อม unit test ยืนยันว่ามีการเรียกจริงและหยุดทันทีถ้า context ถูกยกเลิก)

## Reference

- [internal/workflow/workflow.go](internal/workflow/workflow.go), [internal/workflow/temporal.go](internal/workflow/temporal.go), [internal/api/http/handler.go](internal/api/http/handler.go) — โค้ดฝั่ง BOP ที่เกี่ยวข้องโดยตรง
- [EXTERNAL-JOB-BOT-TODO.md](EXTERNAL-JOB-BOT-TODO.md) — ทางเลือกแรก (local exec ในเครื่อง/container เดียวกัน) ยังใช้งานคู่กันได้ เหมาะกับ binary ที่ไม่คุ้มแก้เป็น Temporal worker
- Temporal Go SDK docs: [`go.temporal.io/sdk/worker`](https://pkg.go.dev/go.temporal.io/sdk/worker), [`go.temporal.io/sdk/activity`](https://pkg.go.dev/go.temporal.io/sdk/activity)
- [FULLSTACK-INFRA-ROADMAP.md](../../FULLSTACK-INFRA-ROADMAP.md) — roadmap เต็มของ BOP
