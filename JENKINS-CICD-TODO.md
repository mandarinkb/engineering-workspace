# TODO: Jenkins CI/CD สำหรับ BOP (Design เต็ม ยังไม่ Implement)

> **สถานะ**: เขียน `Jenkinsfile` ทั้ง 3 repo + `jenkins/rbac.yaml` แล้ว (ดู checklist ด้านล่าง) — **แต่ prerequisite เดิมของเอกสารนี้ยังไม่ครบ**: [KUBERNETES-DEPLOYMENT-TODO.md](KUBERNETES-DEPLOYMENT-TODO.md) ยังไม่เคยทดสอบบน `kind` cluster จริงเลย (รอผู้เขียน repo ทดสอบเองอยู่) ดังนั้น pipeline นี้**รันจริงไม่ได้ทั้งหมดตอนนี้** โดยเฉพาะ stage deploy — เขียน Jenkinsfile ไว้ล่วงหน้าเป็น artifact ที่ตรวจ syntax/gate ผ่านแล้วเท่าที่ตรวจได้โดยไม่มี Jenkins จริง (ดู "ข้อจำกัดที่ตรวจแล้ว" ท้ายหัวข้อนี้) ไม่ใช่ pipeline ที่ verify end-to-end แล้ว — เอกสารนี้คือ [FULLSTACK-INFRA-ROADMAP.md](FULLSTACK-INFRA-ROADMAP.md) Phase 2.1 ส่วน CI/CD ขยายรายละเอียดเต็มรูปแบบ โดยเลือก **Jenkins** เป็นเครื่องมือ (ตามที่ระบุมา — ไม่ใช่ GitHub Actions/GitLab CI ที่มักเจอในโปรเจกต์เล็กทั่วไป Jenkins เป็นของที่องค์กรใหญ่/ธนาคาร-ประกัน-telco ไทยที่ใช้ Control-M อยู่แล้วมักมีอยู่แล้วจริง ตรงกับ positioning ของ BOP ที่เล็งกลุ่มลูกค้าเดียวกัน)

**Prerequisite**: เอกสารนี้ deploy ไปที่ target ที่ออกแบบไว้ใน [KUBERNETES-DEPLOYMENT-TODO.md](KUBERNETES-DEPLOYMENT-TODO.md) — ควร implement เอกสาร K8s ให้เสร็จ (อย่างน้อยทดสอบบน `kind` ได้แล้ว) ก่อนเริ่มเอกสารนี้ เพราะ pipeline stage สุดท้าย (deploy) ไม่มีที่ให้ deploy ไปถ้ายังไม่มี manifest/cluster — **ยังไม่ผ่านจุดนี้จริง** (ดูสถานะด้านบน)

**ข้อจำกัดที่ตรวจแล้วในสภาพแวดล้อมที่เขียนโค้ดนี้ (ไม่มี Docker daemon/`kind`/`kubectl`/Jenkins จริง)**:
- `go vet`/`gofmt -l`/`go test -race` รันผ่านจริงบน `bot-orchestration-platform` และ `remote-job-worker` แล้ว (ยืนยันว่า stage Lint+Vet/Test ของ Jenkinsfile จะผ่านถ้ารันตอนนี้) **ยกเว้น 3 ไฟล์ที่ format ไม่ตรง `gofmt` อยู่ก่อนแล้ว** (ไม่เกี่ยวกับงานนี้): `internal/api/http/handler.go`, `internal/bots/externaljob/externaljob_test.go`, `internal/job/job.go` — stage `Lint + Vet` ของ Jenkinsfile จะ **fail ทันทีในการรันจริงครั้งแรก** จนกว่าจะรัน `gofmt -w` แก้ 3 ไฟล์นี้ก่อน
- `npm run lint`/`npm run typecheck` ของ `bop-dashboard` ผ่านแล้ว — `npm run test` (Vitest) รันไม่ได้บนเครื่องที่เขียนโค้ดนี้เพราะ Node เวอร์ชันเก่ากว่า (v19) ที่ `node:util` ยังไม่มี `styleText` (Vitest ต้องการ Node 22 ตรงกับ `node:22-alpine` ที่ Jenkinsfile ใช้อยู่แล้ว) ไม่ใช่ปัญหาของ Jenkinsfile
- ยังไม่มีการตรวจ Groovy syntax ของ `Jenkinsfile` ทั้ง 3 ไฟล์เลยด้วยเครื่องมือจริง (ต้องใช้ Jenkins instance จริงหรือ Pipeline Linter — ไม่มีในสภาพแวดล้อมนี้) เขียนตามโครง declarative pipeline มาตรฐาน + ตรวจ brace/indent ด้วยมือเท่านั้น

## สารบัญ

1. [Jenkins Architecture](#1-jenkins-architecture)
2. [Multi-repo Pipeline Strategy](#2-multi-repo-pipeline-strategy)
3. [Pipeline Stages (Jenkinsfile ร่างเต็ม)](#3-pipeline-stages-jenkinsfile-ร่างเต็ม)
4. [Credentials Management](#4-credentials-management)
5. [Trigger Strategy (PR vs Merge to Main)](#5-trigger-strategy-pr-vs-merge-to-main)
6. [Security Gates ในไปป์ไลน์](#6-security-gates-ในไปป์ไลน์)
7. [Rollback](#7-rollback)
8. [Notifications](#8-notifications)
9. [Open Questions](#open-questions)
10. [TODO Checklist](#todo-checklist)

---

## 1. Jenkins Architecture

### แนะนำ: Jenkins Controller บน K8s + Kubernetes Plugin (dynamic ephemeral agent pod)

แทนที่จะมี Jenkins agent เป็น VM ที่ตั้งทิ้งไว้ตลอดเวลา (ต้องดูแล patch/security เอง กินทรัพยากรแม้ไม่มี build รันอยู่) ใช้ **[Kubernetes plugin](https://plugins.jenkins.io/kubernetes/)** ให้ Jenkins controller สั่งสร้าง **agent pod ชั่วคราว** ใน cluster เดียวกับที่ deploy BOP (namespace แยกต่างหาก เช่น `jenkins`) ทุกครั้งที่มี build เข้าคิว แล้วลบทิ้งทันทีที่ build จบ — ตรงกับ ethos "ไม่จ่ายเงิน/เปิดทรัพยากรทิ้งไว้เฉยๆ" เดียวกับที่เลือก `kind` ในเอกสาร K8s (ดู [KUBERNETES-DEPLOYMENT-TODO.md](KUBERNETES-DEPLOYMENT-TODO.md) หัวข้อ 1) และเป็น pattern ที่ enterprise จริงใช้กันแพร่หลาย (agent pod ที่ scale เป็น 0 เวลาไม่มี build)

```yaml
# ตัวอย่าง Jenkins Kubernetes Cloud config (ตั้งผ่าน UI หรือ JCasC — ดู Open Question)
apiVersion: v1
kind: ServiceAccount
metadata:
  name: jenkins-agent
  namespace: jenkins
---
# Role จำกัดสิทธิ์ agent ให้สร้าง Pod ได้แค่ใน namespace jenkins เท่านั้น (least privilege
# เดียวกับหลักการที่ใช้กับ bop-worker ใน KUBERNETES-DEPLOYMENT-TODO.md หัวข้อ 4)
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata: { name: jenkins-agent-runner, namespace: jenkins }
rules:
  - apiGroups: [""]
    resources: ["pods", "pods/exec", "pods/log"]
    verbs: ["create", "get", "list", "watch", "delete"]
```

**ทางเลือกที่ง่ายกว่า (ถ้ายังไม่อยากยุ่งกับ Kubernetes plugin ตอนเริ่ม)**: Jenkins controller เดี่ยวบน VM/container ตัวเดียว ใช้ Docker agent ธรรมดา (`docker run` เป็น step ปกติ ไม่ dynamic provisioning) — ใช้งานได้จริงเหมือนกัน แค่ไม่ scale-to-zero และต้องดูแล VM เอง — เหมาะเป็นจุดเริ่มถ้าอยากเห็นผลไวก่อนค่อย migrate ไป K8s-based agent ทีหลัง

### Deploy Jenkins เองยังไง

`helm install jenkins jenkins/jenkins -n jenkins --create-namespace` (official Helm chart) — รันอยู่ใน cluster เดียวกับที่ [KUBERNETES-DEPLOYMENT-TODO.md](KUBERNETES-DEPLOYMENT-TODO.md) ตั้งไว้ namespace `jenkins` แยกจาก `bop-system`/`bop-jobs` (คนละ concern คนละ blast radius เหมือนกัน)

---

## 2. Multi-repo Pipeline Strategy

BOP กระจายอยู่ **3 repo อิสระ** (ตามที่ตัดสินใจไว้ตั้งแต่ [PHASE1-DASHBOARD-TODO.md](projects/bop-dashboard/PHASE1-DASHBOARD-TODO.md): "แยก repo/deploy จาก backend ตั้งใจ"):

| Repo | Path | Image ที่ build |
|---|---|---|
| `bot-orchestration-platform` | `projects/bot-orchestration-platform/` | `bop-api` (จาก `cmd/bop`), `bop-worker` (จาก `cmd/bop-worker`) |
| `bop-dashboard` | `projects/bop-dashboard/` | `bop-dashboard` |
| `remote-job-worker` | `projects/remote-job-worker/` | `remote-job-worker` (ถ้า deploy จริง — ตอนนี้เป็นแค่ reference implementation) |

แต่ละ repo มี **`Jenkinsfile` ของตัวเอง checked-in ที่ root** (ไม่ใช่ config รวมศูนย์ที่ Jenkins UI) — ใช้ **Multibranch Pipeline job** ต่อ repo (Jenkins scan branch/PR อัตโนมัติจาก git provider เอง ไม่ต้องสร้าง job ทีละ branch มือ) จุดสำคัญ: **repo `bot-orchestration-platform` มี 2 image จาก Jenkinsfile เดียว** (build `bop-api`/`bop-worker` เป็น parallel stage ภายใน pipeline เดียวกัน เพราะอยู่ repo/commit เดียวกัน ต้อง version ตรงกันเสมอ)

---

## 3. Pipeline Stages (Jenkinsfile ร่างเต็ม)

ตัวอย่างสำหรับ repo `bot-orchestration-platform` (declarative pipeline, syntax ใหม่ที่แนะนำเสมอเหนือ scripted pipeline เพราะอ่านง่ายกว่าและมี built-in stage visualization):

```groovy
pipeline {
  agent {
    kubernetes {
      // podTemplate ที่มี container "go" (build/test) และ "docker" (build image ผ่าน
      // Kaniko — ดูเหตุผลที่ไม่ใช้ docker-in-docker ตรงๆ ในหัวข้อ Open Questions)
      yaml """
        apiVersion: v1
        kind: Pod
        spec:
          containers:
            - name: go
              image: golang:1.26
              command: ['sleep']
              args: ['infinity']
            - name: kaniko
              image: gcr.io/kaniko-project/executor:debug
              command: ['sleep']
              args: ['infinity']
      """
    }
  }

  environment {
    IMAGE_TAG = "${env.GIT_COMMIT.take(8)}"
    REGISTRY  = credentials('bop-registry-url') // ดูหัวข้อ 4
  }

  stages {
    stage('Lint + Vet') {
      steps {
        container('go') {
          sh 'go vet ./...'
          sh 'gofmt -l . | tee /tmp/fmt.out && [ ! -s /tmp/fmt.out ]' // fail ถ้ามีไฟล์ format ไม่ตรง
        }
      }
    }

    stage('Test') {
      steps {
        container('go') {
          // -race บังคับเสมอ — โปรเจกต์นี้เจอ data race จริงมาแล้วครั้งหนึ่งตอน implement
          // worker.Pool (ดู README.md "บทเรียนที่เจอจริงระหว่างสร้าง") ไม่มีเหตุผลจะปิด
          // flag นี้ใน CI เลย
          sh 'go test -race ./...'
        }
      }
    }

    stage('Build Images') {
      when { branch 'main' } // PR build ไม่ต้อง build image จริง (ดูหัวข้อ 5)
      parallel {
        stage('bop-api') {
          steps {
            container('kaniko') {
              sh """
                /kaniko/executor \
                  --dockerfile=cmd/bop/Dockerfile \
                  --context=. \
                  --destination=${REGISTRY}/bop-api:${IMAGE_TAG} \
                  --destination=${REGISTRY}/bop-api:latest
              """
            }
          }
        }
        stage('bop-worker') {
          steps {
            container('kaniko') {
              sh """
                /kaniko/executor \
                  --dockerfile=cmd/bop-worker/Dockerfile \
                  --context=. \
                  --destination=${REGISTRY}/bop-worker:${IMAGE_TAG} \
                  --destination=${REGISTRY}/bop-worker:latest
              """
            }
          }
        }
      }
    }

    stage('Image Scan') {
      when { branch 'main' }
      steps {
        container('trivy') {
          // exit-code 1 = build fail ถ้าเจอ CRITICAL — ดูหัวข้อ 6
          sh "trivy image --severity CRITICAL --exit-code 1 ${REGISTRY}/bop-api:${IMAGE_TAG}"
          sh "trivy image --severity CRITICAL --exit-code 1 ${REGISTRY}/bop-worker:${IMAGE_TAG}"
        }
      }
    }

    stage('Deploy to Staging') {
      when { branch 'main' }
      steps {
        container('kubectl') {
          sh """
            kubectl set image deployment/bop-api bop-api=${REGISTRY}/bop-api:${IMAGE_TAG} -n bop-system --context=staging
            kubectl set image deployment/bop-worker bop-worker=${REGISTRY}/bop-worker:${IMAGE_TAG} -n bop-system --context=staging
            kubectl rollout status deployment/bop-api -n bop-system --context=staging --timeout=120s
          """
        }
      }
    }

    stage('Approve Production') {
      when { branch 'main' }
      // input step หยุด pipeline รอ manual approval — เทียบเท่า Approval Gate ที่ออกแบบ
      // ไว้ให้ BOP เองใน PHASE3-ENTERPRISE-FEATURES-TODO.md (3.4) พอดี ต่างกันแค่ตรงนี้
      // เป็น Jenkins ทำให้เอง ไม่ต้องมี Temporal Signal — สะท้อนว่า "approval gate ก่อน
      // แตะ production" เป็น pattern มาตรฐานที่เจอซ้ำได้ทั้งใน CI/CD และใน business workflow
      steps {
        input message: 'Deploy to production?', submitter: 'platform-team'
      }
    }

    stage('Deploy to Production') {
      when { branch 'main' }
      steps {
        container('kubectl') {
          sh """
            kubectl set image deployment/bop-api bop-api=${REGISTRY}/bop-api:${IMAGE_TAG} -n bop-system --context=production
            kubectl set image deployment/bop-worker bop-worker=${REGISTRY}/bop-worker:${IMAGE_TAG} -n bop-system --context=production
            kubectl rollout status deployment/bop-api -n bop-system --context=production --timeout=120s
          """
        }
      }
    }
  }

  post {
    always { junit allowEmptyResults: true, testResults: '**/test-report.xml' }
    failure { /* ดูหัวข้อ 8 Notifications */ }
  }
}
```

**หมายเหตุเรื่อง Kaniko แทน `docker build` ตรงๆ**: Jenkins agent pod รันบน K8s เอง (หัวข้อ 1) — ให้ container ธรรมดารัน `docker build` ต้อง mount `/var/run/docker.sock` เข้าไป (docker-in-docker) ซึ่งเท่ากับให้สิทธิ์ root บน host node ทางอ้อม (security risk ชัดเจน) **[Kaniko](https://github.com/GoogleContainerTools/kaniko)** build image จาก Dockerfile ได้โดยไม่ต้องมี Docker daemon เลย (รันเป็น unprivileged container ธรรมดา) — ตรงกับหลักการ least-privilege เดียวกับที่ใช้ตัดสินใจเรื่อง RBAC ทั้งเอกสารนี้และเอกสาร K8s

`bop-dashboard`/`remote-job-worker` ใช้โครง Jenkinsfile เดียวกัน สลับ `go test`/`go vet` เป็น `npm run typecheck && npm run lint && npm run test` และ `container('go')` เป็น `container('node')`

---

## 4. Credentials Management

**ห้าม hardcode credential ใน Jenkinsfile เด็ดขาด** (ตรงข้ามกับที่ BOP เองยัง hardcode `admin`/`changeme` อยู่ตอนนี้ — ดู [KUBERNETES-DEPLOYMENT-TODO.md](KUBERNETES-DEPLOYMENT-TODO.md) หัวข้อ 3.4 ที่แก้ปัญหานี้ฝั่ง BOP เอง, เอกสารนี้ต้องไม่ทำผิดซ้ำฝั่ง pipeline) ใช้ **Jenkins Credentials Store** (เก็บเข้ารหัสในตัว Jenkins เอง หรือผูกกับ external vault ผ่าน [HashiCorp Vault plugin](https://plugins.jenkins.io/hashicorp-vault-plugin/)/[OpenBao](https://openbao.org/) ตามที่ CLAUDE.md ของ repo นี้บันทึกไว้ว่าเป็นทางเลือก MPL 2.0 ของ Vault) แล้วอ้างผ่าน `credentials('id')` เท่านั้น:

| Credential ID | ใช้ทำอะไร | ประเภท |
|---|---|---|
| `bop-registry-url` + `bop-registry-creds` | push/pull image | Username/Password หรือ Secret Text |
| `bop-staging-kubeconfig` | `kubectl` เข้า staging cluster/context | Secret File |
| `bop-production-kubeconfig` | `kubectl` เข้า production cluster/context | Secret File (แยกจาก staging เสมอ — คนละสิทธิ์) |
| `bop-slack-webhook` | แจ้งเตือน (หัวข้อ 8) | Secret Text |

**ถ้า Jenkins agent รันใน cluster เดียวกับที่ deploy อยู่แล้ว** (ตามหัวข้อ 1) อาจไม่ต้องมี kubeconfig แยกเลยด้วยซ้ำ — ใช้ ServiceAccount ของ agent pod เอง (เหมือนที่ `cmd/bop-worker` ใช้ `rest.InClusterConfig()` แทน kubeconfig ไฟล์) ผูก RBAC ให้ deploy ได้เฉพาะ namespace/Deployment ที่กำหนดเท่านั้น — ปลอดภัยกว่าเก็บ kubeconfig เป็นไฟล์ลับไว้

---

## 5. Trigger Strategy (PR vs Merge to Main)

| Event | Stage ที่รัน | เหตุผล |
|---|---|---|
| PR เปิด/update | Lint + Vet + Test เท่านั้น | feedback เร็ว, ไม่มีความเสี่ยงต่อ production เลย (ไม่ build/push/deploy) |
| Merge เข้า `main` | ทุก stage รวม deploy staging อัตโนมัติ | staging ต้อง reflect `main` เสมอให้ทดสอบได้ทันที |
| Deploy production | ต้องผ่าน `input` step (manual approval) เสมอ | ไม่มี auto-deploy ตรงเข้า production โดยไม่มีคนยืนยัน — เทียบเท่าหลักการเดียวกับ Approval Gate ของ BOP เอง |

---

## 6. Security Gates ในไปป์ไลน์

1. **Secret scanning** ([gitleaks](https://github.com/gitleaks/gitleaks)) — stage แรกสุดก่อน checkout เสร็จด้วยซ้ำ กัน credential หลุดเข้า commit history โดยไม่ตั้งใจ (fail build ทันทีถ้าเจอ pattern ที่ดูเหมือน secret)
2. **Image scanning** ([Trivy](https://github.com/aquasecurity/trivy)) — ดูตัวอย่างใน Jenkinsfile หัวข้อ 3, fail build ถ้าเจอ CVE ระดับ CRITICAL ใน base image/dependency
3. **`go vet`/`gofmt -l`** และ **`npm run lint`** เป็น gate ปกติ (ไม่ใช่แค่ warning) — build fail ถ้าไม่ผ่าน ไม่ปล่อยผ่านแบบ "เดี๋ยวแก้ทีหลัง"

---

## 7. Rollback

ทางเลือกสำหรับตอน deploy แล้วพัง:

1. **`kubectl rollout undo deployment/bop-api -n bop-system`** — ง่ายสุด ย้อนไปเวอร์ชันก่อนหน้าทันที (K8s เก็บ revision history ให้เองอยู่แล้วผ่าน `ReplicaSet` เก่า)
2. **Jenkins job แบบ parameterized** — สร้าง job แยก "Rollback" ที่รับ parameter เป็น image tag เก่าที่ต้องการ แล้วรัน `kubectl set image` ย้อนกลับตรงๆ (ใช้เมื่อ `rollout undo` ไม่พอ เช่นอยากย้อนไปไกลกว่า revision ล่าสุด)

ไม่ต้องมี logic rollback ซับซ้อนกว่านี้ในระดับ portfolio — K8s `Deployment` built-in revision history ตอบโจทย์พื้นฐานได้ครบอยู่แล้ว

---

## 8. Notifications

[Slack Notification plugin](https://plugins.jenkins.io/slack/) ยิงข้อความเข้า channel ทีมตอน pipeline fail (`post { failure { ... } }` ใน Jenkinsfile หัวข้อ 3) และตอน deploy production สำเร็จ (audit trail แบบ manual — ใครกด approve เมื่อไหร่ deploy อะไรไป เห็นใน Slack history) — webhook URL เก็บเป็น Jenkins Credential (หัวข้อ 4) ไม่ hardcode ใน Jenkinsfile

---

## Open Questions

1. ~~**Kubernetes plugin (dynamic agent) หรือ Docker agent ธรรมดาบน VM เดียว**~~ — **ตัดสินใจแล้ว: Kubernetes plugin** (`Jenkinsfile` ทั้ง 3 repo เขียนด้วย `agent { kubernetes { ... } }` แล้ว) เพราะเอกสารแนะนำเป็นทางเลือกหลักอยู่แล้วและไม่มีเหตุผลให้เริ่มจาก VM เดี่ยวก่อนในเมื่อ K8s cluster (`kind`) มีอยู่แล้วจาก KUBERNETES-DEPLOYMENT-TODO.md
2. **Registry**: self-hosted (Harbor บน K8s เดียวกัน, เพิ่มความซับซ้อนแต่ฟรี) vs cloud registry (Docker Hub private repo/GHCR/cloud provider registry — จ่ายเงินหรือ rate limit ตาม tier)
3. **Staging/production แยก cluster จริง หรือแยกแค่ namespace ใน cluster เดียว** — ถ้าเป็น portfolio demo อย่างเดียว แยก namespace พอ (ประหยัดกว่ามาก) แต่ "enterprise story" จริงมักแยก cluster
4. **Jenkins Configuration as Code (JCasC)** — เก็บ config ของ Jenkins เองเป็น YAML ใน git แทนตั้งผ่าน UI (ทำให้ Jenkins instance เอง reproducible/version control ได้ — คุ้มพูดตอนสัมภาษณ์เรื่อง "infra as code" เพิ่มอีกชั้น)
5. **GitOps (ArgoCD/Flux) แทนที่ Jenkins ยิง `kubectl` ตรงๆ ในขั้น deploy** — สถาปัตยกรรมทางเลือกที่ Jenkins แค่ build/push image แล้ว "commit" ไป manifest repo แยก ให้ ArgoCD เป็นคน sync เข้า cluster เอง (decouple CI ออกจาก CD จริงจัง, audit trail ผ่าน git log) ซับซ้อนกว่าที่ร่างไว้ในเอกสารนี้ แต่เป็น pattern ที่ enterprise จริงจำนวนมากใช้ — คุ้มพูดถึงเป็นทางเลือกที่รู้จักแม้จะยังไม่ implement ตอนนี้

## TODO Checklist

### 1. Jenkins Infrastructure
- [x] เขียน `jenkins/rbac.yaml` (`ServiceAccount`/`Role`/`RoleBinding` ของ agent — หัวข้อ 1) — YAML syntax ตรวจผ่านแล้ว ยังไม่เคย apply จริง
- [ ] Deploy Jenkins ผ่าน Helm chart เข้า namespace `jenkins` (ต่อจาก KUBERNETES-DEPLOYMENT-TODO.md เสร็จก่อน) — **ยังไม่ทำ** ไม่มี cluster ให้ deploy จริงในสภาพแวดล้อมนี้ ยังไม่ได้เขียน `values.yaml` ด้วยเพราะไม่มีทางตรวจ schema ของ Helm chart จริงได้โดยไม่ต่อ internet/cluster — เขียนมั่วแล้วดูน่าเชื่อถือเสี่ยงกว่าปล่อยว่างไว้ให้ทำตอน deploy จริง
- [ ] ตั้งค่า Kubernetes plugin ให้ใช้ `jenkins-agent` ServiceAccount (ผ่าน UI หรือ JCasC — ยังไม่ตัดสินใจ ดู Open Question ข้อ 4)
- [ ] ติดตั้ง plugin: Slack Notification, HashiCorp Vault (หรือ OpenBao)/Credentials Binding

### 2. Pipeline
- [x] เขียน `Jenkinsfile` ที่ root ของทั้ง 3 repo (หัวข้อ 3) — `bot-orchestration-platform`/`bop-dashboard` เต็ม pipeline, `remote-job-worker` แค่ CI (ไม่มี deploy target จริงตอนนี้)
- [ ] ตั้ง Multibranch Pipeline job ต่อ repo ใน Jenkins
- [ ] ทดสอบ PR-triggered build (lint+test เท่านั้น) ก่อน
- [ ] ทดสอบ merge-to-main build เต็ม pipeline รวม deploy staging — **ติด blocker**: ยังไม่มี staging namespace/context แยกจาก `bop-system` เลยใน `k8s/` (Open Question ข้อ 3 ยังไม่ตัดสินใจ) ต้องแก้ก่อน stage นี้จะรันผ่านได้จริง

### 3. Security
- [x] เพิ่ม gitleaks/Trivy stage (หัวข้อ 6) — gitleaks เป็น stage แรกสุดของทั้ง 3 Jenkinsfile, Trivy อยู่หลัง Build Images (bot-orchestration-platform/bop-dashboard เท่านั้น เพราะ remote-job-worker ไม่มี image ให้ scan)
- [ ] ตั้ง Credentials ทั้งหมดในตาราง หัวข้อ 4 (ไม่ hardcode ที่ไหนเลย) — Jenkinsfile อ้างถึง credential ID ไว้ครบแล้ว (`bop-registry-url`, `bop-slack-webhook`) แต่ยังไม่มี Jenkins จริงให้ตั้งค่า

### 4. Production Gate
- [ ] ทดสอบ `input` step (manual approval) ทำงานถูกต้อง (หัวข้อ 5)
- [ ] ทดสอบ rollback จริงอย่างน้อย 1 ครั้ง (หัวข้อ 7) ก่อนเชื่อว่า pipeline "production-ready"

### 5. ก่อนรันจริงครั้งแรก (พบระหว่าง implement เอกสารนี้ ไม่ได้อยู่ใน checklist เดิม)
- [ ] รัน `gofmt -w` แก้ 3 ไฟล์ใน `bot-orchestration-platform` ที่ format ไม่ตรงอยู่ก่อนแล้ว (ดูรายชื่อในหัวข้อสถานะด้านบน) ไม่งั้น stage `Lint + Vet` fail ทันที
- [ ] ตัดสินใจ Open Question ข้อ 3 (staging แยก namespace หรือแยก cluster) แล้วสร้าง manifest/`kubectl config` context ที่ Jenkinsfile อ้างถึง (`--context=staging`/`--context=production`) ให้มีอยู่จริง

## Reference

- [KUBERNETES-DEPLOYMENT-TODO.md](KUBERNETES-DEPLOYMENT-TODO.md) — deploy target ของ pipeline นี้
- [FULLSTACK-INFRA-ROADMAP.md](FULLSTACK-INFRA-ROADMAP.md) Phase 2.1
- [projects/bot-orchestration-platform/PHASE3-ENTERPRISE-FEATURES-TODO.md](projects/bot-orchestration-platform/PHASE3-ENTERPRISE-FEATURES-TODO.md) 3.4 (Approval Gate) — แนวคิดเดียวกับ manual approval step ของเอกสารนี้ แค่คนละ layer (business workflow vs deploy pipeline)
- [books/16-cicd/](books/16-cicd/README.md) — เนื้อหาทฤษฎี CI/CD เต็ม
