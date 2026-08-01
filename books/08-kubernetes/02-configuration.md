# บทที่ 2: ConfigMap และ Secret

## 2.1 ทำไมต้องแยก Configuration ออกจาก Image

หลักการข้อที่สามของ **12-factor app** (แนวปฏิบัติมาตรฐานสำหรับสร้าง cloud-native application) ระบุว่า **configuration ต้องเก็บแยกจากโค้ด** — ไม่ hardcode ค่าอย่าง database connection string, API endpoint, หรือ feature flag ไว้ในตัวโปรแกรมหรือฝังลงใน image ตอน build

เหตุผลเชิงปฏิบัติ: image เดียวกันที่ build ครั้งเดียว (ดู [Volume 7 — Docker](../07-docker/README.md)) ควร deploy ได้ทั้ง dev, staging, production โดยไม่ต้อง build ใหม่แค่เพราะ database host เปลี่ยน — ถ้า config ฝังอยู่ใน image การเปลี่ยน environment แต่ละครั้งจะบังคับให้ต้อง build image ใหม่เสมอ ซึ่งขัดกับหลักการที่ว่า **artifact ตัวเดียวกันต้อง deploy ผ่านทุก environment ได้** (สิ่งที่เปลี่ยนคือ config ที่ inject เข้าไป ไม่ใช่ตัว image)

Kubernetes มี object สองตัวสำหรับงานนี้: **ConfigMap** สำหรับข้อมูลทั่วไปที่ไม่ sensitive และ **Secret** สำหรับข้อมูลอ่อนไหว

## 2.2 ConfigMap

### สร้าง ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: api-config
data:
  LOG_LEVEL: "info"
  FEATURE_NEW_CHECKOUT: "true"
  app.properties: |
    max_connections=100
    timeout_seconds=30
```

`data` เก็บได้ทั้ง key-value ธรรมดา (ใช้เป็น environment variable ได้) และเนื้อหาไฟล์เต็มรูปแบบ (ใช้ mount เป็นไฟล์ได้) — ConfigMap เดียวกันเลือกใช้ได้ทั้งสองแบบตามความเหมาะสม

### Pattern ที่ 1: ใช้เป็น Environment Variable

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
spec:
  template:
    spec:
      containers:
        - name: api
          image: myregistry.internal/api:1.5.0
          envFrom:
            - configMapRef:
                name: api-config
```

### Pattern ที่ 2: Mount เป็นไฟล์

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
spec:
  template:
    spec:
      containers:
        - name: api
          image: myregistry.internal/api:1.5.0
          volumeMounts:
            - name: config-volume
              mountPath: /etc/app
      volumes:
        - name: config-volume
          configMap:
            name: api-config
```

### ความต่างสำคัญ: Mounted File อัปเดตอัตโนมัติ แต่ Env Var ต้อง Restart

นี่คือความแตกต่างที่กระทบการออกแบบระบบจริง:

- **Environment variable** — ค่าถูกอ่านเข้าไปตอน process **start** เท่านั้น ถ้าแก้ ConfigMap แล้ว ตัว process ที่รันอยู่แล้วจะ**ไม่เห็นค่าใหม่เลย** จนกว่าจะถูก restart (เช่นสั่ง `kubectl rollout restart deployment/api`)
- **Mounted file** — kubelet จะ sync เนื้อหาไฟล์ในไดเรกทอรีที่ mount ไว้ให้ตรงกับ ConfigMap ปัจจุบันเป็นระยะ (โดยทั่วไปภายในไม่กี่สิบวินาทีถึง 1-2 นาที ขึ้นกับ cache sync interval ของ kubelet) **โดยไม่ต้อง restart Pod** — แต่ application เองต้องเขียนโค้ดรองรับการอ่านค่าใหม่จากไฟล์ (เช่น watch ไฟล์เปลี่ยนแปลง หรืออ่านใหม่เป็นระยะ) ไม่ใช่ว่า mount แล้วจะ "auto-reload" ให้ฟรีๆ ถ้า application อ่านค่าจากไฟล์แค่ตอน startup เหมือนกันก็จะพลาดค่าที่อัปเดตอยู่ดี

```
Env Var:                              Mounted File:
ConfigMap เปลี่ยน                     ConfigMap เปลี่ยน
      │                                     │
      ▼ (ไม่มีผลกับ process ที่รันอยู่)      ▼ kubelet sync ไฟล์ให้อัตโนมัติ
Pod ต้อง restart ถึงเห็นค่าใหม่          ไฟล์ใน container อัปเดตโดยไม่ต้อง restart Pod
                                             │
                                             ▼ (แต่ application ต้องอ่านไฟล์ใหม่เองด้วย)
```

หลักการเลือก: ถ้าต้องการให้เปลี่ยน config ได้แบบ dynamic โดยไม่ downtime (เช่น log level, feature flag) ควรออกแบบให้อ่านจาก mounted file พร้อม logic reload ในแอป; ถ้าเป็นค่าที่ไม่ค่อยเปลี่ยนหรือ restart ได้ไม่มีปัญหา ใช้ env var ก็เพียงพอและง่ายกว่า

## 2.3 Secret

### โครงสร้างคล้าย ConfigMap แต่เก็บข้อมูลอ่อนไหว

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: api-db-credentials
type: Opaque
data:
  DB_PASSWORD: c3VwZXJzZWNyZXQxMjM=
```

การใช้งานใน Deployment เหมือน ConfigMap ทุกประการ (ผ่าน `envFrom.secretRef` หรือ `volumes.secret`) — ความต่างหลักอยู่ที่วิธีจัดเก็บและสิทธิ์การเข้าถึง

### ข้อเท็จจริงที่ต้องเข้าใจให้ชัด: Base64 ไม่ใช่การเข้ารหัส

ค่าใน `data` ของ Secret ถูกเก็บเป็น **base64-encoded string** — และต้องพูดให้ชัดเจนตรงนี้: **base64 ไม่ใช่การเข้ารหัส (encryption) ไม่ใช่แม้แต่การป้องกันเบื้องต้น** มันคือแค่การ**เข้ารหัสรูปแบบ (encoding)** เพื่อให้ binary/ข้อความที่มีอักขระพิเศษเก็บเป็น plain text ใน YAML ได้เท่านั้น การ decode กลับทำได้ในบรรทัดเดียวโดยไม่ต้องมี key ใดๆ เลย:

```bash
echo "c3VwZXJzZWNyZXQxMjM=" | base64 -d
# supersecret123
```

**สิ่งที่ base64 "ป้องกัน" ได้จริงๆ คือ "ไม่มีอะไรเลย"** — มันไม่ป้องกันไม่ให้คนที่มีสิทธิ์อ่าน object นั้น (ผ่าน `kubectl get secret -o yaml`) เห็นค่าจริง ไม่ป้องกันการรั่วไหลถ้า etcd (ที่เก็บ Secret จริงๆ) ไม่ได้ตั้งค่า encryption-at-rest ไว้ (ดู [บทที่ 4 — etcd](04-scaling-and-control-plane.md)) และไม่ป้องกัน Secret ที่ commit ลง Git repository โดยไม่ตั้งใจ (base64 encode ยังคง decode กลับเป็นความลับได้ทันทีในบรรทัดเดียว) — บทบาทของ base64 ในที่นี้เป็นแค่เรื่อง**การจัดรูปแบบข้อมูล (data format)** เพื่อให้เก็บใน YAML/JSON ที่เป็น text-based ได้เท่านั้น ไม่ใช่เรื่อง security

### สิ่งที่ Kubernetes Secret ให้จริงๆ เมื่อเทียบกับ ConfigMap

- RBAC แยกสิทธิ์ได้ละเอียดกว่า (จำกัดว่า role ไหนอ่าน Secret ได้บ้าง)
- ถ้าตั้งค่า **encryption at rest** ของ etcd ไว้ (ไม่ใช่ default) ข้อมูลจะถูกเข้ารหัสจริงตอนเก็บลง disk
- mount เป็น `tmpfs` (in-memory) แทนที่จะเขียนลง disk ของ node โดย default

แต่สิ่งเหล่านี้เป็น**การป้องกันระดับ infrastructure** ไม่ใช่การเข้ารหัสตัวข้อมูลเองที่ระดับ object — ใครก็ตามที่มีสิทธิ์ `kubectl get secret` บน namespace นั้นเห็นค่าจริงได้เสมอ

## 2.4 ทางออกที่แท้จริงสำหรับ Secret ระดับ Production: Vault

Kubernetes Secret เหมาะกับ use case พื้นฐานหรือ workload ขนาดเล็ก แต่สำหรับระบบระดับ enterprise ที่ต้องการ **dynamic secret** (credential ที่หมดอายุอัตโนมัติ), **audit trail** ว่าใครเข้าถึง secret ตัวไหนเมื่อไหร่, และ **rotation** อัตโนมัติโดยไม่ต้อง restart Pod ด้วยมือ คำตอบที่ถูกต้องคือเครื่องมือจัดการ secret โดยเฉพาะอย่าง **HashiCorp Vault**

(เชื่อมกับ [Volume 15 — Security](../15-security/README.md) ที่อธิบาย Vault ในรายละเอียด — แนวคิดหลักคือแทนที่จะเก็บ static credential ไว้ใน Secret ตรงๆ, application จะขอ credential ชั่วคราวจาก Vault ผ่าน authentication method อย่าง Kubernetes auth ทำให้แต่ละ credential มีอายุจำกัดและถูก revoke ได้ทันทีถ้าจำเป็น ต่างจาก Kubernetes Secret ที่เป็น static value ค้างอยู่จนกว่าจะมีคนไปแก้เอง)

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| 12-factor config | แยก config ออกจาก image เพื่อให้ artifact เดียวกัน deploy ได้ทุก environment |
| ConfigMap: env var | อ่านตอน start เท่านั้น, แก้ ConfigMap แล้วต้อง restart Pod ถึงเห็นค่าใหม่ |
| ConfigMap: mounted file | kubelet sync ไฟล์ให้อัตโนมัติโดยไม่ต้อง restart Pod แต่ application ต้องอ่านไฟล์ใหม่เอง |
| Secret vs ConfigMap | โครงสร้างเหมือนกัน ต่างที่ RBAC, encryption-at-rest (ถ้าตั้งค่า), mount แบบ tmpfs |
| Base64 | เป็นแค่ encoding ไม่ใช่ encryption — decode กลับได้ทันทีโดยไม่ต้องมี key ใดๆ |
| Production secret | ใช้ Vault สำหรับ dynamic secret, audit, rotation แทนการพึ่ง static Kubernetes Secret |

## คำถามทบทวน

1. อธิบายว่าทำไม image เดียวกันที่ build ครั้งเดียวถึงควร deploy ได้ทั้ง dev/staging/production โดยไม่ต้อง build ใหม่ เกี่ยวข้องกับหลัก 12-factor อย่างไร
2. ถ้าแก้ค่าใน ConfigMap ที่ผูกกับ environment variable ของ Pod ที่กำลังรันอยู่ ต้องทำอะไรเพิ่มถึงจะเห็นผล เปรียบเทียบกับกรณี mount เป็นไฟล์
3. เพราะเหตุใด base64 ถึงไม่ถือว่าเป็นมาตรการรักษาความปลอดภัย ทดลองอธิบายด้วยตัวอย่างคำสั่งที่ decode กลับได้
4. Kubernetes Secret ให้การป้องกันอะไรบ้างที่ ConfigMap ไม่มี และสิ่งเหล่านั้นแก้ปัญหาเรื่อง "ใครก็อ่านค่าจริงได้ถ้ามีสิทธิ์" หรือไม่
5. Vault แก้ปัญหาอะไรที่ Kubernetes Secret แบบ static ทำไม่ได้ ยกตัวอย่างสถานการณ์ที่ dynamic secret สำคัญ

---

ก่อนหน้า: [บทที่ 1 — Pod, Deployment, Service, Ingress](01-workloads-and-networking.md) | ถัดไป: [บทที่ 3 — StatefulSet, DaemonSet, PVC, StorageClass](03-stateful-workloads-and-storage.md)
