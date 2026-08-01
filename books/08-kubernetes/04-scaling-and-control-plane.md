# บทที่ 4: HPA, etcd, Scheduler

## 4.1 Horizontal Pod Autoscaler (HPA)

**HPA** ปรับจำนวน replica ของ Deployment/StatefulSet ขึ้น-ลงอัตโนมัติตาม metric ที่สังเกตได้ (ต่างจาก manual scaling ที่ engineer ต้องสั่ง `kubectl scale` เอง)

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: api-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: api
  minReplicas: 3
  maxReplicas: 20
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
```

### กลไกการตัดสินใจ Scale

HPA ทำงานเป็น control loop ที่ทำงานเป็นระยะ (default ทุก 15 วินาที): ดึงค่า metric ปัจจุบันของ Pod ทั้งหมดที่ match `scaleTargetRef` จาก **Metrics Server** (สำหรับ CPU/memory พื้นฐาน) หรือจาก **custom/external metrics adapter** (สำหรับ metric อื่น เช่น queue length, request per second — มักเชื่อมกับ Prometheus ดู [Volume 14 — Observability](../14-observability/README.md)) แล้วคำนวณจำนวน replica ที่ควรจะเป็นด้วยสูตรคร่าวๆ:

```
desiredReplicas = ceil(currentReplicas × (currentMetricValue / desiredMetricValue))
```

เช่น มี 4 replica, CPU เฉลี่ยตอนนี้ 140% ของ target 70% → `4 × (140/70) = 8` replica

### จุดที่มักเข้าใจผิด: เปอร์เซ็นต์อ้างอิงกับ Request ไม่ใช่ Limit และไม่ใช่ spec ของเครื่องจริง

`averageUtilization: 70` ในตัวอย่างข้างบนหมายถึง **70% ของค่าที่ตั้งไว้ใน `resources.requests.cpu` ของ container นั้น** ไม่ใช่ 70% ของ CPU ทั้งหมดบน node และไม่ใช่ 70% ของ `limits`:

```yaml
resources:
  requests:
    cpu: "500m"     # HPA คำนวณ % จากค่านี้
  limits:
    cpu: "1000m"
```

ถ้า Pod นี้ใช้ CPU จริง 400m ตอนนี้ HPA จะเห็นว่า utilization = 400m / 500m = 80% (เกิน target 70%) → trigger scale up **แม้ Pod จะยังใช้ CPU ไม่ถึงครึ่งของ `limits` เลยก็ตาม** นี่คือกับดักคลาสสิกที่ทีมตั้ง `requests` ต่ำเกินไป (เผื่อ "ประหยัด" resource เวลา schedule) แล้วงงว่าทำไม HPA scale บ่อยผิดปกติทั้งที่ node ยังมี capacity เหลือเยอะ — การตั้งค่า `requests` ให้สะท้อนการใช้งานจริงของ application อย่างสมเหตุสมผลจึงเป็นเงื่อนไขที่ HPA ทำงานได้อย่างมีความหมาย ไม่ใช่แค่ตัวเลขที่ตั้งขึ้นมาลอยๆ

## 4.2 etcd

### สิ่งที่ etcd เก็บ

**etcd** คือ **distributed key-value store** ที่เป็น**ฐานข้อมูลเดียว**ของ Kubernetes control plane ทั้งหมด — ทุก object ที่เคยเห็นมาในเล่มนี้ (Pod, Deployment, Service, ConfigMap, Secret, PV, ฯลฯ) รวมถึง **สถานะปัจจุบันจริง** (observed state) ของแต่ละ object ถูกเก็บไว้ที่ etcd ทั้งหมด ไม่มีที่เก็บอื่นอีกแล้ว

```
kubectl apply -f deployment.yaml
        │
        ▼
   API Server ─────► เขียน desired state ลง etcd
        ▲
        │ อ่าน/watch
        │
Controller Manager, Scheduler, kubelet
   (ทุกคนคุยผ่าน API Server เท่านั้น ไม่มีใครแตะ etcd ตรงๆ)
```

**API Server เป็นเพียงประตูเดียวที่คุยกับ etcd โดยตรง** — component อื่นทั้งหมด (scheduler, controller manager, kubelet บนทุก node) ไม่เคยแตะ etcd ตรงๆ เลย แต่คุยผ่าน API Server เสมอ ทำให้ etcd เป็น **single source of truth** ที่แท้จริงของทั้งคลัสเตอร์ — ถ้า etcd หายหรือ corrupt คลัสเตอร์ทั้งหมด "ลืม" ว่าตัวเองควรมีอะไรอยู่บ้างทันที (นี่คือเหตุผลที่ backup etcd สม่ำเสมอเป็นงาน operations ที่สำคัญที่สุดอย่างหนึ่งของทีมดูแลคลัสเตอร์)

### Consistency ผ่าน Raft

etcd รันเป็นคลัสเตอร์หลาย instance (โดยทั่วไป 3 หรือ 5 ตัว เลขคี่เสมอ) เพื่อความทนทาน ใช้ **Raft consensus algorithm** รับประกันว่าทุก node เห็นข้อมูลตรงกันเสมอ (strong consistency) โดยสรุปสั้นๆ: การเขียนข้อมูลทุกครั้งต้องได้รับเสียงเห็นชอบจาก**เสียงข้างมาก (majority/quorum)** ของ node ก่อนถือว่าสำเร็จ — คลัสเตอร์ 5 node ทนได้ถึง 2 node ล่มพร้อมกันโดยยังทำงานต่อได้ปกติ (เพราะยังเหลือ majority 3 node) แต่ถ้า node ล่มเกินครึ่งหนึ่ง etcd จะหยุดรับ write ทันทีเพื่อป้องกัน split-brain (ข้อมูลไม่ตรงกันระหว่าง node) — เลขคี่จึงสำคัญ เพราะทำให้นิยาม "เสียงข้างมาก" ชัดเจนไม่มีทาง tie

## 4.3 Scheduler

เมื่อสร้าง Pod ใหม่ (ที่ยังไม่ถูกกำหนดว่าจะรันบน node ไหน) **Scheduler** คือ component ที่ตัดสินใจว่า Pod นั้นควรไปอยู่บน node ตัวไหน ผ่านกระบวนการสองขั้น:

### Phase 1: Filtering

คัดกรอง node ที่ **ไม่มีทางรัน Pod นี้ได้เลย** ออกไปก่อน เช่น node ที่ resource เหลือไม่พอตาม `requests` ของ Pod, node ที่ไม่มี volume ที่ Pod ต้องการ mount, หรือ node ที่มี taint ที่ Pod ไม่มี toleration รองรับ (หัวข้อถัดไป) — เหลือแค่ node ที่ **เป็นไปได้** เท่านั้น

### Phase 2: Scoring

จาก node ที่ผ่าน filtering แล้ว scheduler จะให้คะแนนแต่ละตัวตามหลายปัจจัย (resource ที่เหลือพอดีไม่มากไม่น้อยเกินไป, การกระจาย Pod ให้ไม่กระจุกตัว node เดียว, affinity ที่กำหนดไว้) แล้ว**เลือก node ที่คะแนนสูงสุด**ให้ Pod นั้นรัน

### Affinity / Anti-affinity

ควบคุมว่า Pod "อยาก" หรือ "ไม่อยาก" อยู่ใกล้ Pod/node แบบไหน:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
spec:
  template:
    spec:
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            - labelSelector:
                matchLabels:
                  app: api
              topologyKey: "kubernetes.io/hostname"
      containers:
        - name: api
          image: myregistry.internal/api:1.5.0
```

ตัวอย่างนี้บังคับ (`required...`) ว่า **Pod ของ Deployment นี้ห้ามอยู่ node เดียวกันเกิน 1 ตัว** (`podAntiAffinity` + `topologyKey: hostname`) — สำคัญมากสำหรับ high availability: ถ้าทุก replica ของ `api` บังเอิญถูก schedule ไปกองที่ node เดียวกันหมด node นั้นล่มตัวเดียวก็ทำให้ service ทั้งหมดล่มไปด้วย ทั้งที่มี replica หลายตัว

### Taints และ Tolerations

**Taint** ถูกใส่ไว้ที่ **node** เพื่อ "ผลักไล่" Pod ที่ไม่มี **toleration** ที่ตรงกันออกไป — เป็นกลไกตรงข้ามกับ affinity (affinity คือ Pod เลือก node, taint/toleration คือ node ปฏิเสธ Pod) ใช้บ่อยที่สุดสำหรับ **dedicate node กลุ่มหนึ่งให้ workload เฉพาะทาง**

ตัวอย่างสถานการณ์จริง: มี node ที่ติดตั้ง GPU ราคาแพง ต้องการให้เฉพาะ workload ที่ต้องใช้ GPU จริงๆ เท่านั้นถูก schedule ไปที่นั่น ไม่ให้ Pod ทั่วไปไปแย่งพื้นที่โดยไม่จำเป็น:

```bash
kubectl taint nodes gpu-node-1 workload=gpu:NoSchedule
```

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: ml-training-job
spec:
  tolerations:
    - key: "workload"
      operator: "Equal"
      value: "gpu"
      effect: "NoSchedule"
  containers:
    - name: trainer
      image: myregistry.internal/ml-trainer:2.0
```

- **Taint** บน `gpu-node-1` บอกว่า "ห้าม schedule Pod ใดๆ มาที่นี่ (`NoSchedule`) เว้นแต่จะ tolerate taint `workload=gpu`"
- **Toleration** ใน Pod `ml-training-job` บอกว่า "ฉันทนต่อ taint นี้ได้" ทำให้เป็นหนึ่งใน Pod ไม่กี่ตัวที่ scheduler พิจารณา node นี้ให้ (แต่ยังต้องผ่าน scoring เหมือน node อื่นตามปกติ ไม่ได้บังคับว่าต้องไปที่นั่นเสมอ — ถ้าต้องการบังคับให้ไปที่ node กลุ่มนี้เท่านั้นจริงๆ ต้องใช้ node affinity ควบคู่ไปด้วย)

Taint/toleration เป็นกลไก "กันไม่ให้เข้า" (exclusion) ในขณะที่ affinity เป็นกลไก "อยากให้ไป" (attraction) — ระบบ production ที่ต้องการ dedicate node จริงจังมักใช้ทั้งสองคู่กัน: taint กัน Pod อื่นออก และ node affinity ดึง Pod เป้าหมายเข้าไปให้แน่ใจ

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| HPA | control loop ปรับ replica ตาม metric เทียบกับ target, ทำงานเป็นระยะผ่าน Metrics Server/custom metrics |
| HPA gotcha | เปอร์เซ็นต์ utilization อ้างอิงกับ `requests` เสมอ ไม่ใช่ `limits` หรือ spec เครื่องจริง — `requests` ต่ำเกินไปทำให้ scale ผิดปกติ |
| etcd | single source of truth ของทั้งคลัสเตอร์, มีแค่ API Server เท่านั้นที่คุยกับมันตรงๆ |
| Raft | ต้องได้ majority vote ก่อน write สำเร็จ, คลัสเตอร์เลขคี่ทนต่อ node ล่มได้โดยไม่เกิด split-brain |
| Scheduler | สองขั้นตอน: filtering (คัดที่รันไม่ได้ออก) แล้ว scoring (เลือกคะแนนสูงสุด) |
| Affinity/Anti-affinity | Pod เลือก/หลีกเลี่ยง node หรือ Pod อื่นตามเงื่อนไข ใช้กระจาย replica ให้ทน node ล่ม |
| Taint/Toleration | node ปฏิเสธ Pod ที่ไม่ tolerate ใช้ dedicate node ให้ workload เฉพาะทาง (เช่น GPU) |

## คำถามทบทวน

1. HPA คำนวณ CPU utilization เทียบกับอะไร ทำไมการตั้ง `requests.cpu` ต่ำเกินไปถึงทำให้ HPA scale ผิดปกติ
2. ทำไม component อื่นในคลัสเตอร์ (scheduler, kubelet) ถึงไม่คุยกับ etcd ตรงๆ แต่ต้องผ่าน API Server เสมอ
3. etcd คลัสเตอร์ 5 node ทนต่อจำนวน node ล่มพร้อมกันได้สูงสุดกี่ตัวโดยยังทำงานต่อได้ เพราะเหตุใด
4. อธิบายความต่างระหว่าง filtering กับ scoring ในกระบวนการตัดสินใจของ scheduler
5. ถ้าต้องการ dedicate node กลุ่มหนึ่งให้เฉพาะ workload ประเภท GPU เท่านั้น ทำไมการใช้ taint/toleration อย่างเดียวถึงอาจไม่พอ ต้องใช้ควบคู่กับอะไรถึงจะรับประกันได้แน่นอน

---

ก่อนหน้า: [บทที่ 3 — StatefulSet, DaemonSet, PVC, StorageClass](03-stateful-workloads-and-storage.md) | กลับสู่ [สารบัญ Volume 8](README.md)
