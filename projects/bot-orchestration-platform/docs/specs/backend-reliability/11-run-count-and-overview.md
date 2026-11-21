# 11: endpoint นับจำนวนและ Overview ที่ถูกต้องเมื่อข้อมูลเยอะ

Spec: [backend-reliability.md](../backend-reliability.md)

**What to build:** ตัวเลขในหน้า Overview ถูกต้องไม่ว่าจะมี Workflow Run กี่ตัว: จำนวนที่กำลังรันมาจากการนับฝั่ง server และรายการล้มเหลวล่าสุดมาจากการกรองฝั่ง server

**Blocked by:** 09 (แบ่งหน้าและ filter สถานะของ Workflow Run)

**Status:** ready-for-agent

- [ ] `GET /workflow-runs/count?status=` คืน `{ "count": n }`
- [ ] seam A: count ตรงกับจำนวนจริงเมื่อมี run มากกว่า 1 หน้า
- [ ] Overview: "Workflow กำลังรัน" มาจาก count และ "ล้มเหลวล่าสุด" มาจาก `status=failed&page_size=5`
- [ ] golangci-lint, vitest และ eslint ผ่าน
- [ ] Bruno: `count` ของสถานะหนึ่งเท่ากับจำนวนที่ได้จากการเดินหน้าด้วย filter เดียวกัน (ใช้สถานะที่จำนวนน้อย เช่น running หรือ failed)
- [ ] เพิ่ม request และ test ของพฤติกรรมใหม่ลงใน Bruno collection (`bruno/`) ตามหลักใน `bruno/README.md` และแก้ test เดิมที่พฤติกรรมเปลี่ยน
- [ ] orchestrator deploy integration branch แล้ว `bru run --env minipc` ผ่านทั้งชุด (implementer ไม่ deploy และไม่รัน Bruno กับ Mini PC เอง)
