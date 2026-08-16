# วิธีไล่โค้ด BOP ให้เข้าใจเร็ว

> ไฟล์นี้เป็น**ทริกการอ่านโค้ด** ไม่ใช่ design doc — ใช้ตอนกลับมาไล่โค้ดที่มีอยู่แล้วเพื่อทำความเข้าใจว่าทำงานยังไงจริงๆ (ต่างจาก `FULLSTACK-INFRA-ROADMAP.md` ที่บอกว่าต้องทำอะไรต่อ)
>
> **จุดแข็งที่ควรใช้ประโยชน์เต็มที่**: โค้ดทุกไฟล์ในโปรเจกต์ BOP มี comment ภาษาไทยอธิบาย "ทำงานยังไง" ไว้ละเอียดตาม convention ที่ตั้งไว้ใน `CLAUDE.md` — ไม่ต้องเดาเจตนาของโค้ดเองเหมือน codebase ทั่วไป

## 1. เริ่มจาก Entry Point ไม่ใช่ไล่ตาม File Tree

อ่านไฟล์ที่เป็นจุดเริ่มต้นของ process ก่อนเสมอ — มี comment หัวไฟล์อธิบาย "ขั้นตอนที่ 1, 2, 3..." ไว้ให้แล้ว ทำให้เห็น mental map ว่าชิ้นส่วนต่อกันยังไงก่อนไปดูรายละเอียดทีละไฟล์:

- `projects/bot-orchestration-platform/cmd/bop/main.go` — HTTP API server
- `projects/bot-orchestration-platform/cmd/bop-worker/main.go` — Temporal worker
- `projects/bop-dashboard/src/app/(dashboard)/layout.tsx` — จุดเริ่มของ dashboard UI

## 2. ไล่ตาม "1 Flow เต็ม" แทนที่จะอ่านทุกไฟล์เรียงลำดับ

เลือก flow เดียวที่เป็น user action จริง แล้วไล่ตามเส้นทางจริงจนจบ แทนที่จะอ่านไฟล์แยกกันทีละไฟล์ตามลำดับ folder — ได้ working mental model เร็วกว่ามาก เพราะเห็นว่าชิ้นส่วนคุยกันจริงยังไง ไม่ใช่แค่โครงสร้างนิ่งๆ

ตัวอย่าง flow ที่คุ้มไล่ก่อน:

- `POST /bots/{name}/jobs` → `internal/api/http/handler.go` → `worker.Pool` → `bot.Run()` → `logger.Logger` → SSE stream (`/jobs/{id}/logs/stream`) → `use-job-log-stream.ts` ฝั่ง React
- `POST /workflow-runs` (step หลายตัว) → `internal/workflow/temporal.go` → Temporal Activity → `internal/workflow/activity.go`

## 3. อ่าน `*-TODO.md` ก่อนเข้าโค้ด ไม่ใช่หลัง

repo นี้มี decision-log แบบนี้เยอะมาก อธิบาย "ทำไมเลือกทางนี้ ไม่เลือกทางนั้น" ซึ่ง comment ในโค้ดเองไม่ได้ให้บริบทระดับนี้เสมอไป อ่านก่อนเข้าโค้ดจะเข้าใจ "ทำไม" ก่อนเจอ "ทำอะไร" — ช่วยไม่ให้งงตอนเจอโค้ดที่ดู over-engineered หรือ under-engineered โดยไม่มีบริบท

ตัวอย่าง: `projects/bot-orchestration-platform/TEMPORAL-MIGRATION-TODO.md`, `PHASE1-DASHBOARD-TODO.md`, `K8S-JOB-HYBRID-EXECUTION-TODO.md`, `EXTERNAL-JOB-BOT-TODO.md`, `REMOTE-JOB-WORKER-TODO.md`

## 4. อ่าน Test คู่กับ Implementation

table-driven test ของ Go (`*_test.go`) และ Vitest ของ dashboard (`*.test.ts`/`*.test.tsx`) บอก "พฤติกรรมที่คาดหวัง" แบบแยกส่วนชัดกว่าอ่าน implementation ตรงๆ โดยเฉพาะจุดที่ยาก:

- `internal/worker/*_test.go` — concurrency ของ `worker.Pool`
- `internal/auth/jwt_test.go` — JWT issue/verify
- `internal/api/http/handler_test.go` — HTTP handler ทุกเส้นทาง

## 5. Feynman Technique (ตามที่ `ROADMAP.md` เขียนไว้แล้ว)

อ่านจบ 1 component แล้วลองอธิบายเป็นคำพูดตัวเองโดยไม่ดูโค้ด — ถ้าติดตรงไหนคือจุดที่ยังไม่เข้าใจจริง กลับไปอ่านซ้ำเฉพาะจุดนั้น ดีกว่าไล่อ่านทุกบรรทัดเท่ากันหมดทั้งไฟล์

## ลำดับที่แนะนำเวลาเริ่มไล่จริง

```
1. อ่าน README.md ของ project นั้นก่อน (ภาพรวม)
2. อ่าน *-TODO.md ที่เกี่ยวข้องกับ feature ที่จะไล่ (บริบท "ทำไม")
3. เริ่มจาก entry point (cmd/*/main.go หรือ layout.tsx)
4. ไล่ตาม 1 flow เต็มที่สนใจ ไม่ใช่ทุกไฟล์
5. อ่าน test ของจุดที่งงที่สุด
6. อธิบายด้วยคำพูดตัวเอง (Feynman) ก่อนไปจุดถัดไป
```
