# บทที่ 2: Consumer และ Plugins

## 2.1 Consumer คือ "ตัวตน" ของผู้เรียก API

[บทที่ 1](01-routing.md) พูดถึง Route และ Upstream ซึ่งทั้งคู่ตอบคำถามฝั่ง **ปลายทาง** — "request นี้ควรไปที่ไหน" **Consumer** ตอบคำถามคนละด้านโดยสิ้นเชิง: "**ใครเป็นคนเรียก request นี้เข้ามา**" — Consumer แทนตัวตนของแอป/ทีม/partner ที่เรียกใช้ API ผ่าน gateway (ไม่ใช่ตัวตนของ user ปลายทางที่นั่งอยู่หน้าจอ — เทียบได้กับ **Service Identity** ที่อธิบายไว้ใน [Volume 9 — บทที่ 3](../09-authentication/03-rbac-and-identity.md))

การแยก Consumer ออกจาก Route ทำให้ gateway ตอบคำถามสองมิติพร้อมกันได้อย่างอิสระต่อกัน: route ตัวเดียวกัน (`/api/orders/*`) อาจถูกเรียกโดยหลาย consumer (mobile app, partner integration, internal dashboard) ที่แต่ละตัวมี rate limit/สิทธิ์ต่างกัน โดยไม่ต้องสร้าง route แยกสำหรับแต่ละ consumer

## 2.2 ผูก Consumer เข้ากับ Auth Plugin

Consumer เพียวๆ ยังไม่มีความหมายอะไรจนกว่าจะผูกเข้ากับ **auth plugin** ที่ทำหน้าที่ระบุว่า request ที่เข้ามานี้ "เป็น" consumer ตัวไหน:

```json
PUT /apisix/admin/consumers
{
  "username": "mobile-app",
  "plugins": {
    "key-auth": {
      "key": "mobile-app-secret-key-xxxx"
    }
  }
}
```

```json
PUT /apisix/admin/consumers
{
  "username": "partner-team-a",
  "plugins": {
    "jwt-auth": {
      "key": "partner-team-a-issuer",
      "algorithm": "RS256"
    }
  }
}
```

แล้วเปิดใช้ auth plugin นั้นที่ระดับ Route (หรือ Service ที่เป็นกลุ่ม route):

```json
PUT /apisix/admin/routes/1
{
  "uri": "/api/orders/*",
  "plugins": { "key-auth": {} },
  "upstream": { "...": "..." }
}
```

เมื่อ request เข้ามาที่ route นี้พร้อม header `apikey: mobile-app-secret-key-xxxx` APISIX จะจับคู่กับ consumer `mobile-app` ได้ทันที ทำให้ plugin อื่น (เช่น rate limit ใน [บทที่ 3](03-auth-and-rate-limit.md)) เอา identity นี้ไปใช้กำหนดขอบเขตต่อได้ (เช่น "จำกัด 100 req/min **ต่อ consumer**")

## 2.3 Plugin Architecture: Pluggable, Ordered Execution Chain

APISIX ออกแบบทุกความสามารถ (auth, rate limit, transform, logging ฯลฯ) เป็น **plugin** ที่เปิด/ปิด/ตั้งค่าแยกอิสระต่อ route ได้ — request หนึ่งตัวที่วิ่งผ่าน gateway จะไหลผ่าน**ลำดับ**ของ plugin ที่เปิดใช้งานไว้ ก่อนไปถึง upstream จริง (และ response ก็ไหลผ่าน plugin ฝั่งขาออกก่อนกลับไปหา client เช่นกัน):

```
Client Request
      │
      ▼
┌─────────────┐   ┌─────────────┐   ┌──────────────────┐   ┌──────────────┐
│  key-auth /  │──►│ limit-count │──►│ request-transform │──►│   Upstream    │
│  jwt-auth    │   │ (rate limit)│   │  (แก้ header/body) │   │  (backend)    │
└─────────────┘   └─────────────┘   └──────────────────┘   └──────┬───────┘
   ระบุตัวตน           จำกัดสิทธิ์          ปรับ request                │
   consumer            ตาม consumer                                    │
                                                                        ▼
                                                              response-transform, cors,
                                                              logging ฯลฯ ก่อนตอบกลับ client
```

แต่ละ plugin มี `priority` กำหนดลำดับการทำงาน (เช่น plugin auth ต้องรันก่อน plugin ที่ต้องรู้ว่า request มาจาก consumer ไหนเสมอ) และแต่ละ plugin ตัดสินใจได้ว่าจะ**ปล่อยผ่าน**ไป plugin ถัดไป หรือ**ตัดจบ**ทันที (short-circuit) เช่น `key-auth` ที่ตรวจ key ไม่ผ่านจะตอบ `401` กลับไปเลยโดยไม่ปล่อยให้ request ไปถึง upstream

## 2.4 Plugin ที่ใช้บ่อย

| Plugin | หน้าที่ |
|---|---|
| `cors` | จัดการ CORS header (`Access-Control-Allow-Origin` ฯลฯ) ให้อัตโนมัติ ไม่ต้องเขียนโค้ดจัดการเองในทุก service ที่ต้องรองรับ browser เรียกข้าม origin |
| `request-transformer` | แก้ไข request ก่อนส่งถึง upstream — เพิ่ม/ลบ/แก้ header, เปลี่ยน query param, แม้กระทั่งแก้ body |
| `response-transformer` | แก้ไข response ก่อนส่งกลับ client — เช่น ลบ header ที่ leak รายละเอียด internal (เช่น เวอร์ชัน framework), ปรับรูปแบบ error message ให้เป็นมาตรฐานเดียวกันทุก service |
| logging (เช่น `http-logger`, `kafka-logger`) | ส่ง log ของทุก request/response ออกไปยัง log pipeline ภายนอก (เชื่อมกับ [Volume 14 — Observability](../14-observability/README.md) ที่ครอบคลุมการเก็บ/ค้นหา log ด้วย Fluent Bit + OpenSearch) |

## 2.5 ทำไมต้องรวม Cross-Cutting Concern ไว้ที่ Gateway

Auth, rate limit, CORS, logging — สังเกตว่าไม่มีตัวไหนเลยที่เป็น "business logic" ของ service ใดตัวหนึ่งโดยเฉพาะ ทั้งหมดเป็น **cross-cutting concern**: ความต้องการที่ตัดขวางทุก service เหมือนกันหมด ถ้าไม่รวมศูนย์ไว้ที่ gateway แต่ละทีมต้อง**implement เรื่องเดียวกันซ้ำ**ในทุก service ของตัวเอง — เสี่ยงที่แต่ละทีมจะทำไม่เหมือนกัน (เช่น validate JWT คนละวิธี, บาง service ลืมเช็ค rate limit ไปเลย) และเพิ่ม cognitive load ให้ทุกทีมต้องรู้เรื่อง auth/security แทนที่จะโฟกัสที่ domain logic ของตัวเอง

แนวคิดนี้ตรงกับ**Middleware** ใน [Volume 4 — Golang บทที่ Repository, DI, Middleware](../04-golang/README.md) พอดี — HTTP middleware ในแอปเดียวคือการห่อ handler ด้วย logic ที่ใช้ร่วมกัน (logging, auth, recovery) ก่อนถึง business logic จริง เพียงแค่ API Gateway ยกแนวคิดเดียวกันนี้ขึ้นไปอีกระดับ (จาก "ทุก handler ในแอปเดียว" เป็น "ทุก service ในทั้งระบบ") — Gateway จึงเปรียบเสมือน **middleware chain ระดับ infrastructure** ที่ครอบคลุมทั้ง microservices architecture แทนที่จะเป็น middleware ในแอปเดียว

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| Consumer | ตัวตนของผู้เรียก API (แอป/ทีม/partner) แยกอิสระจาก Route ที่บอกแค่ปลายทาง |
| ผูก Auth Plugin | key-auth/jwt-auth ระบุว่า request มาจาก consumer ไหน แล้ว plugin อื่นเอาไปใช้ต่อได้ |
| Plugin Architecture | pluggable, มี priority กำหนดลำดับ, แต่ละตัว short-circuit ได้ |
| Plugin ที่ใช้บ่อย | cors, request/response-transformer, logging (เชื่อม Volume 14) |
| ทำไมรวมที่ Gateway | cross-cutting concern ไม่ควรถูก implement ซ้ำในทุก service — เหมือน middleware แต่ยกระดับเป็น infrastructure |

## คำถามทบทวน

1. Consumer ต่างจาก Route อย่างไร ตอบคำถามคนละมิติกันอย่างไร?
2. ทำไม auth plugin (เช่น key-auth) ต้องรันก่อน plugin อื่นๆ ในลำดับ execution chain เสมอ?
3. ยกตัวอย่าง plugin ที่ "ตัดจบ" (short-circuit) request ได้ทันทีโดยไม่ปล่อยไปถึง upstream พร้อมอธิบายว่าทำไมถึงต้องทำแบบนั้น
4. อธิบายว่าทำไมการรวม cross-cutting concern ไว้ที่ gateway ถึงลด cognitive load ให้ทีมพัฒนา service แต่ละทีม
5. เปรียบเทียบ API Gateway plugin chain กับ HTTP middleware ใน Go ว่าคล้ายกันตรงไหน ต่างกันตรงไหน (ในแง่ขอบเขตที่ครอบคลุม)

---

ก่อนหน้า: [บทที่ 1 — Route และ Upstream](01-routing.md) | ถัดไป: [บทที่ 3 — JWT และ Rate Limit](03-auth-and-rate-limit.md)
