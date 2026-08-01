# บทที่ 2: Fluent Bit และ OpenSearch

## 2.1 Log Pipeline คืออะไร

ในระบบที่มีแค่ server เดียว การดู log ง่ายมาก — `ssh` เข้าไปแล้ว `tail -f /var/log/app.log` ก็จบ แต่ในระบบ microservices ที่รันเป็นสิบ services บนหลายสิบ pod กระจายอยู่หลาย node การไล่ `ssh` เข้าไปทีละเครื่องเพื่อหา log ที่เกี่ยวข้องเป็นไปไม่ได้ในทางปฏิบัติ — ต้องมี **log pipeline** ที่รวบรวม log จากทุกที่มาไว้ที่เดียวและค้นหาได้

Pipeline โดยทั่วไปมี 5 ขั้นตอน:

```
┌─────────┐    ┌───────────────┐    ┌────────┐    ┌───────────┐    ┌────────┐
│ Collect │───►│ Parse / Filter│───►│  Ship  │───►│   Index   │───►│ Search │
└─────────┘    └───────────────┘    └────────┘    └───────────┘    └────────┘
 อ่าน log จาก      แปลง raw text       ส่งข้าม        เก็บให้ค้นหา       query ผ่าน
 ไฟล์/stdout       เป็นโครงสร้าง        network        เร็ว (inverted    UI หรือ API
 ของแต่ละ pod      (JSON), เติม/       ไปยังปลายทาง     index)
                   กรอง field
```

- **Collect** — อ่าน log ดิบจากแหล่งกำเนิด (ไฟล์, stdout ของ container, journal)
- **Parse/Filter** — แปลง log ดิบให้เป็นโครงสร้าง (structured), เติม metadata (pod name, namespace), กรอง log ที่ไม่ต้องการทิ้ง
- **Ship** — ส่งข้าม network ไปยังปลายทางเก็บถาวร
- **Index** — เก็บในรูปแบบที่ค้นหาได้เร็ว (ไม่ใช่แค่เก็บเป็นไฟล์ text เรียงลำดับ)
- **Search** — engineer หรือ dashboard เข้ามา query

บทนี้ใช้ **Fluent Bit** ทำหน้าที่ collect+parse/filter+ship และ **OpenSearch** ทำหน้าที่ index+search พร้อม **OpenSearch Dashboards** สำหรับ visualize

## 2.2 Structured Logging — เงื่อนไขที่ทำให้ pipeline มีประโยชน์จริง

ก่อนพูดถึงเครื่องมือ ต้องเข้าใจก่อนว่า pipeline ทั้งหมดนี้จะไร้ประโยชน์ถ้า log ที่ผลิตออกมาเป็น **unstructured text** ธรรมดา:

```
2026-08-01 14:32:01 ERROR failed to process order 12345 for user 987: connection timeout
```

log บรรทัดนี้อ่านด้วยตาคนได้ แต่ query ด้วยเครื่องยากมาก — ถ้าอยากหา error ทั้งหมดของ `user_id=987` ต้องเขียน regex เดา format เอาเอง และ format มักไม่คงที่ (แต่ละจุดในโค้ดเขียน log message ไม่เหมือนกัน)

เทียบกับ **structured logging** (JSON):

```json
{"timestamp":"2026-08-01T14:32:01Z","level":"error","message":"failed to process order","order_id":12345,"user_id":987,"error":"connection timeout","service":"order-service","trace_id":"a1b2c3d4"}
```

รูปแบบนี้ query ตรงๆ ได้ทันที เช่น "หา log ทั้งหมดที่ `level=error` AND `user_id=987`" โดยไม่ต้องเดา format เพราะทุก field แยกกันชัดเจนตั้งแต่ต้นทาง **Structured logging จึงเป็นข้อกำหนดเบื้องต้น (prerequisite)** ที่ทำให้ log pipeline ทั้งสายมีประโยชน์จริง — ถ้า log ยังเป็น text อิสระ ต่อให้ pipeline ข้างหลังดีแค่ไหนก็ยังต้อง parse/เดา format อยู่ดี ทีมควรบังคับใช้ structured logging library (เช่น `zerolog`, `zap` ใน Go) ตั้งแต่ระดับ application ไม่ใช่ไปหวังพึ่ง Fluent Bit filter มาแปลง text เป็น JSON ย้อนหลัง (ทำได้แต่เปราะบางกว่ามาก)

field ที่ควรมีในทุก log entry: `timestamp`, `level`, `message`, `service` (ชื่อ service), และที่สำคัญที่สุดคือ **`trace_id`** — field นี้เป็นตัวเชื่อม log เข้ากับ distributed tracing (บทที่ 3) ทำให้ค้นหา log ทั้งหมดที่เกี่ยวข้องกับ request เดียวข้าม service ได้จาก trace ID เดียว

## 2.3 Fluent Bit — Lightweight Log Collector

**Fluent Bit** เป็น log collector ที่ออกแบบมาให้เบามาก (memory footprint เล็กกว่า Fluentd ต้นตระกูลมาก, เขียนด้วย C) เหมาะกับการรันคู่กับทุก node/pod โดยไม่กินทรัพยากรมาก

### สถาปัตยกรรมแบบ Plugin: Input → Filter → Output

Fluent Bit config ประกอบด้วยสามส่วนเรียงต่อกันเป็น pipeline:

```
[INPUT]  ──►  [FILTER] (0 หรือมากกว่า)  ──►  [OUTPUT]
อ่าน log จากที่ไหน    แปลง/เติม/กรอง            ส่งไปที่ไหน
```

```ini
[INPUT]
    Name              tail
    Path              /var/log/containers/*.log
    Parser            docker
    Tag               kube.*

[FILTER]
    Name              kubernetes
    Match             kube.*
    Merge_Log         On
    Keep_Log          Off

[FILTER]
    Name              grep
    Match             kube.*
    Exclude           level debug

[OUTPUT]
    Name              opensearch
    Match             kube.*
    Host              opensearch.logging.svc.cluster.local
    Port              9200
    Index             app-logs
```

- **Input plugin** (`tail`, `systemd`, `forward`) — กำหนดว่าอ่าน log จากไหน ในตัวอย่างนี้คือ `tail` อ่านจากไฟล์ log ของ container ใน node
- **Filter plugin** (`kubernetes`, `grep`, `parser`, `record_modifier`) — ประมวลผล log ระหว่างทาง ตัวอย่างนี้ filter `kubernetes` เติม metadata (pod name, namespace, label) เข้าไปในทุก log entry อัตโนมัติ ส่วน filter `grep` กรอง log level debug ทิ้งไม่ให้เปลืองพื้นที่เก็บ
- **Output plugin** (`opensearch`, `s3`, `kafka`, `stdout`) — กำหนดปลายทาง ในที่นี้ส่งไป OpenSearch โดยตรง

โครงสร้างแบบ plugin นี้ทำให้ Fluent Bit ยืดหยุ่นมาก — สลับปลายทางจาก OpenSearch ไปเป็น S3 หรือ Kafka ได้แค่เปลี่ยน output block โดยไม่ต้องแตะ input/filter เลย

### Fluent Bit เป็น DaemonSet ใน Kubernetes

Deployment pattern ที่พบบ่อยที่สุดคือรัน Fluent Bit เป็น **DaemonSet** (เชื่อมกับ [Volume 8 — Kubernetes](../08-kubernetes/03-stateful-workloads-and-storage.md) ที่อธิบาย DaemonSet ไว้ว่าคือ workload type ที่การันตีว่ามี 1 pod รันอยู่บนทุก node เสมอ) — เหตุผลที่ DaemonSet เหมาะกับงานนี้มาก:

```
Node 1                    Node 2                    Node 3
┌──────────────────┐      ┌──────────────────┐      ┌──────────────────┐
│ Pod A  Pod B      │      │ Pod C  Pod D      │      │ Pod E  Pod F      │
│  │      │         │      │  │      │         │      │  │      │         │
│  ▼      ▼         │      │  ▼      ▼         │      │  ▼      ▼         │
│ /var/log/containers│      │ /var/log/containers│      │ /var/log/containers│
│        │           │      │        │           │      │        │           │
│  [Fluent Bit] ◄────┘      │  [Fluent Bit] ◄────┘      │  [Fluent Bit] ◄────┘
│   (DaemonSet pod)         │   (DaemonSet pod)         │   (DaemonSet pod)
└─────────┬─────────┘      └─────────┬─────────┘      └─────────┬─────────┘
          └──────────────────────────┴──────────────────────────┘
                                      ▼
                              OpenSearch cluster
```

pod ที่รันงานจริง (Pod A-F) เขียน log ออกทาง `stdout`/`stderr` ตามธรรมเนียมของ container — container runtime เขียนต่อไปยังไฟล์บน node นั้นๆ (`/var/log/containers/*.log`) Fluent Bit ที่รันเป็น DaemonSet บน**ทุก node**จึงเห็น log ของ**ทุก pod ที่รันอยู่บน node นั้น**โดยอัตโนมัติ ไม่ว่าจะมี pod ใหม่ถูกสร้างหรือลบไปกี่ตัวก็ตาม — ไม่ต้องไปตามติดตั้ง log collector แยกในทุกแอปเอง (sidecar pattern ก็ทำได้แต่กินทรัพยากรกว่ามากเพราะสร้างสำเนา collector ต่อ pod แทนที่จะเป็นต่อ node)

## 2.4 OpenSearch — เก็บและค้นหา Log

**OpenSearch** เป็น fork ของ Elasticsearch (แยกตัวออกมาหลัง Elastic เปลี่ยน license) ทำหน้าที่เก็บ log จำนวนมหาศาลและค้นหาได้เร็วระดับ sub-second แม้ข้อมูลจะมีหลายพันล้าน entry

### Index และ Mapping

**Index** ใน OpenSearch เทียบได้คร่าวๆ กับ "database/table" ใน RDBMS — เป็นที่รวมของ document (แต่ละ log entry คือ 1 document) ที่มีโครงสร้างคล้ายกัน โดยทั่วไป log จะถูกแบ่ง index ตามวัน (เช่น `app-logs-2026.08.01`) เพื่อให้ลบข้อมูลเก่าทิ้งได้ง่าย (ลบทั้ง index ทีเดียว เร็วกว่าลบ document ทีละตัวมาก) และควบคุมขนาดต่อ index ไม่ให้ใหญ่เกินไป

**Mapping** คือ schema ของ index — กำหนดว่าแต่ละ field เป็น type อะไร (`text` สำหรับค้นหาแบบ full-text, `keyword` สำหรับ exact match/aggregation, `date`, `long`) การกำหนด mapping ให้ถูกต้องตั้งแต่ต้นสำคัญมาก เพราะ field ที่ควร filter แบบ exact match (เช่น `service`, `level`, `trace_id`) ถ้าถูก map เป็น `text` (ซึ่งจะถูก tokenize/วิเคราะห์คำ) จะ query ผิดพลาดหรือช้ากว่าที่ควร

```json
PUT /app-logs-2026.08.01
{
  "mappings": {
    "properties": {
      "timestamp": { "type": "date" },
      "level":     { "type": "keyword" },
      "service":   { "type": "keyword" },
      "trace_id":  { "type": "keyword" },
      "message":   { "type": "text" },
      "user_id":   { "type": "long" }
    }
  }
}
```

### Query DSL พื้นฐาน

OpenSearch ให้ query ผ่าน **Query DSL** (JSON-based)

```json
GET /app-logs-*/_search
{
  "query": {
    "bool": {
      "must": [
        { "match": { "message": "connection timeout" } }
      ],
      "filter": [
        { "term":  { "level": "error" } },
        { "term":  { "service": "order-service" } },
        { "range": { "timestamp": { "gte": "now-1h" } } }
      ]
    }
  },
  "sort": [{ "timestamp": "desc" }]
}
```

query นี้หา log ที่ message มีคำว่า "connection timeout" (full-text match) **และ** `level=error` **และ** `service=order-service` **และ**เกิดขึ้นภายใน 1 ชั่วโมงล่าสุด — สังเกตความต่างระหว่าง `match` (full-text search, วิเคราะห์คำ) กับ `term`/`filter` (exact match, ไม่วิเคราะห์คำ, เร็วกว่าและ cache ได้) การใส่เงื่อนไข exact-match ไว้ใน `filter` แทน `must` ยังช่วยให้ OpenSearch cache ผลลัพธ์และข้าม scoring (คำนวณความเกี่ยวข้อง) ที่ไม่จำเป็นสำหรับเงื่อนไขแบบ exact ได้ด้วย ทำให้เร็วขึ้น

## 2.5 OpenSearch Dashboards — Visualize Log

**OpenSearch Dashboards** (fork ของ Kibana) เป็น UI สำหรับสำรวจและสร้าง visualization จาก log ที่เก็บใน OpenSearch — engineer พิมพ์ query แบบง่าย (Discover tab) เพื่อไล่ดู log ดิบระหว่าง debug, หรือสร้าง dashboard สรุป เช่น กราฟจำนวน error ต่อ service ตามเวลา, ตาราง top error message ที่เจอบ่อยที่สุด

ความแตกต่างจาก Grafana (บทที่ 1): Grafana เป็นเครื่องมือ visualize แบบ data-source-agnostic (เชื่อมได้หลายแหล่งพร้อมกัน) ในขณะที่ OpenSearch Dashboards ผูกกับ OpenSearch โดยเฉพาะแต่มีความสามารถด้าน full-text search และ log exploration ที่ลึกกว่า Grafana มาก (เพราะออกแบบมาเพื่องานนี้โดยเฉพาะ) — หลายทีมจึงใช้ทั้งคู่คู่กัน: Grafana สำหรับภาพรวม dashboard ข้าม metrics/logs, OpenSearch Dashboards สำหรับดำดิ่งไปดู log ดิบตอน debug

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| Log Pipeline | Collect → Parse/Filter → Ship → Index → Search |
| Structured Logging | เป็น prerequisite ที่ทำให้ pipeline มีประโยชน์จริง — JSON query ได้ตรงๆ ไม่ต้องเดา format |
| Fluent Bit | lightweight collector, input/filter/output plugin model |
| Fluent Bit เป็น DaemonSet | 1 pod ต่อ node เก็บ log ของทุก pod บน node นั้นอัตโนมัติ ไม่ต้องติดตั้งต่อแอป |
| OpenSearch Index/Mapping | index แบ่งตามวันเพื่อลบง่าย, mapping กำหนด type ให้ query ถูกและเร็ว |
| Query DSL | `match` = full-text (วิเคราะห์คำ), `term`/`filter` = exact match (เร็วกว่า, cache ได้) |
| OpenSearch Dashboards | เก่งด้าน log exploration/full-text เฉพาะทางกว่า Grafana |

## คำถามทบทวน

1. ทำไม structured logging (JSON) ถึงเป็นเงื่อนไขที่ทำให้ log pipeline มีประโยชน์จริง อธิบายด้วยตัวอย่าง query ที่ทำไม่ได้ถ้า log เป็น text ธรรมดา
2. อธิบายว่าทำไม Fluent Bit ที่รันเป็น DaemonSet ถึงเก็บ log จากทุก pod บน node ได้โดยไม่ต้องตั้งค่าต่อแอป
3. Input, Filter, Output plugin ใน Fluent Bit แต่ละอันทำหน้าที่อะไร ยกตัวอย่างพลักอินแต่ละประเภท
4. ทำไม field อย่าง `service` หรือ `level` ควร map เป็น `keyword` ไม่ใช่ `text` ใน OpenSearch mapping?
5. `match` กับ `term` ใน Query DSL ต่างกันอย่างไร และทำไมการใส่เงื่อนไข exact-match ไว้ใน `filter` ถึงเร็วกว่า?

---

ก่อนหน้า: [บทที่ 1 — Prometheus และ Grafana](01-metrics.md) | ถัดไป: [บทที่ 3 — OpenTelemetry และ Jaeger](03-tracing.md)
