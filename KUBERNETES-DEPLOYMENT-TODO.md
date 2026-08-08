# TODO: Kubernetes Deployment สำหรับ BOP (Design เต็ม ยังไม่ Implement)

> **สถานะ**: prerequisite เชิงโค้ด + Dockerfile + K8s manifest ทั้งหมด **implement แล้ว** (ดู checklist ด้านล่าง) — **ยังไม่เคยทดสอบกับ `docker build`/`kind` cluster จริงเลย** เพราะ environment ที่เขียนโค้ดนี้ไม่มีสิทธิ์เข้าถึง Docker daemon และไม่มี `kind`/`kubectl` ติดตั้ง ต้องรัน checklist ข้อ "ทดสอบ" ทั้งหมดด้วยมือ — เอกสารนี้คือ [FULLSTACK-INFRA-ROADMAP.md](FULLSTACK-INFRA-ROADMAP.md) Phase 2.1 ("Containerize + Deploy พื้นฐาน") ขยายรายละเอียดเต็มรูปแบบ

**ทำไมเอกสารนี้อยู่ที่ root ของ workspace ไม่ใช่ในโฟลเดอร์ project ใดโฟลเดอร์หนึ่ง**: เพราะ deployment topology นี้ครอบคลุม **3-4 repo แยกกัน** (`projects/bot-orchestration-platform/` มี 2 image คือ `cmd/bop` + `cmd/bop-worker`, `projects/bop-dashboard/` อีก 1 image, `projects/remote-job-worker/` อีก 1 image ถ้า deploy จริง) ไม่มี repo ไหนเป็นเจ้าของภาพรวมทั้งหมดคนเดียว เหมือนกับที่ `FULLSTACK-INFRA-ROADMAP.md` เองก็อยู่ที่ root ด้วยเหตุผลเดียวกัน

**เชื่อมกับงานที่ทำค้างไว้แล้ว**: เอกสารนี้คือ prerequisite ตัวจริงของ [K8S-JOB-HYBRID-EXECUTION-TODO.md](projects/bot-orchestration-platform/K8S-JOB-HYBRID-EXECUTION-TODO.md) ที่ implement โค้ดเสร็จไปแล้ว (`internal/workflow/k8sjob.go`) แต่ "ยังไม่เคยทดสอบกับ K8s cluster จริงเลย" — Open Question ข้อ 2/3/5 ของเอกสารนั้น (RBAC, namespace, resource limit) ถูกตอบละเอียดในเอกสารนี้

## สารบัญ

1. [Cluster Topology และกลยุทธ์ Environment](#1-cluster-topology-และกลยุทธ์-environment)
2. [Containerization (Dockerfile ต่อ component)](#2-containerization-dockerfile-ต่อ-component)
3. [K8s Manifests หลัก (Deployment/Service/Ingress/Config)](#3-k8s-manifests-หลัก-deploymentserviceingressconfig)
4. [RBAC สำหรับ K8s Job Hybrid Execution](#4-rbac-สำหรับ-k8s-job-hybrid-execution)
5. [Resource Quota / LimitRange สำหรับ namespace bop-jobs](#5-resource-quota--limitrange-สำหรับ-namespace-bop-jobs)
6. [Temporal บน Kubernetes](#6-temporal-บน-kubernetes)
7. [Health Check Endpoints (ต้องเพิ่มโค้ดใหม่)](#7-health-check-endpoints-ต้องเพิ่มโค้ดใหม่)
8. [Scaling](#8-scaling)
9. [Local Dev Workflow ด้วย kind](#9-local-dev-workflow-ด้วย-kind)
10. [Open Questions](#open-questions)
11. [TODO Checklist](#todo-checklist)

---

## 1. Cluster Topology และกลยุทธ์ Environment

### แนะนำ: `kind` (Kubernetes-in-Docker) สำหรับตอนนี้ ไม่ใช่ cloud managed K8s

เหตุผล (YAGNI เหมือนกับทุกการตัดสินใจอื่นในโปรเจกต์นี้): นี่คือ portfolio project ที่ demo ได้จากเครื่อง dev — จ่ายเงินเปิด GKE/EKS/AKS ทิ้งไว้ทั้งเดือนเพื่อ cluster ที่ demo เป็นครั้งคราวไม่คุ้ม `kind` รัน K8s จริง (ใช้ containerd ข้างในเหมือน production) ผ่าน Docker container ธรรมดา ฟรี รันบนเครื่อง dev เดียวกับที่รัน `docker compose up -d` (Temporal) อยู่แล้วได้เลย — **พฤติกรรมของ manifest ที่ทดสอบผ่านบน kind ตรงกับ cloud K8s เป๊ะ** (ทั้งคู่เป็น K8s conformant) ย้ายไป cloud จริงทีหลังแค่เปลี่ยน kubeconfig context ไม่ต้องแก้ manifest เลย

### Namespace Design

```
bop-system      Deployment/Service/Ingress ของ cmd/bop, cmd/bop-worker, dashboard, Temporal (ถ้า self-host)
bop-jobs        Job ที่สร้างโดย K8sJobActivities (RunKubernetesJob) เท่านั้น — ตรงกับ
                k8sJobDefaultNamespace ที่ hardcode ไว้แล้วใน cmd/bop-worker/main.go
```

**เหตุผลที่แยก 2 namespace** (ตอบ Open Question ข้อ 3 ของ K8S-JOB-HYBRID-EXECUTION-TODO.md): blast radius — ถ้า job ที่ผู้ใช้สั่งผ่าน `K8sJobSpec.Image` (image อะไรก็ได้ที่ผู้ใช้ระบุเอง) มีปัญหา (bug, resource leak, malicious image) จะกระทบแค่ namespace `bop-jobs` ไม่ลามไปกระทบ `cmd/bop`/`cmd/bop-worker` เองที่รันอยู่ใน `bop-system` — ResourceQuota/NetworkPolicy ก็ตั้งกฎคนละชุดกันได้อิสระ (ดูหัวข้อ 5)

---

## 2. Containerization (Dockerfile ต่อ component)

**สถานะปัจจุบัน: ไม่มี Dockerfile เลยสักไฟล์ในทั้ง 2 repo — ต้องเขียนใหม่ทั้งหมด 3 ไฟล์**

### 2.1 `projects/bot-orchestration-platform/cmd/bop/Dockerfile`

Multi-stage build — stage แรก compile binary ด้วย image ที่มี Go toolchain เต็ม (~800MB) stage สุดท้ายมีแค่ binary + CA certificates (~20MB) ไม่ลาก toolchain ติดไปด้วย:

```dockerfile
# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# CGO_ENABLED=0 บังคับ static binary (ไม่ผูกกับ libc ของ image ต้นทาง) จำเป็นเพราะ
# stage สุดท้ายใช้ distroless ที่ไม่มี dynamic linker ให้เลย
RUN CGO_ENABLED=0 GOOS=linux go build -o /bop ./cmd/bop

FROM gcr.io/distroless/static-debian12:nonroot
# distroless "nonroot" variant รันเป็น UID 65532 อัตโนมัติอยู่แล้ว (ไม่ต้องสร้าง user เอง
# เหมือน image ทั่วไป) — ไม่มี shell/package manager ในนี้เลยด้วยซ้ำ (ลด attack surface
# ให้เหลือน้อยที่สุด ตรงกับ Phase 2.7 Security ที่วางแผนไว้)
COPY --from=builder /bop /bop
EXPOSE 8080
ENTRYPOINT ["/bop"]
```

### 2.2 `projects/bot-orchestration-platform/cmd/bop-worker/Dockerfile`

โครงเดียวกันเป๊ะ แค่ build target ต่างกัน (`./cmd/bop-worker`) — ไม่มี `EXPOSE` เพราะ process นี้ไม่เปิด HTTP server (จนกว่าจะเพิ่ม health check ตามหัวข้อ 7 ที่ต้องเปิด port เล็กๆ ไว้ตอบ probe)

### 2.3 `projects/bop-dashboard/Dockerfile`

Next.js รองรับ **`output: "standalone"`** ใน `next.config.ts` ที่ bundle เฉพาะไฟล์ที่ runtime ต้องใช้จริง (ไม่ต้อง `npm install` ใน production image เลย) — ต้องเพิ่ม `output: "standalone"` เข้า `next.config.ts` ก่อน (ยังไม่มีตอนนี้):

```dockerfile
# syntax=docker/dockerfile:1
FROM node:22-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

FROM node:22-alpine AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

FROM node:22-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
RUN addgroup --system --gid 1001 nodejs && adduser --system --uid 1001 nextjs
COPY --from=builder /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static
USER nextjs
EXPOSE 3000
ENV PORT=3000
CMD ["node", "server.js"]
```

### 2.4 `.dockerignore` (ทั้ง 2 repo)

ต้องมีทั้งคู่ กัน build context บวมและกัน secret หลุดเข้า image โดยไม่ตั้งใจ:

```
.git
node_modules
.next
*.md
docker-compose.yml
```

---

## 3. K8s Manifests หลัก (Deployment/Service/Ingress/Config)

### 3.1 ปัญหาที่ต้องแก้ก่อนเขียน manifest: ค่า config ยัง hardcode เป็น Go constant ทั้งหมด

โค้ดปัจจุบันมี comment กำกับไว้ทุกจุดแล้วว่า "hardcode ไปก่อน...พอ deploy จริงค่อยเปลี่ยนเป็นอ่านจาก environment variable" (`":8080"` ใน `cmd/bop/main.go`, `"localhost:7233"` ทั้ง `cmd/bop` และ `cmd/bop-worker`, `authSecret`/`adminUsername`/`adminPassword` ใน `handler.go`, `k8sJobDefaultNamespace` ใน `cmd/bop-worker/main.go`) — **จังหวะนี้คือจังหวะที่ต้องแก้จริงแล้ว** ก่อนเขียน manifest ไม่งั้น `ConfigMap`/`Secret` ในหัวข้อ 3.3/3.4 ไม่มีอะไรให้ inject เข้าไปเลย

ตัวอย่างการแก้ (`cmd/bop/main.go`):
```go
addr := os.Getenv("BOP_HTTP_ADDR")
if addr == "" {
    addr = ":8080" // fallback ค่าเดิมสำหรับ local dev ที่ไม่ได้ตั้ง env var
}
temporalHostPort := os.Getenv("TEMPORAL_HOST_PORT")
if temporalHostPort == "" {
    temporalHostPort = "localhost:7233"
}
```
รูปแบบเดียวกันนี้ต้องทำกับทุกค่าที่ตอนนี้ hardcode อยู่ — **นี่คือ prerequisite เชิงโค้ดของเอกสารนี้** ไม่ใช่แค่เรื่อง manifest

### 3.2 Deployment (ตัวอย่าง `cmd/bop`)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: bop-api
  namespace: bop-system
spec:
  replicas: 2
  selector:
    matchLabels: { app: bop-api }
  template:
    metadata:
      labels: { app: bop-api }
    spec:
      containers:
        - name: bop-api
          image: <registry>/bop-api:<tag>   # tag = git SHA — ดู JENKINS-CICD-TODO.md
          ports:
            - containerPort: 8080
          envFrom:
            - configMapRef: { name: bop-config }
            - secretRef: { name: bop-secrets }
          resources:
            requests: { cpu: 100m, memory: 128Mi }
            limits: { cpu: 500m, memory: 256Mi }
          livenessProbe:
            httpGet: { path: /healthz, port: 8080 }
            initialDelaySeconds: 5
            periodSeconds: 10
          readinessProbe:
            httpGet: { path: /readyz, port: 8080 }
            initialDelaySeconds: 5
            periodSeconds: 5
      # securityContext ระดับ Pod: ไม่ต้องรัน root, ปิด privilege escalation — ตรงกับที่
      # distroless "nonroot" image (หัวข้อ 2.1) รันเป็น non-root อยู่แล้ว ยืนยันซ้ำอีกชั้น
      securityContext:
        runAsNonRoot: true
        seccompProfile: { type: RuntimeDefault }
```

`cmd/bop-worker` ใช้โครงเดียวกัน (ต่าง image, ไม่มี `ports`/`Service` เพราะไม่รับ HTTP traffic จากภายนอกเลย — แต่ยังต้องมี `livenessProbe` ผ่าน health server เล็กๆ ตามหัวข้อ 7) `replicas` ของ `bop-worker` ปรับตามปริมาณ Activity/Workflow ที่ต้องรันพร้อมกัน (Temporal เองกระจายงานข้าม replica ให้อัตโนมัติผ่าน task queue — ไม่ต้องเขียน load balancing logic เอง)

### 3.3 ConfigMap (ค่าไม่ลับ)

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: bop-config
  namespace: bop-system
data:
  BOP_HTTP_ADDR: ":8080"
  TEMPORAL_HOST_PORT: "temporal-frontend.bop-system.svc.cluster.local:7233"
  K8S_JOB_DEFAULT_NAMESPACE: "bop-jobs"
```

### 3.4 Secret (ค่าลับ — แก้ปัญหา `admin`/`changeme` ที่ hardcode มาตั้งแต่ต้น)

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: bop-secrets
  namespace: bop-system
type: Opaque
stringData:
  JWT_SECRET: "<สุ่มใหม่ด้วย openssl rand -base64 32 — ห้ามใช้ค่า dev เดิมเด็ดขาด>"
  ADMIN_USERNAME: "<กำหนดเอง>"
  ADMIN_PASSWORD: "<สุ่มใหม่ ห้ามเป็น changeme>"
```

**ห้าม commit ไฟล์ที่มี `stringData` จริงเข้า git เด็ดขาด** — ใน production ควรใช้ [Sealed Secrets](https://github.com/bitnami-labs/sealed-secrets) หรือ external secret manager (เช่น cloud provider's secret manager ผ่าน External Secrets Operator) แทนการ apply Secret manifest ตรงๆ — เอกสารนี้แค่โชว์ shape ของ Secret ที่โค้ดต้องอ่าน ไม่ใช่ค่าจริง

### 3.5 Service + Ingress

```yaml
apiVersion: v1
kind: Service
metadata: { name: bop-api, namespace: bop-system }
spec:
  selector: { app: bop-api }
  ports: [{ port: 80, targetPort: 8080 }]
---
apiVersion: v1
kind: Service
metadata: { name: bop-dashboard, namespace: bop-system }
spec:
  selector: { app: bop-dashboard }
  ports: [{ port: 80, targetPort: 3000 }]
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: bop-ingress
  namespace: bop-system
  annotations:
    # cert-manager ออก TLS cert อัตโนมัติผ่าน Let's Encrypt — ดู Open Question เรื่อง TLS
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
    - hosts: [bop.example.com]
      secretName: bop-tls
  rules:
    - host: bop.example.com
      http:
        paths:
          - path: /api
            pathType: Prefix
            backend: { service: { name: bop-api, port: { number: 80 } } }
          - path: /
            pathType: Prefix
            backend: { service: { name: bop-dashboard, port: { number: 80 } } }
```

**หมายเหตุ**: path `/api` ตรงนี้หมายความว่าต้องแก้ dashboard ให้เรียก backend ผ่าน path เดียวกันแทน CORS ข้าม origin แบบตอน local dev (`dashboardOrigin` constant ใน `handler.go`) — ทางเลือกอื่นคือคง 2 subdomain แยกกัน (`api.bop.example.com`/`app.bop.example.com`) แล้วคง CORS ไว้เหมือนเดิม เลือกทางไหนกระทบโค้ด frontend/backend ต่างกัน (ดู Open Questions)

---

## 4. RBAC สำหรับ K8s Job Hybrid Execution

ตอบ Open Question ข้อ 2 ของ [K8S-JOB-HYBRID-EXECUTION-TODO.md](projects/bot-orchestration-platform/K8S-JOB-HYBRID-EXECUTION-TODO.md) โดยตรง — `cmd/bop-worker` (ที่รัน `K8sJobActivities.RunKubernetesJob`) ต้องมีสิทธิ์สร้าง/ดู/ลบ `Job` และอ่าน log ของ `Pod` **เฉพาะใน namespace `bop-jobs` เท่านั้น** ไม่ใช่ cluster-wide — ใช้ `Role`+`RoleBinding` (ไม่ใช่ `ClusterRole`+`ClusterRoleBinding`) เพื่อ scope ให้แคบที่สุด:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: bop-worker
  namespace: bop-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: bop-job-runner
  namespace: bop-jobs   # สิทธิ์นี้ผูกกับ namespace bop-jobs เท่านั้น ไม่ข้ามไป bop-system
rules:
  - apiGroups: ["batch"]
    resources: ["jobs"]
    verbs: ["create", "get", "list", "watch", "delete"]
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list"]
  - apiGroups: [""]
    resources: ["pods/log"]
    verbs: ["get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: bop-worker-job-runner
  namespace: bop-jobs
subjects:
  - kind: ServiceAccount
    name: bop-worker
    namespace: bop-system   # ServiceAccount อยู่ namespace bop-system แต่ได้สิทธิ์ใน bop-jobs
roleRef:
  kind: Role
  name: bop-job-runner
  apiGroup: rbac.authorization.k8s.io
```

ต้องผูก `serviceAccountName: bop-worker` เข้า `Deployment` ของ `cmd/bop-worker` (หัวข้อ 3.2) ด้วย — พอ Pod รันด้วย ServiceAccount นี้แล้ว `rest.InClusterConfig()` ใน `cmd/bop-worker/main.go` (ที่ทำ best-effort fallback ไว้อยู่แล้ว) จะสำเร็จอัตโนมัติ (K8s mount token ของ ServiceAccount ให้ Pod เองโดยไม่ต้องตั้งค่าอะไรเพิ่มในโค้ด Go เลย)

---

## 5. Resource Quota / LimitRange สำหรับ namespace `bop-jobs`

ตอบ Open Question ข้อ 5 ของ K8S-JOB-HYBRID-EXECUTION-TODO.md ("`K8sJobSpec` ไม่ได้ตั้ง `resources.requests`/`limits` ให้ container เลย") — แก้ที่ระดับ namespace แทนที่จะบังคับผ่าน field ใหม่ใน `K8sJobSpec` (เลือกทางนี้เพราะ **ไม่ต้องแก้โค้ด Go เลยสักบรรทัด** — K8s เติม default resource ให้ container ที่ไม่ได้ระบุเองโดยอัตโนมัติผ่าน `LimitRange`):

```yaml
apiVersion: v1
kind: LimitRange
metadata:
  name: bop-jobs-default-limits
  namespace: bop-jobs
spec:
  limits:
    - type: Container
      default: { cpu: "500m", memory: "512Mi" }        # ใช้ถ้า container ไม่ได้ระบุ limit เอง
      defaultRequest: { cpu: "100m", memory: "128Mi" } # ใช้ถ้า container ไม่ได้ระบุ request เอง
---
apiVersion: v1
kind: ResourceQuota
metadata:
  name: bop-jobs-quota
  namespace: bop-jobs
spec:
  hard:
    pods: "20"              # กัน job รันพร้อมกันเยอะเกินจนกิน node ทั้งลูก
    requests.cpu: "4"
    requests.memory: 8Gi
    limits.cpu: "8"
    limits.memory: 16Gi
```

---

## 6. Temporal บน Kubernetes

`docker-compose.yml` ปัจจุบันรัน Temporal server + PostgreSQL ของ Temporal เองสำหรับ local dev เท่านั้น — ต้องเลือก 1 ใน 2 ทางสำหรับ production:

1. **Self-host ผ่าน [Temporal Helm chart](https://github.com/temporalio/helm-charts)** — ควบคุมเองได้เต็มที่ ไม่มีค่าใช้จ่ายรายเดือนเพิ่ม แต่ต้องดูแล PostgreSQL/Elasticsearch ของ Temporal เอง (operational burden เพิ่ม)
2. **Temporal Cloud (managed)** — ไม่ต้องดูแล infra เลย จ่ายตาม usage แต่มีค่าใช้จ่าย และ `client.Dial` ต้องเปลี่ยนไปใช้ mTLS แทน plaintext localhost

**แนะนำสำหรับ portfolio/demo**: self-host ผ่าน Helm chart ใน namespace `bop-system` เดียวกัน (ค่าใช้จ่าย $0, ตรงกับ ethos "ไม่จ่ายเงินเปิด service ทิ้งไว้เฉยๆ" ของทั้งโปรเจกต์) — เปลี่ยนไป Temporal Cloud ได้ทีหลังถ้าต้องการ "enterprise SLA story" เพิ่มตอนสัมภาษณ์ (แค่เปลี่ยน `TEMPORAL_HOST_PORT` env var + เพิ่ม mTLS cert ไม่ต้องแก้ business logic เลย เพราะโค้ดคุยผ่าน Temporal SDK เดียวกันทั้งคู่)

---

## 7. Health Check Endpoints (ต้องเพิ่มโค้ดใหม่)

**ยังไม่มีเลยตอนนี้** — `internal/api/http/handler.go` ไม่มี route `/healthz`/`/readyz` เลย จำเป็นต้องเพิ่มก่อนถึงจะเขียน `livenessProbe`/`readinessProbe` (หัวข้อ 3.2) ได้จริง:

```go
// GET /healthz — liveness: process ยังตอบสนองอยู่ไหม (ไม่เช็ค dependency ภายนอกเลย
// ตั้งใจ — liveness fail = K8s restart Pod ทันที ไม่ควร restart แค่เพราะ Temporal
// server ช้าชั่วคราว นั่นเป็นหน้าที่ของ readiness ไม่ใช่ liveness)
s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
})

// GET /readyz — readiness: process พร้อมรับ traffic จริงไหม (เช็คว่าต่อ Temporal ได้อยู่)
// K8s หยุดส่ง traffic มาที่ Pod นี้ชั่วคราว (ไม่ restart) ถ้า readiness fail — ต่างจาก
// liveness ตรงนี้แหละ: Temporal ล่มชั่วคราวไม่ควรทำให้ Pod ถูก restart วนไปเรื่อยๆ
// (CrashLoopBackOff) แค่หยุดรับ traffic จนกว่า Temporal จะกลับมา
s.mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
    if _, err := s.temporal.CheckHealth(r.Context(), &client.CheckHealthRequest{}); err != nil {
        http.Error(w, "temporal unreachable", http.StatusServiceUnavailable)
        return
    }
    w.WriteHeader(http.StatusOK)
})
```

`cmd/bop-worker` ไม่มี HTTP server เลยตอนนี้ (ทำงานแค่ poll Temporal task queue) — ต้องเปิด HTTP server เล็กๆ แยกต่างหากเฉพาะไว้ตอบ probe (`http.ListenAndServe(":9090", ...)` รันใน goroutine แยก คู่ขนานกับ `w.Run(worker.InterruptCh())` เดิม) มิฉะนั้น K8s ไม่มีทางรู้เลยว่า worker process ยัง alive อยู่ไหม

---

## 8. Scaling

- **`bop-api`**: `HorizontalPodAutoscaler` อิง CPU utilization ธรรมดา (HTTP request-driven workload มาตรฐาน) — ต้องมี `metrics-server` ติดตั้งใน cluster ก่อน (มากับ `kind` default ไม่ครบ ต้องติดตั้งเพิ่มเอง)
- **`bop-worker`**: scale ผ่านจำนวน `replicas` ธรรมดา (ไม่ต้อง HPA ก็ได้ในระดับ demo) — Temporal เองกระจายงานข้าม worker replica ผ่าน task queue polling อัตโนมัติ (เหมือนที่อธิบายไว้ใน `README.md` เรื่อง "ทำไมแยก process") **ข้อควรระวัง**: `worker.Options.MaxConcurrentActivityExecutionSize` (ยังไม่ได้ตั้งค่าตอนนี้ — ใช้ default ของ SDK) ควบคุม concurrency **ภายใน** 1 replica เท่านั้น การเพิ่ม replica คือมิติที่แยกต่างหากจากมิตินี้

---

## 9. Local Dev Workflow ด้วย `kind`

```bash
# 1) สร้าง cluster local (ทำครั้งเดียว)
kind create cluster --name bop

# 2) build image ในเครื่อง dev แล้ว "load" เข้า kind ตรงๆ (ไม่ต้อง push ขึ้น registry
#    จริงตอน dev — kind มี mechanism โหลด image ท้องถิ่นเข้า cluster ให้เลย)
docker build -t bop-api:dev -f projects/bot-orchestration-platform/cmd/bop/Dockerfile projects/bot-orchestration-platform
kind load docker-image bop-api:dev --name bop

# 3) apply manifest ทั้งหมด (namespace ก่อนเสมอ แล้วค่อย RBAC/config/quota/deployment/
#    service/ingress — ทุกไฟล์อยู่ใน k8s/ ที่ root ของ workspace)
kubectl apply -f k8s/namespaces.yaml
kubectl apply -f k8s/rbac.yaml
kubectl apply -f k8s/config.yaml
kubectl apply -f k8s/resource-quota.yaml
kubectl apply -f k8s/deployments.yaml
kubectl apply -f k8s/services.yaml
kubectl apply -f k8s/ingress.yaml   # ต้องมี ingress controller ใน cluster ก่อน (เช่น ingress-nginx) ไม่งั้น apply ผ่านแต่ไม่มีอะไรรับ traffic จริง

# 4) ดู log/สถานะ
kubectl -n bop-system get pods -w
kubectl -n bop-system logs deploy/bop-worker -f
```

---

## Open Questions

1. ~~**Ingress path-based (`/api` + `/`) หรือ subdomain แยก (`api.`/`app.`)**~~ — **ตัดสินใจแล้ว: subdomain แยก** (`k8s/ingress.yaml`) เพราะไม่ต้องแก้โค้ด frontend ให้เรียกผ่าน path prefix ใหม่ (backend ยังคง CORS ผ่าน `BOP_DASHBOARD_ORIGIN` เหมือนเดิม) — เปลี่ยนเป็น path-based ทีหลังได้ถ้าต้องการ แต่ต้องแก้ CORS logic ฝั่ง backend คู่กับ path prefix ฝั่ง frontend
2. **Cloud provider สำหรับ deploy จริง (ถ้าอยากมี live demo link ให้คนสัมภาษณ์กด) หรือพอแค่ `kind` + สาธิตสด** — งบประมาณ vs ความสมจริง
3. **Temporal: self-host ผ่าน Helm chart หรือ Temporal Cloud** — ดูหัวข้อ 6
4. **cert-manager (TLS อัตโนมัติ) หรือจัดการ cert เอง** — ถ้ามี cloud domain จริงแนะนำ cert-manager
5. **Sealed Secrets/External Secrets Operator หรือจัดการ Secret ด้วยมือ** — สำคัญถ้าจะ commit manifest เข้า git จริง (GitOps)
6. **Kustomize หรือ Helm chart ของ BOP เอง** — จัดการความต่างระหว่าง local (`kind`) กับ production overlay

## TODO Checklist

### 1. Prerequisite เชิงโค้ด (ก่อนเขียน manifest ได้จริง)
- [x] แก้ `cmd/bop/main.go`/`cmd/bop-worker/main.go` ให้อ่าน `addr`/`TEMPORAL_HOST_PORT`/ค่าอื่นๆ จาก env var (fallback เป็นค่า hardcode เดิมถ้าไม่ได้ตั้ง)
- [x] แก้ `internal/api/http/handler.go` ให้อ่าน `authSecret`/`adminUsername`/`adminPassword`/`dashboardOrigin` จาก env var (`BOP_JWT_SECRET`/`BOP_ADMIN_USERNAME`/`BOP_ADMIN_PASSWORD`/`BOP_DASHBOARD_ORIGIN`)
- [x] เพิ่ม `GET /healthz`/`GET /readyz` ใน `cmd/bop` (หัวข้อ 7)
- [x] เพิ่ม HTTP server เล็กๆ ใน `cmd/bop-worker` สำหรับ probe (หัวข้อ 7) — `GET /healthz` บน `:9090`
- [x] เพิ่ม `output: "standalone"` ใน `next.config.ts` ของ dashboard
- [x] `go build ./...`, `go vet ./...`, `go test ./...` ผ่านหมดหลังแก้ (ยืนยันแล้ว)

### 2. Containerization
- [x] เขียน `Dockerfile` ทั้ง 3 ไฟล์ (หัวข้อ 2) + `.dockerignore`
- [ ] ทดสอบ `docker build` local ก่อน ยังไม่ต้อง push — **ยังไม่ได้ทดสอบ** environment ที่เขียนโค้ดนี้ต่อ Docker daemon ไม่ได้ (permission denied) ต้องรันเองที่เครื่อง dev

### 3. K8s Manifests
- [x] เขียน manifest ทั้งหมด (namespace, ConfigMap, Secret ตัวอย่าง, Deployment, Service, Ingress, RBAC, LimitRange, ResourceQuota) — เก็บใน `k8s/` ที่ root แล้ว (`namespaces.yaml`, `rbac.yaml`, `config.yaml`, `resource-quota.yaml`, `deployments.yaml`, `services.yaml`, `ingress.yaml`) — YAML syntax ตรวจผ่านแล้วด้วย `yaml.safe_load_all` แต่ยังไม่เคย `kubectl apply` จริง
- [ ] ทดสอบบน `kind` ตามหัวข้อ 9 ก่อนคิดเรื่อง cloud — **ยังไม่ได้ทดสอบ** ไม่มี `kind`/`kubectl` ใน environment นี้ ต้องรันเองที่เครื่อง dev

### 4. Verify K8s Job Hybrid Execution จริง
- [ ] ยิง `POST /workflow-runs` ที่มี step `k8s_job` จริง ยืนยันว่า `RunKubernetesJob` สร้าง Job ใน `bop-jobs` ได้จริง (ปิด loop ของ K8S-JOB-HYBRID-EXECUTION-TODO.md ที่ค้างมาตั้งแต่ implement) — รอ `kind` cluster จริงจากข้อ 3
- [ ] ยืนยัน RBAC ทำงานถูกต้อง (bop-worker สร้าง/ลบ Job ได้ แต่แตะ namespace อื่นไม่ได้) — รอ `kind` cluster จริงจากข้อ 3

## Reference

- [FULLSTACK-INFRA-ROADMAP.md](FULLSTACK-INFRA-ROADMAP.md) Phase 2.1
- [projects/bot-orchestration-platform/K8S-JOB-HYBRID-EXECUTION-TODO.md](projects/bot-orchestration-platform/K8S-JOB-HYBRID-EXECUTION-TODO.md) — โค้ดที่รอ cluster จริงมาทดสอบ
- [JENKINS-CICD-TODO.md](JENKINS-CICD-TODO.md) — pipeline ที่ build/push image ตามเอกสารนี้แล้ว deploy อัตโนมัติ
- [books/07-docker/](books/07-docker/README.md), [books/08-kubernetes/](books/08-kubernetes/README.md) — เนื้อหาทฤษฎีเต็ม (ยังไม่มีบทเรื่อง `batch/v1 Job`/`LimitRange`/`ResourceQuota` โดยตรง อาจต้องเสริม)
