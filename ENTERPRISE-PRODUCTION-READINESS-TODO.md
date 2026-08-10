# TODO: ช่องว่างระหว่าง K8s/Jenkins ตอนนี้ กับระดับ Enterprise Production จริง

> **ขอบเขตของเอกสารนี้**: [KUBERNETES-DEPLOYMENT-TODO.md](KUBERNETES-DEPLOYMENT-TODO.md) กับ [JENKINS-CICD-TODO.md](JENKINS-CICD-TODO.md) ออกแบบไว้ระดับ **"portfolio ที่ demo ได้จริงบน `kind`"** ไม่ใช่ระดับ enterprise production — เอกสารนี้ list เฉพาะสิ่งที่ต้องเพิ่มถึงจะเรียกว่า production-grade แบบที่องค์กรใหญ่ (แบงก์/ประกัน/telco) ยอมรับได้จริง
>
> **สิ่งที่ไม่ซ้ำ**: หัวข้อ Observability, Auth/RBAC ระดับแอป, Data layer scale, Kafka, System Design, Security (SSRF/Vault) **มีอยู่แล้วใน [FULLSTACK-INFRA-ROADMAP.md](FULLSTACK-INFRA-ROADMAP.md) Phase 2.2-2.7** ไม่ list ซ้ำในนี้ — เอกสารนี้โฟกัสเฉพาะช่องว่างที่อยู่ที่ **ชั้น infra ของ K8s cluster เองและ Jenkins เอง** ที่ยังไม่มีที่ไหนพูดถึง

## สารบัญ

1. [Cluster-level HA & Disaster Recovery](#1-cluster-level-ha--disaster-recovery)
2. [Network Isolation & Multi-tenancy](#2-network-isolation--multi-tenancy)
3. [Policy Enforcement & Supply Chain Security](#3-policy-enforcement--supply-chain-security)
4. [Autoscaling & Resource Governance](#4-autoscaling--resource-governance)
5. [GitOps: แยก CD ออกจาก CI](#5-gitops-แยก-cd-ออกจาก-ci)
6. [Jenkins-specific Enterprise Gaps](#6-jenkins-specific-enterprise-gaps)
7. [Change Management & Compliance (บริบทองค์กรไทย)](#7-change-management--compliance-บริบทองค์กรไทย)
8. [สิ่งที่ไม่ต้อง list ซ้ำในนี้](#8-สิ่งที่ไม่ต้อง-list-ซ้ำในนี้)
9. [Priority Tiering](#priority-tiering)
10. [Reference](#reference)

---

## 1. Cluster-level HA & Disaster Recovery

`kind` เป็น cluster เดียว node เดียว — พอ node/cluster นี้หายไป ทุกอย่างหายหมด ไม่มี "กู้คืน" เลย ระดับ enterprise ต้องมี:

- [ ] **Control plane HA** — ถ้า self-manage K8s จริง (ไม่ใช่ managed service) ต้องมี etcd 3-5 node แยกกัน + control plane หลายตัวหลัง load balancer (ถ้าใช้ managed K8s อย่าง EKS/GKE/AKS ข้อนี้ cloud provider ทำให้แล้ว — เป็นเหตุผลหลักที่องค์กรใหญ่มักไม่ self-manage control plane เอง)
- [ ] **etcd backup** — `etcdctl snapshot save` ตามตารางเวลา + ทดสอบ restore จริงอย่างน้อย 1 ครั้ง (backup ที่ไม่เคย restore ทดสอบ ถือว่ายังไม่มี backup)
- [ ] **[Velero](https://velero.io/)** — backup ทั้ง K8s object state และ PersistentVolume ข้าม cluster/region ได้ (ต่างจาก etcd snapshot ตรงที่ Velero backup ระดับ "resource" ทำให้ restore ไป cluster ใหม่หรือ namespace ใหม่ได้ ไม่ใช่แค่กู้ cluster เดิม)
- [ ] **Multi-AZ node pool** — node กระจายหลาย availability zone อย่างน้อย เพื่อรอด AZ ล่ม (multi-region เต็มรูปแบบมักเกินความจำเป็นสำหรับ workload ขนาดนี้ ยกเว้นมี RTO/RPO ที่เข้มงวดจริง)
- [ ] **RTO/RPO ที่ระบุชัดเจน** — ก่อนจะเลือกกลยุทธ์ DR ต้องตอบให้ได้ก่อนว่า "ถ้า cluster ล่ม ยอมรับ downtime ได้กี่นาที (RTO) และยอมเสียข้อมูลย้อนหลังได้กี่นาที (RPO)" ตัวเลขนี้กำหนดว่าต้องลงทุน DR มากแค่ไหน — พูดเรื่องนี้ได้ในสัมภาษณ์คือสัญญาณของคนที่คิดแบบ senior จริง ไม่ใช่แค่ตั้งค่า default ตามคู่มือ

---

## 2. Network Isolation & Multi-tenancy

RBAC ที่ทำไว้ใน `k8s/rbac.yaml` ป้องกันแค่ระดับ **K8s API** (ใครสร้าง/ลบ resource อะไรได้) — ไม่ได้ป้องกันระดับ **network** เลย Pod คุยกันข้าม namespace ได้อย่างอิสระโดย default:

- [ ] **NetworkPolicy default-deny** — ทุก namespace (`bop-system`, `bop-jobs`) ควรเริ่มจาก deny-all แล้วเปิดเฉพาะ traffic ที่จำเป็น (DNS, Temporal, ที่ระบุชัดเจน) ตอนนี้ image ที่ผู้ใช้กำหนดเองใน `bop-jobs` ยังคุยกับ `bop-api`/Temporal ใน `bop-system` ผ่าน network ได้ตามปกติ ทั้งที่ RBAC บอกว่าควร isolate — ช่องโหว่นี้ทำให้ "blast radius" story ที่ใช้อธิบายเรื่องแยก namespace ไม่สมบูรณ์
- [ ] **Service Mesh (Istio/Linkerd)** — ถ้าต้องการ mTLS ระหว่าง service ภายใน cluster เอง (zero-trust จริงจัง ไม่ใช่แค่เชื่อ network perimeter) — เกินความจำเป็นสำหรับ workload ขนาดนี้ แต่ควรรู้จักพูดถึงเป็นทางเลือกขั้นถัดไปจาก NetworkPolicy เฉยๆ
- [ ] **PodDisruptionBudget** — `replicas: 2` ให้ความรู้สึก HA แต่ไม่มีอะไรกันไม่ให้ทั้ง 2 pod ถูก evict พร้อมกันตอน node drain/upgrade — ต้องมี `minAvailable: 1` อย่างน้อยต่อ Deployment ที่สำคัญ
- [ ] **Namespace resource isolation ให้ครบ** — ตอนนี้มี `LimitRange`/`ResourceQuota` เฉพาะ `bop-jobs` เท่านั้น (`k8s/resource-quota.yaml`) `bop-system` เองไม่มีเลย — ถ้า `bop-api`/`bop-worker` มี memory leak ก็ไม่มีเพดานกันไม่ให้กิน node ทั้งลูกเหมือนกัน

---

## 3. Policy Enforcement & Supply Chain Security

Trivy scan ที่ทำไว้ใน Jenkinsfile เป็นแค่ **CI-time gate** (เช็คตอน build) — ไม่มีอะไรกันไม่ให้คนหนึ่ง `kubectl apply` image ที่ไม่ผ่านสแกนเข้าไปตรงๆ โดยข้าม pipeline เลย:

- [ ] **Admission Controller ([OPA/Gatekeeper](https://open-policy-agent.github.io/gatekeeper/) หรือ [Kyverno](https://kyverno.io/))** — บังคับ policy ที่ระดับ cluster เอง ไม่ใช่แค่ CI: เช่น "ห้าม deploy image ที่ไม่ได้มาจาก registry ที่อนุญาต", "ทุก container ต้องมี resource limit", "ห้ามรัน container เป็น root" — แม้ Jenkinsfile จะทำถูกทุกครั้ง แต่ถ้าไม่มี admission control คนที่มีสิทธิ์ `kubectl` ตรงๆ ก็ยัง bypass ได้อยู่ดี (Kyverno เขียน policy เป็น YAML ธรรมดา เรียนรู้ง่ายกว่า OPA/Rego ถ้าต้องเลือกเริ่มจากอันไหน)
- [ ] **Image signing ([cosign](https://github.com/sigstore/cosign)/Sigstore)** — เซ็น image ตอน build ใน Jenkins แล้วให้ admission controller ตรวจลายเซ็นก่อนรัน ป้องกัน image ที่ไม่ได้ผ่าน pipeline จริง (หรือถูกแก้ไขระหว่างทางใน registry) ถูก deploy
- [ ] **SBOM (Software Bill of Materials)** — generate SBOM (เช่นด้วย `trivy image --format cyclonedx`) เก็บคู่กับทุก image ที่ build — องค์กรกำกับดูแลเข้ม (การเงิน/ประกัน) เริ่มเรียกร้องเรื่องนี้มากขึ้นเรื่อยๆ ตาม supply-chain security trend (SLSA framework)
- [ ] **Trivy scan ไม่ใช่แค่ CRITICAL** — Jenkinsfile ตอนนี้ fail เฉพาะ CRITICAL เท่านั้น ระดับ enterprise มักมี policy แยกตาม severity + deadline แก้ (เช่น HIGH ต้องแก้ภายใน 30 วัน ไม่ block build ทันทีแต่ track ผ่าน dashboard)

---

## 4. Autoscaling & Resource Governance

- [ ] **HorizontalPodAutoscaler (HPA)** — [KUBERNETES-DEPLOYMENT-TODO.md](KUBERNETES-DEPLOYMENT-TODO.md) หัวข้อ 8 พูดถึงแนวคิดไว้แต่ไม่มี manifest จริง ต้องเขียน `HorizontalPodAutoscaler` ให้ `bop-api` จริง (ต้องมี `metrics-server` ใน cluster ก่อน)
- [ ] **VerticalPodAutoscaler (VPA)** หรืออย่างน้อย right-sizing ด้วยมือเป็นระยะ — `requests`/`limits` ที่ตั้งไว้ตอนนี้ (`100m/500m` CPU, `128Mi/256Mi` memory) เป็นค่าประมาณ ไม่ได้มาจาก load test จริง ต้องวัดของจริงแล้วปรับ ไม่งั้นจะ over-provision (จ่ายแพงเกิน) หรือ under-provision (throttle/OOMKill บ่อย) โดยไม่รู้ตัว
- [ ] **Cluster Autoscaler** (หรือเทียบเท่าของ cloud provider) — ถ้า deploy จริงบน cloud managed K8s ต้องมี node autoscaling ตาม demand ไม่ใช่ node pool ขนาดคงที่ตลอดเวลา (คนละเรื่องกับ HPA — HPA scale pod, cluster autoscaler scale node)
- [ ] **Cost visibility** ([Kubecost](https://www.kubecost.com/) หรือเทียบเท่า) — รู้ว่าแต่ละ namespace/team กิน cost เท่าไหร่จริง องค์กรใหญ่ที่มีหลายทีมใช้ cluster ร่วมกันต้องมีข้อมูลนี้ก่อนจะ chargeback/optimize ได้

---

## 5. GitOps: แยก CD ออกจาก CI

[JENKINS-CICD-TODO.md](JENKINS-CICD-TODO.md) Open Question ข้อ 5 พูดถึงไว้แล้วแต่ยังไม่ implement — Jenkinsfile ตอนนี้ยิง `kubectl set image` ตรงๆ จาก pipeline (CI ยิง CD เอง):

- [ ] **แยก concern**: Jenkins ทำแค่ build/test/scan/push image แล้ว "commit" image tag ใหม่ไปยัง **manifest repo แยกต่างหาก** (เช่น `bop-k8s-manifests`) — [ArgoCD](https://argo-cd.readthedocs.io/) หรือ [Flux](https://fluxcd.io/) เป็นคน watch repo นั้นแล้ว sync เข้า cluster เอง
- [ ] **ประโยชน์ที่ได้จริง**: audit trail สมบูรณ์ผ่าน git log ล้วนๆ (ไม่ต้องพึ่ง Jenkins build log), rollback = `git revert` ธรรมดา, cluster state กับ git state ไม่มีวันเพี้ยนกัน (ArgoCD เตือนทันทีถ้ามีคน `kubectl apply` มือแล้ว state ไม่ตรงกับ git) — ตรงกับปัญหาจริงของ pipeline ปัจจุบันที่ไม่มีทางรู้เลยว่า cluster state ตรงกับ manifest ใน git จริงไหมถ้ามีคนแก้มือ
- [ ] **Progressive delivery** — พอมี ArgoCD/Flux แล้ว ต่อยอดง่ายไปเป็น canary/blue-green deployment ผ่าน [Argo Rollouts](https://argoproj.github.io/rollouts/) แทนการ `kubectl set image` แบบ rolling update ตรงๆ ที่ Jenkinsfile ทำอยู่ตอนนี้ (ไม่มีทาง gradually shift traffic หรือ auto-rollback ตาม metric เลย)

---

## 6. Jenkins-specific Enterprise Gaps

- [ ] **Jenkins Shared Library** — ตอนนี้ 3 `Jenkinsfile` (bot-orchestration-platform/bop-dashboard/remote-job-worker) ก็อปโครง stage ซ้ำกันเกือบหมด (gitleaks → lint → test → build → scan → deploy) พอมี repo เพิ่มขึ้นเรื่อยๆ (แบบที่ enterprise จริงมีเป็นร้อย repo) การก็อปวางแบบนี้ไม่ scale — ควรย้าย stage ที่ซ้ำเข้า [Shared Library](https://www.jenkins.io/doc/book/pipeline/shared-libraries/) (`vars/buildGoService.groovy` เป็นต้น) แล้วให้แต่ละ Jenkinsfile เรียกสั้นๆ แค่บรรทัดเดียว
- [ ] **Jenkins Configuration as Code (JCasC)** — [JENKINS-CICD-TODO.md](JENKINS-CICD-TODO.md) Open Question ข้อ 4 ยังไม่ตัดสินใจ — ถ้าไม่มี JCasC การตั้งค่า Jenkins เองทั้งหมด (plugin, credential ID ที่คาดหวัง, security realm) อยู่ใน UI ล้วนๆ ไม่มี version control กู้คืนไม่ได้ถ้า Jenkins instance หาย
- [ ] **Jenkins HA / backup** — Jenkins controller เดี่ยว ล่มแล้วคือ build queue/history/credential หายหมด ต้องมี PersistentVolume ที่ backup สม่ำเสมอ (JENKINS_HOME ทั้งก้อน) อย่างน้อยที่สุด — Jenkins controller HA จริง (multi-controller) ทำยากและไม่ค่อยคุ้มเมื่อเทียบกับ backup+restore ที่เร็วพอ (RTO ไม่กี่นาที) ยกเว้นมี SLA ที่เข้มงวดมาก
- [ ] **Project-based Authorization** — ตอนนี้เอกสารพูดถึงแค่ RBAC ของ agent pod (K8s level) ไม่มี RBAC ของ Jenkins เอง (ใครเห็น/แก้ job ไหนได้บ้าง) องค์กรที่มีหลายทีมใช้ Jenkins ร่วมกันต้องมี [Role-based Authorization Strategy](https://plugins.jenkins.io/role-strategy/) แยกสิทธิ์ต่อ folder/project
- [ ] **Agent/base image patching automation** — `golang:1.26`, `node:22-alpine` ที่ agent pod ใช้ต้อง track CVE และ rebuild อัตโนมัติเมื่อมี patch ใหม่ (เช่น [Renovate](https://docs.renovatebot.com/)/Dependabot เฝ้า base image tag) ไม่งั้น agent pod เองกลายเป็นจุดอ่อนที่ไม่มีใครอัปเดตซ้ำ

---

## 7. Change Management & Compliance (บริบทองค์กรไทย)

จุดที่มักถูกมองข้ามเพราะไม่ใช่ "เทคโนโลยี" แต่เป็นสิ่งที่แบงก์/ประกัน/telco ไทยให้ความสำคัญมากพอๆ กับของทางเทคนิค (และ Control-M ที่ BOP วางตัวเทียบเคียงมี concept พวกนี้ครบ):

- [ ] **Change ticket integration** — deploy production ควรผูกกับ change request (ITIL-style) เช่นให้ `input` step ใน Jenkinsfile ต้องใส่เลข ticket (ServiceNow/Jira) ก่อน approve ได้ ไม่ใช่แค่กด "yes" เฉยๆ — audit trail ที่ผูกกับ business process จริง ไม่ใช่แค่ log ทางเทคนิค
- [ ] **Segregation of duties** — คนที่เขียนโค้ด ไม่ควรเป็นคนเดียวกับที่กด approve deploy production เอง (`Approve Production` stage ตอนนี้ไม่ได้บังคับเรื่องนี้เลย — `submitter: 'platform-team'` เป็นแค่ string ไม่ได้ validate จริงว่าไม่ใช่คนเดียวกับที่ commit)
- [ ] **Audit log retention policy ที่ระบุชัดเจน** — K8s audit log, Jenkins build log, git log ต้องเก็บนานพอสำหรับ compliance requirement ของอุตสาหกรรม (การเงิน/ประกันมักกำหนดขั้นต่ำหลายปี) ไม่ใช่ default retention ของเครื่องมือ
- [ ] **Data residency** — ถ้าข้อมูลที่ BOP จัดการเข้าข่ายกฎหมาย (PDPA ของไทย) ต้องรู้ว่า cluster/registry/log storage อยู่ region ไหน ข้อมูลออกนอกประเทศได้ไหม — ธนาคาร/ประกันไทยมักมีข้อกำหนดเรื่องนี้ชัดเจน

---

## 8. สิ่งที่ไม่ต้อง list ซ้ำในนี้

หัวข้อเหล่านี้เป็น "enterprise gap" เหมือนกัน แต่ **มี owner อยู่แล้วในเอกสารอื่น** — เช็คที่นั่นแทน:

| หัวข้อ | อยู่ที่ |
|---|---|
| Metrics/Logging/Tracing (Prometheus, OpenSearch, Jaeger) | [FULLSTACK-INFRA-ROADMAP.md](FULLSTACK-INFRA-ROADMAP.md) Phase 2.2 |
| Auth/RBAC ระดับแอป, API Gateway/rate limit | Phase 2.3 |
| DB partitioning/replication, Redis cache | Phase 2.4 |
| Queue จริงจัง (Kafka), retry/circuit breaker | Phase 2.5 |
| CAP/consistency, sharding, HA/DR ระดับแอป | Phase 2.6 |
| SSRF protection, Vault/OpenBao สำหรับ secret ของแอป | Phase 2.7 |
| Sealed Secrets/External Secrets Operator สำหรับ K8s Secret | [KUBERNETES-DEPLOYMENT-TODO.md](KUBERNETES-DEPLOYMENT-TODO.md) Open Question ข้อ 5 |
| Staging/production namespace หรือ cluster แยก | [JENKINS-CICD-TODO.md](JENKINS-CICD-TODO.md) Open Question ข้อ 3 (ยังเป็น blocker ที่ยังไม่ตัดสินใจ) |

---

## Priority Tiering

ไม่ต้องทำทุกข้อพร้อมกัน — เรียงตาม impact/effort สำหรับ portfolio ที่ต้องเล่าเรื่องได้ในสัมภาษณ์ ไม่ใช่ทำให้ครบ checklist:

**Tier 1 — คุ้มทำที่สุด (impact สูง, effort พอจัดการได้คนเดียว)**
1. NetworkPolicy default-deny (หัวข้อ 2) — ปิดช่องโหว่จริงที่ทำให้ "blast radius" story ไม่สมบูรณ์
2. PodDisruptionBudget (หัวข้อ 2) — 3 บรรทัด YAML แก้ปัญหา HA ที่อ้างไว้แต่ไม่มีจริง
3. HPA จริง (หัวข้อ 4) — ต่อยอดตรงจากที่ออกแบบไว้แล้วใน KUBERNETES-DEPLOYMENT-TODO.md หัวข้อ 8
4. GitOps (ArgoCD) (หัวข้อ 5) — เป็น pattern ที่ถูกถามบ่อยที่สุดในสัมภาษณ์ระดับ senior infra และต่อยอดจากของที่มีอยู่แล้วโดยตรง

**Tier 2 — คุ้มพูดถึงว่า "รู้จัก" แม้ยังไม่ implement เต็ม**
5. Kyverno/OPA admission policy (หัวข้อ 3) — เขียน policy ตัวอย่าง 2-3 ข้อพอ ไม่ต้องครบ
6. Jenkins Shared Library (หัวข้อ 6) — สะท้อนว่าคิดเรื่อง scale ของ pipeline เอง ไม่ใช่แค่ทำให้รันได้
7. Velero backup (หัวข้อ 1) — ตั้งค่า + ทดสอบ restore 1 ครั้งพอเป็นหลักฐาน

**Tier 3 — รู้ไว้พูดได้ แต่ไม่คุ้ม implement จริงระดับ portfolio คนเดียว**
8. Control plane HA/multi-region DR, Service Mesh, image signing/SBOM เต็มรูปแบบ, Jenkins controller HA, Kubecost — งานพวกนี้ใหญ่เกินขนาด portfolio หรือต้องพึ่ง infra ที่มีค่าใช้จ่ายจริง (cloud managed service) พูดถึงเป็น "รู้ว่ามีและทำไมต้องมี" ในสัมภาษณ์ได้ โดยไม่ต้องมี live demo

---

## Reference

- [KUBERNETES-DEPLOYMENT-TODO.md](KUBERNETES-DEPLOYMENT-TODO.md) — deploy target ปัจจุบันที่เอกสารนี้ต่อยอด
- [JENKINS-CICD-TODO.md](JENKINS-CICD-TODO.md) — pipeline ปัจจุบันที่เอกสารนี้ต่อยอด
- [FULLSTACK-INFRA-ROADMAP.md](FULLSTACK-INFRA-ROADMAP.md) — หัวข้อ enterprise อื่นที่ไม่ซ้ำในนี้ (ดูหัวข้อ 8)
- [books/08-kubernetes/](books/08-kubernetes/README.md) — เนื้อหาทฤษฎีพื้นฐาน (ยังไม่มีบทเรื่อง `batch/v1 Job`/`LimitRange`/`ResourceQuota`/OPA/Kyverno/Velero/Service Mesh เต็มรูปแบบโดยตรง อาจต้องเสริมถ้าอยากอ่านลึกก่อน implement)
- [books/16-cicd/03-gitops.md](books/16-cicd/03-gitops.md) — เนื้อหา GitOps มีอยู่แล้วจริง (เช็คแล้ว ไม่ใช่ช่องว่างเหมือนที่เคยเข้าใจผิดไว้)
