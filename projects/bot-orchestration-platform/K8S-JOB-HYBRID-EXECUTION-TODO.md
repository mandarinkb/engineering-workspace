# TODO: K8s Job Hybrid Execution (Temporal Activity สั่งสร้าง Kubernetes Job ต่อ step)

> **สถานะ**: ยัง**ไม่เริ่ม implement** — เป็นแค่ design doc เขียนไว้ล่วงหน้าตอนคุยกันเรื่อง production pattern เทียบกับ [REMOTE-JOB-WORKER-TODO.md](REMOTE-JOB-WORKER-TODO.md) — รอ **Phase 2.1 (containerize BOP เข้า K8s)** เสร็จก่อนถึงจะเริ่มทำได้จริง (ต้องมี K8s cluster + BOP รันอยู่ใน cluster นั้นแล้วถึงจะมี ServiceAccount/RBAC ให้ BOP สร้าง Job ของ cluster ตัวเองได้) — session ที่หยิบเอกสารนี้ไปทำต่อ **ต้องถาม Open Questions ให้ครบก่อนเริ่มโค้ดจริง** เหมือนกับที่ [EXTERNAL-JOB-BOT-TODO.md](EXTERNAL-JOB-BOT-TODO.md) ทำไว้เป็นแบบอย่าง

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

## Open Questions (ต้องตอบให้ครบก่อนเริ่มโค้ดจริง — ยังไม่ได้ถามตอนเขียนเอกสารนี้)

1. **Container image มาจากไหน** — registry เดียวกับ image ของ BOP เอง (Phase 2.1) ไหม? เป็น 1 image ทั่วไปที่มี jobcode dispatch ข้างในเหมือน `externaljob`/`remote-job-worker` (generic, ไม่ hard-code ต่อ jobcode) หรือ 1 image ต่อ 1 ประเภทงาน?
2. **RBAC** — BOP เอง (ที่รันบน K8s แล้วตาม Phase 2.1) ต้องมี ServiceAccount ที่มีสิทธิ์สร้าง/ดู/ลบ `Job` และอ่าน log ของ `Pod` — ต้อง scope ให้แคบที่สุด (namespace เฉพาะสำหรับ job พวกนี้ ไม่ใช่ cluster-wide) เชื่อมกับ Phase 2.7 (Security) ที่วางแผนไว้แล้ว
3. **Namespace แยกจาก BOP เอง** — แนะนำให้รัน Job ใน namespace แยกต่างหาก (เช่น `bop-jobs`) ไม่ปนกับ namespace ที่ BOP/dashboard/Temporal worker รันอยู่ (blast radius, resource quota แยกกันชัดเจน)
4. **วิธีแยก non-retryable error โดยไม่มี concept "config error" จาก K8s ตรงๆ** — อาจต้องกำหนด convention เอง เช่น exit code เฉพาะ (เช่น `exit 78` ตาม `sysexits.h` convention ที่มีอยู่แล้วสำหรับ "config error") ให้ job เขียนตามได้ถ้าต้องการ ต้องคุยกับเจ้าของ job แต่ละตัวเหมือนที่ `EXTERNAL-JOB-BOT-TODO.md` Open Question ข้อ 4 เคยทำ
5. **Resource request/limit ต่อ Job** — บังคับให้ทุก `K8sJobSpec` ต้องระบุ CPU/memory request+limit ไหม (กัน job เดียวกิน resource node จนตัวอื่นล้ม) หรือใช้ `LimitRange` default ของ namespace แทน — เชื่อมกับ Phase 3.2 (Resource Pools) ที่วางแผนไว้ในอนาคต
6. **Timeout ระหว่าง `Step.TimeoutSeconds` (Temporal ActivityOptions) กับเวลาที่ K8s ใช้ schedule+pull image** — ถ้า image ใหญ่/cluster busy การ pull image เองอาจกินเวลานานจนชน `StartToCloseTimeout` ก่อน Job จะเริ่มรันจริงด้วยซ้ำ ต้องคิดว่าจะแยก "เวลารอ pod ready" ออกจาก "เวลารันงานจริง" ไหม

## TODO Checklist

### 1. ก่อนเริ่มโค้ด
- [ ] Phase 2.1 (containerize BOP เข้า K8s) ต้องเสร็จก่อน — ไม่มี cluster ให้ BOP คุยด้วยเลยตอนนี้
- [ ] ถามเจ้าของ repo ให้ครบ Open Questions ทั้ง 6 ข้อด้านบน

### 2. Implement
- [ ] เพิ่ม `Step.K8sJob *K8sJobSpec` ใน `internal/workflow/workflow.go`
- [ ] เพิ่ม dependency `k8s.io/client-go` เข้า `go.mod` ของ BOP (เป็น external dependency ตัวใหม่ — เหมือนตอนเพิ่ม `go.temporal.io/sdk` ใน Phase 2.5 ที่เป็นข้อยกเว้นตั้งใจ ดู `TEMPORAL-MIGRATION-TODO.md`)
- [ ] เขียน `Activities.RunKubernetesJob` ใน `internal/workflow/` (ไฟล์ใหม่ เช่น `k8sjob.go`) ตามร่างข้อ 2 ด้านบน
- [ ] แก้ `Execute()` ใน `temporal.go` ให้เลือก activity name ตามว่า `step.K8sJob` เป็น nil หรือไม่
- [ ] Register `RunKubernetesJob` ใน `cmd/bop-worker/main.go`

### 3. Testing
- [ ] unit test ที่ไม่ต้องพึ่ง K8s cluster จริง (mock `client-go` ผ่าน `k8s.io/client-go/kubernetes/fake`)
- [ ] end-to-end จริงต้องมี K8s cluster ทดสอบ (local: kind/minikube พอ ไม่ต้องรอ production cluster)

### 4. Docs
- [ ] อัปเดต README.md เพิ่มหัวข้อที่ 3 คู่กับ "Bot: external-job" และ "รัน step ผ่าน remote Temporal worker"
- [ ] อัปเดตตารางเทียบ 3 ทางเลือกในเอกสารนี้ถ้า trade-off จริงต่างจากที่ร่างไว้

## Reference

- [EXTERNAL-JOB-BOT-TODO.md](EXTERNAL-JOB-BOT-TODO.md), [REMOTE-JOB-WORKER-TODO.md](REMOTE-JOB-WORKER-TODO.md) — สองทางเลือกที่ทำไปแล้ว เทียบ trade-off กันได้ที่ตารางด้านบน
- [FULLSTACK-INFRA-ROADMAP.md](../../FULLSTACK-INFRA-ROADMAP.md) Phase 2.1 (containerize, prerequisite ของเอกสารนี้), 2.2 (Observability — เกี่ยวกับเรื่อง log retention หลัง Job ถูกลบ), 2.7 (Security — RBAC scoping), 3.2 (Resource Pools — resource limit ต่อ job)
- [books/08-kubernetes/](../../books/08-kubernetes/README.md) — ยังไม่มีบทเรื่อง `batch/v1 Job`/`CronJob` โดยตรง (มีแค่ Pod/Deployment/Service/Ingress กับ HPA/etcd/Scheduler) — อาจต้องเสริมเนื้อหานี้ก่อน/ระหว่าง implement
- Production pattern อ้างอิง: Airflow `KubernetesPodOperator`, Argo Workflows, AWS Batch/ECS Task — หลักการเดียวกันคือ orchestrator สั่งสร้าง ephemeral container ต่องาน แทนที่จะมี worker รันค้างไว้
