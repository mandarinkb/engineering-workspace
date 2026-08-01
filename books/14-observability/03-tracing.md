# บทที่ 3: OpenTelemetry และ Jaeger

## 3.1 ทำไมต้องมี Distributed Tracing

Metrics (บทที่ 1) บอกว่า "p95 latency ของ API นี้แย่ลง" และ Logs (บทที่ 2) บอกว่า "request นี้ error ด้วยข้อความอะไร" แต่ทั้งสองอย่างตอบไม่ได้ชัดเจนพอสำหรับคำถามที่สำคัญที่สุดในระบบ microservices: **"เวลาของ request หนึ่งไปหมดไปกับที่ไหนบ้าง เมื่อมันเดินทางผ่าน service A → B → C → D"**

ถ้า service A เรียก B, B เรียก C และ D พร้อมกัน แล้ว request รวมใช้เวลา 2 วินาที — ปัญหาช้าอยู่ที่ B เอง หรืออยู่ที่ C หรือ D หรืออยู่ที่ network ระหว่างทาง? Log ของแต่ละ service แยกกันอยู่คนละไฟล์ ไม่มีอะไรมาเรียงลำดับเวลาและความสัมพันธ์ของการเรียกให้เห็นภาพรวม — **Distributed Tracing** แก้ปัญหานี้โดยตรง

## 3.2 OpenTelemetry — มาตรฐานกลางสำหรับ Instrumentation

ก่อนจะมี OpenTelemetry แต่ละ vendor observability (Jaeger, Zipkin, Datadog, New Relic) มี SDK/format ของตัวเองที่ไม่เข้ากัน ทีมที่อยากเปลี่ยน vendor ต้องเขียน instrumentation ใหม่ทั้งหมด — **OpenTelemetry (OTel)** คือมาตรฐานกลางที่แก้ปัญหานี้ โดยรวม CNCF project สองตัวเดิม (OpenTracing + OpenCensus) เข้าด้วยกัน ให้ instrumentation เขียนครั้งเดียวแล้วส่งออกไปที่ backend ไหนก็ได้

OTel ครอบคลุมทั้งสามเสาหลักของ observability (metrics, logs, traces) ด้วย API/SDK ชุดเดียวกัน — บทนี้เน้นส่วน tracing เป็นหลัก

### สถาปัตยกรรม: SDK → Collector → Exporter

```
┌──────────────┐        ┌──────────────┐        ┌──────────────┐
│  Service A   │        │  Service B   │        │  Service C   │
│ (OTel SDK)   │        │ (OTel SDK)   │        │ (OTel SDK)   │
└──────┬───────┘        └──────┬───────┘        └──────┬───────┘
       │ OTLP (gRPC/HTTP)      │                        │
       └────────────┬──────────┴────────────────────────┘
                     ▼
           ┌───────────────────┐
           │ OpenTelemetry      │   รับ, ประมวลผล (batch, filter,
           │ Collector          │   sample, เติม attribute), ส่งต่อ
           └─────────┬──────────┘
                      │ Exporter (ปรับ format ให้ backend เข้าใจ)
        ┌─────────────┼─────────────┐
        ▼             ▼             ▼
    ┌────────┐   ┌──────────┐  ┌──────────┐
    │ Jaeger │   │Prometheus│  │OpenSearch│
    └────────┘   └──────────┘  └──────────┘
```

- **SDK** — library ที่ฝังในโค้ดแอป สร้าง span/metric/log ตาม API มาตรฐานของ OTel แล้วส่งออกด้วยโปรโตคอล **OTLP** (OpenTelemetry Protocol)
- **Collector** — process กลางที่รับข้อมูลจากทุกแอป ทำหน้าที่ batch (รวมส่งเป็นชุดแทนส่งทีละอัน ลด overhead), filter, sample (เก็บ trace แค่บางเปอร์เซ็นต์เมื่อ traffic สูงมาก เพื่อลดปริมาณข้อมูล), และเติม metadata ก่อนส่งต่อ — สำคัญคือ Collector ทำให้แอปไม่ต้องรู้ปลายทางสุดท้ายเลย แอปส่งเข้า Collector ที่เดียว ส่วน Collector เป็นคนตัดสินใจว่าจะส่งต่อไปที่ไหน (เปลี่ยน backend จาก Jaeger เป็นตัวอื่นได้โดยไม่ต้องแก้โค้ดแอปแม้แต่บรรทัดเดียว)
- **Exporter** — component ใน Collector (หรือใน SDK โดยตรงถ้าไม่ผ่าน Collector) ที่แปลงข้อมูลรูปแบบ OTel ให้เป็นรูปแบบที่ backend ปลายทางเข้าใจ (Jaeger format, Prometheus remote-write ฯลฯ)

### Auto-instrumentation vs Manual Instrumentation

- **Auto-instrumentation** — ใช้ library/agent ที่ hook เข้ากับ framework ที่รู้จักอยู่แล้วโดยอัตโนมัติ (เช่น HTTP router, database driver) สร้าง span ให้เองโดยแทบไม่ต้องแก้โค้ด ข้อดีคือเริ่มได้เร็วมาก ครอบคลุม boundary มาตรฐาน (HTTP request เข้า-ออก, DB query) ได้ทันที ข้อเสียคือมองไม่เห็น business logic ภายในฟังก์ชันของทีมเอง (เช่น ไม่รู้ว่าใน handler ตัวหนึ่งมีการวนลูปคำนวณหนักตรงไหน)
- **Manual instrumentation** — engineer เขียนโค้ดสร้าง span เองตรงจุดที่ auto-instrumentation มองไม่เห็น

```go
tracer := otel.Tracer("order-service")

func ProcessOrder(ctx context.Context, orderID string) error {
    ctx, span := tracer.Start(ctx, "ProcessOrder")
    defer span.End()

    span.SetAttributes(attribute.String("order.id", orderID))

    if err := validateInventory(ctx, orderID); err != nil {
        span.RecordError(err)
        return err
    }
    return nil
}
```

ในทางปฏิบัติทีมส่วนใหญ่ใช้ **ทั้งสองแบบผสมกัน** — เปิด auto-instrumentation เป็น baseline ให้เห็น request/DB call ทั้งหมดฟรีๆ แล้วเสริม manual instrumentation เฉพาะจุดที่เป็น business-critical หรือมักเป็นจุดที่ latency สูงผิดปกติ

## 3.3 Span, Trace, และ Parent-Child Relationship

### Span

**Span** คือหน่วยที่เล็กที่สุดของการทำงานที่ถูกวัด — มี:

- **ชื่อ** (เช่น `ProcessOrder`, `SELECT orders`)
- **เวลาเริ่ม/จบ** (start time, end time) → คำนวณ duration ได้
- **Attribute** — key-value เพิ่มเติม (เช่น `http.status_code`, `db.statement`)
- **Span ID** — ตัวระบุ span นี้โดยเฉพาะ
- **Parent Span ID** — ระบุว่า span นี้เกิดขึ้นระหว่างการทำงานของ span ไหน (ถ้าเป็น span แรกสุดของ trace จะไม่มี parent — เรียกว่า **root span**)

### Trace

**Trace** คือ**ต้นไม้ของ span** ทั้งหมดที่เกี่ยวข้องกับ request เดียวกัน ทุก span ในต้นไม้เดียวกันแชร์ **Trace ID** เดียวกัน

```
Trace ID: a1b2c3d4
                              
  [API Gateway: handle request] ─────────────────────────  (root span, 0-950ms)
        │
        ├── [Order Service: ProcessOrder] ─────────────  (100-900ms, child ของ root)
        │         │
        │         ├── [Order Service: validate] ──  (110-150ms)
        │         │
        │         └── [Order Service: call Payment Service] ──────  (160-880ms)
        │                   │
        │                   └── [Payment Service: charge] ────────  (170-870ms, child ข้าม service)
        │                             │
        │                             └── [Payment Service: DB query] ──────────  (180-850ms) ← ช้าผิดปกติ!
        │
        └── [Order Service: publish event] ──  (900-920ms)
```

แต่ละ span รู้ parent ของตัวเอง ทำให้สร้างเป็นโครงสร้าง**ต้นไม้** (ไม่ใช่แค่ list แบน) ที่แสดงทั้ง **ลำดับเวลา** (start/end) และ **ความสัมพันธ์เชิงเหตุ-ผล** (ใครเรียกใคร) พร้อมกัน — นี่คือสิ่งที่ metrics/logs เดี่ยวๆ ให้ไม่ได้

## 3.4 จาก Trace ID Propagation สู่ Trace ที่สมบูรณ์

[Volume 11 — Microservices บทที่ 5](../11-microservices/05-idempotency-and-tracing.md) อธิบายไว้แล้วว่า service ต้องส่งต่อ **Trace ID** ผ่าน HTTP header (เช่น `traceparent` ตามมาตรฐาน W3C Trace Context) ทุกครั้งที่เรียก service ถัดไป เพื่อให้ log ของทุก service ที่เกี่ยวข้องกับ request เดียวกัน **grep ด้วย trace ID เดียวกันได้** — นั่นคือขั้นแรกที่จำเป็น (correlation)

สิ่งที่ทำให้กลายเป็น **trace ที่สมบูรณ์** (ไม่ใช่แค่ log ที่ correlate กันได้) คือทุก service ต้อง**สร้าง span ของตัวเอง**ด้วย (ผ่าน OTel SDK) แล้วรายงานกลับไปที่ backend เดียวกัน (ผ่าน Collector) โดยที่ span ใหม่แต่ละอันอ้างอิง **parent span ID** จาก header ที่ได้รับมา — เมื่อทุก service ในเส้นทางทำแบบนี้ backend (Jaeger) จะประกอบ span ทั้งหมดที่มี trace ID เดียวกันเข้าเป็นต้นไม้เดียวโดยอัตโนมัติ กลายเป็น **trace ที่มองเห็นทั้งเส้นทางของ request ข้าม service ได้แบบภาพเดียว**

พูดง่ายๆ: Trace ID propagation (ที่ Volume 11 สอน) คือ "เส้นด้าย" ที่ร้อยทุก service เข้าด้วยกัน ส่วน OpenTelemetry instrumentation ในบทนี้คือสิ่งที่ทำให้แต่ละ service "รายงานตัว" กลับเข้าไปในเส้นด้ายนั้น — มีแค่เส้นด้ายอย่างเดียวไม่พอ (ได้แค่ correlate log) ต้องมีทั้งคู่ถึงจะได้ trace เต็มรูปแบบ

## 3.5 Jaeger — จัดเก็บและ Visualize Trace

**Jaeger** เป็น backend สำหรับเก็บ trace ที่ส่งมาจาก OpenTelemetry Collector และให้ UI สำหรับค้นหา/ดู trace แบบ **waterfall view**

### อ่าน Trace Waterfall เพื่อหา Latency Bottleneck

สมมติมี API call ที่ผู้ใช้รายงานว่าช้า (2 วินาที) ทีม engineer เปิด Jaeger ค้นหา trace ของ request นั้นด้วย trace ID (ได้จาก log หรือ response header) แล้วเห็น waterfall แบบนี้:

```
span                                          0ms        500ms       1000ms      1500ms      2000ms
──────────────────────────────────────────── │───────────│───────────│───────────│───────────│
API Gateway: GET /orders/12345                ████████████████████████████████████████████████  (0-2000ms)
  └─ Order Service: GetOrder                    ██████████████████████████████████████████████  (50-2000ms)
       ├─ Order Service: fetch order row              ██                                          (100-150ms)
       ├─ Order Service: call Payment Service            ████████████████████████████████████     (200-1950ms)
       │    └─ Payment Service: GetPaymentHistory          ███████████████████████████████████     (250-1950ms)
       │         └─ Payment Service: DB query SELECT...      ██████████████████████████████████    (300-1900ms) ◄── ยาวที่สุด!
       └─ Order Service: format response                                                        ██  (1950-1980ms)
```

การอ่าน waterfall นี้:

1. Span บนสุด (`API Gateway`) กินเวลาเต็ม 2000ms — เป็น root span ครอบทุกอย่างไว้ (ตามธรรมชาติ เพราะทุก span ลูกเกิดขึ้นระหว่างที่ span แม่ยังไม่จบ)
2. ไล่ลงมาดูว่า **span ไหนกินสัดส่วนความกว้าง (duration) เยอะที่สุด** — ในที่นี้คือ `DB query SELECT...` ของ Payment Service ที่กิน 1600ms จาก 2000ms ทั้งหมด (80% ของเวลาทั้ง request)
3. ส่วนอื่น (`fetch order row` 50ms, `format response` 30ms) เล็กน้อยมากเทียบกัน ไม่ใช่จุดที่ต้อง optimize ก่อน

**ข้อสรุปจากตัวอย่างนี้**: request ช้าไม่ได้เกิดจาก Order Service เอง ไม่ได้เกิดจาก network ระหว่าง service ด้วย — เกิดจาก **query เดียวใน Payment Service** ที่ใช้เวลานานผิดปกติ (อาจเพราะ missing index — เชื่อมกับสิ่งที่ [Volume 5 — PostgreSQL](../05-postgresql/README.md) สอนเรื่องการอ่าน `EXPLAIN ANALYZE`) โดยไม่ต้องเดา ไม่ต้อง ssh ไล่ log ทีละ service — waterfall view ชี้จุดที่ต้องไปแก้ให้ตรงเป้าทันที นี่คือคุณค่าหลักของ distributed tracing เหนือ metrics/logs ล้วนๆ

### การ Sample Trace

ระบบที่มี traffic สูงมากไม่สามารถเก็บทุก trace ได้ (ทั้งเปลืองพื้นที่เก็บและ overhead ที่แอปต้องส่งข้อมูลออกไป) จึงมักตั้งค่า **sampling rate** (เช่นเก็บแค่ 1% ของ trace ทั้งหมด) ผ่าน OpenTelemetry Collector — กลยุทธ์ที่ดีกว่าการสุ่มล้วนๆ (`probabilistic sampling`) คือ **tail-based sampling** ที่ตัดสินใจเก็บ trace หลังจากเห็น trace ทั้งหมดจบแล้วว่ามี error หรือ latency สูงผิดปกติหรือไม่ (เก็บ trace ที่ "น่าสนใจ" ไว้เกือบทั้งหมด แต่สุ่มทิ้ง trace ปกติส่วนใหญ่) ทำให้ engineer ยังมี trace ของปัญหาจริงให้ดูครบ แม้ traffic รวมจะสูงมากก็ตาม

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| OpenTelemetry | มาตรฐานกลาง vendor-neutral สำหรับ instrumentation ทั้ง metrics/logs/traces |
| SDK / Collector / Exporter | SDK สร้างข้อมูลในแอป, Collector รวม/ประมวลผลกลาง, Exporter แปลง format ให้ backend |
| Auto vs Manual instrumentation | Auto ครอบ boundary มาตรฐานได้เร็ว, Manual จำเป็นสำหรับ business logic — ใช้ผสมกัน |
| Span | หน่วยงานเดียว มี start/end time, attribute, parent span ID |
| Trace | ต้นไม้ของ span ทั้งหมดที่มี trace ID เดียวกัน |
| Trace ID propagation vs full trace | propagation คือเส้นด้ายเชื่อม log, ต้องมี span จากทุก service ด้วยถึงจะได้ trace เต็มรูปแบบ |
| Jaeger waterfall | ไล่ดู span ที่กิน duration เยอะที่สุดเพื่อหา bottleneck ตรงจุด |

## คำถามทบทวน

1. OpenTelemetry Collector มีประโยชน์อย่างไรเทียบกับการให้ SDK ในแอปส่งข้อมูลตรงไปที่ Jaeger เอง?
2. อธิบายความสัมพันธ์ระหว่าง Span, Trace, และ Parent-Child relationship
3. ทำไม Trace ID propagation เพียงอย่างเดียว (ไม่มี OTel instrumentation) ถึงยังไม่เพียงพอที่จะได้ "trace" ที่สมบูรณ์?
4. จากตัวอย่าง waterfall ในหัวข้อ 3.5 ทำไมถึงสรุปได้ว่าปัญหาอยู่ที่ DB query ของ Payment Service ไม่ใช่ network ระหว่าง service?
5. Tail-based sampling ต่างจาก probabilistic sampling ธรรมดาอย่างไร ทำไมถึงเหมาะกับการ debug ปัญหามากกว่า?

---

ก่อนหน้า: [บทที่ 2 — Fluent Bit และ OpenSearch](02-logging.md) | กลับสู่ [สารบัญ Volume 14](README.md)
