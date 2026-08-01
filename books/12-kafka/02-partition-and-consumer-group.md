# บทที่ 2: Partition และ Consumer Group

## 2.1 Partition

### Partition คือหน่วยของ Parallelism

Topic หนึ่งตัวไม่ได้เก็บอยู่เป็นก้อนเดียว แต่ถูกแบ่งออกเป็นหลาย **partition** — แต่ละ partition คือ append-only log อิสระของตัวเอง (ตามที่อธิบายใน [บทที่ 1](01-broker-and-topic.md)) กระจายอยู่บน broker ต่างๆ กันในคลัสเตอร์:

```
Topic: "orders" (3 partitions)

Partition 0 (Broker 1): [M0][M3][M6][M9]...
Partition 1 (Broker 2): [M1][M4][M7]...
Partition 2 (Broker 3): [M2][M5][M8]...
```

เหตุผลที่ต้องมีหลาย partition คือ **parallelism** — ถ้า topic มีแค่ partition เดียว ต่อให้มี consumer หลายตัวก็มีแค่ตัวเดียวที่อ่านได้จริงในแต่ละช่วงเวลา (ทุกฝ่ายต้องคิวรอ) แต่ถ้าแบ่งเป็นหลาย partition consumer หลายตัวสามารถอ่านคนละ partition **พร้อมกันจริง** ได้ ทำให้ throughput ของทั้งระบบ scale ขึ้นตามจำนวน partition (จนถึงขีดจำกัดของจำนวน partition ที่มี)

### Ordering รับประกันแค่ภายใน Partition เดียวเท่านั้น

นี่คือข้อเท็จจริงที่สำคัญที่สุดของ Kafka ที่พลาดไม่ได้: **Kafka รับประกันลำดับของ message (ordering) เฉพาะภายใน partition เดียวกันเท่านั้น ไม่รับประกันลำดับข้าม partition ของ topic เดียวกัน**

```
Topic "orders" มี 3 partition:

Partition 0: [OrderCreated #1001][OrderPaid #1001][OrderShipped #1001]
             ← ลำดับนี้การันตีถูกต้องเสมอ เพราะอยู่ partition เดียวกัน

Partition 1: [OrderCreated #1002]...
Partition 2: [OrderCreated #1003]...

ไม่มีการรับประกันว่า message ใน Partition 0 มาก่อนหรือหลัง message ใน Partition 1/2
```

ทำไมเรื่องนี้ถึงสำคัญมากในทางปฏิบัติ: สมมติระบบส่ง event `OrderCreated`, `OrderPaid`, `OrderShipped` ของ order เดียวกัน (order #1001) ถ้า event เหล่านี้กระจายไปคนละ partition (เช่น เพราะไม่ได้กำหนด key ตอนส่ง ทำให้ Kafka กระจาย round-robin แบบสุ่ม) consumer อาจอ่านเจอ `OrderShipped` **ก่อน** `OrderCreated` ได้ ซึ่งทำให้ business logic พังทันที (จะ mark ว่า order ถูกจัดส่งได้อย่างไรในเมื่อ state ปัจจุบันยังไม่รู้จัก order นี้เลย)

วิธีแก้คือต้องรับประกันว่า **event ทั้งหมดของ order เดียวกันไปที่ partition เดียวกันเสมอ** — ทำได้ผ่านการกำหนด **partition key**

### Partition Key และความเสี่ยงเรื่อง Data Skew

เมื่อ producer ส่ง message พร้อม **key** (เช่น `order_id`) Kafka จะ hash key นั้นแล้ว map ไปยัง partition เดิมเสมอสำหรับ key เดียวกัน (แนวคิดคล้าย consistent hashing ที่ใช้ใน load balancer — เชื่อมกับ [Volume 2 — Reverse Proxy/Load Balancer](../02-network/05-reverse-proxy-load-balancer.md)):

```
producer.send("orders", key="order-1001", value=OrderCreated)
producer.send("orders", key="order-1001", value=OrderPaid)
producer.send("orders", key="order-1001", value=OrderShipped)
──► ทั้งสาม message ไปที่ partition เดียวกันเสมอ (hash("order-1001") คงที่)
    รับประกัน ordering ของ order #1001 ทุก event
```

การเลือก partition key ที่ดีต้องคำนึงถึง **data skew** — ถ้า key กระจายค่าไม่สม่ำเสมอ (เช่น ใช้ `region` เป็น key ทั้งที่ 80% ของ traffic มาจาก region เดียว) partition ที่รับ key นั้นจะโหลดหนักกว่า partition อื่นมาก ทำให้ consumer ที่ดูแล partition นั้นทำงานหนักกว่าตัวอื่นอย่างไม่สมดุล (ระบบ scale ได้ไม่เต็มที่ตามจำนวน partition ที่มี เพราะ bottleneck ไปกระจุกอยู่ partition เดียว) หลักการเลือก key ที่ดีคือเลือกค่าที่**กระจายสม่ำเสมอ**และ**สอดคล้องกับหน่วยที่ต้องการ ordering จริงๆ** — เช่น `order_id` เหมาะกว่า `region` ถ้าสิ่งที่ต้องการคือ ordering ต่อ order ไม่ใช่ต่อ region

## 2.2 Consumer Group

### การแบ่ง Partition ให้ Consumer ในกลุ่ม

**Consumer group** คือกลุ่มของ consumer ที่ทำงานร่วมกันเพื่ออ่าน topic เดียวกัน โดย Kafka จะ**แบ่ง partition ให้แต่ละ consumer ในกลุ่มรับผิดชอบคนละส่วน** ไม่ให้ซ้ำกัน (แต่ละ partition มี consumer ในกลุ่มเดียวกันอ่านได้แค่ตัวเดียว ณ ขณะใดขณะหนึ่ง):

```
Topic "orders" มี 3 partition, Consumer Group "order-processor" มี 3 consumer:

Partition 0 ──► Consumer A
Partition 1 ──► Consumer B
Partition 2 ──► Consumer C
(แต่ละ consumer อ่านคนละ partition พร้อมกันจริง = parallelism)
```

ถ้าจำนวน consumer ในกลุ่ม**มากกว่า**จำนวน partition consumer ส่วนเกินจะ**ไม่ได้รับ partition ใดๆ เลย** (ว่างงาน ไม่มีอะไรให้อ่าน) — นี่คือเหตุผลที่จำนวน partition กำหนด**เพดานสูงสุด**ของ parallelism ที่ทำได้จริงสำหรับ consumer group หนึ่งกลุ่ม ถ้าต้องการ scale consumer มากกว่านี้ ต้องเพิ่มจำนวน partition ของ topic ก่อน

ในทางกลับกัน consumer group คนละกลุ่มที่อ่าน topic เดียวกัน**เป็นอิสระจากกันโดยสมบูรณ์** — แต่ละกลุ่มมี offset ของตัวเองแยกกัน กลุ่มหนึ่งอ่านไปถึงไหนไม่กระทบอีกกลุ่มเลย (เช่น กลุ่ม "order-processor" ประมวลผล order ปกติ กับกลุ่ม "order-analytics" ที่อ่าน topic เดียวกันไปทำ analytics แยกต่างหาก อ่านคนละความเร็ว คนละจุดได้อย่างอิสระ)

### Rebalancing: อะไรกระตุ้นและต้นทุนที่ต้องจ่าย

**Rebalancing** คือกระบวนการที่ Kafka แบ่ง partition ใหม่ให้กับ consumer ในกลุ่ม เกิดขึ้นเมื่อสมาชิกของกลุ่มเปลี่ยนแปลง:

- Consumer ใหม่เข้าร่วมกลุ่ม (เช่น scale up)
- Consumer ตัวใดตัวหนึ่งออกจากกลุ่ม (crash, restart, deploy ใหม่, หรือไม่ส่ง heartbeat ทันเวลา)
- จำนวน partition ของ topic เปลี่ยน (เพิ่ม partition)

ระหว่าง rebalancing เกิดขึ้น **consumer ทั้งกลุ่มหยุดประมวลผลชั่วคราว** (ในรูปแบบ rebalance แบบ stop-the-world ดั้งเดิม) เพื่อรอให้ partition ถูกแบ่งใหม่เสร็จก่อนกลับมาทำงานต่อ — นี่คือต้นทุนที่ต้องจ่าย: ยิ่ง consumer group มีสมาชิกเปลี่ยนแปลงบ่อย (เช่น deploy บ่อยมาก, autoscaling ที่ scale up/down ถี่ๆ) ยิ่งเจอ downtime สั้นๆ จาก rebalancing บ่อยตามไปด้วย Kafka รุ่นใหม่มี **cooperative rebalancing** ที่ลดผลกระทบนี้ลง (ให้ consumer ที่ยัง assignment เดิมอยู่ทำงานต่อได้ ไม่ต้อง stop ทั้งกลุ่ม) แต่หลักการพื้นฐานที่ว่า "การเปลี่ยนแปลงสมาชิกกลุ่มมีต้นทุน" ยังคงอยู่เสมอ

### Offset Management และ Delivery Semantics

Consumer ต้อง **commit offset** เพื่อบอก Kafka ว่า "อ่านและประมวลผลถึงจุดนี้แล้ว" — จุดที่ commit offset เทียบกับจุดที่ประมวลผลข้อมูลจริง เป็นตัวกำหนด delivery semantics ที่ได้:

| Semantics | วิธีทำ | สิ่งที่ผิดพลาดได้ |
|---|---|---|
| **At-most-once** | commit offset **ก่อน**ประมวลผลข้อมูล | ถ้า consumer crash หลัง commit แต่ก่อนประมวลผลเสร็จ — message นั้น**หายไปเงียบๆ** ไม่มีทางรู้เลยว่าประมวลผลไม่สำเร็จ เพราะ offset ถูก commit ไปแล้วว่า "อ่านแล้ว" |
| **At-least-once** | ประมวลผลข้อมูล**ก่อน** แล้วค่อย commit offset | ถ้า consumer crash หลังประมวลผลเสร็จแต่ก่อน commit offset สำเร็จ — พอ consumer restart จะอ่าน message เดิมซ้ำอีกครั้ง (เพราะ offset ยังไม่ถูก commit ว่าอ่านแล้ว) ทำให้**ประมวลผลซ้ำ (double-processing)** ได้ |
| **Exactly-once** | ใช้ Kafka transaction (`transactional.id`) ผูกการ commit offset กับการเขียนผลลัพธ์เข้าด้วยกันแบบ atomic | ซับซ้อนที่สุดในการ implement และมี overhead ด้าน performance สูงกว่า ใช้ได้เต็มรูปแบบเมื่อทั้ง input และ output อยู่ใน Kafka เท่านั้น (Kafka-to-Kafka) ถ้า output ไปที่ระบบภายนอก (เช่น เขียนลง database) ต้องอาศัยกลไกเพิ่มเติมเช่น idempotency ที่ปลายทางร่วมด้วย |

ในทางปฏิบัติ **at-least-once เป็นค่าที่ใช้กันมากที่สุด** เพราะ "อาจประมวลผลซ้ำแต่ไม่มีวันเสีย message" ปลอดภัยกว่า "อาจเสีย message เงียบๆ" ในเกือบทุก use case ทางธุรกิจ — แต่การใช้ at-least-once หมายความว่า**ตัว consumer ต้องออกแบบให้ idempotent เอง** (เชื่อมกับแนวคิด idempotency key ใน [Volume 11 — Idempotency](../11-microservices/05-idempotency-and-tracing.md)) เพื่อรองรับการที่ message เดิมอาจถูกประมวลผลซ้ำได้โดยไม่ทำให้ผลลัพธ์สุดท้ายผิดเพี้ยน (เช่น ถ้า consumer เขียนผลลัพธ์ลง database ด้วย `INSERT ... ON CONFLICT DO NOTHING` แทน `INSERT` ตรงๆ การประมวลผลซ้ำจะไม่ทำให้ข้อมูลซ้ำซ้อน)

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| Partition | หน่วยของ parallelism ของ topic แต่ละตัวเป็น log อิสระ |
| Ordering guarantee | รับประกันแค่ภายใน partition เดียวกัน ไม่รับประกันข้าม partition |
| Partition key | กำหนดให้ message ที่ต้องการ ordering ร่วมกันไปที่ partition เดียวกันเสมอ |
| Data skew | key ที่กระจายไม่สม่ำเสมอทำให้บาง partition โหลดหนักกว่าตัวอื่นมาก จำกัด scalability จริง |
| Consumer group | แบ่ง partition ให้ consumer ในกลุ่มคนละส่วน parallelism สูงสุด = จำนวน partition |
| Rebalancing | เกิดเมื่อสมาชิกกลุ่มเปลี่ยน (join/leave/partition เพิ่ม) มี downtime ชั่วคราวระหว่างแบ่งใหม่ |
| At-most-once | commit offset ก่อนประมวลผล — เสี่ยง message หายเงียบๆ ถ้า crash |
| At-least-once | ประมวลผลก่อน commit offset — เสี่ยงประมวลผลซ้ำ ต้องออกแบบ consumer ให้ idempotent |
| Exactly-once | ใช้ Kafka transaction ผูก offset กับผลลัพธ์แบบ atomic ซับซ้อนและ overhead สูงสุด |

## คำถามทบทวน

1. ทำไม Kafka ถึงไม่รับประกัน ordering ข้าม partition ยกตัวอย่างสถานการณ์จริงที่การไม่มี ordering ข้าม partition ก่อปัญหา
2. อธิบายว่าทำไมต้องส่ง `order_id` เป็น partition key เพื่อรับประกัน ordering ของ event ในแต่ละ order
3. ถ้า partition key เลือกไม่ดีจนเกิด data skew จะกระทบ scalability ของระบบอย่างไร
4. ทำไม consumer มากกว่าจำนวน partition ถึงไม่ช่วยเพิ่ม parallelism? ต้องแก้ปัญหานี้อย่างไร
5. เปรียบเทียบ at-most-once กับ at-least-once ว่าแต่ละแบบผิดพลาดแบบไหนได้ ทำไม at-least-once ถึงเป็นตัวเลือกที่ใช้บ่อยกว่าในทางปฏิบัติ และต้องแลกกับอะไร

---

ก่อนหน้า: [บทที่ 1 — Broker และ Topic](01-broker-and-topic.md) | ถัดไป: [บทที่ 3 — Dead Letter Queue (DLQ)](03-dlq.md)
