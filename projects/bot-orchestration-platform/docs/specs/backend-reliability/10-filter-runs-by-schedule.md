# 10: กรอง Workflow Run ตาม Schedule

Spec: [backend-reliability.md](../backend-reliability.md)

**What to build:** Operator ดูประวัติของ Schedule ตัวเดียวได้แม่นยำ และแยก Workflow Run ที่มาจาก Schedule กับที่สั่งเองได้

**Blocked by:** 09 (แบ่งหน้าและ filter สถานะของ Workflow Run)

**Status:** ready-for-agent

- [ ] แต่ละ run มี `schedule_id` (ว่างสำหรับ run ที่สั่งเอง)
- [ ] `GET /workflow-runs?schedule_id=` คืนเฉพาะ run ของ Schedule นั้น ใช้ร่วมกับ `status` และการแบ่งหน้าได้
- [ ] seam A: run จาก Schedule มี `schedule_id` ถูกต้อง ad-hoc run ว่าง และ filter ทำงาน
- [ ] dashboard: dropdown เลือก Schedule แทนช่องค้นหาตาม prefix ของ ID
- [ ] golangci-lint, vitest และ eslint ผ่าน
- [ ] Bruno: run จาก Schedule มี `schedule_id`, ad-hoc run ไม่มี, `?schedule_id=` กรองได้
- [ ] เพิ่ม request และ test ของพฤติกรรมใหม่ลงใน Bruno collection (`bruno/`) ตามหลักใน `bruno/README.md` และแก้ test เดิมที่พฤติกรรมเปลี่ยน
- [ ] orchestrator deploy integration branch แล้ว `bru run --env minipc` ผ่านทั้งชุด (implementer ไม่ deploy และไม่รัน Bruno กับ Mini PC เอง)
