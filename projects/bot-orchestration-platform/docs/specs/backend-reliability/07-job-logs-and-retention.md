# 07: log ของ Job ลง PostgreSQL และลบอัตโนมัติเมื่อเกิน 30 วัน

Spec: [backend-reliability.md](../backend-reliability.md)

**What to build:** log ของ Job อยู่รอดข้าม restart ใช้สืบสาเหตุหลัง deploy ได้ และ log ที่เก่ากว่า 30 วันถูกลบวันละครั้งเพื่อไม่ให้ดิสก์เต็ม Job ที่เกิน 30 วันยังเห็นสถานะและ summary

**Blocked by:** 06 (PostgreSQL และ persistence ของ Job)

**Status:** ready-for-agent

- [ ] seam A: log ของ Job อ่านได้หลัง restart bop-api ทั้งแบบ REST และ stream
- [ ] งานลบรันวันละครั้งใน bop-api ลบเฉพาะ log line ที่เก่ากว่า 30 วัน ไม่ลบ Job
- [ ] มี test ที่พิสูจน์ว่า log เก่ากว่า 30 วันถูกลบ และ log ใหม่กว่าไม่ถูกลบ
- [ ] implementation ใหม่เป็นไปตาม interface ของ logger เดิม; ไม่ตั้งฐานข้อมูล = แบบ memory เดิม
- [ ] golangci-lint ผ่าน; deploy ขึ้น Mini PC แล้วทดสอบ
