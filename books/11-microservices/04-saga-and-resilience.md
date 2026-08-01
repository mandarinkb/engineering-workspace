# บทที่ 4: Saga, Retry, และ Circuit Breaker

บทนี้พูดถึงสามเรื่องที่เกี่ยวข้องกันในหัวข้อใหญ่เดียวคือ "ทำอย่างไรให้ระบบที่กระจายอยู่หลาย service ยังทำงานถูกต้องและอยู่รอดได้ เมื่อบาง service ล้มเหลวหรือช้า" — Saga จัดการ transaction ที่กระจายข้าม service, Retry จัดการความล้มเหลวชั่วคราว, Circuit Breaker ป้องกันความล้มเหลวลุกลาม

## 4.1 Saga Pattern

### ทำไมไม่ใช้ 2PC

Transaction แบบดั้งเดิมในฐานข้อมูลเดียวใช้ ACID รับประกัน all-or-nothing ได้ง่าย แต่เมื่อ transaction หนึ่งต้องข้าม**หลาย service ที่แต่ละตัวมีฐานข้อมูลของตัวเอง** (เช่น สร้าง order ที่ต้องหักสต็อกใน inventory service และตัดเงินใน payment service) จะทำ ACID transaction เดียวข้าม service ไม่ได้โดยตรง

**Two-Phase Commit (2PC)** คือความพยายามแก้ปัญหานี้แบบ synchronous — มี coordinator ถามทุก service ว่า "พร้อม commit ไหม" (phase 1: prepare) แล้วถ้าทุกตัวตอบพร้อม ค่อยสั่ง commit จริงพร้อมกัน (phase 2: commit) แต่ 2PC มีปัญหาสำคัญในโลกจริง: ทุก service ต้อง**lock resource ค้างไว้**ระหว่างรอ phase 2 (ลด throughput อย่างมาก), และถ้า coordinator ล่มระหว่างสอง phase บาง service อาจค้างอยู่ในสถานะ "รอ" ตลอดไป — ด้วยเหตุผลเหล่านี้ 2PC แทบไม่ถูกใช้ในระบบ microservices จริง

### Saga คืออะไร

**Saga** แก้ปัญหานี้ด้วยแนวคิดต่างออกไป: แทนที่จะพยายามทำ transaction เดียวข้าม service แบบ atomic ให้แตก transaction ใหญ่เป็น**ลำดับของ local transaction เล็กๆ** ที่แต่ละตัว commit ทันทีใน service ของตัวเอง ถ้าขั้นตอนใดขั้นตอนหนึ่งล้มเหลวภายหลัง ให้รัน **compensating transaction** (transaction ที่ "ย้อนกลับ" ผลของขั้นตอนก่อนหน้าที่สำเร็จไปแล้ว) แทนการ rollback แบบ ACID

มีสองรูปแบบหลักในการ implement Saga: **Choreography-based** และ **Orchestration-based**

### Choreography-based Saga

แต่ละ service **ฟัง event จาก service อื่น**และตัดสินใจทำงานของตัวเองต่อไปเอง โดยไม่มีตัวกลางควบคุม — เหมือนการเต้นรำที่ทุกคนรู้จังหวะและ cue ของตัวเองโดยไม่ต้องมีผู้กำกับ:

```
Order Service          Payment Service         Inventory Service
     │                        │                        │
     │ OrderCreated ──────────►                         │
     │                        │ (จ่ายเงิน)               │
     │                        │ PaymentSucceeded ────────►
     │                        │                        │ (หักสต็อก)
     │                        │                        │ StockReserved
     │◄───────────────────────────────────────────────┘
     │ (order status = confirmed)

กรณีล้มเหลว: ถ้า InventoryService หักสต็อกไม่สำเร็จ (สินค้าหมด)
     │                        │                        │
     │                        │                        │ StockReservationFailed ──►
     │                        │◄── ฟัง event นี้ ──────────┘
     │                        │ (คืนเงินอัตโนมัติ)
     │                        │ PaymentRefunded ─────────►
     │◄── ฟัง event นี้ ───────┘
     │ (order status = cancelled)
```

ข้อดี: ไม่มี single point of failure ที่ตัวกลาง, แต่ละ service เป็นอิสระต่อกันสูง (loose coupling)
ข้อเสีย: เมื่อจำนวนขั้นตอนเพิ่มขึ้น การตามลำดับ event ทั้งหมดว่า "ตอนนี้ saga อยู่ขั้นตอนไหน" ทำได้ยากขึ้นเรื่อยๆ (logic กระจายอยู่ในหลาย service ไม่มีที่เดียวที่เห็นภาพรวมทั้งหมด) debug ยากเมื่อเกิดปัญหา

### Orchestration-based Saga

มี **orchestrator** ตัวกลางที่รู้ลำดับขั้นตอนทั้งหมดของ saga และ**สั่งการ**แต่ละ service ทีละขั้นตอนอย่างชัดเจน รวมถึงเป็นคนตัดสินใจสั่ง compensating transaction เมื่อขั้นตอนใดล้มเหลว:

```
                    ┌───────────────────┐
                    │   Orchestrator     │
                    └─────────┬──────────┘
        1.สั่งจอง         2.สั่งจ่ายเงิน        3.สั่งหักสต็อก
              │                │                  │
              ▼                ▼                  ▼
        Order Service   Payment Service   Inventory Service
              │                │                  │
              └────► ตอบผล ────┴────► ตอบผล ───────┘
                              │
                    Orchestrator ตัดสินใจขั้นตอนถัดไป
                    หรือสั่ง compensate ถ้าล้มเหลว

กรณีล้มเหลวที่ขั้นตอน 3 (หักสต็อกไม่สำเร็จ):
   Orchestrator สั่ง compensate ย้อนกลับตามลำดับย้อนกลับ:
   3'. (ไม่ต้องทำอะไร เพราะขั้นตอน 3 ไม่สำเร็จตั้งแต่แรก)
   2'. สั่ง Payment Service คืนเงิน (compensating transaction)
   1'. สั่ง Order Service เปลี่ยนสถานะเป็น cancelled
```

ข้อดี: logic ทั้งหมดอยู่ที่เดียว เห็นภาพรวมและ debug ง่ายกว่า, จัดการ error handling/compensation ที่ซับซ้อนได้ชัดเจนกว่า
ข้อเสีย: orchestrator กลายเป็น component สำคัญที่ต้องดูแล (แม้จะไม่ใช่ single point of failure ถ้าออกแบบให้ทนต่อการล่มได้)

ในทางปฏิบัติ **orchestration-based เป็นที่นิยมกว่าเมื่อ saga มีหลายขั้นตอนหรือ logic ซับซ้อน** เพราะความชัดเจนของ logic ที่รวมอยู่ที่เดียวคุ้มค่ากว่าความเป็นอิสระของ choreography — และแทนที่จะเขียน orchestrator เองตั้งแต่ต้น (ซึ่งต้องจัดการเรื่อง state, retry, ความทนทานต่อการล่มของตัว orchestrator เองด้วย) ระบบจริงมักใช้ **workflow orchestration engine** ที่ออกแบบมาสำหรับงานนี้โดยเฉพาะ (เชื่อมกับ [Volume 13 — Temporal](../13-temporal/README.md) ซึ่งจะอธิบายรายละเอียดว่า engine เหล่านี้รับประกันความทนทาน — durable execution — ให้ orchestrator เองได้อย่างไร)

### Compensating Transaction

**Compensating transaction** ไม่ใช่ "rollback" แบบ ACID (ที่ไม่มีอะไรเคยเกิดขึ้นเลย) แต่คือ**การกระทำใหม่**ที่ลบล้างผลของการกระทำก่อนหน้าในทางธุรกิจ เช่น การคืนเงิน (ไม่ใช่การทำให้ "ไม่เคยจ่ายเงิน") หรือการปลดจองสต็อก (ไม่ใช่การทำให้ "ไม่เคยหักสต็อก") — ทุกคนที่เกี่ยวข้อง (รวมถึง log/audit trail) ยังเห็นว่าเหตุการณ์ทั้งสองเกิดขึ้นจริง เพียงแต่ผลสุทธิกลับสู่สถานะที่ถูกต้อง สิ่งนี้ทำให้การออกแบบ compensating transaction ต้องคิดล่วงหน้าสำหรับทุกขั้นตอนที่มี side effect ว่า "ถ้าต้องย้อนกลับขั้นตอนนี้ ต้องทำอะไรบ้าง" ตั้งแต่ตอนออกแบบ saga ไม่ใช่คิดทีหลัง

## 4.2 Retry

### Backoff Strategies

เมื่อ request ไปยัง service อื่นล้มเหลว (timeout, connection refused, 5xx) คำถามแรกคือควร retry หรือไม่ (ดูเรื่อง idempotency ใน [บทที่ 5](05-idempotency-and-tracing.md) ประกอบ) และถ้า retry ควรรอนานแค่ไหนก่อน retry แต่ละครั้ง:

- **Fixed delay** — รอเวลาคงที่ทุกครั้ง (เช่น รอ 1 วินาทีเสมอ) ง่ายที่สุดแต่ถ้า downstream กำลังโหลดหนักอยู่แล้ว การยิงซ้ำถี่ๆ ด้วยช่วงเวลาเท่าเดิมยิ่งซ้ำเติมปัญหา
- **Exponential backoff** — เพิ่มเวลารอเป็นทวีคูณทุกครั้งที่ retry ล้มเหลว (เช่น 1s, 2s, 4s, 8s, ...) ให้เวลา downstream ฟื้นตัวมากขึ้นเรื่อยๆ ถ้ายังไม่ฟื้น
- **Exponential backoff + jitter** — เพิ่ม randomness เล็กน้อยเข้าไปในเวลารอ (เช่น สุ่มในช่วง 0-4s แทนที่จะรอ 4s เป๊ะ) เพื่อแก้ปัญหาที่ exponential backoff ธรรมดายังไม่ได้แก้

### ทำไม Jitter ถึงสำคัญ

ลองนึกภาพ client 1,000 ตัวยิง request ไปยัง service เดียวกันพร้อมกันแล้วล้มเหลวพร้อมกันหมด (เช่น service เพิ่ง restart) ถ้าทุกตัวใช้ exponential backoff แบบ**ไม่มี jitter** พวกมันจะ retry**พร้อมกันเป๊ะ**ทุกรอบ (รอ 1s เท่ากัน retry พร้อมกัน, รอ 2s เท่ากัน retry พร้อมกันอีก) กลายเป็นคลื่น request กระแทกเข้า service ที่เพิ่งจะฟื้นตัวเป็นระลอกๆ ซ้ำแล้วซ้ำเล่า — ปรากฏการณ์นี้เรียกว่า **thundering herd** (ฝูงสัตว์กระทืบพร้อมกัน) ซึ่งเป็นปัญหาเดียวกันในหลักการกับ cache stampede ที่เกิดตอน cache หมดอายุพร้อมกันจำนวนมาก (เชื่อมกับ [Volume 6 — Redis](../06-redis/README.md)) เพียงแต่มุมมองนี้เกิดจาก retry ไม่ใช่ cache expiry

การใส่ **jitter** (สุ่มกระจายเวลา retry ของแต่ละ client ให้ไม่ตรงกัน) ทำให้ระลอกของ request กระจายตัวออกไปตามเวลาแทนที่จะกระแทกพร้อมกันเป็นก้อนเดียว ทำให้ downstream service มีโอกาสฟื้นตัวได้จริงแทนที่จะถูกคลื่น retry ทับซ้ำไปเรื่อยๆ

### Max Retry และ Give Up

Retry ต้องมีขอบเขต — กำหนด max attempt (เช่น retry ไม่เกิน 5 ครั้ง) หรือ max total time (เช่น ยอมรอ retry รวมไม่เกิน 30 วินาที) แล้ว**ยอมแพ้** (fail fast) ไม่ใช่ retry ไม่มีที่สิ้นสุด — ถ้า retry ไปเรื่อยๆ โดยไม่มีขอบเขต จะเป็นการกักตัว resource ของ client เอง (thread, connection) ไว้รอผลลัพธ์ที่อาจไม่มีวันสำเร็จ

## 4.3 Circuit Breaker

### ปัญหา Cascading Failure

Retry ช่วยรับมือความล้มเหลว**ชั่วคราว**ได้ดี แต่ถ้า downstream service ล่มจริงจัง (ไม่ใช่แค่สะดุดชั่วคราว) การให้ client retry ไปเรื่อยๆ ไม่ได้ช่วยอะไร กลับซ้ำเติมปัญหา: request ที่ค้างรอ timeout จำนวนมากจะกิน thread/connection ของ client เองจนหมด ทำให้ client เองก็เริ่มตอบสนองช้าลงหรือล่มตาม แล้ว service ที่เรียก client ตัวนี้ต่อก็ล่มตามเป็นทอดๆ — ปรากฏการณ์นี้เรียกว่า **cascading failure** (ความล้มเหลวลุกลามข้าม service เป็นลูกโซ่)

### State Machine: Closed / Open / Half-Open

**Circuit Breaker** คือกลไกที่ห่อหุ้ม call ไปยัง downstream service แล้วคอยติดตามอัตราความล้มเหลว เมื่อล้มเหลวเกินเกณฑ์ที่ตั้งไว้ จะ**หยุดยิง request ออกไปเลยชั่วคราว** (fail fast ทันทีโดยไม่ต้องรอ timeout จริง) ให้ downstream มีโอกาสฟื้นตัว แนวคิดยืมมาจาก circuit breaker ไฟฟ้าจริงที่ตัดวงจรเมื่อกระแสเกิน เพื่อป้องกันความเสียหายลุกลาม

```
        ┌────────────────────────────────────────────┐
        │                                              │
        ▼                                              │
  ┌───────────┐  failure rate เกิน threshold   ┌───────────┐
  │  CLOSED    │ ─────────────────────────────► │   OPEN     │
  │ (ยิง request│                                 │ (ไม่ยิง     │
  │  ตามปกติ)  │ ◄─────────────────────────────  │  request   │
  └───────────┘   ทดสอบสำเร็จ (call ผ่านหมด)      │  เลย, fail │
        ▲                                        │  fast ทันที)│
        │                                        └─────┬──────┘
        │                                              │
        │              รอครบเวลาที่กำหนด (timeout)         │
        │                                              ▼
        │                                     ┌───────────────┐
        └──────── ทดสอบล้มเหลว ─────────────── │  HALF-OPEN     │
                                               │ (ปล่อย request  │
                                               │  จำนวนน้อยไป     │
                                               │  ทดสอบ)         │
                                               └───────────────┘
```

- **Closed** — สถานะปกติ ปล่อยให้ request ผ่านไปยัง downstream ตามปกติ พร้อมนับอัตราความสำเร็จ/ล้มเหลว ถ้าอัตราล้มเหลวเกิน threshold (เช่น ล้มเหลวเกิน 50% ใน 10 request ล่าสุด) เปลี่ยนไปสถานะ **Open**
- **Open** — หยุดยิง request ไปยัง downstream ทันที ตอบ error กลับทันทีโดยไม่ต้องรอ (fail fast) ลด load ที่ยิงไปหา downstream ที่กำลังมีปัญหาให้เหลือศูนย์ ให้เวลามันฟื้นตัว หลังจากผ่านไปครบเวลาที่กำหนด (เช่น 30 วินาที) เปลี่ยนไปสถานะ **Half-Open**
- **Half-Open** — ปล่อย request จำนวนน้อยผ่านไปทดสอบว่า downstream ฟื้นแล้วหรือยัง ถ้าสำเร็จ กลับไปสถานะ **Closed** (ทำงานปกติ) ถ้ายังล้มเหลว กลับไปสถานะ **Open** อีกครั้ง (รอรอบถัดไป)

Circuit breaker ทำงานคู่กับ retry ได้ดี: retry จัดการความล้มเหลวชั่วคราวระดับ request เดี่ยว ส่วน circuit breaker จัดการความล้มเหลวระดับ downstream ทั้งตัวที่ยืดเยื้อกว่า — เมื่อ circuit เปิด (Open) client ควรหยุด retry ไปด้วย ไม่ใช่ retry ต่อไปเรื่อยๆ ทั้งที่ circuit บอกอยู่แล้วว่า downstream ไม่พร้อม

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| ทำไมไม่ใช้ 2PC | ต้อง lock resource ค้างรอ phase 2 ลด throughput, coordinator ล่มทำให้ค้างได้ |
| Saga | แตก transaction ใหญ่เป็น local transaction เล็กๆ ต่อ service ใช้ compensating transaction แทน rollback |
| Choreography Saga | แต่ละ service ฟัง event กันเอง ไม่มีตัวกลาง, loose coupling แต่ debug ยากเมื่อขั้นตอนเยอะ |
| Orchestration Saga | มี orchestrator สั่งการชัดเจน เห็นภาพรวมง่ายกว่า นิยมใช้ workflow engine เช่น Temporal |
| Compensating transaction | การกระทำใหม่ที่ลบล้างผลทางธุรกิจ ไม่ใช่ทำให้ไม่เคยเกิดขึ้น |
| Backoff strategy | fixed delay ง่ายแต่หยาบ, exponential backoff รอนานขึ้นเรื่อยๆ |
| Jitter | สุ่มกระจายเวลา retry ป้องกัน thundering herd จาก client จำนวนมาก retry พร้อมกัน |
| Circuit Breaker | Closed (ปกติ) → Open (fail fast ทันที) → Half-Open (ทดสอบ) ป้องกัน cascading failure |

## คำถามทบทวน

1. ทำไม 2PC ถึงไม่เหมาะกับระบบ microservices ที่ต้องการ throughput และ availability สูง?
2. เปรียบเทียบ choreography-based กับ orchestration-based saga ในแง่ความชัดเจนของ logic และความเป็นอิสระของ service
3. อธิบายว่า compensating transaction ต่างจาก database rollback อย่างไร ยกตัวอย่างจากสถานการณ์ order+payment+inventory
4. ทำไม exponential backoff อย่างเดียว (ไม่มี jitter) ยังก่อให้เกิด thundering herd ได้? jitter ช่วยแก้ตรงไหน
5. อธิบาย state machine ของ circuit breaker ทั้งสามสถานะ และบอกว่าทำไมสถานะ Half-Open ถึงจำเป็น (ทำไมไม่กลับจาก Open ไป Closed ตรงๆ)

---

ก่อนหน้า: [บทที่ 3 — CQRS และ Event Sourcing](03-data-patterns.md) | ถัดไป: [บทที่ 5 — Idempotency และ Trace ID](05-idempotency-and-tracing.md)
