# TODO: Phase 3 — ฟีเจอร์ระดับ Enterprise แบบ Control-M (Design เต็ม ยังไม่ Implement)

> **สถานะ**: เป็นแค่ design doc เขียนไว้ล่วงหน้า **ยังไม่เริ่ม implement เลยสักฟีเจอร์** — เขียนตาม [FULLSTACK-INFRA-ROADMAP.md](../../FULLSTACK-INFRA-ROADMAP.md) Phase 3.1-3.7 แต่ขยายรายละเอียดทางเทคนิคเต็มรูปแบบ (data model, Temporal pattern, API, Open Questions) session ที่หยิบไปทำต่อ **ควรอ่านหัวข้อ "ลำดับที่แนะนำให้ทำ" ท้ายเอกสารก่อน** เพราะลำดับที่แนะนำไม่ตรงกับลำดับ 3.1→3.7 ของ roadmap เป๊ะ (มีเหตุผลเรื่อง dependency ระหว่างฟีเจอร์กำกับไว้)

**คำเตือนสำคัญที่สุด** (พูดตรงๆ ตาม [FEATURES.md](FEATURES.md)): Phase 3 ทั้งหมดนี้เป็น "ฟีเจอร์ enterprise ทับบน infra ที่มีอยู่" — ถ้า **PostgreSQL persistence** (ยังไม่ทำ, `job.Repository`/`schedule.Repository` ยังเป็น in-memory ทั้งคู่) ยังไม่เสร็จ ฟีเจอร์ในเอกสารนี้แทบทุกตัวจะมีปัญหา "ข้อมูลหายเมื่อ restart" ซ้อนเข้าไปอีกชั้น อย่าเริ่ม implement ฟีเจอร์ในเอกสารนี้ก่อน persistence จริงเสร็จ

## สารบัญ

1. [3.1 Condition-Based Job Triggering](#31-condition-based-job-triggering)
2. [3.2 Resource Pools](#32-resource-pools)
3. [3.3 Flow Control](#33-flow-control)
4. [3.4 Approval Gate (Human-in-the-Loop)](#34-approval-gate-human-in-the-loop)
5. [3.5 Gantt / Timeline Visualization](#35-gantt--timeline-visualization)
6. [3.6 Alert Triage / Ownership](#36-alert-triage--ownership)
7. [3.7 Job Definition Versioning + Environment Promotion](#37-job-definition-versioning--environment-promotion)
8. [ตารางสรุป Dependency ข้ามฟีเจอร์](#ตารางสรุป-dependency-ข้ามฟีเจอร์)
9. [ลำดับที่แนะนำให้ทำ](#ลำดับที่แนะนำให้ทำ)

---

## 3.1 Condition-Based Job Triggering

### Concept (ต่างจาก DAG ตรงไหน)

Control-M ไม่ผูก job ต่อ job แบบ parent-child ตรงๆ (แบบ Airflow DAG) แต่ใช้ **named condition**: job หนึ่งจบแล้ว "set" condition (เช่น `"FILE_READY"`) job อื่นที่ "wait for" condition นั้นถึงจะเริ่มได้ — ยืดหยุ่นกว่า DAG เพราะเป็น many-to-many (หลาย job รอ condition เดียวกันได้, 1 job รอหลาย condition พร้อมกันได้) และ **decouple กันข้าม schedule** (job A กับ job B ไม่ต้องอยู่ workflow เดียวกันเลยด้วยซ้ำ)

### Data Model

```go
// internal/condition/condition.go (package ใหม่)
type Condition struct {
    Name    string    `json:"name"`     // เช่น "FILE_READY-20260101" (ดูเรื่อง date-scoping ด้านล่าง)
    SetBy   string    `json:"set_by"`   // Temporal WorkflowID ที่ set condition นี้ (audit trail)
    SetAt   time.Time `json:"set_at"`
}

type Repository interface {
    Set(ctx context.Context, name, setBy string) error
    Get(ctx context.Context, name string) (*Condition, bool, error)
    Delete(ctx context.Context, name string) error // เทียบ Control-M "manual condition delete"
    List(ctx context.Context) ([]*Condition, error)
}
```

**Date-scoping (ODATE ของ Control-M)**: condition ของจริงใน Control-M ผูกกับ "วันที่ต้นฉบับของ run" (ODATE) เสมอ ป้องกัน condition ของเมื่อวานรั่วมาปนกับวันนี้ — เอกสารนี้ **ไม่บังคับ** เพราะเพิ่มความซับซ้อนที่ยังไม่มี use case จริงต้องการ (YAGNI) แต่ **เปิดช่องให้ทำได้**: ผู้ใช้ที่อยากได้ date-scoping ตั้งชื่อ condition เองด้วย pattern `{name}-{{date}}` แล้ว BOP ไม่ต้องรู้เรื่องวันที่เลย (string ธรรมดา) — ถ้าจำเป็นจริงค่อยเพิ่ม template helper ทีหลัง

### Integration กับ Workflow ที่มีอยู่

```go
// internal/workflow/workflow.go — เพิ่ม field ใหม่
type Step struct {
    // ...fields เดิม (BotName, Config, TimeoutSeconds, TaskQueue, K8sJob)...
    SetConditions []string `json:"set_conditions,omitempty"` // condition ที่จะ set หลัง step นี้สำเร็จ
}

type Workflow struct {
    Name  string `json:"name"`
    Steps []Step `json:"steps"`
    WaitForConditions []string `json:"wait_for_conditions,omitempty"` // condition ที่ workflow ทั้งก้อนต้องรอก่อนเริ่ม step แรก
}
```

### Temporal Pattern: MVP ด้วย Polling (แนะนำ) vs Signal-Push (ทีหลัง)

**MVP — polling ธรรมดา** (สอดคล้องกับ pattern ที่ใช้อยู่แล้วใน `internal/workflow/k8sjob.go` — `awaitJobCompletion` poll สถานะทุก `pollInterval`):

```go
// ใน Execute() (temporal.go) ก่อน loop ของ Steps
if len(wf.WaitForConditions) > 0 {
    if err := awaitConditions(ctx, wf.WaitForConditions); err != nil {
        return nil, err
    }
}

// awaitConditions เรียก Activity เล็กๆ (CheckConditions) วนซ้ำด้วย workflow.Sleep ระหว่างรอบ
// — ต้องเป็น Activity เพราะ workflow function เรียก DB ตรงๆ ไม่ได้ (ผิดกฎ deterministic)
func awaitConditions(ctx workflow.Context, names []string) error {
    for {
        var allMet bool
        ao := workflow.ActivityOptions{StartToCloseTimeout: 10 * time.Second}
        actx := workflow.WithActivityOptions(ctx, ao)
        if err := workflow.ExecuteActivity(actx, CheckConditionsActivityName, names).Get(actx, &allMet); err != nil {
            return err
        }
        if allMet {
            return nil
        }
        if err := workflow.Sleep(ctx, 10*time.Second); err != nil { // poll interval — ปรับได้ตาม use case
            return err
        }
    }
}
```

ข้อเสีย MVP: workflow "ตื่น" มาเช็คทุก 10 วินาทีแม้ไม่มีอะไรเปลี่ยน (เปลือง Activity execution/history event ถ้ารอนาน) — ยอมรับได้สำหรับ MVP เพราะ implement ง่ายกว่า signal-push มาก (ไม่ต้องมี "waiter registry" แยก)

**ทางเลือกที่ดีกว่าทีหลัง — Signal-push**: ตอน `SetConditions` ของ step ไหนสำเร็จ ให้ Activity ที่ set condition ไป query "ใครกำลังรอ condition นี้อยู่" จาก `condition_waiters` table (workflow ID ที่ subscribe ไว้ตอนเข้า `awaitConditions`) แล้วยิง `client.SignalWorkflow` หา workflow นั้นตรงๆ แทนที่จะให้ทุก workflow ที่รออยู่ poll เอง — เร็วกว่า/ประหยัดกว่า แต่ซับซ้อนกว่ามาก (ต้องมี registry + คิดเรื่อง race condition ตอน register/set พร้อมกันพอดี) **ไม่แนะนำทำ MVP รอบแรก**

### API เพิ่มเติม

```
GET    /conditions              list condition ทั้งหมดที่ set ไว้ (debug/ops)
POST   /conditions              set condition มือ (เทียบ Control-M "manual condition set", ใช้ตอน integrate กับระบบภายนอกที่ set condition ไม่ผ่าน BOP workflow)
DELETE /conditions/{name}       ลบ condition มือ
```

### Open Questions

1. Condition store ควรเป็น table แยกใน PostgreSQL หรือ Redis (TTL ธรรมชาติ ถ้าอยากให้ condition หมดอายุอัตโนมัติ)?
2. ต้องมี UI ให้ดู "workflow ไหนกำลังรอ condition อะไรอยู่" ไหม (debug ยากถ้าไม่มี — เหมือน deadlock ที่มองไม่เห็น)?
3. Timeout ของการรอ condition — รอไม่มีที่สิ้นสุดได้ไหม หรือต้องมี max wait time แล้ว fail workflow?

---

## 3.2 Resource Pools

### Concept

Control-M มี "quantitative resource" จำกัดว่ากี่ job พร้อมกันที่แตะ resource เดียวกันได้ (เช่น "database-prod": max 5, "website-x": max 3) ป้องกันปลายทาง overload — คนละเรื่องกับ Temporal `MaxConcurrentActivityExecutionSize` (จำกัด "worker" ไม่ใช่จำกัด "resource เป้าหมาย")

### Data Model

```go
// internal/resourcepool/resourcepool.go (package ใหม่)
type PoolConfig struct {
    Name     string `json:"name"`     // เช่น "website-x"
    Capacity int    `json:"capacity"` // จำนวน unit สูงสุดที่ใช้พร้อมกันได้
}

type Repository interface {
    Get(ctx context.Context, name string) (*PoolConfig, error)
    List(ctx context.Context) ([]*PoolConfig, error)
    Upsert(ctx context.Context, cfg *PoolConfig) error
}
```

```go
// internal/workflow/workflow.go — เพิ่ม field ใหม่ใน Step
type ResourcePoolRequest struct {
    Name  string `json:"name"`
    Units int    `json:"units,omitempty"` // default 1 ถ้าไม่ระบุ (เทียบ Control-M ที่ job ขอ resource ได้มากกว่า 1 unit)
}

type Step struct {
    // ...fields เดิม...
    ResourcePools []ResourcePoolRequest `json:"resource_pools,omitempty"`
}
```

### จุดสำคัญที่สุด: In-Memory Semaphore ใช้ได้แค่ Worker เดียว

`worker.Pool` (internal/worker/pool.go) ใช้ Go buffered channel เป็น semaphore อยู่แล้ว (`sem chan struct{}`) — pattern เดียวกันเอามาทำ resource pool ได้ตรงๆ **แต่ใช้ได้แค่ตอน `cmd/bop-worker` มี instance เดียว** — พอ scale เป็นหลาย replica (ตามแผน HPA ใน Phase 2.5) semaphore ในหน่วยความจำของแต่ละ replica **ไม่รู้จักกัน** ทำให้ resource pool รั่ว (แต่ละ replica คิดว่าตัวเองมี capacity เต็มแยกกัน รวมกันแล้วเกิน limit จริง) — **ต้องใช้ Redis เป็น distributed semaphore** ก่อนถึงจะถูกต้องตอน scale ออก (ตรงกับแผน Phase 2.4 Redis ที่วางไว้อยู่แล้ว: "Redis... ทำ rate limiter (token bucket) จำกัดความถี่ที่แต่ละ bot ยิง request")

```go
// ตัวอย่าง atomic acquire ด้วย Redis (แนวคิด ไม่ใช่โค้ดสมบูรณ์)
// ใช้ Lua script ให้ atomic (check-and-increment ต้องเป็น atomic operation เดียว ไม่งั้น race กันได้)
const acquireScript = `
local current = tonumber(redis.call('GET', KEYS[1]) or '0')
local capacity = tonumber(ARGV[1])
local units = tonumber(ARGV[2])
if current + units > capacity then
  return 0
end
redis.call('INCRBY', KEYS[1], units)
return 1
`
```

### Temporal Pattern

Acquire/Release เป็น Activity คู่กัน เรียกจาก `Execute()` ห่อ Activity หลักของ step ไว้ (Go `defer` ใช้ได้ปกติใน workflow function เพราะ `ExecuteActivity(...).Get(...)` เป็น synchronous call จากมุมมองโค้ด):

```go
for _, req := range step.ResourcePools {
    if err := workflow.ExecuteActivity(stepCtx, AcquireResourcePoolActivityName, req).Get(stepCtx, nil); err != nil {
        return results, err // ขอ resource ไม่ได้ (timeout ระหว่างรอคิว) — หยุด workflow เหมือน step ปกติ fail
    }
    defer func(r ResourcePoolRequest) {
        // ใช้ context ใหม่ไม่ผูกกับ stepCtx เดิม (เผื่อ stepCtx ถูกยกเลิกไปแล้วตอน release)
        _ = workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, releaseOptions), ReleaseResourcePoolActivityName, r).Get(ctx, nil)
    }(req)
}
```

`AcquireResourcePoolActivityName` ควรมี retry policy ที่ "รอคิว" ได้ (retry ด้วย backoff จนกว่าจะ acquire สำเร็จ หรือ timeout) — ไม่ใช่ fail ทันทีถ้า pool เต็ม

### Open Questions

1. Pool capacity config ผ่าน API (`PUT /resource-pools/{name}`) หรือ config file ที่ deploy พร้อม BOP?
2. ถ้า workflow ถูกยกเลิกกลางทางระหว่างถือ resource อยู่ — release ทันเวลาเสมอไหม (ต้องทดสอบกับ Temporal cancellation semantics ให้ดี)
3. Deadlock: step A ถือ resource X รอ resource Y พร้อมกับ step B (คนละ workflow) ถือ Y รอ X — ต้องมี timeout ป้องกันไหม (แนะนำ: บังคับให้ Acquire Activity มี timeout เสมอ ไม่ให้รอไม่มีที่สิ้นสุด)

---

## 3.3 Flow Control

### Concept

Control-M มี "SMART Folder" จัดกลุ่ม job หลายตัว (ที่อาจเป็นคนละ schedule กันเลย) เป็น "Flow" เดียว แล้ว hold/resume/kill ทั้งกลุ่มพร้อมกันได้ — **นี่คือฟีเจอร์ที่มี risk ต่ำที่สุดในเอกสารนี้** เพราะไม่ต้องแตะ Temporal workflow/activity logic เลย เป็นแค่ **aggregation layer บน Schedule CRUD ที่มีอยู่แล้ว 100%**

### Data Model

```go
// internal/flow/flow.go (package ใหม่ — pattern เดียวกับ internal/schedule/schedule.go)
type Flow struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    ScheduleIDs []string  `json:"schedule_ids"` // Temporal Schedule ID ที่เป็นสมาชิกของ flow นี้
    CreatedAt   time.Time `json:"created_at"`
}

type Repository interface {
    Create(ctx context.Context, f *Flow) error
    Get(ctx context.Context, id string) (*Flow, error)
    List(ctx context.Context) ([]*Flow, error)
    Delete(ctx context.Context, id string) error
}
```

### API (ห่อ `client.ScheduleClient` เดิมทั้งหมด — ไม่มี logic ใหม่)

```
POST   /flows                  {"id", "name", "schedule_ids": [...]}
GET    /flows                  list
GET    /flows/{id}             รายละเอียด + สถานะ aggregate (ดูด้านล่าง)
DELETE /flows/{id}             ลบ grouping (ไม่ลบ schedule สมาชิก)
POST   /flows/{id}/pause       วน pause ทุก schedule สมาชิก
POST   /flows/{id}/unpause     วน unpause ทุก schedule สมาชิก
POST   /flows/{id}/trigger     วน trigger ทุก schedule สมาชิก
```

```go
// handleFlowPause (ร่างคร่าวๆ)
func (s *Server) handleFlowPause(w http.ResponseWriter, r *http.Request) {
    f, err := s.flowRepo.Get(r.Context(), r.PathValue("id"))
    // ...
    for _, scheduleID := range f.ScheduleIDs {
        _ = s.temporal.ScheduleClient().GetHandle(r.Context(), scheduleID).Pause(r.Context(), client.SchedulePauseOptions{})
        // เก็บ error รายตัวไว้รายงานกลับ ไม่ fail ทั้งก้อนถ้าตัวใดตัวหนึ่ง error (schedule อาจถูกลบไปแล้วนอกระบบ)
    }
}
```

### Flow Status Aggregation

`GET /flows/{id}` ต้อง Describe ทุก schedule สมาชิก + ดู workflow run ล่าสุดของแต่ละตัว แล้ว derive สถานะรวม:

```go
func aggregateFlowStatus(memberStatuses []string) string {
    if contains(memberStatuses, "failed") { return "failed" }
    if contains(memberStatuses, "running") { return "running" }
    if allEqual(memberStatuses, "succeeded") { return "succeeded" }
    return "mixed"
}
```

### Open Questions

1. Schedule หนึ่งอยู่ได้หลาย Flow พร้อมกันไหม (many-to-many) หรือ 1 schedule อยู่ได้ Flow เดียว?
2. `POST /flows/{id}/pause` ที่ schedule สมาชิกบางตัว pause ไม่สำเร็จ (เช่นถูกลบไปแล้ว) ควร fail ทั้ง request หรือรายงาน partial success?

---

## 3.4 Approval Gate (Human-in-the-Loop)

> **ฟีเจอร์ที่คุ้มพูดในสัมภาษณ์ที่สุดในเอกสารนี้** — เป็น Temporal pattern ขั้นสูงที่ portfolio ทั่วไปไม่มี (workflow.Selector race ระหว่าง signal กับ timer)

### Concept

Control-M รองรับ job ที่ต้องรอ manual confirmation ก่อนรันต่อ (เช่น "ยืนยันก่อน deploy เข้า production") — workflow "ค้าง" รอ input จากมนุษย์กลางทาง ไม่ใช่ retry/fail อัตโนมัติ

### Data Model

```go
// internal/workflow/workflow.go
type Step struct {
    // ...fields เดิม...
    RequireApproval        bool  `json:"require_approval,omitempty"`
    ApprovalTimeoutSeconds int   `json:"approval_timeout_seconds,omitempty"` // 0 = ใช้ default (เช่น 24 ชั่วโมง)
}

type StepResult struct {
    // ...fields เดิม (StepIndex, BotName, Summary, Error)...
    PendingApproval bool `json:"pending_approval,omitempty"` // true ระหว่างที่ query "steps" ถูกเรียกตอน step นี้กำลังรอ approve อยู่
}
```

### Temporal Pattern — หัวใจของฟีเจอร์นี้

```go
// internal/workflow/temporal.go
const defaultApprovalTimeout = 24 * time.Hour

type ApprovalSignal struct {
    Approved   bool   `json:"approved"`
    ApprovedBy string `json:"approved_by"`
    Comment    string `json:"comment,omitempty"`
}

func approvalSignalName(stepIndex int) string {
    return fmt.Sprintf("approval-step-%d", stepIndex)
}

// awaitApproval ใช้ workflow.Selector "แข่ง" ระหว่าง 2 เหตุการณ์: signal จากมนุษย์มาก่อน
// (Approve/Reject) หรือ timer หมดเวลาก่อน (ถือว่า reject โดย default — เทียบ Control-M ที่
// job ค้างรอ approval นานเกินไปมักถูกตั้งให้ fail อัตโนมัติ ไม่ใช่ค้างตลอดไป)
func awaitApproval(ctx workflow.Context, stepIndex int, timeout time.Duration) (ApprovalSignal, error) {
    ch := workflow.GetSignalChannel(ctx, approvalSignalName(stepIndex))
    selector := workflow.NewSelector(ctx)

    var result ApprovalSignal
    timerCtx, cancelTimer := workflow.WithCancel(ctx)
    timer := workflow.NewTimer(timerCtx, timeout)

    selector.AddFuture(timer, func(f workflow.Future) {
        result = ApprovalSignal{Approved: false, Comment: "timed out waiting for approval"}
    })
    selector.AddReceive(ch, func(c workflow.ReceiveChannel, more bool) {
        c.Receive(ctx, &result)
        cancelTimer()
    })
    selector.Select(ctx) // block จนกว่าอันใดอันหนึ่งพร้อม

    return result, nil
}
```

ใน `Execute()` loop เพิ่มก่อนเรียก activity ปกติของ step:

```go
if step.RequireApproval {
    timeout := defaultApprovalTimeout
    if step.ApprovalTimeoutSeconds > 0 {
        timeout = time.Duration(step.ApprovalTimeoutSeconds) * time.Second
    }
    signal, err := awaitApproval(ctx, i, timeout)
    if err != nil {
        return results, err
    }
    if !signal.Approved {
        res.Error = fmt.Sprintf("rejected by %s: %s", signal.ApprovedBy, signal.Comment)
        results = append(results, res)
        return results, errors.New(res.Error)
    }
}
```

**ทำไม `workflow.Selector` ไม่ใช่แค่ `ch.Receive(ctx, &signal)` เฉยๆ**: ถ้ารอ signal เฉยๆ ไม่มี timeout, workflow จะค้างรอ**ไม่มีที่สิ้นสุด**ถ้าไม่มีใครมา approve/reject เลย (Temporal workflow ไม่มี timeout ตายตัวเหมือน HTTP request — ค้างได้เป็นปีถ้าไม่ตั้งเอง) `Selector` คือกลไกมาตรฐานของ Temporal สำหรับ "รอ event ใดก่อนก็ได้ระหว่างหลาย source พร้อมกัน" (เทียบเท่า `select` ของ Go ธรรมดา แต่เป็นเวอร์ชัน deterministic ที่ replay ได้ปลอดภัย)

### API

```
POST /workflow-runs/{id}/steps/{index}/approve   {"comment": "..."} (ดึงชื่อผู้อนุมัติจาก JWT session)
POST /workflow-runs/{id}/steps/{index}/reject    {"comment": "..."}
```

```go
func (s *Server) handleApproveStep(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    stepIndex, _ := strconv.Atoi(r.PathValue("index"))
    username := ... // จาก JWT claims เดียวกับ handleMe

    var req struct{ Comment string `json:"comment"` }
    _ = json.NewDecoder(r.Body).Decode(&req)

    err := s.temporal.SignalWorkflow(r.Context(), id, "", approvalSignalName(stepIndex), workflow.ApprovalSignal{
        Approved: true, ApprovedBy: username, Comment: req.Comment,
    })
    // ...
}
```

Dashboard: `GET /workflow-runs/{id}` (มีอยู่แล้ว) ต้องเพิ่ม `PendingApproval` เข้า `StepResult` ที่คืนจาก query handler `"steps"` — dashboard poll endpoint นี้อยู่แล้วสำหรับ live monitor เห็น flag นี้แล้วโชว์ปุ่ม Approve/Reject ได้ทันทีไม่ต้องเพิ่ม endpoint ใหม่ฝั่ง read

### Open Questions

1. **ใครมีสิทธิ์ approve** — ตอนนี้ auth เป็น single-user (ทุกคน login ได้สิทธิ์เท่ากัน) approval gate จะมีความหมายจริงจังก็ต่อเมื่อมี RBAC (Phase 2.3) แยกว่าใครเป็น "approver" ได้บ้าง — MVP (ไม่มี RBAC) ทุกคนที่ login approve ได้หมด ต้องบอกไว้ตรงๆ ว่าเป็นข้อจำกัดถ้า demo ฟีเจอร์นี้
2. Reject แล้วอยาก retry step เดิมใหม่ (ไม่ fail ทั้ง workflow) ทำได้ไหม หรือ reject = fail workflow เสมอ?
3. ต้องเก็บ audit log ของการ approve/reject แยกจาก Temporal event history เองไหม (Temporal event history มีอยู่แล้วจริงๆ ผ่าน signal event — อาจไม่ต้องทำซ้ำ)

---

## 3.5 Gantt / Timeline Visualization

### Concept

Control-M มี Forecast view แสดง timeline ของ job ที่จะรัน/รันไปแล้วแบบ Gantt chart — **ฟีเจอร์นี้แทบไม่มีงาน backend ใหม่เลย** เพราะข้อมูลที่ต้องใช้มีอยู่แล้วทั้งหมด:

- อดีต/ปัจจุบัน: `GET /workflow-runs` (มี `StartedAt`/`EndedAt`)
- อนาคต: `GET /schedules` (มี `next_action_times` จาก Temporal `ScheduleListEntry.NextActionTimes` อยู่แล้ว)

### แนะนำ: ไม่สร้าง backend aggregation endpoint ใหม่ (YAGNI)

Frontend รวมสอง response นี้เข้าด้วยกันเองได้ (ยิง 2 request ขนาน แล้ว merge บน client) — เพิ่ม backend endpoint ใหม่แค่เพื่อ "รวมของที่มีอยู่แล้ว" เป็นความซับซ้อนที่ไม่จำเป็น (ตรงข้ามกับ Flow Control ข้างบนที่ backend aggregation จำเป็นจริงเพราะต้องยิงหลาย schedule พร้อมกันแล้วคำนวณ derived state) — ถ้าพบว่า frontend ต้อง fetch บ่อยจน perf มีปัญหาจริงค่อยพิจารณา endpoint รวมทีหลัง

### งานหลักคือฝั่ง Dashboard

- อ่าน [FEATURES.md](FEATURES.md) หัวข้อคำแนะนำ #5 (เติม UI dispatch mode) ประกอบ — ถ้าทำพร้อมกันจะเห็นภาพ job ที่รันผ่าน local/remote/K8s ต่างสี/ต่าง icon กันในผังเดียวได้เลย
- ใช้ **skill `dataviz`** ของ Claude Code (มีอยู่ใน environment นี้) ตอนจะออกแบบ chart จริง — มี design system/palette validator ให้พร้อม ไม่ต้องคิดสีเอง
- Timeline component: ใช้ library สำเร็จรูป (เช่น `vis-timeline`, หรือสร้างเองด้วย CSS grid ตามที่ roadmap แนะนำ) — เป็น scope การตัดสินใจของ Phase 1 dashboard ไม่ใช่ backend

### Open Questions

1. ต้องแสดง "ระยะเวลาเฉลี่ยที่คาดว่า job จะใช้" ด้วยไหม (ต้องมี metric ย้อนหลังมาคำนวณ — เชื่อมกับ Phase 2.2 Observability)?

---

## 3.6 Alert Triage / Ownership

> **Depends หนักที่สุดในเอกสารนี้**: ต้องมี alert เกิดขึ้นจริงก่อนถึงจะมีอะไรให้ triage — Phase 2.2 (Prometheus alert) ยังไม่ได้ทำเลย แนะนำทำเรื่องนี้**หลังสุด** (ดูหัวข้อลำดับท้ายเอกสาร)

### Data Model

```go
// internal/alert/alert.go (package ใหม่)
type Status string
const (
    StatusNew          Status = "new"
    StatusAcknowledged Status = "acknowledged"
    StatusResolved     Status = "resolved"
)

type Alert struct {
    ID             string     `json:"id"`
    Source         string     `json:"source"`   // "prometheus" หรือ "bop-sla-monitor" (ดู FEATURES.md #4)
    Message        string     `json:"message"`
    Severity       string     `json:"severity"` // "warning"/"critical"
    Status         Status     `json:"status"`
    AssignedTo     string     `json:"assigned_to,omitempty"`
    CreatedAt      time.Time  `json:"created_at"`
    AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
    ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
}
```

### Alert Ingestion — รับจาก 2 แหล่งผ่าน endpoint เดียว

```
POST /alerts/webhook    รับ payload จาก Prometheus Alertmanager (webhook receiver มาตรฐาน) — decode ตาม Alertmanager webhook schema แล้วแปลงเป็น Alert ของ BOP
```

BOP's own SLA monitor (FEATURES.md #4) เรียก `alertRepo.Create` ตรงๆ ในโค้ด (ไม่ต้องผ่าน HTTP เพราะอยู่ process เดียวกัน) — ทำให้ `Repository` interface เป็นจุดรวมเดียวที่ทั้งสองแหล่งเขียนเข้ามา ไม่ต้อง duplicate logic

### API

```
GET  /alerts                          list (filter ?status=new ได้)
POST /alerts/{id}/acknowledge         {"assigned_to": "..."} — เปลี่ยน new→acknowledged
POST /alerts/{id}/resolve             เปลี่ยน acknowledged→resolved
```

### Open Questions

1. Alert ที่ status=new ค้างนานเกิน N ชั่วโมงควร escalate (แจ้งซ้ำ/แจ้งคนอื่น) ไหม — ต้องมี background job แยกเช็ค (Temporal Schedule ธรรมดาก็ทำได้ ไม่ต้องเทคโนโลยีใหม่)
2. Deduplicate alert ซ้ำ (bot เดิม fail ซ้ำๆ ยิง alert รัวๆ) — ควร group เป็น alert เดียวไหม หรือปล่อยให้ Prometheus Alertmanager จัดการ deduplication ให้แทน (Alertmanager ทำเรื่องนี้ได้ดีอยู่แล้วเป็นมาตรฐาน — แนะนำพึ่งมันแทนเขียนเอง)

---

## 3.7 Job Definition Versioning + Environment Promotion

### Concept

Control-M แยก environment dev/test/prod และ promote job definition ข้าม environment แบบมีการควบคุม — เทียบเท่าแนวคิด GitOps ที่ใช้กับ K8s manifest ([books/16-cicd/03-gitops.md](../../books/16-cicd/03-gitops.md)) แค่เปลี่ยนจาก "Git เป็น source of truth ของ K8s manifest" เป็น "DB เป็น source of truth ของ job definition"

### ส่วนที่ 1 — Versioning (ทำได้อิสระ ไม่ต้องพึ่งอะไรเพิ่ม)

เปลี่ยน `PUT /schedules/{id}` จาก **overwrite ทำลายของเดิม** เป็น **append-only version history**:

```go
// internal/schedule/schedule.go — ขยายจาก Definition ที่มีอยู่แล้ว
type DefinitionVersion struct {
    ScheduleID string          `json:"schedule_id"`
    Version    int             `json:"version"`
    Steps      []workflow.Step `json:"steps"`
    CreatedAt  time.Time       `json:"created_at"`
    CreatedBy  string          `json:"created_by"` // จาก JWT session เหมือน approval gate
}
```

```
GET /schedules/{id}/versions          list ประวัติทั้งหมด
GET /schedules/{id}/versions/{n}      ดู version เก่าเจาะจง (ใช้ diff กับ version ปัจจุบันได้)
```

`handleUpdateSchedule` เดิม (schedule_handler.go) แก้จาก `s.scheduleRepo.Update` เป็น "สร้าง version ใหม่ + ให้ schedule ชี้ไปที่ version ล่าสุด" — Temporal SDK เองยัง full-replace ตามเดิม (Temporal ไม่มีแนวคิด version ของ Schedule Action) ส่วนที่ versioned คือ **bookkeeping ฝั่ง BOP เท่านั้น** (`internal/schedule` เดิมก็แยกเก็บเองอยู่แล้วด้วยเหตุผลเดียวกัน — ดู comment ในไฟล์นั้น)

### ส่วนที่ 2 — Environment Promotion (ใหญ่กว่ามาก, ต้องมี multi-environment deployment จริงก่อน)

**สมมติฐานที่ต้องมีก่อน (ยังไม่มีตอนนี้)**: BOP ต้อง deploy แยก instance อย่างน้อย 2 ชุด (staging + production) จริงๆ — ตอนนี้มีแค่ local dev เดียว ไม่มี multi-environment เลย เอกสารนี้จึงแค่ **ร่างแนวทาง** ไม่ใช่ design ที่ implement ได้ทันที:

- `POST /schedules/{id}/promote` (เรียกที่ staging BOP) — อ่าน version ล่าสุดจาก staging แล้วยิง HTTP request ไปหา production BOP's `PUT /schedules/{id}` (ต้อง auth ข้าม instance — service-to-service token, ไม่ใช่ user JWT)
- หรือทำผ่าน approval flow ก่อน (เชื่อมกับ 3.4 Approval Gate: "promote" เป็น step ที่ `require_approval: true` ก่อน apply จริงเข้า production)

### Open Questions

1. Version history เก็บกี่ version ย้อนหลัง (unlimited จะโตไม่จำกัด ต้องมี retention policy ไหม)?
2. Promotion ข้าม environment ควรทำผ่าน BOP เองยิง HTTP หา BOP อีก instance หรือควรเป็นงานของ CI/CD pipeline ภายนอก (เช่น GitHub Actions apply ผ่าน BOP API) แทน — แนวทางหลังตรงกับ GitOps philosophy มากกว่า (Git repo เป็น source of truth ไม่ใช่ BOP instance หนึ่งสั่ง BOP instance อื่นตรงๆ)

---

## ตารางสรุป Dependency ข้ามฟีเจอร์

| ฟีเจอร์ | ต้องมีก่อน (จาก Phase 3 อื่น) | ต้องมีก่อน (จาก Phase ก่อนหน้า) |
|---|---|---|
| 3.3 Flow Control | ไม่มี | Schedule CRUD (มีแล้ว) |
| 3.1 Condition-Based Triggering | ไม่มี | Persistence (Postgres แนะนำ, ไม่บังคับ) |
| 3.4 Approval Gate | ไม่มี (MVP ไม่ต้องมี RBAC) | RBAC (Phase 2.3) — สำหรับ "ใครอนุมัติได้" ที่มีความหมายจริง |
| 3.7.1 Versioning | ไม่มี | ไม่มี |
| 3.2 Resource Pools | ไม่มี (MVP single-worker) | Redis (Phase 2.4) — สำหรับ multi-replica ที่ถูกต้อง |
| 3.5 Gantt Visualization | ไม่มีจริงจัง (ใช้ข้อมูลจาก Schedule/WorkflowRun ที่มีอยู่) | Phase 1 Dashboard (มีแล้ว) |
| 3.6 Alert Triage | ไม่มี | **Phase 2.2 Observability/Alert (ยังไม่ได้ทำเลย — blocker จริง)** |
| 3.7.2 Environment Promotion | 3.7.1 Versioning, (แนะนำ) 3.4 Approval Gate | Multi-environment deployment จริง (ยังไม่มี) |

## ลำดับที่แนะนำให้ทำ

เรียงตาม **effort ต่ำ + ไม่ติด blocker** ก่อน แล้วค่อยไปตัวที่ dependency เยอะ/effort สูง — **ต่างจากลำดับ 3.1→3.7 ของ roadmap เดิมโดยตั้งใจ**:

1. **3.3 Flow Control** — risk ต่ำสุด, ไม่แตะ Temporal logic เลย, ใช้ Schedule CRUD ที่มีอยู่ 100%
2. **3.7 (ส่วน Versioning เท่านั้น)** — ไม่มี dependency ใหม่ ได้ audit trail ที่มีประโยชน์แม้ยังไม่ทำ promotion
3. **3.4 Approval Gate** — คุ้มค่าที่สุดต่อสัมภาษณ์ ทำได้แม้ยังไม่มี RBAC (แค่บอกข้อจำกัดตรงๆ ตอน demo)
4. **3.1 Condition-Based Triggering** — ทำ MVP แบบ polling ได้โดยไม่ต้องรอ Redis/Postgres จริง (in-memory condition store ไปก่อนได้ เหมือน pattern เดิมของทั้งโปรเจกต์)
5. **3.2 Resource Pools** — ทำ MVP single-worker ได้เลย แต่ **ต้องบอกไว้ชัดเจนตอน demo ว่ายังไม่ถูกต้องถ้า scale หลาย replica** จนกว่าจะมี Redis
6. **3.5 Gantt Visualization** — รอจังหวะที่อยากโฟกัส frontend/dataviz โดยเฉพาะ (ไม่ผูกกับตัวอื่นเลย ทำเมื่อไหร่ก็ได้)
7. **3.6 Alert Triage** — ทำหลังสุด เพราะ **ต้องมี Phase 2.2 Observability/Alert เกิดขึ้นจริงก่อน** ไม่งั้นไม่มี alert ให้ triage
8. **3.7 (ส่วน Environment Promotion)** — ทำหลังสุดจริงๆ เพราะต้องมี multi-environment deployment จริงก่อน (ยังไม่มีเลยตอนนี้)

## Reference

- [FULLSTACK-INFRA-ROADMAP.md](../../FULLSTACK-INFRA-ROADMAP.md) Phase 3 (checklist ต้นฉบับที่เอกสารนี้ขยายรายละเอียด), Phase 2.2 (Observability, blocker ของ 3.6), Phase 2.3 (Auth/RBAC, ทำให้ 3.4 มีความหมายเต็มที่), Phase 2.4 (Redis, blocker ของ 3.2 ตอน scale)
- [FEATURES.md](FEATURES.md) — สรุป feature ปัจจุบันของ BOP + คำแนะนำที่ควรทำก่อน Phase 3 ทั้งหมด (PostgreSQL persistence เป็นอันดับ 1)
- [projects/bop-dashboard/PHASE3-DASHBOARD-TODO.md](../bop-dashboard/PHASE3-DASHBOARD-TODO.md) — เอกสารคู่กันฝั่ง frontend ที่ทำให้ทุกฟีเจอร์ในนี้ใช้งานได้จริงผ่าน UI ไม่ใช่แค่ API
- [EXTERNAL-JOB-BOT-TODO.md](EXTERNAL-JOB-BOT-TODO.md), [REMOTE-JOB-WORKER-TODO.md](REMOTE-JOB-WORKER-TODO.md), [K8S-JOB-HYBRID-EXECUTION-TODO.md](K8S-JOB-HYBRID-EXECUTION-TODO.md) — ตัวอย่าง decision doc ที่ implement จริงแล้ว ใช้เป็น template เดียวกันตอนหยิบฟีเจอร์ในเอกสารนี้ไปทำจริง (ถาม Open Questions ให้ครบก่อนเริ่มโค้ด)
- Temporal pattern อ้างอิง: `workflow.Selector`, `workflow.GetSignalChannel`, `workflow.NewTimer` (ใช้หนักสุดใน 3.4) — official Temporal docs หัวข้อ "Signals" และ "Selector"
