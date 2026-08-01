# บทที่ 3: CDN, High Availability และ Disaster Recovery

## 3.1 CDN (Content Delivery Network)

**CDN** คือเครือข่ายของ **edge server** กระจายตามพื้นที่ต่างๆ ทั่วโลก ทำหน้าที่เก็บสำเนา (cache) ของเนื้อหาไว้ใกล้ผู้ใช้ที่สุด แทนที่ทุก request จะต้องวิ่งกลับไปถึง **origin server** (server ต้นทางที่มีข้อมูลจริง) ทุกครั้ง

```
ไม่มี CDN:
  User (กรุงเทพฯ) ──────────────────► Origin server (สหรัฐฯ)
                    latency ~200ms ข้ามทวีปทุก request

มี CDN:
  User (กรุงเทพฯ) ──► Edge server (สิงคโปร์) ──(cache miss เท่านั้น)──► Origin (สหรัฐฯ)
                    latency ~20ms                latency ~200ms (นานๆ ครั้ง)
```

### Cache-Hit vs Cache-Miss Latency

- **Cache hit** — edge server มีเนื้อหาที่ขอเก็บอยู่แล้ว ตอบ user ได้ทันทีโดยไม่ต้องติดต่อ origin เลย latency ต่ำมาก (จำกัดด้วยระยะทางถึง edge server ที่ใกล้ที่สุด ไม่ใช่ถึง origin)
- **Cache miss** — edge server ไม่มีเนื้อหานั้น (ครั้งแรกที่มีคนขอ หรือ cache หมดอายุ) ต้องไปดึงจาก origin ก่อนแล้วค่อยเก็บไว้ตอบ request ถัดไป (และมักตอบ user คนแรกไปพร้อมกันด้วย) latency สูงเท่ากับไม่มี CDN สำหรับ request นั้นครั้งเดียว

**Cache-hit ratio** (สัดส่วน request ที่ตอบจาก edge ได้โดยไม่ต้องแตะ origin) คือตัวชี้วัดหลักของ CDN ที่ดี ยิ่งสูงยิ่งลด latency เฉลี่ยและลดโหลดที่ origin ได้มาก เนื้อหาที่เปลี่ยนไม่บ่อย (รูปภาพ, video, static asset, JS/CSS bundle) เหมาะกับ CDN ที่สุดเพราะ cache-hit ratio สูงตามธรรมชาติ

### Edge Compute สำหรับ Dynamic Content

CDN รุ่นใหม่ (Cloudflare Workers, AWS Lambda@Edge) ไม่ได้ทำแค่ cache static file แต่รัน**โค้ดจริง**ที่ edge ได้ด้วย เช่น ตรวจสอบ auth token, personalize เนื้อหาบางส่วนก่อนส่งกลับ, หรือทำ A/B testing routing — ช่วยลด latency สำหรับ logic เบาๆ ที่ไม่จำเป็นต้องวิ่งไปถึง origin แต่ไม่เหมาะกับ logic ที่ต้องพึ่ง state ส่วนกลาง (เช่น อ่าน/เขียน database) ซึ่งยังต้องวิ่งไปถึง origin หรือ regional service อยู่ดี

## 3.2 High Availability (HA)

### Redundancy และการกำจัด Single Point of Failure (SPOF)

**SPOF** คือส่วนประกอบใดๆ ในระบบที่ถ้าล่มแล้ว**ทั้งระบบหยุดทำงาน** หลักการของ HA คือหา SPOF ทุกจุดแล้วเพิ่ม **redundancy** (สำรอง) ให้พอที่จะทนต่อความล้มเหลวของส่วนประกอบเดียวได้เสมอ

```
ก่อน (มี SPOF หลายจุด):
  Client ──► App Server (1 ตัว) ──► Database (1 ตัว)
             ↑ ล่มแล้วระบบหยุดทั้งหมด    ↑ ล่มแล้วระบบหยุดทั้งหมด

หลัง (กำจัด SPOF):
  Client ──► Load Balancer ──┬─► App Server 1
                              ├─► App Server 2      (หลายตัว, ตัวไหนล่ม LB route ไปตัวอื่น)
                              └─► App Server 3
                                      │
                                      ▼
                              DB Primary ──replicate──► DB Replica (standby)
                              (ล่มแล้ว failover ไป replica อัตโนมัติ)
```

จุดสำคัญ: Load Balancer เองก็เป็น SPOF ได้ถ้ามีตัวเดียว — ระบบ production จริงมักมี LB สำรอง (active-passive หรือ DNS-based failover ไปยัง LB คู่ที่สอง) ทุกชั้นของสถาปัตยกรรมต้องถูกตรวจสอบว่ามี "ตัวเดียว" หลงเหลืออยู่หรือไม่

### Failover

กระบวนการสลับไปใช้ส่วนประกอบสำรองเมื่อตัวหลักล่ม แบ่งเป็น:

- **Automatic failover** — ระบบตรวจจับความล้มเหลวและสลับเองโดยไม่ต้องมีคนกด (เช่น PostgreSQL ใช้ tool อย่าง Patroni ตรวจ health ของ primary แล้วเลื่อน replica ขึ้นเป็น primary ใหม่)
- **Manual failover** — ต้องมีคนตัดสินใจสลับเอง มักใช้เมื่อความเสี่ยงของการสลับผิดพลาด (เช่น split-brain — ทั้ง primary เก่าและใหม่คิดว่าตัวเองเป็น primary พร้อมกัน) สูงกว่าความเสี่ยงของการรอสักครู่ให้คนตรวจสอบก่อน

### SLA, SLO, SLI — วัดความพร้อมใช้งานอย่างเป็นรูปธรรม

สามคำนี้มักถูกใช้ปนกัน แต่ความหมายต่างระดับกัน:

- **SLI (Service Level Indicator)** — ตัวเลขที่**วัดได้จริง**สะท้อนคุณภาพของระบบ เช่น "สัดส่วน request ที่ตอบสำเร็จภายใน 200ms", "uptime percentage ต่อเดือน" — วิธีวัด SLI จริงในทางปฏิบัติ (ผ่าน metrics, dashboard) อยู่ใน [Volume 14 — Observability](../14-observability/README.md)
- **SLO (Service Level Objective)** — **เป้าหมายภายใน**ที่ทีมตั้งไว้สำหรับ SLI นั้น เช่น "SLO: 99.9% ของ request ต้องตอบภายใน 200ms" — เป็น target ที่ทีม engineering ใช้ตัดสินใจ (เช่น ตัดสินใจว่าจะ deploy feature ใหม่ไหมถ้า error budget ใกล้หมด)
- **SLA (Service Level Agreement)** — **สัญญาที่ให้กับลูกค้า**ภายนอก มักมีผลทางกฎหมาย/ธุรกิจ (เช่น คืนเงินถ้าทำไม่ได้ตามที่สัญญา) โดยทั่วไป SLA จะตั้ง**หลวมกว่า SLO ภายใน** เพื่อให้มี buffer ไม่ให้ทีม engineering ต้องเครียดจนเกินไปกับทุก incident เล็กๆ

```
SLI (วัดจริง) ──► เทียบกับ ──► SLO (เป้าหมายภายในทีม) ──► หลวมกว่า ──► SLA (สัญญากับลูกค้า)
```

### ตาราง "กี่ Nines" — Downtime ที่ยอมรับได้

| Availability | Downtime ต่อปี | Downtime ต่อเดือน | Downtime ต่อวัน |
|---|---|---|---|
| 99% (2 nines) | ~3.65 วัน | ~7.3 ชั่วโมง | ~14.4 นาที |
| 99.9% (3 nines) | ~8.76 ชั่วโมง | ~43.8 นาที | ~1.44 นาที |
| 99.99% (4 nines) | ~52.6 นาที | ~4.38 นาที | ~8.6 วินาที |
| 99.999% (5 nines) | ~5.26 นาที | ~26.3 วินาที | ~0.86 วินาที |

ทุก nine ที่เพิ่มขึ้น ต้นทุนด้าน engineering (redundancy หลายชั้น, automation, on-call, multi-region) เพิ่มขึ้นแบบไม่เป็นเส้นตรง — การเลือก availability target ที่เหมาะสมจึงเป็นการตัดสินใจทางธุรกิจ ไม่ใช่แค่ทางเทคนิค (ระบบ internal tool อาจพอใจกับ 99% แต่ payment gateway อาจต้องการ 99.99% ขึ้นไป)

## 3.3 Disaster Recovery (DR)

HA ป้องกันความล้มเหลวระดับ component (server ตัวเดียวล่ม) ส่วน **DR** เตรียมพร้อมสำหรับความล้มเหลวระดับใหญ่กว่านั้นมาก — data center ทั้งแห่งล่ม, region ทั้งโซนใช้งานไม่ได้, หรือข้อมูลเสียหายจากเหตุ เช่น ransomware

### RPO (Recovery Point Objective) และ RTO (Recovery Time Objective)

- **RPO** — "เรายอมสูญเสียข้อมูลได้มากแค่ไหน" วัดเป็นระยะเวลา นับย้อนจากจุดเกิดเหตุ ตัวอย่างที่เป็นรูปธรรม: **RPO = 1 ชั่วโมง** หมายความว่าถ้าระบบล่มตอน 14:00 และ backup ล่าสุดทำไว้ตอน 13:00 ข้อมูลที่เกิดขึ้นระหว่าง 13:00–14:00 จะสูญหายไป และนั่นคือระดับความเสียหายที่ยอมรับได้ตามเป้าหมายที่ตั้งไว้ — RPO กำหนด**ความถี่ของการ backup/replicate** ที่จำเป็น
- **RTO** — "เรายอมให้ระบบหยุดทำงานนานแค่ไหนก่อนจะกู้คืนสำเร็จ" วัดเป็นระยะเวลาตั้งแต่เกิดเหตุจนระบบกลับมาใช้งานได้ปกติ ตัวอย่าง: **RTO = 4 ชั่วโมง** หมายความว่าทีมต้องกู้ระบบให้ใช้งานได้ภายใน 4 ชั่วโมงหลังเกิดภัยพิบัติ — RTO กำหนด**ความเร็วของกระบวนการกู้คืน**ที่ต้องเตรียมไว้ล่วงหน้า (automation, runbook, มี standby environment พร้อมสลับหรือไม่)

```
เวลา: ──────●─────────────────●──────────────────►
        Backup ล่าสุด      เกิดเหตุ (ระบบล่ม)      ระบบกลับมาใช้งานได้
        (13:00)             (14:00)                (18:00)

RPO = 14:00 - 13:00 = 1 ชั่วโมง (ข้อมูลที่หาย)
RTO = 18:00 - 14:00 = 4 ชั่วโมง (เวลาที่ระบบหยุด)
```

RPO และ RTO ยิ่งตั้งให้สั้นเท่าไหร่ ต้นทุนยิ่งสูงขึ้นเท่านั้น — RPO ใกล้ 0 ต้องมี synchronous replication ตลอดเวลา (แลกด้วย latency ตามที่เรียนในบทที่ 1), RTO สั้นมากต้องมี standby environment ที่พร้อมรับ traffic ทันที ไม่ใช่แค่มี backup ไฟล์เก็บไว้

### Backup Strategy Tiers

| ระดับ | วิธีการ | RPO โดยประมาณ | ต้นทุน |
|---|---|---|---|
| Full backup รายวัน/รายสัปดาห์ | dump ข้อมูลทั้งหมดเป็นระยะ | เป็นชั่วโมง–วัน | ต่ำสุด |
| Incremental/differential backup | backup เฉพาะส่วนที่เปลี่ยนตั้งแต่ครั้งก่อน | เป็นชั่วโมง | ต่ำ-กลาง |
| Continuous archiving (WAL shipping) | ส่ง transaction log ต่อเนื่อง | เป็นนาที | กลาง |
| Synchronous replication ข้าม region | replica ยืนยันก่อน commit เสมอ | ใกล้ 0 | สูงสุด |

### Multi-Region Deployment — DR Posture ที่แข็งแกร่งที่สุด

การมี infrastructure เต็มรูปแบบ (compute, database, cache) พร้อมทำงานอยู่ใน**หลาย region พร้อมกัน** คือแนวทาง DR ที่ทนทานที่สุด เพราะแม้ทั้ง region หนึ่งล่มสนิท (ไม่ใช่แค่ server ตัวเดียว) ระบบยังทำงานต่อได้จาก region อื่น ลักษณะการ deploy มีได้หลายระดับ:

- **Active-Passive** — region สำรองอยู่เฉยๆ พร้อม standby ไว้ สลับเมื่อ region หลักล่ม (RTO สูงกว่าเพราะต้อง "ปลุก" region สำรองขึ้นมาทำงานจริง)
- **Active-Active** — ทุก region รับ traffic จริงพร้อมกันตลอดเวลา ถ้า region หนึ่งล่ม traffic route ไปยัง region ที่เหลือทันที (RTO ต่ำที่สุด แต่ต้องแก้ปัญหา multi-leader replication conflict ตามที่กล่าวไว้ใน[บทที่ 2](02-scaling-data.md) เพราะทุก region เขียนข้อมูลได้พร้อมกัน)

**Cost Trade-off**: multi-region ทุกแบบเพิ่มต้นทุน infrastructure อย่างน้อยเท่าตัว (ต้องมี compute/storage สำรองพร้อมใช้งานจริง ไม่ใช่แค่ข้อมูล backup เก็บนิ่งๆ) บวกกับต้นทุนด้าน engineering complexity (data replication ข้าม region, conflict resolution, testing DR scenario จริงอย่างสม่ำเสมอ) — การตัดสินใจลงทุน multi-region ต้องเทียบกับความเสียหายทางธุรกิจถ้า region เดียวล่มจริง (ยิ่งระบบ critical เช่น payment ยิ่งคุ้มที่จะลงทุน)

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| CDN | Edge server เก็บ cache ใกล้ user ลด latency; cache-hit ratio คือตัวชี้วัดหลัก |
| Edge compute | รันโค้ดเบาๆ ที่ edge ได้ แต่ logic ที่ต้องพึ่ง state ส่วนกลางยังต้องวิ่งไป origin |
| SPOF elimination | ทุกชั้นของระบบต้องมีสำรอง รวมถึง load balancer เอง |
| SLI/SLO/SLA | SLI วัดจริง, SLO เป้าหมายภายใน, SLA สัญญากับลูกค้า (มักหลวมกว่า SLO) |
| RPO | ข้อมูลสูญเสียได้มากแค่ไหน กำหนดความถี่ backup/replication |
| RTO | หยุดทำงานได้นานแค่ไหน กำหนดความเร็วของกระบวนการกู้คืน |
| Multi-region | DR posture แข็งแกร่งสุด แต่ต้นทุนสูงและต้องแก้ปัญหา conflict แบบ multi-leader |

## คำถามทบทวน

1. อธิบายความต่างระหว่าง cache-hit และ cache-miss latency ใน CDN แล้วบอกว่าทำไม static asset ถึงเหมาะกับ CDN มากกว่า dynamic content ที่ต้องพึ่ง state
2. SLI, SLO, SLA ต่างกันอย่างไร ทำไม SLA ที่ให้ลูกค้ามักตั้งหลวมกว่า SLO ภายในทีมเสมอ
3. ระบบมี RPO = 15 นาที และ RTO = 2 ชั่วโมง ถ้าระบบล่มตอน 09:00 และ backup ล่าสุดทำไว้ตอน 08:50 ระบบควรกลับมาใช้งานได้ภายในเวลาใด และข้อมูลช่วงไหนที่มีความเสี่ยงจะหาย
4. ทำไม availability จาก 99.9% ไปเป็น 99.99% ถึงมีต้นทุนเพิ่มขึ้นแบบไม่เป็นเส้นตรง (ไม่ใช่แค่เพิ่มนิดหน่อย)
5. Active-Active multi-region ให้ RTO ต่ำกว่า Active-Passive แต่ต้องแลกกับปัญหาอะไรเพิ่มเติม เชื่อมโยงกับเนื้อหาบทที่ 2 อย่างไร

---

ก่อนหน้า: [บทที่ 2 — Sharding, Replication และ Distributed Cache](02-scaling-data.md) | ถัดไป: [บทที่ 4 — Case Study: Payment และ Chat](04-case-study-payment-chat.md)
