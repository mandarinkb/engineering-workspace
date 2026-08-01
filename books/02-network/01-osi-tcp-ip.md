# บทที่ 1: OSI Model, TCP/IP, UDP, Socket

## 1.1 OSI Model

**OSI (Open Systems Interconnection) Model** เป็น reference model ที่แบ่งการสื่อสารเครือข่ายออกเป็น 7 layer แต่ละ layer รับผิดชอบหน้าที่เฉพาะของตัวเอง และคุยกับ layer ที่อยู่ติดกันผ่าน interface ที่ชัดเจน ทำให้แต่ละ layer ถูกออกแบบ/แก้ไขแยกจากกันได้

```
7. Application   →  HTTP, DNS, gRPC, SMTP        (ข้อมูลที่แอปเข้าใจ)
6. Presentation  →  TLS/encryption, compression   (แปลงรูปแบบข้อมูล)
5. Session       →  session/connection management (จัดการ session)
4. Transport     →  TCP, UDP                      (ส่งข้อมูลปลายทางถึงปลายทาง)
3. Network       →  IP, routing                   (หาเส้นทางข้าม network)
2. Data Link     →  Ethernet, MAC address, switch  (ส่งข้อมูลภายใน network เดียวกัน)
1. Physical      →  สายไฟ, สัญญาณไฟฟ้า/แสง        (บิตจริงบนสื่อกลาง)
```

จำง่ายๆ: ยิ่งเลข layer สูง ยิ่งใกล้ผู้ใช้/แอปพลิเคชัน ยิ่งเลขต่ำ ยิ่งใกล้ hardware จริง

### ตัวอย่างการไล่ layer: เปิดเว็บเบราว์เซอร์หนึ่งครั้ง

1. **Application (7)** — เบราว์เซอร์สร้าง HTTP request
2. **Presentation (6)** — ข้อมูลถูกเข้ารหัสด้วย TLS (ถ้าเป็น HTTPS)
3. **Session (5)** — จัดการ session ของการเชื่อมต่อ
4. **Transport (4)** — ข้อมูลถูกห่อเป็น TCP segment พร้อม port ปลายทาง (443)
5. **Network (3)** — ห่อเป็น IP packet พร้อม IP ปลายทาง หา route
6. **Data Link (2)** — ห่อเป็น Ethernet frame พร้อม MAC address ของ next hop
7. **Physical (1)** — แปลงเป็นสัญญาณไฟฟ้า/แสง/คลื่นวิทยุส่งออกสาย/อากาศจริง

ฝั่งรับก็ทำย้อนกลับ (แกะจาก layer 1 ขึ้นไปจนถึง layer 7) กระบวนการนี้เรียกว่า **encapsulation** (ขาส่ง) และ **decapsulation** (ขารับ)

### เทียบกับ TCP/IP Model

ในทางปฏิบัติ โลกจริงใช้ **TCP/IP model** ที่มีแค่ 4 layer (ไม่ใช่ 7) ซึ่ง map กับ OSI ได้ประมาณนี้:

| TCP/IP Model | เทียบเท่า OSI Layer |
|---|---|
| Application | Application (7) + Presentation (6) + Session (5) |
| Transport | Transport (4) |
| Internet | Network (3) |
| Link | Data Link (2) + Physical (1) |

โปรโตคอลที่ engineer เจอบ่อยที่สุดในงานประจำวัน (HTTP, TCP, IP) ล้วนมาจาก model นี้

## 1.2 TCP

TCP (Transmission Control Protocol) เป็น **connection-oriented** protocol ที่รับประกันว่าข้อมูลจะถึงปลายทางครบถ้วนและ**เรียงลำดับถูกต้อง**

### 3-Way Handshake

ก่อนส่งข้อมูลจริง TCP ต้องสร้าง connection ก่อนด้วยขั้นตอน 3 จังหวะ:

```
Client                          Server
  │──────── SYN (seq=x) ────────►│
  │◄──── SYN-ACK (seq=y,ack=x+1)─│
  │──────── ACK (ack=y+1) ──────►│
  │                                │
  │        [Connection ESTABLISHED]
```

1. Client ส่ง **SYN** พร้อม initial sequence number ของตัวเอง
2. Server ตอบ **SYN-ACK** — ยืนยัน (ack) sequence ของ client และส่ง sequence number ของตัวเองมาด้วย
3. Client ส่ง **ACK** ยืนยัน sequence ของ server — เริ่มส่งข้อมูลได้

### Sequence Number และ ACK

ทุก byte ที่ส่งมี sequence number กำกับ ฝั่งรับตอบกลับด้วย ACK number บอกว่า "ได้รับข้อมูลถึง byte ไหนแล้ว ส่ง byte ถัดไปมาได้" กลไกนี้ทำให้ TCP รู้ว่า packet ไหนหายไประหว่างทาง (ไม่มี ACK กลับมาภายในเวลาที่กำหนด) แล้วทำ **retransmission** ส่งซ้ำ

### Flow Control

ฝั่งรับบอกฝั่งส่งว่า "รับได้อีกกี่ byte" ผ่าน **window size** เพื่อไม่ให้ฝั่งส่งส่งเร็วเกินกว่าที่ฝั่งรับจะประมวลผลทัน (ป้องกัน buffer overflow ที่ผู้รับ)

### Congestion Control

ต่างจาก flow control ตรงที่ congestion control ป้องกันไม่ให้ **เครือข่าย** โดยรวม (ไม่ใช่แค่ผู้รับ) โดนถล่มจนล่ม TCP เริ่มส่งข้อมูลช้าๆ แล้วค่อยๆ เพิ่มอัตราขึ้น (**slow start** — เพิ่มแบบ exponential จนถึง threshold แล้วเปลี่ยนเป็นเพิ่มทีละน้อยแบบ **congestion avoidance**) เมื่อเจอ packet loss จะลดอัตราลงทันที (สันนิษฐานว่า loss มาจาก congestion) นี่คือเหตุผลที่ connection TCP ใหม่ๆ มักช้ากว่า connection ที่เปิดค้างไว้นานแล้ว (**TCP slow start** ยังไม่ ramp up เต็มที่)

### TCP State Machine

```
CLOSED → (active open) SYN_SENT ──┐
                                    │ recv SYN-ACK, send ACK
CLOSED → (passive open) LISTEN     ▼
              │ recv SYN         ESTABLISHED
              ▼ send SYN-ACK          │
         SYN_RECEIVED ──(recv ACK)───┘
                                       │ close()
                              FIN_WAIT_1 / CLOSE_WAIT
                                       │
                              FIN_WAIT_2 / LAST_ACK
                                       │
                                  TIME_WAIT
                                       │ (รอ 2×MSL)
                                    CLOSED
```

จุดที่ engineer เจอปัญหาจริงบ่อยที่สุดคือ **TIME_WAIT** — connection ที่ปิดแล้วยังค้างอยู่ในสถานะนี้ระยะหนึ่ง (ป้องกัน packet เก่าที่มาช้าปนกับ connection ใหม่) ถ้า service เปิด/ปิด connection จำนวนมากถี่ๆ (เช่น short-lived HTTP connection จำนวนมาก) อาจเจอปัญหา "port exhaustion" เพราะ port ที่ใช้ค้างอยู่ใน TIME_WAIT เยอะเกินไป — แก้ด้วยการทำ **connection pooling / keep-alive** แทนการเปิด connection ใหม่ทุกครั้ง

## 1.3 UDP

UDP (User Datagram Protocol) เป็น **connectionless** protocol — ไม่มี handshake, ไม่มี guarantee ว่าข้อมูลจะถึง, ไม่มีการเรียงลำดับ, ไม่มี retransmission อัตโนมัติ header เล็กกว่า TCP มาก (8 bytes เทียบกับ TCP ที่อย่างน้อย 20 bytes)

### เมื่อไหร่ควรใช้ UDP

UDP เหมาะกับงานที่ **latency สำคัญกว่า reliability**:

- **DNS** — query สั้นๆ ถ้า timeout แค่ retry ใหม่ ง่ายกว่าแบก TCP handshake overhead
- **Video/audio streaming, VoIP** — packet หายไปหนึ่งเฟรมยังพอทนได้ แต่รอ retransmission แล้วภาพกระตุกไม่ดีกว่า
- **Online gaming** — ต้องการ state ล่าสุด ไม่ใช่ state เก่าที่มาช้า (การ retransmit ข้อมูลตำแหน่งเก่าไม่มีประโยชน์)
- **QUIC/HTTP/3** — สร้าง reliability ของตัวเองบน UDP อีกที เพื่อเลี่ยงข้อจำกัดบางอย่างของ TCP (ดูบทที่ 3)

## 1.4 Socket

**Socket** คือ abstraction ที่ OS มอบให้โปรแกรมใช้สื่อสารผ่านเครือข่าย ระบุตัวตนด้วยคู่ **(IP address, port)** และผูกกับ **file descriptor** เหมือน resource อื่นๆ ของ process (เชื่อมกับแนวคิด file descriptor ใน [Volume 3 — Linux](../03-linux/README.md))

### Socket API พื้นฐาน (ฝั่ง Server)

```
socket()  →  สร้าง socket
bind()    →  ผูก socket กับ (IP, port) ที่จะรอรับ
listen()  →  เปลี่ยน socket เป็นโหมดรอรับ connection, กำหนด backlog queue
accept()  →  บล็อกรอจนมี client เชื่อมต่อเข้ามา คืน socket ใหม่สำหรับคุยกับ client รายนั้น
```

### Socket API พื้นฐาน (ฝั่ง Client)

```
socket()   →  สร้าง socket
connect()  →  เริ่ม TCP handshake ไปยัง server
```

จุดที่มักสับสน: `listen()` socket ตัวเดียวใช้**รอรับ**ได้เรื่อยๆ ส่วนทุกครั้งที่มี client ใหม่ `accept()` จะคืน socket **ใหม่**สำหรับคุยกับ client รายนั้นโดยเฉพาะ — นี่คือเหตุผลที่ server รับหลาย connection พร้อมกันได้โดยไม่ชนกัน

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| OSI 7 layers | แต่ละ layer มีหน้าที่แยกกันชัดเจน; TCP/IP model 4 layer คือที่ใช้จริง |
| TCP | connection-oriented, reliable, ordered — แลกด้วย handshake overhead และ TIME_WAIT |
| UDP | connectionless, ไม่ guarantee — เหมาะงานที่ latency สำคัญกว่า reliability |
| Socket | abstraction ผูก (IP, port) เข้ากับ file descriptor; `accept()` คืน socket ใหม่ต่อ connection |

## คำถามทบทวน

1. ทำไม TCP ต้องมี 3-way handshake ไม่ใช่ 2-way?
2. Flow control กับ Congestion control ต่างกันอย่างไร (ป้องกันปัญหาคนละแบบ)?
3. อธิบายว่าทำไม service ที่เปิด/ปิด TCP connection ถี่ๆ อาจเจอปัญหา port exhaustion และวิธีแก้เบื้องต้น
4. ยกตัวอย่างงาน 2 อย่างที่เหมาะกับ UDP มากกว่า TCP พร้อมเหตุผล
5. `listen()` กับ `accept()` ต่างกันอย่างไร ทำไม server ถึงรับหลาย client พร้อมกันได้

---

ถัดไป: [บทที่ 2 — DNS](02-dns.md)
