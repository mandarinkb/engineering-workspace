# Project: Payment Service

โฟกัส consistency, idempotency, และการจัดการ transaction ข้าม service ด้วย Saga — เป็นระบบที่ผิดพลาดไม่ได้ จึงเหมาะเป็นแบบฝึกหัดเรื่อง reliability โดยเฉพาะ

## สถานะ

โครงไว้ก่อน — ยังไม่มีเนื้อหา/โค้ดจริงในรอบนี้

## แนวคิดหลัก (แผน)

- **Idempotency key** ทุก request เพื่อรองรับ retry อย่างปลอดภัย ([books/11-microservices](../../books/11-microservices/README.md))
- **Double-entry ledger** สำหรับความถูกต้องของยอดเงิน
- **Saga / Temporal workflow** สำหรับ orchestrate ขั้นตอน: reserve → charge → confirm/rollback ([books/13-temporal](../../books/13-temporal/README.md))
- **PostgreSQL transaction + isolation level** ที่เหมาะสม ([books/05-postgresql](../../books/05-postgresql/README.md))
