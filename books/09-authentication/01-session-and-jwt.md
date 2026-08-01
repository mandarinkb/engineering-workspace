# บทที่ 1: Session และ JWT

## 1.1 Authentication vs Authorization

ก่อนเข้ารายละเอียด ต้องแยกสองคำนี้ให้ชัดเจน เพราะมักถูกใช้ปนกันจนสับสน:

- **Authentication (AuthN)** — "คุณคือใคร" (who are you) กระบวนการพิสูจน์ตัวตน เช่น login ด้วย username/password, ตรวจ JWT signature
- **Authorization (AuthZ)** — "คุณทำอะไรได้บ้าง" (what can you do) กระบวนการตัดสินใจว่าตัวตนที่พิสูจน์แล้วมีสิทธิ์เข้าถึง resource นั้นหรือไม่ (ดู RBAC ใน [บทที่ 3](03-rbac-and-identity.md))

Session และ JWT ในบทนี้เป็นกลไกของฝั่ง **Authentication** — คือ "วิธีที่ server จำได้ว่า request นี้มาจากใคร" ในทุก request หลังจาก login ครั้งแรก ซึ่งมีสองแนวทางหลักที่ตรงข้ามกันโดยสิ้นเชิงในเชิงสถาปัตยกรรม: **stateful (session)** กับ **stateless (JWT)**

## 1.2 Session-Based Authentication

แนวคิดคือ server เป็นฝ่าย**จำ**ว่า user คนไหน login อยู่ โดยเก็บข้อมูลไว้ที่ฝั่ง server (**session store**) แล้วส่งแค่ "ใบเบิก" สั้นๆ (**session ID**) กลับไปให้ client ถืออ้างอิงในทุก request ถัดไป

```
1) Login
Client                          Server                    Session Store
  │──── POST /login ────────────►│                              │
  │      (username, password)    │──── verify credential ──────►│ (เช่น query DB user)
  │                               │◄─── OK ──────────────────────│
  │                               │──── สร้าง session ───────────►│ session_id: abc123
  │                               │                              │  → { user_id: 42, ... }
  │◄─── Set-Cookie: sid=abc123 ──│                              │
  │      (HttpOnly; Secure)       │                              │

2) Subsequent Request
Client                          Server                    Session Store
  │──── GET /profile ────────────►│                              │
  │      Cookie: sid=abc123       │──── lookup abc123 ──────────►│
  │                               │◄─── { user_id: 42 } ─────────│
  │◄─── 200 OK (profile ของ 42) ──│                              │
```

จุดสำคัญ: **session ID เองไม่มีข้อมูลอะไรอยู่ในตัว** เป็นแค่ key แบบสุ่มที่คาดเดาไม่ได้ (ต้องสุ่มด้วย entropy สูงพอ ไม่งั้นถูก brute-force เดาได้) ข้อมูลจริงทั้งหมดอยู่ที่ server เท่านั้น — นี่คือเหตุผลที่ session **revoke ได้ทันที** แค่ลบ entry ออกจาก session store (เช่น ตอน logout หรือ admin สั่ง force logout) request ถัดไปที่ใช้ session ID เดิมจะหา entry ไม่เจอทันที

## 1.3 Session Store: In-Memory vs Redis

### In-Memory (เก็บใน process ของ application server เอง)

ง่ายที่สุด ไม่ต้องมี infrastructure เพิ่ม แต่มีปัญหาใหญ่เมื่อระบบ scale แนวนอน (หลาย instance หลัง load balancer): session ที่สร้างไว้ที่ instance A จะหาไม่เจอถ้า request ถัดไปถูกส่งไปที่ instance B — ทางแก้แบบ workaround คือ **sticky session** (บังคับ route request จาก client เดิมไป backend ตัวเดิมเสมอ ดู [Volume 2 — Reverse Proxy และ Load Balancer](../02-network/05-reverse-proxy-load-balancer.md)) แต่วิธีนี้ทำลาย statelessness ของ backend และถ้า instance นั้นล่ม session ทั้งหมดที่ sticky อยู่กับมันจะหายไปด้วย

### Redis (external session store)

แนวทางที่ถูกต้องสำหรับระบบ production คือแยก session store ออกมาเป็น service กลางที่ทุก instance เข้าถึงร่วมกันได้ — **Redis** เหมาะมากเพราะเร็ว (in-memory) และมี TTL ในตัว (`EXPIRE` ทำให้ session หมดอายุอัตโนมัติโดยไม่ต้องเขียน cleanup job เอง ดู [Volume 6 — Redis](../06-redis/README.md)) เมื่อ session อยู่ที่ Redis แทนที่จะอยู่ใน memory ของ instance ใดตัวหนึ่ง, load balancer จะส่ง request ไป instance ไหนก็ได้อย่างอิสระ — backend กลับมาเป็น stateless อย่างแท้จริง

## 1.4 Cookie Security Attributes

Session ID มักถูกส่งผ่าน **cookie** ซึ่งเบราว์เซอร์แนบให้อัตโนมัติทุก request ไปยัง domain เดียวกัน — ความสะดวกนี้เองที่ทำให้ cookie ตกเป็นเป้าโจมตีหลายรูปแบบ จึงต้องตั้ง attribute ป้องกันให้ครบ:

| Attribute | ป้องกันอะไร | อธิบาย |
|---|---|---|
| `HttpOnly` | **XSS (Cross-Site Scripting)** | ห้าม JavaScript ฝั่ง client อ่าน cookie ผ่าน `document.cookie` แม้ attacker จะฉีด script เข้าไปในหน้าเว็บได้สำเร็จ ก็ดึง session ID ออกไปไม่ได้ |
| `Secure` | **การดักฟังบนเครือข่าย (network sniffing)** | cookie จะถูกส่งเฉพาะผ่าน HTTPS เท่านั้น ป้องกันไม่ให้ session ID หลุดไปตอนส่งผ่าน HTTP แบบ plaintext (ดู [Volume 2 — TLS](../02-network/04-tls-security.md)) |
| `SameSite` | **CSRF (Cross-Site Request Forgery)** | ควบคุมว่าเบราว์เซอร์จะแนบ cookie นี้ไปกับ request ที่ยิงมาจาก**ต่าง**เว็บไซต์หรือไม่ |

### SameSite ในรายละเอียด

CSRF attack คือการที่เว็บไซต์ร้าย (evil.com) หลอกให้เบราว์เซอร์ของเหยื่อ (ที่ login ค้างอยู่ที่ bank.com) ยิง request ไป bank.com โดยอัตโนมัติ (เช่น ผ่าน `<form>` หรือ `<img>` ที่ชี้ไป bank.com) เบราว์เซอร์จะแนบ cookie ของ bank.com ไปด้วยตามปกติ ทำให้ request ดูเหมือนถูกต้องทั้งที่ user ไม่ได้ตั้งใจส่ง

- **`Strict`** — cookie จะไม่ถูกส่งเลยถ้า request มาจากต่าง site (ปลอดภัยสุด แต่ user ที่คลิก link จากที่อื่นเข้ามาจะเหมือน "ยังไม่ login" ต้อง login ใหม่)
- **`Lax`** (ค่า default ของเบราว์เซอร์ยุคใหม่) — cookie ถูกส่งกับ navigation ระดับบนสุด (top-level, เช่นคลิกลิงก์ธรรมดา) แต่ไม่ส่งกับ request ที่มาจาก `<form>` POST หรือ `<img>`/`fetch` ข้าม site — สมดุลระหว่างความปลอดภัยกับ usability
- **`None`** — ส่งคุกกี้ในทุกกรณี รวมถึงข้าม site (ต้องคู่กับ `Secure` เสมอ) จำเป็นเมื่อมี use case ที่ต้องการ cookie ข้าม site จริงๆ (เช่น embedded widget) แต่ควรหลีกเลี่ยงถ้าไม่จำเป็น

## 1.5 JWT: โครงสร้าง

**JWT (JSON Web Token)** เดินสวนทางกับ session โดยสิ้นเชิง: แทนที่จะเก็บข้อมูลไว้ที่ server แล้วส่งแค่ key อ้างอิง (session ID) JWT เก็บข้อมูล**ทั้งหมด**ไว้ในตัว token เอง แล้วรับรองความถูกต้องด้วย **digital signature** — server ฝั่งที่ตรวจสอบไม่ต้อง query อะไรเลย แค่ verify signature ก็เชื่อเนื้อหาข้างในได้ทันที

JWT ประกอบด้วย 3 ส่วน คั่นด้วย `.` แล้ว encode แต่ละส่วนด้วย Base64URL:

```
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiI0MiIsIm5hbWUiOiJTb21jaGFpIiwiaWF0IjoxNzA2NzgzMjAwLCJleHAiOjE3MDY3ODY4MDB9.TJVA95OrM7E2cBab30RMHrHDcEfxjoYZgeFONFh7HgQ

└──────────── Header ────────────┘└──────────────────── Payload ─────────────────────┘└────────── Signature ──────────┘
```

Decode แต่ละส่วน (Base64URL decode ธรรมดา ไม่ใช่การเข้ารหัส — **ใครก็อ่าน header/payload ได้ อย่าใส่ข้อมูลลับลงไป**):

```json
// Header — บอกว่าใช้ algorithm อะไร signing
{ "alg": "HS256", "typ": "JWT" }

// Payload — claim ต่างๆ
{ "sub": "42", "name": "Somchai", "iat": 1706783200, "exp": 1706786800 }

// Signature — HMACSHA256(base64UrlEncode(header) + "." + base64UrlEncode(payload), secret)
```

### Standard Claims

| Claim | ความหมาย |
|---|---|
| `iss` (issuer) | ใครเป็นคนออก token นี้ (เช่น URL ของ Authorization Server) |
| `sub` (subject) | token นี้พูดถึงใคร ปกติคือ user ID |
| `aud` (audience) | token นี้มีไว้ให้ใครใช้ (service ไหนควรยอมรับ token นี้) |
| `exp` (expiration time) | หมดอายุเมื่อไหร่ (Unix timestamp) — ฝั่งตรวจสอบต้อง reject token ที่หมดอายุแล้วเสมอ |
| `iat` (issued at) | ออกตอนไหน |
| `nbf` (not before) | ยังใช้ไม่ได้ก่อนเวลานี้ |
| `jti` (JWT ID) | unique ID ของ token นี้เอง มีประโยชน์ถ้าต้องทำ blacklist เฉพาะ token |

## 1.6 Signing Algorithm: HMAC vs RSA/ECDSA

Signature คือสิ่งที่ทำให้ JWT เชื่อถือได้ — ถ้าใครแก้ payload (เช่น เปลี่ยน `sub` เป็น user อื่น) แล้วไม่มี key ที่ถูกต้อง signature จะไม่ตรงกับ payload ใหม่ทันที

### HMAC (Symmetric — เช่น HS256)

ใช้ **secret เดียว** ทั้งตอน sign และตอน verify — ฝ่ายที่ verify ได้ต้องถือ secret เดียวกันกับฝ่ายที่ sign เท่านั้น

ปัญหา: ถ้าระบบมีหลาย service ที่ต้อง verify token (เช่น microservices หลายสิบตัว) ทุกตัวต้องถือ secret เดียวกัน — ซึ่งหมายความว่า **ทุก service ที่ verify ได้ ก็ mint token ปลอมได้ด้วย** (เพราะ key เดียวกันใช้ทำได้ทั้งสองอย่าง) ยิ่ง secret กระจายไปหลายที่มากเท่าไหร่ ความเสี่ยงที่ secret รั่วก็ยิ่งสูงขึ้น และถ้ารั่วต้อง rotate secret ทุกที่พร้อมกัน

### RSA / ECDSA (Asymmetric — เช่น RS256, ES256)

ใช้ **key pair**: **private key** ใช้ sign เท่านั้น (เก็บไว้ที่ Authorization Server ที่เดียว) และ **public key** ใช้ verify เท่านั้น (แจกจ่ายให้ service ไหนก็ได้อย่างอิสระ ไม่มีความลับต้องรักษา)

นี่คือเหตุผลที่สถาปัตยกรรม microservices ที่มีหลาย service ต้อง verify token มักเลือก asymmetric: แต่ละ service ดึง public key มา verify ได้เอง (มักผ่าน JWKS endpoint — ดู [บทที่ 2](02-oauth2-and-oidc.md)) โดยไม่มีตัวไหนสามารถ**ปลอม**token ได้เลย แม้ service นั้นจะถูกโจมตีจนโค้ด/memory หลุดออกไปก็ตาม เพราะไม่มี private key อยู่ในนั้นตั้งแต่แรก — แยก "อำนาจออก token" (mint) ออกจาก "อำนาจตรวจ token" (verify) ได้อย่างเด็ดขาด

## 1.7 Stateless Trade-Off: ปัญหา Revocation

ข้อดีที่สุดของ JWT (stateless — ไม่ต้อง query อะไรเพื่อ verify) กลับเป็นข้อเสียที่ใหญ่ที่สุดในเวลาเดียวกัน: **เมื่อ sign ออกไปแล้ว ไม่มีทาง "เพิกถอน" token นั้นก่อนหมดอายุได้** เพราะไม่มี central store ให้ไปลบ entry ทิ้ง (ต่างจาก session ที่ลบ entry แล้วจบทันที) — ต่อให้ user กด logout, บัญชีถูกแฮ็ก, หรือ admin ต้องการ ban user คนหนึ่งทันที token เก่าที่ยังไม่หมดอายุก็ยังใช้งานได้ต่อไปจนกว่า `exp` จะถึง

### ทางแก้ที่ใช้จริง: Short-Lived Access Token + Refresh Token

แนวทางปฏิบัติที่นิยมที่สุดคือประนีประนอมระหว่างสองโลก:

- **Access Token** — เป็น JWT อายุสั้นมาก (5–15 นาที) ใช้แนบไปกับทุก request เพื่อเข้าถึง resource — อายุสั้นทำให้ "หน้าต่างความเสี่ยง" ถ้า token หลุดไปมีจำกัด
- **Refresh Token** — อายุยาว (วัน/สัปดาห์) เก็บไว้ที่ server-side store (เช่น Redis หรือ DB) จึง **revoke ได้ทันที** เหมือน session ใช้แค่ตอนขอ access token ใหม่ ไม่ได้แนบไปกับทุก request

```
Client                                  Auth Server
  │──── login ────────────────────────►│
  │◄─── access_token (15m) +           │
  │      refresh_token (7d) ───────────│

  │  ... ใช้ access_token เรียก API ไปเรื่อยๆ จนหมดอายุ ...

  │──── POST /token/refresh ───────────►│  (ส่ง refresh_token)
  │                                      │── ตรวจใน store: ยัง valid อยู่ไหม?
  │◄─── access_token ใหม่ (15m) ────────│     ถ้า revoke ไปแล้ว → reject ทันที
```

ผลลัพธ์: ระบบได้ความเร็วของ stateless verification สำหรับ 99% ของ request (access token) ในขณะที่ยังมีจุดควบคุมการ revoke ได้จริงอยู่ที่ refresh token — ความเสี่ยงสูงสุดที่เหลืออยู่คือ access token ที่ถูกขโมยไปจะยังใช้ได้จนกว่าจะหมดอายุ (นาทีเดียว ไม่ใช่วัน)

## 1.8 Session vs JWT: ตารางตัดสินใจ

| ประเด็น | Session | JWT |
|---|---|---|
| ที่เก็บข้อมูล | server-side store (memory/Redis) | ในตัว token เอง |
| การ verify | query store ทุกครั้ง | verify signature อย่างเดียว ไม่ query |
| Revoke ก่อนหมดอายุ | ทำได้ทันที (ลบ entry) | ทำไม่ได้โดยตรง ต้องมี infra เสริม (blacklist/refresh token) |
| Scale ข้ามหลาย service | ต้องมี shared store ที่ทุกตัวเข้าถึงได้ | verify ได้อิสระด้วย public key ไม่ต้องคุย service อื่น |
| ขนาด request | เล็ก (แค่ session ID) | ใหญ่กว่า (payload ทั้งก้อนแนบไปทุก request) |
| เหมาะกับ | web app แบบ monolith/few services ที่มี shared store อยู่แล้ว | microservices หลายตัว, mobile/third-party client ที่ verify แบบกระจายศูนย์ |

ไม่มีคำตอบที่ถูกเสมอไป — ระบบจริงจำนวนมากใช้ผสมกัน: session (หรือ refresh token ที่ revoke ได้) เป็นชั้นควบคุมระยะยาว และ short-lived JWT เป็นชั้นตรวจสอบเร็วในแต่ละ request

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| Session | server จำ state ไว้เอง ส่งแค่ session ID กลับ, revoke ได้ทันที แต่ต้องมี shared store เมื่อ scale |
| Cookie Attribute | `HttpOnly` กัน XSS อ่าน cookie, `Secure` กันดักฟัง, `SameSite` กัน CSRF |
| JWT | header.payload.signature, ข้อมูลอยู่ในตัว token เอง verify ได้โดยไม่ query |
| HMAC vs RSA/ECDSA | symmetric ต้องแชร์ secret (verify ได้ = mint ได้), asymmetric แยกอำนาจ sign กับ verify ออกจากกัน |
| Revocation | JWT stateless แต่ revoke ก่อนหมดอายุไม่ได้ตรงๆ → แก้ด้วย short-lived access token + revocable refresh token |

## คำถามทบทวน

1. ทำไม session ถึง revoke ได้ทันที แต่ JWT ทำไม่ได้โดยตรง?
2. `SameSite=Lax` กับ `SameSite=Strict` ต่างกันอย่างไร แต่ละแบบแลกอะไรไปกับความปลอดภัยที่ได้เพิ่มขึ้น?
3. ทำไมระบบที่มีหลาย microservices ต้อง verify JWT ถึงมักเลือกใช้ RS256 แทน HS256?
4. อธิบาย refresh token pattern และบอกว่ามันแก้ปัญหา revocation ของ JWT อย่างไร โดยที่ยังคงข้อดีของ stateless ไว้บางส่วน
5. ในสถานการณ์ที่ backend มีแค่ 1 instance ไม่มีแผน scale จะเลือก session หรือ JWT และเพราะอะไร?

---

ถัดไป: [บทที่ 2 — OAuth2 และ OIDC](02-oauth2-and-oidc.md)
