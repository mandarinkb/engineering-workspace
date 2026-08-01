# Volume 13 — Temporal

Workflow orchestration engine ที่ช่วยจัดการ business process ที่มีหลายขั้นตอน ยาวนาน (long-running) และต้องการความน่าเชื่อถือสูง เช่น กระบวนการชำระเงิน หรือ order fulfillment ที่มีหลาย step ข้าม service

## เป้าหมายการเรียนรู้

- แยกแนวคิด Workflow กับ Activity และรู้ว่าโค้ดส่วนไหนควรอยู่ที่ไหน
- ใช้ retry policy ของ Temporal แทนการเขียน retry logic เอง
- ออกแบบ compensation logic สำหรับ rollback กระบวนการที่ทำไปครึ่งทาง

## สารบัญ

### [บทที่ 1 — Workflow และ Activity](01-workflow-and-activity.md)

- [ ] Workflow definition — deterministic code ที่ describe ลำดับขั้นตอน
- [ ] Workflow execution history, event sourcing ภายใน Temporal (เชื่อมกับแนวคิดใน [Volume 11](../11-microservices/README.md))
- [ ] Durable execution — ทำไม workflow "จำ" สถานะได้แม้ worker restart
- [ ] Activity คือหน่วยงานที่มี side effect (เรียก API, เขียน DB) แยกจาก workflow logic
- [ ] Activity timeout, heartbeat สำหรับงานที่ใช้เวลานาน

### [บทที่ 2 — Retry และ Compensation](02-retry-and-compensation.md)

- [ ] Retry policy ของ Activity — initial interval, backoff coefficient, max attempts
- [ ] ต่างจาก retry แบบเขียนเองอย่างไร (Temporal จัดการ state ให้อัตโนมัติ)
- [ ] การเขียน compensating logic เมื่อ step ก่อนหน้าสำเร็จแต่ step หลังล้มเหลว (Saga pattern ผ่าน Temporal)
- [ ] ตัวอย่าง: order + payment + inventory และการ compensate ย้อนลำดับ

## Lab ที่เกี่ยวข้อง

- [labs/temporal/](../../labs/temporal/README.md)

## แหล่งอ้างอิงแนะนำ

ดู [resources/](../../resources/README.md)
