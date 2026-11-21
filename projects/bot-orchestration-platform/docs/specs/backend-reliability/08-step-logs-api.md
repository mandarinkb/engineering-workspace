# 08: log ของ Step อ่านผ่าน API ได้

Spec: [backend-reliability.md](../backend-reliability.md)

**What to build:** Operator อ่าน log ของแต่ละ Step ได้ตาม Workflow Run และลำดับ Step โดยไม่ต้องรู้ Activity ID ของ Temporal เห็นทุก Attempt เรียงตามเวลาพร้อมระบุ Attempt ของแต่ละบรรทัด (ปัจจุบัน log ของ Step อยู่ใน memory ของ bop-worker จึงอ่านผ่าน API ไม่ได้เลย)

**Blocked by:** 03 (Attempt และ Failed Step), 07 (log ของ Job ลง PostgreSQL และลบอัตโนมัติเมื่อเกิน 30 วัน)

**Status:** ready-for-agent

- [ ] bop-worker เขียน log ของ Step ลง PostgreSQL ผูกกับ workflow ID + ลำดับ Step + หมายเลข Attempt
- [ ] `GET /workflow-runs/{id}/steps/{index}/logs` คืน log ทุก Attempt เรียงตามเวลา แต่ละบรรทัดมี `attempt`
- [ ] ลำดับ Step ที่ไม่มีจริง = 404
- [ ] seam A: Step ที่ล้มเหลว 3 Attempt มี log ของทั้ง 3 Attempt
- [ ] log ของ Step อยู่ภายใต้กฎลบเมื่อเกิน 30 วันเดียวกัน
- [ ] golangci-lint ผ่าน; deploy ขึ้น Mini PC แล้วเรียก endpoint ได้จริง
