# บทที่ 2: Service Discovery

## 2.1 ปัญหาที่ Service Discovery แก้

ใน monolith แบบดั้งเดิม การเรียกฟังก์ชันคือการเรียกในหน่วยความจำเดียวกัน ไม่มีคำถามว่า "โค้ดอีกส่วนอยู่ที่ไหน" — แต่พอแตกเป็น microservices แต่ละ service กลายเป็น process แยก รันอยู่บนเครื่อง (หรือ container) ที่มี **IP address ไม่คงที่**:

- **Autoscaling** — ระบบเพิ่ม/ลดจำนวน instance ของ service ตาม load ตลอดเวลา instance ใหม่เกิดขึ้นพร้อม IP ใหม่ instance เก่าถูกฆ่าทิ้งพร้อม IP ที่หายไป
- **Container scheduling** — orchestrator (เช่น Kubernetes, เชื่อมกับ [Volume 8 — Kubernetes](../08-kubernetes/01-workloads-and-networking.md)) ย้าย container ไปมาระหว่าง node ตาม resource ที่ว่าง IP ของ pod เปลี่ยนทุกครั้งที่ restart
- **Deployment/rolling update** — ตอน deploy version ใหม่ instance เก่าถูกทยอยแทนที่ด้วยตัวใหม่ที่มี IP คนละตัว

ถ้า service A hardcode IP ของ service B ไว้ในโค้ดหรือ config ทุกครั้งที่ B มีการ scale/restart/deploy A จะเรียกไม่ถึง B ทันที **Service Discovery** คือกลไกที่ทำให้ service หา "ที่อยู่ปัจจุบัน" ของ service อื่นได้แบบไดนามิก โดยไม่ต้อง hardcode IP

## 2.2 Client-side Discovery vs Server-side Discovery

มีสองรูปแบบหลักในการแก้ปัญหานี้ ต่างกันตรงที่ "ใคร" เป็นคนถามและตัดสินใจว่าจะไปที่ instance ไหน

### Client-side Discovery

Client (ตัว service ที่เรียก) ถาม **service registry** เองโดยตรงว่า "instance ของ service B มีตัวไหนบ้าง" แล้วเลือก instance เอง (มักใช้ load balancing algorithm ฝังอยู่ใน client library เช่น round-robin):

```
                    ┌──────────────────┐
                    │  Service Registry │
                    └─────────┬─────────┘
                    query      │      register/heartbeat
              ┌────────────────┴────────────────┐
              │                                  │
              ▼                                  │
     ┌─────────────┐   pick instance    ┌───────────────┐
     │  Service A   │ ──────────────►   │  Service B     │
     │  (client)    │   call directly   │  (instance #2) │
     └─────────────┘                    └───────────────┘
```

ข้อดี: ไม่มี network hop เพิ่ม (เรียกตรงถึง instance ที่เลือก), client ควบคุม load balancing logic ได้เอง (เช่น เลือกตาม latency ที่วัดได้จริง)
ข้อเสีย: ทุกภาษา/ทุก service ต้อง implement discovery logic เอง (หรือใช้ library ร่วมกัน) เพิ่มความซับซ้อนให้ client

### Server-side Discovery

Client ไม่ต้องรู้เรื่อง registry เลย แค่ยิง request ไปที่ **load balancer/router** ตัวกลางตัวเดียว (เช่น ผ่าน DNS name คงที่) แล้วตัวกลางนั้นเป็นคนถาม registry และกระจาย request ไปยัง instance ที่ถูกต้องเอง:

```
     ┌─────────────┐        ┌──────────────────┐
     │  Service A   │──────►│  Load Balancer /   │──────► Service B (instance #1, #2, #3, ...)
     │  (client)    │  call  │  Router (เช่น K8s   │
     └─────────────┘  แบบ   │  Service / Ingress) │
                       ง่าย  └─────────┬──────────┘
                                       │ query
                                       ▼
                             ┌──────────────────┐
                             │  Service Registry │
                             └──────────────────┘
```

ข้อดี: client เรียบง่ายมาก (แค่ยิงไป endpoint เดียวเหมือน call service ปกติ ไม่ต้องรู้เรื่อง discovery เลย) เขียน client ภาษาอะไรก็ได้โดยไม่ต้องมี discovery library
ข้อเสีย: มี network hop เพิ่มขึ้นหนึ่งจุด (ผ่าน load balancer ก่อนถึง instance จริง) และตัวกลางกลายเป็นจุดที่ต้อง scale/ดูแลเพิ่ม (แม้ใน K8s มักไม่ใช่ปัญหาเพราะ kube-proxy ทำงานแบบ distributed อยู่แล้ว ไม่ได้เป็น single point of failure จริงๆ)

## 2.3 กลไกจริง: DNS-based vs Registry-based

### DNS-based Discovery (แบบ Kubernetes Service)

ใน Kubernetes, **Service** object สร้าง DNS name คงที่ (เช่น `order-service.default.svc.cluster.local`) ที่ resolve ไปยัง virtual IP (ClusterIP) เดียว โดย `kube-proxy` เป็นตัวจัดการ routing จาก virtual IP นั้นไปยัง pod จริงที่ healthy อยู่เบื้องหลัง (ดูรายละเอียดกลไก networking ใน [Volume 8 — Kubernetes: Workloads and Networking](../08-kubernetes/01-workloads-and-networking.md))

```
Service A ──► DNS lookup "order-service" ──► ClusterIP (คงที่)
                                                    │
                                    kube-proxy route ไปยัง pod จริง
                                                    ▼
                                    Pod #1 (10.244.1.5) / Pod #2 (10.244.2.9) / ...
```

นี่คือรูปแบบผสมของ server-side discovery: client แค่ resolve DNS name คงที่และเรียกไปที่ virtual IP เดียว ส่วน kube-proxy (และ Endpoints/EndpointSlice ที่ควบคุมอัตโนมัติโดย K8s control plane) เป็นคนจัดการว่า pod ไหนยัง healthy อยู่บ้าง — เมื่อ pod ตายหรือ scale ขึ้นใหม่ K8s อัปเดต Endpoints ให้อัตโนมัติ **โดยที่ client ไม่ต้องรู้เรื่องนี้เลย**

ข้อดีของแนวทางนี้คือความเรียบง่าย — ใช้ DNS ที่มีอยู่แล้วทุกที่ ไม่ต้องติดตั้ง component เพิ่ม ทุกภาษาที่ resolve DNS ได้ก็ใช้งานได้ทันที

### Registry-based Discovery (แบบ Consul)

รูปแบบเก่ากว่าที่นิยมก่อนยุค Kubernetes ครองตลาด (และยังใช้ในระบบที่ไม่ได้รันบน K8s ทั้งหมด หรือต้องการ discovery ข้าม cluster/datacenter) — แต่ละ service instance ต้อง**ลงทะเบียนตัวเองอย่างชัดเจน** กับ registry (เช่น Consul, Eureka):

1. **Register** — instance บอก registry ว่า "ฉันคือ order-service instance นี้ อยู่ที่ IP:port นี้" ตอน startup
2. **Heartbeat** — instance ส่งสัญญาณบอกว่า "ฉันยังมีชีวิตอยู่" เป็นระยะ (เช่นทุก 10 วินาที) ถ้า registry ไม่ได้รับ heartbeat ภายในเวลาที่กำหนด จะถือว่า instance นั้นตายแล้วและลบออกจากรายการ
3. **Deregister** — ตอน shutdown แบบปกติ (graceful shutdown) instance ควรแจ้ง registry ให้ลบตัวเองออกทันที ไม่ต้องรอ heartbeat timeout

```
Service B instance startup
       │
       ▼
   Register กับ Consul ──► Consul เก็บ: order-service → [10.0.1.5:8080, 10.0.1.9:8080]
       │
       ▼
   ส่ง heartbeat ทุก 10s ──► Consul ต่ออายุ TTL ของ entry นี้
       │
       ▼ (ถ้า instance ตายกะทันหันโดยไม่ deregister)
   heartbeat ขาดหาย > timeout ──► Consul ลบ instance ออกจากรายการอัตโนมัติ
```

Consul ยังมี **health check** แบบ active เสริมได้ (เช่น ยิง `GET /healthz` เข้าไปเช็คเอง ไม่ใช่รอ heartbeat อย่างเดียว) ทำให้ตรวจจับ instance ที่ "ยัง alive แต่ตอบสนองผิดปกติ" ได้ไวกว่าการรอ heartbeat ขาดหายเพียงอย่างเดียว

## 2.4 Trade-off ระหว่างสองแนวทาง

| ประเด็น | DNS-based (K8s Service) | Registry-based (Consul) |
|---|---|---|
| ความซับซ้อนในการตั้งค่า | ต่ำ — ได้มาพร้อม K8s ไม่ต้องติดตั้งเพิ่ม | สูงกว่า — ต้อง deploy/ดูแล registry cluster เอง |
| ความเร็วในการรับรู้การเปลี่ยนแปลง | ขึ้นกับ DNS TTL/cache และรอบ sync ของ Endpoints — อาจมี delay สั้นๆ | เร็วกว่าปกติ เพราะ heartbeat/health check ถี่และ config ปรับได้ละเอียด |
| ใช้ข้าม cluster/datacenter | ทำได้ยากกว่า (K8s DNS ผูกกับ cluster เดียวโดย default) | ออกแบบมาให้รองรับ multi-datacenter ได้ดีกว่า |
| Metadata เพิ่มเติม (version, region, weight) | จำกัด — DNS คืนแค่ IP | ยืดหยุ่นกว่า — เก็บ metadata (tag, version) ไปกรอง/route ตามได้ |
| การพึ่งพา ecosystem | ผูกกับ K8s โดยตรง | ใช้ได้กับ deployment platform หลากหลาย ไม่ผูกกับ K8s |

ในทางปฏิบัติ ถ้าระบบรันบน Kubernetes อยู่แล้ว **DNS-based ผ่าน K8s Service คือค่าเริ่มต้นที่เหมาะสมที่สุด** เพราะได้มาฟรีโดยไม่ต้องเพิ่ม component ให้ดูแล ส่วน registry-based (Consul หรือคล้ายกัน) เหมาะกับสถานการณ์ที่ต้องการ discovery ข้าม cluster/on-prem ผสม cloud, หรือระบบที่ไม่ได้รันบน K8s ทั้งหมด และต้องการ metadata ที่ซับซ้อนกว่าการ route ตาม IP อย่างเดียว (เช่น route ตาม version สำหรับ canary release)

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| ปัญหาที่ต้องแก้ | IP ของ service instance เปลี่ยนตลอดเวลาจาก autoscaling/scheduling/deployment |
| Client-side discovery | client ถาม registry เองแล้วเลือก instance เอง — ไม่มี hop เพิ่มแต่ client ซับซ้อนขึ้น |
| Server-side discovery | client ยิงไปที่ router ตัวเดียว router เป็นคนหา instance ให้ — client เรียบง่ายแต่มี hop เพิ่ม |
| DNS-based (K8s) | ใช้ DNS name คงที่ + kube-proxy route ไป pod จริง เรียบง่าย ได้มาฟรีกับ K8s |
| Registry-based (Consul) | register/heartbeat/deregister ชัดเจน ยืดหยุ่นกว่า รองรับ multi-datacenter |
| เลือกใช้อย่างไร | รันบน K8s → ใช้ DNS-based เป็นค่าเริ่มต้น, ต้องการ metadata/multi-cluster ซับซ้อน → registry-based |

## คำถามทบทวน

1. ทำไม hardcode IP address ของ service อื่นในโค้ดถึงเป็นปัญหาในระบบที่ใช้ autoscaling?
2. เปรียบเทียบ client-side กับ server-side discovery ในแง่จำนวน network hop และความซับซ้อนที่ตกอยู่กับ client
3. อธิบายบทบาทของ heartbeat และ deregister ใน registry-based discovery แบบ Consul จะเกิดอะไรขึ้นถ้า instance ตายกะทันหันโดยไม่ได้ deregister?
4. ทำไมระบบที่รันบน Kubernetes อยู่แล้วส่วนใหญ่ถึงไม่จำเป็นต้องติดตั้ง service registry แยกต่างหาก?
5. ยกตัวอย่างสถานการณ์ที่ DNS-based discovery ของ K8s ไม่เพียงพอ และต้องใช้ registry-based แทน

---

ก่อนหน้า: [บทที่ 1 — REST และ gRPC](01-communication.md) | ถัดไป: [บทที่ 3 — CQRS และ Event Sourcing](03-data-patterns.md)
