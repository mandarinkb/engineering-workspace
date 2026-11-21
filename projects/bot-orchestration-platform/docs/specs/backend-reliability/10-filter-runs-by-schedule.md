# 10: กรอง Workflow Run ตาม Schedule

Spec: [backend-reliability.md](../backend-reliability.md)

**What to build:** Operator ดูประวัติของ Schedule ตัวเดียวได้แม่นยำ และแยก Workflow Run ที่มาจาก Schedule กับที่สั่งเองได้

**Blocked by:** 09 (แบ่งหน้าและ filter สถานะของ Workflow Run)

**Status:** implemented

- [x] แต่ละ run มี `schedule_id` (ว่างสำหรับ run ที่สั่งเอง)
- [x] `GET /workflow-runs?schedule_id=` คืนเฉพาะ run ของ Schedule นั้น ใช้ร่วมกับ `status` และการแบ่งหน้าได้
- [ ] seam A: run จาก Schedule มี `schedule_id` ถูกต้อง ad-hoc run ว่าง และ filter ทำงาน — Bruno ตรวจแล้ว (run ของ Schedule ตัวอย่าง, ad-hoc run ว่าง, filter ร่วม status/แบ่งหน้า) รอ orchestrator รันกับ stack จริง
- [x] dashboard: dropdown เลือก Schedule แทนช่องค้นหาตาม prefix ของ ID
- [x] golangci-lint, vitest และ eslint ผ่าน
- [x] Bruno: run จาก Schedule มี `schedule_id`, ad-hoc run ไม่มี, `?schedule_id=` กรองได้
- [x] เพิ่ม request และ test ของพฤติกรรมใหม่ลงใน Bruno collection (`bruno/`) ตามหลักใน `bruno/README.md` และแก้ test เดิมที่พฤติกรรมเปลี่ยน
- [ ] orchestrator deploy integration branch แล้ว `bru run --env minipc` ผ่านทั้งชุด (implementer ไม่ deploy และไม่รัน Bruno กับ Mini PC เอง)
