# 05: Schedule อ่านจาก Temporal ที่เดียว (ADR 0001)

Spec: [backend-reliability.md](../backend-reliability.md)

**What to build:** รายละเอียด Schedule แสดง steps ที่จะรันจริงจาก Temporal Schedule ทุกตัว รวม Schedule ที่สร้างนอก API (เช่น example Schedule ของ worker) ไม่หายหลัง restart และ Schedule ที่ถูกลบผ่าน Temporal UI หายจาก BOP ทันที เพราะ BOP เลิกเก็บข้อมูล Schedule ซ้ำเอง ตาม [ADR 0001](../../adr/0001-temporal-is-source-of-truth-for-schedules.md)

**Blocked by:** 01 (Prefactor: test harness ของ seam A), 02 (Catch-up Window ครบทั้งเส้นทาง)

**Status:** ready-for-agent

- [ ] ขั้นแรก: ยืนยันว่าถอดรหัส steps จาก action args ของ Temporal Schedule ด้วย SDK ได้ — ถ้าไม่ได้ให้หยุดและกลับไปถามผู้ใช้ก่อนทำต่อ
- [ ] seam A: รายละเอียดของ Schedule ที่สร้างผ่าน API แสดง steps ตรงกับที่ส่งไป
- [ ] seam A: Schedule ที่สร้างตรงผ่าน Temporal client (นอก API) แสดง steps ได้
- [ ] seam A: steps ยังถูกต้องหลัง restart bop-api
- [ ] ลบ repository ของ Schedule และ implementation แบบ memory ออก
- [ ] golangci-lint ผ่าน
- [ ] Bruno: รายละเอียดของ `check-example-com-schedule` (สร้างโดย worker นอก API) มี steps
- [ ] เพิ่ม request และ test ของพฤติกรรมใหม่ลงใน Bruno collection (`bruno/`) ตามหลักใน `bruno/README.md` และแก้ test เดิมที่พฤติกรรมเปลี่ยน
- [ ] orchestrator deploy integration branch แล้ว `bru run --env minipc` ผ่านทั้งชุด (implementer ไม่ deploy และไม่รัน Bruno กับ Mini PC เอง)
