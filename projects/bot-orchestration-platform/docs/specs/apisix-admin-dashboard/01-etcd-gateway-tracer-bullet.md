# 01: Tracer bullet: gateway โหมด etcd + Admin API ที่พอร์ตชั่วคราว

Spec: [apisix-admin-dashboard.md](../apisix-admin-dashboard.md) · ADR: [0003](../../adr/0003-apisix-gateway-with-etcd-and-dashboard.md)

**What to build:** ยก gateway โหมด traditional (`config_provider: etcd`) พร้อม etcd, Admin API และ dashboard ในตัว **ข้างๆ** gateway โหมดไฟล์เดิม ที่พอร์ตชั่วคราว `:18233` ด้วย route เดียวกัน (ตรวจ `/me`, 302, 503, ตัด Cookie) route เก็บเป็น JSON ของ Admin API ใน repo และเขียนลง etcd ด้วย `seed.sh` ที่ตรวจ 302 ท้ายสุด ticket นี้ตอบข้อ "ยังไม่พิสูจน์" ของ ADR 0003 ทุกข้อที่ไม่เกี่ยวกับ etcd ล่ม: tag ของ etcd, การใส่ admin key จาก env, `allow_admin` เมื่อ publish ผ่าน Docker, Lua เก็บเป็น JSON ใช้ได้ ถ้าข้อใดพิสูจน์ไม่ได้ ให้หยุดและรายงาน (ADR 0003 ต้องทบทวน) gateway เดิมที่ `:8233` และชุด dev **ต้องไม่เปลี่ยน**

**Blocked by:** None (can start immediately)

**Status:** done

**Verified:** deploy commit 7d3bc4a บน Mini PC (branch `feat/apisix-etcd-gateway`), 2026-11-25 — dry-run แสดงว่าสร้างแค่ `etcd` กับ `temporal-ui-gateway-next` (container เดิมทั้งหมดแค่ Running); `seed.sh` เขียน route HTTP 201 และพิมพ์ "ยืนยัน: 302" exit 0; ทางล้มของ seed: ชี้ไปที่ที่ตอบ 401 → exit 1, ไม่ตั้ง key → exit 2; `bru run TemporalUiGateway --env-var gatewayUrl=http://192.168.1.46:18233` 5 request / 6 test ผ่าน (ยืนยันจาก reporter JSON ว่ายิง `:18233` จริง ไม่ใช่ `:8233`); Playwright ที่ `:18233` ผ่าน 4 ข้อ (ไม่ login ถูกพาไป login, login แล้วเข้าได้, title "Workflows | default", `/api/` 8 คำขอ 2xx); `/ui` ผ่าน `ssh -L` ตอบ 200; Admin API: ไม่ใส่ key 401, key ผิด 401, key ถูก 200, จาก Mac ที่ `192.168.1.46:9180` ต่อไม่ได้; key ไม่ปรากฏใน log ของ gateway; gateway เดิม `:8233` ยัง 302 และไม่ถูก restart (Up 2 ชั่วโมง) **ยังไม่ได้ตรวจ:** หน้า dashboard ใช้งานจริง (ticket 02), พฤติกรรมเมื่อ etcd ล่ม (ticket 04)

- [x] service `etcd` ใน `docker-compose.yml` ที่ root: image pin เวอร์ชันที่ตรวจ tag แล้วด้วย `docker buildx imagetools inspect`, named volume, ไม่ publish พอร์ต
- [x] `docker/temporal-ui-gateway/config-etcd.yaml`: `role: traditional`, `config_provider: etcd`, `enable_admin_ui: true`, `admin_key_required: true`, `allow_admin` แคบ, admin key มาจาก env (`APISIX_ADMIN_KEY`) ไม่มีค่าจริงใน repo; service ต้องไม่ start ถ้าไม่ตั้ง key (วิธีออกแบบโดยผู้ทำ ห้ามทำให้ `up` ของ service อื่นล้มเมื่อไม่ได้ตั้งตัวแปร เช่น อย่าใช้ `${VAR:?}` ระดับไฟล์ ถ้าทำให้ล้ม ให้ตรวจแล้วเลือกวิธีอื่น)
- [x] service gateway ใหม่ (เช่น `temporal-ui-gateway-next`) publish `18233:9080` และ `127.0.0.1:9180:9180` เท่านั้น; คอมเมนต์ไทยอธิบายเหตุผลทุกจุดที่ไม่ชัด
- [x] `docker/temporal-ui-gateway/routes/temporal-ui.json`: route เดียวกับ `apisix.yaml` ในรูป Admin API (Lua เนื้อเดียวกัน)
- [x] `docker/temporal-ui-gateway/seed.sh`: `PUT /apisix/admin/routes/{id}` ต่อไฟล์ (idempotent) พิมพ์ drift ของ route ที่ไม่ได้มาจาก git แล้วตรวจ `GET :<พอร์ต>/` ไม่มี cookie ได้ **302** ถ้าไม่ใช่ exit ≠ 0 (ผ่านเมื่อเห็นบรรทัดยืนยัน 302 ไม่ใช่ดู exit code อย่างเดียว)
- [x] Bruno: ไม่ต้องเพิ่มโฟลเดอร์ใหม่ ใช้โฟลเดอร์ `TemporalUiGateway` เดิมกับ `gatewayUrl` ที่ override เป็น `:18233` (ผู้ทำตรวจว่า Bruno CLI รองรับ `--env-var` และบันทึกคำสั่งใน README ของ bruno)
- [x] **ผู้ดำเนินการ (Mini PC):** ตรวจพอร์ต 9180 และ 18233 ว่าง, ขออนุมัติแผน, ตั้ง `APISIX_ADMIN_KEY` นอก repo, ยกขึ้น, รัน `seed.sh` ผ่าน (302)
- [x] **ผู้ดำเนินการ:** `bru run TemporalUiGateway --env minipc --env-var gatewayUrl=http://192.168.1.46:18233` ผ่านทั้งโฟลเดอร์ และ Playwright ชุดเดิมผ่านที่ `:18233`
- [x] **ผู้ดำเนินการ:** `/ui` ของ dashboard ตอบ 200 ผ่าน tunnel และ gateway เดิม `:8233` ยังตอบ 302 ไม่ถูก restart
