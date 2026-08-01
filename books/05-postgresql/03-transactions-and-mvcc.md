# บทที่ 3: Transaction และ MVCC

นี่คือบทที่ลึกที่สุดของ volume นี้ เพราะเป็นเรื่องที่ backend engineer ส่วนใหญ่ใช้งานทุกวันโดยไม่เข้าใจกลไกเบื้องหลังจริงๆ — เขียน `BEGIN...COMMIT` ได้ แต่ตอบไม่ได้ว่าทำไม transaction สอง connection ถึงเห็นข้อมูลไม่ตรงกันชั่วขณะ หรือทำไม production ถึงต้องรัน `VACUUM` บทนี้จะอธิบาย ACID, isolation level และ locking ก่อน แล้วเชื่อมทุกอย่างเข้ากับกลไกจริงที่ PostgreSQL ใช้ implement มันทั้งหมด นั่นคือ **MVCC (Multi-Version Concurrency Control)**

## 3.1 ACID Properties

Transaction คือกลุ่มคำสั่งที่ database รับประกันว่าจะทำงานเป็น "หน่วยเดียว" ตามคุณสมบัติ 4 ข้อ:

- **Atomicity** — transaction ทำสำเร็จทั้งหมดหรือไม่สำเร็จเลย (all-or-nothing) ถ้ามีคำสั่งใดคำสั่งหนึ่งล้มเหลว ทุกอย่างที่ทำไปแล้วใน transaction นั้นต้องถูก rollback กลับ ตัวอย่างที่จับต้องได้: โอนเงินจากบัญชี A ไป B ต้อง `UPDATE` บัญชี A (หัก) และ `UPDATE` บัญชี B (เพิ่ม) พร้อมกัน ถ้าระบบล่มหลังหักเงิน A แต่ยังไม่ทันเพิ่มเงิน B, atomicity รับประกันว่าเงินที่หักจาก A จะถูก rollback กลับ ไม่ใช่หายไปเฉยๆ
- **Consistency** — transaction ที่สำเร็จต้องพาข้อมูลจากสถานะที่ถูกต้องหนึ่ง ไปสู่สถานะที่ถูกต้องอีกอัน (ตาม constraint, foreign key, check constraint ที่กำหนดไว้) — นี่คือ "C" ที่ database บังคับด้วย constraint ต่างๆ ไม่ใช่สิ่งที่ transaction "ทำให้เกิด" เองทั้งหมด แต่เป็นสิ่งที่แอปพลิเคชัน + constraint ของ database ร่วมกันรับประกัน
- **Isolation** — transaction ที่รันพร้อมกันหลายตัว ต้องไม่เห็นสถานะกลางๆ ที่ยังไม่ commit ของกันและกัน (ระดับความเข้มงวดปรับได้ตาม isolation level — หัวข้อ 3.2)
- **Durability** — เมื่อ `COMMIT` สำเร็จแล้ว ข้อมูลต้องไม่หายแม้ระบบจะ crash ทันทีหลังจากนั้น PostgreSQL ทำผ่าน **WAL (Write-Ahead Log)** — เขียน log การเปลี่ยนแปลงลง disk ก่อนที่จะตอบว่า commit สำเร็จ ถ้า crash ก่อน data file จะถูกเขียนจริง ตอน restart PostgreSQL จะ "replay" WAL เพื่อนำข้อมูลกลับมาให้ตรงกับสถานะล่าสุดที่ commit แล้ว

## 3.2 Isolation Level

Isolation level คือการปรับ "ความเข้มงวด" ของ isolation แลกกับ concurrency/performance — ยิ่งเข้มงวดมาก ยิ่งป้องกัน anomaly ได้มากขึ้น แต่ก็ยิ่งลด concurrency ลง (block กันมากขึ้น หรือต้อง retry บ่อยขึ้น) SQL standard นิยาม anomaly 3 แบบที่ isolation level แต่ละระดับป้องกันต่างกัน:

- **Dirty Read** — เห็นข้อมูลที่ transaction อื่น **ยังไม่ commit** (ถ้า transaction นั้น rollback ทีหลัง ข้อมูลที่เราเห็นไปแล้วก็ผิดตั้งแต่แรก)
- **Non-repeatable Read** — อ่านแถวเดียวกันสองครั้งใน transaction เดียวกัน ได้ค่าไม่เหมือนกัน เพราะ transaction อื่น `UPDATE` แล้ว commit คั่นกลาง
- **Phantom Read** — รัน query แบบมีเงื่อนไข (เช่น `WHERE amount > 100`) สองครั้งใน transaction เดียวกัน ได้จำนวนแถวไม่เท่ากัน เพราะ transaction อื่น `INSERT`/`DELETE` แถวใหม่ที่เข้าเงื่อนไขคั่นกลาง

| Isolation Level | Dirty Read | Non-repeatable Read | Phantom Read |
|---|---|---|---|
| Read Uncommitted | อาจเกิด | อาจเกิด | อาจเกิด |
| Read Committed (default ของ PostgreSQL) | ป้องกันได้ | อาจเกิด | อาจเกิด |
| Repeatable Read | ป้องกันได้ | ป้องกันได้ | ป้องกันได้ (ใน PostgreSQL — ดูหมายเหตุด้านล่าง) |
| Serializable | ป้องกันได้ | ป้องกันได้ | ป้องกันได้ |

> **หมายเหตุสำคัญ**: PostgreSQL **ไม่มี** Read Uncommitted จริงๆ — ถ้าตั้งค่านี้ PostgreSQL จะทำงานเหมือน Read Committed เพราะสถาปัตยกรรม MVCC ของ PostgreSQL (หัวข้อ 3.4) ทำให้ dirty read เกิดไม่ได้อยู่แล้วโดยธรรมชาติ ไม่ว่าจะตั้ง isolation level เป็นอะไร นี่คือตัวอย่างที่ชัดว่า isolation level เป็นแค่ "ข้อกำหนดพฤติกรรมขั้นต่ำ" ตาม SQL standard ส่วนวิธี implement จริงเป็นเรื่องของแต่ละ database

### ตัวอย่างจับต้องได้: Non-repeatable Read ใน Read Committed

```sql
-- Transaction A                          -- Transaction B
BEGIN;
SELECT amount FROM orders WHERE id = 101;
-- ได้ 500.00
                                           BEGIN;
                                           UPDATE orders SET amount = 600.00 WHERE id = 101;
                                           COMMIT;
SELECT amount FROM orders WHERE id = 101;
-- ได้ 600.00 !! (ค่าต่างจากครั้งแรกใน transaction เดียวกัน)
COMMIT;
```

ที่ Read Committed แต่ละคำสั่ง `SELECT` เห็น snapshot ข้อมูล ณ**เวลาที่คำสั่งนั้นเริ่มรัน** ไม่ใช่ ณ เวลาที่ transaction เริ่ม — ถ้าต้องการให้ทั้ง transaction เห็นข้อมูลคงที่ตลอด (snapshot เดียวตั้งแต่ต้นจนจบ) ต้องใช้ **Repeatable Read** ขึ้นไป ซึ่งที่ระดับนี้ transaction A จะยังเห็น `500.00` ตลอดแม้ B จะ commit ไปแล้วก็ตาม (แต่ transaction A จะ `UPDATE` แถวเดียวกันไม่ได้ถ้า B แก้ไปแล้ว — จะได้ serialization error แทน)

### Serializable

ระดับเข้มงวดสุด — รับประกันว่าผลลัพธ์ของการรัน transaction พร้อมกันหลายตัว จะเหมือนกับผลลัพธ์ที่ได้ถ้ารันทีละตัวเรียงลำดับกัน (แม้ในการทำงานจริงจะรันพร้อมกันเพื่อ performance ก็ตาม) PostgreSQL implement ด้วย **SSI (Serializable Snapshot Isolation)** ซึ่งตรวจจับ pattern ที่อาจทำให้ผลลัพธ์ผิดจาก serial order แล้วบังคับให้ transaction หนึ่งล้มเหลวด้วย serialization error (ต้อง retry ที่ฝั่งแอปพลิเคชัน) แทนที่จะ lock ข้อมูลตลอดเวลาแบบ pessimistic — เหมาะกับ operation ที่ correctness สำคัญที่สุด (เช่น ระบบจองที่นั่ง, ธุรกรรมการเงินที่ซับซ้อน) แต่ต้องเขียนแอปพลิเคชันให้ retry เมื่อเจอ serialization error เสมอ

## 3.3 Locking

### Row-level Lock vs Table-level Lock

- **Row-level lock** — ล็อกเฉพาะแถวที่กำลังถูกแก้ไข (เช่นจาก `UPDATE`, `DELETE`, หรือ `SELECT ... FOR UPDATE`) ไม่กระทบแถวอื่นในตารางเดียวกัน เป็น lock ที่ใช้บ่อยที่สุดในงาน OLTP ทั่วไป
- **Table-level lock** — ล็อกทั้งตาราง เกิดจากคำสั่งเช่น `LOCK TABLE`, หรือบาง DDL (`ALTER TABLE` บางรูปแบบ) PostgreSQL มี lock mode หลายระดับตั้งแต่เบาสุด (`ACCESS SHARE` — แค่ `SELECT` ธรรมดา) ไปจนหนักสุด (`ACCESS EXCLUSIVE` — บล็อกทุกอย่างแม้แต่ `SELECT`) ยิ่ง DDL ที่ต้อง `ACCESS EXCLUSIVE` lock บนตารางที่ traffic สูง ยิ่งต้องระวังเพราะจะบล็อก query ทุกตัวจนกว่า DDL จะเสร็จ

```sql
-- SELECT ... FOR UPDATE: ล็อกแถวที่ query ได้ ป้องกันไม่ให้ transaction อื่นแก้ไข/ล็อกซ้ำ
-- จนกว่า transaction ปัจจุบันจะ COMMIT/ROLLBACK
BEGIN;
SELECT * FROM accounts WHERE id = 1 FOR UPDATE;
UPDATE accounts SET balance = balance - 100 WHERE id = 1;
COMMIT;
```

### Deadlock

เกิดเมื่อ transaction สองตัว (หรือมากกว่า) ล็อก resource คนละตัว แล้วต่างฝ่ายต่างรออีกฝ่ายปล่อย resource ที่ตัวเองต้องการ — วนรอกันไม่มีวันจบถ้าไม่มีใครมาแทรกแซง

```sql
-- Transaction A                          -- Transaction B
BEGIN;                                    BEGIN;
UPDATE accounts SET balance = balance - 10
  WHERE id = 1;                           UPDATE accounts SET balance = balance - 10
                                             WHERE id = 2;
UPDATE accounts SET balance = balance + 10
  WHERE id = 2;  -- รอ B ปล่อย lock id=2      UPDATE accounts SET balance = balance + 10
                                             WHERE id = 1;  -- รอ A ปล่อย lock id=1
                       -- ติดตายทั้งคู่ (deadlock)
```

```
Transaction A ──ต้องการ lock id=2──► ถูก B ล็อกอยู่
     ▲                                        │
     │                                        ▼
   ล็อก id=1 อยู่ ◄──ต้องการ lock id=1── Transaction B
```

PostgreSQL มี **deadlock detector** ทำงานเป็นระยะ (ค่า default ทุก 1 วินาที ตาม `deadlock_timeout`) เมื่อตรวจพบ deadlock จริง จะเลือก transaction หนึ่ง (มักเป็นตัวที่ทำให้เกิด cycle ล่าสุด) มา **abort** ให้อัตโนมัติ พร้อม error message แล้วปล่อยให้อีกฝ่ายทำงานต่อได้ — แอปพลิเคชันต้องจับ error นี้แล้ว retry เอง

### Debug Deadlock/Lock Contention ด้วย pg_locks

```sql
SELECT
    pl.pid,
    pl.locktype,
    pl.relation::regclass AS table_name,
    pl.mode,
    pl.granted,
    psa.query,
    psa.state
FROM pg_locks pl
JOIN pg_stat_activity psa ON pl.pid = psa.pid
WHERE NOT pl.granted;   -- แสดงเฉพาะ lock ที่ "รออยู่" ยังไม่ได้ granted
```

`pg_locks` แสดง lock ทั้งหมดในระบบ ณ ขณะนี้ — คอลัมน์ `granted = false` คือกำลังรอ ส่วน `granted = true` คือถืออยู่แล้ว การ join กับ `pg_stat_activity` ทำให้เห็นว่า process ไหน (`pid`) กำลังรัน query อะไรอยู่ และกำลังรอ/ถือ lock ตัวไหน — ใช้หา process ที่บล็อกกันอยู่ได้ตรงจุด (หรือใช้ view สำเร็จรูป `pg_blocking_pids(pid)` เพื่อดูว่า pid ไหนกำลังบล็อก pid ที่รออยู่)

## 3.4 MVCC (Multi-Version Concurrency Control)

นี่คือกลไกที่ทำให้ทุกอย่างในหัวข้อ 3.2-3.3 เกิดขึ้นได้จริง PostgreSQL **ไม่ได้ใช้การล็อกเพื่อป้องกัน dirty read** (ต่างจาก database บางตัวที่ใช้ read lock) แต่ใช้แนวคิด **หลายเวอร์ชันของแถวเดียวกันอยู่ร่วมกันได้พร้อมกัน**

### หลักการ

ทุกแถวใน PostgreSQL มี metadata ซ่อนอยู่ 2 คอลัมน์สำคัญ: `xmin` (transaction ID ที่สร้างแถวนี้) และ `xmax` (transaction ID ที่ลบ/แทนที่แถวนี้ ถ้ายังไม่ถูกลบจะเป็น 0)

เมื่อ `UPDATE` เกิดขึ้น PostgreSQL **ไม่ได้แก้ไขแถวเดิม in-place** แต่:

1. สร้างแถวใหม่ (เวอร์ชันใหม่) พร้อม `xmin` = transaction ID ปัจจุบัน
2. ตั้ง `xmax` ของแถวเก่าให้เท่ากับ transaction ID ปัจจุบัน (หมายความว่า "แถวนี้ถูกแทนที่โดย transaction นี้")
3. แถวเก่ากลายเป็น **dead tuple** — ยังอยู่บน disk จริง แต่ไม่มี transaction ไหนควรเห็นมันอีกต่อไป (ยกเว้น transaction ที่เริ่มก่อน `xmax` นี้)

```
UPDATE orders SET amount = 600 WHERE id = 101;

ก่อน:  [id=101, amount=500, xmin=100, xmax=0]        ← แถวเดิม active

หลัง:  [id=101, amount=500, xmin=100, xmax=205]      ← dead tuple (ถูก "ปิด" โดย txn 205)
       [id=101, amount=600, xmin=205, xmax=0]        ← แถวใหม่ active
```

### แต่ละ Transaction เห็น Snapshot ของตัวเอง

เมื่อ transaction เริ่มทำงาน (หรือเริ่มแต่ละ statement ที่ Read Committed) มันจะได้ **snapshot** — รายการ transaction ID ที่ "ถือว่า commit แล้ว ณ จุดนั้น" เวลาอ่านแถวใดๆ PostgreSQL เช็คว่า `xmin`/`xmax` ของแถวนั้น "มองเห็นได้" จาก snapshot นี้หรือไม่:

- ถ้า `xmin` ของแถวมาจาก transaction ที่ยังไม่ commit (หรือ commit หลัง snapshot ถูกสร้าง) → **มองไม่เห็นแถวนี้** (นี่คือเหตุผลที่ dirty read เกิดไม่ได้เลยโดยธรรมชาติ — transaction อื่นไม่มีทางเห็นแถวที่ยังไม่ commit)
- ถ้า `xmax` ของแถวมาจาก transaction ที่ commit ไปแล้วก่อน snapshot ถูกสร้าง → แถวนี้ถูกลบ/แทนที่ไปแล้วในมุมมองของ snapshot นี้ → มองไม่เห็นเช่นกัน

นี่คือคำตอบว่าทำไม **Read Committed** สร้าง snapshot ใหม่ทุกคำสั่ง (เห็นข้อมูลล่าสุดที่ commit แล้วเสมอ → เกิด non-repeatable read ได้) ในขณะที่ **Repeatable Read**/**Serializable** สร้าง snapshot เดียวตอนเริ่ม transaction แล้วใช้ตลอด (เห็นข้อมูลคงที่ทั้ง transaction) — isolation level ที่ต่างกันในทางปฏิบัติคือ **"สร้าง snapshot บ่อยแค่ไหน"** นั่นเอง

### ข้อดีของแนวทางนี้

Reader ไม่ต้องรอ writer และ writer ไม่ต้องรอ reader (ต่างจาก database ที่ใช้ read lock แบบดั้งเดิม) เพราะ reader อ่านจากเวอร์ชันเก่าที่ยัง valid อยู่ ในขณะที่ writer กำลังสร้างเวอร์ชันใหม่คู่ขนานกันไป — นี่คือเหตุผลหลักที่ PostgreSQL รองรับ concurrent read/write ได้ดีโดยไม่ต้องพึ่ง lock หนักๆ ในกรณีทั่วไป

## 3.5 Dead Tuple, VACUUM, Autovacuum และ Table Bloat

### ปัญหาที่ตามมา: Dead Tuple สะสม

เพราะ `UPDATE`/`DELETE` ไม่ได้ลบข้อมูลเก่าออกจาก disk ทันที (แค่ทำเครื่องหมาย `xmax`) พื้นที่ที่ dead tuple เหล่านี้ใช้จะไม่ถูกคืนกลับให้ระบบใช้ใหม่โดยอัตโนมัติ ถ้าตารางถูก `UPDATE` บ่อยมาก dead tuple จะสะสมเรื่อยๆ ทำให้ตารางใหญ่ขึ้นทั้งที่จำนวนแถว "จริง" ไม่ได้เพิ่ม — ปรากฏการณ์นี้เรียกว่า **table bloat**

ผลกระทบของ table bloat:

- ตารางบนดิสก์ใหญ่ขึ้นโดยไม่จำเป็น กิน storage เพิ่ม
- Sequential Scan ต้องอ่านข้อมูลมากขึ้น (รวม dead tuple ที่ไม่มีประโยชน์) ช้าลง
- Index ก็บวมตามไปด้วย เพราะ entry ของ dead tuple ยังอยู่ใน index จนกว่าจะถูกทำความสะอาด

### VACUUM

`VACUUM` คือคำสั่งที่กวาด dead tuple ที่**ไม่มี transaction ไหนมองเห็นได้อีกแล้ว** (ไม่มี snapshot เก่าที่ยังต้องใช้มันอยู่) ออกจาก page แล้วทำเครื่องหมายพื้นที่นั้นว่า "ใช้ซ้ำได้" สำหรับแถวใหม่ในอนาคต — **ไม่ได้คืนพื้นที่ให้ OS** (ตารางบน disk จะไม่เล็กลง) เพียงแค่ทำให้ PostgreSQL เอาพื้นที่ว่างนั้นกลับมาใช้ใหม่ได้ ถ้าต้องการคืนพื้นที่ให้ OS จริงๆ ต้องใช้ `VACUUM FULL` (ล็อกตารางทั้งตารางระหว่างทำงาน ไม่ควรรันบน production table ที่ traffic สูงโดยไม่วางแผน) หรือ `pg_repack` (extension ที่ทำแบบไม่ต้องล็อกตารางนาน)

```sql
VACUUM orders;              -- กวาด dead tuple ธรรมดา ไม่ล็อกตาราง
VACUUM ANALYZE orders;      -- กวาดพร้อมอัปเดต statistics ให้ query planner ด้วย (แนะนำ)
VACUUM FULL orders;         -- คืนพื้นที่ให้ OS จริง แต่ล็อกตารางทั้งตาราง (ระวัง)
```

### Autovacuum

รัน `VACUUM` มือเองตลอดเวลาไม่ scale ได้จริง PostgreSQL จึงมี **autovacuum** — background process ที่คอยตรวจตารางแต่ละตัว แล้วรัน `VACUUM`/`ANALYZE` อัตโนมัติเมื่อสัดส่วนของ dead tuple เกิน threshold ที่กำหนด (ค่า default ประมาณ 20% ของขนาดตาราง บวกจำนวนคงที่เล็กน้อย ปรับได้ผ่าน `autovacuum_vacuum_scale_factor`)

ปัญหาที่พบบ่อยในทางปฏิบัติ:

- **Long-running transaction** — ถ้ามี transaction ที่เปิดค้างไว้นาน (เช่น debug session ที่ลืม commit หรือ query ที่ค้างนานผิดปกติ) `VACUUM` จะกวาด dead tuple ที่ transaction นั้นยังอาจต้องเห็นไม่ได้เลย เพราะต้องคงไว้จนกว่า snapshot เก่าที่สุดที่ยังทำงานอยู่จะจบ — นี่คือเหตุผลที่ transaction ค้างนานๆ เป็นสาเหตุอันดับต้นๆ ของ table bloat ในโปรดักชันจริง
- **Autovacuum ตามไม่ทัน** — ตารางที่ update/delete หนักมากในช่วงเวลาสั้นๆ (burst) อาจทำให้ autovacuum วิ่งตามไม่ทัน ต้องปรับ config ให้ aggressive ขึ้น (เช่นลด `autovacuum_vacuum_scale_factor`, เพิ่ม `autovacuum_max_workers`) หรือปรับ per-table setting เฉพาะตารางที่ hot

## 3.6 Transaction ID Wraparound

`xmin`/`xmax` ที่กล่าวถึงในหัวข้อ 3.4 เป็นตัวเลข **32-bit** ซึ่งมีขอบเขตจำกัด (ประมาณ 4 พันล้านค่า) PostgreSQL เปรียบเทียบ transaction ID แบบ "circular" (คล้ายเข็มนาฬิกา) เพื่อตัดสินว่า transaction ไหน "เกิดก่อน" — ถ้า transaction ID วิ่งวนครบรอบโดยไม่มีการจัดการ transaction เก่าที่เคย "อยู่ในอนาคต" ของเข็มนาฬิกา อาจถูกตีความว่า "อยู่ในอดีต" กลับกลายเป็นมองไม่เห็นข้อมูลที่เคย commit ไปแล้วอย่างถูกต้อง — เป็นสถานการณ์ **data loss ที่ร้ายแรงมาก** ถ้าเกิดขึ้นจริง

PostgreSQL ป้องกันด้วยกลไก **freezing**: เมื่อ `VACUUM` เจอแถวเก่าที่ transaction ID ของมันเก่าพอ (ผ่าน threshold ที่กำหนดโดย `vacuum_freeze_min_age`) จะเขียนทับ `xmin` ของแถวนั้นด้วยค่าพิเศษ `FrozenTransactionId` ที่ถือว่า "เกิดก่อนทุก transaction เสมอ" ทำให้แถวนั้นไม่ต้องพึ่งการเปรียบเทียบ transaction ID ตามปกติอีกต่อไป และไม่มีวันถูกตีความผิดจาก wraparound

ถ้า autovacuum ถูกปิดหรือตามไม่ทันเป็นเวลานานมากจนใกล้ถึงจุดวิกฤต PostgreSQL จะเริ่มเตือนใน log (`WARNING: database "X" must be vacuumed within N transactions`) และถ้าปล่อยจนถึงขีดสุดจริงๆ ระบบจะ**ปฏิเสธการรับ transaction ใหม่ทั้งหมด** (เข้าสู่โหมด read-only บังคับ) เพื่อป้องกัน data corruption จนกว่าจะรัน `VACUUM` เพื่อแก้ปัญหาได้สำเร็จ — เป็นเหตุการณ์ที่ร้ายแรงระดับ production incident เต็มรูปแบบ จึงต้อง monitor `age(datfrozenxid)` ของแต่ละ database เป็นประจำ ไม่ปล่อยให้ autovacuum ทำงานผิดปกติเป็นเวลานานโดยไม่รู้ตัว

## เชื่อมทุกอย่างเข้าด้วยกัน

```
Isolation Level ที่เลือก
        │
        ▼
กำหนดว่า "สร้าง snapshot ใหม่บ่อยแค่ไหน" (ทุก statement vs ทุก transaction)
        │
        ▼
MVCC ใช้ snapshot ตัดสินว่าแถวเวอร์ชันไหน "มองเห็นได้"
        │
        ▼
UPDATE/DELETE สร้างเวอร์ชันใหม่ ทิ้งเวอร์ชันเก่าไว้เป็น dead tuple
        │
        ▼
VACUUM/autovacuum กวาด dead tuple ที่ไม่มี snapshot ไหนต้องใช้แล้ว
        │
        ▼
Freeze ป้องกัน transaction ID wraparound ในระยะยาว
```

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| ACID | Atomicity/Consistency/Isolation/Durability — Durability ทำผ่าน WAL |
| Isolation level | ยิ่งเข้มงวด ยิ่งป้องกัน anomaly ได้มากขึ้นแต่ concurrency ลดลง; PostgreSQL ไม่มี Read Uncommitted จริง |
| Locking | row-level ใช้บ่อยสุด (`FOR UPDATE`), deadlock ตรวจจับอัตโนมัติแล้ว abort ฝั่งหนึ่ง, debug ด้วย `pg_locks` |
| MVCC | UPDATE ไม่แก้ในที่ แต่สร้างเวอร์ชันใหม่ (`xmin`/`xmax`) แต่ละ transaction เห็น snapshot ตามเวลาที่กำหนด |
| Dead tuple / VACUUM | เวอร์ชันเก่าค้างบน disk จนกว่าจะถูกกวาด, table bloat เกิดถ้ากวาดไม่ทัน |
| Autovacuum | อัตโนมัติแต่ระวัง long-running transaction และ burst write ที่ตามไม่ทัน |
| Transaction ID wraparound | ป้องกันด้วย freezing — ถ้าปล่อยถึงขีดสุดจริงระบบจะปฏิเสธ transaction ใหม่ทั้งหมด |

## คำถามทบทวน

1. อธิบายว่าทำไม PostgreSQL ถึงไม่มี "Read Uncommitted" จริงๆ ทั้งที่ตั้งค่านั้นได้
2. Non-repeatable read เกิดขึ้นได้อย่างไรที่ระดับ Read Committed ยกตัวอย่างประกอบด้วยคำพูดตัวเอง
3. อธิบายความสัมพันธ์ระหว่าง isolation level กับความถี่ในการสร้าง MVCC snapshot
4. Table bloat เกิดขึ้นได้อย่างไร และทำไม long-running transaction ถึงเป็นสาเหตุสำคัญของปัญหานี้
5. Transaction ID wraparound คืออะไร และ freezing ป้องกันปัญหานี้อย่างไร

---

ก่อนหน้า: [บทที่ 2 — Index](02-indexing.md) | ถัดไป: [บทที่ 4 — JSONB](04-jsonb.md)
