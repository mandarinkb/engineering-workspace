# บทที่ 4: JSONB

PostgreSQL รองรับข้อมูลแบบ semi-structured ผ่านชนิดข้อมูล `JSON` และ `JSONB` ทำให้ RDBMS ที่เข้มงวดเรื่อง schema สามารถเก็บข้อมูลที่มีโครงสร้างไม่แน่นอนได้โดยไม่ต้องเปลี่ยนไปใช้ NoSQL แต่ความสะดวกนี้มาพร้อมกับดาบสองคม — บทนี้จะอธิบายความต่างของ `JSON`/`JSONB`, operator ที่ใช้บ่อย, การทำ index และที่สำคัญที่สุดคือ**เมื่อไหร่ควรใช้ JSONB และเมื่อไหร่ไม่ควร**

## 4.1 JSON vs JSONB

ทั้งสองชนิดเก็บข้อมูลแบบ JSON ได้เหมือนกัน แต่รูปแบบการจัดเก็บภายในต่างกันโดยสิ้นเชิง:

| | `JSON` | `JSONB` |
|---|---|---|
| รูปแบบเก็บ | เก็บเป็น **text ดิบ** ตามที่ input เข้ามา | เก็บเป็น **binary รูปแบบ decomposed** (แปลงโครงสร้างไว้ล่วงหน้า) |
| ลำดับ key | คงลำดับเดิมตามที่ใส่เข้ามา รวม whitespace/duplicate key | ไม่คงลำดับ key เดิม (จัดเรียงใหม่ภายใน) และตัด duplicate key ทิ้ง (เก็บค่าสุดท้าย) |
| ความเร็วในการเขียน (`INSERT`) | เร็วกว่า (แค่ validate syntax แล้วเก็บ text) | ช้ากว่าเล็กน้อย (ต้อง parse + แปลงเป็น binary ทุกครั้ง) |
| ความเร็วในการอ่าน/query | ช้ากว่า (ต้อง parse text ใหม่ทุกครั้งที่ query) | เร็วกว่ามาก (ไม่ต้อง parse ซ้ำ เพราะเก็บเป็น structure พร้อมใช้แล้ว) |
| Indexing | ทำ GIN index โดยตรงไม่ได้ | ทำ GIN index ได้ (หัวข้อ 4.3) |
| Operator สำหรับ query (`->`, `@>`, ฯลฯ) | ใช้ได้บางส่วน | ใช้ได้เต็มรูปแบบ |

**สรุปเชิงปฏิบัติ**: ใช้ `JSONB` แทบทุกกรณี ยกเว้นกรณีพิเศษจริงๆ ที่ต้องเก็บ JSON ดิบตามที่ client ส่งมาแบบเป๊ะๆ (เช่น เก็บ log request/response ต้นฉบับเพื่อ audit โดยไม่สนเรื่อง query ทีหลัง) — งาน backend ทั่วไปที่ต้อง query/filter ข้อมูลใน JSON บ่อยๆ `JSONB` เหมาะกว่าเสมอ

## 4.2 Operator ที่ใช้บ่อย

```sql
CREATE TABLE events (
    id      SERIAL PRIMARY KEY,
    payload JSONB NOT NULL
);

INSERT INTO events (payload) VALUES
    ('{"type": "login", "user": {"id": 42, "name": "Somchai"}, "tags": ["mobile", "beta"]}'),
    ('{"type": "logout", "user": {"id": 42, "name": "Somchai"}, "tags": ["web"]}');
```

### `->` และ `->>`

```sql
SELECT payload -> 'type' FROM events;
-- คืนค่าเป็น JSONB: "login", "logout"

SELECT payload ->> 'type' FROM events;
-- คืนค่าเป็น TEXT: login, logout (ไม่มี double quote ครอบ)

SELECT payload -> 'user' ->> 'name' FROM events;
-- เข้าถึงซ้อนกันหลายชั้น: -> ครั้งแรกได้ JSONB ({"id": 42, "name": "Somchai"})
-- ->> ครั้งสุดท้ายแปลงเป็น TEXT (Somchai)
```

กฎจำง่ายๆ: **`->` คืน JSONB (เข้าถึงต่อได้)**, **`->>` คืน TEXT (จบการเข้าถึง แปลงเป็น string)** — ถ้าจะเข้าถึงหลายชั้นแล้วเปรียบเทียบค่าสุดท้ายเป็นข้อความ ต้องใช้ `->` ทุกชั้นยกเว้นชั้นสุดท้ายที่ใช้ `->>`

### `@>` (Contains)

เช็คว่า JSONB ฝั่งซ้าย "ครอบคลุม" โครงสร้างฝั่งขวาหรือไม่:

```sql
SELECT * FROM events WHERE payload @> '{"type": "login"}';
-- ได้แถวที่ payload มี key "type" ค่า "login" (ไม่สนว่าจะมี key อื่นเพิ่มอีกกี่ตัว)

SELECT * FROM events WHERE payload @> '{"user": {"id": 42}}';
-- เช็ค nested object ได้เช่นกัน
```

### `?` (Key Exists)

เช็คว่ามี key ระดับบนสุดชื่อนี้อยู่หรือไม่:

```sql
SELECT * FROM events WHERE payload ? 'tags';
-- ได้แถวที่มี key "tags" อยู่ที่ระดับบนสุด (ไม่สนใจค่าข้างใน)

SELECT * FROM events WHERE payload -> 'tags' ? 'beta';
-- เช็คว่า array "tags" มีสมาชิก "beta" อยู่หรือไม่
```

มี variant เพิ่มเติม: `?|` (มี key ใดๆ ในลิสต์ที่ให้มา) และ `?&` (ต้องมี**ทุก**key ในลิสต์ที่ให้มา)

## 4.3 GIN Indexing บน JSONB

Query ที่กรองด้วย `@>` หรือ `?` บ่อยๆ บนตารางใหญ่ ควรมี GIN index รองรับ (เชื่อมกับแนวคิด GIN ใน [บทที่ 2](02-indexing.md)):

```sql
-- Default GIN operator class: รองรับ @>, ?, ?|, ?& ครบทุก key/value ในเอกสาร
CREATE INDEX idx_events_payload ON events USING GIN(payload);

-- jsonb_path_ops: index เล็กกว่า เร็วกว่าสำหรับ @> โดยเฉพาะ แต่ไม่รองรับ ?, ?|, ?&
CREATE INDEX idx_events_payload_path ON events USING GIN(payload jsonb_path_ops);
```

ถ้ารู้ล่วงหน้าว่า query ทั้งหมดใช้แค่ `@>` (containment) ไม่เคยใช้ `?` เลย `jsonb_path_ops` ให้ index ที่เล็กกว่าและเร็วกว่า default operator class อย่างชัดเจน — ตัดสินใจจาก query pattern จริงเหมือนกับการเลือก index type อื่นๆ

ถ้าต้องการ query ที่กรองบน key ใดตัวหนึ่งเป็นประจำ (เช่น `payload ->> 'type' = 'login'` บ่อยมาก) การสร้าง **expression index** บน B-tree ตรงๆ อาจเหมาะกว่า GIN ทั้งเอกสาร:

```sql
CREATE INDEX idx_events_type ON events ((payload ->> 'type'));
```

## 4.4 เมื่อไหร่ควรใช้ JSONB vs เมื่อไหร่คือ Anti-pattern

### กรณีที่ JSONB เหมาะสม

- **โครงสร้างข้อมูลไม่แน่นอนจริงๆ และไม่รู้ล่วงหน้า** เช่น เก็บ event payload จากหลาย third-party ที่แต่ละเจ้าส่ง field มาไม่เหมือนกัน (webhook จาก payment provider หลายเจ้า) การ normalize ทุก field ที่เป็นไปได้ล่วงหน้าไม่คุ้มและเป็นไปไม่ได้จริง
- **ข้อมูล metadata/config ที่ query แบบ exact structure ไม่บ่อย** เช่น เก็บ user preference/settings ที่ product เปลี่ยน field บ่อยมาก การเพิ่ม column ใหม่ทุกครั้งที่ product ต้องการ setting ใหม่ทำให้ migration รันบ่อยเกินจำเป็น
- **Audit log / raw request-response** ที่ต้องการเก็บข้อมูลต้นฉบับให้ครบ ไม่สนใจ query ซับซ้อนบนมัน แค่ต้องการ retrieve กลับมาดูทั้งก้อนเป็นหลัก

### กรณีที่ JSONB เป็น Anti-pattern (ควร Normalize แทน)

**ตัวอย่างที่ชัดเจน**: ระบบ e-commerce เก็บรายการสินค้าใน order เป็น JSONB:

```sql
-- Anti-pattern
CREATE TABLE orders (
    id    SERIAL PRIMARY KEY,
    items JSONB  -- [{"product_id": 1, "qty": 2, "price": 100}, {"product_id": 5, "qty": 1, "price": 300}]
);
```

ดูเหมือนสะดวกตอนเขียน แต่พอระบบโตขึ้น จะเจอปัญหาเหล่านี้:

- อยากรู้ "สินค้าตัวไหนขายดีที่สุด" ต้อง `jsonb_array_elements` แกะทุก order ทุกแถวมา aggregate — ช้ากว่าและซับซ้อนกว่า `GROUP BY product_id` บนตารางปกติมาก
- ไม่มี foreign key constraint บังคับว่า `product_id` ที่อ้างถึงต้องมีอยู่จริงในตาราง `products` — ข้อมูลเพี้ยนได้ง่ายโดยไม่มี database ช่วยป้องกัน
- อัปเดตราคาสินค้าเฉพาะรายการเดียวในหนึ่ง order ต้องเขียนทับทั้ง JSONB array ใหม่ ทำไม่ได้แบบ atomic ในระดับ field เหมือน `UPDATE order_items SET price = ... WHERE id = ...`
- Reporting/analytics ที่ต้อง join ข้าม order หลายพันรายการ ทำผ่าน JSONB ได้ช้ากว่า normalized table ที่มี index รองรับตรงๆ มาก

**วิธีที่ควรทำแทน**: normalize เป็นตารางลูก

```sql
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    customer_id INT REFERENCES customers(id)
);

CREATE TABLE order_items (
    id         SERIAL PRIMARY KEY,
    order_id   INT REFERENCES orders(id),
    product_id INT REFERENCES products(id),
    qty        INT NOT NULL,
    price      NUMERIC(10,2) NOT NULL
);
```

ได้ foreign key integrity, index ตรงๆ, `GROUP BY`/`JOIN` เร็วและอ่านง่ายกว่า — สัญญาณเตือนว่ากำลังใช้ JSONB ผิดที่คือ **เมื่อเริ่ม query ข้างในโครงสร้าง JSONB บ่อยๆ ด้วย `jsonb_array_elements`/`jsonb_each` เพื่อทำ aggregate/join แบบที่ SQL ปกติทำได้ดีอยู่แล้ว** นั่นคือสัญญาณว่าโครงสร้างข้อมูลนี้ "รู้ล่วงหน้าแล้ว" และควรเป็น relational schema ตั้งแต่แรก ไม่ใช่ JSONB

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| JSON vs JSONB | JSON เก็บ text ดิบ เขียนเร็วอ่านช้า; JSONB เก็บ binary decomposed เขียนช้ากว่าเล็กน้อยแต่อ่าน/query เร็วกว่ามาก ทำ index ได้ |
| `->` vs `->>` | `->` คืน JSONB (เข้าถึงต่อได้), `->>` คืน TEXT (จบการเข้าถึง) |
| `@>` / `?` | `@>` เช็ค containment ของโครงสร้าง, `?` เช็คว่ามี key ระดับบนสุด |
| GIN index | รองรับ query แบบ `@>`/`?`; `jsonb_path_ops` เล็กกว่าเร็วกว่าถ้าใช้แค่ `@>` |
| เมื่อไหร่ใช้ JSONB | โครงสร้างไม่แน่นอนจริง, metadata/config ที่เปลี่ยนบ่อย, audit log ดิบ |
| เมื่อไหร่ควร normalize แทน | โครงสร้างรู้ล่วงหน้าแน่นอน ต้อง query/join/aggregate ข้าม element บ่อย ต้องการ referential integrity |

## คำถามทบทวน

1. ทำไม `JSONB` ถึง query เร็วกว่า `JSON` ทั้งที่ `JSON` เขียนได้เร็วกว่า?
2. อธิบายความต่างระหว่าง `payload -> 'user'` กับ `payload ->> 'user'` ด้วยตัวอย่าง
3. `jsonb_path_ops` ต่างจาก default GIN operator class อย่างไร ควรเลือกใช้เมื่อไหร่?
4. ยกตัวอย่างสถานการณ์ (ที่ไม่ใช่ตัวอย่างในบทเรียน) ที่ใช้ JSONB เก็บข้อมูลแล้วภายหลังพบว่าควร normalize เป็นตารางแทน อธิบายสัญญาณที่บ่งบอกว่าถึงเวลาต้อง normalize
5. เพราะเหตุใด JSONB ถึงไม่มี foreign key constraint บังคับความถูกต้องของข้อมูลข้างในได้เหมือนตารางปกติ

---

ก่อนหน้า: [บทที่ 3 — Transaction และ MVCC](03-transactions-and-mvcc.md) | ถัดไป: [บทที่ 5 — Partitioning และ Replication](05-partitioning-and-replication.md)
