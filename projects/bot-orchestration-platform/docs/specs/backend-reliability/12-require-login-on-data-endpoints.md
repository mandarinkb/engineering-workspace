# 12: บังคับ login กับทุก endpoint ข้อมูล

Spec: [backend-reliability.md](../backend-reliability.md)

**What to build:** ทุก endpoint ที่อ่านหรือเปลี่ยนข้อมูล (bots, jobs, workflow runs, schedules) ตอบ 401 เมื่อไม่มี session ที่ถูกต้อง — ปัจจุบัน `GET /jobs` ตอบ 200 ทั้งที่ไม่ได้ login ใครก็ตามใน LAN สร้าง Job สั่งรัน Workflow หรือลบ Schedule ได้ ขณะที่ dashboard เขียนไว้ว่า security boundary ตัวจริงอยู่ที่ backend endpoint ที่ยังเปิดโดยไม่ต้อง login มีแค่ `/login`, `/logout`, `/healthz`, `/readyz` (และ `/me` ที่ตอบ 401 เองอยู่แล้ว)

**Blocked by:** 01 (Prefactor: Bruno collection ของ seam A)

**Status:** ready-for-agent

- [ ] ทุก endpoint ข้อมูลตอบ 401 เมื่อไม่มี cookie หรือ token ไม่ถูกต้อง/หมดอายุ
- [ ] `/login`, `/logout`, `/healthz`, `/readyz` และ preflight `OPTIONS` ยังเรียกได้โดยไม่ต้อง login
- [ ] SSE stream ของ log ใช้ cookie เดิมได้ (EventSource ส่ง cookie ด้วย `withCredentials`)
- [ ] Go HTTP test ครอบทั้งกรณีมีและไม่มี session
- [ ] dashboard: ได้ 401 ระหว่างใช้งาน (เช่น session หมดอายุ) แล้วพาไปหน้า login แทนการแสดง error ค้าง
- [ ] Bruno: โฟลเดอร์ทดสอบว่าทุก endpoint ข้อมูลตอบ 401 เมื่อไม่มี session (ส่ง header `Cookie` ว่าง) และ request เดิมทั้งหมดยังผ่านเพราะส่ง cookie อยู่แล้ว
- [ ] golangci-lint, vitest และ eslint ผ่าน
- [ ] orchestrator deploy integration branch แล้ว `bru run --env minipc` ผ่านทั้งชุด
