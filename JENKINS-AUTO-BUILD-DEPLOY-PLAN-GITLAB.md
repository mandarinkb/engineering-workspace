# แผนพัฒนา (เวอร์ชัน GitLab): Jenkins Build + Deploy อัตโนมัติผ่าน GitLab ที่ Self-host เอง

> **ความสัมพันธ์กับ [JENKINS-AUTO-BUILD-DEPLOY-PLAN.md](JENKINS-AUTO-BUILD-DEPLOY-PLAN.md)**: เอกสารนั้นใช้ repo จริงบน GitHub (`github.com/mandarinkb/engineering-workspace`) — เอกสารนี้เป็น**แทร็กฝึกส่วนตัวคู่ขนาน** ใช้ GitLab CE self-hosted (`make gitlab-up`) แทน เพื่อจำลองสภาพแวดล้อมแบบเดียวกับที่บริษัทใช้จริง (GitLab + Jenkins) — Phase 4/5 (monorepo, registry/K8s) เหมือนกันทุกประการกับเอกสารต้นฉบับ จึงลิงก์กลับไปแทนที่จะเขียนซ้ำ
>
> **ข่าวดีของแทร็กนี้เทียบกับ GitHub**: เพราะ GitLab กับ Jenkins รันเป็น container บนเครื่องเดียวกันและอยู่ network เดียวกันแล้ว (`bop-dev-net` — ดู [jenkins/docker-compose.yml](jenkins/docker-compose.yml) / [gitlab/docker-compose.yml](gitlab/docker-compose.yml)) **Phase 3 (auto-trigger) ไม่ต้องพึ่ง ngrok/tunnel เลย** ต่างจากแทร็ก GitHub ที่ต้องเลือกระหว่าง polling หรือ tunnel — นี่คือข้อดีที่ทำให้แทร็กนี้ปิด loop "push แล้ว trigger อัตโนมัติจริง" ได้ง่ายกว่า
>
> **สถานะตอนนี้**:
> - [x] Jenkins รันอยู่ผ่าน `make jenkins-up`
> - [x] GitLab CE รันอยู่ผ่าน `make gitlab-up` (รอบูต 2-5 นาที)
> - [x] ทั้งคู่อยู่ใน network เดียวกันแล้ว (`bop-dev-net`)
> - [ ] **ยังไม่เคย login เข้า GitLab เลย** (ต้อง `make gitlab-password` เอา root password ก่อน)
> - [ ] **ยังไม่มี project ใน GitLab เลย** — repo นี้ยังไม่เคย push เข้าไป
> - [ ] **ยังไม่เคยต่อ Jenkins ↔ GitLab เลย**

---

## Phase 1 — เข้า GitLab ครั้งแรก + สร้าง Project

1. `make gitlab-password` เอา root password (username เสมอคือ `root` — ใช้ได้แค่ 24 ชม.แรกหลังบูตครั้งแรก ถ้าหมดอายุแล้วต้องรีเซ็ตผ่าน `gitlab-rake gitlab:password:reset`)
2. เปิด `http://localhost:8929` → login ด้วย `root` + password ที่ได้
3. **New project** → "Create blank project" → ตั้งชื่อ เช่น `engineering-workspace` → เลือก Visibility ตามใจ (Private พอสำหรับฝึกส่วนตัว)
4. **สร้าง Personal Access Token** ไว้ใช้ทั้ง push โค้ดและผูก Jenkins: Avatar (มุมขวาบน) → Edit profile → Access Tokens → สร้างใหม่ ให้สิทธิ์ `api`, `read_repository`, `write_repository` — **copy เก็บไว้ทันที เห็นครั้งเดียว**

**เช็คว่าผ่าน**: เห็นหน้า project ว่างเปล่าของ GitLab พร้อมคำสั่ง `git remote add`/`git push` ตัวอย่างที่ GitLab generate ให้เอง

---

## Phase 2 — Push โค้ดเข้า GitLab ที่ Self-host เอง

ใช้ repo นี้ตัวเดิม (มี `Jenkinsfile`/manifest ทุกอย่างพร้อมอยู่แล้ว) เพิ่ม remote ใหม่แยกจาก `origin` (GitHub) — **ไม่กระทบ `origin` เดิมเลย** ทั้งสอง remote อยู่คู่กันได้:

```bash
git remote add gitlab http://localhost:8929/root/engineering-workspace.git
git push gitlab main
```

ตอน push จะถูกถาม username/password — ใส่ `root` + **Personal Access Token ที่สร้างไว้ใน Phase 1** (ไม่ใช่ password จริงของ root — GitLab ให้ใช้ token แทน password เสมอสำหรับ git operation)

**เช็คว่าผ่าน**: refresh หน้า project ใน GitLab UI เห็นไฟล์ทั้งหมดของ repo (รวม `Jenkinsfile` ทั้ง 3 ไฟล์)

---

## Phase 3 — ต่อ Jenkins เข้ากับ GitLab

1. **ติดตั้ง plugin** (Manage Jenkins → Plugins → Available): `GitLab` (id `gitlab-plugin` — ให้ Jenkins รับ webhook จาก GitLab ได้) และ `GitLab Branch Source` (id `gitlab-branch-source` — เทียบเท่า GitHub Branch Source แต่สำหรับ GitLab ใช้กับ Multibranch Pipeline)
2. **ผูก connection**: Manage Jenkins → System → หาช่อง "GitLab connections" → Add → GitLab host URL ใส่ `http://gitlab:8929` (**สำคัญ**: ใช้ชื่อ container `gitlab` ไม่ใช่ `localhost` — เพราะ Jenkins เรียกจากข้างใน container ของตัวเอง ผ่าน network `bop-dev-net` ไม่ใช่ผ่าน host) → ใส่ credential เป็น GitLab PAT ที่สร้างไว้ (ชนิด "GitLab API token" ใน Jenkins Credentials Store) → Test Connection ต้องผ่าน
3. **สร้าง Multibranch Pipeline job** เหมือน Phase 2 ของเอกสาร GitHub: New Item → Multibranch Pipeline → Branch Source เลือก GitLab → ใส่ project (`root/engineering-workspace`) → **Script Path**: `projects/bot-orchestration-platform/Jenkinsfile`

**เช็คว่าผ่าน**: "Test Connection" ที่ GitLab connections config ขึ้นเขียว + Save job แล้ว Jenkins scan เจอ branch `main` เริ่ม build เองครั้งแรก (เหมือน Phase 2 ของเอกสาร GitHub)

---

## Phase 4 — Auto-trigger ตอน Push (จุดที่ไม่ต้องง้อ ngrok)

1. ในหน้า project settings ของ GitLab: Settings → Webhooks → Add new webhook
2. URL: `http://jenkins:8080/project/<job-name>` **หรือ** path ที่ GitLab Branch Source plugin แสดงให้เองในหน้า job configuration (แต่ละเวอร์ชัน plugin โชว์ path ไม่เหมือนกันเป๊ะ — ดูจากหน้า UI จริงตรงๆ แม่นกว่าเชื่อ path ที่เขียนในเอกสารนี้) — **จุดสำคัญคือใช้ hostname `jenkins` ไม่ใช่ `localhost`** ด้วยเหตุผลเดียวกับข้อ 2 ของ Phase 3 (webhook นี้ยิงจาก container GitLab ไปหา container Jenkins โดยตรงผ่าน `bop-dev-net`)
3. เลือก Trigger events: "Push events" (และ "Merge request events" ถ้าอยากเห็นพฤติกรรมแบบ PR/MR ด้วย)
4. กด "Test" ที่ปุ่มท้ายหน้า webhook — ต้องได้ response 200 กลับมา (ถ้า fail ส่วนใหญ่มาจากใช้ `localhost` แทน `jenkins` หรือ network `bop-dev-net` ยังไม่ได้ attach ให้ครบทั้งคู่)

**เช็คว่าผ่าน**: แก้โค้ดอะไรสักอย่างเล็กๆ แล้ว `git push gitlab main` → กลับมาเช็ค Jenkins UI ต้องเห็น build ใหม่เกิดขึ้นเองทันที (เร็วกว่าทางเลือก SCM Polling ของแทร็ก GitHub มาก เพราะเป็น webhook จริง ไม่ใช่ polling)

---

## Phase 5 เป็นต้นไป

เหมือนกับเอกสารต้นฉบับทุกประการ ไม่ต้องเขียนซ้ำ — ไปต่อที่ [JENKINS-AUTO-BUILD-DEPLOY-PLAN.md](JENKINS-AUTO-BUILD-DEPLOY-PLAN.md) หัวข้อ **Phase 4 (แก้ปัญหา Monorepo)** และ **Phase 5 (Deploy Stage รันผ่านจริง)** ได้เลย บล็อกเกอร์เดียวกันทั้งหมด (registry, K8s reachability, staging namespace) ไม่เกี่ยวกับว่าใช้ GitHub หรือ GitLab เป็น git host

---

## สรุปลำดับ

```
Phase 1: เข้า GitLab ครั้งแรก + สร้าง project + PAT     ──► ทำครั้งเดียว
Phase 2: Push repo เข้า GitLab (remote คู่กับ GitHub)   ──► ไม่กระทบ origin เดิม
Phase 3: ต่อ Jenkins ↔ GitLab (plugin + connection)     ──► ใช้ hostname "gitlab" ไม่ใช่ localhost
Phase 4: Webhook auto-trigger (ไม่ต้อง ngrok)           ──► จุดเด่นของแทร็กนี้เทียบ GitHub
Phase 5+: เหมือนเอกสาร GitHub ทุกประการ                 ──► ไปต่อที่ JENKINS-AUTO-BUILD-DEPLOY-PLAN.md
```

## Reference

- [JENKINS-AUTO-BUILD-DEPLOY-PLAN.md](JENKINS-AUTO-BUILD-DEPLOY-PLAN.md) — แทร็กหลัก (GitHub) Phase 4-6 ใช้ร่วมกัน
- [JENKINS-CICD-TODO.md](JENKINS-CICD-TODO.md) — design เต็มของ pipeline stage/credential/security gate
- [gitlab/docker-compose.yml](gitlab/docker-compose.yml), [jenkins/docker-compose.yml](jenkins/docker-compose.yml) — infra ที่ทั้งสอง service ต่อกันผ่าน `bop-dev-net`
- [Makefile](Makefile) — `make gitlab-up`/`gitlab-logs`/`gitlab-password`, `make jenkins-up`/`jenkins-logs`/`jenkins-password`
