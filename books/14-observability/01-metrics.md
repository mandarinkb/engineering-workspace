# บทที่ 1: Prometheus และ Grafana

## 1.1 Observability คืออะไร และทำไมต้องมี 3 เสาหลัก

**Observability** คือความสามารถของทีมที่จะ "ถามคำถามใหม่ๆ" เกี่ยวกับระบบที่กำลังรันอยู่ได้ โดยไม่ต้อง deploy โค้ดใหม่เพื่อไปเพิ่ม log หรือ debug point ก่อน — ต่างจาก **monitoring** แบบเดิมที่มักจะตั้งค่า dashboard/alert ไว้ล่วงหน้าสำหรับปัญหาที่ "รู้อยู่แล้วว่าอาจเกิด" เท่านั้น

สามเสาหลักที่ประกอบกันเป็น observability แต่ละอันตอบคำถามคนละแบบ:

| เสาหลัก | ตอบคำถาม | ตัวอย่าง |
|---|---|---|
| **Metrics** | "ระบบตอนนี้เป็นอย่างไรในภาพรวม" (aggregate, ตัวเลขตามเวลา) | error rate สูงขึ้นตั้งแต่ 14:00 |
| **Logs** | "เกิดอะไรขึ้นกับ request/event เฉพาะเจาะจงนี้" | request ไหนที่ throw exception และ error message คืออะไร |
| **Traces** | "เวลาไปอยู่ที่ไหนเมื่อ request เดินทางข้าม service" | ทำไม request นี้ถึงใช้เวลา 2 วินาที |

บทนี้เริ่มจากเสาแรก — **Metrics** — ด้วยเครื่องมือมาตรฐานอุตสาหกรรมสองตัวคือ **Prometheus** (เก็บและ query metric) และ **Grafana** (visualize)

## 1.2 Pull-based vs Push-based Metric Collection

ระบบเก็บ metric มีสองสถาปัตยกรรมหลัก:

- **Push-based** — แต่ละแอปพลิเคชัน "ส่ง" metric ออกไปหา collector เอง (เช่น StatsD, Graphite รุ่นเก่า) แอปต้องรู้ที่อยู่ของ collector และเป็นฝ่าย initiate connection
- **Pull-based** — collector (Prometheus) เป็นฝ่าย "ไปดึง" ค่า metric จากแอปเป็นระยะๆ (scrape) ผ่าน HTTP endpoint ที่แอปเปิดไว้

Prometheus เลือกใช้ **pull-based** โดยตั้งใจ ด้วยเหตุผลหลายข้อ:

1. **Failure semantics ที่ชัดเจนกว่า** — ถ้า scrape ไม่สำเร็จ Prometheus รู้ทันทีว่า "target นี้ down" (metric `up == 0`) กลายเป็นสัญญาณมีประโยชน์ในตัวมันเองโดยไม่ต้องตั้ง alert แยก ในขณะที่ push-based ถ้าแอปหยุดส่ง collector แยกไม่ออกว่า "แอป down" หรือ "เครือข่ายมีปัญหาแค่ตอนส่ง" หรือ "แอปไม่มี traffic เลยจริงๆ เลยไม่มีอะไรส่ง"
2. **ควบคุม load จากศูนย์กลาง** — Prometheus เป็นคนกำหนด scrape interval เอง (เช่นทุก 15 วินาที) ไม่ถูกแอปที่เขียนโค้ดไม่ดี spam เข้ามา ต่างจาก push-based ที่ collector ต้องรับภาระตามจังหวะที่ client กำหนด
3. **ทดสอบง่าย** — engineer สามารถ `curl` ไปที่ `/metrics` endpoint ของแอปตรงๆ เพื่อดูค่าปัจจุบันได้ทันที โดยไม่ต้องพึ่ง collector เลย
4. **เข้ากับ service discovery ได้เป็นธรรมชาติ** — Prometheus รองรับ service discovery หลายแบบ (Kubernetes API, Consul, static config) คอยถามว่า "ตอนนี้มี target ไหนอยู่บ้าง" แล้วไปดึงเองอัตโนมัติ เวลา pod ใหม่ถูกสร้างหรือถูกลบใน Kubernetes Prometheus จะปรับ target list ตามโดยไม่ต้องแก้ config ด้วยมือ

ข้อเสียของ pull-based ที่ต้องรู้ไว้: ใช้ยากกับงานที่เป็น **short-lived job** ที่ทำงานเสร็จแล้วหายไปก่อนที่ Prometheus จะมาถึงรอบ scrape (แก้ด้วย **Pushgateway** ซึ่งเป็นตัวกลางที่ job สั้นๆ push ค่าเข้าไปเก็บไว้ชั่วคราว ให้ Prometheus มา pull จาก Pushgateway อีกที) และต้องเปิด network ให้ Prometheus เข้าถึงแอปได้ (ในสถาปัตยกรรมที่ firewall เข้มงวดอาจต้องคิดเพิ่ม)

```
Push-based                          Pull-based (Prometheus)
┌────────┐                          ┌────────┐
│  App   │──push metric───►┌──────┐ │  App   │◄──scrape (GET /metrics)──┐
└────────┘                 │Collec│ └────────┘                          │
┌────────┐                 │ tor  │                                ┌────────────┐
│  App   │──push metric───►└──────┘                                │ Prometheus │
└────────┘                                                          └────────────┘
App เป็นฝ่ายเริ่ม ต้องรู้ที่อยู่ collector          Prometheus เป็นฝ่ายเริ่ม รู้ target list จาก service discovery
```

## 1.3 `/metrics` Exposition Format

แอปที่จะให้ Prometheus เก็บ metric ได้ต้องเปิด HTTP endpoint (ปกติคือ `/metrics`) ที่ตอบกลับเป็น plain text ตามฟอร์แมตที่ Prometheus กำหนด:

```
# HELP http_requests_total จำนวน HTTP request ทั้งหมดที่รับเข้ามา
# TYPE http_requests_total counter
http_requests_total{method="GET",path="/api/orders",status="200"} 15234
http_requests_total{method="GET",path="/api/orders",status="500"} 12
http_requests_total{method="POST",path="/api/orders",status="201"} 892

# HELP process_resident_memory_bytes หน่วยความจำที่ process ใช้จริง
# TYPE process_resident_memory_bytes gauge
process_resident_memory_bytes 5.2428800e+07
```

ทุก metric ประกอบด้วย **ชื่อ** (`http_requests_total`) และ **label** (`method="GET"`, `status="200"`) — label คือสิ่งที่ทำให้ Prometheus ทรงพลัง เพราะแยกและรวม (aggregate) metric ตาม dimension ต่างๆ ได้ทีหลังผ่าน PromQL โดยไม่ต้องแก้โค้ดแอปเพิ่ม แอปส่วนใหญ่ใช้ **client library** (เช่น `prometheus/client_golang` สำหรับ Go) สร้าง endpoint นี้ให้อัตโนมัติ ไม่ต้องเขียน text format เองด้วยมือ

สำหรับระบบที่ไม่ได้เขียนขึ้นเองแต่เก็บ metric ไม่ได้ (เช่น PostgreSQL, Redis, Nginx) จะใช้ **exporter** — โปรแกรมแยกที่คุยกับระบบนั้นแล้วแปลงผลลัพธ์ให้เป็น `/metrics` format ให้ Prometheus scrape ได้ (เช่น `postgres_exporter`, `redis_exporter`, `node_exporter` สำหรับ metric ระดับ OS)

## 1.4 Metric Type ทั้งสี่แบบ

### Counter

ตัวเลขที่ **เพิ่มขึ้นอย่างเดียว** (หรือ reset เป็น 0 ตอนแอป restart) ห้ามลด เหมาะกับการนับสิ่งที่สะสมไปเรื่อยๆ เช่น จำนวน request ทั้งหมด, จำนวน error ทั้งหมด

```go
var requestsTotal = prometheus.NewCounterVec(
    prometheus.CounterOpts{Name: "http_requests_total"},
    []string{"method", "status"},
)
requestsTotal.WithLabelValues("GET", "200").Inc()
```

**กับดักสำคัญ**: ค่าดิบของ Counter (`http_requests_total` ตอนนี้เท่าไหร่) แทบไม่มีประโยชน์เอาไปแสดงตรงๆ เพราะมันแค่ "นับสะสม" ไม่บอกว่าช่วงนี้เร็วหรือช้า สิ่งที่ต้องดูคือ **อัตราการเปลี่ยนแปลง** ซึ่งต้องใช้ฟังก์ชัน `rate()` ใน PromQL แปลงให้ (อธิบายในหัวข้อ 1.5)

### Gauge

ตัวเลขที่ **ขึ้นหรือลงก็ได้** ณ ขณะใดขณะหนึ่ง เหมาะกับค่าที่แทนสถานะปัจจุบัน เช่น จำนวน connection ที่เปิดอยู่ตอนนี้, ความลึกของ queue ตอนนี้, จำนวน goroutine, memory ที่ใช้อยู่

```go
var queueDepth = prometheus.NewGauge(
    prometheus.GaugeOpts{Name: "job_queue_depth"},
)
queueDepth.Set(42)
queueDepth.Inc() // เมื่อ job เข้าคิว
queueDepth.Dec() // เมื่อ job ถูกดึงออกไปทำ
```

### Histogram

เก็บการกระจายตัว (distribution) ของค่าที่วัดได้ โดยแบ่งเป็น **bucket** (ช่วง) ที่กำหนดไว้ล่วงหน้า เหมาะที่สุดสำหรับ **latency ของ request** เพราะค่าเฉลี่ย (average) เพียงอย่างเดียวหลอกตาได้ง่าย (request ส่วนใหญ่เร็วมาก แต่มี tail 1% ที่ช้ามาก ค่าเฉลี่ยจะไม่สะท้อนความเจ็บปวดของ 1% นั้นเลย) Histogram เก็บทั้ง cumulative count ต่อ bucket, sum ของค่าทั้งหมด, และ count รวม ทำให้คำนวณ **percentile** (p50, p95, p99) ย้อนหลังผ่าน PromQL ได้

```go
var requestDuration = prometheus.NewHistogramVec(
    prometheus.HistogramOpts{
        Name:    "http_request_duration_seconds",
        Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5},
    },
    []string{"path"},
)
timer := prometheus.NewTimer(requestDuration.WithLabelValues("/api/orders"))
defer timer.ObserveDuration()
```

### Summary

คล้าย Histogram ตรงที่วัด distribution เหมือนกัน แต่ต่างกันตรงที่ Summary คำนวณ percentile (เช่น p95) **ที่ฝั่งแอปเอง** ก่อนส่งออกมา (client-side quantile) ในขณะที่ Histogram ส่งแค่ raw bucket count ออกมา แล้วให้ Prometheus server เป็นคนคำนวณ percentile ทีหลังตอน query

ข้อดี/เสียตัดกัน: Summary ให้ percentile ที่แม่นยำกว่าสำหรับ instance เดียว แต่ **รวม (aggregate) ข้าม instance ไม่ได้** (p95 ของแต่ละเครื่องเอามาเฉลี่ยกันไม่ได้ทางคณิตศาสตร์) ในขณะที่ Histogram รวมข้าม instance ได้ถูกต้อง (เพราะรวม bucket count ธรรมดาแล้วค่อยคำนวณ percentile ทีหลัง) — ด้วยเหตุนี้ **Histogram จึงเป็นตัวเลือกที่นิยมกว่ามากในระบบ distributed** ที่มีหลาย instance รันพร้อมกัน (เช่น หลาย pod ใน Kubernetes) ส่วน Summary เหมาะกับกรณีที่วัดจาก instance เดียวจริงๆ และต้องการความแม่นยำสูงสุด

| Type | ขึ้น/ลงได้ไหม | ใช้กับ | ตัวอย่าง |
|---|---|---|---|
| Counter | ขึ้นอย่างเดียว | นับสะสม, ต้องใช้ `rate()` ดูอัตรา | จำนวน request ทั้งหมด |
| Gauge | ขึ้นลงได้ | ค่าสถานะปัจจุบัน | queue depth, memory ปัจจุบัน |
| Histogram | สะสมเป็น bucket | distribution, คำนวณ percentile ที่ server, รวมข้าม instance ได้ | latency ของ HTTP request |
| Summary | สะสม quantile ที่ client | distribution ต่อ instance เดียว, รวมข้าม instance ไม่ได้ | เหมือน histogram แต่ single-instance |

## 1.5 PromQL พื้นฐาน

**PromQL** คือภาษา query ของ Prometheus

### `rate()` — แปลง Counter ให้เป็นอัตรา

```promql
rate(http_requests_total[5m])
```

คำนวณ **อัตราเฉลี่ยต่อวินาที** ของการเพิ่มขึ้นของ counter ในช่วง 5 นาทีล่าสุด (ต่อ time series แยกตาม label แต่ละชุด) `rate()` ฉลาดพอที่จะจัดการกรณี counter reset เป็น 0 (เช่นแอป restart) ได้อัตโนมัติ โดยไม่ทำให้กราฟเป็นค่าลบหลอกๆ — นี่คือเหตุผลที่ **ห้ามใช้ `-` (subtraction) ธรรมดากับ counter เอง** ต้องผ่าน `rate()` เสมอ

### `sum by` — รวมข้าม label ที่ไม่สนใจ

```promql
sum by (service) (rate(http_requests_total[5m]))
```

รวมอัตรา request ของทุก path/method/status เข้าด้วยกัน เหลือแค่แยกตาม `service` — มีประโยชน์มากเวลาต้องการภาพรวมโดยไม่สนใจ dimension ย่อย

### `histogram_quantile()` — คำนวณ percentile latency

```promql
histogram_quantile(0.95,
  sum by (le, path) (rate(http_request_duration_seconds_bucket[5m]))
)
```

คำนวณ **p95 latency** (95% ของ request เร็วกว่าค่านี้) จาก histogram buckets — `le` (less than or equal) คือ label พิเศษที่ Prometheus ใส่ให้อัตโนมัติในทุก bucket ของ histogram ต้องรวม (`sum by`) โดยเก็บ `le` ไว้เสมอ ไม่งั้น `histogram_quantile()` คำนวณผิด

### ตัวอย่าง query ที่ใช้บ่อยในการหา error rate

```promql
sum(rate(http_requests_total{status=~"5.."}[5m]))
  /
sum(rate(http_requests_total[5m]))
```

สัดส่วน error (status 5xx) ต่อ request ทั้งหมดในช่วง 5 นาทีล่าสุด — pattern แบบนี้ (errors / total) เป็นหัวใจของการทำ SLO-based alerting

## 1.6 Alertmanager

Prometheus server เองมีหน้าที่แค่ **ประเมิน alert rule** ว่าเงื่อนไข true/false — ส่วนการจัดการว่าจะแจ้งเตือนใคร ทางไหน (Slack, email, PagerDuty), จัดกลุ่ม, และปิดเสียงชั่วคราว เป็นหน้าที่ของ **Alertmanager** ซึ่งเป็น component แยก

### Alert Rule

```yaml
groups:
  - name: api-alerts
    rules:
      - alert: HighErrorRate
        expr: |
          sum(rate(http_requests_total{status=~"5.."}[5m]))
            / sum(rate(http_requests_total[5m])) > 0.05
        for: 10m
        labels:
          severity: critical
        annotations:
          summary: "Error rate สูงกว่า 5% นานกว่า 10 นาที"
```

`for: 10m` สำคัญมาก — ทำให้ alert ไม่ยิงทันทีที่ query เป็น true ครั้งเดียว (ป้องกัน false alarm จาก spike ชั่ววูบ) ต้อง true ต่อเนื่องครบ 10 นาทีก่อนถึงจะเปลี่ยนสถานะเป็น `firing` และส่งต่อไปที่ Alertmanager

### Routing และ Grouping

Alertmanager รับ alert ที่ firing จาก Prometheus แล้วตัดสินใจว่าจะส่งไปทางไหนตาม label:

```yaml
route:
  receiver: default-slack
  group_by: ['alertname', 'service']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h
  routes:
    - match:
        severity: critical
      receiver: pagerduty-oncall
receivers:
  - name: default-slack
    slack_configs:
      - channel: '#alerts'
  - name: pagerduty-oncall
    pagerduty_configs:
      - service_key: '...'
```

- **`group_by`** — รวม alert หลายตัวที่มี label ตรงกันเป็นการแจ้งเตือนเดียว (เช่น ถ้า 20 pod ของ service เดียวกัน error พร้อมกันหมด ไม่อยากได้ 20 ข้อความแยก อยากได้ข้อความเดียวสรุปรวม)
- **`group_wait`** — รอสักพักก่อนส่งครั้งแรก เผื่อมี alert อื่นในกลุ่มเดียวกันตามมาจะได้รวมกัน
- **`repeat_interval`** — ถ้า alert ยัง firing อยู่ต่อเนื่อง จะเตือนซ้ำทุกกี่ชั่วโมง (ไม่ใช่ยิงรัวๆ ทุกครั้งที่ evaluate)

### Silencing

เมื่อรู้อยู่แล้วว่ากำลังมีการ maintenance หรือกำลังแก้ปัญหาอยู่ สามารถสั่ง **silence** alert ที่ match label เงื่อนไขหนึ่งๆ ไว้ชั่วคราว (มี timeout) เพื่อไม่ให้ทีมถูกปลุกซ้ำๆ ด้วยปัญหาที่รู้อยู่แล้วและกำลังจัดการอยู่ — ต่างจากการปิด alert rule ทิ้งถาวรตรงที่ silence จะหมดอายุอัตโนมัติ ลดความเสี่ยงที่จะลืมเปิดกลับ

## 1.7 Grafana — Data Source, Dashboard, Panel

Grafana ไม่ได้เก็บ metric เอง แต่เป็น **ชั้น visualization** ที่ query จาก data source ภายนอก — เชื่อมได้กับ Prometheus, OpenSearch (บทที่ 2), MySQL, CloudWatch และอื่นๆ พร้อมกันในระบบเดียว ทำให้ dashboard เดียวผสมข้อมูลจากหลายแหล่งได้

### เชื่อม Data Source

ตั้งค่าใน Grafana ครั้งเดียวว่า Prometheus server อยู่ที่ URL ไหน จากนั้นทุก panel ที่สร้างขึ้นสามารถเลือก data source นี้แล้วเขียน PromQL ได้ตรงในตัว query editor

### Dashboard และ Panel

**Dashboard** คือหน้ารวมของหลาย **Panel** — panel แต่ละอันคือ query หนึ่งชุด (หรือมากกว่า) บวกกับวิธีแสดงผล ประเภท panel ที่ใช้บ่อย:

- **Time series (graph)** — เส้นกราฟตามเวลา เหมาะกับ rate/latency
- **Stat** — ตัวเลขเดี่ยวใหญ่ๆ เหมาะกับสรุปค่าปัจจุบัน (เช่น error rate ตอนนี้)
- **Gauge** — เข็มวัด เหมาะกับค่าที่มี threshold ชัดเจน (เช่น disk usage %)
- **Table** — ตารางข้อมูล เหมาะกับการเทียบหลาย series พร้อมกัน (เช่น latency ต่อ endpoint)
- **Heatmap** — เหมาะกับแสดง distribution ของ histogram ตามเวลา (เห็นการเปลี่ยนแปลงของ latency distribution)

Dashboard ที่ดีมักออกแบบตามหลัก **RED method** (Rate, Errors, Duration — สำหรับ service ที่รับ request) หรือ **USE method** (Utilization, Saturation, Errors — สำหรับ resource เช่น CPU/disk) เพื่อไม่ให้ panel เยอะจนหาอะไรไม่เจอตอน incident

### Grafana Alerting vs Alertmanager — เลือกใช้ตัวไหน

ทั้งสองระบบทำ "alert" ได้เหมือนกัน แต่ตอบโจทย์คนละสถานการณ์:

| | Alertmanager | Grafana Alerting |
|---|---|---|
| แหล่งข้อมูล | ผูกกับ Prometheus โดยตรง (alert rule ประเมินโดย Prometheus) | query ได้จากทุก data source ที่ Grafana เชื่อมไว้ (Prometheus, OpenSearch, SQL ฯลฯ) |
| การจัดการ routing/grouping/silence | ทรงพลังกว่า ออกแบบมาเพื่อสิ่งนี้โดยเฉพาะ | มีแต่พื้นฐานกว่า |
| เหมาะกับ | ทีมที่ alert ส่วนใหญ่มาจาก metric ของ Prometheus และต้องการ routing ซับซ้อน (on-call ต่าง team ต่าง severity) | ทีมที่ต้องการ alert ข้าม data source เดียวกับที่ดู dashboard อยู่แล้ว หรือทีมเล็กที่ไม่อยากดูแล component เพิ่ม |

ในทางปฏิบัติ หลายทีมใช้ **Alertmanager เป็นหลักสำหรับ alert ที่มาจาก metric** (เพราะ routing ยืดหยุ่นกว่ามาก) และใช้ **Grafana Alerting เสริมเฉพาะกรณีที่ query ข้าม data source ที่ Alertmanager เข้าไม่ถึง** เช่น alert จาก log-based metric ใน OpenSearch

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| Pull vs Push | Prometheus เลือก pull เพราะ failure semantics ชัดเจนกว่า (`up==0`), คุม load ได้, เข้ากับ service discovery |
| Counter | เพิ่มอย่างเดียว ต้องใช้ `rate()` ดูอัตรา ห้ามอ่านค่าดิบตรงๆ |
| Gauge | ขึ้นลงได้ แทนค่าสถานะปัจจุบัน |
| Histogram vs Summary | Histogram รวมข้าม instance ได้ (นิยมกว่าในระบบ distributed), Summary คำนวณ percentile ที่ client แม่นกว่าแต่รวมไม่ได้ |
| PromQL | `rate()` แปลง counter เป็นอัตรา, `sum by` รวมข้าม label, `histogram_quantile()` คำนวณ percentile |
| Alertmanager | แยกหน้าที่จาก Prometheus — จัดการ routing/grouping/silence ของ alert ที่ firing แล้ว |
| Grafana | ชั้น visualization ที่ query ได้จากหลาย data source, alerting มีในตัวแต่ routing สู้ Alertmanager ไม่ได้ |

## คำถามทบทวน

1. อธิบายว่าทำไม pull-based model ทำให้ Prometheus รู้ได้ทันทีว่า target หนึ่ง "down" ในขณะที่ push-based model แยกแยะสถานการณ์นี้ได้ยากกว่า
2. ทำไมการอ่านค่า Counter ดิบๆ (ไม่ผ่าน `rate()`) แล้วเอาไปแสดงบนกราฟถึงไม่มีประโยชน์ ยกตัวอย่างประกอบ
3. ทำไม Histogram ถึงเป็นตัวเลือกที่นิยมกว่า Summary ในระบบที่มีหลาย instance รันพร้อมกัน (เช่นใน Kubernetes)?
4. อธิบายความแตกต่างระหว่าง `for` ใน alert rule กับ `group_wait`/`repeat_interval` ใน Alertmanager — แต่ละตัวแก้ปัญหาอะไร?
5. ในสถานการณ์ไหนที่ควรใช้ Grafana Alerting แทน Alertmanager?

---

ถัดไป: [บทที่ 2 — Fluent Bit และ OpenSearch](02-logging.md)
