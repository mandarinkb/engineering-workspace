# บทที่ 5: Case Study — E-commerce

นี่คือบทสังเคราะห์สุดท้ายของหลักสูตร ที่นำความรู้จากทุก Volume มาประกอบเป็นระบบเดียว — ระบบ e-commerce ขนาดใหญ่ที่ต้องรองรับ catalog ขนาดมหาศาล, inventory ที่ต้องแม่นยำภายใต้การแข่งขันซื้อพร้อมกัน, payment ที่ผิดพลาดไม่ได้, และการแจ้งเตือนที่ทันเวลา — ทั้งหมดต้อง trace ได้ว่า request หนึ่งวิ่งผ่านอะไรบ้าง **บทนี้คือ design document ที่การ implement จริงที่ [projects/ecommerce/](../../projects/ecommerce/README.md) ต้องยึดตาม**

## 5.1 ภาพรวมสถาปัตยกรรม

```
                          ┌─────────────────┐
Client ──HTTPS──►│   API Gateway    │ (APISIX)
                          └────────┬─────────┘
             ┌───────────┬─────────┼─────────┬─────────────┬──────────────┐
             ▼           ▼         ▼         ▼             ▼              ▼
        ┌────────┐ ┌──────────┐ ┌───────┐ ┌────────┐ ┌────────────┐ ┌──────────┐
        │Catalog │ │Inventory │ │ Order │ │Payment │ │Notification│ │Observ.   │
        │Service │ │ Service  │ │Service│ │Service │ │  Service   │ │Stack     │
        └───┬────┘ └────┬─────┘ └───┬───┘ └───┬────┘ └─────┬──────┘ └────┬─────┘
            │            │           │         │            │             │
      OpenSearch    PostgreSQL  PostgreSQL  PostgreSQL     Kafka      Traces/Metrics/Logs
      (search)      (stock)     +Temporal   (ledger)     (event bus)    (ทุก service ส่งเข้ามา)
```

แต่ละ service ในบทนี้ถูกออกแบบให้สอดคล้องกับหลักที่เรียนมาตลอดหลักสูตร ไม่ใช่แค่เลือกเทคโนโลยีตามความนิยม — ทุกการเลือกมีเหตุผลผูกกับ trade-off ที่อธิบายไว้ในบทก่อนหน้า

## 5.2 Catalog Service — Search

Catalog service รับผิดชอบข้อมูลสินค้า (ชื่อ, รายละเอียด, ราคา, หมวดหมู่) และการค้นหา การค้นหาสินค้าด้วยข้อความอิสระ (full-text search) ไม่เหมาะกับการ query ผ่าน relational database โดยตรง (ทำ relevance ranking, fuzzy matching, faceted search ได้ไม่ดี) จึงใช้ **OpenSearch** เป็น search engine แยกต่างหาก (เครื่องมือตัวเดียวกับที่ [Volume 14 — Logging](../14-observability/02-logging.md) ใช้เก็บ log แบบ full-text search ได้ — หลักการ inverted index เดียวกันที่ทำให้ค้นหาข้อความในกองข้อมูลมหาศาลได้เร็ว ถูกนำมาใช้ค้นหาสินค้าในบริบทนี้)

Catalog service เขียนข้อมูลสินค้าลง PostgreSQL เป็น source of truth ก่อน แล้ว sync (ผ่าน event หรือ batch job) ไปยัง OpenSearch index สำหรับการค้นหา — เป็นตัวอย่างของ pattern ที่ **read path กับ write path ใช้ storage คนละตัว** เพื่อให้แต่ละฝั่งเหมาะกับงานของตัวเองที่สุด

## 5.3 Inventory Service — Overselling Race Condition

นี่คือปัญหาคลาสสิกที่สุดของระบบ e-commerce และเป็นตัวอย่างที่ดีที่สุดของ race condition ในระบบจริง

### Timeline ของปัญหา

สมมติสินค้าชิ้นสุดท้ายเหลือ `stock = 1` มีลูกค้า 2 คนกดซื้อพร้อมกันเป๊ะๆ:

```
เวลา    Request A (ลูกค้า 1)              Request B (ลูกค้า 2)
─────   ────────────────────────          ────────────────────────
t1      SELECT stock FROM product
        WHERE id = 42;  → ได้ 1
                                            SELECT stock FROM product
t2                                          WHERE id = 42;  → ได้ 1  (ยังไม่เห็น update จาก A เพราะ A ยังไม่ commit)
t3      ตรวจสอบใน app: 1 > 0 → ผ่าน
                                            ตรวจสอบใน app: 1 > 0 → ผ่าน
t4      UPDATE product SET stock = 0
        WHERE id = 42;  (1 - 1)
                                            UPDATE product SET stock = 0
t5                                          WHERE id = 42;  (1 - 1)  ← ผิด! ควรเป็น -1 หรือถูกปฏิเสธ

ผลลัพธ์: ทั้ง A และ B คิดว่าซื้อสำเร็จ แต่สินค้ามีแค่ 1 ชิ้น → oversell
```

ต้นเหตุคือ **"read stock แล้วเช็คใน application, ค่อย update"** เป็นสาม step แยกกัน (read-check-write) ที่ไม่ atomic — ระหว่าง step เหล่านี้ request อีกตัวแทรกเข้ามาอ่านค่าเดิมได้เสมอ (classic **check-then-act race condition**)

### ทางแก้ที่ 1: Pessimistic Row Lock

ใช้ `SELECT ... FOR UPDATE` ล็อกแถวนั้นไว้ตั้งแต่ step อ่าน ทำให้ transaction ที่สองต้อง**รอ**จนกว่า transaction แรกจะ commit/rollback เสร็จก่อน — กลไก locking นี้อยู่ใน [Volume 5 — MVCC และ Transaction Isolation](../05-postgresql/03-transactions-and-mvcc.md)

```sql
BEGIN;
SELECT stock FROM product WHERE id = 42 FOR UPDATE;  -- ล็อกแถวนี้ทันที
-- Request B ที่มาถึงตอนนี้ต้องรอจนกว่า A จะ COMMIT/ROLLBACK
UPDATE product SET stock = stock - 1 WHERE id = 42;
COMMIT;  -- ปลดล็อก, B ได้อ่านค่าที่อัปเดตแล้ว (stock = 0) และจะถูกปฏิเสธถ้าเช็คแล้วไม่พอ
```

**ข้อดี** — เข้าใจง่าย, รับประกันความถูกต้อง 100%
**ข้อเสีย** — request ที่ต้องรอ lock ทำให้ throughput ลดลงถ้ามีคนแย่งซื้อสินค้าตัวเดียวกันพร้อมกันจำนวนมาก (เช่น flash sale) — lock กลายเป็นคอขวด

### ทางแก้ที่ 2: Atomic Conditional Decrement

หลีกเลี่ยงการแยก read กับ write ออกจากกัน โดยรวมเป็น statement เดียวที่มีเงื่อนไขในตัว:

```sql
UPDATE product
SET stock = stock - 1
WHERE id = 42 AND stock > 0
RETURNING stock;
-- ถ้า UPDATE ไม่กระทบแถวไหนเลย (0 rows affected) แปลว่า stock ไม่พอ → ปฏิเสธ order
```

Database รับประกันว่า `UPDATE` แต่ละครั้งเป็น atomic operation ในตัวเอง — request สองตัวที่ยิงเข้ามาพร้อมกันจะถูกจัดคิวโดย database engine เองโดยอัตโนมัติ (ไม่มีช่องให้ทั้งคู่อ่านค่าเดิมพร้อมกันแบบ pessimistic lock version ที่ต้องเขียน `FOR UPDATE` เอง) วิธีนี้**ไม่ต้อง lock แถวค้างไว้ระหว่างรอ business logic อื่นทำงาน** (ต่างจาก pessimistic lock ที่ล็อกค้างจนกว่า transaction ทั้งก้อนจะ commit) จึงมัก throughput ดีกว่าภายใต้ contention สูง

### เลือกแบบไหน

| | Pessimistic Row Lock | Atomic Conditional Decrement |
|---|---|---|
| ความซับซ้อน | ตรงไปตรงมา แต่ต้องระวัง lock ordering (deadlock) | ต้องออกแบบ query ให้ atomic ตั้งแต่ต้น |
| Throughput ภายใต้ contention สูง | ลดลง (request รอคิว) | ดีกว่า (ไม่มี lock ค้างระหว่างรอ logic อื่น) |
| เหมาะกับ | flow ที่มี business logic หลายขั้นตอนต้องทำระหว่างถือ lock | flow ง่ายๆ ที่ตัดสินใจจบในคำสั่งเดียว เช่น decrement stock ล้วนๆ |

ระบบจริงจำนวนมากใช้ **atomic conditional decrement** เป็นค่าเริ่มต้นสำหรับ hot path (checkout ปกติ) และสำรอง pessimistic lock ไว้เฉพาะ flow ที่ซับซ้อนกว่า (เช่น ต้อง reserve stock ข้ามหลายรายการพร้อมกันในคำสั่งเดียว)

## 5.4 Order Service — Saga/CQRS Orchestrator

Order service คือจุดศูนย์กลางที่ประสาน Catalog, Inventory, Payment, Notification เข้าด้วยกันเป็น flow เดียว (create order → reserve stock → charge payment → confirm → notify) — เป็นตัวอย่างจริงของ Saga pattern ที่อธิบายหลักการไว้ใน [Volume 11 — Data Patterns](../11-microservices/03-data-patterns.md) และเขียนเป็น durable workflow จริงด้วย [Volume 13 — Workflow และ Activity](../13-temporal/01-workflow-and-activity.md)

Order service มักออกแบบตาม **CQRS** (Command Query Responsibility Segregation) — แยก write model (การสร้าง/อัปเดต order ผ่าน workflow ที่ซับซ้อน หลาย step) ออกจาก read model (การแสดงรายการ order ของลูกค้า ที่ต้องการอ่านเร็วและบ่อยกว่ามาก) เพื่อให้แต่ละฝั่ง optimize ได้อย่างอิสระโดยไม่ชนกัน — write model เน้นความถูกต้องของ state machine, read model เน้น query ที่เร็วและอาจ denormalize ข้อมูลไว้ล่วงหน้า

Order workflow แต่ละ step (reserve inventory, charge payment, send confirmation) คือ **Activity** ใน Temporal ที่มี retry policy ของตัวเอง ส่วนลำดับ/เงื่อนไข/compensation ทั้งหมดถูกกำหนดไว้ใน **Workflow definition** เดียว ทำให้ logic ของ Saga ทั้งก้อนอยู่ในที่เดียวที่อ่านตามลำดับได้ ไม่กระจัดกระจายเป็น event handler หลายจุดแบบ choreography-based saga

## 5.5 Payment Service

รายละเอียดการออกแบบ payment (idempotency, double-entry ledger, saga compensation เมื่อ charge ล้มเหลวกลางทาง) อธิบายไว้ครบแล้วใน [บทที่ 4 ของ Volume นี้](04-case-study-payment-chat.md) — ในบริบท e-commerce, Payment service ถูกเรียกเป็น Activity หนึ่งภายใน Order workflow เท่านั้น (ไม่ได้ผูกกับ order logic โดยตรง) ทำให้ payment service เป็น service ที่ reusable ข้าม flow อื่นได้ด้วย (เช่น refund, subscription renewal) โดยไม่ต้องเขียน payment logic ซ้ำ

## 5.6 Notification Service — Event-Driven ผ่าน Kafka

เมื่อ order เปลี่ยนสถานะ (สร้างสำเร็จ, ชำระเงินสำเร็จ, จัดส่งแล้ว) ลูกค้าต้องได้รับแจ้งเตือน (email, SMS, push notification) Notification service **ไม่ควรถูกเรียกตรงแบบ synchronous** จาก Order service เพราะจะทำให้ order flow ต้องรอ notification provider ภายนอกตอบกลับก่อนถึงจะถือว่า order เสร็จ (ผูก latency ของ critical path เข้ากับ service ที่ไม่ critical)

แนวทางที่ถูกต้องคือให้ Order service **publish event** (เช่น `order.confirmed`) ไปยัง [Kafka topic](../12-kafka/01-broker-and-topic.md) แล้วให้ Notification service เป็น consumer ที่ subscribe topic นั้นแยกต่างหาก:

```
Order Service ──publish "order.confirmed"──► Kafka Topic ──consume──► Notification Service ──► Email/SMS/Push
```

**ข้อดีของแนวทางนี้** — Order service ไม่ต้องรู้จักหรือรอ Notification service เลย (decoupling สมบูรณ์), ถ้า Notification service ล่มชั่วคราว event ยังอยู่ใน Kafka รอให้ consumer กลับมาอ่านต่อได้ (durability เดียวกับที่อธิบายไว้ในบทที่ 4 สำหรับ chat fan-out), และเพิ่ม consumer ใหม่ในอนาคต (เช่น analytics service ที่อยากรู้ทุกครั้งที่มี order confirmed) ทำได้โดยไม่ต้องแก้โค้ด Order service เลยแม้แต่บรรทัดเดียว

## 5.7 API Gateway — จุดเข้าระบบเดียว

Client ทุกตัว (web, mobile app) เข้าระบบผ่าน [APISIX](../10-apisix/README.md) เป็นจุดเดียว ทำหน้าที่เป็น reverse proxy/load balancer (ตามหลักการใน [Volume 2](../02-network/05-reverse-proxy-load-balancer.md)) พร้อม feature เพิ่มเติมที่จำเป็นสำหรับระบบ production จริง: authentication/authorization ก่อนส่งต่อไปยัง service ภายใน, rate limiting ป้องกัน abuse, และ routing request ไปยัง service ที่ถูกต้องตาม path (`/catalog/*` → Catalog service, `/orders/*` → Order service เป็นต้น)

การมี gateway ชั้นเดียวยังทำให้ **cross-cutting concern** (auth, rate limit, logging, TLS termination) ถูกจัดการรวมศูนย์ที่จุดเดียว แทนที่ทุก service ภายในต้อง implement ซ้ำๆ กันเอง

## 5.8 Observability — Trace ID เดียวข้ามทุก Service

Checkout request หนึ่งครั้งวิ่งผ่าน API Gateway → Order Service → Inventory Service → Payment Service → Notification Service (ผ่าน Kafka) — เมื่อเกิดปัญหา (order ค้าง, ลูกค้าโดนตัดเงินแต่ order ไม่ถูกสร้าง) ต้องสามารถตามรอย request นั้นข้ามทุก service ได้ ไม่ใช่ไล่ดู log ทีละ service แยกกันแบบเดา

หลักการและเครื่องมือทำ distributed tracing เต็มรูปแบบ (trace ID propagation, span, การเชื่อม log/metric/trace เข้าด้วยกัน) อยู่ใน [Volume 14 — Observability](../14-observability/README.md) ในบริบท e-commerce นี้ **trace ID เดียว**ถูกสร้างตั้งแต่ request แรกเข้า API Gateway แล้วส่งต่อ (propagate) ผ่าน header ไปทุก service ที่เกี่ยวข้อง รวมถึงถูกแนบไปกับ Kafka event ที่ publish ไปยัง Notification service ด้วย:

```
API Gateway (สร้าง trace_id = abc123)
   │
   ▼
Order Service (span: create_order, trace_id=abc123)
   │
   ├──► Inventory Service (span: reserve_stock, trace_id=abc123)
   │
   ├──► Payment Service (span: charge, trace_id=abc123)
   │
   └──► Kafka event {trace_id: abc123, type: "order.confirmed"}
              │
              ▼
        Notification Service (span: send_email, trace_id=abc123)
```

เมื่อลูกค้าร้องเรียนเรื่อง order หนึ่งรายการ engineer ค้นหาด้วย trace ID เดียว (ที่มักผูกกับ order ID) แล้วเห็น**ภาพรวมทั้งหมด**ว่า request วิ่งผ่านที่ไหนบ้าง ใช้เวลาที่แต่ละ service เท่าไหร่ และล้มเหลวที่จุดไหน — โดยไม่ต้องเดาหรือไล่ log แยกทีละระบบ

## 5.9 สรุป: บทนี้คือ Blueprint

สถาปัตยกรรมทั้งหมดในบทนี้ — Catalog+OpenSearch, Inventory ที่ป้องกัน race condition ด้วย atomic decrement หรือ row lock, Order เป็น Saga/CQRS orchestrator บน Temporal, Payment ที่ยึดหลัก idempotency+double-entry ledger, Notification แบบ event-driven ผ่าน Kafka, APISIX เป็น API Gateway จุดเดียว, และ observability ที่ผูกทุก service เข้าด้วยกันผ่าน trace ID เดียว — **คือ design document ที่การ implement จริงใน [projects/ecommerce/](../../projects/ecommerce/README.md) ต้องยึดเป็นแนวทางหลัก** ทุกการตัดสินใจทางเทคนิคใน implementation ควรย้อนกลับมาตรวจสอบกับหลักการในบทนี้และบทก่อนหน้าของ Volume นี้เสมอ ว่าการเลือกนั้นสอดคล้องกับ trade-off ที่ตั้งใจไว้หรือไม่

## สรุปย่อ

| Service | เทคโนโลยีหลัก | ประเด็นออกแบบสำคัญที่สุด |
|---|---|---|
| Catalog | PostgreSQL + OpenSearch | แยก write model กับ search/read model คนละ storage |
| Inventory | PostgreSQL | ป้องกัน overselling ด้วย atomic conditional decrement หรือ pessimistic row lock |
| Order | Temporal (Saga/CQRS) | รวม logic ของ multi-step flow ไว้ที่เดียวใน workflow definition |
| Payment | PostgreSQL (ledger) | Idempotency + double-entry ledger + compensation (ดูบทที่ 4) |
| Notification | Kafka (event-driven) | Decouple จาก Order service เต็มรูปแบบผ่าน publish/subscribe |
| API Gateway | APISIX | จุดเดียวจัดการ auth, rate limit, routing สำหรับทุก service |
| Observability | Trace ID propagation | ตามรอย request เดียวข้ามทุก service ได้ผ่าน trace ID เดียวกัน |

## คำถามทบทวน

1. อธิบาย race condition ของการ oversell สินค้าชิ้นสุดท้ายด้วย timeline ของ 2 request ที่แข่งกัน แล้วอธิบายว่า atomic conditional decrement แก้ปัญหานี้อย่างไรโดยไม่ต้องใช้ `FOR UPDATE`
2. เปรียบเทียบ pessimistic row lock กับ atomic conditional decrement ในแง่ throughput ภายใต้สถานการณ์ flash sale ที่มีคนแย่งซื้อสินค้าตัวเดียวกันจำนวนมาก
3. ทำไม Notification service ถึงไม่ควรถูกเรียกแบบ synchronous จาก Order service โดยตรง แนวทาง event-driven ผ่าน Kafka แก้ปัญหาอะไร
4. อธิบายว่า trace ID เดียวที่ถูกสร้างตั้งแต่ API Gateway ช่วยให้ debug ปัญหา order ที่ค้างระหว่าง service ต่างๆ ได้อย่างไร ต่างจากการไล่ดู log ทีละ service อย่างไร
5. ทำไม Order service ถึงถูกออกแบบเป็น CQRS แยก write model กับ read model แทนที่จะใช้ model เดียวกันทั้งสองฝั่ง

---

ก่อนหน้า: [บทที่ 4 — Case Study: Payment และ Chat](04-case-study-payment-chat.md) | กลับสู่ [สารบัญ Volume 17](README.md)
