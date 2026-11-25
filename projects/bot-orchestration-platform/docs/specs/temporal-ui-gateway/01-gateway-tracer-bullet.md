# 01: Tracer bullet: gateway หน้า Temporal UI พิสูจน์ cookie ข้ามพอร์ตในเบราว์เซอร์

Spec: [temporal-ui-gateway.md](../temporal-ui-gateway.md) · ADR: [0002](../../adr/0002-temporal-ui-behind-apisix-gateway.md)

**What to build:** ผู้ใช้ที่ login BOP กดลิงก์ Temporal ใน nav-bar แล้วเห็น Temporal UI ส่วนคนที่ยังไม่ login เปิด `:8233` ตรงๆ แล้วถูกพาไปหน้า login งานนี้ตัดผ่านทุกชั้นบางๆ: service `temporal-ui-gateway` (APISIX standalone, route เดียวด้วย Lua ตรวจ `/me` + 302 + ตัด Cookie ตาม spec) ย้ายพอร์ต 8233 มาที่ gateway และพิสูจน์ในเบราว์เซอร์จริงว่า cookie `bop_session` ถูกส่งไป `:8233` ถ้าไม่ส่ง ให้หยุดและรายงาน (ADR 0002 ต้องทบทวน)

**Blocked by:** None (can start immediately)

**Status:** implemented

- [x] `docker-compose.yml` มี service `temporal-ui-gateway` (image pin เวอร์ชัน) และ config ใน `docker/temporal-ui-gateway/` (`config.yaml`, `apisix.yaml` ลงท้าย `#END`, คอมเมนต์ไทยอธิบายว่าทำไม รวมเรื่อง `proxy-rewrite` รันก่อนตรวจ)
- [x] พอร์ต 8233 เป็นของ gateway; `temporal-ui` ไม่ publish พอร์ตออกนอก network (`docker compose config` ยืนยัน)
- [x] ไม่แก้ `bop-api` และ dashboard
- [x] Bruno: request ใหม่ในโฟลเดอร์ `bruno/TemporalUiGateway/` อย่างน้อยหนึ่งเคส: ไม่มี cookie → 302 + `Location` ลงท้าย `/login` (วางหลัง logout) และ login แล้ว → 200
- [ ] **ผู้ดำเนินการ:** deploy แล้ว container ขึ้น log ของ gateway ไม่มี error `bru run --env minipc` ผ่านทั้งชุด
- [ ] **ผู้ดำเนินการ (Playwright):** login ที่ dashboard → กดลิงก์ Temporal → เห็น DOM ของ Temporal UI จริง และคำขอไป `:8233` มี cookie `bop_session`; เปิด `:8233` ตอนไม่ login ถูกพาไป `/login`
