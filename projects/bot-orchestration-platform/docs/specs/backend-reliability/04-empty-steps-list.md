# 04: steps เป็นรายการว่าง

Spec: [backend-reliability.md](../backend-reliability.md)

**What to build:** API client วนอ่าน steps ของ Workflow Run ได้อย่างปลอดภัยเสมอ: Workflow Run ที่ยังไม่มี Step ไหนจบส่ง `steps: []` แทน `null`

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] รายละเอียด Workflow Run ที่ยังไม่มี Step จบตอบ `"steps": []`
- [ ] มี test ที่ครอบกรณีนี้
- [ ] golangci-lint ผ่าน
