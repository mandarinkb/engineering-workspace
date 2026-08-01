# บทที่ 4: HTTPS, TLS, Certificates, mTLS

## 4.1 HTTPS คืออะไร

**HTTPS = HTTP + TLS** ข้อมูลที่ส่งยังเป็น HTTP เหมือนเดิมทุกประการ แต่ถูกเข้ารหัสด้วย **TLS (Transport Layer Security)** ก่อนส่งออกไปบนเครือข่าย ทำให้ผู้ที่ดักฟังระหว่างทาง (man-in-the-middle) อ่านเนื้อหาไม่ได้ และตรวจสอบได้ว่าข้อมูลไม่ถูกแก้ไขระหว่างทาง

TLS ให้การรับประกัน 3 อย่าง:

1. **Confidentiality** — ข้อมูลถูกเข้ารหัส อ่านไม่ออกถ้าไม่มี key
2. **Integrity** — ตรวจจับได้ถ้าข้อมูลถูกแก้ไขระหว่างทาง
3. **Authentication** — ยืนยันได้ว่ากำลังคุยกับ server ตัวจริง ไม่ใช่ผู้แอบอ้าง

## 4.2 Symmetric vs Asymmetric Encryption

TLS ใช้การเข้ารหัสสองแบบผสมกัน เพราะแต่ละแบบมีจุดแข็งคนละด้าน:

| | Symmetric | Asymmetric |
|---|---|---|
| Key | key เดียวกันทั้งเข้ารหัสและถอดรหัส | มี key pair (public + private) |
| ความเร็ว | เร็วมาก | ช้ากว่ามาก (คำนวณหนัก) |
| ปัญหา | ต้องแชร์ key กันก่อน แต่แชร์อย่างไรให้ปลอดภัย? | ไม่ต้องแชร์ private key เลย |

**TLS แก้ปัญหานี้ด้วยการใช้ asymmetric encryption แค่ตอน "แลก key"** (key exchange) ในช่วงเริ่ม handshake แล้วหลังจากนั้นสลับไปใช้ **symmetric encryption** (เร็วกว่ามาก) สำหรับเข้ารหัสข้อมูลจริงตลอด session — เป็นการเอาจุดแข็งของทั้งสองแบบมารวมกัน

## 4.3 TLS Handshake

### TLS 1.2 (2 round-trip)

```
Client                              Server
  │──── ClientHello ─────────────────►│  (เสนอ cipher suite ที่รองรับ)
  │◄─── ServerHello ───────────────────│  (เลือก cipher suite)
  │◄─── Certificate ───────────────────│  (ส่ง certificate มายืนยันตัวตน)
  │◄─── ServerHelloDone ───────────────│
  │──── ClientKeyExchange ────────────►│  (สร้าง pre-master secret ส่งไปแบบเข้ารหัส)
  │──── ChangeCipherSpec + Finished ──►│
  │◄─── ChangeCipherSpec + Finished ───│
  │                                     │
  │      [เริ่มส่งข้อมูลจริงแบบเข้ารหัส]
```

### TLS 1.3 (1 round-trip — เร็วขึ้น)

TLS 1.3 ลดจำนวน round-trip ลงเหลือรอบเดียว โดย client เดาและส่ง key share มาพร้อมกับ ClientHello เลย (สำหรับ cipher suite ที่นิยมที่สุด) ทำให้ handshake เสร็จเร็วขึ้นเกือบเท่าตัว นอกจากนี้ TLS 1.3 ยังตัด cipher suite ที่ไม่ปลอดภัย/ล้าสมัยออกไปจำนวนมาก บังคับใช้เฉพาะ algorithm ที่ปลอดภัยเท่านั้น

## 4.4 Certificate และ Certificate Chain

Server ต้องพิสูจน์ว่าตัวเองเป็นเจ้าของ domain จริง ผ่าน **certificate** ที่ออกโดย **CA (Certificate Authority)** ที่เชื่อถือได้

```
Root CA (ฝังอยู่ใน OS/เบราว์เซอร์แล้ว)
    │ ลงนามให้
    ▼
Intermediate CA
    │ ลงนามให้
    ▼
Server Certificate (example.com)
```

เบราว์เซอร์/OS มีรายชื่อ **Root CA** ที่เชื่อถือฝังไว้อยู่แล้ว เมื่อ server ส่ง certificate มา เบราว์เซอร์จะไล่ตรวจ **chain of trust** ย้อนขึ้นไปจนถึง Root CA ที่ตัวเองเชื่อถือ ถ้าไล่ถึงและทุกลายเซ็นถูกต้อง แปลว่า certificate นี้เชื่อถือได้

### Self-Signed Certificate

Certificate ที่เซ็นด้วยตัวเอง ไม่มี CA คนกลางรับรอง — เบราว์เซอร์จะเตือน "ไม่ปลอดภัย" เพราะไม่มีใครยืนยันตัวตนให้ ใช้ได้ในงาน internal/testing แต่ไม่เหมาะกับ production ที่ผู้ใช้ทั่วไปเข้าถึง

### SNI (Server Name Indication)

ปัญหา: ก่อนที่ TLS handshake จะเสร็จ (ก่อนเห็น HTTP request จริง) server จะรู้ได้อย่างไรว่าต้องส่ง certificate ของ domain ไหน ถ้า 1 IP โฮสต์หลาย domain (virtual hosting)? **SNI** แก้ปัญหานี้โดยให้ client ระบุชื่อ domain ที่ต้องการเชื่อมต่อไปตั้งแต่ `ClientHello` (เป็น plaintext ตอนนั้น) ทำให้ server เลือก certificate ที่ถูกต้องส่งกลับได้

## 4.5 mTLS (Mutual TLS)

TLS ปกติ (ที่อธิบายมาทั้งหมด) มีแค่ **server** ที่ต้องพิสูจน์ตัวตนด้วย certificate ส่วน client ไม่ต้อง — เรียกว่า **one-way TLS**

**mTLS** บังคับให้ **ทั้งสองฝ่าย** ต้องมี certificate ยืนยันตัวตนกัน:

```
Client                              Server
  │──── ClientHello ─────────────────►│
  │◄─── ServerHello + Certificate ────│
  │◄─── CertificateRequest ───────────│  ← จุดต่างจาก TLS ปกติ
  │──── Client Certificate ──────────►│  ← client ต้องส่ง cert ของตัวเองด้วย
  │──── CertificateVerify ────────────►│
  │                                     │
  │  [ทั้งสองฝ่ายยืนยันตัวตนกันแล้ว]
```

### ใช้ mTLS ทำอะไร

mTLS ใช้เป็น **Service Identity** เป็นหลัก (เชื่อมกับ [Volume 9 — Authentication](../09-authentication/README.md)) — ในระบบ microservices ที่ service คุยกันเองภายใน network ปิด (ไม่ใช่ user ทั่วไปเข้าถึง) แต่ละ service มี certificate เป็นของตัวเอง เวลาคุยกันจะยืนยันตัวตนซึ่งกันและกันก่อนเสมอ ป้องกันไม่ให้ service แปลกปลอมที่หลุดเข้ามาใน network แอบอ้างเป็น service ที่ถูกต้องได้ — เป็นหัวใจของสถาปัตยกรรมแบบ **zero-trust network** ที่ไม่เชื่อ traffic ใดๆ โดย default แม้จะมาจากภายใน network เดียวกันก็ตาม

Service mesh (เช่น Istio, Linkerd) มักเปิดใช้ mTLS ระหว่าง service ทั้งหมดในคลัสเตอร์โดยอัตโนมัติ โดยที่ตัวแอปพลิเคชันเองไม่ต้องเขียนโค้ดจัดการ certificate เอง (จัดการผ่าน sidecar proxy แทน)

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| Symmetric vs Asymmetric | TLS ใช้ asymmetric แลก key ตอนเริ่ม แล้วสลับไป symmetric (เร็วกว่า) สำหรับข้อมูลจริง |
| TLS 1.2 vs 1.3 | 1.3 ลด round-trip เหลือรอบเดียว, ตัด cipher ที่ไม่ปลอดภัยออก |
| Certificate Chain | ไล่ตรวจลายเซ็นย้อนขึ้นไปจนถึง Root CA ที่เครื่องเชื่อถืออยู่แล้ว |
| SNI | บอก server ว่าต้องการ certificate ของ domain ไหน ก่อน handshake จะเสร็จ |
| mTLS | ยืนยันตัวตนสองทาง ใช้เป็น service identity ในสถาปัตยกรรม zero-trust |

## คำถามทบทวน

1. ทำไม TLS ถึงไม่ใช้ asymmetric encryption ตลอดทั้ง session แทนที่จะสลับไป symmetric?
2. TLS 1.3 เร็วกว่า TLS 1.2 เพราะอะไร?
3. Self-signed certificate ต่างจาก certificate ที่ CA ออกให้อย่างไร เหมาะกับสถานการณ์ไหน?
4. mTLS ต่างจาก TLS ทั่วไปอย่างไร และทำไมถึงเหมาะกับการยืนยันตัวตนระหว่าง service กับ service มากกว่า JWT อย่างเดียว?

---

ก่อนหน้า: [บทที่ 3 — HTTP/1.1, HTTP/2, HTTP/3](03-http.md) | ถัดไป: [บทที่ 5 — Reverse Proxy และ Load Balancer](05-reverse-proxy-load-balancer.md)
