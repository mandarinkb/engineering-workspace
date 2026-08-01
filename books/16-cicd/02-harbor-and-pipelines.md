# บทที่ 2: Harbor และ Pipeline Design

## 2.1 Harbor — Private Container Registry

[Volume 7 — Docker บทที่ 1](../07-docker/01-images-and-dockerfile.md) อธิบายไว้แล้วว่า **image** คือชุด layer ที่ประกอบกันเป็น filesystem ของ container, **tag** คือชื่ออ้างอิงที่มนุษย์อ่านง่าย (เช่น `order-service:1.4.0`) ที่ชี้ไปยัง **digest** — hash ของเนื้อหา image จริงๆ (`sha256:abc123...`) ซึ่งเปลี่ยนทันทีถ้าเนื้อหา image เปลี่ยนแม้แต่ byte เดียว (tag แก้ทับได้ แต่ digest ไม่มีทางเปลี่ยนได้โดยไม่เปลี่ยนเนื้อหา นี่คือเหตุผลที่ production ที่ต้องการความแน่นอนสูงสุดมักอ้างอิง image ด้วย digest ไม่ใช่ tag)

**Harbor** คือ **private container registry** — ที่เก็บ image ขององค์กรเอง แทนที่จะพึ่ง public registry (Docker Hub) สำหรับ image ภายในที่ไม่ต้องการเปิดเผยสู่สาธารณะ

### Project และ Namespace

Harbor จัดกลุ่ม image เป็น **project** (เทียบเท่า namespace) — แต่ละทีม/แต่ละระบบมี project ของตัวเอง พร้อมสิทธิ์การเข้าถึงแยกกัน (RBAC ระดับ project — ทีม A push/pull ได้เฉพาะ project ของทีม A) ทำให้จัดการสิทธิ์ในองค์กรใหญ่ที่มีหลายทีมได้เป็นระเบียบ ไม่ใช่ทุกคนเข้าถึง image ทุกตัวได้เท่ากันหมด

```
harbor.company.com/
├── order-service/          (project — ทีม order)
│   ├── api:1.4.0
│   └── worker:1.4.0
├── payment-service/        (project — ทีม payment)
│   └── api:2.1.0
└── shared/                 (project — image ที่ใช้ร่วมกัน เช่น base image)
    └── golang-base:1.22
```

### Vulnerability Scanning — ทำไมต้อง Scan ที่ Registry ไม่ใช่แค่ตอน Build

Harbor สแกนหาช่องโหว่ (CVE) ใน image ที่ถูก push เข้ามาทุกตัว (มักผ่าน Trivy ที่ผนวกเข้ามาในตัว) — คำถามสำคัญคือทำไมต้องสแกน**ที่ registry** ในเมื่อ pipeline (บทถัดไป) ก็สแกนตอน build อยู่แล้ว?

เหตุผลคือ **CVE ใหม่ถูกค้นพบและประกาศทุกวัน** — image ที่ build และ scan ผ่านตอนวันที่ push อาจไม่มี CVE เลยในตอนนั้น แต่หลังจากนั้น 3 เดือน มีคนค้นพบช่องโหว่ใหม่ใน library เวอร์ชันหนึ่งที่ image นั้นใช้อยู่ (แม้ตัวโค้ดของทีมเองไม่ได้เปลี่ยนอะไรเลย) การสแกนตอน build ครั้งเดียวจึงเป็นแค่ snapshot ของความรู้ ณ วันนั้น **ไม่ครอบคลุมความเสี่ยงที่เกิดขึ้นทีหลัง** — Harbor แก้ปัญหานี้ด้วยการสแกน image ที่เก็บอยู่**ซ้ำเป็นระยะ** (ไม่ใช่แค่ตอน push ครั้งแรก) ทำให้ทีมรู้ทันทีที่ CVE ใหม่ถูกประกาศและกระทบ image ที่ **ถูก deploy อยู่แล้วใน production** โดยไม่ต้องรอให้มี commit ใหม่มา trigger pipeline ก่อนถึงจะรู้ตัว

### Image Signing — Supply-chain Integrity

**Image signing** (เช่นผ่าน Cosign/Notation) คือการเซ็นลายเซ็นดิจิทัลลงบน image หลัง build เสร็จ — ทำให้ตรวจสอบได้ก่อน deploy ว่า image นี้**มาจากกระบวนการ build ที่ถูกต้องจริง** ไม่ถูกแก้ไข/สับเปลี่ยนระหว่างทาง (เชื่อมกับ Software and Data Integrity Failures ใน [Volume 15 — Security บทที่ 1](../15-security/01-owasp-top-10.md)) cluster ที่ deploy สามารถตั้งค่าให้**ปฏิเสธ image ที่ไม่มีลายเซ็นที่เชื่อถือได้โดยอัตโนมัติ** ปิดช่องที่ผู้โจมตี (หรือแม้แต่ insider) จะ push image ปลอมเข้าไปแทนที่ image จริงในจุดใดจุดหนึ่งของ supply chain

### Replication — กระจาย Image ข้าม Region/Cluster

Harbor รองรับ **replication** — ตั้งค่าให้ image ที่ push เข้า instance หนึ่งถูก sync ไปยัง Harbor instance อื่นโดยอัตโนมัติ (ข้าม region, ข้าม cluster, หรือ sync ไป public registry สำหรับ image ที่ต้องเปิดเผยบางส่วน) มีประโยชน์สำหรับองค์กรที่มี cluster กระจายหลาย region — แต่ละ cluster pull image จาก Harbor instance ที่อยู่ใกล้ตัวเอง (latency ต่ำกว่า, ไม่ต้องพึ่ง network ข้าม region ตอน deploy ซึ่งอาจช้าหรือไม่เสถียร) โดยที่ยังคง image เดียวกันทุก region ผ่าน replication

## 2.2 Pipeline Design — ลำดับ Stage และเหตุผล

### ลำดับมาตรฐาน: Lint → Test → Build → Scan → Push → Deploy

```
┌──────┐   ┌──────┐   ┌───────┐   ┌──────┐   ┌──────┐   ┌────────┐
│ Lint │──►│ Test │──►│ Build │──►│ Scan │──►│ Push │──►│ Deploy │
└──────┘   └──────┘   └───────┘   └──────┘   └──────┘   └────────┘
 วินาที      นาที        นาที       นาที        วินาที      นาที
 ราคาถูก                                                    เสี่ยงสูงสุด
```

หลักการเรียงลำดับคือ **"fail fast บนสิ่งที่ราคาถูกก่อน"** — แต่ละ stage ยิ่งอยู่ท้ายยิ่งใช้เวลา/ทรัพยากรมากกว่า และยิ่งอยู่ท้ายยิ่งมีความเสี่ยงสูงกว่าถ้าเกิดปัญหา (deploy ผิดพลาดกระทบ production จริง):

1. **Lint** — เช็ค syntax/style ใช้เวลาแค่วินาที ถ้าโค้ดมีปัญหาพื้นฐาน (ตัวแปรไม่ได้ใช้, format ผิด) ไม่มีเหตุผลต้องเสียเวลารัน test ที่ใช้เวลานานกว่ามากก่อน
2. **Test** — รัน unit/integration test ใช้เวลานานกว่า lint แต่ยังเร็วกว่า build image เต็มรูปแบบ ถ้า test fail ไม่มีประโยชน์ที่จะไป build image ต่อเลย
3. **Build** — สร้าง binary/image จริง ใช้เวลาและทรัพยากรมากขึ้น
4. **Scan** — สแกนช่องโหว่ใน image ที่เพิ่ง build เสร็จ ต้องมี image ก่อนถึงจะสแกนได้ จึงอยู่หลัง build เสมอ ถ้าเจอ CVE severity สูง pipeline ควร fail ที่ตรงนี้ ไม่ปล่อยให้ image ที่มีช่องโหว่รู้จักอยู่แล้วไหลต่อไปถึง push/deploy
5. **Push** — ส่ง image ที่ผ่านทุกด่านแล้วเข้า Harbor
6. **Deploy** — นำ image ไปรันจริงใน cluster เป้าหมาย เป็น stage ที่**เสี่ยงที่สุด**เพราะกระทบผู้ใช้จริงโดยตรง จึงควรเป็นขั้นตอนสุดท้ายที่ผ่านการตรวจสอบทุกด่านก่อนหน้ามาแล้วเท่านั้น

หากสลับลำดับ (เช่น scan ก่อน test) ทีมจะเสียเวลารอผลสแกนซึ่งมักช้ากว่า lint ไปเปล่าๆ ทั้งที่โค้ดยังไม่ผ่าน test ด้วยซ้ำ — การเรียง "ราคาถูก/เร็วก่อน ราคาแพง/ช้าทีหลัง" ทำให้ทีมรู้ผลเร็วที่สุดเท่าที่จะเป็นไปได้เมื่อมีปัญหา ประหยัดทั้งเวลาของ engineer และทรัพยากร compute ของ CI

### Pipeline-as-Code

หลักการที่ทั้ง GitLab CI และ Jenkins ยึดถือร่วมกัน (บทที่ 1) คือ **pipeline-as-code** — นิยาม pipeline เป็นไฟล์ที่เก็บอยู่ใน repository เดียวกับโค้ด (`.gitlab-ci.yml`, `Jenkinsfile`) ไม่ใช่ configuration ที่ตั้งผ่าน UI แล้วเก็บแยกไว้ที่อื่น ข้อดี:

- **Version control** — เปลี่ยน pipeline ก็ผ่าน pull/merge request เหมือนเปลี่ยนโค้ด review ได้ ย้อนดู history ได้ว่าใครเปลี่ยนอะไรเมื่อไหร่
- **Reproducibility** — clone repo มาที่ไหนก็ได้ pipeline เดียวกันเป๊ะ ไม่ต้องมานั่งตั้งค่า CI server ด้วยมือใหม่ทุกครั้ง
- **Consistency ระหว่างโค้ดกับ pipeline** — เมื่อโค้ดเปลี่ยนโครงสร้าง (เช่น เพิ่ม module ใหม่ที่ต้อง test แยก) การแก้ pipeline ให้ตรงกันอยู่ใน commit เดียวกันได้ ไม่หลุด sync กัน

### แยก Pipeline ตาม Environment (Dev / Staging / Prod)

Pipeline เดียวที่ deploy ทุก environment เหมือนกันหมดมักไม่เหมาะกับความเสี่ยงที่ต่างกันของแต่ละ environment — แนวทางที่ดีคือแยก stage การ deploy ออกตาม environment พร้อม **gate** ที่เข้มงวดขึ้นเรื่อยๆ ตามความเสี่ยง:

```yaml
deploy-dev:
  stage: deploy
  script: kubectl apply -f k8s/dev/
  environment: dev
  only:
    - develop
  # ไม่มี manual gate — deploy ทันทีที่ merge เข้า develop

deploy-staging:
  stage: deploy
  script: kubectl apply -f k8s/staging/
  environment: staging
  only:
    - main
  # อาจมี automated smoke test เป็น gate ก่อนไป prod

deploy-prod:
  stage: deploy
  script: kubectl apply -f k8s/prod/
  environment: production
  only:
    - main
  when: manual        # ต้องมีคนกดยืนยันก่อน ไม่ deploy อัตโนมัติ
```

- **Dev** — deploy อัตโนมัติทันที ความเสี่ยงต่ำ (ไม่มีผู้ใช้จริง) เน้นความเร็วในการ iterate
- **Staging** — deploy อัตโนมัติเช่นกัน แต่มักมี automated gate เพิ่ม (integration test, smoke test) ก่อนถือว่าพร้อมสำหรับ prod
- **Production** — มักตั้ง `when: manual` บังคับให้มีคนกดยืนยันก่อน deploy จริงเสมอ (หรือใช้ approval workflow ที่ต้องมีคนอนุมัติ) เพราะผลกระทบจากความผิดพลาดสูงที่สุด — gate นี้ไม่ได้มีไว้เพื่อ "ไม่เชื่อใจ automation" แต่เพื่อให้มี**จุดตรวจสอบสุดท้ายโดยมนุษย์**ก่อนกระทบผู้ใช้จริง โดยเฉพาะกับการเปลี่ยนแปลงที่มีความเสี่ยงสูงหรือ deploy ในช่วงเวลาที่ทีมสนับสนุนน้อย (เช่น ปลายสัปดาห์)

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| Harbor Project | จัดกลุ่ม image ตามทีม/ระบบ พร้อม RBAC แยกสิทธิ์ |
| Scan ที่ Registry | จับ CVE ใหม่ที่ถูกประกาศทีหลัง แม้ image จะ build ผ่านสแกนไปแล้วตอน push |
| Image Signing | ยืนยันว่า image มาจาก build process ที่ถูกต้อง ไม่ถูกสับเปลี่ยนระหว่างทาง |
| Replication | sync image ข้าม region/cluster ลด latency ตอน pull |
| ลำดับ Stage | fail fast — เรียงจากราคาถูก/เร็วไปหาแพง/เสี่ยงสูง (lint→test→build→scan→push→deploy) |
| Pipeline-as-code | นิยาม pipeline เป็นไฟล์ใน repo, version control ได้, reproducible |
| แยก Pipeline ตาม Environment | gate เข้มงวดขึ้นตามความเสี่ยง, prod มักบังคับ manual approval |

## คำถามทบทวน

1. ทำไม Harbor ถึงต้องสแกนหาช่องโหว่ซ้ำเป็นระยะ ทั้งที่ image ผ่านการสแกนตอน build ใน​pipeline มาแล้ว?
2. Image signing ป้องกันปัญหาอะไรที่ vulnerability scanning ป้องกันไม่ได้?
3. อธิบายหลักการ "fail fast" ที่อยู่เบื้องหลังลำดับ lint → test → build → scan → push → deploy ทำไมถึงไม่เรียงสลับกัน?
4. Pipeline-as-code ให้ประโยชน์อะไรเหนือกว่าการตั้งค่า CI ผ่าน UI ล้วนๆ?
5. ทำไม deploy ไป production มักตั้งเป็น manual gate ในขณะที่ dev/staging มักปล่อยให้ deploy อัตโนมัติ?

---

ก่อนหน้า: [บทที่ 1 — GitLab CE และ Jenkins](01-gitlab-and-jenkins.md) | ถัดไป: [บทที่ 3 — GitOps](03-gitops.md)
