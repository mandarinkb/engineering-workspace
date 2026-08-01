# บทที่ 4: mTLS

## 4.1 ทบทวนสั้นๆ: mTLS ต่างจาก TLS ทั่วไปอย่างไร

กลไก handshake ของ mTLS ถูกอธิบายไว้อย่างละเอียดแล้วใน [Volume 2 — HTTPS, TLS, Certificates, mTLS](../02-network/04-tls-security.md) บทนี้จะไม่ทวนซ้ำระดับ handshake แต่สรุปสั้นๆ เพื่อเชื่อมเข้ากับหัวข้อ Service Identity: TLS ทั่วไป (**one-way TLS**) มีแค่ **server** ที่ต้องพิสูจน์ตัวตนด้วย certificate — client ทั่วไป (เช่น browser ของ user) ไม่ต้องมี certificate ของตัวเอง ส่วน **mTLS (mutual TLS)** บังคับให้ **ทั้งสองฝ่าย** ต้องส่ง certificate มายืนยันตัวตนซึ่งกันและกัน ก่อน connection จะสำเร็จ

ความหมายเชิงสถาปัตยกรรมของความต่างนี้สำคัญมาก: one-way TLS ตอบคำถาม "เรากำลังคุยกับ server ตัวจริงไหม" ส่วน mTLS ตอบเพิ่มอีกคำถามหนึ่งคือ "**ฝั่งที่เชื่อมต่อเข้ามาคือใคร**" — คำถามหลังนี้แหละคือสิ่งที่ทำให้ mTLS กลายเป็นกลไก **Service Identity** ที่ทรงพลัง

## 4.2 mTLS เป็น Service Identity ที่แข็งแกร่งที่สุด

จาก [บทที่ 3](03-rbac-and-identity.md) เราเห็นแล้วว่า Service Identity มีตัวเลือกหลายแบบ (API key, service account, mTLS) — mTLS แข็งแกร่งกว่าตัวเลือกอื่นด้วยเหตุผลเชิงโครงสร้างดังนี้:

- **ผูกกับ transport layer โดยตรง** — การยืนยันตัวตนเกิด**ก่อน**ที่ application data แม้แต่ byte เดียวจะถูกส่ง ต่างจาก API key/JWT ที่เป็นข้อมูลระดับ application (อยู่ใน header) ซึ่งต้องรอ connection เปิดสำเร็จก่อนถึงจะตรวจสอบได้
- **Private key ไม่เคยถูกส่งผ่านเครือข่ายเลย** — การพิสูจน์ตัวตนใช้กลไก challenge-response ทาง cryptography (เซ็นข้อมูลด้วย private key ให้อีกฝ่าย verify ด้วย public key) ต่างจาก API key ที่ตัว key ถูกส่งแนบไปกับทุก request ตรงๆ (ถ้าหลุดระหว่างทางหรือหลุดจาก log ก็เอาไปใช้ปลอมตัวได้ทันที)
- **แต่ละ connection ถูกยืนยันใหม่ทุกครั้ง** ที่ transport layer ไม่ใช่แค่ครั้งแรกตอน login เหมือนบาง session-based mechanism

## 4.3 mTLS ใน Zero-Trust และ Service Mesh

### Zero-Trust Network

สถาปัตยกรรมแบบดั้งเดิมมักเชื่อ traffic ที่มาจาก "ภายใน network เดียวกัน" โดย default (เช่น traffic ใน private subnet ถือว่าปลอดภัยกว่า traffic จาก internet) — **zero-trust** ปฏิเสธสมมติฐานนี้ทั้งหมด: **ไม่เชื่อ traffic ใดๆ โดย default แม้จะมาจากภายใน network เดียวกันก็ตาม** ทุก connection ต้องพิสูจน์ตัวตนก่อนเสมอ เหตุผลคือถ้า attacker เจาะเข้ามาอยู่ใน network ได้สำเร็จแม้แค่จุดเดียว (เช่นผ่าน container ที่มีช่องโหว่) มันจะเคลื่อนที่ไปยัง service อื่นๆ ใน network เดียวกันได้อย่างอิสระถ้าไม่มีการตรวจสอบตัวตนระหว่าง service เลย — mTLS คือกลไกหลักที่ทำให้ zero-trust เป็นจริงในทางปฏิบัติ: ทุก service ต้องพิสูจน์ certificate ของตัวเองก่อนคุยกับ service อื่นได้ ต่อให้อยู่ใน network เดียวกันก็ตาม

### Service Mesh (Istio, Linkerd)

ปัญหาในทางปฏิบัติ: ถ้าให้ทุกทีมเขียนโค้ดจัดการ certificate เอง (ออก cert, rotate ก่อนหมดอายุ, handle handshake error) ในทุก service จะเป็นภาระซ้ำซ้อนมหาศาลและเสี่ยงทำผิดพลาดสูง (ลืม rotate cert แล้ว service ล่มทั้งคลัสเตอร์ตอนเที่ยงคืนเป็นเหตุการณ์ที่เกิดขึ้นจริงบ่อยในวงการ)

**Service mesh** (เช่น Istio, Linkerd) แก้ปัญหานี้ด้วยการดึงความรับผิดชอบเรื่อง mTLS ออกจากตัวแอปพลิเคชันไปไว้ที่ **sidecar proxy** (container เสริมที่รันคู่กับทุก pod คอยดักจับ traffic เข้า-ออกทั้งหมด):

```
┌─────────────────────────┐        mTLS        ┌─────────────────────────┐
│         Pod A            │◄══════════════════►│         Pod B            │
│  ┌───────┐  ┌─────────┐ │                     │ ┌─────────┐  ┌───────┐  │
│  │  App   │──│ Sidecar │ │                     │ │ Sidecar │──│  App   │  │
│  │(plain  │  │ Proxy   │ │                     │ │ Proxy   │  │(plain  │  │
│  │  HTTP) │  │(Envoy)  │ │                     │ │(Envoy)  │  │  HTTP) │  │
│  └───────┘  └─────────┘ │                     │ └─────────┘  └───────┘  │
└─────────────────────────┘                     └─────────────────────────┘
   App คุยกับ sidecar ตัวเอง                        sidecar ทำ mTLS handshake
   ด้วย plain HTTP ธรรมดา                            แทนตัวแอปทั้งหมด
   ไม่ต้องรู้เรื่อง cert เลย
```

ตัวแอปพลิเคชันคุยกับ sidecar ของตัวเองผ่าน `localhost` แบบ plain HTTP ธรรมดา — sidecar เป็นผู้จัดการ certificate ทั้งหมด (ขอ cert จาก control plane, rotate อัตโนมัติก่อนหมดอายุ, ทำ mTLS handshake จริงกับ sidecar ของ pod ปลายทาง) ผลคือทีมพัฒนาแอปไม่ต้องเขียนโค้ดเกี่ยวกับ TLS/certificate แม้แต่บรรทัดเดียว แต่ทั้งคลัสเตอร์ยังได้ mTLS ครบทุก service-to-service call

## 4.4 ตารางเปรียบเทียบ: JWT vs API Key vs mTLS สำหรับ Service-to-Service Auth

| ประเด็น | JWT | API Key | mTLS |
|---|---|---|---|
| ทำงานที่ layer ไหน | Application (HTTP header) | Application (HTTP header/query param) | Transport (TLS handshake) |
| ข้อมูลที่ส่งผ่านเครือข่าย | token ทั้งก้อน (verify ด้วย signature) | key ตรงๆ (ไม่มีกลไกป้องกันถ้าหลุด) | ไม่มี private key ถูกส่งเลย ใช้ challenge-response |
| การ revoke | ยากถ้าไม่มี refresh token infra ([บทที่ 1](01-session-and-jwt.md)) | ลบ/ปิดใช้งาน key ที่ระบบออก key ได้ทันที | revoke ผ่าน CRL/OCSP หรือสั่ง cert ใหม่ผ่าน control plane |
| ความซับซ้อนในการวางระบบ | กลาง — ต้องมี Authorization Server ออก/verify token | ต่ำสุด — สร้าง/แจก key ได้ง่าย | สูงสุด — ต้องมี CA, cert rotation, มักต้องพึ่ง service mesh ช่วย |
| เหมาะกับ | ระบบที่มี Authorization Server (OAuth2/OIDC) อยู่แล้ว, ต้องการ claim/scope ละเอียด | prototype เร็วๆ, third-party integration ง่ายๆ | zero-trust network, ระบบที่มี service มากและต้องการความปลอดภัยสูงสุดในการยืนยันตัวตนระดับ connection |

ในระบบใหญ่จริงมักใช้ผสมกัน: **mTLS** เป็นชั้นยืนยันตัวตนระดับ network/transport ระหว่าง service (ตอบว่า "connection นี้มาจาก service ที่ถูกต้องจริง") ซ้อนกับ **JWT** เป็นชั้นบอกรายละเอียดว่า "ทำ action อะไรได้บ้าง" (scope/claim ระดับ application) — ทั้งสองไม่ใช่คู่แข่งที่ต้องเลือกอย่างใดอย่างหนึ่ง แต่ทำงานคนละ layer เสริมกัน

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| mTLS vs one-way TLS | mTLS บังคับทั้งสองฝ่ายพิสูจน์ตัวตนด้วย certificate ไม่ใช่แค่ server |
| ทำไมแข็งแกร่ง | ยืนยันที่ transport layer ก่อนส่งข้อมูลจริง, private key ไม่เคยถูกส่งผ่านเครือข่าย |
| Zero-Trust | ไม่เชื่อ traffic ใดๆ โดย default แม้อยู่ network เดียวกัน mTLS ทำให้เป็นจริงได้ |
| Service Mesh | sidecar proxy จัดการ cert/mTLS แทนแอป ทีมพัฒนาไม่ต้องยุ่งกับ certificate เอง |
| JWT vs API Key vs mTLS | คนละ layer, ใช้ร่วมกันได้: mTLS ยืนยัน connection, JWT บอกรายละเอียดสิทธิ์ |

## คำถามทบทวน

1. อธิบายว่าทำไม mTLS ถึงถือว่ายืนยันตัวตนที่ layer ต่ำกว่า JWT หรือ API key และทำไมเรื่องนี้ถึงมีผลต่อความปลอดภัย?
2. Zero-trust network ต่างจากสถาปัตยกรรมแบบดั้งเดิม (เชื่อ internal network) อย่างไร mTLS เข้ามาช่วยสนับสนุนแนวคิดนี้ตรงไหน?
3. Service mesh (เช่น Istio) แก้ปัญหาอะไรให้กับทีมพัฒนาแอปพลิเคชันในเรื่อง mTLS โดยเฉพาะ?
4. ทำไมการ revoke API key ถึงทำได้ง่ายกว่าการ revoke JWT แต่ยังไม่ปลอดภัยเท่า mTLS ในแง่ที่ตัว key เองถูกส่งผ่านเครือข่ายตรงๆ?
5. ยกตัวอย่างสถานการณ์ที่ระบบควรใช้ทั้ง mTLS และ JWT ร่วมกัน พร้อมอธิบายว่าแต่ละตัวรับผิดชอบส่วนไหน

---

ก่อนหน้า: [บทที่ 3 — RBAC และ User/Service Identity](03-rbac-and-identity.md) | กลับสู่ [สารบัญ Volume 9](README.md)
