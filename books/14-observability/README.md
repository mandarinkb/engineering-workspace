# Volume 14 — Observability

"ถ้ามองไม่เห็นระบบ ก็แก้ปัญหาระบบไม่ได้" หมวดนี้ครอบคลุมสามเสาหลักของ observability: Metrics, Logs, Traces

## เป้าหมายการเรียนรู้

- ตั้งค่า Prometheus เก็บ metric และเขียน query (PromQL) พื้นฐานได้
- สร้าง dashboard ใน Grafana ที่สื่อสารสถานะระบบได้ชัดเจน
- ออกแบบ log pipeline (collect → ship → index → search) ด้วย Fluent Bit + OpenSearch
- ใช้ OpenTelemetry/Jaeger ทำ distributed tracing เชื่อม request ข้าม service

## สารบัญ

### [บทที่ 1 — Prometheus และ Grafana](01-metrics.md)

- [ ] Pull-based metric collection, exporter, `/metrics` endpoint
- [ ] Metric type: Counter, Gauge, Histogram, Summary
- [ ] PromQL พื้นฐาน: `rate()`, `sum by`, `histogram_quantile()`
- [ ] Alertmanager — alert rule, routing, silence
- [ ] เชื่อม data source (Prometheus, OpenSearch)
- [ ] สร้าง dashboard, panel type ต่างๆ
- [ ] Alert ใน Grafana เทียบกับ Alertmanager

### [บทที่ 2 — Fluent Bit และ OpenSearch](02-logging.md)

- [ ] Log collector แบบ lightweight, input/filter/output plugin
- [ ] ใช้เป็น DaemonSet ใน K8s เก็บ log จากทุก node (เชื่อมกับ [Volume 8](../08-kubernetes/README.md))
- [ ] เก็บและค้นหา log (fork ของ Elasticsearch)
- [ ] Index, mapping, query DSL พื้นฐาน
- [ ] OpenSearch Dashboards สำหรับ visualize log

### [บทที่ 3 — OpenTelemetry และ Jaeger](03-tracing.md)

- [ ] มาตรฐานกลางสำหรับ instrumentation (metrics, logs, traces)
- [ ] SDK, Collector, Exporter
- [ ] Auto-instrumentation vs manual instrumentation
- [ ] Distributed tracing — span, trace, parent-child relationship
- [ ] เชื่อม Trace ID ข้าม service (เชื่อมกับ [Volume 11 — Microservices](../11-microservices/README.md))
- [ ] วิเคราะห์ latency bottleneck จาก trace waterfall

## Lab ที่เกี่ยวข้อง

ยังไม่มีแล็บเฉพาะในรอบนี้ — แนะนำฝึกผ่าน [projects/ecommerce/](../../projects/ecommerce/README.md) ซึ่งรวม observability stack ทั้งหมด

## แหล่งอ้างอิงแนะนำ

ดู [resources/](../../resources/README.md)
