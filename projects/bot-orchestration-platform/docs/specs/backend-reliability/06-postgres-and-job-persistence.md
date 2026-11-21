# 06: PostgreSQL และ persistence ของ Job

Spec: [backend-reliability.md](../backend-reliability.md)

**What to build:** Job อยู่รอดข้าม restart และ deploy ของ bop-api หน้า Jobs และตัวเลข Job ใน Overview ไม่กลับเป็นศูนย์อีก BOP ใช้ database `bop` และ user `bop` แยกบน PostgreSQL ตัวเดียวกับ Temporal ถ้าไม่ได้ตั้งค่าฐานข้อมูลยังทำงานแบบ memory ได้เหมือนเดิมพร้อมคำเตือน

**Blocked by:** 01 (Prefactor: test harness ของ seam A)

**Status:** ready-for-agent

- [ ] อ่าน config จาก `BOP_DATABASE_URL`; ไม่ตั้ง = repository แบบ memory พร้อม log เตือนตอน start
- [ ] bop-api รัน migration ตอน start (golang-migrate แบบฝังไฟล์, driver pgx); bop-worker ไม่รัน migration
- [ ] seam A: สร้าง Job แล้ว restart bop-api ยังอ่าน Job และสถานะได้
- [ ] repository ใหม่เป็นไปตาม interface ของ Job repository เดิม
- [ ] compose ที่ root: init script สร้าง database/user `bop` (รหัสผ่าน dev `bop` พร้อมคำเตือนห้ามใช้ production) และตั้ง `BOP_DATABASE_URL` ให้ API และ worker
- [ ] README มีคำสั่งสร้าง database/user ครั้งเดียวสำหรับ volume เดิม
- [ ] บน Mini PC: ขออนุญาตผู้ใช้ก่อนรันคำสั่งสร้าง database/user บน volume เดิม
- [ ] golangci-lint ผ่าน; deploy ขึ้น Mini PC แล้ว Job ยังอยู่หลัง recreate container bop-api
