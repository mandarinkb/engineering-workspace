# Engineering Workspace

พื้นที่เก็บเนื้อหาการเรียนรู้ แล็บฝึกปฏิบัติ และโปรเจกต์ สำหรับสายงาน **Backend & Infrastructure Engineering** ระดับ Enterprise ครอบคลุมตั้งแต่พื้นฐานคอมพิวเตอร์ไปจนถึงการออกแบบระบบขนาดใหญ่

หลักสูตรต้นฉบับ: [Enterprise_Backend_Infrastructure_Curriculum.md](Enterprise_Backend_Infrastructure_Curriculum.md)

## โครงสร้างโฟลเดอร์

| โฟลเดอร์ | ใช้ทำอะไร |
|---|---|
| [`books/`](books/) | เนื้อหาความรู้ภาคทฤษฎี แบ่งเป็น 17 เล่มตามหัวข้อ |
| [`labs/`](labs/) | แล็บฝึกปฏิบัติจริง (hands-on) แยกตามเทคโนโลยี |
| [`source-code/`](source-code/) | โค้ดตัวอย่าง/สนิปเป็ตประกอบการเรียน แยกตามภาษา |
| [`projects/`](projects/) | โปรเจกต์จริงขนาดใหญ่ที่รวมความรู้จากหลายเล่มเข้าด้วยกัน |
| [`diagrams/`](diagrams/) | ไดอะแกรมประกอบ (architecture, sequence, ER, network) |
| [`scripts/`](scripts/) | สคริปต์อัตโนมัติสำหรับ setup/provision/utility |
| [`resources/`](resources/) | ลิงก์ เอกสารอ้างอิง cheatsheet จากภายนอก |

## ลำดับการเรียน

ดูแผนการเรียนแบบเรียงลำดับพร้อม checklist ได้ที่ [ROADMAP.md](ROADMAP.md)

## Books ทั้งหมด

1. [Computer Foundation](books/01-computer-foundation/README.md)
2. [Network](books/02-network/README.md)
3. [Linux & Git](books/03-linux/README.md)
4. [Golang](books/04-golang/README.md)
5. [PostgreSQL](books/05-postgresql/README.md)
6. [Redis](books/06-redis/README.md)
7. [Docker](books/07-docker/README.md)
8. [Kubernetes](books/08-kubernetes/README.md)
9. [Authentication](books/09-authentication/README.md)
10. [APISIX](books/10-apisix/README.md)
11. [Microservices](books/11-microservices/README.md)
12. [Kafka](books/12-kafka/README.md)
13. [Temporal](books/13-temporal/README.md)
14. [Observability](books/14-observability/README.md)
15. [Security](books/15-security/README.md)
16. [CI/CD](books/16-cicd/README.md)
17. [System Design](books/17-system-design/README.md)

## Final Project

**Enterprise E-commerce** — ระบบ e-commerce ระดับ enterprise ที่ใช้ Golang, Kubernetes, APISIX, PostgreSQL, Redis, Kafka, Temporal, OpenSearch, Prometheus, Grafana, GitLab, Jenkins, Harbor ทำงานร่วมกันทั้งหมด ดูรายละเอียดที่ [projects/ecommerce/](projects/ecommerce/README.md)
