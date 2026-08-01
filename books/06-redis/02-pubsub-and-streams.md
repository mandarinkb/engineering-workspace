# บทที่ 2: Pub/Sub และ Streams

นอกจากเป็น cache แล้ว Redis ยังมีความสามารถด้าน messaging สองรูปแบบที่แตกต่างกันมาก: **Pub/Sub** ที่เรียบง่ายแต่ไม่เก็บข้อมูล และ **Streams** ที่หนักแน่นกว่า เก็บ log ได้และ replay ย้อนหลังได้ บทนี้จะพาดูว่าแต่ละแบบทำงานอย่างไร ข้อจำกัดของ Pub/Sub คืออะไร และทำไมในหลายสถานการณ์ Streams ถึงเป็นทางเลือกที่เหมาะกว่า

## 2.1 Pub/Sub ทำงานอย่างไร

`PUBLISH`/`SUBSCRIBE` ใช้โมเดล **fan-out** แบบง่ายที่สุด — publisher ส่งข้อความไปยัง channel หนึ่ง ทุก subscriber ที่ผูกอยู่กับ channel นั้น ณ ขณะนั้นจะได้รับข้อความพร้อมกัน:

```
Publisher ──PUBLISH channel:news "breaking news"──► Redis
                                                        │
                          ┌─────────────────────────────┼─────────────────────────────┐
                          ▼                              ▼                              ▼
                    Subscriber A                   Subscriber B                   Subscriber C
                    (ได้รับข้อความ)                 (ได้รับข้อความ)                 (ได้รับข้อความ)
```

```
# Terminal 1: subscribe
SUBSCRIBE channel:news

# Terminal 2: publish
PUBLISH channel:news "breaking news"
```

Redis ยังรองรับ `PSUBSCRIBE` (subscribe ด้วย pattern เช่น `channel:*` รับข้อความจากทุก channel ที่ชื่อขึ้นต้นด้วย `channel:`) ทำให้ยืดหยุ่นกว่าการ subscribe ทีละ channel

## 2.2 ข้อจำกัดของ Pub/Sub: Fire-and-Forget

จุดสำคัญที่สุดที่ต้องเข้าใจคือ Redis Pub/Sub เป็น **fire-and-forget โดยสมบูรณ์** — Redis **ไม่เก็บข้อความไว้เลย** แม้แต่มิลลิวินาทีเดียว ข้อความที่ publish ออกไปจะถูกส่งให้เฉพาะ subscriber ที่ connect และ subscribe อยู่ ณ ขณะนั้นเท่านั้น

```
เวลา T1: Subscriber A subscribe channel:news
เวลา T2: Publisher PUBLISH "message 1"  → A ได้รับ
เวลา T3: Subscriber A connection หลุด (network glitch, restart, ฯลฯ)
เวลา T4: Publisher PUBLISH "message 2"  → หายไปเลย ไม่มีใครได้รับ (A ไม่ได้ subscribe อยู่แล้ว)
เวลา T5: Subscriber A reconnect + subscribe ใหม่
เวลา T6: Publisher PUBLISH "message 3"  → A ได้รับ (แต่ "message 2" หายไปตลอดกาล ไม่มีทาง replay)
```

ผลกระทบเชิงปฏิบัติ:

- **ไม่มี persistence** — restart Redis หรือ subscriber offline ชั่วขณะ = เสียข้อความที่ publish ระหว่างนั้นไปตลอดกาล
- **ไม่มี message queue/buffer** — ถ้า publish เร็วกว่าที่ subscriber ประมวลผลทัน ข้อความไม่ได้ถูกเก็บรอ (ต่างจาก queue ที่แท้จริง)
- **ไม่มี consumer group** — ทุก subscriber ที่ subscribe channel เดียวกันจะได้รับข้อความ**ทุกข้อความซ้ำกันหมด** ไม่สามารถแบ่งงานกันประมวลผลข้อความคนละส่วนได้ (ไม่มีแนวคิด "load balance งานระหว่าง consumer หลายตัว")

### เทียบกับ Kafka

Pub/Sub ของ Redis กับ [Kafka](../12-kafka/README.md) ต่างกันโดยพื้นฐานตรงจุดประสงค์การออกแบบ:

| | Redis Pub/Sub | Kafka |
|---|---|---|
| Persistence | ไม่มีเลย — ข้อความหายทันทีที่ส่งเสร็จ | มี — เก็บเป็น log บน disk ตาม retention policy |
| Consumer offline | พลาดข้อความถาวร | กลับมาอ่านต่อจากจุดที่ค้างไว้ได้ (offset) |
| Replay ข้อความเก่า | ทำไม่ได้ | ทำได้ (อ่านจาก offset ใดก็ได้ภายใน retention) |
| Consumer group / load balancing | ไม่มี | มี (partition ถูกแบ่งให้ consumer ใน group) |
| Use case ที่เหมาะ | real-time notification ที่ไม่สนใจข้อความที่พลาดไป (เช่น "someone is typing...") | event streaming ที่ต้อง durable, replay ได้, ordering รับประกัน |

Redis Pub/Sub เหมาะกับงานที่ **"ถ้าไม่มีคนฟังตอนนี้ ก็ไม่เป็นไร ข้อความนี้ไม่สำคัญพอที่จะเก็บไว้"** เช่น real-time cursor position ในเอกสารที่แก้ไขร่วมกัน, notification แบบ "กำลังพิมพ์..." ในแชท ส่วนงานที่ข้อความทุกตัว**ต้องไม่หาย**และต้อง**ประมวลผลได้แน่นอนสักครั้ง** (เช่น order event, payment event) ต้องมองหาเครื่องมือที่ durable กว่านี้

## 2.3 Redis Streams: ทางเลือกที่ Durable และ Replay ได้

**Streams** (เพิ่มเข้ามาใน Redis 5.0) ถูกออกแบบมาแก้ข้อจำกัดของ Pub/Sub โดยตรง — เก็บข้อความเป็น **append-only log** ที่มี ID เรียงลำดับตามเวลา คล้ายแนวคิดของ Kafka topic แต่เบากว่าและรันอยู่ใน Redis ตัวเดียวกับที่ใช้เป็น cache อยู่แล้ว

### XADD — เพิ่มข้อความเข้า Stream

```
XADD orders:stream * order_id 1001 status "created" amount 500
```

`*` บอกให้ Redis สร้าง ID อัตโนมัติ (รูปแบบ `<timestamp-ms>-<sequence>` เช่น `1721000000000-0`) รับประกัน ID เรียงจากน้อยไปมากเสมอ ใช้เป็น cursor สำหรับอ่านต่อได้

### XRANGE / XREAD — อ่านข้อมูล

```
XRANGE orders:stream - +
-- อ่านทุกข้อความตั้งแต่ต้นจนจบ (- คือ ID ต่ำสุด, + คือ ID สูงสุด)

XREAD COUNT 10 STREAMS orders:stream 0
-- อ่านสูงสุด 10 ข้อความ เริ่มจาก ID 0 (ตั้งแต่ต้น stream)

XREAD BLOCK 5000 STREAMS orders:stream $
-- รอข้อความใหม่แบบ blocking สูงสุด 5000ms, $ หมายถึง "เฉพาะข้อความที่มาใหม่หลังจากนี้"
```

ข้อความใน stream **ไม่หายไปเมื่ออ่านแล้ว** (ต่างจาก queue ทั่วไปที่ pop ออกแล้วหาย) — สามารถอ่านซ้ำ, อ่านย้อนหลังจาก ID ไหนก็ได้ ตราบใดที่ข้อความนั้นยังไม่ถูก trim ทิ้ง (จำกัดขนาด stream ได้ด้วย `MAXLEN`)

### Consumer Group: XGROUP / XREADGROUP

นี่คือความสามารถที่ทำให้ Streams ใกล้เคียง Kafka มากที่สุด — แบ่งงานให้ consumer หลายตัวใน group เดียวกันอ่านข้อความคนละส่วน (ไม่ซ้ำกัน) แทนที่ทุกตัวจะได้ข้อความเดียวกันหมดเหมือน Pub/Sub:

```
XGROUP CREATE orders:stream order-processors 0
-- สร้าง consumer group ชื่อ "order-processors" เริ่มอ่านจาก ID 0

XREADGROUP GROUP order-processors worker-1 COUNT 5 STREAMS orders:stream >
-- worker-1 ขอข้อความใหม่ (>) สูงสุด 5 ข้อความจาก group นี้
-- Redis จะไม่ส่งข้อความเดียวกันซ้ำให้ worker-2 ที่อยู่ group เดียวกัน

XACK orders:stream order-processors 1721000000000-0
-- worker-1 ประมวลผลเสร็จแล้ว ยืนยันว่ารับผิดชอบข้อความนี้เรียบร้อย
```

```
                     orders:stream (log เดียว)
                            │
              ┌─────────────┴─────────────┐
              ▼                           ▼
        worker-1 (อ่านข้อความ 1,3,5)   worker-2 (อ่านข้อความ 2,4,6)
        ── แบ่งงานกัน ไม่ได้รับซ้ำ ──
```

ถ้า `worker-1` รับข้อความไปแล้วแต่ล่มก่อนจะ `XACK` ข้อความนั้นจะค้างอยู่ใน **Pending Entries List (PEL)** ของ consumer group — ใช้ `XPENDING` ดูรายการที่ค้าง และ `XCLAIM` ให้ consumer ตัวอื่นเข้ามารับช่วงประมวลผลข้อความที่ค้างนั้นแทนได้ นี่คือกลไกที่ทำให้ Streams รับประกัน "ข้อความจะถูกประมวลผลอย่างน้อยหนึ่งครั้ง" (at-least-once delivery) ซึ่ง Pub/Sub ไม่มีให้เลย

## 2.4 เมื่อไหร่เลือก Streams แทน Pub/Sub

| ต้องการ | เลือก |
|---|---|
| Fire-and-forget, real-time เท่านั้น, ข้อความพลาดได้ไม่เป็นไร | Pub/Sub |
| ต้องรับประกันว่าข้อความไม่หาย แม้ consumer offline ชั่วคราว | Streams |
| ต้องการ replay ข้อความเก่าย้อนหลังได้ | Streams |
| ต้องแบ่งงานระหว่าง worker หลายตัว (load balancing) | Streams (consumer group) |
| Volume ข้อความสูงมาก ต้องการ retention ยาวนาน, multi-datacenter | พิจารณา [Kafka](../12-kafka/README.md) แทน (Streams เหมาะกับ scale ระดับกลาง ไม่ได้ถูกออกแบบมาแข่งกับ Kafka โดยตรงในงาน volume สูงมากๆ) |

กฎตัดสินใจง่ายๆ: ถ้าคำถามคือ **"ถ้า consumer offline ไปห้านาที ข้อความที่พลาดไปสำคัญไหม"** — ถ้าคำตอบคือสำคัญ ต้องใช้ Streams (หรือ Kafka) ไม่ใช่ Pub/Sub เด็ดขาด

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| PUBLISH/SUBSCRIBE | fan-out แบบง่าย ทุก subscriber ที่ subscribe อยู่ได้รับข้อความเดียวกันทันที |
| ข้อจำกัด Pub/Sub | fire-and-forget เต็มรูปแบบ ไม่มี persistence, subscriber offline = พลาดข้อความถาวร |
| Redis Streams | append-only log มี ID เรียงลำดับ อ่านซ้ำ/ย้อนหลังได้ ไม่ลบข้อความหลังอ่าน |
| XADD/XRANGE/XREAD | เพิ่ม/อ่านข้อความจาก stream ตาม ID |
| Consumer group (XGROUP/XREADGROUP) | แบ่งงานระหว่าง consumer หลายตัว ไม่ได้รับข้อความซ้ำกัน |
| XACK/XPENDING/XCLAIM | รับประกัน at-least-once delivery ผ่าน Pending Entries List |
| Streams vs Kafka | Streams เหมาะ scale ระดับกลาง, Kafka เหมาะ volume สูงมาก/retention ยาว/multi-datacenter |

## คำถามทบทวน

1. อธิบายว่าทำไม Redis Pub/Sub ถึงเรียกว่า "fire-and-forget" ยกตัวอย่างสถานการณ์ที่ subscriber จะพลาดข้อความถาวร
2. Consumer group ใน Streams แก้ปัญหาอะไรที่ Pub/Sub ทำไม่ได้?
3. `XACK` มีความสำคัญอย่างไรกับกลไก at-least-once delivery ถ้า consumer ไม่เรียก `XACK` เลยจะเกิดอะไรขึ้น?
4. ยกตัวอย่าง use case ที่เหมาะกับ Pub/Sub จริงๆ (ที่ข้อความพลาดได้ไม่เป็นไร) และ use case ที่ต้องใช้ Streams แทน
5. เพราะเหตุใดระบบที่ volume ข้อความสูงมากและต้องการ retention ยาวนาน มักเลือก Kafka แทนที่จะใช้ Redis Streams

---

ก่อนหน้า: [บทที่ 1 — Cache และ TTL](01-caching-and-ttl.md) | ถัดไป: [บทที่ 3 — Cluster](03-cluster.md)
