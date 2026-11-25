# Spec: เปิด Temporal UI ได้เฉพาะผ่านผู้ใช้ที่ login BOP (Temporal UI gateway)

> สถานะ: implemented และ deploy บน Mini PC แล้ว (2026-11-25) · ศัพท์ตาม [GLOSSARY.md](../../GLOSSARY.md) · การตัดสินใจ: [ADR 0002](../adr/0002-temporal-ui-behind-apisix-gateway.md)

## Problem Statement

ลิงก์ Temporal ใน nav-bar ของ dashboard ชี้ไป `http://<host>:8233` ซึ่งเป็น Temporal UI โดยตรง ใครในเครือข่ายก็เปิดได้โดยไม่ต้อง login BOP จึงดู ค้นหา และสั่งแก้ Workflow/Schedule ได้โดยไม่ผ่านการควบคุมสิทธิ์ของ BOP

## Solution

วาง **Temporal UI gateway** (APISIX standalone) ไว้หน้า `temporal-ui` และให้พอร์ต 8233 เป็นของ gateway ทุกคำขอ gateway ถาม `bop-api` `GET /me` พร้อม cookie `bop_session`:

- 200 → ส่งต่อไป `temporal-ui:8080` โดยตัด header `Cookie` ออกก่อน
- ไม่ใช่ 200 → 302 ไป `http://<Host>:3000/login`
- ถาม `/me` ไม่ได้ → 503 ไม่ปล่อยผ่าน

`temporal-ui` ไม่เปิดพอร์ตออกนอก network ของ compose ลิงก์ใน nav-bar ไม่ต้องเปลี่ยน (ยังเป็นพอร์ต 8233)

## User Stories

1. ในฐานะผู้ใช้ที่ login BOP แล้ว ฉันกดลิงก์ Temporal ใน nav-bar แล้วเห็น Temporal UI ได้ทันทีโดยไม่ต้อง login ซ้ำ
2. ในฐานะคนที่ยังไม่ login ฉันเปิด `:8233` ตรงๆ แล้วถูกพาไปหน้า login ของ dashboard
3. ในฐานะคนที่มี cookie ปลอมหรือหมดอายุ ฉันถูกพาไปหน้า login เหมือนคนไม่มี cookie
4. ในฐานะผู้ดูแล ถ้า `bop-api` ล่ม ฉันต้องไม่เห็น Temporal UI เปิดโล่ง (fail closed)
5. ในฐานะผู้ดูแล ฉันต้องการให้ Temporal UI ไม่ได้รับ cookie `bop_session` ของผู้ใช้
6. ในฐานะผู้ใช้ Temporal UI ฉันใช้งานหน้าที่อัปเดตสด (WebSocket/SSE) ผ่าน gateway ได้ตามปกติ
7. ในฐานะผู้ดูแลที่อยากเรียนรู้ gateway ฉันอ่านการตั้งค่า (`apisix.yaml`) แล้วเข้าใจ พร้อมคอมเมนต์ไทยอธิบายว่าทำไม

## Implementation Decisions

- service `temporal-ui-gateway` ใน `docker-compose.yml` ที่ root image `apache/apisix` pin เวอร์ชัน (spike ใช้ `3.21.0-debian`) โหมด standalone `config_provider: yaml` ไม่มี etcd
- config อยู่ที่ `docker/temporal-ui-gateway/` (`config.yaml`, `apisix.yaml` ลงท้ายด้วย `#END`) คอมเมนต์ภาษาไทย
- route เดียว `/*`: `serverless-pre-function` (เฟส access) ใช้ `resty.http` เรียก `http://bop-api:8080/me` พร้อม Cookie ของคำขอ, timeout 2 วินาที; 200 ผ่าน, ไม่ใช่ 200 → `Location: http://<$host>:3000/login` + 302; เรียกไม่สำเร็จ → 503
- ตัด Cookie ด้วย `serverless-post-function` (เฟส access) ห้ามใช้ `proxy-rewrite` เพราะรันก่อนตรวจและทำให้ทุกคนไม่ผ่าน
- พอร์ต 8233 map ไป gateway (container port 9080) ย้ายออกจาก `temporal-ui` ซึ่งเหลือ `expose` ภายใน network
- `TEMPORAL_CORS_ORIGINS` ของ `temporal-ui` ทบทวนว่ายังใช้ได้เมื่อ origin เป็น gateway
- gateway ตั้ง `restart: on-failure` ใน compose ตาม `bop-api` (ผลจาก code review: ทางเข้าเดียวไม่ควรพึ่ง override นอก repo) override ของ LAN (`~/Workspace/bop-lan.override.yml`) ยังตั้งเป็น `unless-stopped` ได้ถ้าต้องการ โดยแสดง diff ขออนุมัติก่อน
- bop-api ตอบ 401 = ยังไม่ login → 302; ตอบอย่างอื่นที่ไม่ใช่ 200 หรือถามไม่ได้ → 503 และบันทึก error log
- ไม่แก้ `bop-api` และ dashboard

## Testing Decisions

seam ที่ผู้ใช้ยืนยันแล้ว (สองชั้น):

1. **Bruno black-box กับ `:8233`** (โฟลเดอร์ใหม่ `bruno/TemporalUiGateway/`): ไม่มี cookie → 302 + `Location` ลงท้าย `/login`; cookie ปลอม → 302; login แล้ว → 200; request "ไม่มี cookie" วางหลัง logout เพราะ cookie jar ของ Bruno CLI แนบ cookie ให้ทุก request
2. **Playwright ในเบราว์เซอร์จริง**: login ที่ dashboard แล้วกดลิงก์ Temporal ใน nav-bar เห็น Temporal UI; เปิด `:8233` ตอนยังไม่ login ถูกพาไป login; ตรวจว่าเบราว์เซอร์ส่ง `bop_session` ไป `:8233` จริง (ข้อที่ยังไม่พิสูจน์ใน ADR); หน้าที่อัปเดตสดทำงาน
3. ตรวจ fail closed ด้วยมือบน Mini PC: หยุด `bop-api` ชั่วคราว (ขออนุมัติก่อน) แล้วต้องได้ 503; ตัด Cookie ตรวจด้วยบันทึกฝั่ง upstream หรือ route ทดลองที่ไม่เข้า production
4. เนื้อหาที่ได้ต้องเป็น Temporal UI จริง (ตรวจจาก DOM ในเบราว์เซอร์ ไม่ใช่การนับคำใน response ที่ถูกบีบอัด)

## Out of Scope

- พอร์ต gRPC 7233 ของ Temporal ที่เปิดตรง
- ย้าย `bop-api`/dashboard มาอยู่หลัง APISIX
- สิทธิ์แยกตามบทบาท (ใครเข้าได้ = ผู้ที่ login BOP ทุกคน)
- HTTPS/TLS, rate limit, การดู metrics ของ APISIX, Admin API/Dashboard ของ APISIX
- การเปลี่ยนลิงก์ใน nav-bar

## Further Notes

- ความเสี่ยงหลัก: cookie `bop_session` เป็น host-only ไม่ผูกพอร์ต ตามสเปกเบราว์เซอร์ควรถูกส่งไป `:8233` แต่ยังไม่เคยพิสูจน์ ticket แรกจึงเป็น tracer bullet ที่พิสูจน์เรื่องนี้ ถ้าไม่ส่ง ต้องกลับไปทบทวน ADR 0002
- ผล spike: branch `prototype/apisix-temporal-ui-gateway`

## ต้นทุนการทำงาน

tickets=4 · subagents=2 (implementer 1, code-review 1; spike ก่อนหน้า 3 รอบรันเอง) · รอบแก้=1 (แก้ 6 ข้อ) · verify.sh ไม่ได้ใช้ (ไม่มีโค้ด Go/TS) · หนักสุด: เฟส 7 Playwright รัน 4 รอบ (ล้ม 3 รอบเพราะสคริปต์เอง) และ spike 3 รอบ
