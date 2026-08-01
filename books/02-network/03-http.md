# บทที่ 3: HTTP/1.1, HTTP/2, HTTP/3

HTTP คือ application-layer protocol ที่ backend engineer เจอบ่อยที่สุด แต่ละเวอร์ชันแก้ปัญหาของเวอร์ชันก่อนหน้า การเข้าใจว่า "แก้ปัญหาอะไร" สำคัญกว่าจำ spec ได้ทั้งหมด

## 3.1 HTTP/1.1

### Request/Response แบบ Text

HTTP/1.1 เป็น protocol แบบ plain text อ่านง่ายด้วยตาเปล่า:

```
GET /api/users/42 HTTP/1.1
Host: api.example.com
Accept: application/json

HTTP/1.1 200 OK
Content-Type: application/json
Content-Length: 27

{"id":42,"name":"Somchai"}
```

### Keep-Alive

HTTP/1.0 เปิด TCP connection ใหม่ทุกครั้งที่ request — แพงมากเพราะต้องทำ 3-way handshake ใหม่ทุกครั้ง (บทที่ 1) HTTP/1.1 แก้ด้วย **`Connection: keep-alive`** เป็นค่า default ทำให้ใช้ TCP connection เดิมส่ง request หลายครั้งได้

### Head-of-Line Blocking (Application Layer)

แม้จะมี keep-alive แต่ HTTP/1.1 ยังส่ง request/response **ทีละตัวเรียงลำดับ**บน connection เดียว (แม้จะมี pipelining ใน spec แต่แทบไม่มีใครใช้จริงเพราะ implementation ซับซ้อนและมีปัญหา) request ที่ช้าจะบล็อก request ที่มาหลังบน connection เดียวกัน — เบราว์เซอร์แก้ปัญหานี้แบบ workaround ด้วยการเปิดหลาย TCP connection พร้อมกัน (ปกติจำกัด ~6 connection ต่อ host) ซึ่งก็มี cost ของมันเอง (handshake, TCP slow start ซ้ำหลายรอบ)

## 3.2 HTTP/2

### Multiplexing

HTTP/2 แก้ head-of-line blocking ระดับ application ด้วยการส่งหลาย request/response พร้อมกันบน **TCP connection เดียว** ผ่านแนวคิด **stream** — ข้อมูลถูกตัดเป็น **frame** เล็กๆ สลับกันส่งบน connection เดียว แล้วประกอบกลับที่ปลายทาง

```
HTTP/1.1: [==Req1==][==Req2==][==Req3==]   (เรียงลำดับ ทีละตัว)

HTTP/2:   [Req1-f1][Req2-f1][Req1-f2][Req3-f1][Req2-f2]...  (สลับ frame กัน)
          ทั้งหมดอยู่บน TCP connection เดียว
```

### Binary Framing

ต่างจาก HTTP/1.1 ที่เป็น text, HTTP/2 ใช้ **binary framing layer** ทำให้ parse เร็วขึ้นและ error-prone น้อยลง (ไม่ต้องมานั่ง parse text ที่ format หลากหลาย)

### Header Compression (HPACK)

Header ของ HTTP request/response ซ้ำกันเยอะมากในแต่ละ request (cookie, user-agent, ฯลฯ) HTTP/2 ใช้ **HPACK** บีบอัด header โดยอ้างอิง header ที่เคยส่งไปแล้ว (dynamic table) ลด overhead ต่อ request ได้มาก

### Server Push (เกือบเลิกใช้แล้ว)

แนวคิดให้ server ส่งข้อมูลที่รู้ว่า client จะต้องขอแน่ๆ (เช่น CSS/JS ของหน้าเว็บ) มาให้ล่วงหน้าโดยไม่ต้องรอ client ขอ — ในทางปฏิบัติได้ผลลัพธ์ไม่คุ้มค่าเท่าที่คาดและ implement ยาก เบราว์เซอร์หลักๆ เลิกรองรับไปแล้ว (Chrome ถอดออกปี 2022) ควรรู้ไว้ว่ามีแนวคิดนี้แต่ไม่ต้องคาดหวังว่าจะได้ใช้จริง

## 3.3 HTTP/3

### ปัญหาที่เหลืออยู่ใน HTTP/2

HTTP/2 แก้ head-of-line blocking ระดับ **application** แล้ว แต่ยังวิ่งอยู่บน **TCP** ซึ่งถ้า TCP segment หนึ่งหายไป **ทุก stream บน connection นั้นต้องรอ retransmission** (เพราะ TCP รับประกัน ordering ระดับ byte stream ทั้ง connection) กลายเป็น head-of-line blocking ระดับ **transport layer** แทน

### QUIC

HTTP/3 แก้ปัญหานี้ด้วยการเปลี่ยน transport จาก TCP เป็น **QUIC** ซึ่งวิ่งอยู่บน **UDP** (บทที่ 1) แต่ QUIC สร้างกลไก reliability ของตัวเองขึ้นมาใหม่ โดยจัดการ stream แยกจากกันจริงๆ ระดับ transport — segment ของ stream หนึ่งหายไป ไม่กระทบ stream อื่นบน connection เดียวกันอีกต่อไป

### ข้อดีอื่นของ QUIC

- **0-RTT / 1-RTT connection setup** — รวม TCP handshake กับ TLS handshake (บทที่ 4) เข้าด้วยกัน ลดจำนวน round-trip ก่อนเริ่มส่งข้อมูลจริง
- **Connection migration** — ถ้า client เปลี่ยนเครือข่าย (เช่น สลับจาก WiFi ไป 4G) connection ยังคงอยู่ได้เพราะ QUIC ระบุ connection ด้วย Connection ID ไม่ใช่ (IP, port) tuple แบบ TCP

## 3.4 สรุปเปรียบเทียบ

| | HTTP/1.1 | HTTP/2 | HTTP/3 |
|---|---|---|---|
| Transport | TCP | TCP | QUIC (บน UDP) |
| Format | Text | Binary framing | Binary framing |
| Multiplexing | ไม่มี (ต้องเปิดหลาย connection) | มี (หลาย stream บน 1 connection) | มี (stream อิสระจากกันจริง) |
| Head-of-line blocking | มี (application level) | มีที่ transport level (TCP) | แก้แล้วทั้งสองระดับ |
| Header compression | ไม่มี | HPACK | QPACK |

## คำถามทบทวน

1. อธิบายว่า head-of-line blocking คืออะไร และ HTTP/2 แก้ปัญหานี้ "บางส่วน" อย่างไร ยังเหลือปัญหาอะไรอยู่?
2. ทำไม HTTP/3 ถึงเลือกใช้ UDP เป็น transport แทนที่จะปรับปรุง TCP ต่อไป?
3. HPACK ช่วยลด overhead อย่างไร ทำไม header ของ HTTP request ถึงบีบอัดได้ดี?
4. Connection migration ของ QUIC มีประโยชน์กับ mobile client อย่างไร?

---

ก่อนหน้า: [บทที่ 2 — DNS](02-dns.md) | ถัดไป: [บทที่ 4 — HTTPS, TLS, mTLS](04-tls-security.md)
