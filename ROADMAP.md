# Roadmap

แผนการเรียนรู้แบบเรียงลำดับ ออกแบบให้เรียนจากพื้นฐานไปสู่ระบบที่ซับซ้อนขึ้นเรื่อยๆ แต่ละ Volume ควรเรียนควบคู่กับแล็บที่เกี่ยวข้องใน [`labs/`](labs/) เพื่อให้เข้าใจแบบลงมือทำจริง ไม่ใช่แค่ทฤษฎี

ทำเครื่องหมาย `[x]` เมื่อเรียนจบแต่ละหัวข้อ เพื่อติดตามความคืบหน้าของตัวเอง

## Phase 1 — พื้นฐาน (Foundation)

- [ ] [Volume 1: Computer Foundation](books/01-computer-foundation/README.md)
- [ ] [Volume 2: Network](books/02-network/README.md)
- [ ] [Volume 3: Linux & Git](books/03-linux/README.md)

## Phase 2 — ภาษาและฐานข้อมูล (Language & Data)

- [ ] [Volume 4: Golang](books/04-golang/README.md)
- [ ] [Volume 5: PostgreSQL](books/05-postgresql/README.md)
- [ ] [Volume 6: Redis](books/06-redis/README.md)

## Phase 3 — Containerization & Orchestration

- [ ] [Volume 7: Docker](books/07-docker/README.md)
- [ ] [Volume 8: Kubernetes](books/08-kubernetes/README.md)

## Phase 4 — Platform & API Gateway

- [ ] [Volume 9: Authentication](books/09-authentication/README.md)
- [ ] [Volume 10: APISIX](books/10-apisix/README.md)

## Phase 5 — Distributed Systems

- [ ] [Volume 11: Microservices](books/11-microservices/README.md)
- [ ] [Volume 12: Kafka](books/12-kafka/README.md)
- [ ] [Volume 13: Temporal](books/13-temporal/README.md)

## Phase 6 — Production Readiness

- [ ] [Volume 14: Observability](books/14-observability/README.md)
- [ ] [Volume 15: Security](books/15-security/README.md)
- [ ] [Volume 16: CI/CD](books/16-cicd/README.md)

## Phase 7 — สังเคราะห์ความรู้ (Synthesis)

- [ ] [Volume 17: System Design](books/17-system-design/README.md)

## Final Project

- [ ] [Enterprise E-commerce](projects/ecommerce/README.md) — รวมทุก Volume เข้าด้วยกันเป็นระบบจริง
- [ ] [Payment Service](projects/payment/README.md) — โฟกัส consistency, idempotency, Saga
- [ ] [Chat Service](projects/chat/README.md) — โฟกัส realtime, websocket, pub/sub
- [ ] [Monolith Starter](projects/monolith/README.md) — จุดเริ่มต้นก่อนแตกเป็น microservices

## หลักการเรียน

1. อ่านสารบัญของแต่ละ book ก่อน เพื่อดูภาพรวมว่าต้องรู้อะไรบ้าง
2. ลงมือทำแล็บที่เกี่ยวข้องใน `labs/` ควบคู่กันไป อย่าอ่านอย่างเดียว
3. เขียนโค้ดจริงเก็บไว้ใน `source-code/` เพื่อกลับมาอ้างอิงภายหลัง
4. เมื่อเรียนครบ 1 phase ให้ลองอธิบายหัวข้อนั้นด้วยคำพูดตัวเอง (Feynman technique) ก่อนไป phase ถัดไป
5. เมื่อครบทุก phase ให้ลงมือทำ Final Project เพื่อรวมความรู้ทั้งหมดเข้าด้วยกัน
