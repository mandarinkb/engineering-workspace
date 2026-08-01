# บทที่ 1: GitLab CE และ Jenkins

## 1.1 CI/CD คืออะไร ทำไมต้องมี

**Continuous Integration (CI)** คือแนวปฏิบัติที่ทีมรวมโค้ดของทุกคนเข้าด้วยกันบ่อยๆ (หลายครั้งต่อวัน) พร้อม build/test อัตโนมัติทุกครั้งที่มีการเปลี่ยนแปลง เพื่อจับปัญหาให้เร็วที่สุด — **Continuous Delivery/Deployment (CD)** ต่อยอดจากนั้นด้วยการ automate การส่งโค้ดที่ผ่านการทดสอบแล้วไปสู่ environment ถัดไปจนถึง production

หัวใจของทั้งคู่คือ **pipeline** — ลำดับขั้นตอนอัตโนมัติที่รันทุกครั้งที่มีการเปลี่ยนแปลงโค้ด บทนี้เริ่มจากเครื่องมือ CI/CD สองตัวที่พบบ่อยที่สุดในองค์กร: **GitLab CI** (ในตัว GitLab CE) และ **Jenkins**

## 1.2 GitLab CI — `.gitlab-ci.yml`

GitLab CI ผูกอยู่กับ GitLab repository โดยตรง — แค่มีไฟล์ `.gitlab-ci.yml` อยู่ที่ root ของ repo ทุก push จะ trigger pipeline ให้เองโดยอัตโนมัติ ไม่ต้องตั้งค่า webhook หรือ server แยกเพิ่ม

### โครงสร้างพื้นฐาน: Stage, Job, Runner

- **Stage** — กลุ่มของงานตามลำดับขั้น (เช่น `lint`, `test`, `build`, `push`) — ทุก job ใน stage เดียวกันรันพร้อมกัน (parallel) แต่ stage ถัดไปจะไม่เริ่มจนกว่าทุก job ใน stage ก่อนหน้าจะผ่านหมด
- **Job** — หน่วยงานเดี่ยว มี script ที่จะรันจริง อยู่ภายใต้ stage ใดstage หนึ่ง
- **Runner** — เครื่อง (หรือ container) ที่รัน job จริงๆ — GitLab.com มี shared runner ให้ใช้ฟรีตามโควตา หรือทีมติดตั้ง **self-hosted runner** เองเพื่อควบคุม environment/ทรัพยากรเอง (จำเป็นถ้าต้องการ runner ที่เข้าถึง internal network หรือมี hardware เฉพาะทาง เช่น GPU)

### ตัวอย่าง Pipeline สำหรับ Go Service

```yaml
stages:
  - lint
  - test
  - build
  - push

variables:
  IMAGE_TAG: "$CI_REGISTRY_IMAGE:$CI_COMMIT_SHORT_SHA"

lint:
  stage: lint
  image: golangci/golangci-lint:latest
  script:
    - golangci-lint run ./...

test:
  stage: test
  image: golang:1.22
  script:
    - go test -race -coverprofile=coverage.out ./...
  coverage: '/coverage: \d+\.\d+% of statements/'
  artifacts:
    paths:
      - coverage.out
    expire_in: 7 days

build:
  stage: build
  image: golang:1.22
  script:
    - go build -o bin/app ./cmd/app
  artifacts:
    paths:
      - bin/app
    expire_in: 1 day
  cache:
    key: go-mod-cache
    paths:
      - /go/pkg/mod

push:
  stage: push
  image: docker:24
  services:
    - docker:24-dind
  script:
    - docker build -t $IMAGE_TAG .
    - docker login -u $CI_REGISTRY_USER -p $CI_REGISTRY_PASSWORD $CI_REGISTRY
    - docker push $IMAGE_TAG
  only:
    - main
```

pipeline นี้ไล่ลำดับ **lint → test → build → push** — ถ้า `lint` fail จะไม่รัน `test` ต่อ (ประหยัดเวลาและทรัพยากร ไม่ต้องรอ test ที่ใช้เวลานานกว่าถ้ารู้อยู่แล้วว่าโค้ดมีปัญหาเบื้องต้น)

### Variable, Artifact, Cache

- **Variable** — ค่าที่ใช้ข้าม job ได้ ทั้งที่ GitLab สร้างให้อัตโนมัติ (`CI_COMMIT_SHORT_SHA`, `CI_REGISTRY_IMAGE`) และที่ทีมกำหนดเอง (ตั้งใน `.gitlab-ci.yml` หรือใน GitLab UI สำหรับค่าที่เป็นความลับ เช่น `CI_REGISTRY_PASSWORD` — ไม่ควร hardcode secret ลงไฟล์ตรงๆ เชื่อมกับหลักการจัดการ secret ใน [Volume 15 — Security](../15-security/README.md))
- **Artifact** — ไฟล์ที่ job หนึ่งสร้างขึ้นแล้วส่งต่อให้ job ถัดไปใน pipeline เดียวกันใช้ได้ (ในตัวอย่างข้างต้น `build` สร้าง binary เก็บเป็น artifact ที่ `push` เอาไปใส่ image ได้ โดยไม่ต้อง build ซ้ำ) artifact มี `expire_in` กำหนดวันหมดอายุเพื่อไม่ให้พื้นที่เก็บบวมขึ้นเรื่อยๆ
- **Cache** — ต่างจาก artifact ตรงที่ cache ใช้เพื่อ **เร่งความเร็ว build ครั้งถัดไป** ไม่ใช่ส่งต่อข้อมูลระหว่าง job ในรันเดียวกัน (เช่น cache โฟลเดอร์ `go/pkg/mod` ที่เก็บ dependency ที่ดาวน์โหลดแล้ว ทำให้รันครั้งถัดไปไม่ต้องดาวน์โหลดใหม่ทั้งหมด)

## 1.3 Jenkins — Jenkinsfile

**Jenkins** เป็นเครื่องมือ CI/CD ที่เก่าแก่และเป็น open-source อิสระจาก platform ใดๆ (ไม่ผูกกับ Git hosting provider ตัวใดตัวหนึ่งเหมือน GitLab CI) กำหนด pipeline ผ่านไฟล์ **Jenkinsfile** ที่เก็บไว้ใน repo เดียวกับโค้ด (pipeline-as-code เหมือนกัน)

### Declarative vs Scripted Pipeline

Jenkins รองรับการเขียน pipeline สองสไตล์:

```groovy
// Declarative — โครงสร้างตายตัว อ่านง่าย เป็นมาตรฐานที่แนะนำในปัจจุบัน
pipeline {
    agent any
    stages {
        stage('Lint') {
            steps { sh 'golangci-lint run ./...' }
        }
        stage('Test') {
            steps { sh 'go test -race ./...' }
        }
        stage('Build') {
            steps { sh 'go build -o bin/app ./cmd/app' }
        }
    }
}
```

```groovy
// Scripted — เขียนเป็น Groovy script เต็มรูปแบบ ยืดหยุ่นกว่ามาก แต่ดูแลยากกว่า
node {
    stage('Lint') {
        sh 'golangci-lint run ./...'
    }
    stage('Test') {
        sh 'go test -race ./...'
    }
    stage('Build') {
        if (env.BRANCH_NAME == 'main') {
            sh 'go build -o bin/app ./cmd/app'
        }
    }
}
```

**Declarative** บังคับโครงสร้าง `pipeline { stages { stage { steps { ... } } } }` ที่ตายตัว ทำให้อ่านง่าย ตรวจสอบ syntax ได้ง่ายกว่า เหมาะกับ pipeline ส่วนใหญ่ที่ไม่ซับซ้อน **Scripted** เป็น Groovy เต็มรูปแบบ (Jenkins เขียนด้วย Groovy) ให้ควบคุม logic ซับซ้อนได้อย่างอิสระ (loop, condition ที่ซับซ้อน, try-catch) แต่ดูแลรักษายากกว่าและอ่านยากกว่าสำหรับคนที่ไม่คุ้น Groovy — ทีมส่วนใหญ่ในปัจจุบันเริ่มจาก declarative เป็น default แล้วสอด scripted block เข้าไปเฉพาะจุดที่จำเป็นจริงๆ ผ่าน `script { }` block

### Plugin Ecosystem

จุดแข็งที่สุดของ Jenkins คือ **plugin ecosystem ที่ใหญ่และเก่าแก่ที่สุดในบรรดา CI/CD tool** — มี plugin สำหรับแทบทุกเครื่องมือที่มีในอุตสาหกรรม (integration กับ Slack, JIRA, SonarQube, ทุก cloud provider, ทุก registry) ข้อดีคือถ้ามีเครื่องมือแปลกๆ ที่ต้อง integrate มักมี plugin รองรับอยู่แล้ว ข้อเสียคือ plugin จำนวนมากทำให้การดูแล (patch, compatibility ระหว่าง plugin, security) เป็นภาระต่อเนื่อง

### Agent / Node

Jenkins กระจายงานไปรันบนเครื่องต่างๆ ผ่านแนวคิด **agent (node)** — Jenkins master รับคำสั่งและตัดสินใจว่างานไหนรันที่ไหน ส่วน agent คือเครื่องจริงที่ execute job (คล้าย runner ของ GitLab CI) ทีมกำหนด agent ที่มี label เฉพาะได้ (เช่น agent ที่มี Docker, agent ที่มี GPU) แล้วสั่งให้ stage หนึ่งๆ รันบน agent ที่ตรง label เท่านั้น ทำให้กระจาย build load ข้ามหลายเครื่องได้ (distributed build)

## 1.4 เลือก GitLab CI หรือ Jenkins

| | GitLab CI | Jenkins |
|---|---|---|
| Setup | ผูกกับ GitLab อยู่แล้ว ใช้งานได้ทันทีถ้าใช้ GitLab hosting repo | ต้องติดตั้ง/ดูแล server แยกต่างหาก |
| Maintenance burden | ต่ำกว่า (เป็นส่วนหนึ่งของ GitLab) | สูงกว่า (patch, plugin compatibility, scaling agent เอง) |
| Plugin/Integration | มีในตัวพอสมควรแต่ไม่กว้างเท่า Jenkins | ระบบ plugin ใหญ่ที่สุด ครอบคลุมเกือบทุก use case |
| เหมาะกับ | ทีมที่ใช้ GitLab เป็น source control อยู่แล้ว ต้องการ setup เร็ว ดูแลน้อย | องค์กรที่ลงทุน infrastructure ของ Jenkins ไว้แล้ว หรือมี pipeline ที่ซับซ้อนมากต้องพึ่ง plugin เฉพาะทาง |

ในทางปฏิบัติ ทีมที่เริ่มต้นใหม่ (greenfield) และใช้ GitLab อยู่แล้วมักเลือก **GitLab CI** เพราะ integration แนบสนิทและ maintenance ต่ำกว่ามาก ส่วน **Jenkins** มักถูกเลือกเมื่อองค์กร**ลงทุนไว้แล้ว** (มี pipeline จำนวนมากที่พึ่งพา plugin เฉพาะทาง, มี expertise ในทีมอยู่แล้ว) หรือมีความต้องการเฉพาะที่ plugin ecosystem ของ Jenkins ตอบโจทย์ได้ดีกว่า — ไม่มีตัวไหน "ดีกว่า" อีกตัวโดยสมบูรณ์ เป็นเรื่องของ trade-off ระหว่างความสะดวก/maintenance ต่ำ กับความยืดหยุ่น/ecosystem ที่กว้างกว่า

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| Stage / Job / Runner (GitLab CI) | Stage เรียงลำดับ, job ใน stage เดียวกันรันขนาน, runner คือเครื่องที่ execute |
| Variable / Artifact / Cache | Variable ส่งค่าข้าม job, Artifact ส่งไฟล์ต่อ job ถัดไปในรันเดียวกัน, Cache เร่งความเร็ว build ครั้งถัดไป |
| Declarative vs Scripted (Jenkins) | Declarative อ่านง่าย โครงสร้างตายตัว, Scripted ยืดหยุ่นเต็มที่ (Groovy) แต่ดูแลยากกว่า |
| Plugin ecosystem | จุดแข็งที่สุดของ Jenkins แต่เป็นภาระ maintenance ต่อเนื่อง |
| Agent/Node | กระจาย build load ข้ามหลายเครื่อง |
| เลือกใช้ตัวไหน | GitLab CI เหมาะกับทีมที่ใช้ GitLab อยู่แล้วและต้องการ maintenance ต่ำ, Jenkins เหมาะกับองค์กรที่ลงทุน ecosystem ไว้แล้ว |

## คำถามทบทวน

1. ทำไม pipeline ตัวอย่างในหัวข้อ 1.2 ถึงเรียง stage เป็น lint → test → build → push ไม่ใช่ลำดับอื่น?
2. อธิบายความแตกต่างระหว่าง artifact กับ cache ใน GitLab CI — แต่ละอันแก้ปัญหาอะไร ทำไมถึงใช้แทนกันไม่ได้?
3. Declarative pipeline กับ Scripted pipeline ใน Jenkins ต่างกันอย่างไร แต่ละแบบเหมาะกับสถานการณ์ไหน?
4. Agent/Node ใน Jenkins เทียบเคียงได้กับ concept ไหนใน GitLab CI?
5. ทีมที่กำลังเริ่มโปรเจกต์ใหม่และใช้ GitLab เป็น source control อยู่แล้ว ควรพิจารณาอะไรก่อนตัดสินใจว่าจะใช้ GitLab CI หรือ Jenkins?

---

ถัดไป: [บทที่ 2 — Harbor และ Pipeline Design](02-harbor-and-pipelines.md)
