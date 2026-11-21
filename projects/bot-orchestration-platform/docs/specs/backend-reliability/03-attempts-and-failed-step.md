# 03: Attempt และ Failed Step

Spec: [backend-reliability.md](../backend-reliability.md)

**What to build:** Step ที่พังจริงกลายเป็น Failed Step ภายในเวลาไม่นาน แทนที่จะถูกลองใหม่ไม่มีที่สิ้นสุดจน Workflow Run ค้าง running ค่าเริ่มต้น 3 Attempt พร้อม exponential backoff (เริ่ม 1 วินาที สูงสุด 1 นาที) ตั้งต่อ Step ได้ด้วย `max_attempts` ใช้กับทุก dispatch mode ส่วน Job เดี่ยวไม่เปลี่ยน

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] Step มี field `max_attempts` (optional, ≥ 1) ทั้งใน Schedule และ ad-hoc Workflow Run; ค่าที่ไม่ถูกต้องตอบ 400
- [ ] seam B: Step ที่ error ทุกครั้งกลายเป็น Failed Step หลังครบ 3 Attempt เมื่อไม่ตั้งค่า และหลังครบ `max_attempts` เมื่อตั้ง
- [ ] seam B: error ชนิด ConfigError/UnknownBot ล้มเหลวใน Attempt แรกเสมอ
- [ ] seam B: Step ถัดจาก Failed Step ไม่ถูกรัน และ Workflow Run ล้มเหลว
- [ ] ใช้กับ activity ของ Bot ปกติและของ Kubernetes Job
- [ ] `StepsEditor` ใน dashboard มีช่อง "จำนวนครั้งที่ลองสูงสุด" (ว่าง = ค่าเริ่มต้น) ใช้ได้ทั้งฟอร์ม Schedule และ Run workflow
- [ ] deploy ขึ้น Mini PC: Workflow Run ที่ยิง URL ที่ไม่มีอยู่จริงกลายเป็น failed และขึ้นในกล่อง "ล้มเหลวล่าสุด" ของ Overview
- [ ] golangci-lint, vitest และ eslint ผ่าน
