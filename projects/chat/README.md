# Project: Chat Service

โฟกัส realtime communication, WebSocket, และ pub/sub — เป็นแบบฝึกหัดเรื่อง system design ที่ต้องการ low latency และ fan-out ที่ scale ได้

## สถานะ

โครงไว้ก่อน — ยังไม่มีเนื้อหา/โค้ดจริงในรอบนี้

## แนวคิดหลัก (แผน)

- **WebSocket connection management** — จัดการ connection จำนวนมากพร้อมกัน
- **Message ordering & delivery guarantee**
- **Pub/Sub ผ่าน Redis หรือ Kafka** สำหรับ fan-out ข้าม instance ([books/06-redis](../../books/06-redis/README.md), [books/12-kafka](../../books/12-kafka/README.md))
- **Presence system** — ใครออนไลน์อยู่บ้าง
