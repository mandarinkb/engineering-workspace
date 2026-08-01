# บทที่ 1: SQL

SQL (Structured Query Language) เป็นภาษาที่ backend engineer ใช้คุยกับฐานข้อมูลบ่อยที่สุด แต่หลายคนเขียน SQL ได้โดยไม่เข้าใจว่า database ประมวลผลคำสั่งตามลำดับตรรกะแบบไหน ทำให้ debug query ที่ผิดพลาด หรือ optimize query ที่ช้า ได้ยาก บทนี้ปูพื้นตั้งแต่การแบ่งประเภทคำสั่ง ไปจนถึง JOIN, CTE, window function และลำดับการประมวลผลจริงของ query

## 1.1 DDL, DML, DQL

คำสั่ง SQL แบ่งเป็น 3 กลุ่มหลักตามหน้าที่:

- **DDL (Data Definition Language)** — นิยามโครงสร้างข้อมูล: `CREATE`, `ALTER`, `DROP`, `TRUNCATE` คำสั่งกลุ่มนี้ส่วนใหญ่ commit ทันที (implicit commit) ใน database บางตัว แต่ PostgreSQL พิเศษตรงที่ **DDL อยู่ใน transaction ได้จริง** — สามารถ `BEGIN; ALTER TABLE ...; ROLLBACK;` แล้วโครงสร้างตารางจะไม่เปลี่ยนเลย ต่างจาก MySQL ที่ DDL ส่วนใหญ่ commit ทันทีไม่ว่าจะอยู่ใน transaction หรือไม่
- **DML (Data Manipulation Language)** — จัดการข้อมูลภายในตาราง: `INSERT`, `UPDATE`, `DELETE`
- **DQL (Data Query Language)** — ดึงข้อมูล: `SELECT` (บางตำราจัด `SELECT` ไว้ใน DML ด้วย แต่แยกออกมาเป็น DQL ช่วยให้คิดเรื่อง read-only query กับ query ที่แก้ไขข้อมูลแยกจากกันชัดเจนขึ้น ซึ่งสำคัญมากตอนคิดเรื่อง read replica ใน [บทที่ 5](05-partitioning-and-replication.md))

> **ทำไมการแบ่งนี้สำคัญ**: เวลาคิดเรื่อง permission (GRANT SELECT แต่ไม่ให้ INSERT/UPDATE) หรือเรื่อง routing query ไป read replica เทียบกับ primary การแยกแนวคิด DQL/DML ออกจากกันช่วยให้ตัดสินใจได้ตรงประเด็นกว่า

## 1.2 JOIN Types

ใช้ schema ตัวอย่างนี้ตลอดทั้งบท:

```sql
CREATE TABLE customers (
    id   INT PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE orders (
    id          INT PRIMARY KEY,
    customer_id INT REFERENCES customers(id),
    amount      NUMERIC(10,2) NOT NULL,
    created_at  DATE NOT NULL
);

INSERT INTO customers VALUES
    (1, 'Somchai'), (2, 'Malee'), (3, 'Anan');

INSERT INTO orders VALUES
    (101, 1, 500.00, '2026-07-01'),
    (102, 1, 250.00, '2026-07-03'),
    (103, 2, 800.00, '2026-07-05');
-- สังเกตว่า customer_id = 3 (Anan) ไม่มี order เลย
-- และไม่มี order ไหนที่ customer_id ชี้ไปยังลูกค้าที่ไม่มีอยู่จริง (เพราะมี FK)
```

### INNER JOIN

คืนแถวที่ **จับคู่ได้ทั้งสองฝั่ง** เท่านั้น:

```sql
SELECT c.name, o.amount
FROM customers c
INNER JOIN orders o ON c.id = o.customer_id;
```

```
 name    | amount
---------+--------
 Somchai | 500.00
 Somchai | 250.00
 Malee   | 800.00
```

`Anan` หายไปเพราะไม่มี order ที่จับคู่ได้

### LEFT JOIN (LEFT OUTER JOIN)

คืนแถวทั้งหมดจากตารางซ้าย แม้จะไม่มีคู่ในตารางขวา (เติม `NULL` แทน):

```sql
SELECT c.name, o.amount
FROM customers c
LEFT JOIN orders o ON c.id = o.customer_id;
```

```
 name    | amount
---------+--------
 Somchai | 500.00
 Somchai | 250.00
 Malee   | 800.00
 Anan    | NULL
```

ใช้บ่อยที่สุดในงานจริงเวลาต้องการ "รายการหลักทั้งหมด พร้อมข้อมูลเสริมถ้ามี" เช่น รายชื่อลูกค้าทั้งหมดพร้อมยอดสั่งซื้อล่าสุด (ถ้ายังไม่เคยสั่งก็ให้เห็นเป็น NULL ไม่ใช่หายไปจากรายงาน)

### RIGHT JOIN (RIGHT OUTER JOIN)

ตรงข้ามกับ LEFT JOIN — คืนแถวทั้งหมดจากตารางขวา ในทางปฏิบัติแทบไม่มีใครเขียน RIGHT JOIN เพราะสลับตำแหน่งตารางแล้วใช้ LEFT JOIN ได้ผลลัพธ์เหมือนกันและอ่านง่ายกว่า

### FULL JOIN (FULL OUTER JOIN)

คืนแถวทั้งหมดจากทั้งสองฝั่ง ไม่ว่าจะจับคู่ได้หรือไม่ — เติม `NULL` ในฝั่งที่ไม่มีคู่:

```sql
SELECT c.name, o.amount
FROM customers c
FULL JOIN orders o ON c.id = o.customer_id;
```

ใช้เมื่อต้องการเห็น "ทุกอย่างจากทั้งสองตาราง" เช่น หา record ที่กำพร้า (orphan) ทั้งสองทิศทางพร้อมกัน — orders ที่ไม่มี customer จริง กับ customers ที่ไม่มี order เลย ในคำสั่งเดียว

### CROSS JOIN

จับคู่ **ทุกแถวกับทุกแถว** (Cartesian product) ไม่มีเงื่อนไข ON:

```sql
SELECT c.name, p.day
FROM customers c
CROSS JOIN (VALUES ('Mon'), ('Tue'), ('Wed')) AS p(day);
```

ได้ผลลัพธ์ 3 customers × 3 days = 9 แถว ใช้จริงตอนสร้าง "ตารางความเป็นไปได้ทั้งหมด" เช่น matrix ของ (สาขา × วันในสัปดาห์) เพื่อเติม default value ให้ครบทุก combination ก่อนจะ LEFT JOIN กับข้อมูลจริง

```
              INNER          LEFT           FULL
customers   ┌──────┐      ┌──────┐       ┌──────┐
   ●●●○     │  ●●  │      │ ●●●  │       │ ●●●○ │
orders      │      │      │      │       │      │
   ●●□      │      │      │      │       │  □   │
            └──────┘      └──────┘       └──────┘
● = จับคู่ได้   ○ = customer ไม่มี order   □ = order กำพร้า (สมมติ)
```

## 1.3 Subquery vs CTE

### Subquery

query ที่ซ้อนอยู่ใน query อื่น มีทั้งแบบ **scalar** (คืนค่าเดียว), **row** (คืนหนึ่งแถว), และ **correlated** (อ้างอิงค่าจาก query ภายนอก — รันซ้ำทุกแถว):

```sql
-- Correlated subquery: หา order ที่มากที่สุดของแต่ละ customer
SELECT o1.*
FROM orders o1
WHERE o1.amount = (
    SELECT MAX(o2.amount)
    FROM orders o2
    WHERE o2.customer_id = o1.customer_id
);
```

correlated subquery มักถูก execute ซ้ำต่อแถวของ query ภายนอก (แม้ query planner จะพยายาม optimize เป็น join ภายในให้ในหลายกรณี) ถ้า logic ซับซ้อน มักอ่านยากกว่าการเขียนเป็น JOIN หรือ CTE

### CTE (Common Table Expression)

ใช้ `WITH` ตั้งชื่อผลลัพธ์ของ query ย่อยไว้ล่วงหน้า อ่านง่ายกว่า subquery ที่ซ้อนลึกมาก เพราะแยก logic เป็นขั้นตอนที่มีชื่อ:

```sql
WITH customer_totals AS (
    SELECT customer_id, SUM(amount) AS total
    FROM orders
    GROUP BY customer_id
)
SELECT c.name, ct.total
FROM customers c
JOIN customer_totals ct ON ct.customer_id = c.id
WHERE ct.total > 400;
```

> **หมายเหตุเรื่อง performance**: PostgreSQL ตั้งแต่เวอร์ชัน 12 เป็นต้นไป จะ **inline** CTE เข้าไปใน query หลักโดยอัตโนมัติถ้าเป็นไปได้ (เหมือน subquery) ยกเว้นจะระบุ `MATERIALIZED` บังคับให้แยกคำนวณเป็นก้อนต่างหาก ก่อนเวอร์ชัน 12 ทุก CTE ถือเป็น **optimization fence** (จุดที่ query planner หยุด optimize ข้ามผ่าน) ทำให้บาง query ที่ย้ายจาก subquery มาเป็น CTE แล้วช้าลงอย่างไม่คาดคิด ถ้าต้องรองรับเวอร์ชันเก่าหรือไม่แน่ใจ ควรเช็คด้วย `EXPLAIN ANALYZE` เสมอ (ดูรายละเอียดการอ่าน plan ใน [บทที่ 2](02-indexing.md))

### Recursive CTE

ใช้แก้ปัญหาโครงสร้างแบบต้นไม้/กราฟ เช่น organization hierarchy ที่ query ปกติทำไม่ได้เพราะไม่รู้ล่วงหน้าว่าจะลึกกี่ level:

```sql
CREATE TABLE employees (
    id        INT PRIMARY KEY,
    name      TEXT,
    manager_id INT REFERENCES employees(id)
);

INSERT INTO employees VALUES
    (1, 'CEO Somsak', NULL),
    (2, 'VP Malee',   1),
    (3, 'Manager Anan', 2),
    (4, 'Engineer Kob', 3);

WITH RECURSIVE org_chart AS (
    -- Base case: จุดเริ่มต้น (CEO ไม่มี manager)
    SELECT id, name, manager_id, 0 AS depth
    FROM employees
    WHERE manager_id IS NULL

    UNION ALL

    -- Recursive case: หาลูกน้องของแต่ละคนใน org_chart ที่คำนวณไปแล้ว
    SELECT e.id, e.name, e.manager_id, oc.depth + 1
    FROM employees e
    JOIN org_chart oc ON e.manager_id = oc.id
)
SELECT REPEAT('  ', depth) || name AS org_tree
FROM org_chart
ORDER BY depth;
```

```
 org_tree
------------------------
 CEO Somsak
   VP Malee
     Manager Anan
       Engineer Kob
```

Recursive CTE ทำงานเป็นวงวน: base case สร้างแถวตั้งต้น แล้ว recursive case ดึงแถวใหม่จากรอบก่อนหน้ามาต่อยอดเรื่อยๆ จนไม่มีแถวใหม่เกิดขึ้นอีก (สำคัญ: ต้องมั่นใจว่าข้อมูลไม่มี cycle เช่น A เป็น manager ของ B และ B เป็น manager ของ A ไม่งั้น query จะวนไม่จบ)

## 1.4 Window Function

Window function คำนวณค่าข้าม **กลุ่มแถว (window)** โดย**ไม่ยุบแถวให้เหลือแถวเดียวแบบ** `GROUP BY` — นี่คือความต่างที่สำคัญที่สุด: `GROUP BY` ลดจำนวนแถว แต่ window function ยังคงจำนวนแถวเดิมไว้ครบ แค่แนบผลคำนวณเพิ่มเข้าไปในแต่ละแถว

### Running Total

```sql
SELECT
    customer_id,
    created_at,
    amount,
    SUM(amount) OVER (
        PARTITION BY customer_id
        ORDER BY created_at
    ) AS running_total
FROM orders;
```

```
 customer_id | created_at | amount | running_total
-------------+------------+--------+---------------
 1           | 2026-07-01 | 500.00 | 500.00
 1           | 2026-07-03 | 250.00 | 750.00
 2           | 2026-07-05 | 800.00 | 800.00
```

`PARTITION BY customer_id` แบ่ง window เป็นกลุ่มย่อยตาม customer (เหมือน GROUP BY แต่ไม่ยุบแถว) และ `ORDER BY created_at` กำหนดลำดับสะสมภายในแต่ละ partition

### RANK, DENSE_RANK, ROW_NUMBER

```sql
SELECT
    customer_id,
    amount,
    ROW_NUMBER() OVER (ORDER BY amount DESC) AS row_num,
    RANK()       OVER (ORDER BY amount DESC) AS rank,
    DENSE_RANK() OVER (ORDER BY amount DESC) AS dense_rank
FROM orders;
```

ความต่างสำคัญเมื่อมีค่าเท่ากัน (tie): สมมติมี order 2 รายการที่ `amount = 500.00` เท่ากันอยู่อันดับ 1

- **ROW_NUMBER** — ให้เลขไม่ซ้ำเสมอ (1, 2, 3, 4, ...) แม้ค่าจะเท่ากัน ต้องมี tie-breaker ที่ deterministic ไม่งั้นลำดับระหว่างแถวที่ tie กันอาจไม่คงที่ทุกครั้งที่รัน
- **RANK** — ให้อันดับเท่ากันถ้าค่าเท่ากัน แล้ว "ข้าม" อันดับถัดไป (1, 1, 3, 4, ...)
- **DENSE_RANK** — ให้อันดับเท่ากันถ้าค่าเท่ากัน แต่ "ไม่ข้าม" อันดับถัดไป (1, 1, 2, 3, ...)

ใช้ `RANK()`/`DENSE_RANK()` แบบ `PARTITION BY` บ่อยมากในงาน "หา top-N ต่อกลุ่ม" เช่น order ที่มูลค่าสูงสุด 3 อันดับแรกของลูกค้าแต่ละคน:

```sql
SELECT * FROM (
    SELECT *, DENSE_RANK() OVER (PARTITION BY customer_id ORDER BY amount DESC) AS rnk
    FROM orders
) t
WHERE rnk <= 3;
```

## 1.5 Aggregate Function, GROUP BY/HAVING และลำดับการประมวลผลจริงของ Query

### Aggregate Function พื้นฐาน

`COUNT`, `SUM`, `AVG`, `MIN`, `MAX` — ยุบหลายแถวให้เหลือค่าเดียวต่อกลุ่ม (หรือค่าเดียวทั้งตารางถ้าไม่มี `GROUP BY`)

```sql
SELECT customer_id, COUNT(*) AS order_count, SUM(amount) AS total_amount
FROM orders
GROUP BY customer_id
HAVING SUM(amount) > 400;
```

`WHERE` กรอง**แถว**ก่อนจะจัดกลุ่ม ส่วน `HAVING` กรอง**กลุ่ม**หลังจากคำนวณ aggregate แล้ว — นี่คือเหตุผลที่ `HAVING SUM(amount) > 400` เขียนได้ แต่ `WHERE SUM(amount) > 400` เขียนไม่ได้ (ตอน WHERE ทำงาน ยังไม่มีการคำนวณ SUM เกิดขึ้นเลย)

### ลำดับการประมวลผลจริงของ SQL (Logical Query Processing Order)

จุดที่ทำให้ผู้เขียน SQL สับสนบ่อยที่สุดคือ **ลำดับที่เขียนคำสั่ง** (`SELECT ... FROM ... WHERE ...`) **ไม่ใช่**ลำดับที่ database ประมวลผลจริง ลำดับจริงคือ:

```
1. FROM      — กำหนดตารางต้นทาง (รวม JOIN ทั้งหมด)
2. WHERE     — กรองแถวดิบ ก่อนจัดกลุ่มใดๆ
3. GROUP BY  — จัดกลุ่มแถวที่เหลือ
4. HAVING    — กรองกลุ่มที่จัดแล้ว (ใช้ aggregate function ได้)
5. SELECT    — เลือกคอลัมน์ที่จะแสดง (รวมถึงคำนวณ expression/alias)
6. DISTINCT  — ตัดแถวซ้ำ (ถ้ามี)
7. ORDER BY  — เรียงลำดับผลลัพธ์สุดท้าย
8. LIMIT/OFFSET — ตัดจำนวนแถวที่จะคืน
```

```
เขียนแบบนี้:        SELECT col FROM table WHERE ... GROUP BY ... HAVING ... ORDER BY ...
ประมวลผลจริงแบบนี้:  FROM → WHERE → GROUP BY → HAVING → SELECT → DISTINCT → ORDER BY → LIMIT
```

ผลลัพธ์เชิงปฏิบัติของลำดับนี้ที่เจอบ่อย:

- **ใช้ column alias ใน `WHERE` ไม่ได้** เพราะ `WHERE` ทำงานก่อน `SELECT` จะคำนวณ alias นั้นเสร็จ (`SELECT amount * 1.07 AS total FROM orders WHERE total > 100` — ผิด! `total` ยังไม่มีตัวตนตอน `WHERE` รัน)
- **ใช้ column alias ใน `ORDER BY` ได้** เพราะ `ORDER BY` ทำงานหลัง `SELECT` เสร็จแล้ว (`... ORDER BY total` ใช้ได้)
- **ใช้ aggregate function ใน `WHERE` ไม่ได้** ต้องใช้ `HAVING` แทน เพราะตอน `WHERE` ทำงาน ยังไม่มีการ `GROUP BY`/คำนวณ aggregate เกิดขึ้นเลย
- PostgreSQL อนุญาตให้ `GROUP BY` อ้างชื่อ alias จาก `SELECT` ได้ (ส่วนขยายที่สะดวกกว่า SQL standard บาง engine) แต่ `WHERE`/`HAVING` ยังคงทำตามกฎเดิม

การเข้าใจลำดับนี้ยังช่วยอธิบายว่าทำไม window function (หัวข้อ 1.4) ถึงคำนวณ**หลัง** `WHERE`/`GROUP BY`/`HAVING` แต่**ก่อน** `ORDER BY`/`LIMIT` สุดท้าย — window function จึง "เห็น" เฉพาะแถวที่ผ่านการกรองมาแล้ว แต่ยังไม่ถูกตัด `LIMIT` ทิ้ง

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| DDL/DML/DQL | DDL นิยามโครงสร้าง (อยู่ใน transaction ได้ใน PostgreSQL), DML แก้ข้อมูล, DQL อ่านข้อมูล |
| JOIN | INNER ตัดแถวไม่จับคู่ทิ้ง, LEFT/RIGHT/FULL เก็บแถวที่ไม่จับคู่ไว้ (เติม NULL), CROSS จับคู่ทุกแถว |
| Subquery vs CTE | CTE อ่านง่ายกว่า แต่ PostgreSQL ≥12 inline CTE อัตโนมัติ ต่างจากก่อนหน้าที่เป็น optimization fence |
| Recursive CTE | แก้โครงสร้างต้นไม้/กราฟที่ไม่รู้ความลึกล่วงหน้า ระวังข้อมูลที่มี cycle |
| Window function | คำนวณข้ามกลุ่มแถวโดยไม่ยุบจำนวนแถว ต่างจาก GROUP BY ที่ยุบแถว |
| Logical query order | FROM → WHERE → GROUP BY → HAVING → SELECT → ORDER BY → LIMIT ไม่ใช่ลำดับที่เขียน |

## คำถามทบทวน

1. ทำไม `WHERE SUM(amount) > 100` ถึงเขียนไม่ได้ ต้องใช้ `HAVING` แทน อธิบายด้วยลำดับการประมวลผลจริง
2. LEFT JOIN กับ INNER JOIN ต่างกันอย่างไรในผลลัพธ์ ยกตัวอย่างสถานการณ์ที่ต้องใช้ LEFT JOIN เท่านั้น
3. RANK, DENSE_RANK, ROW_NUMBER ให้ผลต่างกันอย่างไรเมื่อมีค่าเท่ากัน (tie)?
4. เพราะเหตุใด CTE ในเวอร์ชัน PostgreSQL เก่าถึงอาจทำให้ query ช้าลงเมื่อเทียบกับการเขียนเป็น subquery?
5. อธิบาย recursive CTE ด้วยคำพูดตัวเอง และบอกอันตรายที่ต้องระวังเมื่อข้อมูลมี cycle

---

ถัดไป: [บทที่ 2 — Index](02-indexing.md)
