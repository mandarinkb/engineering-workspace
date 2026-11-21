# 08: log ของ Step อ่านผ่าน API ได้

Spec: [backend-reliability.md](../backend-reliability.md)

**What to build:** Operator อ่าน log ของแต่ละ Step ได้ตาม Workflow Run และลำดับ Step โดยไม่ต้องรู้ Activity ID ของ Temporal เห็นทุก Attempt เรียงตามเวลาพร้อมระบุ Attempt ของแต่ละบรรทัด (ปัจจุบัน log ของ Step อยู่ใน memory ของ bop-worker จึงอ่านผ่าน API ไม่ได้เลย)

**Blocked by:** 03 (Attempt และ Failed Step), 07 (log ของ Job ลง PostgreSQL และลบอัตโนมัติเมื่อเกิน 30 วัน)

**Status:** implemented

- [x] bop-worker เขียน log ของ Step ลง PostgreSQL ผูกกับ workflow ID + ลำดับ Step + หมายเลข Attempt
- [x] `GET /workflow-runs/{id}/steps/{index}/logs` คืน log ทุก Attempt เรียงตามเวลา แต่ละบรรทัดมี `attempt`
- [x] ลำดับ Step ที่ไม่มีจริง = 404
- [x] seam A: Step ที่ล้มเหลว 3 Attempt มี log ของทั้ง 3 Attempt
- [x] log ของ Step อยู่ภายใต้กฎลบเมื่อเกิน 30 วันเดียวกัน
- [x] golangci-lint ผ่าน
- [ ] Bruno: log ของ Step ที่ล้มเหลวมีครบ 3 Attempt, ลำดับ Step ที่ไม่มีจริงได้ 404
- [x] เพิ่ม request และ test ของพฤติกรรมใหม่ลงใน Bruno collection (`bruno/`) ตามหลักใน `bruno/README.md` และแก้ test เดิมที่พฤติกรรมเปลี่ยน
- [ ] orchestrator deploy integration branch แล้ว `bru run --env minipc` ผ่านทั้งชุด (implementer ไม่ deploy และไม่รัน Bruno กับ Mini PC เอง)
