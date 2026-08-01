# บทที่ 3: GitOps

## 3.1 แนวคิดหลัก — Git เป็น Single Source of Truth

บทที่ผ่านมาพูดถึง pipeline ที่จบด้วย stage "deploy" — คำถามที่ **GitOps** ตั้งคำถามใหม่คือ: **ใครควรเป็นผู้ตัดสินใจว่า cluster ควรมีสถานะเป็นอย่างไร?**

แนวทางดั้งเดิม (traditional CI/CD) คือให้ **pipeline เป็นผู้สั่งการ** — เมื่อถึง stage deploy สคริปต์ใน pipeline จะรัน `kubectl apply` หรือคำสั่งเทียบเท่าเพื่อ**ผลักดัน (push)** การเปลี่ยนแปลงเข้าไปใน cluster ตรงๆ

**GitOps** เปลี่ยนมุมมองนี้ทั้งหมด: **Git repository (ที่เก็บ manifest ของ Kubernetes) คือ single source of truth ว่า cluster ควรมีสถานะเป็นอย่างไร** — ไม่ใช่แค่แหล่งเก็บโค้ดแอปพลิเคชันเหมือนที่คุ้นเคยกันมาโดยตลอด แต่ Git ยังเป็นแหล่งความจริงของ **desired state ของ infrastructure/deployment** ด้วย ถ้าอยากรู้ว่า production ควรมี pod กี่ตัว, ใช้ image เวอร์ชันไหน, มี config อะไรบ้าง — คำตอบทั้งหมดอยู่ใน Git ไม่ใช่ใน cluster โดยตรง (cluster เป็นแค่ผลลัพธ์ที่ควร "สะท้อน" สิ่งที่ Git บอกไว้)

## 3.2 Push-based vs Pull-based Deployment

### Push-based (แบบ CI ดั้งเดิม)

```
┌─────────────┐        kubectl apply / helm upgrade         ┌─────────────┐
│  CI Pipeline │ ─────────────────────────────────────────► │  Kubernetes  │
│ (มี cluster   │        CI ต้องถือ credential ที่เข้าถึง          │   Cluster    │
│  credential) │        cluster ได้โดยตรง                      └─────────────┘
└─────────────┘
```

CI pipeline ต้องมี **credential ที่เข้าถึง cluster ได้โดยตรง** (kubeconfig หรือ service account token ที่มีสิทธิ์ deploy) เก็บไว้เป็น secret ใน CI system — ปัญหาคือ **CI system กลายเป็นจุดที่มีสิทธิ์เข้าถึง production cluster สูงมาก** ถ้า CI ถูกเจาะ (เช่นผ่าน plugin ที่มีช่องโหว่ หรือ pipeline configuration ที่ถูกแก้ไขโดยไม่ได้รับอนุญาต) ผู้โจมตีจะได้สิทธิ์เข้าถึง cluster ไปด้วยทันที — attack surface กว้างขึ้นเพราะ credential นี้ต้องกระจายไปอยู่ในระบบภายนอก cluster

### Pull-based (GitOps)

```
┌──────────────┐                                    ┌─────────────────────────────┐
│  Git Repo     │                                    │  Kubernetes Cluster           │
│ (manifest repo│                                    │  ┌─────────────────────────┐ │
│  = desired    │  ◄────── ดึง (pull) มาเปรียบเทียบ ──── │  │ GitOps Agent (ArgoCD/   │ │
│  state)       │          อย่างต่อเนื่อง              │  │ Flux) reconcile loop     │ │
└──────────────┘                                    │  └───────────┬─────────────┘ │
                                                      │              ▼               │
                                                      │  Deployment / Service / ...  │
                                                      └─────────────────────────────┘
CI ไม่แตะ cluster โดยตรงเลย — แค่ push manifest ใหม่เข้า Git
```

**GitOps agent** (เช่น ArgoCD, Flux) รันอยู่**ภายใน cluster เอง** ทำหน้าที่คอย **pull** ดู Git repo เป็นระยะ (หรือรับ webhook แจ้งเตือนเมื่อมีการเปลี่ยนแปลง) เปรียบเทียบ desired state ใน Git กับ actual state ของ cluster ปัจจุบัน — ถ้าไม่ตรงกัน agent จะ **reconcile** (ปรับ cluster ให้ตรงกับ Git) โดยอัตโนมัติ กระบวนการนี้เรียกว่า **reconciliation loop** ซึ่งทำงานต่อเนื่องตลอดเวลา ไม่ใช่แค่ตอน deploy ครั้งเดียว — แม้จะมีใครไป `kubectl edit` แก้ไข resource ใน cluster ตรงๆ โดยไม่ผ่าน Git (configuration drift) reconciliation loop จะตรวจพบความต่างและปรับกลับให้ตรงกับ Git โดยอัตโนมัติในรอบถัดไป

### ทำไม Pull-based ถึงลด Attack Surface ได้จริง

ความต่างที่สำคัญที่สุด: ใน pull-based model **ไม่มี credential ของ cluster หลุดออกไปอยู่นอก cluster เลย** — GitOps agent ที่รันอยู่ข้างในเป็นผู้ถือสิทธิ์ deploy เอง (service account ภายใน cluster) และเป็นฝ่าย**ริเริ่มการเชื่อมต่อออกไปหา Git** (outbound) แทนที่จะให้ระบบภายนอก (CI) เชื่อมต่อ**เข้ามา**ยัง cluster (inbound) — CI ในโมเดลนี้มีหน้าที่แค่ build/test/push image และ push manifest ใหม่เข้า Git repo เท่านั้น **ไม่เคยต้องถือ credential ที่เข้าถึง cluster ได้เลยแม้แต่น้อย**

ถ้า CI system ถูกเจาะในโมเดล pull-based ผู้โจมตีจะได้แค่สิทธิ์แก้ Git repo (ซึ่งยังต้องผ่าน code review/merge request ตามปกติ และตรวจสอบย้อนหลังได้จาก git history) ไม่ได้สิทธิ์เข้าถึง cluster โดยตรงทันทีเหมือนโมเดล push-based — เป็นการลด **blast radius** (ขอบเขตความเสียหายเมื่อระบบใดระบบหนึ่งถูกเจาะ) อย่างมีนัยสำคัญ ซึ่งสอดคล้องกับหลักการ least privilege ที่ [Volume 15 — Security บทที่ 2](../15-security/02-vault-secrets-network-policy.md) อธิบายไว้

## 3.3 Loop เต็มรูปแบบ: จาก Commit ถึง Cluster

ประกอบทุกอย่างที่เรียนมาในเล่มนี้เข้าด้วยกัน ภาพรวมของ workflow ทั้งสายตั้งแต่ developer commit โค้ดจนถึง cluster รัน image ใหม่จริงมีดังนี้:

```
1. Developer merge feature branch เข้า main (application repo)
        │
        ▼
2. CI Pipeline (บทที่ 1-2): lint → test → build → scan → push
        │  push image ใหม่เข้า Harbor พร้อม tag ใหม่ (เช่น order-service:a1b2c3d)
        ▼
3. CI แก้ manifest repo (แยกจาก application repo) — อัปเดต image tag
   ใน Deployment manifest (เชื่อมกับ [Volume 8 — Kubernetes](../08-kubernetes/README.md)
   ที่สอนโครงสร้าง Deployment object) แล้ว commit + push เข้า manifest repo
   *** ขั้นตอนนี้คือขั้นตอนสุดท้ายที่ CI แตะต้อง — ไม่มีการ kubectl apply ใดๆ จาก CI เลย ***
        │
        ▼
4. GitOps Agent (ArgoCD/Flux) ที่รันอยู่ใน cluster ตรวจพบว่า manifest repo
   เปลี่ยนแปลง (image tag ใหม่ไม่ตรงกับที่ deploy อยู่ปัจจุบัน)
        │
        ▼
5. GitOps Agent reconcile — สั่ง rolling update ให้ Deployment ใน cluster
   ใช้ image tag ใหม่ ตรวจสุขภาพ pod ใหม่ (readiness probe) ก่อนสลับ traffic
        │
        ▼
6. Cluster สถานะปัจจุบัน = สถานะที่ประกาศไว้ใน Git (desired state) ตรงกันแล้ว
```

จุดที่ควรสังเกต: **repo ของโค้ดแอปพลิเคชัน (application repo)** กับ **repo ของ manifest ที่ประกาศ desired state (manifest repo)** มักแยกกันเป็นคนละ repo โดยตั้งใจ — การแยกนี้ทำให้ manifest repo เป็น audit trail ที่ชัดเจนของ**สิ่งที่ถูก deploy จริงในแต่ละ environment** โดยไม่ปนกับ commit history ของการพัฒนาโค้ดแอป (ซึ่งมีจำนวนมากกว่าและไม่เกี่ยวกับ deployment โดยตรงทุก commit) และทำให้ policy การอนุมัติ (เช่น ใครมีสิทธิ์ merge เข้า manifest repo ของ production) แยกจาก policy การอนุมัติโค้ดแอปได้ ซึ่งมักเข้มงวดกว่า

## 3.4 ข้อดีอื่นที่ได้มาโดยธรรมชาติจาก GitOps

- **Rollback ง่ายและปลอดภัย** — ถ้า deploy ใหม่มีปัญหา แค่ `git revert` commit ที่เปลี่ยน image tag ใน manifest repo แล้ว push — GitOps agent จะ reconcile คืนกลับสถานะเดิมให้อัตโนมัติ เหมือนกับกระบวนการ deploy ปกติทุกประการ (ไม่ต้องมีขั้นตอน "rollback" แยกที่ทำงานต่างจาก deploy ปกติ)
- **Audit trail ที่สมบูรณ์** — ทุกการเปลี่ยนแปลงสถานะของ cluster มีร่องรอยอยู่ใน git history ของ manifest repo ทั้งหมด (ใครเปลี่ยน อะไร เมื่อไหร่ ผ่าน merge request ที่มี review) ต่างจาก push-based ที่ประวัติการ deploy กระจายอยู่ใน CI log ซึ่งมักมีอายุเก็บจำกัดกว่า git history
- **Self-healing จาก configuration drift** — ถ้ามีใครไปแก้ resource ใน cluster ตรงๆ โดยไม่ผ่าน Git (ตั้งใจหรือไม่ตั้งใจก็ตาม) reconciliation loop จะตรวจพบและปรับกลับให้ตรงกับ Git เสมอ ทำให้ cluster ไม่ "เพี้ยน" ออกจากสิ่งที่ประกาศไว้ไปเรื่อยๆ ตามเวลา

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| Git เป็น Single Source of Truth | ไม่ใช่แค่เก็บโค้ดแอป แต่เก็บ desired state ของ deployment/infrastructure ด้วย |
| Push-based | CI ถือ credential เข้าถึง cluster เอง แล้ว `kubectl apply` ตรงเข้า cluster |
| Pull-based (GitOps) | Agent ในคลัสเตอร์ pull มาเปรียบเทียบและ reconcile เอง, CI ไม่แตะ cluster เลย |
| Attack surface | pull-based ลดพื้นที่เสี่ยงเพราะ credential ของ cluster ไม่หลุดออกไปอยู่นอก cluster |
| Reconciliation loop | ตรวจสอบและปรับ cluster ให้ตรงกับ Git ต่อเนื่องตลอดเวลา แก้ configuration drift อัตโนมัติ |
| Loop เต็มรูปแบบ | merge → CI build/push image → CI อัปเดต manifest repo → agent reconcile cluster |
| Rollback | แค่ `git revert` แล้วปล่อยให้ reconciliation loop คืนสถานะเดิมให้เอง |

## คำถามทบทวน

1. อธิบายความแตกต่างระหว่าง push-based deployment กับ pull-based deployment (GitOps) ในแง่ว่าใครเป็นผู้ริเริ่มการเชื่อมต่อ
2. ทำไม pull-based model ถึงลด attack surface ได้จริง ถ้า CI system ถูกเจาะ ผู้โจมตีในแต่ละโมเดลจะได้สิทธิ์อะไรบ้าง?
3. Reconciliation loop แก้ปัญหา configuration drift ได้อย่างไร ยกตัวอย่างสถานการณ์ที่แสดงให้เห็นความแตกต่างจากระบบที่ไม่มี reconciliation loop
4. ทำไม application repo กับ manifest repo ถึงมักแยกกันเป็นคนละ repo ในสถาปัตยกรรม GitOps?
5. อธิบาย loop เต็มรูปแบบตั้งแต่ developer merge โค้ดจนถึง cluster รัน image เวอร์ชันใหม่จริง โดยระบุว่าขั้นตอนไหนเป็นหน้าที่ของ CI และขั้นตอนไหนเป็นหน้าที่ของ GitOps agent

---

ก่อนหน้า: [บทที่ 2 — Harbor และ Pipeline Design](02-harbor-and-pipelines.md) | กลับสู่ [สารบัญ Volume 16](README.md)
