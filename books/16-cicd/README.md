# Volume 16 — CI/CD

กระบวนการ automate การ build, test, และ deploy โค้ดจากเครื่อง developer ไปสู่ production อย่างปลอดภัยและเชื่อถือได้

## เป้าหมายการเรียนรู้

- ตั้งค่า pipeline บน GitLab CE / Jenkins ที่ build, test, และ deploy อัตโนมัติ
- ใช้ Harbor เป็น private container registry พร้อม image scanning
- ออกแบบ pipeline หลายขั้นตอน (stage) ที่เหมาะกับ workflow ของทีม
- เข้าใจแนวคิด GitOps และความต่างจาก push-based deployment แบบเดิม

## สารบัญ

### [บทที่ 1 — GitLab CE และ Jenkins](01-gitlab-and-jenkins.md)

- [ ] `.gitlab-ci.yml` structure: stage, job, runner
- [ ] Variable, artifact, cache ระหว่าง job
- [ ] Jenkinsfile (declarative vs scripted pipeline)
- [ ] Plugin ecosystem, agent/node

### [บทที่ 2 — Harbor และ Pipeline Design](02-harbor-and-pipelines.md)

- [ ] Private container registry, project/namespace
- [ ] Image scanning (vulnerability scan), image signing, replication ข้าม registry
- [ ] Stage ทั่วไป: lint → test → build → scan → push → deploy
- [ ] Pipeline as code, การแยก pipeline ตาม environment (dev/staging/prod)

### [บทที่ 3 — GitOps](03-gitops.md)

- [ ] แนวคิด: Git เป็น single source of truth ของ desired state
- [ ] Pull-based deployment (agent ใน cluster ดึง state จาก Git) เทียบกับ push-based (CI สั่ง deploy ตรง)
- [ ] เชื่อมกับ [Volume 8 — Kubernetes](../08-kubernetes/README.md) — Deployment manifest ถูกจัดการผ่าน Git

## Lab ที่เกี่ยวข้อง

ยังไม่มีแล็บเฉพาะในรอบนี้ — แนะนำตั้ง pipeline จริงให้กับ [projects/ecommerce/](../../projects/ecommerce/README.md)

## แหล่งอ้างอิงแนะนำ

ดู [resources/](../../resources/README.md)
