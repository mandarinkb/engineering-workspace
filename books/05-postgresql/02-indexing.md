# บทที่ 2: Index

Index คือโครงสร้างข้อมูลเสริมที่ช่วยให้ database หาแถวที่ต้องการได้เร็วขึ้นโดยไม่ต้องอ่านทุกแถวในตาราง แต่ index ไม่ใช่ของฟรี — มันมี cost ทั้งพื้นที่จัดเก็บและความเร็วในการเขียนข้อมูล บทนี้จะพาทำความเข้าใจว่า index ทำงานอย่างไรจริงๆ เมื่อไหร่ควรใส่ เมื่อไหร่ไม่ควร และวิธีอ่าน `EXPLAIN ANALYZE` เพื่อพิสูจน์ว่า query ใช้ index จริงหรือไม่ — ไม่ใช่แค่เดา

## 2.1 B-tree Index (ค่า Default)

เมื่อสร้าง `CREATE INDEX idx_name ON table(column);` โดยไม่ระบุชนิด PostgreSQL จะสร้าง **B-tree** ให้เป็นค่า default เพราะรองรับการค้นหาได้หลายรูปแบบและมีประสิทธิภาพดีในกรณีทั่วไปที่สุด:

```
                 [50]
                /    \
           [20,35]    [70,90]
          /   |   \    /  |   \
      [10] [25] [40] [60][80][95]
```

B-tree เก็บข้อมูลแบบเรียงลำดับ (sorted) ทำให้รองรับได้ทั้ง:

- Equality: `WHERE id = 42`
- Range: `WHERE amount BETWEEN 100 AND 500`
- Sorting: `ORDER BY created_at` (ถ้ามี index บน `created_at` database อ่านข้อมูลตามลำดับ index ได้เลยโดยไม่ต้อง sort เพิ่ม)
- Prefix match: `WHERE name LIKE 'Som%'`

ความซับซ้อนของการค้นหาคือ **O(log n)** เทียบกับการอ่านทุกแถว (Sequential Scan) ที่เป็น **O(n)** — ยิ่งตารางใหญ่ ส่วนต่างยิ่งชัด

### เมื่อไหร่ Index ช่วย

- ตารางมีจำนวนแถวมาก และ query กรองข้อมูลจนเหลือสัดส่วนน้อย (selective query) เช่น หา user ด้วย `email` ที่ unique
- คอลัมน์ที่ใช้ join บ่อย (foreign key ควรมี index เสมอ — PostgreSQL **ไม่** สร้าง index ให้ FK อัตโนมัติ ต่างจาก primary key ที่มี index ให้อัตโนมัติ)
- คอลัมน์ที่ใช้ `ORDER BY`/`WHERE` ร่วมกันบ่อย

### เมื่อไหร่ Index ไม่ช่วย (หรือช่วยน้อยจนไม่คุ้ม)

- **Low cardinality** — คอลัมน์ที่มีค่าซ้ำกันเยอะมาก เช่น `status` ที่มีแค่ `'active'`/`'inactive'` สองค่า ถ้าครึ่งหนึ่งของตารางเป็น `'active'` การใช้ index scan แล้วต้องกลับไปอ่านข้อมูลจริงทีละแถว (random I/O) จากครึ่งตาราง มักช้ากว่าอ่านทุกแถวตรงๆ แบบ sequential I/O เสียอีก — query planner ของ PostgreSQL รู้เรื่องนี้และมักจะ**เลือกไม่ใช้ index เอง**แม้จะมี index อยู่ก็ตาม (ดูหัวข้อ 2.4)
- **ตารางเล็ก** — ถ้าทั้งตารางมีไม่กี่ร้อยแถว การ sequential scan ทั้งตารางเร็วพอๆ กับหรือเร็วกว่าการเดิน index อยู่แล้ว เพราะ overhead ในการเปิด index structure ไม่คุ้ม
- **ตารางที่เขียนหนักมาก (write-heavy)** — ทุกครั้งที่ `INSERT`/`UPDATE`/`DELETE` database ต้องอัปเดต**ทุก index**ที่เกี่ยวข้องด้วย ไม่ใช่แค่ตารางหลัก ยิ่งมี index เยอะ ยิ่งเขียนช้าลง และ index ยังกิน disk space เพิ่มอีกต่างหาก ตารางที่ insert ถี่มากๆ (เช่น event log) ควรพิจารณาจำกัดจำนวน index ให้เหลือเท่าที่จำเป็นจริงๆ

> **หลักคิดสรุป**: index ไม่ใช่ "ยิ่งใส่ยิ่งดี" ทุก index คือ trade-off ระหว่างความเร็วในการอ่าน (ดีขึ้น) กับความเร็วในการเขียน + พื้นที่จัดเก็บ (แย่ลง) ต้องดูจาก query pattern จริงของระบบ ไม่ใช่ใส่ตามความรู้สึก

## 2.2 Composite Index และลำดับคอลัมน์

**Composite index** (หรือ multi-column index) คือ index ที่สร้างจากหลายคอลัมน์รวมกัน:

```sql
CREATE INDEX idx_orders_customer_date ON orders(customer_id, created_at);
```

### Leftmost-Prefix Rule

B-tree composite index เรียงข้อมูลตาม**ลำดับคอลัมน์ที่ระบุ** เหมือนสมุดโทรศัพท์ที่เรียงตาม (นามสกุล, ชื่อ) — ถ้าจะหาคนนามสกุล "สมชาย" ทำได้เร็วเพราะข้อมูลเรียงตามนามสกุลเป็นหลัก แต่ถ้าจะหาคนที่ชื่อ "สมชาย" (ไม่สนนามสกุล) สมุดเล่มนี้ช่วยอะไรไม่ได้เลยเพราะคนชื่อสมชายกระจายอยู่ทุกที่ในสมุด

หลักการเดียวกันใช้กับ index `(customer_id, created_at)`:

| Query | ใช้ index ได้ไหม | เหตุผล |
|---|---|---|
| `WHERE customer_id = 1` | ได้ | ใช้ prefix ซ้ายสุด |
| `WHERE customer_id = 1 AND created_at > '2026-01-01'` | ได้เต็มประสิทธิภาพ | ใช้ทั้งสองคอลัมน์ตามลำดับ |
| `WHERE created_at > '2026-01-01'` (ไม่มี customer_id) | **ไม่ได้** (หรือได้แบบไม่มีประสิทธิภาพ) | ข้ามคอลัมน์แรกไป ทำลาย leftmost-prefix |
| `WHERE customer_id = 1 ORDER BY created_at` | ได้ ไม่ต้อง sort เพิ่ม | ตรงกับลำดับ index พอดี |

**กฎปฏิบัติ**: เรียงคอลัมน์ใน composite index จากคอลัมน์ที่ใช้ `=` (equality) บ่อยที่สุด/selective ที่สุดไว้ซ้ายมือ ตามด้วยคอลัมน์ที่ใช้ range (`>`, `<`, `BETWEEN`) หรือ `ORDER BY` ถ้า query pattern มีทั้งสองแบบผสมกัน (บาง query กรองแค่ `customer_id`, บาง query กรองแค่ `created_at`) อาจต้องสร้าง index แยกกัน 2 ตัว ไม่ใช่หวังพึ่ง composite index ตัวเดียวให้ครอบคลุมทุกกรณี

## 2.3 Index ชนิดอื่น: GIN, GiST, BRIN

B-tree ไม่ใช่คำตอบสำหรับทุกสถานการณ์ PostgreSQL มี index type อื่นที่ออกแบบมาสำหรับข้อมูลรูปแบบเฉพาะ:

### GIN (Generalized Inverted Index)

เหมาะกับข้อมูลที่ "หนึ่งแถวมีหลายค่า" เช่น array, JSONB, full-text search vector — ทำงานคล้าย inverted index ของ search engine คือ map จาก "ค่าย่อย" กลับไปหา "แถวที่มีค่านั้น"

```sql
CREATE INDEX idx_tags ON products USING GIN(tags);          -- tags เป็น array
CREATE INDEX idx_data ON events USING GIN(payload);          -- payload เป็น JSONB
```

ใช้กับ operator `@>` (contains), `?` (key exists), full-text `@@` — รายละเอียดเรื่อง JSONB + GIN อยู่ใน [บทที่ 4](04-jsonb.md) เต็มๆ

### GiST (Generalized Search Tree)

โครงสร้างที่ยืดหยุ่นกว่า B-tree รองรับการค้นหาแบบ "overlap"/"nearest" ที่ B-tree ทำไม่ได้ ใช้กับ:

- ข้อมูลภูมิศาสตร์ (PostGIS: จุด, เส้น, พื้นที่ — หา "ร้านค้าใกล้ที่สุดในรัศมี 5km")
- Range type (`tsrange`, `int4range`) — หาช่วงเวลาที่ overlap กัน เช่น ห้องประชุมที่ถูกจองทับซ้อนกัน
- Full-text search (ทางเลือกแทน GIN — GIN เร็วกว่าตอนค้นหาแต่ GiST อัปเดตเร็วกว่าและ index เล็กกว่า)

### BRIN (Block Range Index)

เหมาะกับตารางขนาดใหญ่มาก **ที่ข้อมูลมีความสัมพันธ์กับลำดับการจัดเก็บทางกายภาพ (physical order)** เช่นตาราง log ที่ insert เรียงตามเวลา — BRIN ไม่เก็บตำแหน่งของทุกแถวแบบ B-tree แต่เก็บแค่ "ค่า min/max ของแต่ละช่วง block บนดิสก์" ทำให้ index มีขนาดเล็กมาก (เทียบกับ B-tree ที่ใหญ่กว่ามาก) แลกกับความแม่นยำที่หยาบกว่า — เหมาะกับ time-series/log table ที่มักกรองด้วยช่วงเวลาและข้อมูลถูก insert ตามลำดับเวลาอยู่แล้ว (เชื่อมกับแนวคิด partition ใน [บทที่ 5](05-partitioning-and-replication.md) ที่มักใช้คู่กัน)

```
B-tree: ชี้ตำแหน่งแถวทุกแถว → แม่นยำ 100% แต่ index ใหญ่
BRIN:   เก็บแค่ min/max ต่อ block (เช่น 128 หน้า) → index เล็กมาก แต่ scan ต้องเช็คทั้ง block ที่ range เข้าเงื่อนไข
```

### สรุปเลือก Index Type

| ชนิดข้อมูล/pattern | Index type ที่เหมาะ |
|---|---|
| Equality/range ทั่วไป, sorting | B-tree |
| Array, JSONB, full-text search | GIN |
| Geometric, range overlap | GiST |
| ตารางใหญ่มาก, ข้อมูลเรียงตามเวลา/ลำดับการ insert | BRIN |

## 2.4 อ่าน EXPLAIN ANALYZE

`EXPLAIN` แสดง**แผนการทำงาน (execution plan)** ที่ query planner เลือกใช้ โดยไม่รัน query จริง ส่วน `EXPLAIN ANALYZE` **รันจริง** แล้วแสดงทั้งค่าประมาณการ (estimate) และค่าจริง (actual) คู่กัน — สำคัญมากเพราะความต่างระหว่างสองค่านี้บอกได้ว่า query planner "เข้าใจข้อมูลผิด" หรือไม่

```sql
EXPLAIN ANALYZE
SELECT * FROM orders WHERE customer_id = 1;
```

```
Index Scan using idx_orders_customer_id on orders
  (cost=0.29..8.31 rows=2 width=24)
  (actual time=0.015..0.018 rows=2 loops=1)
  Index Cond: (customer_id = 1)
Planning Time: 0.083 ms
Execution Time: 0.041 ms
```

อ่านทีละส่วน:

- **Node type** (`Index Scan using idx_orders_customer_id`) — วิธีที่ใช้เข้าถึงข้อมูล
- **cost=0.29..8.31** — ค่าประมาณการ (หน่วยไม่ใช่ millisecond แต่เป็นหน่วย cost สัมพัทธ์ของ planner) ตัวเลขแรกคือ cost ก่อนแถวแรกจะออกมา ตัวเลขหลังคือ cost รวมทั้งหมด
- **rows=2** (ใน cost line) — จำนวนแถวที่ planner **คาดว่า**จะได้ (ประมาณการจาก statistics)
- **actual time=0.015..0.018 rows=2 loops=1** — เวลาจริงที่ใช้ (ms) และจำนวนแถวจริงที่ได้ — ถ้า `rows` ใน `actual` ต่างจาก `rows` ใน `cost` มาก (เช่น estimate บอก 2 แต่ actual คือ 50,000) แปลว่า **statistics ของตารางล้าสมัย** ควรรัน `ANALYZE table_name;` เพื่ออัปเดต

### Seq Scan vs Index Scan vs Index Only Scan

```sql
-- Seq Scan: อ่านทุกแถวในตารางเรียงลำดับ physical
EXPLAIN ANALYZE SELECT * FROM orders WHERE amount > 0;
```
```
Seq Scan on orders (cost=0.00..18.50 rows=850 width=24) (actual time=0.010..0.512 rows=850 loops=1)
  Filter: (amount > 0)
```

```sql
-- Index Scan: ใช้ index หาตำแหน่งแถว แล้วย้อนกลับไปอ่านข้อมูลจริงจากตาราง (heap)
EXPLAIN ANALYZE SELECT * FROM orders WHERE customer_id = 1;
```
```
Index Scan using idx_orders_customer_id on orders (cost=0.29..8.31 rows=2 width=24) (actual time=0.015..0.018 rows=2 loops=1)
  Index Cond: (customer_id = 1)
```

```sql
-- Index Only Scan: ข้อมูลที่ query ต้องการ "อยู่ครบใน index อยู่แล้ว" ไม่ต้องย้อนไปอ่านตารางหลักเลย
EXPLAIN ANALYZE SELECT customer_id FROM orders WHERE customer_id = 1;
```
```
Index Only Scan using idx_orders_customer_id on orders (cost=0.29..4.30 rows=2 width=4) (actual time=0.010..0.012 rows=2 loops=1)
  Index Cond: (customer_id = 1)
  Heap Fetches: 0
```

ความต่าง:

- **Seq Scan** — อ่านทุกแถว เหมาะเมื่อต้องอ่านสัดส่วนข้อมูลเยอะ (ไม่ selective) หรือตารางเล็ก
- **Index Scan** — เดิน index หาตำแหน่ง แล้ว**กลับไปอ่านตารางหลัก (heap)** อีกครั้งเพื่อดึงคอลัมน์ที่ query ต้องการแต่ไม่ได้อยู่ใน index (เช่น `SELECT *`) — มี cost เพิ่มจากการ random I/O ไปที่ heap
- **Index Only Scan** — ถ้าทุกคอลัมน์ที่ query ต้องการอยู่ใน index อยู่แล้ว (เรียกว่า **covering index**) ไม่ต้องย้อนไปอ่าน heap เลย เร็วที่สุดในสามแบบ สังเกต `Heap Fetches: 0` ในตัวอย่าง — ถ้าเห็นค่านี้มากกว่า 0 แปลว่า visibility map ยังไม่ update (เกี่ยวกับ MVCC ใน [บทที่ 3](03-transactions-and-mvcc.md)) ทำให้ต้องแวะไปเช็ค heap อยู่บางส่วน

```
                    ┌──────────────┐
Index Scan:         │  B-tree      │──find──►  ┌─────────────┐
                     │  index       │           │  Heap (ตาราง) │──► ได้ข้อมูลครบ
                     └──────────────┘           └─────────────┘
                            │
Index Only Scan:           └────────────────────► ได้ข้อมูลครบจาก index เลย (ถ้า visible)
```

### ทำไม Planner อาจเลือกไม่ใช้ Index ทั้งที่มีอยู่

ถ้าเห็น `Seq Scan` ทั้งที่มี index บนคอลัมน์นั้นอยู่ ไม่ได้แปลว่า index "เสีย" — planner ประเมินจาก **cost model** และ **statistics** (เก็บโดย `ANALYZE`/`autovacuum`) แล้วอาจสรุปว่า sequential scan ถูกกว่าจริงๆ ในกรณี:

- Query จะได้ข้อมูลสัดส่วนใหญ่ของตาราง (low selectivity) ตามที่อธิบายในหัวข้อ 2.1
- ตารางมีขนาดเล็กพอที่ทั้งตารางอยู่ใน memory cache อยู่แล้ว (`shared_buffers`)
- Statistics ล้าสมัย ทำให้ planner ประเมินผิดพลาด (แก้ด้วย `ANALYZE`)

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| B-tree | ค่า default รองรับ equality/range/sort ได้ O(log n) |
| เมื่อไหร่ index ไม่ช่วย | low cardinality, ตารางเล็ก, write-heavy table |
| Composite index | เรียงตามลำดับคอลัมน์ ใช้ได้เฉพาะ leftmost-prefix เท่านั้น |
| GIN | array/JSONB/full-text — inverted index |
| GiST | geometric, range overlap — โครงสร้างยืดหยุ่น |
| BRIN | ตารางใหญ่มาก ข้อมูลเรียงตามลำดับ insert — index เล็กมาก |
| EXPLAIN ANALYZE | เทียบ estimate vs actual rows, Seq/Index/Index Only Scan บอก strategy ที่ใช้จริง |

## คำถามทบทวน

1. อธิบาย leftmost-prefix rule ด้วยตัวอย่างของตัวเอง (ไม่ใช้ตัวอย่างในบทเรียน) — query แบบไหนใช้ composite index `(a, b)` ได้ แบบไหนใช้ไม่ได้
2. ทำไมตารางที่ insert ถี่มากๆ ถึงควรระวังเรื่องจำนวน index ที่สร้าง?
3. Index Scan กับ Index Only Scan ต่างกันตรงไหน อะไรทำให้ query หนึ่งได้ Index Only Scan แต่อีก query หนึ่งบนตารางเดียวกันได้แค่ Index Scan?
4. ถ้าเห็นว่า `EXPLAIN ANALYZE` แสดง estimated rows กับ actual rows ต่างกันมาก ควรทำอะไรก่อน?
5. เพราะเหตุใด BRIN index ถึงเหมาะกับตาราง log ขนาดใหญ่มากกว่า B-tree

---

ก่อนหน้า: [บทที่ 1 — SQL](01-sql-fundamentals.md) | ถัดไป: [บทที่ 3 — Transaction และ MVCC](03-transactions-and-mvcc.md)
