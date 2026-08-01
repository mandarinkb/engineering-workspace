# Volume 5 — PostgreSQL

ฐานข้อมูลหลักของหลักสูตรนี้ PostgreSQL เป็น RDBMS ที่แข็งแรงเรื่อง consistency และ feature ครบ (JSONB, full-text search, extension) เหมาะกับระบบ enterprise ที่ต้องการความถูกต้องของข้อมูลสูง เช่น payment, order

## เป้าหมายการเรียนรู้

- เขียน SQL query ที่ถูกต้องและมีประสิทธิภาพ อ่าน execution plan ได้
- ออกแบบ index ให้เหมาะกับ query pattern จริง ไม่ใช่ใส่ index มั่ว
- เข้าใจ transaction isolation level และ MVCC ลึกพอจะ debug ปัญหา lock/deadlock ได้
- ออกแบบ partition และ replication สำหรับข้อมูลขนาดใหญ่

## สารบัญ

### [บทที่ 1 — SQL](01-sql-fundamentals.md)

- [ ] DDL/DML/DQL, JOIN ชนิดต่างๆ (INNER, LEFT, RIGHT, FULL, CROSS)
- [ ] Subquery, CTE (Common Table Expression), Window function
- [ ] Aggregate function, `GROUP BY`/`HAVING`

### [บทที่ 2 — Index](02-indexing.md)

- [ ] B-tree index (ค่า default), เมื่อไหร่ index ช่วย เมื่อไหร่ไม่ช่วย (low cardinality, small table)
- [ ] Composite index, ลำดับคอลัมน์มีผลอย่างไร
- [ ] Index ชนิดอื่น: GIN (สำหรับ JSONB/full-text), GiST, BRIN
- [ ] อ่าน `EXPLAIN ANALYZE` เพื่อดูว่า query ใช้ index จริงหรือไม่

### [บทที่ 3 — Transaction และ MVCC](03-transactions-and-mvcc.md)

- [ ] ACID properties
- [ ] Isolation level: Read Uncommitted, Read Committed, Repeatable Read, Serializable
- [ ] Lock: row-level lock, table-level lock, deadlock และวิธี debug
- [ ] Multi-Version Concurrency Control ทำงานอย่างไร (แต่ละ transaction เห็น snapshot ของตัวเอง)
- [ ] Dead tuple, `VACUUM`, `autovacuum`, ปัญหา table bloat
- [ ] Transaction ID wraparound

### [บทที่ 4 — JSONB](04-jsonb.md)

- [ ] JSON vs JSONB (storage format, indexing)
- [ ] Operator: `->`, `->>`, `@>`, `?`
- [ ] เมื่อไหร่ควรใช้ JSONB แทน normalized table (และเมื่อไหร่ไม่ควร)

### [บทที่ 5 — Partitioning และ Replication](05-partitioning-and-replication.md)

- [ ] Range partition, list partition, hash partition
- [ ] Partition pruning ช่วย performance อย่างไร
- [ ] Use case: time-series data, log table ขนาดใหญ่
- [ ] Streaming replication (physical), logical replication
- [ ] Synchronous vs asynchronous replication, ผลกระทบต่อ consistency/latency
- [ ] Failover, read replica สำหรับ scale read

## Lab / Source Code ที่เกี่ยวข้อง

- [source-code/golang/](../../source-code/golang/README.md) — ตัวอย่างการเชื่อมต่อ PostgreSQL จาก Go

## แหล่งอ้างอิงแนะนำ

ดู [resources/](../../resources/README.md)
