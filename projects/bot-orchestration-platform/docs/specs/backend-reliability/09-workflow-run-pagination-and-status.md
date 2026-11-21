# 09: แบ่งหน้าและ filter สถานะของ Workflow Run

Spec: [backend-reliability.md](../backend-reliability.md)

**What to build:** รายการ Workflow Run เร็วและครบไม่ว่าจะมีกี่ตัว: API แบ่งหน้าด้วย token และกรองสถานะฝั่ง server หน้า Workflow Runs ใน dashboard โหลดทีละหน้าด้วยปุ่ม "โหลดเพิ่ม"

**Blocked by:** 01 (Prefactor: test harness ของ seam A)

**Status:** implemented

- [x] `GET /workflow-runs` รับ `page_size` (default 50, สูงสุด 200 ปัดลง), `page_token`, `status` (running | completed | failed ซึ่งรวม terminated/timed out | canceled)
- [x] response เป็น `{ "runs": [...], "next_page_token": "..." }` (ไม่มีหน้าถัดไป = ว่าง)
- [ ] seam A: เดินครบทุกหน้าได้ run ครบไม่ซ้ำ; filter สถานะคืนเฉพาะสถานะนั้น — Bruno ตรวจ 2 หน้าแรกไม่ซ้ำและทุก filter สถานะแล้ว รอ orchestrator รันกับ stack จริง
- [x] dashboard: หน้า Workflow Runs ใช้ปุ่ม "โหลดเพิ่ม" แทน pager และส่ง filter สถานะไป server
- [x] dashboard: auto-refresh ทุก 10 วินาทีไม่ทำให้แถวที่โหลดเพิ่มไว้หายและเห็น run ใหม่
- [x] Overview ยังทำงานได้กับ response รูปแบบใหม่
- [x] golangci-lint, vitest และ eslint ผ่าน
- [x] Bruno: แก้ `04-workflow-runs/04-list-runs` ให้รับ response แบบ object, ตรวจ 2 หน้าแรกด้วย `page_size` เล็ก (ไม่เดินทุกหน้า เพราะ Mini PC มี run สะสมหลายพันตัว) ว่าไม่ซ้ำกัน, filter สถานะ, `page_size` เกิน 200 ถูกปัดลง
- [x] เพิ่ม request และ test ของพฤติกรรมใหม่ลงใน Bruno collection (`bruno/`) ตามหลักใน `bruno/README.md` และแก้ test เดิมที่พฤติกรรมเปลี่ยน
- [ ] orchestrator deploy integration branch แล้ว `bru run --env minipc` ผ่านทั้งชุด (implementer ไม่ deploy และไม่รัน Bruno กับ Mini PC เอง)
