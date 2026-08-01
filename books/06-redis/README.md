# Volume 6 — Redis

In-memory data store ที่ใช้แก้ปัญหา performance และ concurrency หลายรูปแบบ ตั้งแต่ caching ธรรมดา ไปจนถึง pub/sub messaging และ stream processing ระดับเบา

## เป้าหมายการเรียนรู้

- ใช้ Redis เป็น cache layer ได้อย่างถูกต้อง พร้อมกลยุทธ์ invalidation ที่เหมาะสม
- เข้าใจ TTL และผลกระทบต่อ memory/consistency
- ใช้ Pub/Sub และ Streams แก้ปัญหา messaging เบื้องต้น
- เข้าใจแนวคิด Redis Cluster สำหรับ scale แนวนอน

## สารบัญ

### 1. Cache

- [ ] Cache pattern: cache-aside, write-through, write-behind
- [ ] Cache invalidation strategy — ปัญหาที่ยากที่สุดใน computer science (การตั้งชื่อ + cache invalidation)
- [ ] Cache stampede / thundering herd และวิธีป้องกัน (lock, jitter)

### 2. TTL

- [ ] `EXPIRE`, `TTL`, `PERSIST`
- [ ] Eviction policy เมื่อ memory เต็ม (`noeviction`, `allkeys-lru`, `volatile-lru` ฯลฯ)

### 3. Pub/Sub

- [ ] `PUBLISH`/`SUBSCRIBE` ทำงานอย่างไร
- [ ] ข้อจำกัด: ไม่มีการเก็บ message (fire-and-forget), ต่างจาก Kafka อย่างไร (เชื่อมกับ [Volume 12 — Kafka](../12-kafka/README.md))

### 4. Streams

- [ ] `XADD`, `XREAD`, `XRANGE`, consumer group (`XGROUP`, `XREADGROUP`)
- [ ] เปรียบเทียบกับ Pub/Sub — Streams เก็บ log ได้ ใช้ replay ได้

### 5. Cluster

- [ ] Redis Cluster — hash slot (16384 slots), sharding แบบ built-in
- [ ] High availability: Sentinel vs Cluster
- [ ] ข้อจำกัดของ multi-key operation ใน cluster mode

## Lab / Source Code ที่เกี่ยวข้อง

- [source-code/golang/](../../source-code/golang/README.md) — ตัวอย่างการเชื่อมต่อ Redis จาก Go

## แหล่งอ้างอิงแนะนำ

ดู [resources/](../../resources/README.md)
