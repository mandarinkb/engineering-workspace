# 04: steps เป็นรายการว่าง

Spec: [backend-reliability.md](../backend-reliability.md)

**What to build:** API client วนอ่าน steps ของ Workflow Run ได้อย่างปลอดภัยเสมอ: Workflow Run ที่ยังไม่มี Step ไหนจบส่ง `steps: []` แทน `null`

**Blocked by:** None (can start immediately)

**Status:** done

**Verified:** deploy integration branch (commit 311a6d2) ขึ้น Mini PC แล้ว `bru run --env minipc` ผ่านทั้งชุด 88 request / 101 test, `bruno/run-with-restart.sh` ผ่าน (after-restart 5/5 + stream ของ log), ตรวจหน้า dashboard ด้วย Playwright — 2026-11-21

- [x] รายละเอียด Workflow Run ที่ยังไม่มี Step จบตอบ `"steps": []`
- [x] มี test ที่ครอบกรณีนี้
- [x] golangci-lint ผ่าน
- [x] Bruno: อ่าน Workflow Run ทันทีหลังสั่งรัน (ก่อน Step แรกจบ) ได้ `steps` เป็น array
- [x] เพิ่ม request และ test ของพฤติกรรมใหม่ลงใน Bruno collection (`bruno/`) ตามหลักใน `bruno/README.md` และแก้ test เดิมที่พฤติกรรมเปลี่ยน
- [x] orchestrator deploy integration branch แล้ว `bru run --env minipc` ผ่านทั้งชุด (implementer ไม่ deploy และไม่รัน Bruno กับ Mini PC เอง)
