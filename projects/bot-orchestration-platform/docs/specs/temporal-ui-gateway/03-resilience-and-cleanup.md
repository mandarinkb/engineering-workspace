# 03: ความทนทาน (fail closed, ตัด Cookie, WebSocket/SSE) และเก็บงาน

Spec: [temporal-ui-gateway.md](../temporal-ui-gateway.md)

**What to build:** ผู้ดูแลมั่นใจว่า gateway ไม่เปิดโล่งเมื่อ `bop-api` ล่ม, Temporal UI ไม่ได้รับ cookie ของผู้ใช้, หน้าที่อัปเดตสดใช้ได้ และเอกสารตรงกับของจริง

**Blocked by:** 01

**Status:** ready-for-agent (ผู้ใช้อนุมัติ 2026-11-25 ให้หยุด `bop-api` ชั่วคราวบน Mini PC เพื่อทดสอบ fail closed แล้วเปิดกลับ)

- [x] ตรวจ `TEMPORAL_CORS_ORIGINS` ของ `temporal-ui`: ไม่ต้องแก้ dashboard ใช้ `TEMPORAL_UI_URL` เป็นลิงก์ `href` เท่านั้น (`nav-bar.tsx`, `temporal-ui.ts`) ไม่มีโค้ดฝั่งเบราว์เซอร์เรียก API ของ Temporal UI ข้าม origin และ Temporal UI เรียก API ของตัวเองที่ origin `:8233` เดียวกัน (ยืนยันจริงใน Playwright)
- [ ] หลักฐานว่า upstream ไม่เห็น header `Cookie` (เช่น route ทดลองชั่วคราวที่ไม่เข้า production หรือ log ฝั่ง upstream) ลบหลังตรวจ
- [ ] **ผู้ดำเนินการ:** หยุด `bop-api` ชั่วคราวแล้วคำขอที่ `:8233` ได้ 503 ไม่ใช่หน้า Temporal UI จากนั้นเปิดกลับและกลับมา 302/200 ตามปกติ
- [ ] **ผู้ดำเนินการ (Playwright):** หน้าที่อัปเดตสดของ Temporal UI (WebSocket/SSE) ทำงานผ่าน gateway ตรวจจาก DOM ที่เปลี่ยนเอง
- [x] เอกสารอัปเดต (README ของ bop และ `environment.md` ของ skill; compose ใน `projects/bot-orchestration-platform/` เป็นชุด dev แยกที่ไม่เกี่ยวกับ stack บน Mini PC ไม่แก้): README/เอกสารที่พูดถึงพอร์ต 8233 หรือ Temporal UI ตรงกับพฤติกรรมใหม่; spec เปลี่ยนสถานะเป็นตามจริง
