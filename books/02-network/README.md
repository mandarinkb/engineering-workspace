# Volume 2 — Network

Backend engineer ต้องเข้าใจเครือข่ายในระดับที่สามารถ debug ปัญหา "ทำไม request ช้า" หรือ "ทำไม connection หลุด" ได้ด้วยตัวเอง ไม่ใช่แค่รู้ว่า HTTP คืออะไร แต่ต้องเข้าใจตั้งแต่ layer ล่างสุดไปจนถึง protocol ระดับ application

## เป้าหมายการเรียนรู้

- อธิบาย flow การส่งข้อมูลจาก application layer ลงไปจนถึง physical layer และย้อนกลับได้
- เข้าใจความแตกต่างของ TCP กับ UDP และเลือกใช้ให้เหมาะกับสถานการณ์
- อธิบายกระบวนการ TLS handshake และวิธีที่ mTLS ใช้ยืนยันตัวตนทั้งสองฝั่ง
- ออกแบบ/วิเคราะห์การทำงานของ reverse proxy และ load balancer ได้

## สารบัญ

### [บทที่ 1 — OSI Model, TCP/IP, UDP, Socket](01-osi-tcp-ip.md)

- [ ] 7 layers ของ OSI คืออะไรบ้าง เทียบกับ TCP/IP model (4 layers)
- [ ] **TCP** — 3-way handshake, sequence number, ACK, retransmission, flow control (window size), congestion control, TCP state machine
- [ ] **UDP** — connectionless, ไม่มี guarantee, เหมาะกับงานแบบไหน (DNS, streaming, gaming)
- [ ] **Socket** — socket API, file descriptor, `bind`/`listen`/`accept`/`connect`

### [บทที่ 2 — DNS](02-dns.md)

- [ ] Domain hierarchy, root/TLD/authoritative nameserver
- [ ] Record types: A, AAAA, CNAME, MX, TXT, SRV
- [ ] Resolution flow: recursive resolver, caching, TTL
- [ ] DNS กับ service discovery / load balancing (round-robin DNS)

### [บทที่ 3 — HTTP/1.1, HTTP/2, HTTP/3](03-http.md)

- [ ] **HTTP/1.1** — request/response, keep-alive, head-of-line blocking
- [ ] **HTTP/2** — multiplexing, header compression (HPACK), server push
- [ ] **HTTP/3** — QUIC บน UDP, แก้ head-of-line blocking ระดับ transport

### [บทที่ 4 — HTTPS, TLS, Certificates, mTLS](04-tls-security.md)

- [ ] TLS handshake (แบบ 1.2 และ 1.3 ต่างกันอย่างไร)
- [ ] Certificate chain, CA, self-signed cert, SNI
- [ ] Symmetric vs asymmetric encryption ที่ใช้ใน TLS
- [ ] **mTLS** — client certificate, ใช้ยืนยันตัวตนระหว่าง service กับ service (เชื่อมกับ [Volume 9 — Authentication](../09-authentication/README.md))

### [บทที่ 5 — Reverse Proxy และ Load Balancer](05-reverse-proxy-load-balancer.md)

- [ ] Reverse proxy ต่างจาก forward proxy อย่างไร, use case: TLS termination, caching, compression, routing
- [ ] Layer 4 vs Layer 7 load balancing
- [ ] Algorithm: round-robin, least connection, weighted, consistent hashing
- [ ] Health check, session affinity (sticky session)

## Lab ที่เกี่ยวข้อง

ยังไม่มีแล็บเฉพาะสำหรับ Network ในรอบนี้ — แนะนำให้ฝึกผ่าน `curl`/`tcpdump`/`nc` ประกอบกับ [Volume 3 — Linux](../03-linux/README.md)

## แหล่งอ้างอิงแนะนำ

ดู [resources/](../../resources/README.md)
