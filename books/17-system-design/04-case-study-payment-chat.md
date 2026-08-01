# บทที่ 4: Case Study — Payment และ Chat

บทนี้เริ่มสังเคราะห์ความรู้จากทุก Volume ก่อนหน้าเข้าด้วยกันเป็นระบบจริง 2 แบบที่มี requirement ต่างขั้วกัน — Payment ต้องการ **consistency สูงสุด** (ยอมช้าแต่ต้องถูกต้องเป๊ะ) ส่วน Chat ต้องการ **throughput/latency สูงสุด** (real-time เป็นหัวใจ ยอมรับ eventual consistency บางส่วนได้)

## 4.1 Case Study: Payment System

ระบบ payment คือตัวอย่างคลาสสิกที่สุดของ CP-leaning system (ตามกรอบบทที่ 1) — ความผิดพลาดแม้เพียงเล็กน้อย (เงินหาย, เงินซ้ำ) สร้างความเสียหายทางธุรกิจและกฎหมายที่ยอมรับไม่ได้ ระบบต้องออกแบบให้ **ผิดพลาดไม่ได้เลย แม้ในสถานการณ์ที่เครือข่ายล้มเหลวกลางทาง**

### Idempotency — รากฐานที่ขาดไม่ได้

Payment API ทุกตัวต้องรองรับ **idempotency**: ยิง request เดิมซ้ำกี่ครั้ง (เพราะ client timeout แล้ว retry, หรือ network ส่ง request ซ้ำโดยไม่ตั้งใจ) ผลลัพธ์สุดท้ายต้องเหมือนกับยิงครั้งเดียว — รายละเอียดกลไก (idempotency key, การเก็บผลลัพธ์ของ request ที่เคยประมวลผลแล้ว) อยู่ใน [Volume 11 — Idempotency และ Tracing](../11-microservices/05-idempotency-and-tracing.md) จุดสำคัญที่ต้องย้ำในบริบท payment คือ **ทุก mutation ที่แตะเงินต้องมี idempotency key ผูกกับ request นั้นเสมอ ไม่มีข้อยกเว้น** เพราะ retry คือสิ่งที่เกิดขึ้นแน่นอนใน distributed system (timeout ไม่ได้แปลว่า request ไม่สำเร็จ อาจแค่ response หายระหว่างทาง)

### Double-Entry Ledger — ทำไมต้องมีสองด้านเสมอ

ระบบ payment จำนวนมากออกแบบผิดพลาดด้วยการเก็บ "ยอดเงินคงเหลือ" เป็นตัวเลขเดียวแล้ว update ทับ (เช่น `UPDATE accounts SET balance = balance - 100`) วิธีนี้มีปัญหาใหญ่: **ถ้าข้อมูลเสียหายหรือ transaction บางส่วนหายไป ไม่มีทางรู้เลยว่าตัวเลขที่เหลืออยู่ถูกต้องหรือไม่** เพราะไม่มีอะไรให้ตรวจสอบไขว้กัน

**Double-entry ledger** (แนวคิดจากบัญชีคู่ที่ใช้กันมาหลายร้อยปีในวงการบัญชี) แก้ปัญหานี้ด้วยหลักการง่ายๆ: **ทุก transaction มีสองด้านที่สมดุลกันเสมอ** เงินที่ออกจากบัญชีหนึ่งต้องเท่ากับเงินที่เข้าอีกบัญชีหนึ่งพอดี

```
โอนเงิน 100 บาท จาก Wallet A ไป Wallet B:

Single-balance mutation (เสี่ยง):        Double-entry ledger (ตรวจสอบได้):
  UPDATE wallet_a SET balance -= 100      ledger_entry: { account: A, debit:  100 }
  UPDATE wallet_b SET balance += 100      ledger_entry: { account: B, credit: 100 }
  → ถ้า statement ที่สองหาย เงินหายลอย    → sum(debit) ต้องเท่ากับ sum(credit) เสมอ
     ไม่มีทางรู้จากตัวเลขคงเหลืออย่างเดียว   → ถ้าไม่เท่ากัน แปลว่ามีข้อมูลเสียหาย ตรวจจับได้ทันที
```

จุดที่สำคัญที่สุดคือ **trial balance** — การรวมยอด debit และ credit ของทุก ledger entry เป็นระยะ ถ้าผลรวมทั้งสองฝั่งไม่เท่ากัน แปลว่ามีข้อมูลเสียหายหรือ transaction ที่ทำไม่ครบเกิดขึ้นแน่นอน วิธีนี้เปลี่ยนปัญหา "ข้อมูลเงินหายแบบเงียบๆ" (ตรวจจับไม่ได้เลยจนกว่าลูกค้าจะร้องเรียน) ให้กลายเป็นปัญหาที่**ตรวจจับได้อัตโนมัติ**ผ่านการรัน trial balance เป็นระยะ ยอดคงเหลือของบัญชีหนึ่งๆ ก็ไม่ได้เก็บเป็นตัวเลขลอยๆ อีกต่อไป แต่คำนวณได้จาก sum ของทุก ledger entry ที่เกี่ยวข้อง (source of truth คือ log ของ entry ทั้งหมด ไม่ใช่ตัวเลข current balance)

### Saga/Temporal Orchestration: Reserve → Charge → Confirm

การชำระเงินจริงมักไม่ใช่ operation เดียว แต่เป็นหลายขั้นตอนที่ต้องประสานกันข้าม service (เช่น service เก็บ inventory, service เรียก payment gateway ภายนอก, service ยืนยัน order) — รูปแบบ orchestration ที่จัดการ multi-step workflow พร้อม compensation เมื่อล้มเหลวกลางทาง อยู่ใน [Volume 13 — Retry และ Compensation](../13-temporal/02-retry-and-compensation.md)

ขั้นตอนทั่วไปของ payment flow แบบ 3 จังหวะ:

1. **Reserve** — จองวงเงิน/สินค้าไว้ชั่วคราว (ยังไม่ตัดเงินจริง) เพื่อกันไม่ให้ resource ถูกใช้ซ้ำระหว่างรอ confirm
2. **Charge** — เรียก payment gateway ภายนอกตัดเงินจริงจากบัตร/wallet ของลูกค้า
3. **Confirm** — ยืนยัน order ว่าสำเร็จ ปลด reserve เป็นสถานะถาวร

### Sequence Diagram: Payment ที่ล้มเหลวกลางทางและการ Compensate

```
Client       Order Service     Inventory Service   Payment Gateway
  │                │                   │                  │
  │──create order─►│                   │                  │
  │                │──reserve stock───►│                  │
  │                │◄──reserved OK─────│                  │
  │                │                   │                  │
  │                │──charge 500 บาท──────────────────────►│
  │                │                   │        ✕ timeout / gateway ล่ม
  │                │◄─── ไม่มี response กลับมา (ไม่รู้ว่า charge สำเร็จหรือไม่) │
  │                │                   │                  │
  │                │── (compensation เริ่มทำงาน) ──────────│
  │                │──release stock───►│                  │
  │                │◄──released────────│                  │
  │                │──query charge status (idempotency key เดิม)──►│
  │                │◄── ผลลัพธ์จริง: "not charged" ──────────────│
  │◄──order failed, ไม่มีการตัดเงิน────│                   │                  │
```

จุดสำคัญของ diagram นี้: เมื่อ charge step timeout **ห้ามสันนิษฐานว่า charge ล้มเหลว** เพราะ gateway อาจตัดเงินสำเร็จแล้วแค่ response หายไประหว่างทาง ขั้นตอนที่ถูกต้องคือ **query สถานะจริงด้วย idempotency key เดิม** ก่อนตัดสินใจ compensate — นี่คือเหตุผลที่ idempotency (หัวข้อก่อนหน้า) กับ Saga compensation ต้องทำงานร่วมกันเสมอ ถ้าพบว่าไม่ได้ถูกตัดเงินจริง ค่อย release stock ที่ reserve ไว้ (compensating transaction) แล้วแจ้ง client ว่า order ล้มเหลว — ถ้าใช้ Temporal เป็น orchestrator ทั้ง flow นี้ (reserve, charge, query status, compensate) ถูกเขียนเป็น workflow เดียวที่ Temporal รับประกันว่าจะรันจนจบแม้ worker process ที่รัน workflow ล่มกลางทาง (durable execution)

## 4.2 Case Study: Chat System

Chat system มี requirement ต่างจาก payment โดยสิ้นเชิง: เน้น **real-time delivery** และ **scale ของจำนวน connection พร้อมกัน** มากกว่าความถูกต้องระดับธุรกรรมการเงิน

### WebSocket Connection Lifecycle และการเปลี่ยน Scaling Model

HTTP ทั่วไปเป็น request-response แบบสั้นๆ (เปิด connection, ยิง request, ปิด) แต่ chat ต้องการ **WebSocket** — connection ที่**เปิดค้างไว้ตลอดเวลา**ระหว่าง client กับ server เพื่อให้ server push ข้อความใหม่ไปหา client ได้ทันทีโดยไม่ต้อง client มา poll เอง

ความเปลี่ยนแปลงสำคัญที่ตามมา: **จำนวน concurrent connection กลายเป็นมิติของ capacity ที่ต้องวางแผนแยกต่างหากจาก request throughput** ระบบ REST API ทั่วไปวัด capacity ด้วย "รับกี่ request ต่อวินาที" แต่ server ที่ hold WebSocket connection ต้องตอบคำถามเพิ่มว่า **"1 server รับ connection ที่เปิดค้างพร้อมกันได้กี่ตัว"** ก่อน — แม้ server จะไม่มี traffic เข้ามาเลย connection ที่เปิดค้างไว้เฉยๆ ก็ยังกิน memory (buffer ต่อ connection) และ file descriptor (แต่ละ connection คือ 1 file descriptor — เชื่อมกับแนวคิด socket ใน [Volume 2 — OSI/TCP/IP](../02-network/01-osi-tcp-ip.md)) การ scale chat server จึงมักคำนวณจาก "connection ต่อ server" เป็นตัวตั้งต้น ไม่ใช่ "request ต่อวินาที" แบบ REST API ทั่วไป

### Message Ordering Guarantee

ผู้ใช้คาดหวังว่าข้อความในห้องแชทจะเรียงตามลำดับเวลาที่ส่งจริง (หรืออย่างน้อยก็เรียงตามลำดับที่เข้าใจได้) การรับประกัน ordering ในระบบ distributed ทำได้หลายระดับ:

- **Per-conversation ordering** — ข้อความใน conversation เดียวกันต้องเรียงลำดับถูกต้องเสมอ (มักทำได้ด้วยการกำหนดให้ทุกข้อความของ conversation หนึ่งผ่าน partition/queue เดียวกันเสมอ เพื่อรักษาลำดับ FIFO)
- **Global ordering ข้าม conversation** — ไม่จำเป็นและมักไม่คุ้มที่จะทำ (เพิ่ม coordination overhead มหาศาลโดยไม่มีประโยชน์ต่อผู้ใช้)

### Fan-Out ที่ Scale ได้: Redis Pub/Sub vs Kafka

เมื่อข้อความหนึ่งต้องถูกส่งไปหา**ผู้รับหลายคนพร้อมกัน** (group chat ขนาดใหญ่, broadcast) กระบวนการนี้เรียกว่า **fan-out** — มี 2 backbone หลักที่ใช้กัน โดยแลกด้วย trade-off คนละแบบ:

| | Redis Pub/Sub | Kafka |
|---|---|---|
| ความเรียบง่าย | ตั้งค่าง่าย, latency ต่ำมาก | ซับซ้อนกว่า (partition, consumer group) |
| Durability | **ไม่มี** — ถ้า subscriber ไม่ online ตอนส่ง ข้อความหายไปเลย | **มี** — message เก็บไว้บน disk ตาม retention period |
| Replay | ทำไม่ได้ | ทำได้ — consumer อ่านย้อนจาก offset เก่าได้ |
| เหมาะกับ | ข้อความ ephemeral เช่น typing indicator, presence update | ข้อความที่ต้องรับประกันการส่งถึง เช่น chat message ที่ user ต้องเห็นแม้ offline ตอนที่ส่ง |

รายละเอียดกลไกของแต่ละตัวอยู่ใน [Volume 6 — Redis Pub/Sub และ Streams](../06-redis/02-pubsub-and-streams.md) และ [Volume 12 — Kafka](../12-kafka/README.md) — ระบบ chat จริงส่วนใหญ่**ใช้ทั้งสองแบบผสมกัน**: Kafka (หรือ Redis Streams ที่มี durability) เป็นแหล่งความจริงเก็บประวัติข้อความและรับประกันว่าข้อความจะไม่หายแม้ผู้รับ offline อยู่ ส่วน Pub/Sub ใช้สำหรับ signal ที่ไม่ต้องรับประกันการส่งถึง (เช่น "กำลังพิมพ์อยู่") ที่ไม่คุ้มจะแบก overhead ของ durable storage

```
ข้อความใหม่เข้ามา
      │
      ├──► เขียนลง durable log (Kafka/Redis Stream) → เก็บประวัติ, ผู้รับที่ offline อ่านย้อนได้ตอนกลับมา online
      │
      └──► publish ผ่าน Pub/Sub → connection ที่ online อยู่ตอนนี้ได้รับทันที (low latency)
```

### Presence System — ใครออนไลน์อยู่บ้าง

Presence (สถานะ online/offline ของผู้ใช้) เป็นปัญหาคลาสสิกของ chat system ที่ดูเหมือนง่ายแต่ scale ยาก แนวทางมาตรฐานคือ **TTL-based heartbeat** — รายละเอียดกลไก TTL อยู่ใน [Volume 6 — Caching และ TTL](../06-redis/01-caching-and-ttl.md):

```
Client ──heartbeat ทุก 15 วินาที──► Server ──SET user:123:online EX 30──► Redis

ถ้า client หายไปกลางทาง (ปิดแอป, เน็ตหลุด) โดยไม่ได้แจ้ง "logout" อย่างเป็นทางการ:
  → heartbeat หยุดส่ง → key หมดอายุหลัง 30 วินาที → ระบบถือว่า user offline โดยอัตโนมัติ
```

ข้อดีของแนวทางนี้คือ**ไม่ต้องพึ่ง event "disconnect" ที่เชื่อถือได้ 100%** (เพราะ TCP connection ที่ขาดกะทันหันมักไม่ส่ง event ปิดอย่างเป็นทางการมาให้ server เสมอไป — network เน็ตหลุด, มือถือปิดแอปโดยไม่บอก OS) TTL เป็นกลไกที่ **fail-safe โดยธรรมชาติ**: ถ้าไม่มี heartbeat มาต่อเนื่อง ระบบจะถือว่า offline เองเสมอ ไม่ต้องมีใครมาบอกอย่างชัดเจนว่า user ออกไปแล้ว

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| Idempotency ใน payment | ทุก mutation ที่แตะเงินต้องมี idempotency key เพราะ retry เกิดขึ้นแน่นอน |
| Double-entry ledger | ทุก transaction มีสองด้านสมดุลกันเสมอ ตรวจจับข้อมูลเสียหายได้ผ่าน trial balance |
| Saga/Temporal สำหรับ payment | Reserve → Charge → Confirm พร้อม compensation; ต้อง query สถานะจริงก่อน compensate เสมอเมื่อ timeout |
| WebSocket เปลี่ยน scaling model | connection count กลายเป็นมิติ capacity แยกจาก request throughput |
| Fan-out backbone | Pub/Sub เร็วแต่ไม่ durable, Kafka durable และ replay ได้ ระบบจริงมักใช้ทั้งคู่ผสมกัน |
| Presence | TTL-based heartbeat — fail-safe โดยธรรมชาติ ไม่ต้องพึ่ง disconnect event ที่เชื่อถือไม่ได้ |

## คำถามทบทวน

1. ทำไม single-balance mutation ถึงเสี่ยงกว่า double-entry ledger เมื่อข้อมูลบางส่วนหายไประหว่างทาง อธิบายบทบาทของ trial balance
2. เมื่อ charge step ใน payment saga timeout ทำไมถึงห้ามสันนิษฐานทันทีว่า charge ล้มเหลวแล้วสั่ง compensate เลย ควรทำอะไรก่อน
3. ทำไม WebSocket ถึงเปลี่ยนวิธีคิดเรื่อง capacity planning จาก "request ต่อวินาที" เป็นอย่างอื่น
4. เปรียบเทียบ Redis Pub/Sub กับ Kafka ในบริบท fan-out ของ chat message ควรใช้ตัวไหนสำหรับ chat message ปกติ และตัวไหนสำหรับ typing indicator เพราะอะไร
5. อธิบายว่าทำไม TTL-based heartbeat ถึงเป็นแนวทางที่ "fail-safe" สำหรับ presence system มากกว่าการพึ่ง disconnect event อย่างเดียว

---

ก่อนหน้า: [บทที่ 3 — CDN, High Availability และ Disaster Recovery](03-cdn-ha-dr.md) | ถัดไป: [บทที่ 5 — Case Study: E-commerce](05-case-study-ecommerce.md)
