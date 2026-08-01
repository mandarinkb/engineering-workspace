# Volume 12 — Kafka

Distributed event streaming platform ที่เป็นกระดูกสันหลังของการสื่อสารแบบ asynchronous ระหว่าง microservices ในระบบขนาดใหญ่

## เป้าหมายการเรียนรู้

- เข้าใจสถาปัตยกรรมของ Kafka (broker, topic, partition) และผลต่อ ordering/scalability
- ออกแบบ consumer group ให้ scale การประมวลผลได้โดยไม่เสีย message
- จัดการ message ที่ประมวลผลไม่สำเร็จด้วย DLQ

## สารบัญ

### [บทที่ 1 — Broker และ Topic](01-broker-and-topic.md)

- [ ] Broker คือ node หนึ่งใน Kafka cluster, controller broker
- [ ] Replication factor, leader/follower partition, ISR (In-Sync Replica)
- [ ] Topic คือ log แบบ append-only, retention policy (by time / by size)
- [ ] Compaction (log compaction) สำหรับ topic แบบ key-value snapshot

### [บทที่ 2 — Partition และ Consumer Group](02-partition-and-consumer-group.md)

- [ ] Partition กำหนด parallelism และ ordering guarantee (ordering รับประกันแค่ภายใน partition เดียว)
- [ ] Partition key, การเลือก partition key ที่ดีเพื่อไม่ให้ data skew
- [ ] Consumer group แชร์งานกันอ่าน partition, rebalance
- [ ] Offset management — commit offset, at-least-once / at-most-once / exactly-once semantics

### [บทที่ 3 — Dead Letter Queue (DLQ)](03-dlq.md)

- [ ] เมื่อ message ประมวลผลไม่สำเร็จซ้ำๆ ควรทำอย่างไร
- [ ] ออกแบบ DLQ topic, retry topic, การ monitor และ replay จาก DLQ

## Lab ที่เกี่ยวข้อง

- [labs/kafka/](../../labs/kafka/README.md)

## แหล่งอ้างอิงแนะนำ

ดู [resources/](../../resources/README.md)
