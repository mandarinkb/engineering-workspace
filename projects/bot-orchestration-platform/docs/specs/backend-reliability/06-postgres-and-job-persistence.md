# 06: PostgreSQL และ persistence ของ Job

Spec: [backend-reliability.md](../backend-reliability.md)

**What to build:** Job อยู่รอดข้าม restart และ deploy ของ bop-api หน้า Jobs และตัวเลข Job ใน Overview ไม่กลับเป็นศูนย์อีก BOP ใช้ database `bop` และ user `bop` แยกบน PostgreSQL ตัวเดียวกับ Temporal ถ้าไม่ได้ตั้งค่าฐานข้อมูลยังทำงานแบบ memory ได้เหมือนเดิมพร้อมคำเตือน

**Blocked by:** 01 (Prefactor: test harness ของ seam A)

**Status:** done

**Verified:** deploy integration branch (commit 3247052) ขึ้น Mini PC แล้ว `bru run --env minipc` ผ่านทั้งชุด 86 request / 98 test, `bruno/run-with-restart.sh` ผ่าน (after-restart 5/5 + stream ของ log), ตรวจหน้า dashboard ด้วย Playwright — 2026-11-21

- [x] อ่าน config จาก `BOP_DATABASE_URL`; ไม่ตั้ง = repository แบบ memory พร้อม log เตือนตอน start
- [x] bop-api รัน migration ตอน start (golang-migrate แบบฝังไฟล์, driver pgx); bop-worker ไม่รัน migration
- [x] seam A: สร้าง Job แล้ว restart bop-api ยังอ่าน Job และสถานะได้ — ยืนยันตอน orchestrator รัน `bruno/run-with-restart.sh` (ระดับ Go พิสูจน์แล้วด้วย `TestPostgresJobRepository_SurvivesReconnect`)
- [x] repository ใหม่เป็นไปตาม interface ของ Job repository เดิม
- [x] compose ที่ root: init script สร้าง database/user `bop` (รหัสผ่าน dev `bop` พร้อมคำเตือนห้ามใช้ production) และตั้ง `BOP_DATABASE_URL` ให้ API และ worker
- [x] README มีคำสั่งสร้าง database/user ครั้งเดียวสำหรับ volume เดิม
- [x] บน Mini PC: ขออนุญาตผู้ใช้ก่อนรันคำสั่งสร้าง database/user บน volume เดิม
- [x] golangci-lint ผ่าน
- [x] Bruno: โฟลเดอร์ `after-restart` ที่รันหลัง recreate container bop-api ตรวจว่า Job จากรอบก่อนยังอยู่ พร้อมสคริปต์ที่รันชุดหลัก → recreate bop-api → รัน `after-restart` (Bruno สั่ง restart เองไม่ได้)
- [x] เพิ่ม request และ test ของพฤติกรรมใหม่ลงใน Bruno collection (`bruno/`) ตามหลักใน `bruno/README.md` และแก้ test เดิมที่พฤติกรรมเปลี่ยน
- [x] orchestrator deploy integration branch แล้ว `bru run --env minipc` ผ่านทั้งชุด (implementer ไม่ deploy และไม่รัน Bruno กับ Mini PC เอง)
