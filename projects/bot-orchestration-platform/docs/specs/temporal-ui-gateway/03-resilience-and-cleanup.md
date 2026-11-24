# 03: ความทนทาน (fail closed, ตัด Cookie, WebSocket/SSE) และเก็บงาน

Spec: [temporal-ui-gateway.md](../temporal-ui-gateway.md)

**What to build:** ผู้ดูแลมั่นใจว่า gateway ไม่เปิดโล่งเมื่อ `bop-api` ล่ม, Temporal UI ไม่ได้รับ cookie ของผู้ใช้, หน้าที่อัปเดตสดใช้ได้ และเอกสารตรงกับของจริง

**Blocked by:** 01

**Status:** needs-decision (ต้องให้ผู้ใช้อนุมัติการหยุด `bop-api` ชั่วคราวบน Mini PC ก่อนทดสอบ fail closed)

- [ ] ตรวจ `TEMPORAL_CORS_ORIGINS` ของ `temporal-ui` ว่ายังถูกต้องเมื่อเข้าผ่าน gateway ปรับถ้าจำเป็น พร้อมเหตุผล
- [ ] หลักฐานว่า upstream ไม่เห็น header `Cookie` (เช่น route ทดลองชั่วคราวที่ไม่เข้า production หรือ log ฝั่ง upstream) ลบหลังตรวจ
- [ ] **ผู้ดำเนินการ:** หยุด `bop-api` ชั่วคราวแล้วคำขอที่ `:8233` ได้ 503 ไม่ใช่หน้า Temporal UI จากนั้นเปิดกลับและกลับมา 302/200 ตามปกติ
- [ ] **ผู้ดำเนินการ (Playwright):** หน้าที่อัปเดตสดของ Temporal UI (WebSocket/SSE) ทำงานผ่าน gateway ตรวจจาก DOM ที่เปลี่ยนเอง
- [ ] เอกสารอัปเดต: README/เอกสารที่พูดถึงพอร์ต 8233 หรือ Temporal UI ตรงกับพฤติกรรมใหม่; spec เปลี่ยนสถานะเป็นตามจริง
