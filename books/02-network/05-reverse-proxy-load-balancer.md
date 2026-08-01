# บทที่ 5: Reverse Proxy และ Load Balancer

## 5.1 Reverse Proxy

### Forward Proxy vs Reverse Proxy

- **Forward Proxy** — อยู่ฝั่ง**client** ทำงานแทน client (เช่น องค์กรตั้ง proxy ให้พนักงานทุกคนออก internet ผ่านจุดเดียว เพื่อกรอง/ควบคุม) server ปลายทางไม่รู้ว่า client ตัวจริงเป็นใคร เห็นแค่ proxy
- **Reverse Proxy** — อยู่ฝั่ง**server** ทำงานแทน server client ไม่รู้ว่าจริงๆ แล้ว request ถูกส่งต่อไปที่ server ตัวไหนข้างหลัง

```
Forward Proxy:
  Client ──► Forward Proxy ──► Internet ──► Server จริง
  (proxy ปกป้อง/ควบคุม client)

Reverse Proxy:
  Client ──► Internet ──► Reverse Proxy ──► Server จริง (หลายตัว)
  (proxy ปกป้อง/กระจายงานให้ server)
```

### Use Case ของ Reverse Proxy

- **TLS Termination** — รับการเข้ารหัส TLS จาก client ที่ proxy แล้วส่งต่อเป็น plain HTTP ไปยัง service ภายใน network ที่เชื่อถือได้อยู่แล้ว (ลดภาระ TLS handshake ให้ service backend แต่ละตัวไม่ต้องทำเอง)
- **Caching** — เก็บ response ที่ไม่เปลี่ยนบ่อยไว้ตอบ client โดยตรงโดยไม่ต้องยิงไปหา backend ทุกครั้ง
- **Compression** — บีบอัด response (gzip/brotli) ก่อนส่งกลับ client ลด bandwidth
- **Routing** — route request ไปยัง service ที่ถูกต้องตาม path/host (เช่น `/api/*` ไป service A, `/static/*` ไป service B)

เครื่องมือที่นิยม: Nginx, HAProxy, และ [APISIX](../10-apisix/README.md) ที่หลักสูตรนี้ใช้เป็น API Gateway (ซึ่งทำหน้าที่เป็น reverse proxy ที่มี feature เพิ่มเรื่อง plugin/auth/rate-limit)

## 5.2 Load Balancer

Reverse proxy หลายตัวทำหน้าที่ **load balancer** ไปด้วยในตัว (กระจาย request ไปยัง backend หลายตัว) แต่แนวคิด load balancing แยกพิจารณาได้เป็นเรื่องของตัวเอง

### Layer 4 vs Layer 7 Load Balancing

| | Layer 4 (Transport) | Layer 7 (Application) |
|---|---|---|
| ตัดสินใจจากอะไร | IP address + port เท่านั้น | เนื้อหา HTTP จริง (path, header, cookie) |
| ความเร็ว | เร็วกว่า (ไม่ต้องแกะ payload) | ช้ากว่าเล็กน้อย (ต้อง parse HTTP) |
| ความยืดหยุ่น | ต่ำ — route ตาม IP/port ได้อย่างเดียว | สูง — route ตาม path (`/api` vs `/static`), A/B testing, canary |
| ตัวอย่าง | L4 LB ทั่วไป (เช่น AWS NLB) | Nginx, APISIX, AWS ALB |

Backend engineer ส่วนใหญ่เจอ **Layer 7** บ่อยกว่า เพราะต้องการ routing logic ที่ซับซ้อนกว่าระดับ IP/port

### Algorithm การกระจาย Request

- **Round Robin** — วนไปทีละ backend เรียงลำดับ เรียบง่ายแต่ไม่สนใจว่า backend ตัวไหนกำลังโหลดหนักอยู่
- **Least Connections** — ส่งไปยัง backend ที่มี active connection น้อยที่สุด ณ ขณะนั้น เหมาะกับ request ที่ใช้เวลาประมวลผลไม่เท่ากัน
- **Weighted (Round Robin/Least Connections)** — ให้ backend ที่แรงกว่า (spec สูงกว่า) รับ request มากกว่าตามสัดส่วนที่กำหนด
- **Consistent Hashing** — hash ค่าบางอย่าง (เช่น user ID, session ID) แล้ว map ไปยัง backend คงที่ ทำให้ request จาก key เดียวกันไปหา backend ตัวเดิมเสมอ (สำคัญมากสำหรับ cache locality — ดู [Volume 6 — Redis](../06-redis/README.md) และ [Volume 17 — Sharding](../17-system-design/README.md)) ข้อดีอีกอย่างคือเมื่อเพิ่ม/ลด backend จำนวน key ที่ต้อง remap ไปยัง backend อื่นน้อยกว่า hashing แบบธรรมดามาก

### Health Check

Load balancer ต้องรู้ว่า backend ตัวไหน "พร้อมรับ traffic" จริงๆ ผ่าน health check สองแบบ:

- **Active health check** — LB ยิง request ไปเช็ค backend เองเป็นระยะ (เช่น `GET /healthz` ทุก 5 วินาที)
- **Passive health check** — สังเกตจาก traffic จริงที่ผ่านไป ถ้า backend ตอบ error/timeout ถี่ๆ ก็ตัดออกจาก pool ชั่วคราว

### Session Affinity (Sticky Session)

บาง application เก็บ state ไว้ที่ backend ตัวใดตัวหนึ่ง (เช่น session ใน memory ของ server นั้นโดยเฉพาะ ไม่ได้แชร์ผ่าน Redis) ทำให้ request จาก user เดิมต้องถูกส่งไปยัง backend ตัวเดิมเสมอ เรียกว่า **sticky session** (ทำได้ผ่าน cookie หรือ consistent hashing ตาม client IP)

ข้อเสีย: ทำลาย "ความไร้สถานะ (statelessness)" ของ horizontal scaling — ถ้า backend ตัวนั้นล่ม user ที่ sticky อยู่กับมันจะเสีย session ไปด้วย แนวทางที่ดีกว่าในระบบยุคใหม่คือทำให้ backend **stateless** และเก็บ session ไว้ที่ external store (เช่น Redis) แทน เพื่อให้ LB ส่ง request ไปตัวไหนก็ได้อย่างอิสระ

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| Forward vs Reverse Proxy | Forward ทำงานแทน client, Reverse ทำงานแทน server |
| Reverse Proxy use case | TLS termination, caching, compression, routing |
| L4 vs L7 LB | L4 เร็วแต่หยาบ (ตาม IP/port), L7 ยืดหยุ่นกว่า (ตามเนื้อหา HTTP) |
| LB Algorithm | Round robin เรียบง่าย, Least connections รู้โหลด, Consistent hashing รักษา cache locality |
| Sticky Session | แก้ปัญหา state ที่ backend เดียว แต่แลกมาด้วยการเสีย statelessness |

## คำถามทบทวน

1. อธิบายความต่างระหว่าง forward proxy กับ reverse proxy ด้วยตัวอย่างของตัวเอง
2. ทำไม Layer 7 load balancer ถึงทำ routing ตาม path ได้ แต่ Layer 4 ทำไม่ได้?
3. Consistent hashing ช่วยอะไรที่ round robin ทำไม่ได้? เกี่ยวข้องกับ cache อย่างไร?
4. ทำไม sticky session ถึงขัดกับแนวคิด stateless backend? ทางเลือกที่ดีกว่าคืออะไร?

---

ก่อนหน้า: [บทที่ 4 — HTTPS, TLS, mTLS](04-tls-security.md) | กลับสู่ [สารบัญ Volume 2](README.md)
