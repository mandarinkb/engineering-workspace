# บทที่ 3: JWT และ Rate Limit ที่ APISIX

## 3.1 jwt-auth Plugin: Validate Token ที่ Gateway

โครงสร้างของ JWT เอง (header.payload.signature, claim มาตรฐาน, HMAC vs RSA/ECDSA) ถูกอธิบายไว้ละเอียดแล้วใน [Volume 9 — บทที่ 1](../09-authentication/01-session-and-jwt.md) บทนี้จะไม่ทวนซ้ำเรื่องโครงสร้าง JWT แต่โฟกัสที่คำถามสถาปัตยกรรม: **ควร validate JWT ตรงไหนในระบบ**

Plugin `jwt-auth` ของ APISIX ทำหน้าที่ตรวจสอบ JWT **ที่ gateway** ก่อนที่ request จะถูกส่งต่อไปยัง service ปลายทางเลย:

```json
PUT /apisix/admin/routes/1
{
  "uri": "/api/orders/*",
  "plugins": {
    "jwt-auth": {}
  },
  "upstream": { "...": "..." }
}
```

เมื่อ consumer ถูกผูกกับ `jwt-auth` (ดู [บทที่ 2](02-consumers-and-plugins.md)) APISIX จะตรวจ signature, `exp`, `iss` ของ token ที่แนบมาก่อนตัดสินใจว่าจะปล่อย request ผ่านไปหรือตอบ `401` กลับไปเลย — service ที่อยู่หลัง gateway (`10.0.1.11:8080` เป็นต้น) **ไม่มีทางเห็น request ที่ token ไม่ถูกต้องเลย**

## 3.2 ทำไม Validate ครั้งเดียวที่ Edge ดีกว่าทุก Service Validate ซ้ำ

ลองเทียบสองสถาปัตยกรรม:

```
แบบ A: ทุก service validate JWT เอง
Client ──► Gateway (ส่งผ่านเฉยๆ) ──┬──► Service A (validate JWT เอง)
                                    ├──► Service B (validate JWT เอง)
                                    └──► Service C (validate JWT เอง)

แบบ B: Gateway validate ครั้งเดียวที่ edge
Client ──► Gateway (jwt-auth validate) ──┬──► Service A (เชื่อได้เลยว่า request ผ่านการ auth แล้ว)
                                          ├──► Service B (เชื่อได้เลย)
                                          └──► Service C (เชื่อได้เลย)
```

แบบ A มีปัญหาซ้ำกับที่กล่าวไว้ใน [บทที่ 2 หัวข้อ 2.5](02-consumers-and-plugins.md#25-ทำไมต้องรวม-cross-cutting-concern-ไว้ที่-gateway) — auth logic ถูก implement ซ้ำในทุก service เสี่ยงไม่สอดคล้องกัน (เช่น service หนึ่งลืมเช็ค `exp`, อีก service ตรวจ `aud` ผิด) นอกจากนี้ทุก service ยังต้องดึง public key/JWKS มาเก็บเองและจัดการ rotation เอง เพิ่มความซับซ้อนซ้ำซ้อน

แบบ B รวมจุด validate ไว้ที่เดียว — เมื่อ request ผ่าน gateway มาถึง service ปลายทางแล้ว **service เชื่อได้ทันทีว่า token ถูกต้อง** (มักส่งผ่าน claim ที่ verify แล้วต่อไปในรูป header เช่น `X-Consumer-Username` แทนที่จะส่ง JWT ดิบไปให้ service เอง) ผลลัพธ์คือ:

- **ลด load ซ้ำซ้อน** — งาน cryptographic verify (ซึ่งไม่ฟรี โดยเฉพาะ RSA) ทำครั้งเดียวต่อ request แทนที่จะทำซ้ำทุก service ที่ request วิ่งผ่าน (ถ้า request หนึ่งเรียกต่อกันหลาย service internal)
- **จุดควบคุมเดียว** — เปลี่ยนนโยบาย auth (เช่น เพิ่มการตรวจ `aud`, เปลี่ยน key rotation) แก้ที่ gateway จุดเดียว ไม่ต้อง deploy ใหม่ทุก service
- **ลด attack surface** — service ภายในไม่ต้อง expose logic การ parse/verify JWT ที่อาจมีช่องโหว่ (เช่น algorithm confusion attack) ให้ต้องดูแลเองในหลายที่

## 3.3 Rate Limiting: 4 Algorithm คลาสสิก

หลังจากรู้ว่า "ใครเรียก" (auth) แล้ว คำถามถัดมาคือ "เรียกได้บ่อยแค่ไหน" — มี algorithm หลักที่ใช้กันในทางปฏิบัติ 4 แบบ แต่ละแบบมี failure mode ที่ต้องเข้าใจก่อนเลือกใช้

### Fixed Window

แบ่งเวลาเป็นช่วงคงที่ (เช่น ทุก 1 นาที) นับจำนวน request ในแต่ละช่วง ถ้าเกิน limit ก็ปฏิเสธ พอขึ้นช่วงใหม่ตัวนับ reset กลับเป็น 0

**ปัญหาคลาสสิก — boundary burst**: สมมติ limit คือ 100 request/นาที ถ้า client ยิง 100 request ในวินาทีสุดท้ายของนาทีที่ 1 (เช่น วินาทีที่ 00:59) แล้วยิงอีก 100 request ทันทีในวินาทีแรกของนาทีที่ 2 (00:60 = 01:00) — ทั้งสองครั้งอยู่คนละ "window" จึงไม่ผิดกฎเลยแม้แต่ข้อเดียว แต่ในความเป็นจริง **backend โดนยิง 200 request ภายในเวลาแค่ 2 วินาที** ซึ่งขัดกับเจตนาที่ตั้งไว้ว่า "ไม่เกิน 100 ต่อนาที" อย่างสิ้นเชิง

```
Window 1 (00:00-00:59)              Window 2 (01:00-01:59)
                          │
        ...  [100 req ยิงรัวใน 00:59]│[100 req ยิงรัวใน 01:00]  ...
                          │
                    ← เพียง ~1-2 วินาที แต่ผ่าน limit ไป 200 req รวด →
```

### Sliding Window

แก้ปัญหา boundary burst ด้วยการนับ request ใน "หน้าต่างเวลาที่เลื่อนไปกับเวลาปัจจุบันเสมอ" (ไม่ใช่ช่วงคงที่ที่ตัดตรงเป๊ะ) เช่น ณ วินาทีนี้ นับ request ย้อนหลังไป 60 วินาทีที่ผ่านมาแบบต่อเนื่อง (มักคำนวณแบบประมาณด้วยการถ่วงน้ำหนักระหว่าง window ปัจจุบันกับ window ก่อนหน้า เพื่อไม่ต้องเก็บ timestamp ของทุก request จริงๆ) แก้ปัญหา burst ที่รอยต่อได้ แต่คำนวณซับซ้อนกว่า fixed window

### Leaky Bucket

เปรียบ request เหมือนน้ำที่ไหลเข้าถัง (bucket) ที่มีรูรั่วระบายออกด้วย**อัตราคงที่**เสมอ ไม่ว่าน้ำจะไหลเข้าเร็วแค่ไหน อัตราที่ไหลออกไปประมวลผลจริง (ไปหา backend) จะคงที่ตลอด — ถ้าถังเต็ม (น้ำไหลเข้าเร็วกว่าที่รูระบายออกได้) request ใหม่ที่เข้ามาจะถูกปฏิเสธ (หรือรอคิว) จุดเด่นคือ **traffic ที่ไปถึง backend จริงจะเรียบสม่ำเสมอเสมอ (smoothing)** ไม่มี burst หลุดออกไปเลย แม้ client จะยิงรัวแค่ไหนก็ตาม

### Token Bucket

คล้าย leaky bucket แต่กลับด้าน: มีถังเก็บ **token** ที่ถูกเติมเข้าไปด้วยอัตราคงที่ (เช่น 10 token/วินาที) ทุก request ที่จะผ่านต้อง "จ่าย" token 1 ใบ ถ้าไม่มี token เหลือ → ถูกปฏิเสธ จุดต่างสำคัญจาก leaky bucket คือถ้า traffic ว่างมาสักพัก token จะสะสมอยู่ในถังได้ (จนถึง capacity สูงสุด) ทำให้ **รองรับ burst ช่วงสั้นๆ ได้** (ใช้ token ที่สะสมไว้รัวเดียว) ก่อนจะกลับไปถูกจำกัดด้วยอัตราเติม token ปกติ — เหมาะกับ traffic ที่เป็นธรรมชาติ (ไม่สม่ำเสมอ) มากกว่า leaky bucket ที่บังคับ smoothing ตลอดเวลา

## 3.4 APISIX Rate Limit Plugins

APISIX แยก plugin ตามสิ่งที่จำกัดออกเป็น 3 ตัว ซึ่งเข้าใจผิดกันบ่อยว่าเป็นตัวเดียวกัน:

| Plugin | จำกัดอะไร | Algorithm ที่ใช้ |
|---|---|---|
| `limit-count` | จำนวน request ทั้งหมดในช่วงเวลาที่กำหนด (เช่น 1000 req/ชั่วโมง) | Fixed window (มี option เลือก sliding ได้) |
| `limit-req` | **อัตรา** การเข้ามาของ request (req/วินาที) โดยยอมให้ burst ได้ในระดับที่ตั้งไว้ | Leaky bucket |
| `limit-conn` | จำนวน **connection ที่เปิดค้างพร้อมกัน** ณ ขณะหนึ่ง (ไม่ใช่จำนวน request สะสม) | นับ concurrent connection ตรงๆ |

ตัวอย่าง `limit-count`:

```json
{
  "limit-count": {
    "count": 1000,
    "time_window": 3600,
    "key_type": "var",
    "key": "consumer_name",
    "rejected_code": 429
  }
}
```

`limit-conn` สำคัญมากสำหรับปกป้อง backend ที่มี resource จำกัดต่อ connection (เช่น connection pool ไป database) — ต่อให้ request rate ไม่สูงมาก แต่ถ้าแต่ละ request ใช้เวลานาน (เช่น upload ไฟล์ใหญ่) จำนวน connection ที่ค้างพร้อมกันอาจทำให้ backend ล้นได้เหมือนกัน ซึ่ง `limit-count`/`limit-req` (ที่นับตาม "จำนวน/อัตรา" ไม่ใช่ "ที่ค้างอยู่พร้อมกัน") ป้องกันไม่ได้

## 3.5 Rate Limit Scoped by Consumer vs by IP vs by Route

Rate limit plugin ทุกตัวใน APISIX ตั้งค่าได้ว่าจะ "นับแยกตามอะไร" ผ่าน `key_type`/`key`:

| Scope | ใช้เมื่อ | ตัวอย่าง key |
|---|---|---|
| **By Consumer** | ต้องการจำกัดตาม "สัญญา" ที่ตกลงกับ partner/แอปแต่ละราย (ตรงกับ Consumer ใน [บทที่ 2](02-consumers-and-plugins.md)) | `consumer_name` |
| **By IP** | ป้องกัน abuse จาก client ที่ไม่ได้ authenticate (เช่น endpoint สาธารณะ, endpoint login เอง) หรือป้องกัน DDoS เบื้องต้น | `remote_addr` |
| **By Route** | จำกัดโหลดรวมที่ไปถึง backend ตัวใดตัวหนึ่ง ไม่สนว่าใครเรียก — ปกป้อง backend เองจากการโดนถล่มรวม | ไม่ต้องระบุ key แยก (นับรวมทุก request ที่เข้า route นั้น) |

ระบบจริงมักผสมหลาย scope พร้อมกันในหลาย layer: **by IP** เป็นเกราะป้องกันชั้นนอกสุดสำหรับ endpoint ที่ยังไม่ auth (เช่น `/login`), **by Consumer** เป็นการบังคับใช้ตาม SLA/สัญญาที่ตกลงไว้กับผู้เรียกแต่ละราย และ **by Route** เป็นเพดานความปลอดภัยสุดท้ายที่ปกป้อง backend ไม่ให้ล้นไม่ว่า request จะมาจากใครก็ตาม

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| jwt-auth ที่ gateway | validate token ก่อนถึง service ปลายทางเลย service ไม่ต้อง verify ซ้ำ |
| ทำไม validate ที่ edge ดีกว่า | ลด compute ซ้ำซ้อน, จุดควบคุม policy เดียว, ลด attack surface ของ service ภายใน |
| Fixed Window | ง่ายสุดแต่มีปัญหา boundary burst — traffic กระจุกที่รอยต่อ window หลบ limit ได้ |
| Sliding Window | แก้ boundary burst ด้วยหน้าต่างที่เลื่อนตามเวลาจริง แลกกับความซับซ้อนที่เพิ่มขึ้น |
| Leaky Bucket | บังคับ smoothing อัตราคงที่เสมอ ไม่มี burst หลุดออกไปถึง backend |
| Token Bucket | สะสม token รองรับ burst สั้นๆ ได้ เหมาะกับ traffic ธรรมชาติที่ไม่สม่ำเสมอ |
| limit-count/limit-req/limit-conn | จำนวนรวม / อัตรา / connection ค้างพร้อมกัน — จำกัดคนละมิติกัน |
| Scope | by IP กันชั้นนอกสุด, by Consumer ตาม SLA, by Route ปกป้อง backend โดยรวม |

## คำถามทบทวน

1. อธิบายว่าทำไมการ validate JWT ที่ gateway ครั้งเดียวถึงดีกว่าให้ทุก service verify ซ้ำเอง ทั้งในแง่ performance และ security
2. อธิบายปัญหา "boundary burst" ของ fixed window ด้วยตัวเลขของตัวเอง (สมมติ limit ต่อนาที)
3. Leaky bucket กับ token bucket ต่างกันตรงไหน แบบไหนเหมาะกับ traffic ที่มาเป็น burst ธรรมชาติมากกว่า?
4. `limit-conn` ป้องกันปัญหาอะไรที่ `limit-count` และ `limit-req` ป้องกันไม่ได้?
5. ทำไมระบบจริงถึงมักใช้ rate limit หลาย scope (IP, Consumer, Route) พร้อมกันแทนที่จะเลือกแค่แบบเดียว?

---

ก่อนหน้า: [บทที่ 2 — Consumer และ Plugins](02-consumers-and-plugins.md) | กลับสู่ [สารบัญ Volume 10](README.md)
