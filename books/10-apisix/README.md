# Volume 10 — APISIX

Apache APISIX เป็น API Gateway ระดับ enterprise ที่ทำหน้าที่เป็นด่านหน้าของระบบ microservices ทั้งหมด รวม routing, authentication, rate limiting และอื่นๆ ไว้ในที่เดียว

## เป้าหมายการเรียนรู้

- ตั้งค่า Route/Upstream เพื่อ route traffic ไปยัง service ต่างๆ ได้
- เข้าใจแนวคิด Consumer และใช้ Plugin จัดการ auth/rate-limit
- ออกแบบ API Gateway layer ให้เป็นจุดรวมของ cross-cutting concern แทนที่จะเขียนซ้ำในทุก service

## สารบัญ

### [บทที่ 1 — Route และ Upstream](01-routing.md)

- [ ] Route คืออะไร — จับคู่ request (path, method, host) ไปยัง upstream
- [ ] Priority, path matching (exact, prefix, regex)
- [ ] Upstream คือกลุ่มปลายทางจริง (service instance), load balancing algorithm
- [ ] Health check (active/passive), retry

### [บทที่ 2 — Consumer และ Plugins](02-consumers-and-plugins.md)

- [ ] Consumer คือ "ตัวตน" ของผู้เรียกใช้ API (แอป/ทีมที่เรียก API)
- [ ] ผูก Consumer เข้ากับ plugin (เช่น key-auth, jwt-auth) เพื่อระบุ/จำกัดสิทธิ์แต่ละราย
- [ ] สถาปัตยกรรม plugin แบบ pluggable, plugin lifecycle
- [ ] Plugin ที่ใช้บ่อย: cors, request-transformer, response-transformer, logging (เชื่อมกับ [Volume 14 — Observability](../14-observability/README.md))

### [บทที่ 3 — JWT และ Rate Limit](03-auth-and-rate-limit.md)

- [ ] `jwt-auth` plugin — validate token ที่ gateway ก่อนถึง service
- [ ] ลด load ของแต่ละ service ไม่ต้อง validate JWT ซ้ำเอง (เชื่อมกับ [Volume 9 — Authentication](../09-authentication/README.md))
- [ ] Algorithm: fixed window, sliding window, leaky bucket, token bucket
- [ ] `limit-req`, `limit-count`, `limit-conn` plugin ต่างกันอย่างไร
- [ ] Rate limit ตาม consumer/IP/route

## Lab ที่เกี่ยวข้อง

- [labs/apisix/](../../labs/apisix/README.md)

## แหล่งอ้างอิงแนะนำ

ดู [resources/](../../resources/README.md)
