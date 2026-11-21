# 09: แบ่งหน้าและ filter สถานะของ Workflow Run

Spec: [backend-reliability.md](../backend-reliability.md)

**What to build:** รายการ Workflow Run เร็วและครบไม่ว่าจะมีกี่ตัว: API แบ่งหน้าด้วย token และกรองสถานะฝั่ง server หน้า Workflow Runs ใน dashboard โหลดทีละหน้าด้วยปุ่ม "โหลดเพิ่ม"

**Blocked by:** 01 (Prefactor: test harness ของ seam A)

**Status:** ready-for-agent

- [ ] `GET /workflow-runs` รับ `page_size` (default 50, สูงสุด 200 ปัดลง), `page_token`, `status` (running | completed | failed ซึ่งรวม terminated/timed out | canceled)
- [ ] response เป็น `{ "runs": [...], "next_page_token": "..." }` (ไม่มีหน้าถัดไป = ว่าง)
- [ ] seam A: เดินครบทุกหน้าได้ run ครบไม่ซ้ำ; filter สถานะคืนเฉพาะสถานะนั้น
- [ ] dashboard: หน้า Workflow Runs ใช้ปุ่ม "โหลดเพิ่ม" แทน pager และส่ง filter สถานะไป server
- [ ] dashboard: auto-refresh ทุก 10 วินาทีไม่ทำให้แถวที่โหลดเพิ่มไว้หายและเห็น run ใหม่
- [ ] Overview ยังทำงานได้กับ response รูปแบบใหม่
- [ ] golangci-lint, vitest และ eslint ผ่าน; deploy ขึ้น Mini PC แล้วทดสอบผ่าน browser
