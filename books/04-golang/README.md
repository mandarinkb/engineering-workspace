# Volume 4 — Golang

Golang เป็นภาษาหลักของหลักสูตรนี้ เพราะเป็นภาษาที่นิยมใช้สร้าง backend/infrastructure ระดับ enterprise (Kubernetes, Docker, Prometheus ก็เขียนด้วย Go) จุดเด่นคือ concurrency model ที่เรียบง่าย, compile เร็ว, deploy เป็น binary เดียวจบ

## เป้าหมายการเรียนรู้

- เขียนโปรแกรม Go ที่ถูกต้องตาม idiom ของภาษา ไม่ใช่เขียนแบบแปลจากภาษาอื่นตรงๆ
- ใช้ goroutine/channel แก้ปัญหา concurrency ได้อย่างปลอดภัย (ไม่มี race condition)
- ใช้ `context` ควบคุม cancellation/timeout/deadline ข้าม call chain ได้
- เขียน test ที่มีความหมาย และออกแบบโครงสร้างโปรเจกต์ตามหลัก Clean Architecture / Hexagonal / DDD

## สารบัญ

### [บทที่ 1 — Language Fundamentals](01-language-fundamentals.md)

- [ ] Syntax พื้นฐาน, type system, struct, interface (implicit implementation)
- [ ] Pointer vs value receiver, เมื่อไหร่ควรใช้แบบไหน
- [ ] Error handling แบบ Go (`error` เป็น value, `errors.Is`/`errors.As`, wrapping)
- [ ] Slice vs Array, การทำงานภายในของ slice (len, cap, underlying array)
- [ ] Generics (type parameters)

### [บทที่ 2 — Concurrency และ Context](02-concurrency-and-context.md)

- [ ] Goroutine คืออะไร ต่างจาก OS thread อย่างไร (เชื่อมกับ [Volume 1](../01-computer-foundation/README.md))
- [ ] Channel — unbuffered vs buffered, select statement
- [ ] `sync` package — Mutex, RWMutex, WaitGroup, Once
- [ ] Race condition และการตรวจจับด้วย `-race`
- [ ] Pattern: worker pool, fan-in/fan-out, pipeline
- [ ] `context.Context` ใช้ทำอะไร (cancellation, deadline, value propagation)
- [ ] `WithCancel`, `WithTimeout`, `WithDeadline`, `WithValue`
- [ ] แนวทางการส่ง context ผ่าน call chain อย่างถูกต้อง (เป็น argument แรกเสมอ)

### [บทที่ 3 — Testing](03-testing.md)

- [ ] `testing` package พื้นฐาน, table-driven test
- [ ] Mock/stub, interface สำหรับ testability
- [ ] Benchmark test, `go test -bench`
- [ ] Test coverage, integration test vs unit test

### [บทที่ 4 — Architecture Patterns](04-architecture-patterns.md)

- [ ] **Clean Architecture** — dependency rule, layer (entity, use case, interface adapter, framework)
- [ ] **DDD (Domain-Driven Design)** — entity, value object, aggregate, domain service, bounded context
- [ ] **Hexagonal (Ports & Adapters)** — แยก core logic ออกจาก infrastructure ผ่าน interface (port) และ implementation (adapter)
- [ ] **Repository pattern** — แยก data access ออกจาก business logic
- [ ] **Dependency Injection** — constructor injection, wiring แบบ manual vs library (เช่น wire, fx)
- [ ] **Middleware** — HTTP middleware chain, use case: logging, auth, recovery, tracing

## Lab / Source Code ที่เกี่ยวข้อง

- [source-code/golang/](../../source-code/golang/README.md) — เก็บโค้ดตัวอย่างประกอบเล่มนี้

## แหล่งอ้างอิงแนะนำ

ดู [resources/](../../resources/README.md)
