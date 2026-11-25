# เปิด Temporal UI ได้เฉพาะผ่าน gateway ที่ตรวจ session ของ BOP

> **เลิกใช้แล้ว ([ADR 0004](0004-remove-apisix-gateway.md)):** gateway ถูกถอดออกจากระบบ Temporal UI เปิดตรงที่พอร์ต 8233 โค้ดเดิมเก็บไว้ที่ git tag `archive/apisix-gateway` เอกสารนี้เก็บไว้เป็นประวัติการตัดสินใจ

> **ถูกแทนที่บางส่วนโดย [ADR 0003](0003-apisix-gateway-with-etcd-and-dashboard.md):** ส่วน "APISIX standalone โหมดไฟล์ ไม่มี etcd/Admin API" เปลี่ยนเป็นโหมด etcd + dashboard ส่วนที่เหลือ (ตรวจ `/me`, redirect, ตัด Cookie, fail closed) ยังใช้ตามเดิม

Temporal UI (พอร์ต 8233) เดิมเปิดได้ทุกคนในเครือข่ายโดยไม่ต้อง login เราวาง **Temporal UI gateway** ด้วย APISIX ไว้หน้า `temporal-ui` ให้ gateway ถาม `bop-api` `GET /me` พร้อม cookie `bop_session` ทุกคำขอ: ผ่านเมื่อได้ 200, ไม่ผ่านพาไป `http://<host>:3000/login` และถ้าถาม `/me` ไม่ได้ให้ปิด (ไม่ปล่อยผ่าน) ลิงก์ใน nav-bar ของ dashboard ยังชี้พอร์ต 8233 เดิม

เลือก APISIX ทั้งที่ nginx/Traefik ก็ทำได้ เพราะผู้ใช้ต้องการเรียนรู้ API gateway และเป็นก้าวแรกที่ภายหลังอาจย้าย `bop-api`/dashboard มาอยู่หลัง gateway เดียวกัน

## Considered Options

- **ให้ bop-api เป็น reverse proxy ของ Temporal UI เอง**: ไม่เพิ่ม service แต่ผูกโค้ด proxy (รวม WebSocket/asset) เข้ากับ API และไม่ได้เรียนรู้ gateway
- **ให้ dashboard (Next.js middleware) proxy**: ต้องส่งต่อทุก path และ asset ของ Temporal UI ผ่าน Next.js
- **APISIX `forward-auth` ล้วน (declarative)**: เป็นวิธีมาตรฐานของ gateway แต่ผู้ใช้ที่ไม่ login ได้หน้า 401 ไม่ถูกพาไป login (`forward-auth` ตอบได้แค่สิ่งที่บริการตรวจสิทธิ์ส่งกลับ)
- **APISIX + Lua `serverless-pre-function` (ที่เลือก)**: ตรวจ `/me` และ redirect เอง แลกกับ Lua ~12 บรรทัดที่ต้องดูแล

## Consequences

- ต้องแก้ port: 8233 ย้ายจาก `temporal-ui` ไปเป็นของ gateway และ `temporal-ui` ไม่เปิดพอร์ตออกนอก network ของ compose
- พอร์ต gRPC 7233 ของ Temporal ยังเปิดตรง อยู่นอกขอบเขตงานนี้
- **พิสูจน์แล้ว (2026-11-25, Playwright ใน Chrome):** เบราว์เซอร์ส่ง `bop_session` (host-only, ไม่ผูกพอร์ต) ไป `:8233` จริง แนวทางนี้ใช้ได้
- ต้องทดสอบ WebSocket/SSE ของ Temporal UI ผ่าน gateway

## สิ่งที่ได้เรียนรู้จาก APISIX

ผลที่ได้จาก spike (APISIX standalone ทดลองบน Mini PC 2026-11-25) สรุปไว้ด้านล่างทั้งหมด branch ทดลอง `prototype/apisix-temporal-ui-gateway` ถูกลบแล้วโดยไม่ได้ push สคริปต์จึงไม่เหลือ

- ใช้โหมด standalone (`config_provider: yaml`) ไม่ต้องมี etcd ไฟล์ `apisix.yaml` ต้องลงท้ายด้วย `#END` ใช้ RAM ราว 110 MiB
- **ลำดับ plugin ตามเฟส ไม่ใช่ตามลำดับที่เขียน:** `proxy-rewrite` (เฟส rewrite) รันก่อน `forward-auth` (เฟส access) การลบ Cookie ด้วย `proxy-rewrite` จึงทำให้ตัวตรวจไม่เห็น cookie และทุกคนได้ 401 ต้องตัด Cookie ด้วย `serverless-post-function` ที่เฟส access หลังตรวจแล้ว (ยืนยันว่า upstream เห็น Cookie 0 บรรทัด)
- plugin ที่หยุดคำขอในเฟส access ตอบเองได้ (เลือก status/header เช่น 302) ส่วน `forward-auth` ตอบได้เท่าที่ auth service ส่งกลับ
- fail closed: auth service ติดต่อไม่ได้ → `forward-auth` ตอบ 403 (`allow_degradation: false`) ส่วนทาง Lua ตอบ 503 เอง
