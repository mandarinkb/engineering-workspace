# 03: ความทนทาน (fail closed, ตัด Cookie, WebSocket/SSE) และเก็บงาน

Spec: [temporal-ui-gateway.md](../temporal-ui-gateway.md)

**What to build:** ผู้ดูแลมั่นใจว่า gateway ไม่เปิดโล่งเมื่อ `bop-api` ล่ม, Temporal UI ไม่ได้รับ cookie ของผู้ใช้, หน้าที่อัปเดตสดใช้ได้ และเอกสารตรงกับของจริง

**Blocked by:** 01

**Status:** done ยกเว้นหนึ่งเกณฑ์ (หลักฐานว่า Cookie ถูกตัดบน stack จริง ผู้ใช้ตกลงรับหลักฐานจาก spike)

**Verified:** deploy commit c7ec188 บน Mini PC, 2026-11-25 — fail closed: หยุด `bop-api` ชั่วคราวแล้ว `:8233` ได้ 503 ทั้งไม่มี cookie และ cookie ปลอม log ระบุ timeout เปิดกลับแล้ว `/readyz` 200 และ `:8233` กลับเป็น 302; Playwright เปิดหน้า workflows และ schedules ค้างหน้าละ 6 วินาที ไม่พบ WebSocket ใช้ fetch polling 17 คำขอ ทุกคำขอ <300 ยกเว้น `schedule-count` ได้ 501 ซึ่งเป็นของ upstream เอง (เรียก `temporal-ui` ตรงผ่าน IP ของ container ก็ได้ 501 เหมือนกัน ไม่เกี่ยวกับ gateway)

- [x] ตรวจ `TEMPORAL_CORS_ORIGINS` ของ `temporal-ui`: ไม่ต้องแก้ dashboard ใช้ `TEMPORAL_UI_URL` เป็นลิงก์ `href` เท่านั้น (`nav-bar.tsx`, `temporal-ui.ts`) ไม่มีโค้ดฝั่งเบราว์เซอร์เรียก API ของ Temporal UI ข้าม origin และ Temporal UI เรียก API ของตัวเองที่ origin `:8233` เดียวกัน (ยังต้องยืนยันจริงใน Playwright ตอนเฟส 7 เกณฑ์ข้างล่าง)
- [ ] (ไม่ทำบน stack จริง: ผู้ใช้รับหลักฐานจาก spike ที่ใช้ route แบบเดียวกัน — upstream เห็น Cookie 0 บรรทัด เทียบกับ route ควบคุม 1 บรรทัด) หลักฐานว่า upstream ไม่เห็น header `Cookie` (เช่น route ทดลองชั่วคราวที่ไม่เข้า production หรือ log ฝั่ง upstream) ลบหลังตรวจ
- [x] **ผู้ดำเนินการ:** หยุด `bop-api` ชั่วคราวแล้วคำขอที่ `:8233` ได้ 503 ไม่ใช่หน้า Temporal UI จากนั้นเปิดกลับและกลับมา 302/200 ตามปกติ
- [x] **ผู้ดำเนินการ (Playwright):** หน้าที่อัปเดตสดของ Temporal UI (WebSocket/SSE) ทำงานผ่าน gateway ตรวจจาก DOM ที่เปลี่ยนเอง
- [x] เอกสารอัปเดต (README ของ bop และ `environment.md` ของ skill; compose ใน `projects/bot-orchestration-platform/` เป็นชุด dev แยกที่ไม่เกี่ยวกับ stack บน Mini PC ไม่แก้): README/เอกสารที่พูดถึงพอร์ต 8233 หรือ Temporal UI ตรงกับพฤติกรรมใหม่; spec เปลี่ยนสถานะเป็นตามจริง
