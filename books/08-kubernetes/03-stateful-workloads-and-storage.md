# บทที่ 3: StatefulSet, DaemonSet, PVC, StorageClass

## 3.1 StatefulSet

### ปัญหาที่ Deployment แก้ไม่ได้

Deployment ([บทที่ 1](01-workloads-and-networking.md#12-deployment)) เหมาะกับ workload แบบ **stateless** ที่ Pod แต่ละตัวเหมือนกันทุกประการ แทนที่กันได้อิสระ — Pod ตายไปตัวหนึ่ง สร้างตัวใหม่มาแทนที่ก็ไม่มีใครสังเกตความต่าง เพราะ Pod ไม่มี "ตัวตน" เฉพาะของมันเอง

แต่ workload บางประเภท (โดยเฉพาะ **self-hosted database** หรือระบบ distributed ที่ node แต่ละตัวรู้จักกันด้วยชื่อ/ตำแหน่งเฉพาะ เช่น PostgreSQL replica, Kafka broker, Elasticsearch node) ต้องการ Pod ที่มี**ตัวตนที่คงที่** — Pod ตัวที่ 0 ต้องยังคงเป็น "ตัวที่ 0" เสมอแม้จะถูกสร้างใหม่ ไม่ใช่กลายเป็น Pod สุ่มชื่อใหม่ทุกครั้งที่ restart

### Stable Network Identity

StatefulSet แก้ปัญหานี้ด้วยการตั้งชื่อ Pod แบบ**คาดเดาได้และคงที่** ตามรูปแบบ `<statefulset-name>-<ordinal-index>`:

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres
spec:
  serviceName: postgres-headless
  replicas: 3
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
        - name: postgres
          image: postgres:16-alpine
          volumeMounts:
            - name: data
              mountPath: /var/lib/postgresql/data
  volumeClaimTemplates:
    - metadata:
        name: data
      spec:
        accessModes: ["ReadWriteOnce"]
        resources:
          requests:
            storage: 20Gi
```

Pod ที่ได้จะชื่อ `postgres-0`, `postgres-1`, `postgres-2` เสมอ — ต่อให้ `postgres-1` crash แล้วถูกสร้างใหม่ Pod ที่มาแทนก็ยังชื่อ `postgres-1` เหมือนเดิม พร้อมกับ **PersistentVolumeClaim (ดูหัวข้อ 3.3) ของตัวเองที่ผูกกับชื่อนั้นโดยเฉพาะ** (ผ่าน `volumeClaimTemplates`) ทำให้ Pod ตัวที่กลับมาใหม่ยัง mount กลับไปหา volume ข้อมูลเดิมของมันได้ ไม่ใช่ volume ของ Pod ตัวอื่นในกลุ่ม

เมื่อใช้คู่กับ **headless Service** (`clusterIP: None`) แต่ละ Pod จะได้ DNS name เฉพาะตัวเพิ่มเติมด้วย เช่น `postgres-0.postgres-headless.default.svc.cluster.local` ทำให้ระบบ distributed ที่ node ต้องรู้จักกันด้วยชื่อเฉพาะ (เช่น PostgreSQL streaming replication ที่ config replica ต้องรู้ hostname ของ primary ชัดเจน) ทำงานได้

### Ordered Deployment/Scaling

StatefulSet สร้างและลบ Pod **ตามลำดับ index เสมอ** (0, 1, 2, ... ตอนสร้าง และย้อนกลับตอนลด) ไม่สร้างพร้อมกันทั้งหมดเหมือน Deployment — สำคัญมากสำหรับระบบที่ node ตัวหลัง**ต้องรอ**ให้ node ก่อนหน้าพร้อมก่อน (เช่น replica ต้อง join คลัสเตอร์หลัง primary พร้อมแล้วเท่านั้น)

### เมื่อไหร่ถึงต้องใช้ StatefulSet จริงๆ

**ใช้เมื่อ**: self-hosted database, message queue แบบ cluster (Kafka), หรือระบบใดก็ตามที่ instance แต่ละตัวมี **state เฉพาะตัวที่ผูกกับ identity** และ instance อื่นต้องรู้จัก identity นั้นเพื่อทำงานร่วมกัน (ดู [Volume 12 — Kafka](../12-kafka/README.md))

**ไม่จำเป็นต้องใช้เมื่อ**: application ทั่วไปที่แค่ต้องการ persistent volume แต่ Pod ไม่ต้องมี identity เฉพาะตัว (เช่น API service ที่เขียน uploaded file ลง shared storage) — กรณีนี้ Deployment ธรรมดาพร้อม PVC แยกต่างหาก (ไม่ใช่ `volumeClaimTemplates`) ก็เพียงพอแล้ว เป็นความเข้าใจผิดที่พบบ่อยที่ทีมเลือกใช้ StatefulSet เพียงเพราะ "มี state" โดยไม่ได้ต้องการ stable identity จริงๆ ซึ่งทำให้ operation ซับซ้อนขึ้นโดยไม่จำเป็น (rolling update ช้ากว่า เพราะต้อง ordered เสมอ)

## 3.2 DaemonSet

**DaemonSet** รับประกันว่า**ทุก node** ในคลัสเตอร์ (หรือ node กลุ่มที่เลือกด้วย `nodeSelector`) จะมี Pod ของ DaemonSet นี้รันอยู่เสมอ**พอดี 1 ตัว** — ไม่ใช่การกำหนด `replicas` เหมือน Deployment แต่จำนวน Pod ผูกกับจำนวน node โดยตรง เมื่อมี node ใหม่เข้าคลัสเตอร์ DaemonSet จะสร้าง Pod ใหม่บน node นั้นให้อัตโนมัติทันที

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: fluent-bit
spec:
  selector:
    matchLabels:
      app: fluent-bit
  template:
    metadata:
      labels:
        app: fluent-bit
    spec:
      containers:
        - name: fluent-bit
          image: fluent/fluent-bit:2.2
          volumeMounts:
            - name: varlog
              mountPath: /var/log
              readOnly: true
      volumes:
        - name: varlog
          hostPath:
            path: /var/log
```

### Use Case: Log Collector / Monitoring Agent

DaemonSet เหมาะกับงานที่ต้อง**เก็บข้อมูลระดับ node** โดยธรรมชาติ — ตัวอย่างคลาสสิกคือ **log collector** (เชื่อมกับ [Volume 14 — Observability](../14-observability/README.md)) เช่น Fluent Bit ที่ mount `/var/log` ของ node เข้ามาอ่าน log ของทุก container ที่รันอยู่บน node นั้น แล้วส่งต่อไปยัง central log pipeline — เพราะต้องการอ่าน log จากทุก Pod บน node นั้นแบบ node-level ไม่ใช่ per-application การรันเป็น sidecar ในทุก Pod (แบบ [บทที่ 1](01-workloads-and-networking.md#sidecar-pattern)) จะสิ้นเปลือง resource กว่ามาก เพราะซ้ำซ้อนกันในทุก Pod ทั้งที่ทำหน้าที่เดียวกัน ในขณะที่ DaemonSet ใช้ instance เดียวต่อ node ครอบคลุมทุก Pod บน node นั้นพร้อมกัน — ตัวอย่างอื่นที่ใช้ pattern เดียวกัน: node-level monitoring agent (Prometheus Node Exporter), network plugin (CNI agent)

## 3.3 PersistentVolume (PV) vs PersistentVolumeClaim (PVC)

Kubernetes แยกแนวคิดเรื่อง storage ออกเป็นสองฝั่งเพื่อให้ application กับ infrastructure ไม่ต้องผูกติดกันโดยตรง:

- **PersistentVolume (PV)** — ตัวแทนของพื้นที่ storage จริงที่มีอยู่ในระบบ (disk บน cloud, NFS share, local disk) สร้างโดย admin หรือถูกสร้างอัตโนมัติผ่าน dynamic provisioning (หัวข้อถัดไป)
- **PersistentVolumeClaim (PVC)** — คำขอของ application ที่บอกว่า "ต้องการพื้นที่เท่าไหร่ ด้วย access mode แบบไหน" โดยไม่ต้องรู้ว่า storage จริงเบื้องหลังเป็นอะไร

```
Application (Pod)
      │ อ้างอิงผ่าน volumeMounts
      ▼
PersistentVolumeClaim (PVC)     ← "ขอ storage 20Gi, ReadWriteOnce"
      │ Kubernetes จับคู่ (bind) ให้อัตโนมัติ
      ▼
PersistentVolume (PV)           ← storage จริงที่มีอยู่ (หรือถูกสร้างใหม่แบบ dynamic)
      │
      ▼
Disk จริงบน infrastructure (cloud disk / NFS / local disk)
```

การแยกชั้นนี้ทำให้ทีม application เขียน manifest แค่ระดับ PVC (บอกความต้องการ) โดยไม่ต้องรู้รายละเอียด infrastructure เบื้องหลังเลย — เหมือนหลักการ dependency injection ใน [Volume 4 — Golang](../04-golang/README.md) ที่แยก interface (PVC) ออกจาก implementation จริง (PV)

## 3.4 StorageClass และ Dynamic Provisioning

การสร้าง PV ด้วยมือทีละตัวไม่ scale ในระบบใหญ่ **StorageClass** แก้ปัญหานี้ด้วยการนิยาม "แม่แบบ" ของ storage ที่ระบบสร้าง PV ให้อัตโนมัติ (**dynamic provisioning**) ทันทีที่มี PVC ร้องขอ โดยไม่ต้องมีใครสร้าง PV ล่วงหน้าไว้รอ:

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: fast-ssd
provisioner: kubernetes.io/aws-ebs
parameters:
  type: gp3
reclaimPolicy: Delete
volumeBindingMode: WaitForFirstConsumer
```

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: api-data
spec:
  storageClassName: fast-ssd
  accessModes: ["ReadWriteOnce"]
  resources:
    requests:
      storage: 10Gi
```

เมื่อ PVC นี้ถูกสร้าง Kubernetes จะเรียก **provisioner** ที่ระบุ (ในที่นี้คือปลั๊กอินของ cloud provider) ให้สร้าง disk จริงและ PV ที่ผูกกับ PVC นี้ให้อัตโนมัติทันที — engineer ไม่ต้องไปสร้าง disk ด้วยมือบน cloud console เลย

## 3.5 Access Mode

| Access Mode | ความหมาย | สถานการณ์ที่ใช้ |
|---|---|---|
| **ReadWriteOnce (RWO)** | mount แบบอ่าน-เขียนได้จาก **node เดียว** เท่านั้น ณ เวลาหนึ่ง | Database ทั่วไป (PostgreSQL, MySQL) ที่มี process เดียวเขียนข้อมูลของมันเอง — ตรงกับตัวอย่าง StatefulSet ในหัวข้อ 3.1 |
| **ReadOnlyMany (ROX)** | mount แบบอ่านอย่างเดียวได้จากหลาย node พร้อมกัน | ข้อมูล/asset ที่ต้องแชร์ให้หลาย Pod อ่านพร้อมกันแต่ไม่มีใครเขียน เช่น static config หรือ reference dataset ที่ update ไม่บ่อย |
| **ReadWriteMany (RWX)** | mount แบบอ่าน-เขียนได้จากหลาย node พร้อมกัน | Shared file storage ที่หลาย Pod ต้องเขียนร่วมกัน เช่น upload directory ที่ทุก replica ของ web app ต้องเห็นไฟล์เดียวกัน — ต้องการ storage backend ที่รองรับจริง (เช่น NFS) ไม่ใช่ cloud block storage ทั่วไปที่มักรองรับแค่ RWO |

ข้อควรระวังเชิงปฏิบัติ: **ไม่ใช่ storage backend ทุกตัวรองรับทุก access mode** — cloud block storage (AWS EBS, GCP Persistent Disk) ส่วนใหญ่รองรับแค่ `ReadWriteOnce` เท่านั้น ถ้าต้องการ `ReadWriteMany` จริงๆ ต้องเลือก storage backend ที่ออกแบบมาสำหรับ shared access โดยเฉพาะ (เช่น NFS, CephFS, EFS บน AWS) — การเลือก access mode ผิดตั้งแต่ต้นทำให้ PVC ค้าง `Pending` ไม่ถูก bind เพราะไม่มี provisioner ตัวไหนสร้าง PV ที่ตรงเงื่อนไขให้ได้

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| StatefulSet | Pod ชื่อคงที่ตามลำดับ, ผูก PVC เฉพาะตัว, deploy/scale ตามลำดับ — ใช้กับ workload ที่ต้องการ stable identity จริงๆ เท่านั้น |
| DaemonSet | 1 Pod ต่อ node เสมอ เหมาะกับงานระดับ node เช่น log collector, monitoring agent |
| PV vs PVC | PV = storage จริง, PVC = คำขอของ application ที่แยกจาก implementation เบื้องหลัง |
| StorageClass | แม่แบบสร้าง PV อัตโนมัติ (dynamic provisioning) ไม่ต้องสร้าง PV ล่วงหน้าด้วยมือ |
| Access Mode | RWO = node เดียว (database ทั่วไป), ROX = อ่านหลาย node, RWX = เขียนหลาย node (ต้องการ storage backend เฉพาะ) |

## คำถามทบทวน

1. อธิบายว่าทำไมระบบอย่าง PostgreSQL replication cluster ถึงต้องการ StatefulSet แทน Deployment ธรรมดา
2. ทำไม log collector อย่าง Fluent Bit ถึงเหมาะกับการรันเป็น DaemonSet มากกว่ารันเป็น sidecar ในทุก Pod
3. PV กับ PVC แยกหน้าที่กันอย่างไร การแยกนี้ให้ประโยชน์อะไรกับทีมที่เขียน application manifest
4. StorageClass กับ dynamic provisioning ช่วยลดงานอะไรที่ต้องทำด้วยมือถ้าไม่มีกลไกนี้
5. ถ้า Pod ต้องการ PVC แบบ `ReadWriteMany` แต่ storage backend ที่มีอยู่รองรับแค่ `ReadWriteOnce` จะเกิดอะไรขึ้น และแก้ปัญหาอย่างไร

---

ก่อนหน้า: [บทที่ 2 — ConfigMap และ Secret](02-configuration.md) | ถัดไป: [บทที่ 4 — HPA, etcd, Scheduler](04-scaling-and-control-plane.md)
