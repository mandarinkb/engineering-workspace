# Spec: ความทนทานของ backend (Catch-up Window, Retry, Persistence, แบ่งหน้า Workflow Run)

> สถานะ: ready-for-agent · ศัพท์ทั้งหมดตาม [GLOSSARY.md](../../GLOSSARY.md) · เกี่ยวข้องกับ [ADR 0001](../adr/0001-temporal-is-source-of-truth-for-schedules.md)

## Problem Statement

ผู้ใช้ BOP เจอปัญหาจริงตอนใช้งาน stack บน Mini PC:

- หลังระบบหยุดไป 3 เดือน Schedule ที่รันทุกนาทีค้างจนใช้งานไม่ได้ เพราะ Temporal พยายามรัน Missed Run ย้อนหลังทั้งหมด ต้องลบ Schedule แล้วสร้างใหม่
- Workflow Run ที่ Step พังจริง (เช่นยิง URL ที่ไม่มีอยู่) ค้างสถานะ running ไม่ยอมเป็น failed เพราะ Step ถูกลองใหม่ไม่มีที่สิ้นสุด หน้า Overview จึงแทบไม่เคยแสดง Workflow Run ที่ล้มเหลว
- ทุกครั้งที่ deploy หรือ restart ประวัติ Job และ log หายหมด หน้า Jobs กลับเป็นศูนย์
- log ของ Step อ่านผ่าน API ไม่ได้เลย เพราะถูกเขียนไว้ใน memory ของอีก process
- รายการ Workflow Run มาไม่ครบเมื่อมีจำนวนมาก (Schedule รายนาทีสร้างวันละ 1,440 ตัว) และ dashboard ต้องโหลดทุกตัวมากรองเอง
- รายละเอียด Schedule ที่ worker สร้างเองไม่มี steps ให้เห็น และข้อมูล Schedule ที่ BOP จำไว้หายเมื่อ restart
- API ส่ง steps เป็น `null` สำหรับ Workflow Run ที่ยังไม่มี Step ไหนจบ ทำให้ client ที่ไม่ระวังพัง

## Solution

- Schedule มี Catch-up Window: Missed Run ที่เก่ากว่าระยะนี้ถูกข้าม ค่าเริ่มต้น 1 ชั่วโมง ตั้งต่อ Schedule ได้
- Step มีจำนวน Attempt สูงสุด ค่าเริ่มต้น 3 ครั้ง ตั้งต่อ Step ได้ เมื่อครบแล้วยังไม่สำเร็จเป็น Failed Step และ Workflow Run ล้มเหลว
- Job และ log (ทั้งของ Job และของ Step) เก็บใน PostgreSQL อยู่รอดข้าม restart log ที่เก่ากว่า 30 วันถูกลบอัตโนมัติ
- log ของ Step อ่านได้ผ่าน API ตาม Workflow Run และลำดับของ Step
- Schedule อ่านข้อมูลทั้งหมด (รวม steps) จาก Temporal ที่เดียว
- รายการ Workflow Run แบ่งหน้าและกรองฝั่ง server ได้ พร้อม endpoint นับจำนวน dashboard ใช้ปุ่ม "โหลดเพิ่ม" และเลือกกรองตาม Schedule
- API ส่ง steps เป็นรายการว่างแทน `null` เสมอ

## User Stories

### Catch-up Window

1. As an operator, I want a Schedule to skip Missed Runs older than its Catch-up Window after an outage, so that the system comes back healthy instead of getting stuck replaying months of runs.
2. As an operator, I want Missed Runs within the Catch-up Window to still run once after recovery, so that a short outage does not silently skip a daily report.
3. As an operator, I want every new Schedule to get a 1-hour Catch-up Window when I don't specify one, so that I am protected without having to remember the setting.
4. As an operator, I want to set the Catch-up Window per Schedule when creating it, so that reports that are useless when late can skip quickly and critical batches can catch up longer.
5. As an operator, I want to change the Catch-up Window when editing a Schedule, so that I can tune it after seeing real behaviour.
6. As an operator, I want editing an older Schedule to apply the default Catch-up Window if it has none, so that existing Schedules become safe the next time they are touched.
7. As an operator, I want the API to reject a Catch-up Window shorter than 10 seconds with a clear error, so that I don't create a Schedule Temporal will refuse.
8. As an operator, I want to see a Schedule's Catch-up Window on its detail page, so that I know how it will behave after an outage.
9. As an operator, I want a Catch-up Window field in the dashboard's Schedule form, so that I don't need curl to set it.
10. As an operator, I want the example Schedule created by the worker to have the default Catch-up Window, so that the demo Schedule never gets stuck again.
11. As an operator, I want Backfill to keep working exactly as before, so that explicitly requested historical runs are not limited by the Catch-up Window.

### Attempts และ Failed Step

12. As an operator, I want a Step to be tried at most 3 times by default, so that a broken Step turns into a Failed Step in minutes instead of running forever.
13. As an operator, I want retries to wait with exponential backoff starting at 1 second and capped at 1 minute, so that transient errors get a fair chance without hammering the target.
14. As an operator, I want to set the maximum number of Attempts per Step, so that flaky external calls can get more Attempts and expensive Steps can get fewer.
15. As an operator, I want a Step with a config error or an unknown Bot to fail on the first Attempt regardless of the maximum, so that errors that can never fix themselves don't waste time.
16. As an operator, I want the maximum Attempts to apply to every dispatch mode (local Bot, remote task queue, Kubernetes Job), so that behaviour is the same wherever the Step runs.
17. As an operator, I want a Workflow Run to become failed when one of its Steps becomes a Failed Step, and later Steps not to run, so that the run status reflects reality.
18. As an operator, I want failed Workflow Runs to appear in the Overview's "latest failed" panel, so that I notice problems from the landing page.
19. As an operator, I want a "maximum attempts" field on each Step in the dashboard (left empty means the default of 3), so that I can set it in both the Schedule form and the Run workflow form.
20. As an operator, I want standalone Jobs to keep their current no-retry behaviour, so that this change does not alter how one-off Jobs behave.

### Persistence ของ Job และ log

21. As an operator, I want Jobs to survive an API restart or redeploy, so that Job history and Overview numbers don't reset to zero.
22. As an operator, I want a Job's logs to survive a restart, so that I can investigate a failure after a deploy.
23. As an operator, I want Step logs written by the worker to be stored in the same place as Job logs, so that they can be read through the API.
24. As an operator, I want logs older than 30 days to be deleted automatically once a day, so that minute-level Schedules don't fill the disk.
25. As an operator, I want a Job older than 30 days to still show its status and summary even though its logs are gone, so that history stays useful.
26. As a developer, I want the API to fall back to in-memory storage with a clear warning when no database is configured, so that the lightweight `go run` workflow keeps working.
27. As a developer, I want database migrations to run automatically when the API starts, so that I never have to apply schema changes by hand.
28. As a developer, I want only the API process to run migrations, so that the API and worker never race on schema changes.
29. As a developer, I want a fresh stack to create the BOP database and user automatically, so that `compose up` on a new machine just works.
30. As an operator, I want a documented one-time command to create the BOP database on an existing Postgres volume, so that machines with existing data can upgrade.
31. As an operator, I want BOP's database to live on the same Postgres instance as Temporal but in its own database and user, so that the stack stays light while BOP data stays separate.

### log ของ Step ผ่าน API

32. As an operator, I want to read a Step's logs by Workflow Run and Step position, so that I can see what each Step did without knowing Temporal internals.
33. As an operator, I want a Step's logs to include every Attempt in order with each line marked by its Attempt, so that I can see why earlier Attempts failed.
34. As an operator, I want a clear "not found" response for a Step position that doesn't exist, so that mistakes are obvious.

### Schedule อ่านจาก Temporal ที่เดียว

35. As an operator, I want a Schedule's steps on its detail page to come from Temporal, so that what I see is exactly what will run.
36. As an operator, I want Schedules created outside the API (such as the worker's example Schedule) to show their steps, so that every Schedule is inspectable.
37. As an operator, I want a Schedule deleted through Temporal UI to disappear from BOP immediately, so that there is no stale data.
38. As an operator, I want Schedule details to stay correct after an API restart, so that BOP never "forgets" what a Schedule does.

### แบ่งหน้าและกรอง Workflow Run

39. As an operator, I want the Workflow Run list to return pages of 50 by default with a token for the next page, so that the list stays fast however many runs exist.
40. As an API client, I want to request up to 200 Workflow Runs per page, with larger values clamped to 200, so that I can fetch in bigger chunks without errors.
41. As an operator, I want to filter Workflow Runs by status (running, completed, failed which includes terminated and timed out, canceled), so that I can find failures without scanning everything.
42. As an operator, I want to filter Workflow Runs by the Schedule that started them, so that I can review one Schedule's history.
43. As an operator, I want each Workflow Run to state which Schedule started it (or none for ad-hoc runs), so that I can tell scheduled and manual runs apart.
44. As an operator, I want to count Workflow Runs by status, so that the Overview shows how many are running even when there are more than one page.
45. As an operator, I want the dashboard Workflow Runs page to load more rows with a "load more" button, so that I can keep scrolling through history.
46. As an operator, I want a Schedule dropdown on the Workflow Runs page instead of the ID prefix search, so that filtering by Schedule is exact.
47. As an operator, I want the Overview's "Workflow running" number to come from the count endpoint and the "latest failed" panel from a failed-status page of 5, so that both stay correct at any volume.
48. As an operator, I want the auto-refresh of the Workflow Runs page to keep working with pagination, so that new runs appear without losing the rows I already loaded.

### steps เป็นรายการว่าง

49. As an API client, I want a Workflow Run with no finished Step to return an empty steps list rather than null, so that I can iterate safely.

## Implementation Decisions

### Catch-up Window
- Schedule input (create และ update) รับ `catchup_window_seconds` แบบ optional: ไม่ส่ง = 3600, น้อยกว่า 10 = invalid (400)
- Schedule detail ส่ง `catchup_window_seconds` กลับมา
- adapter ของ Temporal Scheduler ตั้ง CatchupWindow ทั้งตอน create และ update
- example Schedule ของ worker ตั้งค่าเริ่มต้นเดียวกัน
- ไม่มีโค้ดไล่อัปเดต Schedule เดิมตอน start (ปัจจุบันมีแค่ example Schedule ซึ่ง worker สร้างให้อยู่แล้ว)

### Attempts
- Step ได้ field ใหม่ `max_attempts` (optional, จำนวนเต็ม ≥ 1) คู่กับ `timeout_seconds` เดิม
- RetryPolicy ของทุก Step: MaximumAttempts = `max_attempts` หรือ 3, InitialInterval 1s, BackoffCoefficient 2, MaximumInterval 1m, NonRetryableErrorTypes เดิม (`ConfigError`, `UnknownBot`)
- ใช้กับทั้ง activity ของ Bot ปกติและ activity ของ Kubernetes Job
- Job เดี่ยว (worker pool) ไม่เปลี่ยน

### Persistence
- PostgreSQL ตัวเดียวกับ Temporal แต่ database `bop` และ user `bop` แยก (dev ใช้รหัสผ่าน `bop` เขียนกำกับว่าเป็นค่า dev เหมือน `BOP_JWT_SECRET`)
- ตั้งค่าผ่าน `BOP_DATABASE_URL` ทั้ง API และ worker; ไม่ตั้ง = repository/logger แบบ memory เดิมพร้อม log เตือนตอน start
- driver `pgx`; migration ด้วย `golang-migrate` แบบฝังไฟล์ SQL ใน binary; API รัน migration ตอน start, worker ไม่รัน
- ข้อมูลที่เก็บ: Job และ log line เท่านั้น
- log line ผูกกับเจ้าของได้ 2 แบบ: Job (ด้วย Job ID) หรือ Step (ด้วย Workflow ID + ลำดับ Step + หมายเลข Attempt) — แทนรูปแบบ log ID เดิมที่ใช้ Activity ID
- งานลบ log ที่เก่ากว่า 30 วันรันวันละครั้งใน API process
- implementation ใหม่ต้องเป็นไปตาม interface `job.Repository` และ logger เดิม เพื่อสลับกับแบบ memory ได้
- compose ที่ root: เพิ่ม init script สร้าง database/user สำหรับ volume ใหม่ และตั้ง `BOP_DATABASE_URL` ให้ API กับ worker; README อธิบายคำสั่งสร้างครั้งเดียวสำหรับ volume เดิม

### Schedule จาก Temporal (ADR 0001)
- steps ของ Schedule detail ถอดรหัสจาก action args ของ Temporal Schedule
- ลบ `schedule.Repository` และ implementation แบบ memory ออก
- ต้องยืนยันก่อนลงมือว่าถอดรหัส action args ด้วย SDK ได้จริง ถ้าไม่ได้ให้กลับไปตัดสินใจใหม่ก่อนทำต่อ

### log ของ Step
- endpoint ใหม่ `GET /workflow-runs/{id}/steps/{index}/logs` คืน log line ตามลำดับเวลาของทุก Attempt แต่ละบรรทัดมี `attempt`; ลำดับ Step ที่ไม่มีจริง = 404

### Workflow Run list
- `GET /workflow-runs` รับ `page_size` (default 50, สูงสุด 200 ปัดลง), `page_token` (opaque จาก Temporal), `status` (running | completed | failed | canceled; failed รวม terminated และ timed out), `schedule_id`
- response เปลี่ยนจาก array เป็น object: `{ "runs": [...], "next_page_token": "..." }` (ไม่มีหน้าถัดไป = ว่าง) — breaking change, dashboard แก้ในรอบเดียวกัน
- แต่ละ run มี `schedule_id` (มาจาก search attribute `TemporalScheduledById`; ว่างสำหรับ run ที่สั่งเอง)
- endpoint ใหม่ `GET /workflow-runs/count?status=` ใช้ CountWorkflowExecutions คืน `{ "count": n }` — รับ `schedule_id` ด้วย (filter เดียวกับ list)
- ตามไป Temporal ด้วย `NextPageToken` จริง ไม่ใช่เรียกหน้าเดียวแบบเดิม

### steps เป็นรายการว่าง
- Workflow Run detail ส่ง `steps: []` เมื่อยังไม่มี Step ไหนจบ

### Dashboard
- `StepsEditor` เพิ่มช่อง "จำนวนครั้งที่ลองสูงสุด" (ว่างได้)
- ฟอร์ม Schedule เพิ่มช่อง Catch-up Window (หน่วยวินาที ค่าเริ่มต้น 3600)
- หน้า Workflow Runs: ปุ่ม "โหลดเพิ่ม" แทน pager, dropdown เลือก Schedule แทนช่องค้นหา prefix, filter สถานะส่งไป server
- Overview: "Workflow กำลังรัน" จาก count endpoint, "ล้มเหลวล่าสุด" จาก `status=failed&page_size=5`
- ลำดับงาน: 1 Catch-up Window → 2 Attempts → 5 steps ว่าง → 3 Persistence (หลาย commit) → 4 แบ่งหน้า; แต่ละข้อมี test, golangci-lint ผ่าน, commit แยก และ deploy ขึ้น Mini PC พร้อมทดสอบก่อนข้อถัดไป

## Testing Decisions

- test ที่ดีตรวจพฤติกรรมที่มองเห็นจากภายนอก (HTTP response, สถานะ Workflow Run, สิ่งที่อ่านได้หลัง restart) ไม่ตรวจรายละเอียดภายใน เช่นชื่อ column หรือลำดับการเรียก method
- **Seam A (ใหม่) — Bruno collection ยิง HTTP API แบบ black-box กับ stack ที่รันอยู่จริง** (`bru run --env minipc`) ครอบคลุม Catch-up Window ที่ตั้งผ่าน API แล้วอ่านกลับได้, steps ของ Schedule จาก Temporal, การแบ่งหน้า/filter/count ของ Workflow Run, log ของ Step รวมหลาย Attempt, `steps: []` — Step ที่ต้องล้มเหลวใช้ Bot จริงยิง URL ที่ไม่มีอยู่ (ใส่ stub Bot แบบ in-process ไม่ได้) และกรณี "อยู่รอดหลัง restart" ใช้การ recreate container `bop-api` คั่นระหว่างการรัน collection
- **Seam B (มีอยู่แล้ว) — Temporal workflow testsuite** ของ `Execute`: Step ล้มเหลวหลังครบ `max_attempts`, ค่าเริ่มต้น 3, error ที่ retry ไม่ได้ล้มเหลวใน Attempt แรก, Step ถัดไปไม่รัน — prior art คือ `TestExecute_StopsAtFirstFailedStep`
- **Seam C (มีอยู่แล้ว) — dashboard**: vitest สำหรับ logic ล้วน (prior art: filter ของ workflow run, `validateSteps`) และ Playwright กับ stack บน Mini PC สำหรับฟอร์ม, "โหลดเพิ่ม", dropdown Schedule และตัวเลข Overview
- HTTP test เดิมที่สร้าง server ด้วย Temporal client เป็น nil ยังใช้ทดสอบส่วนที่ไม่แตะ Temporal ต่อไป (เช่น validation ของ input) — prior art คือ `newTestServer`
- ข้อจำกัดที่ยอมรับ: พฤติกรรม Catch-up Window ตอนล่มจริงหลายชั่วโมงไม่ได้จำลองแบบอัตโนมัติ seam A ตรวจแค่ว่าค่าถูกตั้งใน Temporal Schedule

## Out of Scope

- แบ่งหน้าของ `/jobs` (Job เพิ่มช้า ทำเมื่อเริ่มช้าจริง)
- Retry ให้ Job เดี่ยว
- หน้า dashboard ที่แสดง log ของ Step (API พร้อมแล้ว UI เป็นงานถัดไป)
- โค้ดไล่อัปเดต Catch-up Window ให้ Schedule เดิมทุกตัวตอน start
- Prometheus metric, Job dependency (DAG) และฟีเจอร์ Phase 3
- เก็บรหัสผ่านฐานข้อมูลเป็น secret สำหรับ production

## Further Notes

- เหตุการณ์จริงที่เป็นที่มาของข้อ Catch-up Window: log ของ Temporal ขึ้น "Buffer too large" แล้ว "Workflow panic" สำหรับ scheduler workflow ของ example Schedule และ backend ตอบ "Unable to query workflow due to Workflow Task in failed state"
- ค่าเริ่มต้นของ CatchupWindow ฝั่ง Temporal (เข้าใจว่า 1 ปี) ยังไม่ได้ยืนยันกับเอกสาร — ไม่กระทบการตัดสินใจเพราะเราตั้งค่าเองทุกครั้ง
- ขนาดหน้าของ ListWorkflowExecutions เมื่อไม่ระบุยังไม่ได้ยืนยัน — ไม่กระทบเพราะจะตั้ง `page_size` เองทุกครั้ง
- คำตอบ 2 ข้อสุดท้ายของรอบซัก (ช่อง max attempts ใน dashboard และรหัสผ่าน dev `bop`/`bop`) ใช้ตามที่แนะนำ ผู้ใช้ยังไม่ได้ยืนยันโดยตรง
