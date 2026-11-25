# 05: ย้ายจริง: สลับ :8233 ไปที่ gateway โหมด etcd

Spec: [apisix-admin-dashboard.md](../apisix-admin-dashboard.md) · ADR 0003 กลไกข้อ 5

**What to build:** สลับพอร์ต `8233` จาก gateway โหมดไฟล์ไปที่ gateway โหมด etcd และเลิกใช้ gateway เดิมใน compose ที่ root ผู้ใช้ Temporal UI ต้องไม่เห็นความต่าง (302 เมื่อไม่ login, 503 เมื่อถาม `/me` ไม่ได้, เข้าได้เมื่อ login) ชุด dev ไม่เปลี่ยน

**Blocked by:** 01, 02, 03, 04

**Status:** implemented (โค้ดและเอกสารแก้แล้ว ยังไม่ deploy; ต้องได้รับอนุมัติแผน deploy: recreate gateway ตัวจริง Temporal UI ดับหลายวินาที และ `bop-lan.override.yml` ไม่ต้องแก้เพิ่ม เพราะมี `temporal-ui-gateway: restart: unless-stopped` อยู่แล้ว)

- [x] `docker-compose.yml` ที่ root: gateway โหมดไฟล์เดิมถูกแทนที่ด้วยตัวโหมด etcd ที่ publish `8233:9080` (ไม่ใช่ `18233`) ชื่อ service ที่ override ใน `bop-lan.override.yml` ตรงกัน (`restart: unless-stopped`) และ Admin API ยัง bind `127.0.0.1:9180` เท่านั้น
- [x] ไม่ลบ `config.yaml`/`apisix.yaml` ของโหมดไฟล์ เพราะชุด dev ยังใช้
- [x] เอกสารอัปเดต: ADR 0002 (หมายเหตุ), `README` ของ bop และ `environment.md` ของ skill `grill-to-deploy` ที่พูดถึงพอร์ต/วิธีรัน gateway; spec ต่างๆ เปลี่ยนสถานะตามจริง
- [ ] **ผู้ดำเนินการ:** แผน deploy ได้รับอนุมัติ, snapshot ก่อนสลับ, `seed.sh` ผ่าน (302), gateway เดิมถูกแทนที่แล้วโดยไม่แตะ container อื่น
- [ ] **ผู้ดำเนินการ:** regression ทั้งชุดที่ `:8233`: `bru run --env minipc` ผ่านทั้งชุด, Playwright ชุดเดิม 9 ข้อผ่าน, fail closed (หยุด `bop-api` ชั่วคราว ได้ 503) และตรวจว่า Admin API ต่อไม่ได้จาก Mac
- [ ] **ผู้ดำเนินการ:** ย้อนกลับได้จริง: บันทึกคำสั่งย้อน (`git switch` ไป commit ก่อนหน้า + `up -d`) ไว้ใน ticket และทดสอบอย่างน้อยด้วย dry-run
- [ ] **ผู้ดำเนินการ:** บันทึกต้นทุนการทำงานท้ายไฟล์ spec ตามเฟส 8 ของ `grill-to-deploy`
