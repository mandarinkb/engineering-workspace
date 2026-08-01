# บทที่ 3: CQRS และ Event Sourcing

CQRS และ Event Sourcing มักถูกพูดถึงคู่กันจนหลายคนเข้าใจผิดว่าเป็นเรื่องเดียวกัน หรือต้องใช้คู่กันเสมอ — ความจริงคือทั้งสองเป็น**การตัดสินใจสถาปัตยกรรมที่แยกจากกันโดยสิ้นเชิง** แต่ละตัวมีปัญหาของตัวเองที่แก้ และมีต้นทุนความซับซ้อนของตัวเองที่ต้องจ่าย บทนี้จะอธิบายแต่ละตัวแยกกันก่อน แล้วค่อยดูว่าทำไมถึงมักเจอด้วยกัน

## 3.1 CQRS (Command Query Responsibility Segregation)

### ปัญหาของ Model เดียวที่ทำทั้งอ่านและเขียน

ระบบทั่วไปมักใช้ **model เดียวกัน** ทั้งตอนเขียนข้อมูล (command: create/update/delete) และตอนอ่านข้อมูล (query: read) — เช่น table `orders` เดียวที่ใช้ทั้งตอน insert order ใหม่ และตอน query แสดงหน้า dashboard สรุปยอดขาย

ปัญหาเกิดเมื่อ**รูปแบบการอ่านกับการเขียนต่างกันมาก**:

- การเขียนต้องการ **normalized structure** ที่รักษา consistency ได้ง่าย (แยก table ตาม entity, ใช้ foreign key, validate constraint)
- การอ่านมักต้องการ **denormalized/aggregated view** ที่ query เร็ว (เช่น "ยอดขายรวมต่อวันแยกตามสินค้า" ที่ต้อง join/group จากหลาย table ทุกครั้งถ้าใช้ model เดียวกัน)

ยิ่งระบบมี read pattern หลากหลาย (dashboard, report, search, API สำหรับ mobile app) การพยายามให้ table เดียวรองรับทุกแบบพร้อมกันจะทำให้ query ซับซ้อนขึ้นเรื่อยๆ และ index ที่เพิ่มมาเพื่อช่วย read บางแบบก็ไปถ่วง write performance ของอีกฝั่ง

### แนวคิดของ CQRS

**CQRS** แยก model ออกเป็นสองฝั่งอย่างชัดเจน:

- **Write Model (Command side)** — รับผิดชอบ business logic, validation, consistency ตอนเขียนข้อมูล ออกแบบตาม domain (เชื่อมกับแนวคิด DDD ใน [Volume 4 — Golang](../04-golang/README.md))
- **Read Model (Query side)** — เก็บข้อมูลในรูปแบบที่ optimize สำหรับการอ่านโดยเฉพาะ (denormalized, pre-aggregated, อาจอยู่คนละฐานข้อมูลไปเลย เช่น write ไป PostgreSQL แต่ read ผ่าน Elasticsearch สำหรับ full-text search)

```
        Command (write)                    Query (read)
             │                                   │
             ▼                                   ▼
     ┌───────────────┐   sync ผ่าน event    ┌───────────────┐
     │  Write Model   │ ────────────────►   │  Read Model    │
     │  (normalized)  │   (มักเป็น async)    │  (denormalized)│
     └───────────────┘                       └───────────────┘
```

Write model เขียนข้อมูล แล้ว sync ไปยัง read model ผ่าน event (มักเป็นแบบ async) — จุดสำคัญคือ read model จึง**ไม่ consistent แบบทันที (eventually consistent)** กับ write model เสมอไป มี delay สั้นๆ ระหว่างเขียนกับที่ read model อัปเดตตาม

### เมื่อไหร่คุ้มค่า เมื่อไหร่ overkill

| คุ้มค่าเมื่อ | overkill เมื่อ |
|---|---|
| Read-heavy อย่างชัดเจน (อ่านมากกว่าเขียนหลายเท่า) | Read/write ratio ใกล้เคียงกัน หรือ CRUD ง่ายๆ ไม่ซับซ้อน |
| รูปแบบการอ่านกับการเขียนต่างกันมาก (เช่น เขียนทีละ transaction แต่ต้องอ่านเป็น aggregate report) | Query ปัจจุบันใช้ table เดียวกันได้อย่างมีประสิทธิภาพอยู่แล้ว |
| ต้องการ scale ฝั่งอ่านกับฝั่งเขียนแยกอิสระจากกัน (เช่น read replica เยอะ, write node น้อย) | ทีมเล็ก ระบบยังไม่มีปัญหา performance จริง |
| ยอมรับ eventual consistency ได้ในทาง business (เช่น dashboard ช้ากว่าความจริงไม่กี่วินาทีไม่เป็นปัญหา) | ต้องการ strong consistency ทันที (เช่น เช็ค balance ก่อนหักเงินต้องแม่นยำ real-time) |

CQRS เพิ่มความซับซ้อนของระบบอย่างมีนัยสำคัญ — มีสอง model ที่ต้อง maintain, มี pipeline sync ที่ต้องดูแล (ถ้า sync ล้มเหลว read model จะเพี้ยนไปจาก write model โดยไม่มีใครรู้ทันที) และ debug ยากขึ้นเพราะข้อมูลที่ "เห็น" ตอนอ่านอาจไม่ตรงกับที่ "เขียน" ไปเมื่อครู่นี้ **อย่าใช้ CQRS เป็นค่าเริ่มต้น** — เริ่มจาก model เดียวก่อนเสมอ แล้วค่อยแยกเมื่อพิสูจน์แล้วว่ามีปัญหาจริงที่ CQRS แก้ได้คุ้มกับต้นทุนที่เพิ่ม

## 3.2 Event Sourcing

### เก็บ State ปัจจุบัน vs เก็บลำดับ Event

ฐานข้อมูลทั่วไปเก็บ **state ปัจจุบัน** — row ใน table `accounts` มี column `balance = 1500` เพียงค่าเดียว เมื่อมีการฝาก/ถอน ก็ `UPDATE` ทับค่าเดิม ประวัติว่า balance มาถึง 1500 ได้อย่างไร (ฝากกี่ครั้ง ถอนกี่ครั้ง) หายไปเลยเว้นแต่จะมี audit log แยกต่างหาก

**Event Sourcing** เปลี่ยนมุมมอง: แทนที่จะเก็บ state ปัจจุบัน ให้เก็บ**ลำดับเหตุการณ์ทั้งหมดที่เกิดขึ้น**แบบ append-only แล้ว **state ปัจจุบันคือผลลัพธ์ที่คำนวณได้จากการ replay event ทั้งหมดตามลำดับ**:

```
Event Store (append-only, ห้ามแก้/ลบของเดิม):
  1. AccountOpened(balance=0)
  2. MoneyDeposited(amount=1000)
  3. MoneyDeposited(amount=800)
  4. MoneyWithdrawn(amount=300)

Replay ทั้งหมดตามลำดับ ──► state ปัจจุบัน: balance = 0 + 1000 + 800 - 300 = 1500
```

ทุก event ที่เกิดขึ้นแล้ว**ไม่มีวันถูกแก้ไขหรือลบ** — ถ้าพบว่าการฝากครั้งที่ 2 ผิดพลาด ก็ไม่ไป `UPDATE` event เดิม แต่เพิ่ม event ใหม่เข้าไปแก้ไข (เช่น `DepositCorrected`) เหมือนบัญชีบัญชีในโลกจริงที่นักบัญชีไม่ลบรายการเก่า แต่ลงรายการปรับปรุงเพิ่มเข้าไปแทน

### Event Store

**Event Store** คือฐานข้อมูลที่ออกแบบมาสำหรับเก็บ event stream แบบ append-only โดยเฉพาะ (มี implementation เฉพาะทางอย่าง EventStoreDB หรือสร้างเองบนฐานข้อมูลทั่วไปที่บังคับ insert-only ก็ได้) คุณสมบัติสำคัญ:

- **Append-only** — ไม่มีการ `UPDATE`/`DELETE` row เดิม มีแต่ `INSERT` event ใหม่ต่อท้าย
- **Ordered per stream** — event ของ entity เดียวกัน (เช่น account เดียวกัน) ต้องเรียงลำดับตามที่เกิดขึ้นจริงเสมอ (แนวคิดคล้าย partition ordering ใน [Volume 12 — Kafka](../12-kafka/02-partition-and-consumer-group.md))
- **Immutable** — event ที่บันทึกแล้วคือความจริงทางประวัติศาสตร์ แก้ไม่ได้

### Replay เพื่อสร้าง State และปัญหาที่ตามมา

ข้อดีใหญ่ของ event sourcing คือได้ **audit trail แบบสมบูรณ์ฟรี** (รู้ว่าเกิดอะไรขึ้นบ้างกับ entity นี้ตั้งแต่ต้นจนจบ) และสามารถ**ตอบคำถามย้อนหลัง**ได้ เช่น "balance ของบัญชีนี้เมื่อวานตอนเที่ยงเท่าไหร่" ก็แค่ replay event จนถึงจุดเวลานั้น — สิ่งที่ระบบเก็บแค่ state ปัจจุบันทำไม่ได้เลยถ้าไม่มี snapshot ย้อนหลังแยกไว้ต่างหาก

แต่การ replay event **ทุกตัวตั้งแต่ต้น** ทุกครั้งที่ต้องการ state ปัจจุบันมีต้นทุนสูงขึ้นเรื่อยๆ ตามจำนวน event ที่สะสม — บัญชีที่มี event หนึ่งล้านรายการ ต้อง replay ทั้งล้านรายการทุกครั้งที่ต้องการรู้ balance ปัจจุบัน ช้าเกินกว่าจะใช้งานจริงได้

### Snapshotting

วิธีแก้คือ **snapshot** — บันทึก state ที่คำนวณไว้แล้ว ณ จุดเวลาหนึ่ง (เช่น ทุก 100 event) แล้วตอนต้องการ state ปัจจุบัน ให้เริ่มจาก snapshot ล่าสุด แล้ว replay เฉพาะ event ที่เกิดขึ้น**หลังจาก**snapshot นั้นเท่านั้น:

```
Event: 1 ... 100 [SNAPSHOT: balance=5000] 101 102 103 104 (ปัจจุบัน)

แทนที่จะ replay event 1-104 ทั้งหมด
──► เริ่มจาก snapshot (balance=5000) แล้ว replay แค่ event 101-104
```

Snapshot เป็นเพียง **optimization ด้าน performance** ไม่ใช่ source of truth — ถ้า snapshot หายหรือเสียหาย ระบบยังสร้าง state ที่ถูกต้องได้เสมอจากการ replay event ทั้งหมดตั้งแต่ต้น (แค่ช้ากว่า) ความจริงทั้งหมดของระบบยังคงอยู่ใน event log เท่านั้น

## 3.3 CQRS + Event Sourcing: เกี่ยวข้องกันแต่เป็นคนละการตัดสินใจ

ทั้งสอง pattern มักถูกใช้ร่วมกันเพราะ**เข้ากันได้ดีเป็นธรรมชาติ**: event ที่เกิดจาก write model ใน event sourcing เป็น "แหล่งข้อมูล" ที่สมบูรณ์แบบสำหรับ build read model หลายแบบใน CQRS (แค่ subscribe event stream แล้ว project ออกมาเป็น read model รูปแบบต่างๆ ตามต้องการ) — นี่คือเหตุผลที่คนมักเห็นสองคำนี้อยู่ด้วยกันในบทความ/ระบบเดียวกันจนเข้าใจผิดว่าเป็นเรื่องเดียวกัน

แต่ในทางเทคนิค **แยกกันใช้ได้อิสระ**:

- **ทำ CQRS โดยไม่ทำ Event Sourcing** — write model ยังเก็บแค่ state ปัจจุบันแบบปกติ (ไม่มี event log) แต่ sync ไปยัง read model แยกต่างหากผ่านกลไกอื่น เช่น Change Data Capture (CDC) จับการเปลี่ยนแปลงจาก database log โดยตรง
- **ทำ Event Sourcing โดยไม่ทำ CQRS** — เก็บ state เป็น event log ทั้งหมด แต่ query ยังอ่านผ่าน model เดียวกัน (replay event แล้วอ่าน state ปัจจุบันตรงๆ ไม่ได้แยก read model ต่างหากไปในรูปแบบอื่น)

ทั้งคู่เป็นการตัดสินใจสถาปัตยกรรมที่**เพิ่มความซับซ้อนอย่างมีนัยสำคัญ** ต้องประเมินแยกกันว่าปัญหาที่แต่ละตัวแก้นั้นคุ้มกับต้นทุนที่ต้องจ่ายในระบบของตัวเองหรือไม่ ไม่ใช่หยิบมาใช้คู่กันเพราะเห็นคนอื่นทำ

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| ปัญหาที่ CQRS แก้ | model เดียวรองรับทั้ง read/write pattern ที่ต่างกันมากได้ไม่ดี |
| CQRS | แยก write model (normalized) กับ read model (denormalized) sync กันแบบ eventually consistent |
| เมื่อไหร่ใช้ CQRS | read-heavy, read/write shape ต่างกันมาก, ยอมรับ eventual consistency ได้ |
| ปัญหาที่ Event Sourcing แก้ | state ปัจจุบันอย่างเดียวไม่เก็บประวัติว่ามาถึงจุดนี้ได้อย่างไร |
| Event Sourcing | เก็บ event แบบ append-only, state ปัจจุบัน = ผลจากการ replay event ทั้งหมด |
| Snapshot | optimization ลดจำนวน event ที่ต้อง replay ไม่ใช่ source of truth |
| CQRS + Event Sourcing | เข้ากันได้ดีตามธรรมชาติแต่เป็นการตัดสินใจแยกกัน ใช้ตัวเดียวหรือทั้งคู่ก็ได้ |

## คำถามทบทวน

1. ยกตัวอย่างระบบที่ read pattern กับ write pattern ต่างกันมากจนควรพิจารณาใช้ CQRS
2. ทำไม read model ใน CQRS ถึง "eventually consistent" ไม่ใช่ "strongly consistent" กับ write model?
3. อธิบายว่า snapshot ใน event sourcing ช่วยแก้ปัญหาอะไร และทำไมถึงไม่ใช่ source of truth
4. ทำไม event ใน event sourcing ถึงห้ามแก้ไขหรือลบ ถ้าพบข้อมูลผิดพลาดควรทำอย่างไรแทน?
5. อธิบายว่าทำไม CQRS กับ Event Sourcing ถึงเป็นการตัดสินใจที่แยกจากกัน ยกตัวอย่างระบบที่ใช้ตัวใดตัวหนึ่งเพียงอย่างเดียว

---

ก่อนหน้า: [บทที่ 2 — Service Discovery](02-service-discovery.md) | ถัดไป: [บทที่ 4 — Saga, Retry, Circuit Breaker](04-saga-and-resilience.md)
