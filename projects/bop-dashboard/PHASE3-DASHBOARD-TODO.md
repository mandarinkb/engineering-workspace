# TODO: Dashboard ฝั่ง Enterprise Feature (คู่กับ Backend Phase 3)

> **สถานะ**: **6 ฟีเจอร์ในกลุ่ม "ทำได้เลยตอนนี้" implement ครบแล้ว** (log viewer ยกระดับ, job list ยกระดับ, Pipeline View เฟส A, dispatch mode picker, dark mode, toast + watch-job notification) — `npm run typecheck`/`lint`/`test`/`build` ผ่านหมด (ต้องใช้ Node >=22 ตาม `.nvmrc` — เครื่อง dev เดิมมี Node 19 ต้อง `nvm use 22` ก่อนเสมอ) ดู [FEATURES.md](../bot-orchestration-platform/FEATURES.md) หรือ git log ของ commit ที่ implement งานนี้สำหรับรายละเอียดไฟล์ — กลุ่ม **"รอ backend Phase 3"** (Flow Control view, Approve/Reject button, Condition dashboard, Alert triage inbox, Schedule version/diff view) **ยังไม่เริ่ม** เพราะ backend ฝั่งนั้นยังไม่มี API ให้ต่อเลย (ดู [projects/bot-orchestration-platform/PHASE3-ENTERPRISE-FEATURES-TODO.md](../bot-orchestration-platform/PHASE3-ENTERPRISE-FEATURES-TODO.md))
>
> **ข้อจำกัดที่รู้ตัวแล้วจากตอน implement**: (1) Dark mode ใส่ `dark:` variant ให้แค่ component ใหม่ที่สร้างรอบนี้ + `StatusBadge`/`NavBar` — หน้าเดิม (`job-table`, `bot-form`, `schedule-form` ฯลฯ) ยังไม่ได้ retrofit ครบ เป็น follow-up (2) Pipeline View เฟส A render ได้แค่ step ที่รันไปแล้วจริง + node "กำลังทำงาน" ท้ายสุดถ้ายังไม่จบ — ไม่รู้ total step count ล่วงหน้า (ข้อจำกัดของ backend `StepResult`) (3) Toast-on-status-change ใช้ polling ทุก 3 วิผ่าน `WatchedJobsProvider` ไม่ใช่ SSE (ดูเหตุผลในหัวข้อ "แนวทางที่ตัดสินใจไว้" ของ plan ตอน implement) (4) ยังไม่เคยทดสอบ end-to-end จริงกับ backend running (เหตุผลเดียวกับทุกฟีเจอร์ก่อนหน้านี้ในโปรเจกต์นี้ — ยังไม่มี Temporal server รันจริงตอน implement)

**สิ่งสำคัญที่สุดที่ต้องรู้ก่อนอ่านต่อ**: feature ที่ขอมา 3 ข้อ (workflow relationship, job success/fail, log) **มีพื้นฐานอยู่แล้วใน dashboard ปัจจุบัน** (ดูหัวข้อถัดไป) — เอกสารนี้จึงแบ่งชัดเจนเป็น 2 กลุ่มเสมอในทุกหัวข้อ: **"ทำได้เลยตอนนี้"** (ไม่ต้องรอ backend ใหม่) กับ **"รอ backend Phase 3 ก่อน"** (ต้องมี API ใหม่จากเอกสารคู่กันก่อนถึงจะมี UI ให้ต่อ) — อย่าเริ่ม implement ฝั่งที่ "รอ backend" ก่อน API นั้นพร้อมจริง

## สารบัญ

1. [สถานะปัจจุบันของ dashboard (สิ่งที่มีอยู่แล้ว)](#สถานะปัจจุบันของ-dashboard-สิ่งที่มีอยู่แล้ว)
2. [Feature ที่ขอมา 1: Workflow Relationship View](#feature-ที่ขอมา-1-workflow-relationship-view)
3. [Feature ที่ขอมา 2: Job Success/Fail (ยกระดับของเดิม)](#feature-ที่ขอมา-2-job-successfail-ยกระดับของเดิม)
4. [Feature ที่ขอมา 3: Log Viewer (ยกระดับของเดิม)](#feature-ที่ขอมา-3-log-viewer-ยกระดับของเดิม)
5. [Feature เพิ่มเติมที่แนะนำ](#feature-เพิ่มเติมที่แนะนำ)
6. [ตาราง Dependency กับ Backend](#ตาราง-dependency-กับ-backend)
7. [ลำดับที่แนะนำให้ทำ](#ลำดับที่แนะนำให้ทำ)
8. [Reference](#reference)

---

## สถานะปัจจุบันของ dashboard (สิ่งที่มีอยู่แล้ว)

เช็คจากโค้ดจริงตอนเขียนเอกสารนี้ (`src/app/`, `src/features/`) ไม่ใช่จากความจำ:

| หน้า | Route | Feature module | สถานะ |
|---|---|---|---|
| Bot list + dynamic config form | `/bots` | `features/bots` | มีแล้ว |
| Job history (table + filter) | `/jobs` | `features/jobs` | มีแล้ว — มี `job-table.tsx`, `job-filter.tsx`, `job-status-badge.tsx` |
| **Live job monitor (log แบบ real-time ผ่าน SSE)** | `/jobs/[id]` | `features/jobs` | มีแล้ว — `job-log-viewer.tsx` + `use-job-log-stream.ts` |
| Workflow run list | `/workflow-runs` | `features/workflow-runs` | มีแล้ว |
| **Workflow run detail (steps)** | `/workflow-runs/[id]` | `features/workflow-runs` | มีแล้ว **แต่แสดงเป็น `<ol>` list ธรรมดา** (`step-result-list.tsx`) ไม่ใช่ diagram/graph — นี่คือช่องว่างที่ Feature ขอมา 1 เติมให้ |
| Schedule CRUD + trigger/backfill | `/schedules` | `features/schedules` | มีแล้ว |
| Login | `/login` | `features/auth` | มีแล้ว |

สรุป: **"หน้าแสดงว่า job สำเร็จหรือไม่" และ "หน้าแสดง log" มีอยู่แล้วจริงๆ** — งานของเอกสารนี้ในสองหัวข้อนั้นคือ **ยกระดับ** ไม่ใช่สร้างใหม่ ส่วน **workflow relationship** เป็นของใหม่จริง (ของเดิมมีแค่ list เรียงลำดับ ไม่มี visual ของความสัมพันธ์เลย)

---

## Feature ที่ขอมา 1: Workflow Relationship View

### ข้อจำกัดที่ต้องเข้าใจก่อนออกแบบ: backend ตอนนี้รองรับแค่ "สายตรง" (linear) เท่านั้น

`workflow.Workflow.Steps` (Go) เป็น `[]Step` ธรรมดา — `Execute()` (`internal/workflow/temporal.go`) วน `for i, step := range wf.Steps` **ตามลำดับเป๊ะ** step ถัดไปเริ่มได้ก็ต่อเมื่อ step ก่อนหน้าสำเร็จเท่านั้น **ไม่มี fan-out/fan-in/branching ในโมเดลปัจจุบันเลย** — ดังนั้น "ความสัมพันธ์" ที่มีข้อมูลจริงให้แสดงตอนนี้คือ **pipeline เส้นตรง** (เทียบเท่า GitHub Actions job graph แบบไม่มี `needs` ข้ามสาย, GitLab CI stage ธรรมดา) ไม่ใช่ DAG เต็มรูปแบบ — ความสัมพันธ์แบบ many-to-many จริงๆ (job หลายตัวรอ condition เดียวกัน) ต้องรอ [backend 3.1 Condition-Based Triggering](../bot-orchestration-platform/PHASE3-ENTERPRISE-FEATURES-TODO.md#31-condition-based-job-triggering) ก่อน

**ออกแบบเป็น 2 เฟสตามความจริงข้อนี้ — อย่าข้ามไปทำเฟส B ก่อน A**

### เฟส A — Pipeline View (ทำได้เลยตอนนี้ ไม่ต้องรอ backend)

แทนที่ `step-result-list.tsx` (หรือเพิ่มเป็นอีก view mode คู่กับของเดิม) ด้วย horizontal pipeline: node ต่อ node เรียงแนวนอน (หรือแนวตั้งบนจอเล็ก) เชื่อมด้วยเส้น มี status สีต่าง node ตามสถานะ

```
features/workflow-runs/components/
├── workflow-pipeline-view.tsx     # component ใหม่ — แสดง StepResult[] เป็น node เรียงกัน
├── pipeline-node.tsx              # node เดียว: bot_name + status icon + คลิกแล้วเห็น summary/error
└── step-result-list.tsx           # (ของเดิม) เก็บไว้เป็น fallback/list view ให้สลับดูได้
```

**ตั้งใจไม่ใช้ library ใหม่สำหรับเฟสนี้** (เช่น `react-flow`) — ข้อมูลเป็นเส้นตรงเสมอ สร้างด้วย Tailwind flexbox + border/connector line ธรรมดาพอ ตรงกับหลักการ "zero external dependency เท่าที่จำเป็น" ที่ backend ยึดมาตลอด (เพิ่ม dependency หนักเพื่อ render กราฟที่มีแค่เส้นตรงคือ over-engineering)

Status ต่อ node มาจาก `StepResult` ที่มีอยู่แล้ว (`step_index`, `bot_name`, `summary`, `error`) บวก field ใหม่ที่ backend 3.4 จะเพิ่ม (`pending_approval`) — ออกแบบ `PipelineNode` ให้รับ status enum ที่ derive ได้ตอนนี้แล้ว (`succeeded`/`failed`/`pending`) และเผื่อช่องสำหรับ `pending_approval`/`waiting_condition` ที่จะมาทีหลัง (ดูหัวข้อ 3.4/3.1 ด้านล่าง) — ไม่ต้อง refactor ใหญ่ตอนต่อ backend จริง

### เฟส B — DAG จริง (รอ backend 3.1 ก่อน)

ต่อเมื่อ backend มี condition-based relationship จริง (job หลายตัวรอ condition เดียวกัน ข้าม schedule) ค่อยพิจารณาสลับไปใช้ **`reactflow`** (หรือเทียบเท่า) เพราะตอนนั้นเป็นกราฟทั่วไปจริง (ไม่ใช่เส้นตรง) render เองด้วย CSS จะซับซ้อนเกินคุ้ม — **Open Question**: ตอนนั้นค่อยตัดสินใจ ไม่ต้องเลือก library ล่วงหน้าตอนนี้

---

## Feature ที่ขอมา 2: Job Success/Fail (ยกระดับของเดิม)

### มีอยู่แล้ว
`job-table.tsx` + `job-status-badge.tsx` — ตาราง job พร้อมสถานะ, `job-filter.tsx` — filter พื้นฐาน

### ยกระดับที่ทำได้เลยตอนนี้ (ข้อมูลมีอยู่แล้วใน response ของ `GET /jobs`)

- **Summary stat cards** เหนือตาราง — success rate %, จำนวน job วันนี้, ล้มเหลวล่าสุดกี่ตัว — คำนวณ client-side จาก list ที่ query มาอยู่แล้ว ไม่ต้องมี endpoint ใหม่
- **Filter เพิ่ม**: ตาม bot name (มีข้อมูล `bot_name` อยู่แล้ว), ช่วงวันที่ (`created_at`)
- **Bulk action**: เลือกหลาย job พร้อมกันแล้วสั่ง cancel รวด (ใช้ `DELETE /jobs/{id}` เดิม วนยิงหลายครั้ง)

### ยกระดับที่ต้องรอ backend ก่อน

- **SLA/duration anomaly highlight** (job ที่ใช้เวลานานผิดปกติเทียบค่าเฉลี่ยเดิม ไฮไลต์สีเตือน) — ต้องมี metric ย้อนหลังจาก [backend Phase 2.2 Observability](../bot-orchestration-platform/FEATURES.md) ก่อน (ยังไม่มี)

---

## Feature ที่ขอมา 3: Log Viewer (ยกระดับของเดิม)

### มีอยู่แล้ว
`job-log-viewer.tsx` + `use-job-log-stream.ts` — SSE real-time, backfill ก่อนแล้ว stream ต่อ (ดู decision log ที่ [PHASE1-DASHBOARD-TODO.md](PHASE1-DASHBOARD-TODO.md))

### ยกระดับที่ทำได้เลยตอนนี้ (ข้อมูลมีอยู่แล้วใน `logger.Line`: `level`, `message`, `time`)

- **Filter ตาม level** (info/warn/error) — `logger.Line.Level` มีอยู่แล้วในทุกบรรทัดที่ SSE ส่งมา แค่ยังไม่ได้ใช้ฝั่ง frontend
- **Search/highlight ข้อความใน log** — client-side text search ธรรมดาบน log line ที่ถืออยู่ใน state แล้ว
- **ปุ่ม download log เป็น `.txt`** — join `message` ทุกบรรทัดที่มีอยู่ใน state สร้าง Blob แล้ว trigger download
- **Auto-scroll toggle** — ปุ่ม pin/unpin ไม่ให้ log ใหม่ดันจอเลื่อนตอนกำลังอ่าน log เก่าอยู่

ทั้งหมดนี้ **ไม่ต้องแก้ backend เลยแม้แต่บรรทัดเดียว** เพราะข้อมูลที่ต้องใช้ (`level`, `message`) ส่งมาอยู่แล้วทุกบรรทัด แค่ frontend ยังไม่ได้ใช้ประโยชน์เต็มที่ — **แนะนำทำก่อนสุดในเอกสารทั้งฉบับ** เพราะ effort ต่ำที่สุดเทียบกับ value ที่ได้

---

## Feature เพิ่มเติมที่แนะนำ

เรียงตาม priority — กลุ่มแรกทำได้เลย กลุ่มหลังรอ backend Phase 3:

### ทำได้เลยตอนนี้

1. **Dispatch mode picker ตอนสร้าง job/step** — backend มี 3 ทางเลือกแล้ว (local bot / remote Temporal task queue / K8s Job hybrid ดู [backend FEATURES.md](../bot-orchestration-platform/FEATURES.md) ข้อ 2) แต่ dashboard ยังมีแค่ฟอร์ม local bot อย่างเดียว — เพิ่ม field เสริมในฟอร์มสร้าง step (`task_queue`, `k8s_job`) ให้ผู้ใช้เลือกได้ผ่าน UI แทนที่จะยิง `curl` เอง เป็นช่องว่างที่ชัดเจนที่สุดระหว่าง backend capability กับ frontend ตอนนี้
2. **Dark mode** — enterprise tool ควรมี, Tailwind v4 รองรับผ่าน `prefers-color-scheme`/class strategy อยู่แล้วโดยไม่ต้องเพิ่ม dependency
3. **Global toast/notification** เมื่อ job ที่กำลังดูอยู่เปลี่ยนสถานะ (สำเร็จ/ล้มเหลว) ระหว่างเปิดหน้าอื่นอยู่ — ใช้ SSE connection ที่มีอยู่แล้วเป็นฐาน ไม่ต้องเปิด connection ใหม่

### รอ backend Phase 3 ก่อน (เรียงตามที่ backend design doc แนะนำให้ทำก่อน-หลัง)

4. **Flow Control view** (`/flows`) — รายการ Flow + สถานะ aggregate + ปุ่ม pause/resume/trigger ทั้งกลุ่ม — รอ [backend 3.3](../bot-orchestration-platform/PHASE3-ENTERPRISE-FEATURES-TODO.md#33-flow-control) (backend แนะนำให้ทำก่อนสุดในบรรดา Phase 3 เพราะ risk ต่ำ)
5. **Approve/Reject button บน Pipeline View** (ต่อยอด Feature 1 เฟส A โดยตรง) — node ที่ `pending_approval: true` โชว์ปุ่ม Approve/Reject แทน status icon เฉยๆ ยิง `POST /workflow-runs/{id}/steps/{index}/approve` — รอ [backend 3.4](../bot-orchestration-platform/PHASE3-ENTERPRISE-FEATURES-TODO.md#34-approval-gate-human-in-the-loop)
6. **Condition dashboard** (`/conditions`) — ดู condition ที่ set ไว้ทั้งหมด + ใครรอ condition อะไรอยู่ (debug UI ที่ backend doc เองก็ระบุว่าจำเป็น เพราะ "รอ condition ที่มองไม่เห็นสถานะ" debug ยากมาก) — รอ [backend 3.1](../bot-orchestration-platform/PHASE3-ENTERPRISE-FEATURES-TODO.md#31-condition-based-job-triggering)
7. **Alert triage inbox** (`/alerts`) — list + acknowledge/resolve — รอ [backend 3.6](../bot-orchestration-platform/PHASE3-ENTERPRISE-FEATURES-TODO.md#36-alert-triage--ownership) (backend เองก็บอกว่า blocker หนักสุด ต้องมี Phase 2.2 alert เกิดขึ้นจริงก่อน)
8. **Schedule version history / diff view** — ต่อยอด [backend 3.7](../bot-orchestration-platform/PHASE3-ENTERPRISE-FEATURES-TODO.md#37-job-definition-versioning--environment-promotion) ส่วน versioning

### ไม่แนะนำให้ทำเร็วเกินไป (stretch, ไม่ผูกกับ backend Phase 3 เลย แต่ effort/value ไม่คุ้มทำก่อน)

- Command palette (⌘K quick search) — "ดูล้ำ" แต่ไม่ได้แก้ pain point จริงตอนนี้ (จำนวนหน้า/entity ยังน้อย)
- Gantt/timeline visualization เต็มรูปแบบ — backend เองก็จัดเป็นลำดับท้ายๆ ([3.5](../bot-orchestration-platform/PHASE3-ENTERPRISE-FEATURES-TODO.md#35-gantt--timeline-visualization)) เพราะไม่ผูกกับอะไร ทำเมื่อไหร่ก็ได้ ไม่จำเป็นต้องรีบ

---

## ตาราง Dependency กับ Backend

| Dashboard feature | Backend ที่ต้องมีก่อน | สถานะ backend |
|---|---|---|
| Pipeline View (เฟส A) | ไม่มี (ใช้ `GET /workflow-runs/{id}` เดิม) | มีแล้ว |
| Job/Log ยกระดับทั้งหมดในหัวข้อ 2-3 | ไม่มี | มีแล้ว |
| Dispatch mode picker | ไม่มี (`task_queue`/`k8s_job` มีใน API แล้ว) | มีแล้ว |
| DAG จริง (เฟส B) | 3.1 Condition-Based Triggering | ยังไม่ implement |
| Approve/Reject button | 3.4 Approval Gate | ยังไม่ implement |
| Flow Control view | 3.3 Flow Control | ยังไม่ implement |
| Condition dashboard | 3.1 Condition-Based Triggering | ยังไม่ implement |
| Alert triage inbox | 3.6 Alert Triage (+ Phase 2.2 ที่ backend เองก็ยังไม่ทำ) | ยังไม่ implement (2 ชั้น) |
| Schedule version/diff view | 3.7 Versioning | ยังไม่ implement |

## ลำดับที่แนะนำให้ทำ

1. **Log viewer ยกระดับ** (หัวข้อ 3) — effort ต่ำสุด, ข้อมูลมีครบแล้ว, เห็นผลทันที
2. **Job success/fail ยกระดับที่ทำได้เลย** (หัวข้อ 2) — เช่นกัน effort ต่ำ ข้อมูลมีครบ
3. **Pipeline View เฟส A** (หัวข้อ 1) — งานใหม่ที่ใหญ่ที่สุดในกลุ่ม "ทำได้เลยตอนนี้" แต่เป็นสิ่งที่ขอมาโดยตรงและมี visual impact สูงสุด
4. **Dispatch mode picker** — ปิดช่องว่างระหว่าง backend/frontend ที่ชัดเจนที่สุดตอนนี้
5. Dark mode + toast notification — ทำแทรกได้ทุกจังหวะที่ว่าง ไม่ผูกกับอะไร
6. จากนั้นรอ backend Phase 3 ปลดล็อกทีละตัวตามลำดับที่ [backend doc แนะนำ](../bot-orchestration-platform/PHASE3-ENTERPRISE-FEATURES-TODO.md#ลำดับที่แนะนำให้ทำ) แล้วค่อยทำ Flow Control view → Approve/Reject button → Condition dashboard → Alert inbox → Version/diff view ตามลำดับเดียวกัน

## Reference

- [projects/bot-orchestration-platform/PHASE3-ENTERPRISE-FEATURES-TODO.md](../bot-orchestration-platform/PHASE3-ENTERPRISE-FEATURES-TODO.md) — backend design ที่เอกสารนี้อ้างอิงทุกหัวข้อ
- [projects/bot-orchestration-platform/FEATURES.md](../bot-orchestration-platform/FEATURES.md) — สรุป feature ปัจจุบันของทั้งระบบ (backend + frontend)
- [PHASE1-DASHBOARD-TODO.md](PHASE1-DASHBOARD-TODO.md) — decision log ของโครงสร้างโปรเจกต์/convention ที่เอกสารนี้ยึดตาม (feature-based folder, SSE ไม่ใช่ WebSocket, no cross-feature import)
- [README.md](README.md) — วิธีรัน dashboard คู่กับ backend
