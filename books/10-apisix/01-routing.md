# บทที่ 1: Route และ Upstream

## 1.1 Route คืออะไร

**Route** คือหน่วยพื้นฐานที่สุดของ APISIX — กฎที่บอกว่า "request ที่มีลักษณะแบบนี้ (path, method, host) ให้ส่งต่อไปที่ไหน" APISIX เก็บ configuration ทั้งหมดเป็น object ผ่าน **Admin API** (REST API ที่จัดการตัว gateway เอง) แทนที่จะแก้ config file แล้ว reload เหมือน Nginx ดั้งเดิม — ทำให้เปลี่ยน routing rule ได้แบบ dynamic โดยไม่ downtime

```json
PUT /apisix/admin/routes/1
{
  "uri": "/api/users/*",
  "methods": ["GET", "POST"],
  "host": "api.example.com",
  "upstream": {
    "type": "roundrobin",
    "nodes": {
      "10.0.1.11:8080": 1,
      "10.0.1.12:8080": 1
    }
  }
}
```

Route นี้บอกว่า: request ที่เข้ามาที่ host `api.example.com` ด้วย method `GET`/`POST` และ path ขึ้นต้นด้วย `/api/users/` ให้ส่งไปยัง upstream ที่มี 2 node (รายละเอียด upstream ดูหัวข้อ 1.4)

## 1.2 Path Matching Modes

APISIX รองรับการจับคู่ path หลายรูปแบบ เรียงจากเจาะจงน้อยไปมาก:

| Mode | ตัวอย่าง `uri` | จับคู่กับ | Use Case |
|---|---|---|---|
| **Exact** | `/api/health` | ตรงเป๊ะเท่านั้น | endpoint เฉพาะเจาะจง เช่น health check |
| **Prefix** | `/api/users/*` | ทุก path ที่ขึ้นต้นด้วย `/api/users/` | route ทั้ง resource เดียวไปที่ service เดียวกัน |
| **Regex** | `~/api/users/\d+$` | ตาม regular expression (ต้องขึ้นต้นด้วย `~`) | เงื่อนไขซับซ้อน เช่น จับเฉพาะ path ที่ลงท้ายด้วยตัวเลข (user ID) |

Prefix matching (`*`) เป็นโหมดที่ใช้บ่อยที่สุดในทางปฏิบัติ เพราะ microservice หนึ่งตัวมักรับผิดชอบทั้ง path prefix เดียว (เช่น service คำสั่งซื้อรับผิดชอบทุกอย่างใต้ `/api/orders/*`) ส่วน regex ควรใช้เท่าที่จำเป็นเพราะ evaluate ช้ากว่าและอ่านยากกว่า

## 1.3 Priority Resolution เมื่อหลาย Route ตรงกัน

ถ้ามีหลาย route ที่ match request เดียวกันได้พร้อมกัน (เช่น route หนึ่งจับ `/api/users/*` แบบ prefix และอีก route จับ `/api/users/vip` แบบ exact) APISIX ต้องมีกฎตัดสินว่าจะใช้ route ไหน:

1. **`priority` ที่ตั้งไว้ชัดเจน** — ตัวเลขสูงกว่าชนะก่อนเสมอ ไม่ว่ารูปแบบ matching จะเป็นอะไร
2. ถ้า `priority` เท่ากัน — **matching ที่เจาะจงกว่าชนะ**: exact ชนะ prefix, prefix ที่ยาวกว่า (ยาวกว่า = เจาะจงกว่า) ชนะ prefix ที่สั้นกว่า

แนวปฏิบัติที่ดี: อย่าพึ่งพา default priority resolution เพียงอย่างเดียวในระบบที่มี route จำนวนมากและซับซ้อน ให้กำหนด `priority` อย่างชัดเจนเสมอสำหรับ route ที่ตั้งใจให้ override route อื่น (เช่น route เฉพาะสำหรับ canary/feature flag ที่ต้องชนะ route ปกติ) เพื่อไม่ให้พฤติกรรมขึ้นกับกฎ implicit ที่คนอ่าน config ทีหลังอาจไม่รู้

## 1.4 Upstream: กลุ่มปลายทางจริง

ถ้า Route คือ "กฎว่าจะ route ไปไหน" **Upstream** คือ "กลุ่มของ backend instance จริง" ที่ route หนึ่งตัวชี้ไปหา — เปรียบเหมือน object ที่แยกอิสระออกมา ทำให้หลาย route ใช้ upstream เดียวกันได้ (หรือกำหนด upstream ตรงในตัว route เลยก็ได้แบบตัวอย่างหัวข้อ 1.1)

```json
PUT /apisix/admin/upstreams/1
{
  "type": "roundrobin",
  "nodes": {
    "10.0.1.11:8080": 1,
    "10.0.1.12:8080": 1,
    "10.0.1.13:8080": 2
  }
}
```

ตัวเลขท้ายแต่ละ node (`1`, `1`, `2`) คือ **weight** — node `10.0.1.13` รับ traffic ได้มากกว่า node อื่นตามสัดส่วน (ใช้เมื่อ instance มี spec ไม่เท่ากัน)

## 1.5 Load Balancing Algorithm ที่ระดับ Upstream

`type` ใน upstream config กำหนด algorithm ที่ใช้กระจาย request ไปยัง node — APISIX รองรับ round-robin, weighted round-robin, consistent hashing (`chash`), least connections (`least_conn`) ฯลฯ ซึ่งเป็น algorithm มาตรฐานเดียวกับที่อธิบายไว้ลึกแล้วใน [Volume 2 — Reverse Proxy และ Load Balancer](../02-network/05-reverse-proxy-load-balancer.md) (round robin, least connections, consistent hashing) — บทนี้จะไม่อธิบายซ้ำว่าแต่ละ algorithm ทำงานอย่างไร แต่เน้นว่า **APISIX เป็นจุดที่เอา algorithm เหล่านั้นไป apply จริงในรูปแบบ config**

ตัวอย่างการเลือกใช้ `chash` (consistent hashing) เพื่อรักษา cache locality ตาม user:

```json
{
  "type": "chash",
  "hash_on": "header",
  "key": "X-User-Id",
  "nodes": { "10.0.1.11:8080": 1, "10.0.1.12:8080": 1 }
}
```

request จาก user คนเดียวกัน (header `X-User-Id` เหมือนกัน) จะถูก route ไปยัง backend node ตัวเดิมเสมอ — มีประโยชน์มากถ้า backend มี local cache ที่ไม่ได้แชร์ผ่าน Redis

## 1.6 Health Check: Active vs Passive

Upstream ต้องรู้ว่า node ไหน "พร้อมรับ traffic จริง" — แนวคิด active/passive health check เหมือนกับที่อธิบายไว้ใน [Volume 2 บทที่ 5](../02-network/05-reverse-proxy-load-balancer.md) แต่ APISIX ให้ config ทั้งสองแบบพร้อมกันได้ในตัวเดียว:

```json
{
  "checks": {
    "active": {
      "type": "http",
      "http_path": "/healthz",
      "healthy": { "interval": 5, "successes": 2 },
      "unhealthy": { "interval": 3, "http_failures": 3 }
    },
    "passive": {
      "healthy": { "successes": 3 },
      "unhealthy": { "http_failures": 5, "timeouts": 3 }
    }
  }
}
```

- **Active** — APISIX ยิง `GET /healthz` ไปเช็คทุก node เองเป็นระยะ (ทุก 5 วินาทีตามตัวอย่าง) ถ้า fail ติดกัน 3 ครั้งใน interval 3 วินาที → ตัดออกจาก pool ทันที ไม่ต้องรอ traffic จริง
- **Passive** — สังเกตจาก traffic จริงที่ไหลผ่าน ถ้า node ตอบ error/timeout ถี่เกิน threshold (5 ครั้ง หรือ timeout 3 ครั้ง) → ตัดออกจาก pool ชั่วคราวเช่นกัน

ข้อดีของการใช้ทั้งคู่ร่วมกัน: active health check จับปัญหาได้เร็วโดยไม่ต้องรอ user จริงมาเจอ error ก่อน ส่วน passive health check เป็นตาข่ายรองรับกรณีที่ active check endpoint (`/healthz`) เองตอบ OK แต่ endpoint จริงที่ user เรียกกลับมีปัญหา (เช่น connection pool ไป database เต็มเฉพาะบาง endpoint)

## 1.7 Retry Behavior

เมื่อ request ไปถึง node หนึ่งแล้วล้มเหลว (connection refused, timeout) APISIX จะ **retry ไปยัง node อื่นใน upstream เดียวกันโดยอัตโนมัติ** ก่อนที่จะตอบ error กลับไปยัง client (จำนวนครั้งกำหนดได้ผ่าน `retries` ใน upstream config) กลไกนี้ทำงานร่วมกับ passive health check: node ที่ fail ซ้ำๆ จะถูกตัดออกจาก pool ทำให้ retry ครั้งถัดๆ ไปไม่วนไปโดน node ที่เสียอยู่ซ้ำอีก

ข้อควรระวัง: retry ที่ตั้งค่ามากเกินไปกับ request ที่ไม่ idempotent (เช่น POST สร้างคำสั่งซื้อ) เสี่ยงทำให้เกิดผลข้างเคียงซ้ำ (เช่น สร้างคำสั่งซื้อซ้ำ) ถ้า backend ประมวลผลสำเร็จแต่ response หายไปก่อนถึง gateway — การออกแบบ retry ที่ปลอดภัยต้องคำนึงถึง idempotency ของ endpoint ปลายทางเสมอ ไม่ใช่แค่ตั้งค่าตัวเลขให้สูงไว้ก่อน

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| Route | จับคู่ path/method/host กับปลายทาง, config ผ่าน Admin API แบบ dynamic ไม่ต้อง reload |
| Path Matching | exact เจาะจงสุด, prefix ใช้บ่อยสุด, regex ยืดหยุ่นแต่แพงกว่า |
| Priority | ตั้งชัดเจนด้วย `priority` ก่อนเสมอ ถ้าเท่ากันค่อยดูความเจาะจงของ pattern |
| Upstream | กลุ่ม backend node จริง แยก object จาก Route ใช้ร่วมกันได้หลาย route |
| Load Balancing | algorithm เดียวกับ Volume 2 (round robin, chash, least_conn) นำมา apply เป็น config |
| Health Check | active เช็คเชิงรุกจับปัญหาก่อน user เจอ, passive สังเกตจาก traffic จริงเป็นตาข่ายรอง |
| Retry | ระวังกับ non-idempotent endpoint เพราะ retry ซ้ำอาจสร้างผลข้างเคียงซ้ำ |

## คำถามทบทวน

1. ทำไม APISIX ถึงใช้ Admin API แทน config file + reload เหมือน Nginx ดั้งเดิม ข้อดีคืออะไร?
2. ถ้ามี route แบบ prefix `/api/*` และ route แบบ exact `/api/health` ตั้ง priority เท่ากัน ตัวไหนจะถูกเลือกเมื่อ request มาที่ `/api/health` เพราะเหตุใด?
3. ทำไมควรกำหนด `priority` อย่างชัดเจนแทนที่จะพึ่งพา default resolution เมื่อระบบมี route จำนวนมาก?
4. Active กับ Passive health check ต่างกันอย่างไร ทำไมระบบ production ถึงมักใช้ทั้งสองแบบร่วมกัน?
5. ทำไมการตั้งค่า retry สูงๆ ถึงเสี่ยงกับ endpoint ที่ไม่ idempotent เช่น POST สร้างคำสั่งซื้อ?

---

ถัดไป: [บทที่ 2 — Consumer และ Plugins](02-consumers-and-plugins.md)
