# บทที่ 1: Broker และ Topic

## 1.1 Broker

### Broker คือ Node หนึ่งใน Kafka Cluster

**Broker** คือ process ของ Kafka หนึ่งตัวที่รันอยู่บนเครื่อง (หรือ container) หนึ่งเครื่อง — ทำหน้าที่รับ message จาก producer, เก็บ message ลง disk, และเสิร์ฟ message ให้ consumer อ่าน **Kafka cluster** ประกอบด้วย broker หลายตัวทำงานร่วมกัน (ทั่วไปอย่างน้อย 3 ตัวใน production เพื่อทนต่อการที่ broker ตัวใดตัวหนึ่งล่มได้) แต่ละ broker รับผิดชอบเก็บข้อมูลบางส่วนของ cluster ไม่ใช่ทุก broker เก็บข้อมูลเหมือนกันหมด (รายละเอียดการแบ่งข้อมูลอยู่ในเรื่อง partition ใน [บทที่ 2](02-partition-and-consumer-group.md))

```
                Kafka Cluster
     ┌──────────┐  ┌──────────┐  ┌──────────┐
     │ Broker 1  │  │ Broker 2  │  │ Broker 3  │
     └──────────┘  └──────────┘  └──────────┘
          ▲               ▲               ▲
          └───────────────┴───────────────┘
                 Producer / Consumer เชื่อมต่อ broker ตัวไหนก็ได้
                 (client library รู้เองว่าข้อมูลที่ต้องการอยู่ broker ไหน)
```

### Controller Broker

ในบรรดา broker ทั้งหมด จะมี broker หนึ่งตัวถูกเลือกให้ทำหน้าที่พิเศษเป็น **controller** — รับผิดชอบงานบริหารจัดการระดับ cluster ที่ต้องมีผู้ตัดสินใจเพียงคนเดียว เช่น:

- ตรวจจับว่า broker ตัวไหนตายไป (ผ่านการ hеartbeat กับ cluster metadata layer)
- เลือก **leader partition** ใหม่เมื่อ broker ที่เป็น leader เดิมล่ม (ดูรายละเอียด leader/follower ด้านล่าง)
- แจ้ง broker ตัวอื่นๆ ให้รับทราบการเปลี่ยนแปลง metadata ของ cluster

ถ้า controller ตัวปัจจุบันล่ม cluster จะเลือก broker ตัวใหม่ขึ้นมาทำหน้าที่ controller แทนโดยอัตโนมัติ (Kafka รุ่นใหม่ที่ใช้ KRaft แทน ZooKeeper จัดการ election นี้ผ่าน Raft consensus ในตัว Kafka เอง ไม่ต้องพึ่ง ZooKeeper แยกต่างหากเหมือน Kafka รุ่นเก่า) กระบวนการนี้เกิดขึ้นอัตโนมัติโดย client (producer/consumer) ไม่จำเป็นต้องรู้ว่าใครเป็น controller อยู่เลย

### Replication Factor และ Leader/Follower Partition

เพื่อให้ข้อมูลไม่หายเมื่อ broker ตัวใดตัวหนึ่งล่ม Kafka ทำสำเนาข้อมูลของแต่ละ partition ไปเก็บไว้ในหลาย broker ตาม **replication factor** ที่กำหนด (เช่น replication factor = 3 หมายถึงข้อมูลของ partition นี้ถูกเก็บซ้ำใน 3 broker):

```
Partition 0 ของ topic "orders" (replication factor = 3):

     Broker 1: [Partition 0 - LEADER]     ← รับ write ทุกครั้ง, consumer อ่านจากนี่
     Broker 2: [Partition 0 - FOLLOWER]   ← copy ข้อมูลจาก leader ตลอดเวลา
     Broker 3: [Partition 0 - FOLLOWER]   ← copy ข้อมูลจาก leader ตลอดเวลา
```

ในบรรดา replica ทั้งหมดของ partition หนึ่ง จะมี**หนึ่งตัวเป็น leader** — producer เขียนข้อมูลไปที่ leader เท่านั้น, consumer อ่านจาก leader เท่านั้น (โดย default) ส่วน**follower**มีหน้าที่ copy ข้อมูลจาก leader มาเก็บไว้เผื่อ leader ล่ม — ถ้า broker ที่เป็น leader ล่มลง controller จะเลื่อน follower ตัวหนึ่งขึ้นเป็น leader ใหม่โดยอัตโนมัติ เพื่อให้ระบบยังรับ write/read ได้ต่อเนื่อง

### ISR (In-Sync Replicas)

ไม่ใช่ follower ทุกตัวจะ "up-to-date" กับ leader เสมอไป — follower อาจตามหลัง leader ได้จากหลายสาเหตุ (network ช้า, broker โหลดหนัก, disk I/O ช้า) Kafka จึงมีแนวคิด **ISR (In-Sync Replicas)** คือกลุ่มของ replica (รวม leader เอง) ที่ยัง sync ทันกับ leader อยู่จริง (ไม่ตามหลังเกินเวลาที่กำหนดไว้)

```
ISR = { Broker 1 (leader), Broker 2 (follower, sync ทัน) }
Broker 3 (follower) ตามหลังเกินกำหนด ──► ถูกตัดออกจาก ISR ชั่วคราว
```

ทำไม ISR สำคัญ: การตั้งค่า **`acks=all`** ฝั่ง producer (หมายถึง "รอให้ทุก broker ใน ISR ยืนยันว่าเขียนสำเร็จก่อนถือว่า write สำเร็จ") รับประกันความปลอดภัยของข้อมูลได้จริงก็ต่อเมื่อ ISR ยังมีขนาดใหญ่พอ (มักคู่กับตั้งค่า `min.insync.replicas` เพื่อบังคับว่า ISR ต้องมีอย่างน้อยกี่ตัวถึงจะยอมรับ write ได้ ถ้า ISR เหลือน้อยกว่าที่กำหนด producer จะได้ error แทนที่จะเขียนสำเร็จแบบเสี่ยงข้อมูลหาย) ถ้า follower ตามหลังมากจนถูกตัดออกจาก ISR แล้ว leader ล่มไปพอดี ระบบจะเลือก leader ใหม่จากเฉพาะ replica ที่อยู่ใน ISR เท่านั้น (ป้องกันไม่ให้ replica ที่ข้อมูลเก่ากว่ากลายเป็น leader แล้วทำให้ข้อมูลล่าสุดหายไป)

## 1.2 Topic

### Topic คือ Append-Only Log

**Topic** คือหมวดหมู่ของ message ที่ producer เขียนเข้าไปและ consumer อ่านออกมา โครงสร้างข้อมูลภายในของ topic คือ **log แบบ append-only** — message ใหม่ถูกเขียนต่อท้ายเสมอ ไม่มีการแทรกหรือแก้ไข message ที่มีอยู่แล้ว แต่ละ message มี **offset** (เลขลำดับที่เพิ่มขึ้นเรื่อยๆ) กำกับตำแหน่งของตัวเองในลอก:

```
Topic: "orders"
Offset:   0        1        2        3        4        5   (offset ถัดไป)
        ┌────┐  ┌────┐  ┌────┐  ┌────┐  ┌────┐  ┌─── new message
        │ M0 │  │ M1 │  │ M2 │  │ M3 │  │ M4 │  │  จะถูกเขียนที่นี่
        └────┘  └────┘  └────┘  └────┘  └────┘  └───►
        (เก่าสุด) ────────────────────────► (ใหม่สุด)
```

คุณสมบัติ append-only นี้ทำให้ Kafka เขียนข้อมูลได้เร็วมาก (แค่ append ต่อท้ายไฟล์บน disk แบบ sequential write ซึ่งเร็วกว่า random write มาก แม้จะเป็น disk แบบ spinning ก็ตาม) และทำให้ consumer หลายตัวอ่าน message ชุดเดียวกันซ้ำได้อย่างอิสระโดยไม่กระทบกัน (แค่แต่ละ consumer จำ offset ของตัวเองไว้ว่าอ่านถึงไหนแล้ว)

### Retention Policy: Time-based vs Size-based

Kafka ไม่ได้เก็บ message ไว้ตลอดไปเป็นค่าเริ่มต้น — มีนโยบาย **retention** กำหนดว่าจะลบ message เก่าเมื่อไหร่:

- **Time-based retention** (`retention.ms`) — ลบ message ที่เก่าเกินระยะเวลาที่กำหนด (ค่า default ทั่วไปคือ 7 วัน) ไม่ว่า consumer จะอ่านไปแล้วหรือยังก็ตาม
- **Size-based retention** (`retention.bytes`) — จำกัดขนาดรวมของ log ต่อ partition ถ้าเกินจะลบข้อมูลเก่าสุดทิ้งก่อน

ประเด็นสำคัญที่ต้องเข้าใจ: **Kafka ไม่ได้ลบ message ทันทีที่ consumer อ่านไปแล้ว** (ต่างจาก message queue แบบดั้งเดิมอย่าง RabbitMQ ที่ message มักหายไปหลังถูก consume) message ยังอยู่ใน log จนกว่าจะครบ retention period หรือเกิน retention size — นี่คือเหตุผลที่ consumer หลายกลุ่ม (consumer group คนละกลุ่ม) อ่าน topic เดียวกันซ้ำได้อย่างอิสระ แต่ละกลุ่มมี offset ของตัวเองแยกกัน (รายละเอียดใน [บทที่ 2](02-partition-and-consumer-group.md))

### Log Compaction: สำหรับ Topic แบบ Key-Value Snapshot

Retention แบบเวลา/ขนาดเหมาะกับ topic ที่เป็น "event log" ทั่วไป (สนใจทุก event ที่เกิดขึ้น) แต่บาง use case ต้องการ Kafka ทำหน้าที่เหมือน **key-value store ล่าสุด** มากกว่า event log — Kafka มี **log compaction** ตอบโจทย์นี้โดยเฉพาะ

แนวคิด: แทนที่จะลบ message ตามอายุ/ขนาด compaction จะ**เก็บเฉพาะ message ล่าสุดของแต่ละ key** ไว้ ลบ message เก่าที่มี key ซ้ำกันทิ้ง:

```
ตัวอย่าง: topic "user-profile" (compacted) เก็บ current profile ของแต่ละ user

ก่อน compaction:
  Offset 0: key=user-1, value={name: "Somchai", city: "Bangkok"}
  Offset 1: key=user-2, value={name: "Malee",   city: "Chiang Mai"}
  Offset 2: key=user-1, value={name: "Somchai", city: "Khon Kaen"}   ← user-1 ย้ายเมือง
  Offset 3: key=user-1, value={name: "Somchai", city: "Phuket"}      ← user-1 ย้ายอีก

หลัง compaction (offset 0 กับ 2 ถูกลบ เพราะมี key=user-1 ที่ใหม่กว่า):
  Offset 1: key=user-2, value={name: "Malee",   city: "Chiang Mai"}
  Offset 3: key=user-1, value={name: "Somchai", city: "Phuket"}      ← ค่าล่าสุดของ user-1 เท่านั้นที่เหลือ
```

ผลลัพธ์คือ topic นี้ทำหน้าที่เหมือน **snapshot ปัจจุบันของทุก key** แทนที่จะเป็นประวัติทั้งหมด — เหมาะกับ use case เช่น เก็บ "current state" ของ entity ที่อัปเดตบ่อย (user profile, current price ของสินค้า, configuration ล่าสุด) ที่ consumer ใหม่ที่เพิ่งเข้ามาต้องการแค่ "ค่าล่าสุด" ไม่ต้องไล่อ่านประวัติทั้งหมดตั้งแต่ต้น ต่างจาก topic แบบ retention ปกติที่เหมาะกับ event ที่ทุก event มีความหมายของตัวเอง (เช่น "order ถูกสร้าง" กับ "order ถูกยกเลิก" คือคนละเหตุการณ์ ไม่ใช่แค่ค่าล่าสุดของ key เดียวกัน)

การลบ message ด้วย tombstone (ส่ง value เป็น `null` สำหรับ key นั้น) ใน topic แบบ compacted คือวิธีบอกว่า "ลบ key นี้ออกจาก snapshot ไปเลย" — compaction จะเก็บ tombstone ไว้ชั่วระยะหนึ่งก่อนลบออกจริง เพื่อให้ consumer ที่ยังตามอ่านไม่ทันมีโอกาสเห็น tombstone นี้และรู้ว่าต้องลบ key นั้นออกจาก state ของตัวเองด้วย

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| Broker | node หนึ่งใน Kafka cluster รับ write/serve read ของ partition ที่ตัวเองรับผิดชอบ |
| Controller broker | broker พิเศษที่จัดการ metadata ระดับ cluster เช่น เลือก leader partition ใหม่เมื่อ broker ล่ม |
| Replication factor | จำนวนสำเนาของแต่ละ partition ที่กระจายอยู่ในหลาย broker |
| Leader / Follower | producer/consumer คุยกับ leader เท่านั้น, follower copy ข้อมูลเผื่อ leader ล่ม |
| ISR | replica ที่ยัง sync ทันกับ leader — `acks=all` + `min.insync.replicas` ป้องกันข้อมูลหายเมื่อ leader ล่ม |
| Topic | log แบบ append-only แต่ละ message มี offset กำกับลำดับ |
| Retention | time-based (`retention.ms`) หรือ size-based (`retention.bytes`) — Kafka ไม่ลบทันทีที่ consume |
| Log Compaction | เก็บเฉพาะ message ล่าสุดต่อ key เหมาะกับ topic แบบ current-state snapshot |

## คำถามทบทวน

1. Controller broker มีหน้าที่อะไรที่ต่างจาก broker ทั่วไป และเกิดอะไรขึ้นถ้า controller ล่ม
2. อธิบายบทบาทของ leader กับ follower ของ partition หนึ่งตัว ทำไม consumer ถึงอ่านจาก leader เท่านั้นโดย default
3. ISR คืออะไร ทำไมการเลือก leader ใหม่ต้องเลือกจากใน ISR เท่านั้น ถ้าเลือก replica ที่ตามหลังอยู่นอก ISR จะเกิดปัญหาอะไร
4. ทำไม Kafka ถึงไม่ลบ message ทันทีที่ consumer อ่านไปแล้ว ต่างจาก message queue แบบดั้งเดิมอย่างไร
5. ยกตัวอย่าง use case ที่เหมาะกับ log compaction และอธิบายว่าทำไม retention แบบเวลา/ขนาดปกติถึงไม่ตอบโจทย์ use case นั้น

---

ถัดไป: [บทที่ 2 — Partition และ Consumer Group](02-partition-and-consumer-group.md)
