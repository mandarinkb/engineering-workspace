# บทที่ 3: Cluster

Redis instance เดียวจำกัดด้วย memory และ CPU ของเครื่องเดียว เมื่อข้อมูลหรือ throughput เกินกว่าที่ instance เดียวรับไหว ต้องกระจายข้อมูลออกไปหลายเครื่อง บทนี้อธิบายว่า **Redis Cluster** ทำ sharding แบบ built-in ได้อย่างไรผ่านแนวคิด hash slot เปรียบเทียบกับ **Sentinel** ที่แก้ปัญหาคนละมิติ (high availability โดยไม่ sharding) และข้อจำกัดสำคัญที่สุดของการทำงานแบบ cluster คือ multi-key operation

## 3.1 Hash Slot Model

Redis Cluster แบ่งพื้นที่ key ทั้งหมดออกเป็น **16384 slots คงที่** (ไม่ใช่จำนวนที่ปรับเปลี่ยนได้ตามจำนวน node) แต่ละ key ถูก map ไปยัง slot ใดslot หนึ่งด้วยสูตร:

```
slot = CRC16(key) mod 16384
```

แต่ละ node ใน cluster รับผิดชอบช่วง slot หนึ่ง (หรือหลายช่วง) ตัวอย่างเช่น cluster ที่มี 3 node (ไม่นับ replica):

```
Node A: slot 0     - 5460
Node B: slot 5461  - 10922
Node C: slot 10923 - 16383

Client ──GET user:42──► คำนวณ CRC16("user:42") mod 16384 = slot 7000
                          │
                          ▼
                       อยู่ในช่วงของ Node B → ส่ง request ไปที่ Node B โดยตรง
```

ทำไมต้องเป็น 16384 (2^14) ไม่ใช่ตัวเลขอื่น — เป็นการเลือกที่สมดุลระหว่างสองปัจจัย: มากพอที่จะกระจาย slot ระหว่าง node ได้ละเอียดพอสมควร (รองรับ cluster ขนาดใหญ่หลักพัน node ในทางทฤษฎี) แต่เล็กพอที่ bitmap ที่ใช้เก็บสถานะ "slot ไหนอยู่ node ไหน" (ส่งกันระหว่าง node ผ่าน gossip protocol ตอน heartbeat) จะไม่ใหญ่เกินไปจนกิน bandwidth มากเกินจำเป็น

### Client-side Routing

Client ที่คุยกับ Redis Cluster ต้องรู้ mapping ระหว่าง slot กับ node (ผ่านคำสั่ง `CLUSTER SLOTS`/`CLUSTER SHARDS`) เพื่อส่ง request ไปยัง node ที่ถูกต้องได้เลยตั้งแต่ครั้งแรก ถ้า client เดารู้ node ผิด (เช่น cluster เพิ่ง reshard ไป) node ที่รับ request จะตอบกลับด้วย `MOVED <slot> <new-node>` บอกให้ client ไปติดต่อ node ใหม่แทน — client library ที่ดีจะ cache slot mapping ไว้และอัปเดตอัตโนมัติเมื่อเจอ `MOVED`

### Resharding

เมื่อเพิ่ม/ลด node ใน cluster ต้อง**ย้าย slot** ระหว่าง node ใหม่ (ไม่ใช่ย้ายข้อมูลทั้งหมดแบบสุ่ม — ย้ายเฉพาะ slot ที่กำหนดให้ย้าย) กระบวนการนี้ทำได้แบบ **online** (cluster ยังให้บริการต่อได้ระหว่าง reshard) โดย Redis ย้ายทีละ key ภายใน slot ที่กำลัง migrate สถานะของ slot ระหว่างการย้ายจะถูกทำเครื่องหมายเป็น `MIGRATING` (ที่ node ต้นทาง) และ `IMPORTING` (ที่ node ปลายทาง) — ถ้า client ขอ key ที่ยังไม่ทันย้าย node ต้นทางตอบได้ปกติ แต่ถ้าโดน migrate ไปแล้ว node ต้นทางจะตอบ `ASK` บอกให้ client ลองที่ node ปลายทางแทนชั่วคราว

## 3.2 High Availability: Sentinel vs Cluster

Redis มีสองกลไกที่แก้ปัญหาคนละเรื่องกัน แต่บ่อยครั้งถูกเข้าใจผิดว่าเป็นคู่แข่งกันโดยตรง:

### Redis Sentinel

ออกแบบมาสำหรับ **non-clustered deployment** (single primary + replica ธรรมดา ไม่มี sharding) — Sentinel เป็น process แยกต่างหากที่คอย monitor สุขภาพของ primary/replica เมื่อ primary ล่ม Sentinel จะทำ **automatic failover**: เลือก replica ที่เหมาะสมที่สุด (ข้อมูลล่าสุดที่สุด) มา promote เป็น primary ใหม่ แล้วแจ้ง client ให้รู้ตำแหน่ง primary ใหม่ผ่าน Sentinel เอง (client ต้องคุยกับ Sentinel ก่อนเพื่อถามว่า primary ปัจจุบันอยู่ที่ไหน ไม่ได้ต่อไปยัง Redis โดยตรงตั้งแต่แรก)

```
                  Sentinel 1
                     │
Client ──ถาม primary ตอนนี้อยู่ไหน──► Sentinel 2 ──monitor──► Primary ◄──replicate── Replica
                     │                                                                  ▲
                  Sentinel 3 ─────────────────────────────────────────monitor──────────┘
```

Sentinel ทำงานเป็นกลุ่ม (ปกติ 3 ตัวขึ้นไป จำนวนคี่) ใช้ **quorum** (เสียงข้างมาก) ตัดสินใจว่า primary ล่มจริงหรือแค่ network แยกชั่วคราว (ป้องกัน false positive ที่จะสั่ง failover โดยไม่จำเป็น) — Sentinel **ไม่ได้ sharding ข้อมูล** ข้อมูลทั้งหมดยังอยู่ในชุด primary-replica ชุดเดียว จำกัดด้วยขนาด memory ของเครื่องเดียวเหมือนเดิม เพียงแต่มี HA (พร้อมใช้งานสูง) เท่านั้น

### Redis Cluster

รวม**ทั้ง sharding และ HA** เข้าไว้ในตัวเดียว — แต่ละ hash slot range สามารถมี replica ของตัวเองได้ (ไม่ใช่แค่ node เดียว) เมื่อ node ที่รับผิดชอบ slot ใดล่ม cluster จะ promote replica ของ slot นั้นขึ้นมาเป็น primary ใหม่โดยอัตโนมัติ **โดยไม่ต้องพึ่ง Sentinel เลย** (มี failure detection + election ในตัว Cluster เองผ่าน gossip protocol)

```
Node A (slot 0-5460)     ◄──replicate──  Node A-replica
Node B (slot 5461-10922) ◄──replicate──  Node B-replica
Node C (slot 10923-16383)◄──replicate──  Node C-replica

ถ้า Node B ล่ม → Node B-replica ถูก promote เป็น primary ของ slot 5461-10922 อัตโนมัติ
ส่วน Node A, Node C ยังทำงานต่อได้ปกติ (ข้อมูลของ slot อื่นไม่กระทบ)
```

### สรุปเลือกใช้

| ต้องการ | เลือก |
|---|---|
| ข้อมูลพอดีกับ memory เครื่องเดียว แค่ต้องการ HA | Sentinel + primary/replica ธรรมดา |
| ข้อมูลใหญ่เกินเครื่องเดียว ต้อง sharding | Redis Cluster (มี HA ในตัวอยู่แล้ว) |
| ต้องการ sharding แต่ไม่ต้องการความซับซ้อนของ Cluster protocol | พิจารณา client-side sharding เอง หรือ proxy เช่น Redis Cluster Proxy/Twemproxy (แลกความง่ายกับความยืดหยุ่นในการ resharding) |

## 3.3 ข้อจำกัดของ Multi-key Operation ใน Cluster Mode

นี่คือข้อจำกัดที่สำคัญที่สุดที่ต้องรู้ก่อนออกแบบระบบบน Redis Cluster: คำสั่งที่ต้องใช้**หลาย key พร้อมกัน**ในคำสั่งเดียว (เช่น `MGET`, `MSET`, transaction ด้วย `MULTI/EXEC` ที่แตะหลาย key, หรือ Lua script ที่อ่าน/เขียนหลาย key) **ทำงานได้ก็ต่อเมื่อทุก key ที่เกี่ยวข้อง map ไปยัง slot เดียวกันเท่านั้น**

```
MGET user:1 user:2
-- ถ้า "user:1" และ "user:2" คำนวณ CRC16 แล้วตกคนละ slot
-- และ slot เหล่านั้นอยู่คนละ node → Redis Cluster ตอบ error ทันที
-- (CROSSSLOT Keys in request don't hash to the same slot)
```

เหตุผลเชิงสถาปัตยกรรม: การจะรองรับ multi-key operation ข้าม node ต้องมีการประสานงาน (coordination) ข้าม node แบบ distributed transaction ซึ่งเพิ่มความซับซ้อนและ latency มหาศาล Redis Cluster เลือกไม่รองรับกรณีนี้เลยตั้งแต่ต้น เพื่อรักษาความเร็วและความง่ายของโมเดล แลกกับการจำกัดว่า multi-key operation ต้องอยู่ slot เดียวกันเท่านั้น

### Hash Tags: ทางแก้เพื่อบังคับให้ Key กลุ่มหนึ่งอยู่ Slot เดียวกัน

Redis ให้ syntax พิเศษเรียกว่า **hash tag** — ถ้า key มีส่วนที่ครอบด้วยเครื่องหมายปีกกา `{...}` Redis จะคำนวณ hash slot จาก**เฉพาะข้อความในปีกกาเท่านั้น** ไม่ใช่ทั้ง key:

```
user:{42}:profile
user:{42}:orders
user:{42}:settings

-- ทั้งสาม key คำนวณ slot จาก "42" เท่านั้น (ข้อความในปีกกา)
-- จึง map ไปยัง slot เดียวกันเสมอ ไม่ว่าส่วนอื่นของ key จะต่างกันแค่ไหน

MGET user:{42}:profile user:{42}:orders  -- ทำงานได้ปกติ เพราะ slot เดียวกัน
```

```
key ปกติ:        slot = CRC16("user:42:profile") mod 16384
key มี hash tag:  slot = CRC16("42") mod 16384    ← ใช้แค่ข้อความในปีกกา {42}
```

การออกแบบ key schema ที่ดีสำหรับ Redis Cluster ตั้งแต่แรกจึงควรคิดล่วงหน้าว่า **key กลุ่มไหนที่ต้องถูกอ่าน/เขียนพร้อมกันในคำสั่งเดียว** (multi-key operation หรือ transaction) แล้วออกแบบ key ให้มี hash tag ร่วมกันสำหรับกลุ่มนั้น เช่น ระบบที่ต้องอ่าน/เขียนหลาย field ของ user คนเดียวกันพร้อมกันบ่อยๆ ควรใส่ `{user_id}` เป็น hash tag ในทุก key ที่เกี่ยวกับ user คนนั้น เพื่อรับประกันว่า operation ข้าม key เหล่านั้นจะไม่ชนข้อจำกัด `CROSSSLOT`

> **ข้อควรระวัง**: การใช้ hash tag มากเกินไป (บังคับให้ key จำนวนมากไปกระจุกอยู่ slot เดียวกันหมด) ทำลายจุดประสงค์ของการ sharding ไปในตัว เพราะ slot นั้นจะกลายเป็น hotspot รับโหลดหนักกว่า slot อื่นทั้งหมด — ควรใช้ hash tag เท่าที่จำเป็นจริงๆ ตาม access pattern เท่านั้น ไม่ใช่ใส่ตามความเคยชิน

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| Hash slot | key ทุกตัว map ไปยัง 1 ใน 16384 slot คงที่ ด้วย `CRC16(key) mod 16384` |
| Resharding | ย้าย slot ระหว่าง node แบบ online ด้วยสถานะ MIGRATING/IMPORTING, client ได้ ASK ระหว่างย้าย |
| Sentinel | HA สำหรับ deployment ที่ไม่ sharding — monitor + automatic failover ด้วย quorum |
| Cluster | รวม sharding + HA ในตัว แต่ละ slot มี replica ของตัวเอง promote อัตโนมัติเมื่อ primary ล่ม |
| Multi-key operation | ทำงานได้เฉพาะเมื่อทุก key อยู่ slot เดียวกัน ไม่งั้นได้ error CROSSSLOT |
| Hash tag `{...}` | บังคับให้คำนวณ slot จากข้อความในปีกกาเท่านั้น ใช้จัดกลุ่ม key ที่ต้องทำงานร่วมกัน แต่ระวัง hotspot ถ้าใช้มากเกินไป |

## คำถามทบทวน

1. ทำไม Redis Cluster ถึงเลือกใช้ 16384 slot ที่ตายตัว แทนที่จะคำนวณ hash แล้ว map ตรงไปยัง node เลย?
2. อธิบายความต่างระหว่าง Sentinel กับ Cluster — ถ้าระบบมีข้อมูลไม่เกิน 4GB (พอดีกับเครื่องเดียว) แต่ต้องการ high availability ควรเลือกอะไร เพราะเหตุใด?
3. ทำไมคำสั่ง `MGET key1 key2` ถึง error ได้ในโหมด Cluster ทั้งที่คำสั่งเดียวกันทำงานได้ปกติใน Redis instance เดียว?
4. hash tag `{...}` แก้ปัญหาอะไร และมีความเสี่ยงอะไรถ้าใช้ไม่ระมัดระวัง?
5. อธิบายว่า client ที่คุยกับ Redis Cluster รู้ได้อย่างไรว่าต้องส่ง request ไปยัง node ไหน และเกิดอะไรขึ้นถ้าส่งไปผิด node ระหว่างที่ cluster กำลัง resharding

---

ก่อนหน้า: [บทที่ 2 — Pub/Sub และ Streams](02-pubsub-and-streams.md) | กลับสู่ [สารบัญ Volume 6](README.md)
