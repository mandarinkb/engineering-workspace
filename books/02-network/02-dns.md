# บทที่ 2: DNS

## 2.1 ทำไมต้องมี DNS

คอมพิวเตอร์สื่อสารกันด้วย IP address (เช่น `142.250.183.14`) แต่มนุษย์จำชื่อ (`google.com`) ได้ง่ายกว่า **DNS (Domain Name System)** คือระบบแปลงชื่อ domain เป็น IP address แบบกระจายศูนย์ (distributed) ไม่มีจุดศูนย์กลางเดียวที่เก็บข้อมูลทั้งหมด

## 2.2 Domain Hierarchy

```
                    . (root)
                    │
        ┌───────────┼───────────┐
       .com        .org        .th
        │
   ┌────┴────┐
 google    example
    │
   www.google.com
```

- **Root** — จุดบนสุด (เขียนแทนด้วย `.` แต่มักถูกละไว้) มี root server อยู่ 13 กลุ่มทั่วโลก (anycast)
- **TLD (Top-Level Domain)** — `.com`, `.org`, `.th` ฯลฯ ดูแลโดย registry ของแต่ละ TLD
- **Authoritative Nameserver** — server ที่มีข้อมูล DNS จริงของ domain นั้นๆ (เช่น server ของ Google ที่ดูแล `google.com`)

การ resolve ชื่อหนึ่งชื่อ คือการไล่ถามจาก root → TLD → authoritative ตามลำดับ (อธิบายละเอียดในหัวข้อ 2.4)

## 2.3 Record Types

| Record | ใช้ทำอะไร |
|---|---|
| **A** | map hostname → IPv4 address |
| **AAAA** | map hostname → IPv6 address |
| **CNAME** | alias ชี้ไปยัง hostname อื่น (ไม่ใช่ IP ตรง) |
| **MX** | ระบุ mail server ที่รับผิดชอบ domain นี้ |
| **TXT** | เก็บข้อความอิสระ ใช้บ่อยสำหรับ domain verification, SPF/DKIM (email security) |
| **SRV** | ระบุ host+port ของ service เฉพาะ (เช่นใช้ใน service discovery บางระบบ) |

ตัวอย่าง: `www.example.com` อาจเป็น `CNAME` ชี้ไปที่ `example.cdn.com` แล้ว `example.cdn.com` ค่อยเป็น `A` record ชี้ไปยัง IP จริง — การ chain แบบนี้ทำให้เปลี่ยนปลายทางจริงได้โดยไม่ต้องแก้ record ของ `www.example.com` เอง

## 2.4 Resolution Flow

```
Client (browser)
    │ 1. ถาม Recursive Resolver (เช่น 8.8.8.8)
    ▼
Recursive Resolver
    │ 2. ถ้าไม่มีใน cache → ถาม Root Server
    │    Root ตอบ "ไปถาม .com TLD server"
    │ 3. ถาม TLD Server
    │    TLD ตอบ "ไปถาม authoritative NS ของ example.com"
    │ 4. ถาม Authoritative Nameserver
    │    ได้คำตอบ IP จริงกลับมา
    ▼
Recursive Resolver เก็บผลไว้ cache แล้วตอบกลับ Client
```

ผู้ใช้ทั่วไปคุยกับ **Recursive Resolver** เท่านั้น (เช่น resolver ของ ISP, หรือ public resolver อย่าง `8.8.8.8`, `1.1.1.1`) resolver เป็นฝ่ายไล่ถามแทนทั้งหมด (ทำ "recursive query") ส่วนการถามระหว่าง resolver กับ root/TLD/authoritative เรียกว่า "iterative query" (แต่ละฝ่ายตอบแค่ "ไปถามต่อที่ไหน" ไม่ไล่ถามแทน)

### Caching และ TTL

ทุก DNS record มีค่า **TTL (Time To Live)** กำกับ บอกว่า resolver เก็บผลลัพธ์นี้ไว้ cache ได้นานแค่ไหนก่อนต้องถามใหม่ TTL สั้น (เช่น 60 วินาที) ทำให้เปลี่ยนแปลงค่าได้เร็วแต่เพิ่ม query load; TTL ยาว (เช่น 1 วัน) ลด load แต่เปลี่ยนแปลงค่าแล้วต้องรอนานกว่าทุกที่จะเห็นค่าใหม่ — เป็น trade-off ที่ต้องวางแผนล่วงหน้าก่อนทำ migration ใหญ่ (เช่น ลด TTL ล่วงหน้าก่อนย้าย server)

## 2.5 DNS กับ Service Discovery / Load Balancing

- **Round-robin DNS** — ตั้งหลาย `A` record สำหรับ hostname เดียว resolver จะสลับลำดับที่ตอบกลับ ทำให้ client กระจายไปหลาย IP แบบง่ายๆ (ไม่ใช่ load balancing จริงจัง เพราะไม่รู้ health/load ของแต่ละปลายทาง)
- **Kubernetes internal DNS** — ทุก Service ใน K8s (ดู [Volume 8](../08-kubernetes/README.md)) จะได้ DNS name อัตโนมัติ (เช่น `my-service.namespace.svc.cluster.local`) ทำให้ service คุยกันด้วยชื่อแทน IP ที่เปลี่ยนไปมาตลอดเวลา — เป็นกลไก service discovery ที่ง่ายที่สุดแบบหนึ่ง
- ข้อจำกัด: DNS มี caching ในหลายชั้น (OS, application, resolver) ทำให้บางครั้งการเปลี่ยนแปลง IP ไม่ propagate ทันที ต้องระวังเรื่องนี้เวลาทำ failover

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| Hierarchy | root → TLD → authoritative, กระจายศูนย์ ไม่มีจุดเดียวที่รู้ทุกอย่าง |
| Record types | A/AAAA ชี้ IP, CNAME เป็น alias, MX สำหรับ mail, TXT ใช้ verify/SPF |
| Resolution | recursive resolver ไล่ถามแทน client, ผลลัพธ์ถูก cache ตาม TTL |
| Service Discovery | K8s ใช้ DNS ภายในคลัสเตอร์แทน hardcode IP; ระวัง caching หลายชั้นตอน failover |

## คำถามทบทวน

1. อธิบายความต่างระหว่าง recursive query กับ iterative query
2. ทำไมการตั้ง TTL สั้นเกินไป/ยาวเกินไปถึงมีข้อเสียคนละแบบ?
3. `CNAME` ต่างจาก `A` record อย่างไร และทำไมการ chain `CNAME` ถึงมีประโยชน์?
4. เพราะเหตุใด round-robin DNS ถึงไม่ใช่ load balancing ที่สมบูรณ์แบบเมื่อเทียบกับ [Load Balancer จริง](05-reverse-proxy-load-balancer.md)?

---

ก่อนหน้า: [บทที่ 1 — OSI, TCP/IP, UDP, Socket](01-osi-tcp-ip.md) | ถัดไป: [บทที่ 3 — HTTP/1.1, HTTP/2, HTTP/3](03-http.md)
