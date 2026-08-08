# TODO: K8s Job Hybrid Execution (Temporal Activity สั่งสร้าง Kubernetes Job ต่อ step)

> **สถานะ**: **โค้ดหลักทำเสร็จแล้ว** (`Step.K8sJob`/`K8sJobSpec` ใน `internal/workflow/workflow.go`, dispatch logic ใน `Execute()`, `K8sJobActivities.RunKubernetesJob` ใน `internal/workflow/k8sjob.go`, register แบบ best-effort ใน `cmd/bop-worker/main.go`, unit test ด้วย `k8s.io/client-go/kubernetes/fake` ที่ `internal/workflow/k8sjob_test.go`) — Open Questions ทั้ง 6 ข้อ**ยังไม่ได้ถามเจ้าของ repo จริง** ตอน implement รอบนี้ (ทำต่อจากคำสั่ง "implement ต่อได้เลย" โดยใช้ค่า default/แนวทางเดียวกับ `externaljob`/`remote-job-worker` แทนการถามใหม่ทุกข้อ — ดูหมายเหตุกำกับแต่ละข้อด้านล่างว่าข้อไหน "ตัดสินใจแทนแล้ว" กับข้อไหนยังต้องถามจริงก่อนใช้งาน production) **ยังไม่เคยทดสอบกับ K8s cluster จริงเลย** — รอ **Phase 2.1 (containerize BOP เข้า K8s)** เสร็จก่อน (ไม่มี cluster ให้ทดสอบตอนนี้)

## บริบท

BOP มีวิธีรัน job ภายนอกอยู่แล้ว 2 แบบ:

1. **`internal/bots/externaljob`** (local exec) — shell out เรียก binary ในเครื่อง/container เดียวกับ BOP เอง ผ่าน `os/exec`
2. **Remote Temporal task queue** ([REMOTE-JOB-WORKER-TODO.md](REMOTE-JOB-WORKER-TODO.md), reference implementation ที่ [projects/remote-job-worker/](../remote-job-worker/README.md)) — worker process แยกต่างหาก (คนละ repo/deploy) รันค้างไว้ตลอด (long-lived) poll Temporal task queue ของตัวเอง ต้อง embed Temporal Go SDK เอง

เอกสารนี้คือ**ทางเลือกที่สาม**: แทนที่จะมี worker รันค้างไว้ (poll queue ตลอด) ให้ **Temporal Activity ของ BOP เองเป็นคนสั่งสร้าง Kubernetes Job (ephemeral container) ต่อการรันหนึ่งครั้ง** แล้วรอดูผล — เป็น pattern ที่ใช้จริงใน production (Airflow `KubernetesPodOperator`, Argo Workflows, AWS Batch/ECS Task ก็หลักการเดียวกัน) ข้อดีที่ทางเลือกอื่นให้ไม่ได้:

- **Decouple ภาษา/runtime ได้เด็ดขาดที่สุด** — container image เป็นอะไรก็ได้ ไม่ต้อง embed Temporal SDK เลยด้วยซ้ำ (ต่างจาก remote-job-worker ที่ยังต้องมี Temporal SDK อยู่ในกระบวนการ)
- **ไม่มี worker ว่างเปล่ากินทรัพยากรตลอดเวลา** — scale-to-zero จริง จ่ายแค่ตอนมีงานรันจริงเท่านั้น (remote-job-worker ต้องรันค้างไว้ 24/7 แม้ไม่มีงานเข้าเลย)

แลกกับ **cold-start latency** (ต้อง schedule pod + pull image ทุกครั้ง) และต้องพึ่ง K8s cluster จริง (operationally หนักกว่า)

## ตารางเทียบ 3 ทางเลือก (ใช้ตอนไหน)

| ทางเลือก | ภาษา/runtime | ต้อง embed SDK ไหม | Idle cost | Latency | เหมาะกับ |
|---|---|---|---|---|---|
| local `bot.Bot` | Go เท่านั้น | - | ไม่มี | ต่ำสุด | logic ง่าย ผูกกับ BOP อยู่แล้ว |
| `externaljob` (local exec) | อะไรก็ได้ (แค่ compile เป็น binary) | ไม่ต้อง | ไม่มี | ต่ำ | binary อยู่เครื่อง/container เดียวกับ BOP อยู่แล้ว |
| remote-job-worker (Temporal task queue) | อะไรก็ได้ที่มี Temporal SDK | ต้อง | สูง (รันค้าง 24/7) | ต่ำมาก (worker warm อยู่แล้ว) | job ที่มาถี่ๆ ต่อเนื่อง คุ้มเปิด worker ทิ้งไว้ |
| **K8s Job hybrid (เอกสารนี้)** | อะไรก็ได้ (แค่มี container image) | ไม่ต้อง | ไม่มี (scale-to-zero) | สูงกว่า (pod scheduling + image pull) | job ที่มาไม่ถี่/ไม่สม่ำเสมอ, อยาก isolate resource ต่อ job ชัดเจน (CPU/memory limit ต่อ job) |

## ออกแบบสถาปัตยกรรม (ร่าง — ยังไม่ได้ทำ)

### 1. Field ใหม่ใน `workflow.Step`

ตั้งใจ **ไม่ใช้ `Step.TaskQueue` ซ้ำ** เพราะความหมายต่างกันจริง (TaskQueue = "ส่ง activity เดิมไปให้ worker อื่นรัน" ส่วนอันนี้คือ "รัน activity คนละชนิดเลย ที่สร้าง K8s Job") — เพิ่ม field ใหม่แยกเพื่อไม่ให้ความหมายทับซ้อนกันจนอ่านยาก:

```go
type Step struct {
    BotName        string            `json:"bot_name"`
    Config         map[string]string `json:"config"`
    TimeoutSeconds int               `json:"timeout_seconds,omitempty"`
    TaskQueue      string            `json:"task_queue,omitempty"`
    K8sJob         *K8sJobSpec       `json:"k8s_job,omitempty"` // ใหม่ — ไม่ nil = รันผ่าน K8s Job แทน local bot.Bot/remote task queue
}

type K8sJobSpec struct {
    Image     string            `json:"image"`               // container image ที่จะรัน (required)
    Command   []string          `json:"command,omitempty"`   // override entrypoint (optional)
    Args      []string          `json:"args,omitempty"`      // argument ส่งให้ container (เช่น jobcode flag)
    Env       map[string]string `json:"env,omitempty"`       // environment variable เพิ่ม
    Namespace string            `json:"namespace,omitempty"` // K8s namespace (default: namespace เฉพาะของ job, ดู Open Question ข้อ 3)
}
```

### 2. Activity ใหม่ (คนละตัวกับ `Activities.RunStep`)

`Execute()` (temporal.go) ต้องแยกว่า step นี้มี `K8sJob` หรือไม่ — ถ้ามี เรียก activity คนละชื่อ เช่น `ExecuteActivity(stepCtx, "RunKubernetesJob", step.K8sJob)` แทน `"RunStep"` (คนละ activity function, คนละ registry — ไม่ผสมกับ `bot.Bot` registry เดิมเลย)

`Activities.RunKubernetesJob(ctx, spec K8sJobSpec) (string, error)` (ชื่อคร่าวๆ, ตัดสินใจจริงตอน implement) ทำงานคร่าวๆ:

1. สร้าง `batchv1.Job` object ผ่าน `client-go` — ตั้ง `BackoffLimit: 0` เสมอ (**สำคัญ**: ต้องให้ Temporal เป็นเจ้าของ retry decision แต่ผู้เดียว ผ่าน `RetryPolicy`/`NonRetryableErrorTypes` เดิมที่มีอยู่แล้ว — ถ้าปล่อยให้ K8s Job retry เองด้วย (`BackoffLimit > 0`) จะเกิด retry ซ้อนกัน 2 ชั้นที่ debug ยากมาก) และตั้ง `TTLSecondsAfterFinished` (auto-cleanup Job object หลังจบ ไม่ให้ค้างสะสมใน cluster)
2. `Create` Job แล้ว watch จนจบ (`Status.Succeeded > 0` / `Status.Failed > 0`) — ระหว่างรอ เรียก `activity.RecordHeartbeat` เป็นระยะ (pattern เดียวกับ `LONG_TASK` ใน `projects/remote-job-worker/internal/jobs/jobs.go`) เพื่อบอก Temporal ว่า activity นี้ยังทำงานอยู่จริง ไม่ได้ค้าง
3. ดึง log ของ Pod ผ่าน `clientset.CoreV1().Pods(ns).GetLogs(..., &corev1.PodLogOptions{Follow: true})` แล้ว stream เข้า `log()`/Logger ของ BOP แบบเดียวกับที่ `externaljob` stream stdout/stderr ผ่าน `bufio.Scanner` — **สำคัญ**: ต้อง stream ระหว่างรัน ไม่ใช่ดึงหลัง Job จบแล้วเท่านั้น เพราะถ้า `TTLSecondsAfterFinished` เก็บกวาด Job/Pod ไปแล้วก่อนมีคนมาดู log จะหายถาวร (เชื่อมกับ Phase 2.2 Observability ที่จะย้าย log ไป OpenSearch จริงจังทีหลัง)
4. ถ้า `ctx` ถูกยกเลิก (Temporal timeout/cancel) ต้อง `Delete` Job (พร้อม `PropagationPolicy: Foreground` ให้ Pod ลูกถูกลบตามไปด้วย) — หลักการเดียวกับที่ `externaljob` ต้อง kill ทั้ง process group ตอน cancel ไม่ใช่ปล่อย orphan ทิ้งไว้

### 3. Exit code / error semantics

ใช้ convention เดียวกับ `externaljob`: exit code `0` = สำเร็จ, อื่นๆ = error ธรรมดา (retryable) — jobcode/config ผิดที่รู้จากภายนอก K8s Job ไม่ได้ (K8s ไม่มีแนวคิด "config error" ในตัวเอง) ต้องคิดเพิ่มตอน implement จริงว่าจะแยก non-retryable ได้ยังไง (ดู Open Question ข้อ 4)

## Open Questions

> อัปเดตหลัง implement: ข้อ 3/4 ตัดสินใจแทนแบบ YAGNI โดยอิง precedent จาก `externaljob`/`remote-job-worker` ที่ทำไปแล้วในรอบก่อนหน้า (ไม่ได้ถามเจ้าของ repo ใหม่) — ข้อ 1/2/5/6 **ยังไม่ได้ตัดสินใจจริง** เพราะเป็นเรื่อง deployment/ops ที่ต้องมี K8s cluster จริงถึงจะมีคำตอบที่ใช้งานได้ (image registry จริง, RBAC manifest จริง, resource quota จริงของ cluster) — ต้องถามให้ครบก่อนใช้งาน production จริง ไม่ใช่แค่ก่อนเริ่มโค้ด (โค้ดหลักทำเสร็จแล้ว)

1. **[ยังไม่ตัดสินใจ] Container image มาจากไหน** — registry เดียวกับ image ของ BOP เอง (Phase 2.1) ไหม? เป็น 1 image ทั่วไปที่มี jobcode dispatch ข้างในเหมือน `externaljob`/`remote-job-worker` (generic, ไม่ hard-code ต่อ jobcode) หรือ 1 image ต่อ 1 ประเภทงาน? — โค้ดรองรับทั้งสองแบบอยู่แล้ว (`K8sJobSpec.Image` เป็น string อิสระต่อ step) แค่ยังไม่รู้ image จริงที่จะใช้
2. **[ยังไม่ตัดสินใจ] RBAC** — ต้องเขียน Role/RoleBinding manifest จริงตอน deploy (ยังไม่มีในเอกสารนี้/โค้ด — เป็นงานของ Phase 2.1) ให้ ServiceAccount ของ `cmd/bop-worker` มีสิทธิ์แค่ create/get/list/delete `Job` และ get `Pod`/`Pod log` ใน namespace เดียว (ไม่ cluster-wide) เชื่อมกับ Phase 2.7
3. **[ตัดสินใจแทนแล้ว] Namespace แยกจาก BOP เอง** — ใช้ `k8sJobDefaultNamespace = "bop-jobs"` (constant ใน `cmd/bop-worker/main.go`) เป็นค่า default ถ้า `K8sJobSpec.Namespace` ไม่ได้ระบุมา — namespace นี้ยังไม่ถูกสร้างจริงใน cluster ไหนเลย (ต้องสร้างตอน deploy จริง เชื่อมกับ Open Question ข้อ 2)
4. **[ตัดสินใจแทนแล้ว] วิธีแยก non-retryable error** — **ไม่แยก** — ทุก error จาก Job ที่ล้มเหลว (`Status.Failed > 0`) ถือเป็น error ธรรมดา/retryable ทั้งหมด เหมือน convention ที่ตัดสินใจไว้กับ `externaljob` ตอน exit code ไม่ใช่ 0 (K8s ไม่มีทางบอกได้จากภายนอกว่า container fail เพราะ config ผิดหรือปัญหาชั่วคราว) `ConfigError` (non-retryable) สงวนไว้เฉพาะกรณีที่ BOP รู้แน่ชัดเอง คือ `K8sJobSpec.Image` ว่างเปล่า — ถ้าอยากแยกละเอียดกว่านี้ (เช่น convention `exit 78` ตาม `sysexits.h`) ต้องคุยกับเจ้าของ job แต่ละตัวเพิ่มทีหลัง
5. **[ยังไม่ตัดสินใจ] Resource request/limit ต่อ Job** — โค้ดปัจจุบัน**ไม่ได้ตั้ง** `resources.requests`/`resources.limits` ให้ container เลย (ปล่อยว่าง) — ต้องตัดสินใจว่าจะบังคับผ่าน `K8sJobSpec` field ใหม่ หรือพึ่ง `LimitRange` default ของ namespace `bop-jobs` แทน ก่อนใช้งานจริงกับ cluster ที่มี workload อื่นแชร์กันอยู่ (ไม่งั้น job เดียวกิน resource node จนตัวอื่นล้มได้)
6. **[ยังไม่ตัดสินใจ] Timeout ระหว่าง `Step.TimeoutSeconds` กับเวลา schedule+pull image** — โค้ดปัจจุบันใช้ `StartToCloseTimeout` เดียวครอบคลุมทั้งกระบวนการ (schedule pod + pull image + รันจริง) ไม่ได้แยกเป็น 2 เฟส — ถ้า image ใหญ่/cluster busy อาจ timeout ก่อน Job จะเริ่มรันจริงด้วยซ้ำ ยังไม่ได้แก้เพราะยังไม่มี cluster จริงให้วัด latency ที่แท้จริง

## TODO Checklist

### 1. ก่อนเริ่มโค้ด
- [x] ~~Phase 2.1 (containerize BOP เข้า K8s) ต้องเสร็จก่อน~~ — โค้ดเขียนได้โดยไม่ต้องรอ (unit test ผ่าน `k8s.io/client-go/kubernetes/fake` แทน) แต่ **ยังต้องรอ Phase 2.1 ก่อนใช้งานจริง** (`cmd/bop-worker` ยัง `rest.InClusterConfig()` ไม่ผ่านเพราะยังไม่ได้รันอยู่ในเครื่อง/container ใน K8s เลย — ดู best-effort registration ใน `cmd/bop-worker/main.go`)
- [x] ~~ถามเจ้าของ repo ให้ครบ Open Questions ทั้ง 6 ข้อ~~ — ถามแค่บางส่วน ดูหมายเหตุที่หัวข้อ Open Questions ด้านบนว่าข้อไหนตัดสินใจแทนแล้ว

### 2. Implement
- [x] เพิ่ม `Step.K8sJob *K8sJobSpec` ใน `internal/workflow/workflow.go`
- [x] เพิ่ม dependency `k8s.io/client-go` เข้า `go.mod` ของ BOP (`go get k8s.io/client-go@latest` + `go mod tidy` — ดึง `k8s.io/api`/`k8s.io/apimachinery` มาด้วยอัตโนมัติ)
- [x] เขียน `K8sJobActivities.RunKubernetesJob` ที่ `internal/workflow/k8sjob.go`
- [x] แก้ `Execute()` ใน `temporal.go` ให้เลือก activity name ตามว่า `step.K8sJob` เป็น nil หรือไม่
- [x] Register `RunKubernetesJob` ใน `cmd/bop-worker/main.go` — แบบ **best-effort** (`rest.InClusterConfig()` ล้มเหลวตอน local dev เป็นเรื่องปกติที่คาดไว้แล้ว แค่ log คำเตือนแล้วข้าม ไม่ `log.Fatalf` ปิด process ทั้งตัว)

### 3. Testing
- [x] unit test ที่ไม่ต้องพึ่ง K8s cluster จริง — `internal/workflow/k8sjob_test.go` ใช้ `k8s.io/client-go/kubernetes/fake` (จำลอง Job controller เองด้วย goroutine เพราะ fake clientset เป็นแค่ CRUD store ไม่มี controller logic จริง)
- [ ] end-to-end จริงต้องมี K8s cluster ทดสอบ (local: kind/minikube พอ ไม่ต้องรอ production cluster) — **ยังไม่ได้ทำ**

### 4. Docs
- [x] อัปเดต README.md เพิ่มหัวข้อที่ 3 คู่กับ "Bot: external-job" และ "รัน step ผ่าน remote Temporal worker"
- [x] อัปเดตตารางเทียบ 3 ทางเลือกในเอกสารนี้ (ไม่มีอะไรเปลี่ยนจากที่ร่างไว้ — trade-off จริงตรงกับที่คาดไว้)

## Reference

- [EXTERNAL-JOB-BOT-TODO.md](EXTERNAL-JOB-BOT-TODO.md), [REMOTE-JOB-WORKER-TODO.md](REMOTE-JOB-WORKER-TODO.md) — สองทางเลือกที่ทำไปแล้ว เทียบ trade-off กันได้ที่ตารางด้านบน
- [FULLSTACK-INFRA-ROADMAP.md](../../FULLSTACK-INFRA-ROADMAP.md) Phase 2.1 (containerize, prerequisite ของเอกสารนี้), 2.2 (Observability — เกี่ยวกับเรื่อง log retention หลัง Job ถูกลบ), 2.7 (Security — RBAC scoping), 3.2 (Resource Pools — resource limit ต่อ job)
- [books/08-kubernetes/](../../books/08-kubernetes/README.md) — ยังไม่มีบทเรื่อง `batch/v1 Job`/`CronJob` โดยตรง (มีแค่ Pod/Deployment/Service/Ingress กับ HPA/etcd/Scheduler) — อาจต้องเสริมเนื้อหานี้ก่อน/ระหว่าง implement
- Production pattern อ้างอิง: Airflow `KubernetesPodOperator`, Argo Workflows, AWS Batch/ECS Task — หลักการเดียวกันคือ orchestrator สั่งสร้าง ephemeral container ต่องาน แทนที่จะมี worker รันค้างไว้
