# 13: รายละเอียด Schedule คืนชื่อ Workflow ที่ตั้งไว้ ไม่ใช่ชื่อ workflow type

Spec: [backend-reliability.md](../backend-reliability.md)

**What to build:** `GET /schedules/{id}` ส่ง `workflow_name` เป็นชื่อที่ผู้ใช้ตั้งตอนสร้าง (ชื่อที่เก็บอยู่ใน args ของ action ซึ่งถอดรหัสได้อยู่แล้วตาม [ADR 0001](../../adr/0001-temporal-is-source-of-truth-for-schedules.md)) แทนชื่อ workflow type ของ Temporal ซึ่งเป็น `Execute` เสมอ เพื่อให้อ่านแล้วส่งกลับด้วย `PUT` ไม่เปลี่ยนชื่อ

**ที่มา (พบตอน deploy ticket 01-12 บน Mini PC):** อ่าน `check-example-com-schedule` ได้ `workflow_name: "Execute"` แล้วส่งค่านั้นกลับใน `PUT` ผลคือ Action ID ของ workflow ที่ schedule สั่งเปลี่ยนจาก `check-example-com` เป็น `Execute` ทำให้ run ถัดไปมี ID ขึ้นต้นด้วย `Execute-` ฟอร์มแก้ไข Schedule ใน dashboard เติมค่าจาก `GET` เดียวกัน — **ยืนยันแล้วด้วย browser (2026-11-21):** สร้าง Schedule ด้วยชื่อ workflow `uiwalk` แล้วเปิดฟอร์มแก้ไข ช่องชื่อถูกเติมเป็น `Execute` กดบันทึกโดยไม่แก้อะไร Action ID ใน Temporal กลายเป็น `Execute` และ run ถัดไปได้ ID `Execute-…` แทน `uiwalk-…` จึงเกิดกับทุกคนที่เปิดฟอร์มแก้ไขแล้วบันทึกโดยไม่เปลี่ยนชื่อ

**Blocked by:** None (can start immediately)

**Status:** done

**Verified:** deploy branch `fix/schedule-name-and-run-refresh` (commit 5beb18b) ขึ้น Mini PC แล้ว `bru run --env minipc` ผ่านทั้งชุด 91 request / 106 test (รวมอ่านแล้วส่งกลับด้วย PUT ชื่อไม่เปลี่ยน) ตรวจด้วย Playwright: ฟอร์มแก้ไขเติม `uiwalk` ไม่ใช่ `Execute` และ run ถัดไปหลังบันทึกขึ้นต้น `uiwalk-` และ Temporal ยืนยัน Action ID ของ Schedule ยังเป็น `uiwalk` — 2026-11-21

**ข้อจำกัดของ list:** `GET /schedules` ใช้ ListSchedules ซึ่ง `ScheduleListEntry` (SDK v1.47.0) ไม่มี action args มีแค่ `WorkflowType` จึงยังได้ "Execute" ใน `workflow_name` — ไม่เพิ่ม Describe ทีละ Schedule (N+1) ฟอร์มแก้ไขใช้ detail endpoint จึงไม่กระทบ

- [x] `GET /schedules/{id}` ส่ง `workflow_name` เป็นชื่อที่เก็บใน args ของ action; ถ้า args ไม่ใช่รูปแบบของ BOP (Schedule ที่สร้างนอกระบบ) ให้ย้อนกลับไปใช้ชื่อ workflow type เหมือนเดิม
- [x] อ่านแล้วส่งกลับใน `PUT` (round trip) ไม่เปลี่ยน Action ID และชื่อใน args ของ Schedule (ตรวจด้วย Go test แล้ว; ส่วน Temporal จริงตรวจโดย Bruno ตอน orchestrator deploy)
- [x] Go test: Describe คืนชื่อจาก args (List ไม่มี args ให้ถอดรหัส), กรณี args ไม่ใช่รูปแบบ BOP, และ round trip
- [x] Bruno (detail endpoint): สร้าง Schedule ด้วย `workflow_name` ที่ไม่ใช่ `Execute` แล้ว `GET` ได้ชื่อเดิม; `PUT` ด้วยค่าที่ได้จาก `GET` แล้ว `GET` อีกครั้งชื่อเท่าเดิม และ run ที่ trigger หลังจากนั้นมี workflow ID ขึ้นต้นด้วยชื่อนั้น
- [x] dashboard: ฟอร์มแก้ไข Schedule เติมชื่อที่ถูกต้อง และบันทึกโดยไม่แก้ชื่อแล้ว Action ID ไม่เปลี่ยน (ตรวจด้วย Playwright กับ Schedule ที่สร้างเอง ไม่ใช้ `check-example-com-schedule` ของ worker)
- [x] golangci-lint (ผ่านแล้ว), vitest และ eslint ผ่าน
- [x] orchestrator deploy integration branch แล้ว `bru run --env minipc` ผ่านทั้งชุด (implementer ไม่ deploy และไม่รัน Bruno กับ Mini PC เอง)
