# 07: log ของ Job ลง PostgreSQL และลบอัตโนมัติเมื่อเกิน 30 วัน

Spec: [backend-reliability.md](../backend-reliability.md)

**What to build:** log ของ Job อยู่รอดข้าม restart ใช้สืบสาเหตุหลัง deploy ได้ และ log ที่เก่ากว่า 30 วันถูกลบวันละครั้งเพื่อไม่ให้ดิสก์เต็ม Job ที่เกิน 30 วันยังเห็นสถานะและ summary

**Blocked by:** 06 (PostgreSQL และ persistence ของ Job)

**Status:** done

**Verified:** deploy integration branch (commit 3247052) ขึ้น Mini PC แล้ว `bru run --env minipc` ผ่านทั้งชุด 86 request / 98 test, `bruno/run-with-restart.sh` ผ่าน (after-restart 5/5 + stream ของ log), ตรวจหน้า dashboard ด้วย Playwright — 2026-11-21

- [x] seam A: log ของ Job อ่านได้หลัง restart bop-api ทั้งแบบ REST และ stream
- [x] งานลบรันวันละครั้งใน bop-api ลบเฉพาะ log line ที่เก่ากว่า 30 วัน ไม่ลบ Job
- [x] มี test ที่พิสูจน์ว่า log เก่ากว่า 30 วันถูกลบ และ log ใหม่กว่าไม่ถูกลบ
- [x] implementation ใหม่เป็นไปตาม interface ของ logger เดิม; ไม่ตั้งฐานข้อมูล = แบบ memory เดิม
- [x] golangci-lint ผ่าน
- [x] Bruno: `after-restart` ตรวจว่า log ของ Job จากรอบก่อนยังอ่านได้ (การลบเมื่อเกิน 30 วันทดสอบด้วย Go test เพราะ Bruno ย้อนเวลาไม่ได้)
- [x] เพิ่ม request และ test ของพฤติกรรมใหม่ลงใน Bruno collection (`bruno/`) ตามหลักใน `bruno/README.md` และแก้ test เดิมที่พฤติกรรมเปลี่ยน
- [x] orchestrator deploy integration branch แล้ว `bru run --env minipc` ผ่านทั้งชุด (implementer ไม่ deploy และไม่รัน Bruno กับ Mini PC เอง)
