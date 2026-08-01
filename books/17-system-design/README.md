# Volume 17 — System Design

เล่มสุดท้ายที่สังเคราะห์ความรู้จากทุก Volume ก่อนหน้ามาใช้ออกแบบระบบขนาดใหญ่จริง เป็นทักษะที่วัด "ความเป็น Principal Engineer" ได้ชัดที่สุด — ไม่ใช่แค่รู้เทคโนโลยี แต่ต้องรู้ trade-off และเลือกออกแบบให้เหมาะกับ requirement

## เป้าหมายการเรียนรู้

- อธิบาย CAP theorem และใช้ตัดสินใจเลือก consistency/availability ให้เหมาะกับระบบ
- ออกแบบ sharding และ replication strategy สำหรับข้อมูลขนาดใหญ่
- ออกแบบระบบให้ทนต่อความล้มเหลว (HA) และมีแผนกู้คืนภัยพิบัติ (DR)
- ประยุกต์ทุก Volume ก่อนหน้าออกแบบระบบจริง: Payment, Chat, E-commerce

## สารบัญ

### [บทที่ 1 — CAP และ Consistency](01-cap-and-consistency.md)

- [ ] Consistency, Availability, Partition Tolerance — เลือกได้ 2 ใน 3 เมื่อเกิด network partition
- [ ] ตัวอย่าง: PostgreSQL (CP-leaning), DNS (AP-leaning)
- [ ] PACELC — ขยาย CAP ด้วยการพูดถึง trade-off latency/consistency แม้ไม่มี partition

### [บทที่ 2 — Sharding, Replication และ Distributed Cache](02-scaling-data.md)

- [ ] แบ่งข้อมูลตาม key (range-based, hash-based)
- [ ] Resharding — ปัญหาเมื่อต้องเพิ่ม shard ทีหลัง, consistent hashing ช่วยอย่างไร
- [ ] ทบทวนจาก [Volume 5 — PostgreSQL](../05-postgresql/README.md) ในบริบทกว้างขึ้น: leader-follower, multi-leader, leaderless
- [ ] Cache layer ระดับระบบ (ไม่ใช่แค่ instance เดียว) — เชื่อมกับ [Volume 6 — Redis Cluster](../06-redis/README.md)
- [ ] Cache invalidation ข้าม region/instance

### [บทที่ 3 — CDN, High Availability และ Disaster Recovery](03-cdn-ha-dr.md)

- [ ] Content Delivery Network — edge caching, origin server
- [ ] ใช้ลด latency สำหรับ static asset และบางกรณี dynamic content (edge compute)
- [ ] Redundancy, failover, SPOF (Single Point of Failure) elimination
- [ ] SLA/SLO/SLI — วัดความพร้อมใช้งานอย่างไร (เชื่อมกับ [Volume 14 — Observability](../14-observability/README.md))
- [ ] RPO (Recovery Point Objective) / RTO (Recovery Time Objective)
- [ ] Backup strategy, multi-region deployment

### [บทที่ 4 — Case Study: Payment และ Chat](04-case-study-payment-chat.md)

- [ ] Idempotency, double-entry ledger, consistency สูงสุด (เชื่อมกับ [Volume 11 — Saga](../11-microservices/README.md), [Volume 13 — Temporal](../13-temporal/README.md))
- [ ] Realtime delivery (WebSocket), message ordering, presence, fan-out ที่ scale ได้

### [บทที่ 5 — Case Study: E-commerce](05-case-study-ecommerce.md)

- [ ] Catalog, inventory (race condition ตอน checkout), order, payment, notification ประกอบกันเป็นระบบเดียว — ดูรายละเอียดที่ [projects/ecommerce/](../../projects/ecommerce/README.md)

## แหล่งอ้างอิงแนะนำ

ดู [resources/](../../resources/README.md)
