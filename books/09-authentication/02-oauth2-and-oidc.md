# บทที่ 2: OAuth2 และ OIDC

## 2.1 OAuth2 คือ Authorization ไม่ใช่ Authentication

ความเข้าใจผิดที่พบบ่อยที่สุดเกี่ยวกับ OAuth2 คือคิดว่ามันคือ "ระบบ login" — จริงๆ แล้ว **OAuth2 เป็น protocol สำหรับ Authorization (มอบสิทธิ์) เท่านั้น** ไม่ใช่ Authentication (พิสูจน์ตัวตน) โจทย์ที่ OAuth2 แก้คือ: "ทำอย่างไรให้แอปหนึ่ง (เช่น เว็บจัดการปฏิทิน third-party) เข้าถึงข้อมูลของ user ใน service อีกที่หนึ่ง (เช่น Google Calendar) ได้ **โดยไม่ต้องให้ user บอก password ของ Google กับแอปนั้นโดยตรง**"

ผลลัพธ์ของ OAuth2 flow คือ **Access Token** ที่พิสูจน์ว่า "แอปนี้ได้รับอนุญาตให้ทำ action ที่ระบุ (scope) แทน user คนนี้" — มันไม่ได้บอกเลยว่า "user คนนี้คือใคร" ในความหมายที่ตัวแอปเองจะเอาไปสร้างหน้า profile/login session ได้ (ปัญหานี้ OIDC เป็นคนแก้ ดู 2.7)

## 2.2 บทบาท 4 อย่างใน OAuth2

```
┌──────────────┐        1. ขอสิทธิ์          ┌──────────────────────┐
│Resource Owner│◄───────────────────────────►│  Authorization Server │
│   (user)     │        (login + ยินยอม)      │   (เช่น Google, Auth0) │
└──────┬───────┘                              └───────────┬───────────┘
       │                                                    │
       │ 2. redirect กลับพร้อม code                          │ 4. ออก Access Token
       ▼                                                    ▼
┌──────────────┐        3. แลก code เป็น token  ┌──────────────────────┐
│    Client    │────────────────────────────────►│  Authorization Server │
│ (third-party │◄────────────────────────────────│                        │
│     app)     │        Access Token             └───────────────────────┘
└──────┬───────┘
       │ 5. เรียก API พร้อมแนบ Access Token
       ▼
┌──────────────────────┐
│   Resource Server     │   (เช่น Google Calendar API)
│ ตรวจ token → ตอบข้อมูล │
└──────────────────────┘
```

- **Resource Owner** — เจ้าของข้อมูล ปกติคือ user คนหนึ่ง เป็นคนกด "อนุญาต" (consent)
- **Client** — แอปที่ต้องการเข้าถึงข้อมูล (เช่น web app, mobile app, backend service) **ไม่ใช่ web browser** — คำว่า client ใน OAuth2 หมายถึงแอปพลิเคชันที่ลงทะเบียนไว้กับ Authorization Server
- **Authorization Server (AS)** — ผู้ออก token หลังจากตรวจสอบตัวตน resource owner และขอความยินยอมแล้ว (เช่น Keycloak, Auth0, Google Identity Platform)
- **Resource Server** — API ที่เก็บข้อมูลจริง ตรวจ Access Token ก่อนตอบ request (มักเป็นคนละ service กับ AS)

## 2.3 Grant Type: Authorization Code + PKCE

นี่คือ grant type ที่ใช้จริงมากที่สุดสำหรับ web app และ mobile app — และสมควรได้รับความเข้าใจลึกที่สุดในบทนี้ เพราะเป็น flow ที่ซับซ้อนที่สุดและมีเหตุผลด้าน security รองรับทุกขั้นตอน

### ทำไมต้องมี "code" คั่นกลาง แทนที่จะส่ง Access Token ตรงๆ

ในขั้น redirect กลับจาก Authorization Server ไปยัง client (ขั้นตอน 2 ในแผนภาพ 2.2) ข้อมูลวิ่งผ่าน **browser URL** ซึ่งหลุดง่าย (ติดอยู่ใน browser history, log ของ proxy, referrer header) OAuth2 จึงออกแบบให้ขั้นนี้ส่งแค่ **authorization code แบบใช้ครั้งเดียวและอายุสั้นมาก** (ไม่กี่วินาที) แทนที่จะส่ง Access Token ตัวจริง แล้วให้ client เอา code นี้ไปแลก Access Token อีกทีผ่าน **back-channel** (server-to-server, ไม่ผ่าน browser) ซึ่งปลอดภัยกว่ามาก

### ทำไมต้องมี PKCE (Proof Key for Code Exchange)

Client มีสองประเภท: **confidential client** (backend service ที่เก็บ client secret ไว้อย่างปลอดภัยบน server ได้) และ **public client** (mobile app, single-page app — โค้ดฝั่ง client ที่ทุกคนแกะดูได้ เก็บ secret แบบปลอดภัยไม่ได้เลย เพราะ secret จะถูกฝังไปกับ binary/JS ที่ทุกคนเข้าถึงได้)

ปัญหา: ถ้า public client ใช้ Authorization Code flow ปกติ attacker ที่ดักจับ authorization code ได้ (เช่น ผ่าน malicious app บน mobile ที่ดักจับ custom URL scheme) จะเอา code นั้นไปแลก Access Token เองได้ทันที เพราะไม่มี secret อะไรมาป้องกันตอนแลก code

**PKCE แก้ปัญหานี้ด้วยการสร้าง secret แบบชั่วคราวที่สุ่มขึ้นมาเฉพาะ request นี้ (ไม่ต้องฝังไว้ล่วงหน้าในโค้ด):**

```
1) Client สุ่ม code_verifier (string สุ่มยาวๆ)
   คำนวณ code_challenge = SHA256(code_verifier)

2) Client ──── /authorize?...&code_challenge=xxx&code_challenge_method=S256 ───► Authorization Server
                                                     (ส่งแค่ challenge ที่ hash แล้ว)

3) Authorization Server ──── redirect กลับพร้อม authorization code ───► Client

4) Client ──── /token  code + code_verifier (ตัวจริง ไม่ hash) ───► Authorization Server
                                                     │
                                    Authorization Server ตรวจ: SHA256(code_verifier) == code_challenge ที่บันทึกไว้ตอนขั้น 2?
                                                     │
                                              ตรงกัน → ออก Access Token
                                              ไม่ตรง → reject
```

แม้ attacker จะดักจับ authorization code ได้ระหว่างทาง (ขั้นตอน 3) ก็เอาไปแลก token ไม่ได้ เพราะไม่มี `code_verifier` ตัวจริง (ซึ่งไม่เคยถูกส่งผ่านทางไหนที่ดักได้เลย จนกว่าจะถึงขั้นตอน 4 ที่เป็น back-channel) — PKCE จึงกลายเป็นมาตรฐาน**บังคับ**สำหรับ public client ในปัจจุบัน และแนะนำให้ใช้แม้กับ confidential client ด้วยเพื่อความปลอดภัยเพิ่ม (defense in depth)

## 2.4 Grant Type: Client Credentials

ใช้เมื่อไม่มี "user" อยู่ในสมการเลย — เป็นการสื่อสารแบบ **service-to-service** ล้วนๆ (เช่น batch job เรียก internal API, service A เรียก service B) ตัว client เองคือผู้ยืนยันตัวตนด้วย client ID + client secret ของตัวเอง แลก Access Token มาใช้ตรง ไม่มีขั้นตอน redirect ผ่าน browser หรือ consent ใดๆ:

```
Client (service A)                    Authorization Server
      │──── POST /token ─────────────────►│
      │      grant_type=client_credentials │
      │      client_id, client_secret      │
      │◄─── Access Token ───────────────────│
      │
      │──── เรียก API พร้อม Access Token ───► Resource Server (service B)
```

เชื่อมโยงกับแนวคิด **Service Identity** ใน [บทที่ 3](03-rbac-and-identity.md) — grant type นี้คือกลไก OAuth2 มาตรฐานสำหรับให้ service พิสูจน์ตัวตนของ "ตัวมันเอง" ไม่ใช่ตัวแทนของ user คนไหน

## 2.5 Grant Type: Refresh Token

ไม่ใช่วิธี "ขอสิทธิ์" ใหม่ แต่เป็นวิธี "ต่ออายุ" Access Token โดยไม่ต้องให้ user login/ยินยอมใหม่ทุกครั้ง (ดูรายละเอียด flow ใน [บทที่ 1 หัวข้อ 1.7](01-session-and-jwt.md#17-stateless-trade-off-ปัญหา-revocation)) Authorization Server จะออก refresh token คู่กับ access token ตั้งแต่รอบแรกที่ user ยินยอม แล้ว client เก็บ refresh token ไว้ใช้ขอ access token ใหม่เรื่อยๆ จนกว่า refresh token จะถูก revoke หรือหมดอายุ

## 2.6 เลือก Grant Type ให้เหมาะกับสถานการณ์

| สถานการณ์ | Grant Type ที่เหมาะ | เหตุผล |
|---|---|---|
| Web app ที่มี backend (server-side rendered) | Authorization Code (+ PKCE แนะนำ) | มี user ยินยอม, มี back-channel ที่ปลอดภัยพอเก็บ secret ได้ |
| Mobile app / Single-Page App | Authorization Code + PKCE (**บังคับ**) | public client เก็บ secret ไม่ได้ ต้องพึ่ง PKCE แทน |
| Service เรียก service (ไม่มี user) | Client Credentials | ไม่มี resource owner ให้ขอความยินยอม เป็นการยืนยันตัวตนของ service เอง |
| ต่ออายุ session โดยไม่ให้ user login ซ้ำ | Refresh Token | ใช้คู่กับ flow อื่นเสมอ ไม่ใช่ flow เริ่มต้น |

## 2.7 OIDC: Identity Layer บนของ OAuth2

**OpenID Connect (OIDC)** ไม่ใช่ protocol คู่แข่งของ OAuth2 แต่เป็น**ชั้นเพิ่ม (extension layer)** ที่สร้างอยู่**บน**ของ OAuth2 พอดี — ใช้ flow เดียวกันทุกอย่าง (Authorization Code, PKCE ฯลฯ) เพียงแค่เพิ่ม scope พิเศษชื่อ `openid` เข้าไปตอนขอสิทธิ์

**Insight สำคัญที่สุดของบทนี้**: OAuth2 อย่างเดียวพิสูจน์ได้แค่ว่า "**client** นี้ได้รับอนุญาตให้กระทำการแทน user คนหนึ่ง" — มันไม่ได้บอกตัวแอป client เองเลยว่า **user คนนั้นคือใคร**จริงๆ (Access Token ใน OAuth2 บริสุทธิ์มักเป็นแค่ opaque string ที่มีความหมายกับ Resource Server เท่านั้น ตัว client อ่านไม่ออกด้วยซ้ำว่าใครคือเจ้าของ) หลายระบบยุคแรกๆ เอา OAuth2 ไปใช้ทำ "Login with X" อย่างผิดๆ โดยตีความว่า "ขอ Access Token สำเร็จ = user login สำเร็จ" ซึ่งมีช่องโหว่ด้าน security (เช่น token confusion — Access Token ที่ออกให้แอปหนึ่งไปใช้ปลอมเป็นการ login กับอีกแอปได้)

OIDC แก้ปัญหานี้ตรงๆ ด้วยการเพิ่ม **ID Token** เข้ามาเป็นผลลัพธ์ที่สามคู่กับ Access Token — ID Token คือสิ่งที่มีไว้พิสูจน์ตัวตน user ให้กับ**ตัว client เอง** โดยเฉพาะ

## 2.8 ID Token vs Access Token

| | Access Token | ID Token |
|---|---|---|
| มีไว้ให้ใครอ่าน | Resource Server | Client (ตัวแอปเอง) |
| รูปแบบ | มักเป็น opaque string หรือ JWT (แล้วแต่ AS) | เป็น **JWT เสมอ** ตาม OIDC spec |
| เนื้อหา | scope/permission ที่ client ได้รับ | claim เกี่ยวกับตัวตน user (`sub`, `name`, `email`, `auth_time`) |
| ใช้ทำอะไร | แนบไปกับทุก request เรียก API | ตรวจครั้งเดียวตอน login เพื่อสร้าง session/รู้จักว่า user คือใคร ไม่ควรส่งไปเรียก API อื่น |
| Audience (`aud`) | Resource Server | Client ID ของแอปที่ขอ |

ตัว client ต้อง verify ID Token เหมือน JWT ทั่วไป (ตรวจ signature, `exp`, และที่สำคัญคือตรวจ `aud` ให้ตรงกับ client ID ของตัวเอง — ป้องกัน token confusion ที่กล่าวถึงข้างต้น ดูกลไก signature ใน [บทที่ 1](01-session-and-jwt.md))

## 2.9 Discovery Document และ UserInfo Endpoint

### Discovery Document

Authorization Server ที่รองรับ OIDC ต้อง publish ไฟล์ JSON มาตรฐานไว้ที่ path คงที่:

```
GET https://auth.example.com/.well-known/openid-configuration

{
  "issuer": "https://auth.example.com",
  "authorization_endpoint": "https://auth.example.com/authorize",
  "token_endpoint": "https://auth.example.com/token",
  "userinfo_endpoint": "https://auth.example.com/userinfo",
  "jwks_uri": "https://auth.example.com/.well-known/jwks.json",
  "scopes_supported": ["openid", "profile", "email"]
}
```

Client ไม่ต้อง hardcode URL ของทุก endpoint เอง — ดึง discovery document ครั้งเดียวแล้วรู้ว่าต้องคุยกับ endpoint ไหนสำหรับงานอะไร รวมถึง `jwks_uri` ที่ใช้ดึง **public key** มา verify signature ของ ID Token (ตรงกับหลักการ asymmetric signing ใน [บทที่ 1 หัวข้อ 1.6](01-session-and-jwt.md#16-signing-algorithm-hmac-vs-rsaecdsa))

### UserInfo Endpoint

จุดที่ client เรียกโดยแนบ Access Token เพื่อขอข้อมูล profile ของ user เพิ่มเติม (นอกเหนือจากที่มีอยู่แล้วใน ID Token) เหมาะกับกรณีที่ ID Token ตั้งใจให้เบาที่สุด (มีแค่ `sub`) แล้วดึงรายละเอียดเพิ่มเมื่อจำเป็นจริงๆ ผ่าน endpoint นี้แทน

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| OAuth2 | protocol เรื่อง **authorization** (มอบสิทธิ์) ไม่ใช่ authentication |
| 4 บทบาท | Resource Owner, Client, Authorization Server, Resource Server |
| Authorization Code + PKCE | flow หลักของ web/mobile, code แลก token ผ่าน back-channel, PKCE ป้องกัน public client ที่เก็บ secret ไม่ได้ |
| Client Credentials | service-to-service ล้วนๆ ไม่มี user ในสมการ |
| OIDC | ชั้นเพิ่มบน OAuth2 เพื่อแก้ปัญหา "OAuth2 ไม่บอกว่า user คือใคร" |
| ID Token vs Access Token | ID Token มีไว้ให้ client อ่านรู้จัก user, Access Token มีไว้ให้ Resource Server ตรวจสิทธิ์ |

## คำถามทบทวน

1. ทำไม OAuth2 อย่างเดียวถึงไม่เพียงพอสำหรับทำระบบ "Login with X"? OIDC เข้ามาแก้ตรงจุดไหน?
2. อธิบายว่าทำไม Authorization Code flow ถึงไม่ส่ง Access Token กลับมาทาง browser โดยตรง ต้องผ่าน code คั่นกลางก่อน
3. PKCE ป้องกันการโจมตีแบบไหน และทำไมถึงจำเป็นกับ public client (เช่น mobile app) มากเป็นพิเศษ?
4. Client Credentials grant ต่างจาก Authorization Code grant ตรงไหนบ้าง เหมาะกับสถานการณ์แบบไหน?
5. ถ้า client อ่านค่าจาก ID Token โดยไม่ตรวจ `aud` เลย จะเกิดความเสี่ยงอะไรได้บ้าง?

---

ก่อนหน้า: [บทที่ 1 — Session และ JWT](01-session-and-jwt.md) | ถัดไป: [บทที่ 3 — RBAC และ User/Service Identity](03-rbac-and-identity.md)
