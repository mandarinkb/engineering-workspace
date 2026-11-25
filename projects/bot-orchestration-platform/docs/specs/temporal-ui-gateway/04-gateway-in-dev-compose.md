# 04: gateway ใน compose ชุด dev ของโปรเจกต์

Spec: [temporal-ui-gateway.md](../temporal-ui-gateway.md) · ที่มา: code review เฟส 5

**What to build:** compose ที่ `projects/bot-orchestration-platform/docker-compose.yml` ยังเปิด Temporal UI ที่ 8233 ตรงโดยไม่ต้อง login ให้ผู้ที่เปิดชุด dev นั้นเข้า Temporal UI ได้เฉพาะผ่าน gateway เหมือน compose ที่ root (หรือถ้าตัดสินว่าไม่ใช้ชุดนั้นแล้ว ให้เอา port 8233 ออก)

**Blocked by:** 01 (ใช้ config เดียวกันจาก `docker/temporal-ui-gateway/`)

**Status:** needs-decision (ผู้ใช้ตอบรับให้เพิ่ม ticket นี้แล้ว แต่ยังไม่ได้เลือกระหว่างเพิ่ม gateway หรือเอา 8233 ออก; ไม่อยู่ในการ deploy ครั้งนี้)

- [ ] ตัดสินวิธี: เพิ่ม gateway ในชุด dev หรือเอา port 8233 ออก
- [ ] README ของ bop ตรงกับพฤติกรรมจริงของชุด dev
- [ ] **ผู้ดำเนินการ:** เปิดชุด dev แล้วเปิด `:8233` ไม่ login ไม่เห็น Temporal UI
