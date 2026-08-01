# บทที่ 1: CAP และ Consistency

## 1.1 CAP Theorem คืออะไรจริงๆ

**CAP theorem** (Eric Brewer, formalized โดย Gilbert & Lynch ปี 2002) พูดถึงคุณสมบัติ 3 อย่างของระบบแบบ distributed:

- **Consistency (C)** — ทุก node ที่ตอบ request เห็นข้อมูลชุดล่าสุดตรงกัน (จริงๆ คือ **linearizability**: อ่านครั้งไหนก็ตามต้องได้ค่าที่เขียนล่าสุดเสมอ ไม่มี node ไหนตอบข้อมูลเก่ากว่า)
- **Availability (A)** — ทุก request ที่ไปถึง node ที่ยังทำงานอยู่ ต้องได้รับ response กลับเสมอ (ไม่ error, ไม่ timeout) แม้ค่าที่ตอบอาจไม่ใช่ค่าล่าสุดก็ตาม
- **Partition Tolerance (P)** — ระบบยังทำงานต่อไปได้ แม้การสื่อสารระหว่าง node บางกลุ่มขาดหายไป (network partition — สาย network ขาด, switch ล่ม, packet loss รุนแรง)

ข้อความที่ Brewer/Gilbert-Lynch พิสูจน์ไว้แม่นยำกว่าที่คนทั่วไปจำกัน:

> **เมื่อเกิด network partition ขึ้นจริง ระบบ distributed จะรับประกัน Consistency กับ Availability พร้อมกันทั้งคู่ไม่ได้ ต้องเลือกอย่างใดอย่างหนึ่ง**

## 1.2 ความเข้าใจผิดที่พบบ่อย: "เลือกได้ 2 จาก 3 เสมอ"

คนจำนวนมากเข้าใจ CAP ผิดเป็น "ระบบหนึ่งเลือกได้แค่ 2 คุณสมบัติจาก 3 อย่างถาวร ตั้งแต่ตอนออกแบบ" เช่น พูดว่า "เราจะสร้างระบบ CA" — **นี่คือความเข้าใจผิด** เพราะ:

1. CAP ไม่ใช่ข้อความเกี่ยวกับสถาปัตยกรรมแบบตายตัว แต่เป็นข้อความเกี่ยวกับ **พฤติกรรมของระบบ ณ ขณะที่ partition เกิดขึ้นจริง** เท่านั้น
2. ในสภาวะปกติที่ไม่มี partition ระบบสามารถมีทั้ง C และ A พร้อมกันได้สบายๆ — P ยังไม่ถูก "ทดสอบ"
3. ระบบ distributed ที่มีมากกว่า 1 node ผ่าน network จริง **จะเจอ partition แน่นอนสักวันหนึ่ง** (สาย fiber ถูกตัด, switch ล่ม, GC pause นานเกิน timeout จนถูกมองว่า node หาย, region ทั้งโซนขาดการเชื่อมต่อ) ดังนั้น P ไม่ใช่ตัวเลือกที่ "ปฏิเสธได้" ในระบบ multi-node จริง — คำถามที่ engineer ต้องตอบจึงไม่ใช่ "จะเอา P ไหม" แต่คือ **"เมื่อ partition เกิดขึ้น จะเลือก C หรือ A"**

```
สภาวะปกติ (ไม่มี partition):
  Node A ◄──────── sync ────────► Node B
  → ได้ทั้ง Consistency และ Availability พร้อมกัน (P ยังไม่ถูกทดสอบ)

เกิด network partition:
  Node A   ✕────network ขาด────✕   Node B
  ต้องเลือกหนึ่งใน:
    CP → node ที่ไม่แน่ใจว่าข้อมูลตัวเองล่าสุด "ปฏิเสธ" request (ยอมเสีย A เพื่อรักษา C)
    AP → ทุก node ยังตอบ request ต่อไปด้วยข้อมูลที่ตัวเองมี แม้อาจไม่ล่าสุด (ยอมเสีย C เพื่อรักษา A)
```

พูดอีกแบบ: CAP ไม่ใช่ trade-off 3 ทางแบบถาวร แต่เป็น trade-off 2 ทาง (C vs A) ที่**เปิดใช้งานเฉพาะตอนมี partition** — ส่วนเวลาที่ไม่มี partition ระบบก็ยังมี trade-off อื่นอยู่ (latency vs consistency) ซึ่งเป็นสิ่งที่ PACELC พูดถึงในหัวข้อ 1.4

## 1.3 ตัวอย่างระบบจริง

### CP-leaning: PostgreSQL Single-Primary

[PostgreSQL](../05-postgresql/README.md) แบบ single-primary ที่ใช้ **synchronous replication** ไปยัง replica เป็นตัวอย่างของระบบที่เอียงไปทาง **CP**: มี node เดียว (primary) ที่รับ write ได้ ถ้า primary เขียนสำเร็จแต่ยืนยันกับ replica (ตาม `synchronous_commit`) ไม่ได้เพราะ network ขาด ระบบจะ**บล็อกหรือปฏิเสธ transaction นั้น** แทนที่จะยอมให้ commit ไปโดยไม่มั่นใจว่าข้อมูลจะไม่หายหากต้อง failover — ยอมเสีย Availability (client รอหรือได้ error) เพื่อรักษา Consistency ว่าไม่มีทางที่ replica จะเห็นข้อมูลคนละชุดกับ primary

### AP-leaning: DNS

[DNS](../02-network/02-dns.md) เป็นตัวอย่างของระบบที่เอียงไปทาง **AP**: resolver แต่ละตัวเก็บ cache ของตัวเองตาม TTL หาก authoritative server เข้าไม่ถึง (partition) resolver ก็ยังคง**ตอบค่าจาก cache เดิมต่อไป**แทนที่จะปฏิเสธ query — ยอมรับความเสี่ยงที่ค่าที่ตอบอาจไม่ล่าสุด (record เพิ่งถูกเปลี่ยนที่ authoritative server แต่ resolver ยังไม่รู้) เพื่อแลกกับการที่ระบบทั้งโลกยังคง resolve ชื่อโดเมนได้เสมอ นี่คือเหตุผลที่ DNS propagation ใช้เวลา (eventual consistency ผ่าน TTL) แทนที่จะ sync ทันที

```
Partition ระหว่าง client กับ source of truth:

PostgreSQL (CP):            DNS (AP):
  client ──write──► primary    client ──query──► resolver (มี cache)
              │  sync ✕ replica         │
              ▼                          ▼
      ปฏิเสธ / บล็อก request      ตอบค่าจาก cache ทันที (อาจไม่ล่าสุด)
      → รักษา Consistency          → รักษา Availability
```

## 1.4 PACELC — ส่วนขยายของ CAP ที่มองภาพครบกว่า

CAP พูดถึงเฉพาะ**ตอนมี partition** แต่ในความเป็นจริง ระบบใช้เวลาส่วนใหญ่**ไม่มี partition** และก็ยังมี trade-off สำคัญอยู่ดี — Daniel Abadi เสนอกรอบ **PACELC** เพื่อเติมส่วนที่ CAP ไม่ได้พูดถึง:

> **ถ้ามี P (partition) → เลือกระหว่าง A กับ C (เหมือน CAP เดิม)**
> **มิฉะนั้น (Else, ไม่มี partition) → เลือกระหว่าง Latency (L) กับ Consistency (C)**

### ตัวอย่างที่เป็นรูปธรรม: Synchronous vs Asynchronous Replication

พิจารณาระบบที่มี primary database ใน region เอเชีย และ replica ใน region ยุโรป:

- **Synchronous replication** — primary รอ ACK จาก replica ยุโรปก่อนถึงจะตอบ client ว่า commit สำเร็จ รับประกันว่า replica มีข้อมูลตรงกับ primary เสมอ (**C สูง**) แต่ทุก write ต้องรอ round-trip ข้าม region (มักหลักสิบ ms ถึงเป็นร้อย ms) → **latency สูง**
- **Asynchronous replication** — primary commit แล้วตอบ client ทันที ค่อยส่งข้อมูลไปยัง replica ทีหลัง **latency ต่ำ** เพราะไม่ต้องรอข้าม region แต่ถ้า primary ล่มก่อนที่จะส่งข้อมูลถึง replica ทัน ข้อมูลที่เพิ่ง commit อาจ**หายไปตอน failover** (replica กลายเป็น primary ใหม่โดยไม่มีข้อมูลล่าสุด) → **C ต่ำลง**

นี่คือ trade-off ที่เกิดขึ้น**ตลอดเวลาแม้ไม่มี partition เลย** — เป็นการตัดสินใจตอนออกแบบระบบ ไม่ใช่พฤติกรรมฉุกเฉินแบบ CAP

### ตำแหน่งของระบบต่างๆ บนกรอบ PACELC

| ระบบ | เมื่อมี Partition | เมื่อไม่มี Partition | สรุป |
|---|---|---|---|
| PostgreSQL (sync replication) | เลือก C (ปฏิเสธ write) | เลือก C (รอ ACK, latency สูง) | PC/EC |
| PostgreSQL (async replication) | เลือก A (primary รับ write ต่อ) | เลือก L (ไม่รอ replica) | PA/EL |
| DNS | เลือก A (ตอบจาก cache) | เลือก L (ตอบจาก cache local เร็ว) | PA/EL |
| DynamoDB (default) | เลือก A | เลือก L | PA/EL |

## 1.5 CAP/PACELC vs Isolation Level — คนละแกนที่คนมักสับสน

จุดที่วิศวกรจำนวนมากสับสนคือเอา **Consistency ใน CAP** ไปปนกับ **Isolation Level** ของ transaction ใน [Volume 5 — MVCC และ Transaction Isolation](../05-postgresql/03-transactions-and-mvcc.md) ทั้งที่เป็นคนละแกนกัน:

| แกน | ตอบคำถามว่าอะไร | ตัวอย่าง |
|---|---|---|
| **CAP Consistency** | เมื่ออ่านข้อมูลจาก**node ต่างกัน**ใน cluster เดียวกัน จะเห็นค่าตรงกันไหม (multi-node) | replica ยุโรปกับ primary เอเชียเห็นข้อมูลตรงกันหรือไม่ตอนนี้ |
| **Transaction Isolation Level** | เมื่อมีหลาย transaction รันพร้อมกัน**บน node/database เดียวกัน** จะเห็นผลกระทบของกันและกันแค่ไหน (single-node concurrency) | transaction A เห็นข้อมูลที่ transaction B ยังไม่ commit ไหม (Read Committed, Repeatable Read, Serializable) |

ทั้งสองแกนนี้**เป็นอิสระต่อกัน** — ระบบหนึ่งสามารถมี Serializable isolation (การรับประกันสูงสุดในแกน single-node) ได้พร้อมกับเป็น AP ในแกน multi-node ก็ได้ เช่น PostgreSQL primary รัน Serializable isolation อย่างเคร่งครัดสำหรับ transaction ในตัวเอง แต่ถ้า replica ใช้ asynchronous replication การอ่านจาก replica นั้นก็ยังเป็น **eventually consistent** เมื่อเทียบกับ primary อยู่ดี — ความเข้มงวดของ isolation level ไม่ได้แปลว่าข้อมูลระหว่าง node จะ sync กันทันที

ในทางกลับกัน ระบบที่เป็น CP อย่างเคร่งครัดในแกน multi-node (ทุก node เห็นข้อมูลตรงกันเสมอ) ก็ยังสามารถเลือกใช้ isolation level ที่หลวมกว่า (เช่น Read Committed) ภายใน transaction เดียวได้ ทั้งสองเรื่องนี้เป็นการตัดสินใจแยกกันคนละชั้น: **CAP/PACELC ตัดสินใจที่ชั้นสถาปัตยกรรมระบบ (system design) ส่วน isolation level ตัดสินใจที่ชั้น transaction ของ database engine เดียว**

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| CAP theorem | เลือก C หรือ A ได้เต็มที่พร้อมกัน **เฉพาะตอนไม่มี partition**; เมื่อ partition เกิดขึ้นจริง เลือกได้แค่หนึ่งใน C/A |
| P ไม่ใช่ตัวเลือก | ระบบ distributed จริงเจอ partition แน่นอนสักวัน คำถามจริงคือ "ตอนนั้นจะเลือก C หรือ A" |
| CP example | PostgreSQL synchronous replication — ยอมเสีย A เพื่อรักษา C |
| AP example | DNS — ตอบจาก cache แม้ authoritative server เข้าไม่ถึง เพื่อรักษา A |
| PACELC | ขยาย CAP: ไม่มี partition ก็ยังมี trade-off Latency vs Consistency (sync vs async replication) |
| CAP C vs Isolation Level | คนละแกน — CAP C คือความตรงกันข้าม**node**, isolation level คือความตรงกันข้าม**transaction**บน node เดียว |

## คำถามทบทวน

1. อธิบายว่าทำไมการพูดว่า "ระบบนี้เลือกได้ 2 จาก 3 ของ CAP" เป็นความเข้าใจที่คลาดเคลื่อน คำอธิบายที่ถูกต้องกว่าคืออะไร
2. ทำไม Partition Tolerance ถึงถูกมองว่า "ไม่ใช่ตัวเลือก" ในระบบ distributed จริง
3. ยกตัวอย่างระบบหนึ่งที่เอียงไปทาง CP และอีกระบบที่เอียงไปทาง AP พร้อมอธิบายว่าพฤติกรรมตอนเกิด partition ต่างกันอย่างไร
4. PACELC เพิ่มอะไรเข้าไปจาก CAP theorem เดิม พร้อมยกตัวอย่างระบบที่แสดง trade-off นี้แม้ไม่มี partition
5. Consistency ใน CAP กับ Isolation Level ของ transaction (เช่น Serializable) ต่างกันอย่างไร ทำไมระบบหนึ่งถึงมีทั้งสองแบบผสมกันได้

---

ถัดไป: [บทที่ 2 — Sharding, Replication และ Distributed Cache](02-scaling-data.md)
