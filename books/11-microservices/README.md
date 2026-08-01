# Volume 11 — Microservices

หัวข้อนี้รวม pattern สำคัญที่ทำให้ระบบ microservices "ทำงานได้จริง" ในโลกจริงที่ network ไม่น่าเชื่อถือ 100% และ service ล่มได้ตลอดเวลา

## เป้าหมายการเรียนรู้

- เลือกระหว่าง REST กับ gRPC ให้เหมาะกับสถานการณ์การสื่อสาร
- ออกแบบระบบให้ทนต่อความล้มเหลวบางส่วน (partial failure) ด้วย Retry/Circuit Breaker/Idempotency
- เข้าใจ pattern การจัดการ transaction ข้าม service (Saga) และ CQRS/Event Sourcing
- ออกแบบ tracing ข้าม service ด้วย Trace ID

## สารบัญ

### [บทที่ 1 — REST และ gRPC](01-communication.md)

- [ ] Resource-based design, HTTP verb ให้ตรงความหมาย, status code
- [ ] Versioning strategy, HATEOAS (แนวคิด ไม่จำเป็นต้องใช้จริงเสมอ)
- [ ] Protocol Buffers (protobuf), code generation
- [ ] HTTP/2 เป็น transport, streaming (unary, server-streaming, client-streaming, bidirectional)
- [ ] เทียบกับ REST — เมื่อไหร่ควรใช้ gRPC (internal service-to-service ที่ต้องการ performance)

### [บทที่ 2 — Service Discovery](02-service-discovery.md)

- [ ] Client-side vs server-side discovery
- [ ] ตัวอย่างกลไก: DNS-based (K8s Service), registry-based (Consul)

### [บทที่ 3 — CQRS และ Event Sourcing](03-data-patterns.md)

- [ ] Command Query Responsibility Segregation — แยก model สำหรับเขียนกับอ่าน
- [ ] เมื่อไหร่คุ้มค่าที่จะใช้ (read-heavy, model ซับซ้อนต่างกันมาก)
- [ ] เก็บ state เป็นลำดับ event แทนการเก็บ state ปัจจุบัน
- [ ] Event store, replay event เพื่อสร้าง state, snapshot

### [บทที่ 4 — Saga, Retry, Circuit Breaker](04-saga-and-resilience.md)

- [ ] จัดการ distributed transaction โดยไม่ใช้ 2PC
- [ ] Choreography-based vs Orchestration-based Saga
- [ ] Compensating transaction (เชื่อมกับ [Volume 13 — Temporal](../13-temporal/README.md) ที่ใช้ทำ orchestration จริง)
- [ ] Retry policy: fixed delay, exponential backoff, jitter
- [ ] อันตรายของ retry storm และวิธีป้องกัน
- [ ] State: Closed, Open, Half-Open
- [ ] ป้องกัน cascading failure เมื่อ downstream service ล่ม

### [บทที่ 5 — Idempotency และ Trace ID](05-idempotency-and-tracing.md)

- [ ] Idempotency key — ทำให้ retry ปลอดภัย (โดยเฉพาะ payment)
- [ ] ออกแบบ API ให้ idempotent ได้อย่างไร
- [ ] Correlation ID / Trace ID ส่งผ่าน header ข้าม service
- [ ] เชื่อมกับ distributed tracing (เชื่อมกับ [Volume 14 — Observability](../14-observability/README.md), Jaeger/OpenTelemetry)

## Lab / Project ที่เกี่ยวข้อง

- [projects/ecommerce/](../../projects/ecommerce/README.md)
- [projects/payment/](../../projects/payment/README.md)

## แหล่งอ้างอิงแนะนำ

ดู [resources/](../../resources/README.md)
