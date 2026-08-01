# Project: Enterprise E-commerce (Final Project)

โปรเจกต์สุดท้ายของหลักสูตรที่รวมทุก Volume เข้าด้วยกันเป็นระบบเดียว ตามที่ระบุใน [Enterprise_Backend_Infrastructure_Curriculum.md](../../Enterprise_Backend_Infrastructure_Curriculum.md)

## สถานะ

โครงไว้ก่อน — ยังไม่มีเนื้อหา/โค้ดจริงในรอบนี้

## Stack ที่ใช้ (แผน)

Golang, Kubernetes, APISIX, PostgreSQL, Redis, Kafka, Temporal, OpenSearch, Prometheus, Grafana, GitLab, Jenkins, Harbor

## ขอบเขตระบบ (แผน)

- **Catalog service** — จัดการสินค้า, ค้นหา (OpenSearch)
- **Inventory service** — จัดการสต็อก, ป้องกัน race condition ตอน checkout
- **Order service** — orchestrate การสั่งซื้อ (CQRS/Saga จาก [books/11-microservices](../../books/11-microservices/README.md))
- **Payment service** — ดู [projects/payment/](../payment/README.md)
- **Notification service** — แจ้งเตือนผ่าน Kafka event
- **API Gateway** — APISIX เป็นด่านหน้า ([books/10-apisix](../../books/10-apisix/README.md))
- **Observability stack** — Prometheus + Grafana + Fluent Bit + OpenSearch + Jaeger

## ก่อนเริ่มโปรเจกต์นี้

ควรเรียนจบ Phase 1-6 ใน [ROADMAP.md](../../ROADMAP.md) ก่อน เพราะโปรเจกต์นี้ไม่ได้สอนพื้นฐานซ้ำ แต่เป็นที่ประกอบร่างความรู้ทั้งหมด
