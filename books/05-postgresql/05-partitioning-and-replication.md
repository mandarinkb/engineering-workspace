# บทที่ 5: Partitioning และ Replication

บทสุดท้ายของ volume นี้ครอบคลุมสองเรื่องที่จำเป็นเมื่อระบบโตเกินกว่าที่ instance เดียวจะรับมือได้อย่างมีประสิทธิภาพ: **Partitioning** แบ่งตารางใหญ่ให้จัดการง่ายขึ้นภายใน instance เดียว และ **Replication** กระจายข้อมูลไปหลาย instance เพื่อความพร้อมใช้งาน (availability) และการ scale การอ่าน ทั้งสองเรื่องเชื่อมโยงกันบ่อยในสถาปัตยกรรมจริง — ตารางที่ partition ดีมักอยู่ในระบบที่มี replication ด้วยเช่นกัน

## 5.1 Partitioning

### ทำไมต้อง Partition

ตารางที่มีข้อมูลหลักสิบ/หลักร้อยล้านแถว แม้จะมี index ที่ดีแล้ว ก็ยังเจอปัญหาเชิงปฏิบัติ: index ใหญ่มากจนบางส่วนไม่พอดีกับ memory cache, `VACUUM` (จาก[บทที่ 3](03-transactions-and-mvcc.md)) ใช้เวลานานขึ้นเรื่อยๆ ตามขนาดตาราง, และการลบข้อมูลเก่า (เช่น log ที่เก็บเกิน 1 ปี) ด้วย `DELETE` ธรรมดาจะสร้าง dead tuple มหาศาลในครั้งเดียว **Partitioning** คือการแบ่งตารางเชิงตรรกะหนึ่งตาราง ให้ PostgreSQL เก็บเป็นตารางย่อยหลายตารางบนดิสก์จริงๆ (แต่ยังคง query ผ่านตารางแม่เป็นหนึ่งเดียวได้เหมือนเดิม)

### Range Partitioning

แบ่งตามช่วงค่า — เหมาะกับข้อมูล time-series ที่สุด:

```sql
CREATE TABLE events (
    id         BIGSERIAL,
    payload    JSONB,
    created_at DATE NOT NULL
) PARTITION BY RANGE (created_at);

CREATE TABLE events_2026_06 PARTITION OF events
    FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');

CREATE TABLE events_2026_07 PARTITION OF events
    FOR VALUES FROM ('2026-07-01') TO ('2026-08-01');
```

```
                    events (ตารางแม่ เชิงตรรกะ)
                          │
        ┌─────────────────┼─────────────────┐
        ▼                 ▼                 ▼
  events_2026_06    events_2026_07    events_2026_08
  (มิ.ย. เท่านั้น)    (ก.ค. เท่านั้น)    (ส.ค. เท่านั้น)
```

ลบข้อมูลเก่าทำได้เร็วมากด้วย `DROP TABLE events_2026_06;` (หรือ `DETACH PARTITION` ก่อนแล้วค่อย drop) — เร็วกว่า `DELETE FROM events WHERE created_at < '2026-07-01'` มหาศาล เพราะ `DROP TABLE` แค่ลบไฟล์ทิ้งไม่ต้องสร้าง dead tuple ทีละแถวแล้วรอ VACUUM เก็บกวาดทีหลัง

### List Partitioning

แบ่งตามค่าที่กำหนดชัดเจนเป็นกลุ่มๆ (ไม่ใช่ช่วง):

```sql
CREATE TABLE orders (
    id      BIGSERIAL,
    region  TEXT NOT NULL,
    amount  NUMERIC(10,2)
) PARTITION BY LIST (region);

CREATE TABLE orders_th PARTITION OF orders FOR VALUES IN ('TH');
CREATE TABLE orders_sg PARTITION OF orders FOR VALUES IN ('SG');
CREATE TABLE orders_other PARTITION OF orders DEFAULT;  -- รับค่าที่ไม่ตรง partition ไหนเลย
```

เหมาะกับข้อมูลที่แบ่งกลุ่มตาม category ชัดเจน เช่น ภูมิภาค, tenant ID (ในระบบ multi-tenant), หรือ status

### Hash Partitioning

แบ่งด้วย hash function เมื่อไม่มี key ที่แบ่งช่วง/กลุ่มได้ตามธรรมชาติ แต่ต้องการกระจายข้อมูลให้เท่าๆ กันเพื่อลดขนาดแต่ละ partition:

```sql
CREATE TABLE users (
    id   BIGSERIAL,
    name TEXT
) PARTITION BY HASH (id);

CREATE TABLE users_p0 PARTITION OF users FOR VALUES WITH (MODULUS 4, REMAINDER 0);
CREATE TABLE users_p1 PARTITION OF users FOR VALUES WITH (MODULUS 4, REMAINDER 1);
CREATE TABLE users_p2 PARTITION OF users FOR VALUES WITH (MODULUS 4, REMAINDER 2);
CREATE TABLE users_p3 PARTITION OF users FOR VALUES WITH (MODULUS 4, REMAINDER 3);
```

ข้อเสียคือคาดเดายากว่าแถวไหนอยู่ partition ไหนโดยไม่คำนวณ hash เอง จึงไม่ได้ประโยชน์เรื่อง "ลบข้อมูลทั้ง partition แบบ range/list" — ใช้เมื่อเป้าหมายหลักคือกระจายโหลดให้เท่ากันเท่านั้น ไม่ใช่การจัดกลุ่มเชิงความหมาย

### Partition Pruning

ประโยชน์หลักของ partitioning ในแง่ performance คือ **partition pruning** — เมื่อ query มีเงื่อนไขที่ query planner รู้ล่วงหน้าว่าตรงกับ partition ไหนบ้าง (จากค่าคงที่ใน `WHERE`) planner จะ**ข้าม partition ที่ไม่เกี่ยวข้องไปเลยโดยไม่แตะต้อง**:

```sql
EXPLAIN SELECT * FROM events WHERE created_at = '2026-07-15';
```

```
Append
  ->  Seq Scan on events_2026_07
        Filter: (created_at = '2026-07-15')
(ไม่มี events_2026_06 หรือ events_2026_08 ปรากฏใน plan เลย — ถูก prune ทิ้งไปตั้งแต่ planning)
```

ผลลัพธ์คือ query ที่กรองด้วยคอลัมน์ partition key จะเร็วขึ้นมาก เพราะ database ไม่ต้อง scan ข้อมูลของเดือนอื่นที่ไม่เกี่ยวข้องเลย — เหมือนมี index ระดับหยาบตัวใหญ่ที่ตัดทั้งกลุ่มข้อมูลออกได้ทีเดียวก่อนแม้แต่จะเริ่มมองหาแถว (เชื่อมกับแนวคิด BRIN index ใน [บทที่ 2](02-indexing.md) ที่ใช้หลักการคล้ายกันแต่ทำงานในระดับ block ไม่ใช่ระดับตาราง)

> **ข้อควรระวัง**: partition pruning ทำงานได้ดีเมื่อเงื่อนไขใน `WHERE` เป็นค่าคงที่ที่รู้ตอน planning หรืออย่างน้อย runtime pruning (PostgreSQL รุ่นใหม่รองรับ) — ถ้า partition key ไม่ได้ถูกใช้ใน `WHERE` เลย query จะต้อง scan ทุก partition เหมือนตารางไม่ได้ partition อยู่ดี

## 5.2 Replication

### Streaming Replication (Physical)

**Physical replication** ทำงานที่ระดับ **WAL** (จาก[บทที่ 3](03-transactions-and-mvcc.md)) — primary ส่ง WAL record ที่เกิดขึ้นทุกการเปลี่ยนแปลงไปให้ replica แบบ stream ต่อเนื่อง replica นำ WAL มา replay ซ้ำเพื่อให้ข้อมูลตรงกับ primary แบบ byte-for-byte เหมือนกันทุกตาราง ทุก index — ไม่สามารถเลือก replicate เฉพาะบางตารางได้ เพราะทำงานที่ระดับไฟล์ ไม่ใช่ระดับตาราง

```
Primary ──WAL stream──► Replica 1
        ──WAL stream──► Replica 2
```

ใช้เพื่อ high availability (failover) และ scale การอ่าน (read replica) — replica เป็น **สำเนาทั้ง cluster** เหมือนกันทุกประการ

### Logical Replication

ทำงานที่ระดับ **แถวข้อมูล** ไม่ใช่ WAL ไฟล์ดิบ — ใช้ระบบ publish/subscribe เลือก replicate ได้เฉพาะบางตาราง หรือแม้แต่บางคอลัมน์/บาง row (ผ่าน row filter) ได้:

```sql
-- ฝั่ง primary (publisher)
CREATE PUBLICATION orders_pub FOR TABLE orders;

-- ฝั่ง replica (subscriber)
CREATE SUBSCRIPTION orders_sub
    CONNECTION 'host=primary dbname=mydb'
    PUBLICATION orders_pub;
```

ข้อดีของ logical replication ที่ physical ทำไม่ได้:

- Replicate ข้าม PostgreSQL major version ต่างกันได้ (physical replication ต้องเวอร์ชันตรงกัน)
- เลือก replicate เฉพาะตารางที่ต้องการ (เช่น ETL ข้อมูลบางส่วนไปยัง data warehouse โดยไม่ต้อง copy ทั้ง database)
- Replica สามารถเขียนข้อมูลอื่นที่ไม่เกี่ยวกับตารางที่ subscribe ได้ (physical replica เป็น read-only ทั้งหมด)

แลกกับ overhead ที่สูงกว่า physical replication (ต้อง decode การเปลี่ยนแปลงเป็นระดับแถว ไม่ใช่ copy WAL ดิบตรงๆ)

### Synchronous vs Asynchronous Replication

นี่คือจุดที่เชื่อมโยงกับแนวคิด**การแลกเปลี่ยนระหว่าง consistency, latency, และ availability** ที่จะถูกอธิบายอย่างเป็นทางการมากขึ้นด้วย CAP theorem ใน [Volume 17 — System Design](../17-system-design/README.md):

- **Asynchronous replication** (ค่า default) — primary commit transaction แล้ว**ตอบกลับ client ทันที** โดยไม่รอ replica ยืนยันว่าได้รับ WAL แล้ว ข้อดีคือ latency ต่ำ (ไม่ต้องรอ network round-trip ไปยัง replica) ข้อเสียคือถ้า primary ล่มทันทีหลัง commit ก่อนที่ replica จะทัน sync ข้อมูลบางส่วนที่ client เพิ่งได้รับการยืนยันว่า "commit สำเร็จ" อาจ**หายไป**จาก replica ที่กลายเป็น primary ใหม่หลัง failover (**data loss แบบ small window**)
- **Synchronous replication** — primary จะตอบกลับ client ว่า commit สำเร็จ **ก็ต่อเมื่อ** replica (อย่างน้อยตัวที่กำหนดใน `synchronous_standby_names`) ยืนยันว่าได้รับ WAL แล้วเท่านั้น รับประกันว่าข้อมูลที่ commit แล้วจะไม่หายแม้ primary ล่มทันที แลกกับ **latency สูงขึ้น** (ต้องรอ network round-trip ไปกลับกับ replica ทุก commit) และถ้า replica ที่กำหนดไว้ down หรือ network มีปัญหา primary อาจ**บล็อกการ commit ทั้งหมด**จนกว่าจะติดต่อ replica ได้ (กระทบ availability โดยตรง)

```
Async:  Client ──commit──► Primary ──ตอบ OK ทันที──► Client
                              │
                              └──ส่ง WAL──► Replica (อาจตามหลังนิดหน่อย)

Sync:   Client ──commit──► Primary ──รอ ACK จาก Replica──► ตอบ OK
                              │                    ▲
                              └──ส่ง WAL──► Replica ┘
```

นี่คือตัวอย่างรูปธรรมของการแลกเปลี่ยนระหว่าง **consistency** (ข้อมูลไม่หาย), **latency** (เวลาตอบสนอง), และ **availability** (ระบบยังรับ write ต่อได้แม้ replica มีปัญหา) — ไม่มีค่าไหนได้มาโดยไม่เสียอีกค่าหนึ่งไปบางส่วน ทีมออกแบบระบบต้องเลือกจุดสมดุลตาม business requirement จริง เช่น ระบบ payment อาจยอมรับ latency สูงขึ้นเพื่อไม่ยอมให้ transaction หายแม้แต่รายการเดียว ในขณะที่ระบบ analytics/log อาจเลือก async เพื่อความเร็วเพราะยอมรับการสูญเสียข้อมูลเล็กน้อยได้

## 5.3 Failover และ Read Replica

### Failover

เมื่อ primary ล่ม ระบบต้อง **promote** replica ตัวหนึ่งขึ้นเป็น primary ใหม่ ในการทำงานจริงมักไม่ทำ manual แต่ใช้เครื่องมือจัดการอัตโนมัติ (เช่น Patroni, pg_auto_failover) ที่คอย monitor สุขภาพของ primary แล้วสั่ง promote replica ที่ข้อมูลล่าสุดที่สุดโดยอัตโนมัติเมื่อตรวจพบว่า primary ไม่ตอบสนอง — ประเด็นสำคัญที่ต้องระวังคือ **split-brain** (primary เก่ากลับมาออนไลน์พร้อมกับ replica ที่เพิ่งถูก promote เป็น primary ใหม่ ทำให้มี "primary" สองตัวรับ write พร้อมกัน) ซึ่งเครื่องมือ failover ที่ดีต้องมีกลไกป้องกัน (เช่น fencing, consensus ผ่าน distributed lock)

### Read Replica สำหรับ Scale การอ่าน

ระบบที่ read-heavy (อ่านมากกว่าที่เขียนมาก เช่น dashboard, reporting) สามารถกระจาย query แบบ `SELECT` ไปยัง read replica หลายตัว โดยให้ primary รับเฉพาะ write เท่านั้น — เชื่อมโยงกับการแบ่ง DQL/DML จาก [บทที่ 1](01-sql-fundamentals.md): แอปพลิเคชันต้องรู้ตัวว่า query ไหนเป็น DQL (route ไป replica ได้) กับ DML (ต้อง route ไป primary เท่านั้น) ข้อควรระวังคือ **replication lag** — เพราะ replica (โดยเฉพาะ async) อาจข้อมูลตามหลัง primary อยู่เสมอ ถ้าแอปพลิเคชัน `INSERT` ที่ primary แล้วรีบ `SELECT` อ่านค่าที่เพิ่ง insert จาก replica ทันที อาจ**ไม่เจอข้อมูลนั้น**เพราะยัง sync ไม่ทัน (read-your-own-write consistency ไม่รับประกันข้าม replica) รูปแบบทั่วไปในการแก้ปัญหานี้คือ บังคับให้อ่านข้อมูลที่เพิ่งเขียนเองกลับจาก primary โดยตรงในช่วงเวลาสั้นๆ หลัง write หรือใช้ session ที่ pin ไปยัง primary ชั่วคราว

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| Range partition | แบ่งตามช่วงค่า เหมาะกับ time-series, ลบข้อมูลเก่าเร็วด้วย DROP TABLE |
| List partition | แบ่งตามกลุ่มค่าที่กำหนดชัดเจน เช่น region, tenant |
| Hash partition | กระจายโหลดเท่าๆ กันเมื่อไม่มี key แบ่งกลุ่มตามธรรมชาติ |
| Partition pruning | planner ข้าม partition ที่ไม่เกี่ยวข้องได้ทั้งกลุ่ม ทำให้ query เร็วขึ้นมาก |
| Physical replication | ระดับ WAL ทั้ง cluster เหมือนกันทุกไบต์ ใช้ทำ HA/read replica |
| Logical replication | ระดับแถว เลือก replicate เฉพาะตารางได้ ข้าม major version ได้ |
| Sync vs Async replication | Sync: ไม่มี data loss แต่ latency สูง/เสี่ยง block; Async: latency ต่ำแต่เสี่ยง data loss เมื่อ failover |
| Read replica | scale การอ่านได้ แต่ต้องรับมือ replication lag ด้วย |

## คำถามทบทวน

1. ทำไม `DROP TABLE` บน partition ถึงเร็วกว่า `DELETE FROM ... WHERE` บนตารางที่ไม่ได้ partition มาก?
2. Partition pruning ช่วยประหยัด work ของ query planner/executor อย่างไร ต้องมีเงื่อนไขอะไรถึงจะเกิดขึ้นได้?
3. Logical replication ทำอะไรได้ที่ physical replication ทำไม่ได้ ยกตัวอย่าง use case ที่ต้องใช้ logical replication เท่านั้น
4. อธิบาย trade-off ระหว่าง synchronous กับ asynchronous replication ในแง่ consistency, latency, และ availability
5. Replication lag ส่งผลกระทบต่อระบบที่ใช้ read replica อย่างไร ยกตัวอย่างสถานการณ์ที่เป็นปัญหาจริง และวิธีแก้ 1 วิธี

---

ก่อนหน้า: [บทที่ 4 — JSONB](04-jsonb.md) | กลับสู่ [สารบัญ Volume 5](README.md)
