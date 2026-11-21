# 04: steps เป็นรายการว่าง

Spec: [backend-reliability.md](../backend-reliability.md)

**What to build:** API client วนอ่าน steps ของ Workflow Run ได้อย่างปลอดภัยเสมอ: Workflow Run ที่ยังไม่มี Step ไหนจบส่ง `steps: []` แทน `null`

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] รายละเอียด Workflow Run ที่ยังไม่มี Step จบตอบ `"steps": []`
- [ ] มี test ที่ครอบกรณีนี้
- [ ] golangci-lint ผ่าน
- [ ] Bruno: อ่าน Workflow Run ทันทีหลังสั่งรัน (ก่อน Step แรกจบ) ได้ `steps` เป็น array
- [ ] เพิ่ม request และ test ของพฤติกรรมใหม่ลงใน Bruno collection (`bruno/`) ตามหลักใน `bruno/README.md` และแก้ test เดิมที่พฤติกรรมเปลี่ยน
- [ ] orchestrator deploy integration branch แล้ว `bru run --env minipc` ผ่านทั้งชุด (implementer ไม่ deploy และไม่รัน Bruno กับ Mini PC เอง)
