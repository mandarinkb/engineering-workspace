# Volume 15 — Security

ความปลอดภัยไม่ใช่ feature ที่เพิ่มทีหลัง แต่เป็นสิ่งที่ต้องคิดตั้งแต่ตอนออกแบบระบบ หมวดนี้ครอบคลุมช่องโหว่พื้นฐานที่พบบ่อยที่สุด และเครื่องมือจัดการ secret ระดับ enterprise

## เป้าหมายการเรียนรู้

- ระบุและป้องกันช่องโหว่ตาม OWASP Top 10 ได้ในโค้ดที่เขียนเอง
- ใช้ Vault จัดการ secret แทนการ hardcode หรือเก็บใน plain text
- ออกแบบ Network Policy จำกัดการสื่อสารระหว่าง service ตามหลัก least privilege

## สารบัญ

### [บทที่ 1 — OWASP Top 10](01-owasp-top-10.md)

- [ ] Broken Access Control
- [ ] Cryptographic Failures
- [ ] Injection (SQL injection, command injection)
- [ ] Insecure Design
- [ ] Security Misconfiguration
- [ ] Vulnerable and Outdated Components
- [ ] Identification and Authentication Failures
- [ ] Software and Data Integrity Failures
- [ ] Security Logging and Monitoring Failures
- [ ] Server-Side Request Forgery (SSRF)

### [บทที่ 2 — Vault, Secrets, และ Network Policy](02-vault-secrets-network-policy.md)

- [ ] HashiCorp Vault — secret engine (KV, database, PKI)
- [ ] Dynamic secret — ออก credential ชั่วคราวแทน static credential
- [ ] Authentication method (AppRole, Kubernetes auth)
- [ ] หลักการจัดการ secret: ไม่ commit ลง git, rotate สม่ำเสมอ, least privilege access
- [ ] เปรียบเทียบ K8s Secret (ดิบ) กับ Vault (dynamic + audit)
- [ ] K8s NetworkPolicy — จำกัด traffic ระหว่าง pod ตาม label selector
- [ ] Default deny + allow เฉพาะที่จำเป็น (zero-trust ภายในคลัสเตอร์)

## Lab ที่เกี่ยวข้อง

- [scripts/](../../scripts/README.md) — สคริปต์ security scan/check เบื้องต้น

## แหล่งอ้างอิงแนะนำ

ดู [resources/](../../resources/README.md)
