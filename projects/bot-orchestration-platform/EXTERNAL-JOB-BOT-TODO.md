# TODO: External Job Runner Bot (รัน jobcode จาก Go binary ภายนอกผ่าน BOP)

> **สถานะ**: **Implement เสร็จแล้ว** (session ถัดจากที่เขียนแผนนี้ไว้) — Open Questions ทั้ง 6 ข้อถูกถามและตอบครบแล้ว (ดูหัวข้อด้านล่าง) โค้ดอยู่ที่ `internal/bots/externaljob/`, register แล้วทั้ง `cmd/bop/main.go` และ `cmd/bop-worker/main.go`, มี unit test ครบ, `go build`/`go vet`/`go test -race ./...` ผ่านหมด — สิ่งที่ยังไม่ทำ: ยังไม่ทดสอบกับ binary จริงของเจ้าของ repo (ยังไม่มีให้ทดสอบตอนเขียนโค้ดนี้ ใช้ path placeholder ไว้ก่อน) และยังไม่ containerize (Phase 2.1)

## บริบท

โปรเจกต์นี้คือ **Bot Orchestration Platform (BOP)** — ดูภาพรวมทั้งหมดที่ [FULLSTACK-INFRA-ROADMAP.md](../../FULLSTACK-INFRA-ROADMAP.md) และ [README.md](README.md) ตอนนี้ backend มี Bot execution engine + Temporal workflow/schedule + Schedule CRUD API ครบแล้ว (ดู [projects/bop-dashboard/](../bop-dashboard/README.md) ฝั่ง dashboard ด้วย)

เจ้าของ repo มี **Go binary แยกต่างหาก** (compile ไว้แล้ว ไม่ใช่ส่วนหนึ่งของ BOP) ที่มี "jobcode" หลายตัวรวมอยู่ในไฟล์เดียว — ปัจจุบันสั่งรันผ่าน shell script แบบส่ง flag เข้าไปตรงๆ เช่น

```bash
./your-binary --jobcode=SEND_REPORT
```

**งานนี้คือการห่อ binary ตัวนี้ให้เป็น bot ตัวใหม่ของ BOP** เพื่อให้สั่งรัน/schedule ผ่าน BOP ได้เหมือน bot อื่นๆ (`http-status-checker`) — **โดยไม่ต้องแก้โค้ด binary เดิมเลยแม้แต่บรรทัดเดียว** (BOP มองเป็น opaque external program ผ่าน stdin/stdout/exit code เท่านั้น)

## อ่านก่อนเริ่ม

1. [internal/bot/bot.go](internal/bot/bot.go) — `Bot` interface (`Name`/`ConfigSchema`/`Run`) ที่ต้อง implement, `ConfigField`/`ConfigFieldType`
2. [internal/bots/httpstatus/httpstatus.go](internal/bots/httpstatus/httpstatus.go) — ตัวอย่าง bot ที่ง่ายที่สุด ใช้เป็น template ได้เลย
3. [internal/worker/pool.go](internal/worker/pool.go) — ที่ job เดี่ยว (`POST /bots/{name}/jobs`) รันผ่าน — timeout ตอนนี้ hardcode `30*time.Second` ที่ `cmd/bop/main.go` (`worker.NewPool(4, 30*time.Second, ...)`) **อาจไม่พอสำหรับ job ภายนอกที่ใช้เวลานาน** ดู Open Question ข้อ 5
4. [internal/workflow/activity.go](internal/workflow/activity.go), [internal/workflow/temporal.go](internal/workflow/temporal.go) — ถ้าจะสั่งผ่าน workflow/schedule ด้วย (`ActivityOptions.StartToCloseTimeout` hardcode `30*time.Second` เหมือนกัน ปัญหาเดียวกับข้อ 3)
5. [internal/bot/errors.go](internal/bot/errors.go) — `bot.ConfigError`/`bot.NewConfigError` (ใช้ทำ error ประเภท non-retryable ถ้าจำเป็น ดู Open Question ข้อ 4)

## เป้าหมาย

สร้าง bot ใหม่ (ชื่อ package แนะนำ `internal/bots/externaljob` เปลี่ยนได้ถ้ามีชื่อที่สื่อความหมายกว่า) ที่:
1. **shell out** ไปเรียก binary ภายนอกด้วย `os/exec` (`exec.CommandContext`) ส่ง jobcode เป็น flag
2. **stream stdout/stderr กลับมาเป็น log แบบ real-time** ผ่าน `bot.LogFunc` (ให้เห็นผ่าน dashboard live monitor ที่ทำไว้แล้วใน Phase 1.5 ได้เลย)
3. **map exit code เป็นสำเร็จ/ล้มเหลว** ให้ `bot.Result`/`error`
4. ใช้งานได้ทั้ง 3 ทางที่ BOP มี: job เดี่ยว (`POST /bots/{name}/jobs`), workflow step, schedule (ผ่าน Schedule CRUD API ที่ทำไว้แล้ว)

## ตัดสินใจไว้แล้ว (คุยกับเจ้าของ repo แล้วตอนวางแผน — ไม่ต้องถามซ้ำ)

1. **Generic bot เดียว ไม่ทำ 1 bot ต่อ 1 jobcode** — `jobcode` เป็น `ConfigField` ธรรมดา (`required: true`, `type: "text"`) ข้อดี: เพิ่ม jobcode ใหม่ในไฟล์ binary เดิมทีหลัง **ไม่ต้องแก้/deploy BOP ใหม่เลย** แค่กรอกค่าใหม่ในฟอร์ม (เทียบกับทางเลือกที่ปัดตกไป: register 1 bot ต่อ 1 jobcode — โผล่ใน `GET /bots` แยกชื่อชัดเจนกว่า แต่ต้องแก้โค้ด+build ใหม่ทุกครั้งที่เพิ่ม jobcode)
2. **Shell out ผ่าน `exec.CommandContext`** ไม่ใช่วิธีอื่น (RPC, HTTP call ไปอีก service ฯลฯ) — เพราะ binary เดิมเป็น CLI process อยู่แล้ว `exec.CommandContext` ผูก context cancellation ให้ process ลูกถูก kill ให้อัตโนมัติเมื่อ timeout/ถูกยกเลิก โดยไม่ต้องเขียน logic เพิ่มเอง ตรงกับ contract เดิมของ `bot.Bot.Run` ("ต้องเช็ค `ctx.Done()`... หยุดทำงานทันทีถ้า context ถูกยกเลิก")
3. **ไม่แก้โค้ด binary เดิมเลย** — ยึดหลักการเดิมของทั้งโปรเจกต์ (ดู CLAUDE.md: engine ทั่วไปแบบ pluggable ไม่ผูกกับ business logic เฉพาะเจาะจง) bot นี้คือ "ความสามารถรัน external job code" แบบทั่วไป ไม่ใช่ bot เฉพาะของ jobcode ตัวใดตัวหนึ่ง

## Open Questions (ตอบครบแล้ว — ถามเจ้าของ repo ตอน implement)

1. **Path ของ binary** — **ยังไม่มี binary จริง ใช้ path placeholder ไปก่อน** — เก็บเป็น `defaultBinaryPath` (constant) ใน `internal/bots/externaljob/externaljob.go` ต้องเปลี่ยนค่านี้ให้ตรง path จริงก่อนใช้งานจริง และต้อง copy เข้า Dockerfile ทั้ง `cmd/bop`/`cmd/bop-worker` ตอน containerize (Phase 2.1 — ยังไม่ทำ)
2. **Jobcode มีค่าอะไรบ้าง (enumerable ไหม)** — **Free text ไปก่อน (YAGNI)** — ไม่ได้ขยาย `ConfigFieldType` เพิ่ม type `"select"` เพิ่ม dropdown ทีหลังได้ถ้าจำเป็นจริง
3. **Parameter อื่นนอกจาก jobcode** — **มี ใช้ free-text field `extra_args` เพิ่ม** — คั่นด้วยช่องว่าง แล้ว split ด้วย `strings.Fields` ต่อท้าย `--jobcode=<code>` เป็น argument เพิ่มให้ binary
4. **วิธีรู้ว่า job สำเร็จ/ล้มเหลว** — **แค่ exit code พอ (`0` = สำเร็จ, อื่นๆ = ล้มเหลว) ไม่ parse stdout/stderr เพิ่ม** — error ที่เกิดจาก exit code ไม่ใช่ 0 เป็น error ธรรมดา (retryable) ทั้งหมด ส่วน `bot.ConfigError` (non-retryable) สงวนไว้เฉพาะกรณี config ที่ผู้ใช้กรอกผิด/ขาด (เช่นไม่มี jobcode) ตาม pattern เดียวกับ `httpstatus`
5. **เวลาที่ job แต่ละตัวใช้** — **ต้องกำหนดได้แบบ dynamic ต่อ job แต่ละครั้งที่สร้าง (full per-job timeout)** — เพิ่ม `job.Job.TimeoutSeconds`/`workflow.Step.TimeoutSeconds` (optional, `0` = ใช้ default 30 วินาทีเดิม) ให้ `worker.Pool.run()` และ `internal/workflow/temporal.go` (`Execute`) อ่านค่านี้แทน timeout กลางตายตัว — ระบุผ่าน query param `timeout_seconds` สำหรับ job เดี่ยว (`POST /bots/{name}/jobs?timeout_seconds=120`) และ field `timeout_seconds` ใน `workflow.Step` สำหรับ workflow/schedule (ดูรายละเอียดที่ README.md หัวข้อ "Bot: external-job") — `activity.RecordHeartbeat` **ยังไม่ implement** (ยังไม่มี jobcode ตัวไหนที่รู้ว่านานพอที่จะจำเป็นจริง รอ binary จริงมาทดสอบก่อน)
6. **Working directory / environment variable** — **ไม่ต้องการอะไรพิเศษ** — รันจาก working directory ปกติ ไม่ set `cmd.Env` เพิ่ม (inherit จาก process แม่ตาม default ของ `exec.Cmd`)

## TODO Checklist

### 1. ตอบ Open Questions ทั้งหมดข้างบนก่อน

- [x] คุยกับเจ้าของ repo ให้ครบทั้ง 6 ข้อ แล้วอัปเดตเอกสารนี้ด้วยคำตอบก่อนเริ่มโค้ดจริง

### 2. ออกแบบ + Implement Bot

- [x] สร้าง `internal/bots/externaljob/externaljob.go` — implement `bot.Bot` (`Name()`, `ConfigSchema()`, `Run()`) ตาม pattern ของ `internal/bots/httpstatus/httpstatus.go`
- [x] `ConfigSchema()` — มี field `jobcode` (`required: true`, `type: bot.ConfigFieldText`) และ `extra_args` (optional, ตามคำตอบ Open Question ข้อ 3)
- [x] `Run()` — `exec.CommandContext(ctx, binaryPath, "--jobcode="+cfg["jobcode"], extraArgs...)` ต่อ `cmd.StdoutPipe()`/`cmd.StderrPipe()` เข้ากับ `log()` แบบ real-time ผ่าน `bufio.Scanner` คนละ goroutine กันระหว่าง stdout/stderr — เพิ่มเติมจากแผนเดิม: ต้องตั้ง `Setpgid: true` + kill ทั้ง process group ตอน cancel ด้วย (ไม่ใช่แค่ default ของ `exec.CommandContext`) ไม่งั้น grandchild process ที่ binary อาจเปิดต่อเองจะถือ pipe ค้างไว้จน log streaming ไม่จบ (เจอจริงจาก test ข้อ 5 ด้านล่าง)
- [x] จัดการ exit code ที่ไม่ใช่ 0 เป็น error ธรรมดา (retryable) ตามคำตอบ Open Question ข้อ 4 — `bot.NewConfigError` สงวนไว้เฉพาะ jobcode ที่ขาดหายไปเลย
- [x] path ของ binary hardcode เป็น constant `defaultBinaryPath` ในไฟล์นี้ (ยังเป็น placeholder เพราะยังไม่มี binary จริง — ดู Open Question ข้อ 1)

### 3. Register bot เข้าทั้งสอง process

- [x] `pool.Register(externaljob.New())` ใน `cmd/bop/main.go` (สำหรับ job เดี่ยวผ่าน `POST /bots/{name}/jobs`)
- [x] `activities.Register(externaljob.New())` ใน `cmd/bop-worker/main.go` (สำหรับ workflow/schedule)

### 4. Timeout (Open Question ข้อ 5: ต้องเป็น full per-job timeout)

- [x] เพิ่ม per-job timeout เต็มรูปแบบ (ไม่ใช่แค่ per-bot) — `job.Job.TimeoutSeconds` + `worker.Pool.run()` อ่านค่านี้แทน `p.timeout` กลางถ้ามี, `workflow.Step.TimeoutSeconds` + `internal/workflow/temporal.go` (`Execute`) คำนวณ `ActivityOptions` ใหม่ต่อ step แทนที่จะตั้งครั้งเดียวนอก loop — ดูรายละเอียดที่ README.md หัวข้อ "Timeout ต่อ job/step"
- [ ] `activity.RecordHeartbeat` — ยังไม่ implement (รอ binary จริงมาทดสอบก่อนว่าจำเป็นแค่ไหน)

### 5. Testing

- [x] unit test ของ `externaljob.Run` โดย mock ด้วย shell script ปลอม (`internal/bots/externaljob/externaljob_test.go`, ฟังก์ชัน `fakeScript`) ไม่พึ่ง binary จริงเลย
- [x] ทดสอบว่า context ถูกยกเลิกแล้ว process ลูก (รวม grandchild) ถูก kill จริง (`TestRun_ContextCancelKillsProcess`)
- [x] ทดสอบ log streaming ครบทั้ง stdout/stderr (`TestRun_StreamsBothStdoutAndStderr`)
- [x] เพิ่ม `TestPool_PerJobTimeoutOverridesDefault` ใน `internal/worker/pool_test.go` ทดสอบว่า per-job timeout override ค่ากลางของ pool ได้จริง

### 6. Docs

- [x] อัปเดต [README.md](README.md) — บอกว่ามี bot ใหม่ + ตัวอย่าง config schema + curl example สร้าง job + หัวข้อ timeout ต่อ job/step
- [ ] ถ้าถึง containerize จริง (Phase 2.1) ต้อง copy binary ภายนอกเข้า Dockerfile ของทั้ง `cmd/bop` และ `cmd/bop-worker` (ยังไม่ทำ — ยังไม่มี Dockerfile ของ process พวกนี้เลยตอนนี้)

## Reference

- โค้ดปัจจุบันที่เกี่ยวข้องโดยตรง: [internal/bot/bot.go](internal/bot/bot.go), [internal/bot/errors.go](internal/bot/errors.go), [internal/bots/httpstatus/httpstatus.go](internal/bots/httpstatus/httpstatus.go), [internal/worker/pool.go](internal/worker/pool.go), [internal/workflow/activity.go](internal/workflow/activity.go), [internal/workflow/temporal.go](internal/workflow/temporal.go)
- Go stdlib ที่ต้องใช้: [`os/exec`](https://pkg.go.dev/os/exec) โดยเฉพาะ `exec.CommandContext`, `Cmd.StdoutPipe`, `Cmd.StderrPipe`
- [TEMPORAL-MIGRATION-TODO.md](TEMPORAL-MIGRATION-TODO.md) ข้อ 7 — เรื่อง `activity.RecordHeartbeat` ที่ยังค้างอยู่ (เกี่ยวกับ Open Question ข้อ 5 ของเอกสารนี้)
- [FULLSTACK-INFRA-ROADMAP.md](../../FULLSTACK-INFRA-ROADMAP.md) — roadmap เต็มของ BOP
