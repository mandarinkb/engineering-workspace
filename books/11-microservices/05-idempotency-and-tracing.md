# บทที่ 5: Idempotency และ Trace ID

## 5.1 Idempotency

### สถานการณ์: Retry การจ่ายเงินโดยไม่มี Idempotency Key

ลองไล่สถานการณ์นี้ทีละขั้นตอนเพื่อเห็นปัญหาให้ชัดเจน:

1. Client ยิง `POST /payments` เพื่อจ่ายเงิน 1,000 บาท
2. Payment service ประมวลผลสำเร็จ หักเงิน 1,000 บาทจริง แต่**ระหว่างส่ง response กลับ connection หลุด** (network timeout ฝั่ง client)
3. Client เห็นแค่ว่า request "timeout" — **ไม่รู้ว่า server ประมวลผลสำเร็จไปแล้วหรือยัง** (อาจสำเร็จแล้วแต่ response หายระหว่างทาง หรืออาจยังไม่ทันประมวลผลเลยก็ได้ ทั้งสองแบบมีลักษณะเหมือนกันจากมุมมอง client)
4. Client ทำตามหลัก retry ที่ถูกต้อง (เชื่อมกับ [บทที่ 4 — Retry](04-saga-and-resilience.md)) จึงยิง `POST /payments` ซ้ำอีกครั้งด้วยข้อมูลเดิม
5. **ถ้า payment service ไม่มีกลไกป้องกัน** จะมองว่านี่คือ payment request ใหม่ทั้งหมด แล้วหักเงินอีก 1,000 บาท — ลูกค้าโดนหักเงินสองครั้งจาก request เดียวในความตั้งใจของ client

ปัญหานี้เกิดจาก `POST` ไม่ idempotent โดยธรรมชาติ (ตามที่กล่าวใน [บทที่ 1](01-communication.md)) — server ไม่มีทางแยกแยะได้ว่า request ที่สองคือ "retry ของ request เดิม" หรือ "การจ่ายเงินใหม่ที่ตั้งใจจ่ายซ้ำจริงๆ"

### Idempotency Key แก้ปัญหาอย่างไร

วิธีแก้คือให้ **client เป็นคนสร้าง unique key** สำหรับ "การกระทำเชิงตรรกะ" หนึ่งครั้ง (ไม่ใช่ต่อ HTTP request แต่ต่อ "ความตั้งใจ" หนึ่งครั้ง) แล้วส่ง key นี้แนบไปกับทุกครั้งที่ยิง request รวมถึงตอน retry:

```
Client สร้าง idempotency key ครั้งเดียว: "pay-order-4521-attempt"  (เช่น UUID)

Request ครั้งที่ 1:
  POST /payments
  Idempotency-Key: 8f14e45f-ceea-4...
  { "order_id": 4521, "amount": 1000 }
  ──► server ประมวลผล หักเงินจริง บันทึกผลลัพธ์คู่กับ key นี้ไว้
  ──► response หายระหว่างทาง (timeout ฝั่ง client)

Request ครั้งที่ 2 (retry, key เดิม):
  POST /payments
  Idempotency-Key: 8f14e45f-ceea-4...   ← key เดิมเป๊ะ
  { "order_id": 4521, "amount": 1000 }
  ──► server เจอ key นี้เคยประมวลผลไปแล้ว
  ──► ไม่หักเงินซ้ำ แค่ส่ง response เดิม (ที่เก็บไว้) กลับไปให้ client
```

ฝั่ง server ต้องเก็บ mapping ของ `idempotency key → ผลลัพธ์ที่เคยตอบไป` (มักเก็บใน cache เช่น Redis ที่มี TTL พอสมควร เช่น 24 ชั่วโมง — เชื่อมกับ [Volume 6 — Redis](../06-redis/README.md)) เมื่อเจอ key ที่เคยเห็นมาก่อน**ไม่ประมวลผลซ้ำ** แค่คืนผลลัพธ์เดิมที่บันทึกไว้กลับไปทันที ทำให้ retry กี่ครั้งก็ปลอดภัย — นี่คือหัวใจที่ทำให้ operation ที่โดยธรรมชาติไม่ idempotent (เช่น "จ่ายเงิน") กลาย เป็น idempotent ได้จากมุมมองของระบบ

จุดสำคัญ: idempotency key ต้องสร้างขึ้น**ครั้งเดียวต่อความตั้งใจหนึ่งครั้ง** และใช้ key เดิมซ้ำตอน retry เท่านั้น — ถ้า client สุ่ม key ใหม่ทุกครั้งที่ยิง (รวมถึงตอน retry) ก็เท่ากับไม่ได้ป้องกันอะไรเลย เพราะ server จะมองว่าเป็นคนละ operation กันเสมอ

### ออกแบบ API ให้ Idempotent โดยธรรมชาติ

นอกจาก idempotency key ยังมีแนวทางออกแบบ API ให้ idempotent โดยไม่ต้องพึ่ง key เพิ่มเติม:

- **ใช้ `PUT` แทน `POST` เมื่อทำได้** — `PUT /users/42` (แทนที่ทั้ง resource ด้วยข้อมูลที่ระบุ) idempotent โดยธรรมชาติ เรียกกี่ครั้งก็ได้ผลลัพธ์เดียวกัน ต่างจาก `POST /users` ที่สร้างใหม่ทุกครั้ง
- **ใช้ natural key แทน auto-generated ID** — ถ้า client กำหนด ID เอง (เช่น `PUT /orders/{client-generated-order-id}`) แทนที่จะให้ server สุ่ม ID ใหม่ทุกครั้งที่ `POST` การสร้างซ้ำด้วย ID เดิมจะกลายเป็น "แก้ไข resource เดิม" แทนที่จะเป็น "สร้างใหม่ซ้ำซ้อน" โดยอัตโนมัติ
- **Upsert semantics** — ออกแบบให้ operation เป็น "ถ้ามีอยู่แล้วให้ข้าม/อัปเดต ถ้าไม่มีให้สร้าง" แทนที่จะ error เมื่อเจอ duplicate

แนวทางเหล่านี้ใช้ได้ดีเมื่อ operation มี natural key ที่ชัดเจนอยู่แล้ว (เช่น order, user) แต่สำหรับ operation ที่ไม่มี natural key ชัดเจนเป็นตัวเอง (เช่น "การพยายามจ่ายเงินครั้งนี้") การใช้ idempotency key แบบ explicit ยังเป็นแนวทางที่ตรงไปตรงมาและควบคุมได้ชัดเจนที่สุด

## 5.2 Trace ID

### ปัญหา: Request หนึ่งครั้งของ User กลายเป็นหลาย Call ข้าม Service

Request เดียวจาก client อาจไหลผ่านหลาย service ก่อนจะได้ผลลัพธ์กลับไป เช่น request สั่งซื้อสินค้าหนึ่งครั้งอาจไหลผ่าน API Gateway → Order Service → Inventory Service → Payment Service → Notification Service — ถ้า request ล้มเหลวหรือช้าตรงไหนสักจุด การไล่ log แยกทีละ service โดยไม่มีอะไรเชื่อมโยงกันเลยแทบเป็นไปไม่ได้ในทางปฏิบัติ (แต่ละ service มี log ของตัวเอง เต็มไปด้วย request จาก user คนอื่นปนกันอยู่ ไม่รู้ว่า log บรรทัดไหนของ service B เกี่ยวข้องกับ request ที่กำลังสืบสวนอยู่ใน service A)

### Correlation ID / Trace ID: หลักการง่ายที่สุดที่ได้ผลจริง

วิธีแก้ที่เรียบง่ายแต่ทรงพลังคือ: เมื่อ request เข้าสู่ระบบครั้งแรก (มักที่ API Gateway หรือ service แรกที่รับ request) ให้สร้าง **trace ID** (หรือเรียก correlation ID) ที่ไม่ซ้ำกันขึ้นมาหนึ่งตัว แล้ว**ส่งต่อ ID นี้ผ่าน HTTP header ในทุก call ที่ตามมา** ไม่ว่า request จะไหลไปกี่ service ก็ตาม:

```
Client ──► API Gateway
              │ สร้าง Trace-Id: abc-123 (ถ้ายังไม่มีมาจาก client)
              │ ใส่ header: X-Trace-Id: abc-123
              ▼
         Order Service ──(ส่งต่อ header เดิม)──► Inventory Service
              │                                        │
              │ log: "[trace=abc-123] validating order" │ log: "[trace=abc-123] checking stock"
              ▼
         Payment Service
              │ log: "[trace=abc-123] charging card"
              ▼
         Notification Service
              log: "[trace=abc-123] sending email"
```

ทุก service ที่รับ request ต้องทำสองอย่าง: (1) **อ่าน** trace ID จาก header ขาเข้าถ้ามี (ถ้าไม่มีแสดงว่าเป็น entry point แรก ให้สร้างใหม่), (2) **ใส่** trace ID นี้ลงใน log ทุกบรรทัดที่เกี่ยวข้องกับ request นี้ และ**ส่งต่อ**ผ่าน header เดิมไปยัง service ถัดไปที่เรียกต่อ — เพียงเท่านี้ เมื่อต้องสืบสวนปัญหา สามารถค้นหา trace ID เดียวกันใน log aggregation system (เช่น กรอง log จากทุก service ด้วยคำว่า `abc-123`) แล้วเห็น**ลำดับเหตุการณ์ทั้งหมดของ request นั้นข้ามทุก service เรียงตามเวลา** ได้ทันที

### Trace ID คือ Observability ขั้นต่ำสุดที่ต้องมี

การมี trace ID เพียงอย่างเดียว (โดยไม่มีระบบ tracing เต็มรูปแบบ) คือ**เวอร์ชันขั้นต่ำสุดที่ยังใช้งานได้จริง**ของ distributed observability — แค่ propagate ID เดียวผ่าน header และใส่ใน log ทุกจุด ก็เปลี่ยนจาก "log กระจัดกระจายไม่เชื่อมกัน" เป็น "เห็นเส้นทางของ request ทั้งหมด" ได้แล้ว โดยไม่ต้องพึ่งเครื่องมือพิเศษใดๆ นอกจาก log aggregation ธรรมดา

ระบบ **distributed tracing** เต็มรูปแบบ (เช่น Jaeger, OpenTelemetry) ต่อยอดจากแนวคิดพื้นฐานนี้ไปอีกขั้น โดยเพิ่มแนวคิด **span** (แต่ละ operation ย่อยในแต่ละ service มีเวลาเริ่ม/จบของตัวเอง) ที่เชื่อมกันเป็น **trace tree** ทำให้เห็นไม่ใช่แค่ "ลำดับเหตุการณ์" แต่เห็น**ระยะเวลาที่แต่ละขั้นตอนใช้ไป**และ**ความสัมพันธ์แบบ parent-child**ระหว่าง call (เช่น เห็นชัดว่า 80% ของเวลาทั้งหมดถูกใช้ไปที่ span ไหนใน service ไหน) รายละเอียดกลไก span/trace เต็มรูปแบบจะอยู่ใน [Volume 14 — Observability](../14-observability/README.md) — สิ่งสำคัญที่บทนี้ต้องการเน้นคือ**ปัญหาการ propagate ID ข้าม service** เป็นรากฐานที่ทุกระบบ tracing (ไม่ว่าจะซับซ้อนแค่ไหน) ต้องแก้ให้ได้ก่อนเป็นอันดับแรก — ถ้า propagation พังตรงไหนสักจุด (เช่น มี service หนึ่งลืมส่งต่อ header) trace จะขาดตอน ไม่ว่าเครื่องมือ tracing ที่ใช้จะซับซ้อนแค่ไหนก็ตาม

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| ปัญหาที่ idempotency key แก้ | retry โดยไม่รู้ว่า request เดิมสำเร็จหรือยัง อาจทำให้ operation ที่ไม่ idempotent (เช่น จ่ายเงิน) เกิดซ้ำ |
| กลไก idempotency key | client สร้าง key ต่อความตั้งใจหนึ่งครั้ง ใช้ key เดิมตอน retry, server เก็บผลลัพธ์คู่ key ไว้คืนซ้ำ |
| API ที่ idempotent โดยธรรมชาติ | `PUT` แทน `POST`, ใช้ natural key ที่ client กำหนดเอง, upsert semantics |
| ปัญหาที่ trace ID แก้ | request เดียวไหลผ่านหลาย service ทำให้ log แต่ละที่แยกกันไม่เชื่อมโยงกัน |
| กลไก trace ID | สร้างที่ entry point แรก ส่งต่อผ่าน header ทุก call ใส่ใน log ทุกจุด |
| ความสัมพันธ์กับ full tracing | trace ID คือ observability ขั้นต่ำสุด, span/trace tree ของ Jaeger/OpenTelemetry ต่อยอดจากรากฐานนี้ |

## คำถามทบทวน

1. อธิบายทีละขั้นตอนว่าทำไม retry การจ่ายเงินโดยไม่มี idempotency key ถึงทำให้ลูกค้าถูกหักเงินซ้ำได้
2. Idempotency key ต้องสร้างขึ้นตอนไหน และใช้ค่าเดิมซ้ำเมื่อไหร่? ถ้า client สุ่ม key ใหม่ทุกครั้งที่ retry จะเกิดอะไรขึ้น
3. ทำไม `PUT /orders/{client-generated-id}` ถึง idempotent โดยธรรมชาติ ในขณะที่ `POST /orders` ไม่ใช่?
4. อธิบายว่า trace ID ช่วยแก้ปัญหาอะไรตอน debug request ที่ไหลผ่านหลาย service เทียบกับการไม่มี trace ID เลย
5. เพราะเหตุใด trace ID จึงถือเป็น "รากฐาน" ของ distributed tracing เต็มรูปแบบ (เช่น Jaeger) จะเกิดอะไรขึ้นถ้า service หนึ่งลืมส่งต่อ header trace ID ไปยัง service ถัดไป

---

ก่อนหน้า: [บทที่ 4 — Saga, Retry, Circuit Breaker](04-saga-and-resilience.md) | กลับสู่ [สารบัญ Volume 11](README.md)
