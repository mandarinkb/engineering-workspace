# บทที่ 1: Cache และ TTL

Redis ถูกใช้งานบ่อยที่สุดในฐานะ cache layer อยู่หน้าหรือข้างฐานข้อมูลหลัก เพื่อลด latency และลดโหลดที่วิ่งไปหา database (เช่น [Volume 5 — PostgreSQL](../05-postgresql/README.md)) แต่การใช้ cache ให้ถูกต้องซับซ้อนกว่าแค่ "เก็บค่าไว้ก่อน" — ต้องเข้าใจ pattern การอ่าน/เขียนที่เหมาะกับสถานการณ์ ปัญหา cache invalidation, cache stampede และการจัดการ memory เมื่อ cache เต็ม บทนี้ครอบคลุมทั้งหมดนั้น

## 1.1 Cache Pattern

### Cache-Aside (Lazy Loading)

Pattern ที่พบบ่อยที่สุด — แอปพลิเคชันเป็นผู้ควบคุม cache เอง ไม่ใช่ database หรือ cache layer ทำให้อัตโนมัติ:

```
อ่าน:
  1. App เช็ค cache ก่อน
  2. Cache hit  → คืนค่าจาก cache ทันที
  3. Cache miss → App query จาก database → เขียนผลลัพธ์ลง cache → คืนค่าให้ client

  App ──1. GET──► Cache
   │              (miss)
   └──3. Query──► Database
   │
   └──4. SET────► Cache (เก็บไว้ใช้ครั้งถัดไป)

เขียน:
  1. App เขียนลง database โดยตรง
  2. App ลบ (invalidate) key ที่เกี่ยวข้องออกจาก cache — ไม่ใช่อัปเดต cache ทันที
```

```sql
-- Pseudocode
value = cache.get(key)
if value is None:
    value = db.query(...)
    cache.set(key, value, ttl=300)
return value
```

ข้อดี: เขียนโค้ดง่าย, cache เก็บเฉพาะข้อมูลที่**ถูกอ่านจริง**เท่านั้น (ไม่เปลืองพื้นที่กับข้อมูลที่ไม่มีใครขอ) ข้อเสีย: request แรกหลัง cache miss ต้องรอ database เต็มๆ (latency สูงกว่า pattern อื่น) และมีช่วงเวลาสั้นๆ ที่ cache กับ database ไม่ตรงกันได้เสมอ (ระหว่างเขียน database เสร็จ กับก่อนที่จะลบ cache สำเร็จ)

### Write-Through

แอปพลิเคชันเขียนผ่าน cache เป็นตัวกลางเสมอ cache รับผิดชอบเขียนต่อไปยัง database เอง (synchronous):

```
เขียน:
  App ──write──► Cache ──write (sync)──► Database
                   │
                   └── ตอบ App ว่าสำเร็จ ก็ต่อเมื่อเขียนทั้งสองที่เสร็จแล้ว
```

ข้อดี: cache กับ database สอดคล้องกันตลอดเวลา (ไม่มีช่วง stale) ข้อเสีย: เขียนช้าลงเพราะต้องรอทั้งสองระบบเขียนสำเร็จ และข้อมูลที่ไม่เคยถูกอ่านเลยก็ยังถูกเก็บใน cache อยู่ดี (เปลือง memory กับข้อมูลที่อาจไม่มีใครใช้)

### Write-Behind (Write-Back)

แอปพลิเคชันเขียนลง cache ก่อน แล้ว cache เขียนต่อไปยัง database แบบ **asynchronous** (ทีหลัง เป็น batch หรือ queue):

```
เขียน:
  App ──write──► Cache ──ตอบ App ทันที (เร็ว)
                   │
                   └──(async, ทีหลัง)──► Database
```

ข้อดี: เขียนเร็วมาก (client ไม่ต้องรอ database เลย) เหมาะกับงานที่เขียนถี่มาก (เช่น analytics counter, view count) ข้อเสีย: มีความเสี่ยง data loss ถ้า cache ล่มก่อนจะ flush ไปยัง database สำเร็จ ต้องยอมรับ trade-off นี้เฉพาะข้อมูลที่ไม่ critical มาก

```
สรุปเทียบ 3 pattern:

Cache-aside:   App ควบคุมเอง        | เขียนง่าย    | อาจ stale ชั่วขณะ
Write-through: Cache เขียนต่อ sync   | consistency ดี | เขียนช้ากว่า
Write-behind:  Cache เขียนต่อ async  | เขียนเร็วสุด  | เสี่ยง data loss
```

## 1.2 Cache Invalidation Strategy

มีคำกล่าวติดตลกในวงการ computer science: *"There are only two hard things in Computer Science: cache invalidation and naming things."* — ประโยคนี้ไม่ได้พูดเกินจริงมากนัก เพราะการตัดสินใจว่า **"เมื่อไหร่ข้อมูลใน cache ถือว่าเก่าเกินไปจนต้องทิ้ง"** ซับซ้อนกว่าที่คิด

กลยุทธ์หลักที่ใช้จริง:

- **TTL-based** — ตั้งเวลาหมดอายุให้ทุก key ยอมรับความ stale ได้ในช่วงเวลาสั้นๆ ง่ายที่สุดและใช้บ่อยที่สุด (รายละเอียดในหัวข้อ 1.4)
- **Explicit invalidation** — เมื่อข้อมูลต้นทางเปลี่ยน (เช่น `UPDATE` ใน database) แอปพลิเคชันสั่งลบ key ที่เกี่ยวข้องออกจาก cache ทันที (`DEL key`) แม่นยำกว่า TTL แต่ต้องเขียนโค้ดให้ครบทุกจุดที่แก้ไขข้อมูล ถ้าลืมจุดใดจุดหนึ่ง cache จะ stale โดยไม่มีใครรู้
- **Event-based invalidation** — ใช้ message queue/event stream (เชื่อมกับ [Volume 12 — Kafka](../12-kafka/README.md) หรือ Redis Pub/Sub ใน [บทที่ 2](02-pubsub-and-streams.md)) ประกาศว่าข้อมูลไหนเปลี่ยน แล้วให้ทุก service ที่ cache ข้อมูลนั้นอยู่ invalidate เอง เหมาะกับระบบที่มีหลาย service cache ข้อมูลชุดเดียวกันไว้คนละที่

ในทางปฏิบัติ ระบบส่วนใหญ่ใช้ TTL เป็น safety net เสมอ (แม้จะมี explicit invalidation ด้วยก็ตาม) เพราะถ้า explicit invalidation พลาดไปด้วยเหตุผลใดก็ตาม (bug, service ล่มก่อนส่ง event) TTL จะทำให้ข้อมูล stale หายไปเองในที่สุด ไม่ stale ค้างตลอดไป

## 1.3 Cache Stampede / Thundering Herd

### สถานการณ์ที่เกิดปัญหา

สมมติ key ยอดนิยม (เช่น หน้า product ที่คนดูเยอะที่สุด) มี TTL หมดอายุพร้อมกันในวินาทีเดียว และขณะนั้นมี request เข้ามาพร้อมกันหลายพันตัวต่อวินาที:

```
เวลา T: key หมดอายุ

Request 1  ──miss──► Query DB ──┐
Request 2  ──miss──► Query DB ──┤
Request 3  ──miss──► Query DB ──┼──► DB โดนถล่มพร้อมกันหลายพัน query
...                              │    ทั้งที่ทุก request ต้องการผลลัพธ์เดียวกันเป๊ะๆ
Request N  ──miss──► Query DB ──┘
```

ทุก request ที่มาถึงพร้อมกันหลัง key หมดอายุ ต่างคิดว่าตัวเอง "เจอ cache miss" จึงยิง query ไปหา database พร้อมกันทั้งหมด ทั้งที่จริงๆ ต้องการผลลัพธ์เดียวกัน — ถ้า database รับโหลดขนาดนี้ไม่ไหว อาจล่มทั้งระบบ (cascading failure) จากปัญหาที่ต้นเหตุคือแค่ cache key เดียวหมดอายุ

### วิธีป้องกัน

**1. Locking (Mutex/Distributed Lock)** — ให้ request แรกที่เจอ cache miss เป็นคนเดียวที่ไป query database จริง ส่วน request อื่นที่มาพร้อมกันต้องรอ (หรือคืนค่าเก่าไปพลางๆ):

```
Request 1  ──miss──► SETNX lock:key ──ได้ lock──► Query DB ──► SET cache ──► ปล่อย lock
Request 2  ──miss──► SETNX lock:key ──ไม่ได้ lock──► รอสักครู่แล้ว retry อ่าน cache
Request 3  ──miss──► SETNX lock:key ──ไม่ได้ lock──► รอสักครู่แล้ว retry อ่าน cache
```

ใช้ `SET lock:key value NX EX 10` (atomic — ตั้งค่าได้ก็ต่อเมื่อ key ยังไม่มีอยู่ พร้อม TTL กันเหตุการณ์ lock ค้างตลอดไปถ้า process ที่ถือ lock ตายกลางทาง)

**2. Jitter (สุ่มกระจาย TTL)** — แทนที่จะตั้ง TTL ทุก key เท่ากันเป๊ะ (เช่น 300 วินาทีทุกตัว) ให้บวกค่าสุ่มเล็กน้อยเข้าไป (เช่น `300 + random(0, 30)` วินาที) ทำให้ key จำนวนมากไม่หมดอายุพร้อมกันเป๊ะในวินาทีเดียว กระจายโหลดที่จะกระทบ database ออกไปตามเวลา

**3. Probabilistic Early Expiration** — แต่ละ request ที่อ่าน cache (แม้จะยังไม่หมดอายุจริง) คำนวณความน่าจะเป็นเล็กน้อยที่จะ "รีเฟรช cache ล่วงหน้า" ก่อนที่ TTL จริงจะหมด ยิ่งใกล้เวลาหมดอายุ ความน่าจะเป็นยิ่งสูงขึ้น ทำให้มีบาง request (ไม่ใช่ทั้งหมด) ทยอยไป refresh cache ก่อนที่ key จะหมดอายุจริงและเกิด stampede เต็มรูปแบบ — เป็นแนวทางที่ซับซ้อนกว่าแต่กระจายโหลดได้ราบรื่นที่สุดสำหรับ key ที่ traffic สูงมากๆ

## 1.4 TTL Mechanics

```
SET session:42 "user_data" EX 3600     -- ตั้งค่าพร้อม TTL 3600 วินาทีในคำสั่งเดียว
EXPIRE session:42 3600                 -- ตั้ง TTL ให้ key ที่มีอยู่แล้ว (วินาที)
TTL session:42                          -- คืนเวลาที่เหลือ (วินาที); -1 = ไม่มี TTL (อยู่ถาวร), -2 = key ไม่มีอยู่
PERSIST session:42                      -- ยกเลิก TTL ทำให้ key อยู่ถาวรอีกครั้ง
```

ข้อควรระวังที่พบบ่อย: คำสั่งเขียนทับค่าตรงๆ อย่าง `SET` (ไม่ระบุ `EX`/`KEEPTTL`) จะ**ล้าง TTL เดิมทิ้ง** ทำให้ key ที่เคยมีวันหมดอายุกลายเป็นอยู่ถาวรโดยไม่ตั้งใจ ถ้าต้องการเขียนทับค่าแต่คง TTL เดิมไว้ ต้องใช้ `SET key value KEEPTTL` อย่างชัดเจน

## 1.5 Eviction Policy เมื่อ Memory เต็ม

Redis เป็น in-memory store — เมื่อ memory เต็มตามค่า `maxmemory` ที่ตั้งไว้ ต้องตัดสินใจว่าจะทำอย่างไรกับ request เขียนใหม่ กำหนดผ่าน `maxmemory-policy`:

| Policy | พฤติกรรม |
|---|---|
| `noeviction` | ปฏิเสธคำสั่งเขียนใหม่ทั้งหมด (ตอบ error) ไม่ไล่ key ไหนออกเลย — ใช้เมื่อ Redis เก็บข้อมูลสำคัญที่ห้ามหายเด็ดขาด |
| `allkeys-lru` | ไล่ key ที่**ไม่ถูกใช้นานที่สุด**ออก (Least Recently Used) จาก**ทุก key ในระบบ** ไม่ว่าจะมี TTL หรือไม่ |
| `allkeys-lfu` | ไล่ key ที่**ถูกใช้บ่อยน้อยที่สุด**ออก (Least Frequently Used) จากทุก key — ต่างจาก LRU ตรงที่ดูความถี่การใช้ ไม่ใช่แค่เวลาที่ใช้ล่าสุด |
| `allkeys-random` | สุ่มไล่ key ออกจากทุก key |
| `volatile-lru` | ไล่ key ที่ไม่ถูกใช้นานที่สุดออก **เฉพาะ key ที่มี TTL ตั้งไว้เท่านั้น** |
| `volatile-lfu` | เหมือน `volatile-lru` แต่ใช้เกณฑ์ความถี่แทน |
| `volatile-random` | สุ่มไล่ออก เฉพาะ key ที่มี TTL |
| `volatile-ttl` | ไล่ key ที่ TTL **ใกล้หมดอายุที่สุด**ออกก่อน เฉพาะ key ที่มี TTL |

### ความต่างระหว่าง `allkeys-*` กับ `volatile-*`

นี่คือจุดที่สับสนบ่อยที่สุด: **`allkeys-*` พิจารณา key ทุกตัวในระบบเป็นเป้าหมายที่ไล่ออกได้** ไม่ว่า key นั้นจะตั้ง TTL ไว้หรือไม่ก็ตาม ส่วน **`volatile-*` จะไล่ออกเฉพาะ key ที่มี TTL ตั้งไว้เท่านั้น** — ถ้า key ไม่มี TTL เลย (`TTL key` คืน `-1`) จะไม่มีวันถูก evict ด้วย policy กลุ่ม `volatile-*` แม้ memory จะเต็มแค่ไหนก็ตาม

ผลกระทบเชิงปฏิบัติ: ถ้าเลือกใช้ `volatile-lru` แต่แอปพลิเคชันมี key จำนวนมากที่ลืมตั้ง TTL ไว้ (bug หรือความตั้งใจที่ผิดพลาด) key เหล่านั้นจะสะสมโดยไม่มีวันถูกไล่ออก จนสุดท้าย memory เต็มและ Redis ปฏิเสธเขียนข้อมูลใหม่ทั้งหมด (พฤติกรรมกลายเป็นเหมือน `noeviction` สำหรับ key ที่ไม่มี TTL) — ถ้าตั้งใจให้ Redis ทำหน้าที่เป็น **pure cache** (ทุกอย่างควรมีวันหมดอายุ ไม่มีอะไรอยู่ถาวร) `allkeys-lru`/`allkeys-lfu` มักเหมาะสมกว่า แต่ถ้า Redis ถูกใช้ผสมกันทั้งเป็น cache และเก็บข้อมูลถาวรบางส่วน (เช่น session ที่ต้องอยู่ถาวรจนกว่าจะ logout) `volatile-*` จะปลอดภัยกว่าเพราะไม่มีวันไปแตะ key ที่ตั้งใจให้อยู่ถาวร

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| Cache-aside | App ควบคุมเอง เขียนง่ายที่สุด แต่มีช่วง stale ได้ |
| Write-through | เขียนผ่าน cache แบบ sync ไป database — consistency ดีแต่ latency สูงขึ้น |
| Write-behind | เขียนลง cache ก่อนแล้ว flush async — เร็วสุดแต่เสี่ยง data loss |
| Cache invalidation | TTL-based, explicit, event-based — ระบบจริงมักใช้ TTL เป็น safety net เสมอ |
| Cache stampede | key ยอดนิยมหมดอายุพร้อมกัน request จำนวนมากถล่ม database พร้อมกัน แก้ด้วย lock/jitter/probabilistic early expiration |
| TTL | `SET` เขียนทับล้าง TTL เดิม ต้องใช้ `KEEPTTL` ถ้าไม่อยากให้หาย |
| Eviction policy | `allkeys-*` พิจารณาทุก key, `volatile-*` พิจารณาเฉพาะ key ที่มี TTL เท่านั้น |

## คำถามทบทวน

1. เปรียบเทียบ cache-aside, write-through, write-behind — pattern ไหนเหมาะกับข้อมูลที่เขียนถี่มากแต่ยอมรับการสูญเสียข้อมูลเล็กน้อยได้ เพราะเหตุใด?
2. อธิบาย cache stampede ด้วยสถานการณ์ของตัวเอง (ไม่ใช้ตัวอย่างในบทเรียน) แล้วบอกวิธีป้องกันอย่างน้อย 2 วิธี
3. ทำไมการใช้ `SET key value` เขียนทับข้อมูลเดิมถึงอาจทำให้ TTL ที่ตั้งไว้ก่อนหน้าหายไปโดยไม่ตั้งใจ แก้ปัญหานี้อย่างไร?
4. อธิบายความต่างระหว่าง `allkeys-lru` กับ `volatile-lru` และยกตัวอย่างสถานการณ์ที่เลือกใช้ผิดตัวจะเกิดปัญหาอะไร
5. เพราะเหตุใดระบบจริงมักใช้ TTL เป็น "safety net" แม้จะมี explicit invalidation logic อยู่แล้วก็ตาม?

---

ถัดไป: [บทที่ 2 — Pub/Sub และ Streams](02-pubsub-and-streams.md)
