# บทที่ 2: Sharding, Replication และ Distributed Cache

## 2.1 Sharding — แบ่งข้อมูลตาม Key

เมื่อข้อมูลใหญ่เกินกว่าจะเก็บใน node เดียว (ทั้งด้าน storage และ throughput) คำตอบคือ **sharding** — แบ่งข้อมูลออกเป็นชิ้นๆ (shard) กระจายไปยังหลาย node โดยแต่ละ node ถือข้อมูลแค่บางส่วน (ต่างจาก replication ในหัวข้อ 2.2 ที่ทุก node ถือข้อมูล**ชุดเดียวกัน**ทั้งหมด)

### Range-Based Sharding

แบ่งตามช่วงของ key เช่น ระบบเก็บ order แบ่งตาม `order_id`:

```
Shard 1: order_id 1 – 1,000,000
Shard 2: order_id 1,000,001 – 2,000,000
Shard 3: order_id 2,000,001 – 3,000,000
```

**ข้อดี** — query แบบ range (`WHERE order_id BETWEEN x AND y`) ทำได้มีประสิทธิภาพ เพราะข้อมูลที่ต้องการอยู่ใน shard เดียวหรือติดกัน
**ข้อเสีย** — เสี่ยง **hotspot**: ถ้า key ที่สร้างใหม่เรียงตามลำดับเวลา (เช่น order_id auto-increment) shard ล่าสุดจะรับ write ทั้งหมดเสมอ ในขณะที่ shard เก่าแทบไม่มี write เลย ทำให้โหลดไม่กระจายเท่ากัน

### Hash-Based Sharding

แบ่งโดย hash ค่า key แล้ว mod ด้วยจำนวน shard เช่น `shard = hash(user_id) % N`

**ข้อดี** — กระจายโหลดสม่ำเสมอกว่า range-based มาก เพราะ hash function กระจาย key แบบสุ่ม ไม่ขึ้นกับลำดับการสร้าง
**ข้อเสีย** — query แบบ range ทำไม่ได้อีกต่อไป (key ที่ใกล้กันกระจายไปคนละ shard) และมีปัญหาใหญ่ตอน **resharding** ที่ต้องอธิบายละเอียดต่อไป

### ปัญหา Resharding ของ Hash-Based แบบ Naive

สมมติระบบมี 4 shard ใช้สูตร `shard = hash(key) % 4` วันหนึ่งข้อมูลโตขึ้นจนต้องเพิ่มเป็น 5 shard สูตรกลายเป็น `hash(key) % 5`

ปัญหาคือ **ผลลัพธ์ของ `% 4` กับ `% 5` แทบไม่มีความสัมพันธ์กันเลย** — key ส่วนใหญ่จะได้ shard คนละตัวกับก่อนหน้า แม้ว่า hash(key) จะเป็นค่าเดิมทุกประการ ตัวอย่างตัวเลข:

```
hash(key) = 17
% 4 → shard 1     (17 mod 4 = 1)
% 5 → shard 2     (17 mod 5 = 2)   ← ย้าย shard ทั้งที่ key ไม่เปลี่ยน

hash(key) = 23
% 4 → shard 3     (23 mod 4 = 3)
% 5 → shard 3     (23 mod 5 = 3)   ← บังเอิญตรงกัน (โชคดี)
```

โดยเฉลี่ยแล้ว การเปลี่ยนจำนวน shard จาก N เป็น N+1 ด้วยสูตร mod ธรรมดา ทำให้ **เกือบทุก key ต้องถูกย้าย** (ประมาณ (N-1)/N ของ key ทั้งหมด) ซึ่งหมายถึงต้อง copy ข้อมูลปริมาณมหาศาลข้าม shard พร้อมกัน ระบบแทบใช้งานไม่ได้ระหว่างกระบวนการนี้

### Consistent Hashing แก้ปัญหานี้อย่างไร

**Consistent hashing** เป็นเทคนิคเดียวกับที่ใช้อธิบายไว้แล้วใน [Volume 2 — Load Balancer](../02-network/05-reverse-proxy-load-balancer.md) สำหรับกระจาย request ไปยัง backend — หลักการเดียวกันนี้เอามาใช้กับการกระจายข้อมูลไปยัง shard ได้โดยตรง

แทนที่จะ mod ตรงๆ consistent hashing วาง shard ทั้งหมดไว้บน **ring** ของค่า hash (0 ถึงค่าสูงสุด) แล้ว hash key ไปเทียบตำแหน่งบน ring เดียวกัน — key หนึ่งจะถูกจัดเข้า shard ตัวแรกที่เจอเมื่อเดินตาม ring ตามเข็มนาฬิกาจากตำแหน่งของ key นั้น

```
Ring (0 ────────────────────── max):
         Shard A        Shard B        Shard C
            │               │               │
   key1 ──► │  key2 ──►     │   key3 ──►    │
   (ไป A)   (ไป B)          (ไป C)

เพิ่ม Shard D เข้ามาระหว่าง B กับ C:
         Shard A        Shard B    Shard D    Shard C
   key1 ──► A   key2 ──► B      key_new ► D   key3 ──► C

→ เฉพาะ key ที่อยู่ในช่วงระหว่าง B กับ D เดิม (ที่ตอนนี้ต้องไป D แทน C) เท่านั้นที่ย้าย
→ key อื่นทั้งหมดยังชี้ไป shard เดิม ไม่กระทบ
```

เมื่อเพิ่มหรือลด shard ด้วย consistent hashing มีเพียง key ที่อยู่ **ในช่วงติดกับ shard ที่เปลี่ยนแปลง** เท่านั้นที่ต้องย้าย (โดยเฉลี่ยประมาณ 1/N ของ key ทั้งหมด ไม่ใช่เกือบทั้งหมดแบบ mod ธรรมดา) นี่คือเหตุผลที่ระบบ distributed storage จริง (Cassandra, DynamoDB, Redis Cluster) ใช้แนวคิดนี้ (มักในรูปแบบ hash ring ที่มี virtual node เพื่อกระจายโหลดสม่ำเสมอขึ้นอีกชั้น)

## 2.2 Replication Topology

Sharding แบ่งข้อมูล ส่วน **replication** คือการ**ก็อปปี้ข้อมูลชุดเดียวกัน**ไปไว้หลาย node เพื่อความทนทานต่อความล้มเหลวและกระจายโหลดการอ่าน (ระดับกลไกจริงของ PostgreSQL อยู่ใน [Volume 5 — Partitioning และ Replication](../05-postgresql/05-partitioning-and-replication.md) บทนี้พูดถึงภาพรวมระดับ system design ว่า topology แบบไหนเหมาะกับ requirement แบบไหน)

### Leader-Follower (Single-Leader)

Node หนึ่งเป็น **leader** รับ write ทั้งหมด แล้ว replicate ไปยัง **follower** หลายตัวที่รับได้แค่ read (หรือใช้สำรองไว้ตอน leader ล่ม)

```
        write
Client ────────► Leader ──replicate──► Follower 1
                     │                ► Follower 2
                     └──replicate────► Follower 3
```

**ข้อดี** — ไม่มีปัญหาเรื่อง write conflict เพราะมีแหล่งความจริงเดียว (single source of truth) เข้าใจง่าย debug ง่าย
**ข้อจำกัด** — write throughput ถูกจำกัดด้วย leader ตัวเดียว, ถ้า leader ล่มต้องมีกระบวนการ failover (เลือก follower ตัวใหม่เป็น leader)

### Multi-Leader

มี leader หลายตัว (มักอยู่คนละ region) แต่ละตัวรับ write ได้เอง แล้ว replicate ไปหากันและกัน

```
Region A: Leader A ◄──replicate──► Leader B :Region B
   (รับ write ในเอเชีย)              (รับ write ในยุโรป)
```

**ข้อดี** — write latency ต่ำในแต่ละ region เพราะ client เขียนที่ leader ใกล้ตัวเองโดยไม่ต้องข้าม region
**ปัญหาที่ single-leader ไม่มี** — **write conflict**: ถ้า user คนเดียวกันแก้ record เดียวกันที่ Leader A และ Leader B พร้อมกัน (เช่น อัปเดตที่อยู่จัดส่งจากสองอุปกรณ์) ทั้งสอง leader accept write ของตัวเองสำเร็จก่อนจะรู้ว่ามี write อีกฝั่งขัดแย้งอยู่ ระบบต้องมีกลไก**คืนดี (conflict resolution)** เช่น last-write-wins (ใช้ timestamp ตัดสิน แต่เสี่ยงข้อมูลหาย), version vector/vector clock (ตรวจจับว่า write ไหนเกิดก่อน-หลังจริง หรือเกิดพร้อมกัน), หรือ CRDT (Conflict-free Replicated Data Type — ออกแบบโครงสร้างข้อมูลให้ merge กันได้เองโดยอัตโนมัติโดยไม่ขัดแย้ง)

### Leaderless

ไม่มี node ไหนเป็น leader โดยเฉพาะ — client เขียน/อ่านไปยัง node หลายตัวพร้อมกัน แล้วใช้ **quorum** ตัดสินว่า write/read สำเร็จหรือไม่ (เช่น เขียนสำเร็จเมื่อมี node อย่างน้อย W ตัวยืนยัน, อ่านสำเร็จเมื่อรวบรวมคำตอบจาก R ตัว โดยทั่วไปตั้งค่าให้ W + R > N เพื่อรับประกันว่า read กับ write อย่างน้อยจะเจอ node ที่มีข้อมูลล่าสุดร่วมกันเสมอ)

**ปัญหาเดียวกับ multi-leader** — write พร้อมกันจาก client คนละตัวไปยัง node คนละชุดก็ทำให้เกิด conflict ได้เช่นกัน ต้องใช้กลไก conflict resolution แบบเดียวกัน (vector clock, CRDT) Cassandra และ DynamoDB เป็นตัวอย่างของระบบ leaderless

### เปรียบเทียบ

| Topology | Write Throughput | Conflict Resolution ที่ต้องมี | เหมาะกับ |
|---|---|---|---|
| Leader-Follower | จำกัดที่ leader ตัวเดียว | ไม่ต้อง (single source of truth) | ระบบที่ต้องการความถูกต้องสูง เช่น ธุรกรรมการเงิน |
| Multi-Leader | สูงกว่า (เขียนได้หลาย region) | ต้องมี (last-write-wins, vector clock, CRDT) | ระบบที่ client กระจายหลาย region และยอมรับ conflict บ้าง |
| Leaderless | สูงสุด (เขียนไป node ไหนก็ได้) | ต้องมี (คล้าย multi-leader) | ระบบที่เน้น availability สูงสุด เช่น shopping cart, session store |

## 2.3 Distributed Cache ระดับระบบ

กลไกของ Redis Cluster (การแบ่ง key เป็น hash slot, การกระจาย node) อยู่ใน [Volume 6 — Redis Cluster](../06-redis/03-cluster.md) แล้ว หัวข้อนี้พูดถึง**ผลกระทบระดับ system design** เมื่อ cache ต้องทำงานข้าม region/instance หลายชุด ไม่ใช่แค่ instance เดียว

### Cache Invalidation ข้าม Region/Instance

เมื่อระบบมี cache หลายชุดกระจายตาม region (เพื่อลด latency ให้ user ในแต่ละพื้นที่) การอัปเดตข้อมูลต้นทาง (เช่น ราคาสินค้าถูกแก้ที่ region เอเชีย) ต้องหาทาง**แจ้ง invalidate** ไปยัง cache ทุก region ไม่ใช่แค่ region ที่เขียน:

```
เขียนที่ region เอเชีย:
  DB (เอเชีย) ──update──► Cache เอเชีย (invalidate ทันที)
                    │
                    └──event/message──► Cache ยุโรป (invalidate ทีหลัง — มี delay)
                    └──event/message──► Cache สหรัฐฯ (invalidate ทีหลัง — มี delay)
```

ระหว่างช่วง delay นี้ cache ที่ region อื่นยัง**เสิร์ฟข้อมูลเก่า** — เป็นเรื่องปกติที่ต้องยอมรับถ้าเลือกใช้ cache แบบกระจาย ไม่มีทาง invalidate ข้าม region ได้แบบ atomic 100% โดยไม่แลกด้วย latency ที่สูงขึ้นมาก (ตรงกับหลัก PACELC ในบทที่ 1 — latency vs consistency)

### Stale-Cache Risk ภายใต้ Network Partition — เชื่อมกับ CAP

ถ้าเกิด network partition ระหว่าง region ขึ้นมา (เชื่อมกับกรอบ CAP ใน [บทที่ 1](01-cap-and-consistency.md)) cache ที่ region ซึ่งขาดการเชื่อมต่อจากต้นทางจะ**ไม่มีทางรู้เลย**ว่าข้อมูลที่ตัวเองถืออยู่ล้าสมัยไปแล้วหรือยัง มีสองทางเลือก:

- **AP-leaning** — ยังคงเสิร์ฟค่าจาก cache เดิมต่อไปแม้ partition (เหมือน DNS ในบทที่ 1) — availability สูง แต่เสี่ยงเสิร์ฟข้อมูลผิดโดยไม่รู้ตัว
- **CP-leaning** — ตั้ง TTL สั้นและปฏิเสธ/fallback ไปอ่านจาก source of truth โดยตรงเมื่อ cache หมดอายุระหว่าง partition — consistency สูงกว่า แต่ยอม latency สูงขึ้น (ต้องย้อนไปอ่าน DB ตรง) หรือ availability ต่ำลงถ้า DB เองก็เข้าไม่ถึงจาก region นั้นเช่นกัน

ทางเลือกที่ใช้จริงส่วนใหญ่คือ**ผสม**: ตั้ง TTL สั้นพอสมควรเพื่อจำกัดความเสียหายของ staleness (bound the blast radius) และใช้ event-based invalidation (ผ่าน pub/sub หรือ message queue) เป็นกลไกหลักเมื่อ network ปกติ — นี่คือเหตุผลที่ cache แทบทุกระบบไม่เคยพึ่ง "ไม่มี TTL เลย" แม้จะมี invalidation event ที่ดีแค่ไหนก็ตาม เพราะ TTL คือเซฟตี้เน็ตสุดท้ายเมื่อ event หายไปในระหว่าง partition

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| Range-based sharding | Query range ได้ดี แต่เสี่ยง hotspot ถ้า key เรียงตามเวลา |
| Hash-based sharding | กระจายโหลดสม่ำเสมอ แต่ query range ไม่ได้ และมีปัญหา resharding |
| ปัญหา resharding แบบ mod ธรรมดา | เพิ่ม shard แล้ว mod เปลี่ยน ทำให้เกือบทุก key ต้องย้ายพร้อมกัน |
| Consistent hashing | วาง shard บน ring, เพิ่ม/ลด shard กระทบเฉพาะ key ที่อยู่ช่วงติดกัน — remap น้อยมาก |
| Leader-Follower | ไม่มี write conflict แต่ throughput จำกัดที่ leader |
| Multi-Leader / Leaderless | write throughput สูงกว่า แต่ต้องมีกลไก conflict resolution ที่ single-leader ไม่ต้องมี |
| Distributed cache invalidation | ข้าม region มี delay เสมอ, TTL คือเซฟตี้เน็ตสุดท้ายเมื่อ event invalidation หายไปตอน partition |

## คำถามทบทวน

1. อธิบายว่าทำไม hash-based sharding แบบ `hash(key) % N` ธรรมดา ถึงมีปัญหาหนักตอนเพิ่ม shard พร้อมยกตัวอย่างตัวเลข
2. Consistent hashing ลดปัญหา resharding ได้อย่างไร เชื่อมโยงกับที่เคยเรียนในบท Load Balancer อย่างไร
3. Multi-leader replication เจอปัญหาอะไรที่ leader-follower ไม่เจอ ยกตัวอย่างกลไกแก้ปัญหานั้นมา 2 แบบ
4. ทำไม distributed cache ถึงยังต้องมี TTL แม้จะมีระบบ invalidation ผ่าน event/message แล้วก็ตาม เชื่อมโยงกับ CAP อย่างไร
5. Range-based กับ hash-based sharding เหมาะกับ query pattern แบบไหนตามลำดับ ยกตัวอย่างระบบที่เหมาะกับแต่ละแบบ

---

ก่อนหน้า: [บทที่ 1 — CAP และ Consistency](01-cap-and-consistency.md) | ถัดไป: [บทที่ 3 — CDN, High Availability และ Disaster Recovery](03-cdn-ha-dr.md)
