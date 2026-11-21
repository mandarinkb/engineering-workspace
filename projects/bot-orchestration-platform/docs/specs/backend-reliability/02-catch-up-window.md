# 02: Catch-up Window ครบทั้งเส้นทาง

Spec: [backend-reliability.md](../backend-reliability.md)

**What to build:** Operator ตั้ง Catch-up Window ให้ Schedule ได้ตอนสร้างและแก้ไข ถ้าไม่ตั้งได้ 1 ชั่วโมง Missed Run ที่เก่ากว่านี้ถูกข้ามแทนที่จะไล่รันย้อนหลังจน Schedule ค้าง (เหตุการณ์จริง: "Buffer too large" แล้ว "Workflow panic" หลังระบบหยุด 3 เดือน) เห็นค่าในหน้ารายละเอียดและแก้ได้จากฟอร์ม Schedule ใน dashboard Backfill ทำงานเหมือนเดิม

**Blocked by:** 01 (Prefactor: test harness ของ seam A)

**Status:** ready-for-agent

- [ ] สร้าง/แก้ไข Schedule รับ `catchup_window_seconds` แบบ optional: ไม่ส่ง = 3600, น้อยกว่า 10 = 400 พร้อมข้อความที่ชัด
- [ ] รายละเอียด Schedule ส่ง `catchup_window_seconds` ที่ตั้งอยู่จริงใน Temporal Schedule กลับมา (พิสูจน์ด้วย seam A)
- [ ] แก้ไข Schedule เดิมที่ไม่เคยตั้งค่า ได้ค่า default
- [ ] example Schedule ที่ worker สร้างมี Catch-up Window 1 ชั่วโมง
- [ ] ฟอร์ม Schedule ใน dashboard มีช่อง Catch-up Window (วินาที, ค่าเริ่มต้น 3600) ทั้งสร้างและแก้ไข และหน้ารายละเอียดแสดงค่า
- [ ] Backfill ไม่เปลี่ยนพฤติกรรม
- [ ] golangci-lint, vitest และ eslint ผ่าน; deploy ขึ้น Mini PC แล้วทดสอบผ่าน browser
