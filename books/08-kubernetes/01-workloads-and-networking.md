# บทที่ 1: Pod, Deployment, Service, Ingress

## 1.1 Pod — หน่วยที่เล็กที่สุดใน Kubernetes

### ทำไมไม่ใช้แค่ "Container" ตรงๆ

Kubernetes ไม่ deploy container โดยตรง แต่ deploy **Pod** ซึ่งเป็น wrapper รอบ container อย่างน้อยหนึ่งตัวขึ้นไป เหตุผลคือบางครั้งมี container มากกว่าหนึ่งตัวที่ **ต้อง**ทำงานร่วมกันแนบแน่นมากจนสมควรถูกจัดการ scheduling/lifecycle ไปด้วยกันเป็นหน่วยเดียว — เช่น container หลัก (main application) กับ container เสริมที่ทำหน้าที่ log shipper, proxy, หรือ cert renewal ให้ container หลัก

### Container ภายใน Pod เดียวกันแชร์อะไรกันบ้าง

จุดสำคัญที่สุดที่ทำให้ Pod มีความหมายมากกว่าแค่ "กลุ่ม container": ทุก container ใน Pod เดียวกัน**แชร์ network namespace เดียวกัน**

```
Pod
┌───────────────────────────────────────────┐
│  Network Namespace เดียวกัน (Pod IP เดียว) │
│                                             │
│  ┌──────────────┐      ┌────────────────┐ │
│  │  Container A  │      │  Container B    │ │
│  │  (app หลัก)   │      │  (sidecar)      │ │
│  │  :8080        │      │  :9090          │ │
│  └──────────────┘      └────────────────┘ │
│                                             │
│  → คุยกันผ่าน localhost:port ได้ตรงๆ       │
│  → แชร์ Pod IP เดียวกันมองจากภายนอก        │
└───────────────────────────────────────────┘
```

ผลลัพธ์เชิงปฏิบัติ: container สองตัวใน Pod เดียวกันคุยกันผ่าน `localhost` ได้เลย (เหมือนสอง process บนเครื่องเดียวกัน) ไม่ต้องผ่าน Service หรือรู้จัก Pod IP ของกันและกัน และมองจากภายนอก Pod ทั้งก้อนมี **IP เดียว** ไม่ว่าข้างในจะมีกี่ container ก็ตาม (ต่างจาก container แต่ละตัวใน Docker ปกติที่แยก network namespace กันเองโดย default)

### Sidecar Pattern

**Sidecar** คือ container เสริมที่รันคู่กับ container หลักใน Pod เดียวกัน เพื่อทำหน้าที่สนับสนุนโดยไม่ต้องแก้โค้ด application หลักเลย ตัวอย่างที่พบบ่อย:

- **Log shipper** — sidecar อ่าน log ที่ application หลักเขียนลง shared volume แล้วส่งต่อไปยัง log pipeline (เชื่อมกับ [Volume 14 — Observability](../14-observability/README.md))
- **Service mesh proxy** — sidecar (เช่น Envoy) ดักจับ traffic เข้า-ออกของ application หลักทั้งหมด เพื่อทำ mTLS, retry, observability โดย application ไม่ต้องรู้ตัวเลย

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: api-with-log-sidecar
spec:
  containers:
    - name: api
      image: myregistry.internal/api:1.4.0
      ports:
        - containerPort: 8080
      volumeMounts:
        - name: log-dir
          mountPath: /var/log/app
    - name: log-shipper
      image: fluent-bit:2.2
      volumeMounts:
        - name: log-dir
          mountPath: /var/log/app
          readOnly: true
  volumes:
    - name: log-dir
      emptyDir: {}
```

ในทางปฏิบัติ engineer แทบไม่สร้าง Pod ตรงๆ แบบนี้ — เกือบทุกครั้งจะสร้างผ่าน **Deployment** แทน (หัวข้อถัดไป) เพราะ Pod เดี่ยวๆ ถ้าตายแล้ว**ไม่มีอะไรสร้างมันขึ้นมาใหม่ให้อัตโนมัติ**

## 1.2 Deployment

**Deployment** คือ object ที่จัดการ Pod ให้เป็นไปตาม desired state ที่ประกาศไว้ — จำนวน replica เท่าไหร่, ใช้ image เวอร์ชันไหน, และวิธี update เมื่อมีการเปลี่ยนแปลง

### ความสัมพันธ์กับ ReplicaSet

Deployment ไม่ได้จัดการ Pod ตรงๆ แต่สร้างและจัดการ **ReplicaSet** อีกที ซึ่ง ReplicaSet เป็นตัวที่คอยรับประกันว่ามี Pod ที่ match label selector ตามจำนวนที่กำหนดไว้เสมอ (Pod ตายไป ReplicaSet สร้างตัวใหม่ทดแทนทันที):

```
Deployment (desired state: image=v2, replicas=3)
    │ สร้าง/จัดการ
    ▼
ReplicaSet (สำหรับ image=v2)
    │ สร้าง/จัดการ
    ▼
Pod × 3
```

เมื่อ update image เป็น version ใหม่ Deployment จะสร้าง **ReplicaSet ใหม่**ทั้งชุด (สำหรับ version ใหม่) แล้วค่อยๆ ลด replica ของ ReplicaSet เก่าลงพร้อมเพิ่ม replica ของ ReplicaSet ใหม่ขึ้น — นี่คือกลไกเบื้องหลัง rolling update

### Rolling Update: `maxSurge` และ `maxUnavailable`

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
spec:
  replicas: 6
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 2          # สร้าง Pod ใหม่เกินจำนวน replicas ได้สูงสุด 2 ตัวระหว่าง update
      maxUnavailable: 1    # ยอมให้ Pod พร้อมใช้งานน้อยกว่าเป้าหมายได้สูงสุด 1 ตัวระหว่าง update
  selector:
    matchLabels:
      app: api
  template:
    metadata:
      labels:
        app: api
    spec:
      containers:
        - name: api
          image: myregistry.internal/api:1.5.0
          ports:
            - containerPort: 8080
```

- **`maxSurge`** — จำนวน Pod สูงสุดที่สร้าง**เกิน** `replicas` ได้ชั่วคราวระหว่าง rollout (เพื่อให้มี Pod ใหม่พร้อมก่อนที่จะปิด Pod เก่า)
- **`maxUnavailable`** — จำนวน Pod สูงสุดที่ยอมให้ **หายไปจากเป้าหมาย** ได้ชั่วคราวระหว่าง rollout (เพื่อความเร็ว แลกกับ capacity ที่ลดลงชั่วขณะ)

ค่าเริ่มต้นของทั้งคู่คือ 25% ของ `replicas` การปรับสองค่านี้คือการ trade-off ระหว่าง **ความเร็วของ rollout** กับ **capacity ที่พร้อมใช้งานระหว่าง rollout** — ถ้าตั้ง `maxSurge` สูงและ `maxUnavailable` เป็น 0 จะได้ rollout ที่ capacity ไม่ตกเลยแต่ใช้ resource ชั่วคราวมากกว่า (zero-downtime deployment)

### Rollback

ทุกครั้งที่ Deployment เปลี่ยน spec ที่ทำให้เกิด ReplicaSet ใหม่ Kubernetes จะเก็บ **revision history** ไว้ (จำนวน revision ที่เก็บกำหนดได้ด้วย `revisionHistoryLimit`) ทำให้ย้อนกลับไป version ก่อนหน้าได้ทันทีถ้า deploy แล้วพบปัญหา:

```bash
kubectl rollout status deployment/api        # ดูสถานะ rollout ปัจจุบัน
kubectl rollout history deployment/api       # ดู revision history ทั้งหมด
kubectl rollout undo deployment/api          # rollback กลับไป revision ก่อนหน้าทันที
kubectl rollout undo deployment/api --to-revision=3   # rollback ไป revision ที่ระบุเจาะจง
```

`rollout undo` ทำงานโดยสลับกลับไปใช้ ReplicaSet ของ revision เก่า (ถ้ายังไม่ถูกลบทิ้งตาม `revisionHistoryLimit`) ผ่านกลไก rolling update แบบเดียวกับตอน deploy ไปข้างหน้า — ไม่ใช่การ "ยกเลิก" แบบทันทีทันใด แต่ค่อยๆ transition กลับตาม `maxSurge`/`maxUnavailable` เช่นกัน

## 1.3 Service

Pod เป็นสิ่งที่**ตายและเกิดใหม่ตลอดเวลา** (rolling update, crash, node ล่ม) และทุกครั้งที่ Pod ใหม่เกิดขึ้น จะได้ **Pod IP ใหม่เสมอ** — ถ้า client ต้อง track Pod IP ตรงๆ ระบบจะพังทันทีที่มี Pod ตัวไหนถูกแทนที่ **Service** แก้ปัญหานี้ด้วยการให้ **endpoint ที่นิ่ง (stable)** ตัวหนึ่ง ที่ route ไปยัง Pod กลุ่มหนึ่งเสมอ ไม่ว่า Pod ข้างในจะเปลี่ยนไปกี่รอบก็ตาม

### ClusterIP, NodePort, LoadBalancer

| ประเภท | เข้าถึงได้จากไหน | ใช้เมื่อไหร่ |
|---|---|---|
| **ClusterIP** (default) | ภายในคลัสเตอร์เท่านั้น | การสื่อสารระหว่าง service กับ service ภายในระบบเดียวกัน (กรณีส่วนใหญ่) |
| **NodePort** | เปิด port เดียวกันบนทุก node ของคลัสเตอร์ เข้าถึงได้จากภายนอกผ่าน `<node-ip>:<node-port>` | dev/test หรือกรณีไม่มี LoadBalancer ให้ใช้ |
| **LoadBalancer** | ขอ external load balancer จาก cloud provider (หรือ on-prem ผ่าน MetalLB) ให้อัตโนมัติ ได้ external IP ตายตัว | เปิด service สู่ internet โดยตรงในระดับ production |

```yaml
apiVersion: v1
kind: Service
metadata:
  name: api
spec:
  type: ClusterIP
  selector:
    app: api
  ports:
    - port: 80
      targetPort: 8080
```

`selector: app: api` คือกลไกที่ Service ใช้เลือกว่า Pod ตัวไหนคือปลายทาง — Pod ไหนก็ตามที่มี label `app: api` ตรงกัน จะถูกนับเป็นสมาชิกของ Service นี้โดยอัตโนมัติ (label เดียวกับที่ Deployment ใช้ใน `template.metadata.labels`)

### kube-proxy และการ Route ไปยัง Pod IP

Service ไม่ได้มี process รับ-ส่ง traffic เองจริงๆ — มันเป็นแค่ **abstraction** ที่ประกาศ IP/DNS name คงที่ไว้ ส่วนตัวที่ทำงานจริงเบื้องหลังคือ **kube-proxy** ซึ่งรันอยู่บนทุก node และคอย watch การเปลี่ยนแปลงของ Service/Endpoint ผ่าน API server แล้วเขียนกฎ **iptables** (หรือ IPVS ใน mode ที่ใหม่กว่า) ลงบน node นั้นๆ เพื่อดักจับ traffic ที่มุ่งไปยัง ClusterIP แล้ว **DNAT** (เปลี่ยนปลายทางจริง) ไปยัง Pod IP ตัวใดตัวหนึ่งที่ตรงกับ selector

```
Client → ClusterIP:80
              │  (iptables rule ที่ kube-proxy สร้างไว้)
              ▼  DNAT สุ่มเลือก 1 ใน Pod ที่ match selector
        Pod IP จริง:8080 (Pod A, B, หรือ C)
```

เพราะกลไกนี้ทำงานที่ระดับ kernel networking (iptables) ของแต่ละ node ไม่ใช่ผ่าน process เดียวตรงกลาง Service จึงไม่มี single point of failure และไม่เพิ่ม latency มากจากการ proxy ผ่าน process พิเศษ

### Service Discovery ผ่าน DNS

ทุก Service ที่สร้างขึ้นจะได้ DNS name ภายในคลัสเตอร์อัตโนมัติในรูปแบบ `<service-name>.<namespace>.svc.cluster.local` (หรือย่อเป็น `<service-name>` ถ้าเรียกจาก Pod ใน namespace เดียวกัน) ทำให้ application เชื่อมต่อ service อื่นด้วยชื่อได้เลยโดยไม่ต้องรู้ ClusterIP ตรงๆ — กลไกนี้ต่อยอดจากหลักการ DNS resolution พื้นฐานที่อธิบายไว้ใน [Volume 2 — DNS](../02-network/02-dns.md) เพียงแต่ระดับคลัสเตอร์ใช้ **CoreDNS** เป็นตัว resolve แทน public DNS resolver

## 1.4 Ingress

Service ประเภท `LoadBalancer` แก้ปัญหาเปิด service สู่ภายนอกได้ แต่ถ้ามีหลายสิบ service ที่ต้องเปิดสู่ภายนอก การขอ external load balancer แยกทีละตัว (ซึ่งมักมีค่าใช้จ่ายต่อตัวใน cloud) ไม่คุ้มค่าและจัดการยาก **Ingress** แก้ปัญหานี้ด้วยการเป็น **จุดเดียว** ที่รับ traffic จากภายนอกทั้งหมด แล้ว route ต่อไปยัง Service ภายในตาม path/host ที่กำหนด — คล้ายกับหลักการของ [reverse proxy](../02-network/05-reverse-proxy-load-balancer.md) แต่ทำงานเป็น native object ของ Kubernetes

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: api-ingress
spec:
  rules:
    - host: api.example.com
      http:
        paths:
          - path: /v1
            pathType: Prefix
            backend:
              service:
                name: api
                port:
                  number: 80
          - path: /admin
            pathType: Prefix
            backend:
              service:
                name: admin-api
                port:
                  number: 80
```

### Ingress Controller

**Ingress object เองเป็นแค่คำประกาศกฎ routing — ไม่มีอะไรทำงานจริงถ้าไม่มี Ingress Controller ติดตั้งอยู่ในคลัสเตอร์** Ingress Controller คือ Pod (มักรันเป็น Deployment ที่ expose ผ่าน Service ประเภท `LoadBalancer` เพียงตัวเดียว) ที่ watch Ingress object ทั้งหมด แล้วแปลงเป็น configuration จริงของตัวเอง (เช่น nginx.conf) เพื่อ route traffic ตามกฎที่ประกาศไว้

(เชื่อมกับ [Volume 10 — APISIX](../10-apisix/README.md) — APISIX สามารถทำหน้าที่เป็น Ingress Controller ได้เช่นกัน โดยนอกจาก routing พื้นฐานแล้วยังได้ความสามารถของ API Gateway เต็มรูปแบบไปด้วย เช่น rate limiting, JWT validation ที่ทำได้ตั้งแต่ชั้น ingress โดยไม่ต้องให้แต่ละ service จัดการเอง)

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| Pod | หน่วยเล็กสุด, container ในนั้นแชร์ network namespace เดียวกัน (Pod IP เดียว, คุยกันผ่าน localhost) |
| Sidecar | container เสริมใน Pod เดียวกัน ทำงานสนับสนุน (log shipper, mesh proxy) โดยไม่แก้โค้ด app หลัก |
| Deployment / ReplicaSet | Deployment จัดการ ReplicaSet, ReplicaSet คุม Pod count ให้ตรงเป้าหมายเสมอ |
| Rolling Update | `maxSurge`/`maxUnavailable` คุม trade-off ระหว่างความเร็ว rollout กับ capacity ที่พร้อมใช้งาน |
| Rollback | `kubectl rollout undo` สลับกลับไป ReplicaSet ของ revision ก่อนหน้าผ่านกลไก rolling update |
| Service | endpoint นิ่งที่ route ไปยัง Pod ที่เปลี่ยน IP ตลอดเวลา ผ่าน kube-proxy เขียน iptables/IPVS |
| ClusterIP/NodePort/LoadBalancer | ภายในคลัสเตอร์ / เปิดผ่านทุก node / ขอ external LB จาก provider |
| Ingress | จุดเดียวรับ traffic ภายนอกแล้ว route ตาม path/host, ต้องมี Ingress Controller ถึงทำงานได้จริง |

## คำถามทบทวน

1. อธิบายว่าทำไม container สองตัวใน Pod เดียวกันถึงคุยกันผ่าน `localhost` ได้ตรงๆ โดยไม่ต้องผ่าน Service
2. Deployment, ReplicaSet, Pod สัมพันธ์กันอย่างไร เมื่อ update image เป็น version ใหม่เกิดอะไรขึ้นกับ object ทั้งสามระดับนี้
3. ถ้าตั้ง `maxSurge: 0` และ `maxUnavailable: 1` สำหรับ Deployment ที่มี `replicas: 4` ระหว่าง rollout จะมี Pod พร้อมใช้งานต่ำสุดกี่ตัว เพราะเหตุใด
4. อธิบายกลไกที่ทำให้ traffic ที่ยิงไปยัง ClusterIP ของ Service ไปถึง Pod จริงได้ โดยไม่มี process กลางคอย proxy ตลอดเวลา
5. Ingress ต่างจาก Service ประเภท LoadBalancer อย่างไร และทำไม Ingress object เพียงอย่างเดียวถึงยังทำงานไม่ได้ถ้าไม่มี Ingress Controller

---

ถัดไป: [บทที่ 2 — ConfigMap และ Secret](02-configuration.md)
