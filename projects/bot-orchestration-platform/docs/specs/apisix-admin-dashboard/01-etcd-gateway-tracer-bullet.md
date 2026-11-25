# 01: Tracer bullet: gateway โหมด etcd + Admin API ที่พอร์ตชั่วคราว

Spec: [apisix-admin-dashboard.md](../apisix-admin-dashboard.md) · ADR: [0003](../../adr/0003-apisix-gateway-with-etcd-and-dashboard.md)

**What to build:** ยก gateway โหมด traditional (`config_provider: etcd`) พร้อม etcd, Admin API และ dashboard ในตัว **ข้างๆ** gateway โหมดไฟล์เดิม ที่พอร์ตชั่วคราว `:18233` ด้วย route เดียวกัน (ตรวจ `/me`, 302, 503, ตัด Cookie) route เก็บเป็น JSON ของ Admin API ใน repo และเขียนลง etcd ด้วย `seed.sh` ที่ตรวจ 302 ท้ายสุด ticket นี้ตอบข้อ "ยังไม่พิสูจน์" ของ ADR 0003 ทุกข้อที่ไม่เกี่ยวกับ etcd ล่ม: tag ของ etcd, การใส่ admin key จาก env, `allow_admin` เมื่อ publish ผ่าน Docker, Lua เก็บเป็น JSON ใช้ได้ ถ้าข้อใดพิสูจน์ไม่ได้ ให้หยุดและรายงาน (ADR 0003 ต้องทบทวน) gateway เดิมที่ `:8233` และชุด dev **ต้องไม่เปลี่ยน**

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent (ผู้ช่วยหลักทำเองเพราะมีความไม่แน่นอนสูงและต้องทดสอบบน Mini PC)

- [ ] service `etcd` ใน `docker-compose.yml` ที่ root: image pin เวอร์ชันที่ตรวจ tag แล้วด้วย `docker buildx imagetools inspect`, named volume, ไม่ publish พอร์ต
- [ ] `docker/temporal-ui-gateway/config-etcd.yaml`: `role: traditional`, `config_provider: etcd`, `enable_admin_ui: true`, `admin_key_required: true`, `allow_admin` แคบ, admin key มาจาก env (`APISIX_ADMIN_KEY`) ไม่มีค่าจริงใน repo; service ต้องไม่ start ถ้าไม่ตั้ง key (วิธีออกแบบโดยผู้ทำ ห้ามทำให้ `up` ของ service อื่นล้มเมื่อไม่ได้ตั้งตัวแปร เช่น อย่าใช้ `${VAR:?}` ระดับไฟล์ ถ้าทำให้ล้ม ให้ตรวจแล้วเลือกวิธีอื่น)
- [ ] service gateway ใหม่ (เช่น `temporal-ui-gateway-next`) publish `18233:9080` และ `127.0.0.1:9180:9180` เท่านั้น; คอมเมนต์ไทยอธิบายเหตุผลทุกจุดที่ไม่ชัด
- [ ] `docker/temporal-ui-gateway/routes/temporal-ui.json`: route เดียวกับ `apisix.yaml` ในรูป Admin API (Lua เนื้อเดียวกัน)
- [ ] `docker/temporal-ui-gateway/seed.sh`: `PUT /apisix/admin/routes/{id}` ต่อไฟล์ (idempotent) พิมพ์ drift ของ route ที่ไม่ได้มาจาก git แล้วตรวจ `GET :<พอร์ต>/` ไม่มี cookie ได้ **302** ถ้าไม่ใช่ exit ≠ 0 (ผ่านเมื่อเห็นบรรทัดยืนยัน 302 ไม่ใช่ดู exit code อย่างเดียว)
- [ ] Bruno: ไม่ต้องเพิ่มโฟลเดอร์ใหม่ ใช้โฟลเดอร์ `TemporalUiGateway` เดิมกับ `gatewayUrl` ที่ override เป็น `:18233` (ผู้ทำตรวจว่า Bruno CLI รองรับ `--env-var` และบันทึกคำสั่งใน README ของ bruno)
- [ ] **ผู้ดำเนินการ (Mini PC):** ตรวจพอร์ต 9180 และ 18233 ว่าง, ขออนุมัติแผน, ตั้ง `APISIX_ADMIN_KEY` นอก repo, ยกขึ้น, รัน `seed.sh` ผ่าน (302)
- [ ] **ผู้ดำเนินการ:** `bru run TemporalUiGateway --env minipc --env-var gatewayUrl=http://192.168.1.46:18233` ผ่านทั้งโฟลเดอร์ และ Playwright ชุดเดิมผ่านที่ `:18233`
- [ ] **ผู้ดำเนินการ:** `/ui` ของ dashboard ตอบ 200 ผ่าน tunnel และ gateway เดิม `:8233` ยังตอบ 302 ไม่ถูก restart
