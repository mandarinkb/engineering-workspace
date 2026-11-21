# 01: Prefactor: Bruno collection ของ seam A

Spec: [backend-reliability.md](../backend-reliability.md)

**What to build:** นักพัฒนารันชุดทดสอบ HTTP API ของ bop-api แบบ black-box กับ stack ที่รันอยู่จริงได้ด้วยคำสั่งเดียว (`bru run --env minipc`) เป็นฐานให้ ticket อื่นเพิ่ม request และ test ของพฤติกรรมใหม่ — HTTP test เดิมใน Go สร้าง server ด้วย Temporal client เป็น nil จึงทดสอบส่วนที่แตะ Temporal ไม่ได้

**Blocked by:** None (can start immediately)

**Status:** done — collection ครอบ endpoint ปัจจุบัน และแก้ bug `PUT /schedules` แล้ว (ยืนยันผ่านครบทั้งชุดตอน orchestrator deploy ครั้งแรก)

- [x] Bruno collection ใน repo ครอบ auth, bots, jobs, workflow runs และ schedules พร้อม environment `minipc` และ `local`
- [x] login เก็บ cookie แล้วส่งต่อทุก request; ข้อมูลที่สร้างขึ้นต้นด้วย `bruno-` และ Schedule ถูกลบเองตอนจบ
- [x] README อธิบายการติดตั้งและการรัน
- [x] แก้ `PUT /schedules/{id}` ที่ทำ bop-api panic (ไม่ส่ง Schedule.State ให้ SDK)
- [ ] `bru run --env minipc` ผ่านทั้งชุด — ยืนยันตอน orchestrator deploy ครั้งแรก
