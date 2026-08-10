# แผนพัฒนา: ทำให้ Jenkins Build + Deploy อัตโนมัติตอน Push โค้ด

> **เป้าหมาย**: จำลองสิ่งที่เห็นที่บริษัท (push ขึ้น GitLab → Jenkins build/deploy ให้เอง) แต่ทำเองตั้งแต่ศูนย์บน repo นี้ (`github.com/mandarinkb/engineering-workspace`) — เพื่อเข้าใจ**ทุกจุดที่ทีมของบริษัทน่าจะ config ไว้ให้แล้วโดยที่คุณไม่เคยเห็นมาก่อน** ไม่ใช่แค่ทำให้มันรัน แต่ให้อธิบายได้ทุกขั้นตอนตอนสัมภาษณ์
>
> **ข้อแตกต่างสำคัญที่ต้องรู้ก่อนเริ่ม**: บริษัทใช้ **GitLab** (มี Jenkins integration ผ่าน GitLab Plugin) แต่ repo นี้อยู่บน **GitHub** — concept เหมือนกันทุกอย่าง (webhook, credential, pipeline-as-code) แค่ชื่อ plugin/UI ต่างกัน เอกสารนี้ใช้ฝั่ง GitHub เพราะตรงกับของจริงที่มี
>
> **สถานะตอนนี้ (จุดเริ่มต้นของแผนนี้)**:
> - [x] Jenkins รันอยู่แล้วผ่าน `make jenkins-up` (ดู [jenkins/docker-compose.yml](jenkins/docker-compose.yml)) — เข้าได้ที่ `http://localhost:8090`
> - [x] ผ่าน setup wizard แล้ว (ติดตั้ง suggested plugins + สร้าง admin user)
> - [x] มี `Jenkinsfile` พร้อมใช้แล้ว 3 ไฟล์ (`projects/bot-orchestration-platform/`, `projects/bop-dashboard/`, `projects/remote-job-worker/`)
> - [ ] **ยังไม่เคยสร้าง job ไหนเลย** — Jenkins ยังไม่รู้จัก repo นี้เลยด้วยซ้ำ
> - [ ] **ยังไม่เคย trigger อัตโนมัติ** — เป้าหมายหลักของเอกสารนี้

---

## Phase 1 — ต่อ Jenkins เข้ากับ GitHub (ทำครั้งเดียว)

**เป้าหมาย**: Jenkins คุยกับ GitHub ได้ (clone repo ได้, อ่าน branch ได้)

1. **ติดตั้ง plugin เพิ่ม** (Manage Jenkins → Plugins → Available): `GitHub Branch Source` (ชื่อ id จริง `github-branch-source` — ตัวนี้คือตัวที่ทำให้ item type "Multibranch Pipeline" รู้จัก GitHub เป็น branch source ได้) ปกติมากับ suggested plugins อยู่แล้วส่วนใหญ่ เช็คใน Installed ก่อนว่ามีหรือยัง
2. **สร้าง GitHub Personal Access Token (PAT)**: GitHub → Settings → Developer settings → Personal access tokens → Fine-grained token → ให้สิทธิ์ `Contents: Read`, `Metadata: Read`, `Commit statuses: Read and write` (อันสุดท้ายจำเป็นถ้าอยากให้ Jenkins รายงานผล build กลับไปโชว์เป็น ✅/❌ บนหน้า PR ของ GitHub — ฟีเจอร์ที่เห็นบริษัทใช้แน่ๆ)
3. **เก็บ PAT เข้า Jenkins Credentials Store**: Manage Jenkins → Credentials → System → Global credentials → Add Credentials → เลือกชนิด "Username with password" (username = GitHub username, password = PAT) ตั้ง ID ให้จำง่าย เช่น `github-pat`
4. **Verify**: Manage Jenkins → System → หาช่อง GitHub Server (ถ้ามาจาก plugin GitHub) → Add GitHub Server → ผูก credential ที่สร้างไว้ → กด "Test connection" ต้องขึ้น "Credentials verified"

**เช็คว่าผ่าน**: ปุ่ม "Test connection" ขึ้นสีเขียว ไม่ error

---

## Phase 2 — สร้าง Job แรก แล้วรันด้วยมือก่อน (ยังไม่ auto trigger)

**เป้าหมาย**: พิสูจน์ว่า `Jenkinsfile` ที่เขียนไว้รันได้จริงบน Jenkins จริง ก่อนจะไปยุ่งเรื่อง trigger อัตโนมัติ — แยกปัญหาออกจากกัน (ถ้า auto-trigger ไม่ทำงาน จะได้รู้ว่าไม่ใช่เพราะ pipeline เองพัง)

1. New Item → ตั้งชื่อ เช่น `bop-api` → เลือกชนิด **Multibranch Pipeline**
2. **Branch Sources** → Add source → GitHub → ใส่ credential (`github-pat`) → ใส่ repository URL (`https://github.com/mandarinkb/engineering-workspace`)
3. **Build Configuration** → Mode: "by Jenkinsfile" → **Script Path**: `projects/bot-orchestration-platform/Jenkinsfile` (**สำคัญมาก** — ต่างจากปกติที่ Jenkinsfile อยู่ root repo เพราะ repo นี้เป็น monorepo มี 3 project ปนกัน ดู Phase 4 เรื่องนี้เพิ่มเติม)
4. Save → Jenkins จะ scan repo หา branch ที่มี Jenkinsfile ตาม path ที่ระบุ เจอ `main` → เริ่ม build เองทันทีครั้งแรก (นี่ไม่ใช่ auto-trigger จาก push นะ — เป็นการ scan ตอนสร้าง job เท่านั้น)
5. ดูผล build — **คาดหวังว่าจะ fail บาง stage** (เช่น `Build Images`/`Deploy to Staging`) เพราะยังไม่มี Credential `bop-registry-url`/kubeconfig จริง (ดู Phase 5) แต่ **stage `Secret Scan`/`Lint + Vet`/`Test` ควรผ่าน** เพราะไม่พึ่งอะไรภายนอก

**เช็คว่าผ่าน**: เห็น stage view ของ pipeline ใน Jenkins UI ครบทุก stage (แม้บาง stage จะแดง) — ถ้า stage แรกๆ (`Secret Scan`/`Lint + Vet`/`Test`) เขียวหมด ถือว่า Phase 2 สำเร็จ

---

## Phase 3 — ทำให้ Trigger อัตโนมัติตอน Push (หัวใจของสิ่งที่อยากได้)

**ปัญหาที่ต้องแก้ก่อน**: Jenkins รันอยู่บน `localhost:8090` บนเครื่องคุณเอง — **GitHub ยิง webhook เข้ามาหา `localhost` ไม่ได้** (GitHub อยู่บน internet, เครื่องคุณไม่มี public IP) ต่างจากตอนทำงานที่บริษัทที่ Jenkins มักรันบน server ที่มี URL เข้าถึงได้จาก GitLab อยู่แล้ว — นี่คือจุดที่ local setup กับของบริษัทต่างกันจริง ต้องเลือก 1 ใน 2 ทาง:

### ทางเลือก A: SCM Polling (ง่ายกว่า ไม่ต้องเปิด network ออกสู่ internet)

Jenkins เช็ค GitHub เองเป็นรอบๆ แทนที่จะรอ webhook — ตั้งค่าใน job: Configure → Scan Multibranch Pipeline Triggers → ติ๊ก "Periodically if not otherwise run" → ตั้ง interval เช่น 1 นาที

**ข้อดี**: ไม่ต้องเปิดอะไรออก internet เลย เหมาะกับ local dev/demo
**ข้อเสีย**: ไม่ real-time จริง (delay ตาม interval) ไม่ใช่แบบที่บริษัทใช้จริง (บริษัทใช้ webhook)

### ทางเลือก B: Webhook ผ่าน Tunnel (จำลองพฤติกรรมจริงแบบที่บริษัทใช้)

ใช้เครื่องมืออย่าง [ngrok](https://ngrok.com/) หรือ Cloudflare Tunnel เปิด public HTTPS URL ชี้กลับมาที่ `localhost:8090` ชั่วคราว แล้วเอา URL นั้นไปตั้งเป็น Webhook ใน GitHub repo settings (`https://<ngrok-url>/github-webhook/`)

**ข้อดี**: real-time จริง พฤติกรรมเหมือนของบริษัทเป๊ะ — ใช้พูดในสัมภาษณ์ได้ตรงประเด็นกว่า
**ข้อเสีย**: URL เปลี่ยนทุกครั้งที่เปิด tunnel ใหม่ (ฟรี tier ของ ngrok) ต้องมาอัปเดต webhook URL ใน GitHub ใหม่ทุกรอบ ถ้าจะ demo ให้คนสัมภาษณ์ดูสดต้องเตรียมล่วงหน้า

**คำแนะนำ**: เริ่มจาก **ทางเลือก A** ก่อน (เร็ว ไม่ยุ่งยาก พิสูจน์ concept ได้เหมือนกัน) พอเข้าใจภาพรวมครบแล้วค่อยลอง **ทางเลือก B** ทีหลังเพื่อจะได้พูดเรื่อง webhook ได้จริงจังตอนสัมภาษณ์ (อธิบายได้ว่าทำไมของจริงในบริษัทไม่ต้องใช้ tunnel — เพราะ Jenkins มี public/internal URL ที่ GitLab เข้าถึงได้อยู่แล้ว)

**เช็คว่าผ่าน**: แก้โค้ดอะไรสักอย่างเล็กๆ (เช่นแก้ comment) แล้ว push ขึ้น `main` → กลับมาเช็คที่ Jenkins UI ต้องเห็น build ใหม่เกิดขึ้นเองโดยไม่ต้องกด "Build Now" มือ

---

## Phase 4 — แก้ปัญหา Monorepo (3 Project ใน Repo เดียว)

**ปัญหา**: `JENKINS-CICD-TODO.md` ออกแบบไว้ว่าแต่ละ project เป็น repo แยกกัน (`bot-orchestration-platform`, `bop-dashboard`, `remote-job-worker`) แต่ตอนนี้ทั้งหมดอยู่ใน repo เดียว (`engineering-workspace`) — ถ้าสร้าง Multibranch Pipeline 3 job ชี้ repo เดียวกัน (Script Path ต่างกัน) โดยไม่ทำอะไรเพิ่ม จะเจอปัญหา: **push แก้แค่ dashboard ก็จะไป trigger build ของ bot-orchestration-platform ด้วยทั้งที่ไม่เกี่ยวกันเลย** (เปลืองเวลา build เปล่าๆ)

ทางเลือก (เลือก 1 อย่างพอ ไม่ต้องทำทั้งคู่):

1. **ปล่อยไว้แบบนี้ก่อน** — ยอมรับว่า build เกินความจำเป็นบ้างตอนนี้ (repo เล็ก build เร็ว ไม่ใช่ปัญหาจริงจังระดับ portfolio) โฟกัสที่ทำให้ auto-trigger ทำงานให้ได้ก่อน ค่อยแก้เรื่องนี้ทีหลัง
2. **เพิ่ม path filter ในแต่ละ Jenkinsfile** — stage แรกสุดเช็คว่า commit ที่ trigger มามีไฟล์ในโฟลเดอร์ของตัวเองเปลี่ยนไหม (`git diff --name-only HEAD^ HEAD | grep '^projects/bot-orchestration-platform/'`) ถ้าไม่มีให้ jump ข้าม stage ที่เหลือทั้งหมด — เพิ่ม logic นิดหน่อยในแต่ละ Jenkinsfile ไม่ต้องพึ่ง plugin เพิ่ม
3. **แยก repo จริง** — ตรงกับที่เอกสารออกแบบไว้แต่แรก แต่ต้องทำ migration ย้าย git history จริง (ใหญ่กว่า 2 ตัวเลือกบน)

**คำแนะนำ**: เริ่ม Phase 1-3 ให้ผ่านก่อนด้วย job เดียว (`bot-orchestration-platform`) แล้วค่อยกลับมาตัดสินใจ Phase 4 ตอนจะเพิ่ม job ที่ 2/3

---

## Phase 5 — ทำให้ Stage Deploy รันผ่านจริง (ไม่ fail กลางทาง)

ต่อยอดจาก Phase 2 ที่ยอมรับว่าบาง stage fail ไว้ก่อน — Phase นี้ค่อยแก้ทีละ blocker:

1. **Registry** (Open Question ข้อ 2 ของ `JENKINS-CICD-TODO.md`) — ต้องเลือกจริงสักอัน (Docker Hub free tier ง่ายสุดสำหรับ portfolio) แล้วสร้าง Credential `bop-registry-url`/`bop-registry-creds` ใน Jenkins ให้ตรงกับที่ `Jenkinsfile` อ้างถึง
2. **K8s reachability** — Jenkins agent (container ที่รันตอนนี้) ต้องเห็น `kind` cluster ของคุณด้วย ถึงจะ `kubectl apply`/`set image` ได้จริง — เรื่องนี้มีรายละเอียด network เพิ่ม (Jenkins container ต้องมี kubeconfig ที่ชี้ถูก cluster + endpoint ที่ container เข้าถึงได้จริง ซึ่งต่างจากตอนรันจาก host โดยตรงผ่าน `make deploy-local`) — ยังไม่ได้ออกแบบละเอียดตรงนี้ รอคุยเพิ่มตอนถึง phase นี้จริง
3. **Staging namespace** — blocker เดิมที่บันทึกไว้ใน `JENKINS-CICD-TODO.md` checklist ข้อ 5 (Open Question ข้อ 3 ของเอกสารเดียวกันยังไม่ตัดสินใจ)

**เช็คว่าผ่าน**: pipeline วิ่งเขียวครบทุก stage รวม `Deploy to Staging` (ยังไม่ต้องถึง `Approve Production` ก็ได้ในรอบแรก)

---

## Phase 6 (Stretch) — ย้ายจาก Docker Agent เดี่ยวไปเป็น K8s Dynamic Agent

พอคุ้นกับการ config Jenkins พื้นฐานแล้ว (credential, job, trigger) ค่อยย้ายจาก `jenkins/docker-compose.yml` (agent เดี่ยวรันบน controller เอง) ไปเป็นสถาปัตยกรรมที่ `JENKINS-CICD-TODO.md` หัวข้อ 1 ออกแบบไว้จริง (Kubernetes plugin สร้าง agent pod ชั่วคราวต่อ build) — `jenkins/rbac.yaml` เตรียม `ServiceAccount`/`Role` ไว้รอแล้ว แค่ต้องติดตั้ง Kubernetes plugin + ตั้งค่า Kubernetes Cloud ใน Jenkins ให้ชี้ไปที่ cluster — **ไม่ต้องแก้ `Jenkinsfile` เลย** เพราะเขียนแบบ `agent { kubernetes { ... } }` รองรับไว้อยู่แล้วตั้งแต่ต้น

---

## สรุปลำดับ

```
Phase 1: ต่อ Jenkins ↔ GitHub (plugin + credential)     ──► ทำครั้งเดียว
Phase 2: สร้าง job แรก รันมือให้ผ่านก่อน                  ──► พิสูจน์ Jenkinsfile ใช้ได้จริง
Phase 3: Auto-trigger ตอน push (polling หรือ webhook)     ──► เป้าหมายหลักของเอกสารนี้
Phase 4: แก้ปัญหา monorepo (ถ้าจะเพิ่ม job ที่ 2/3)         ──► ทำตอนจะขยาย ไม่ต้องรีบ
Phase 5: Deploy stage รันผ่านจริง (registry + K8s)         ──► ปิด loop "push แล้ว deploy จริง"
Phase 6: ย้ายไป K8s dynamic agent (stretch)               ──► ทำเมื่อ Phase 1-5 นิ่งแล้ว
```

ทำทีละ phase ตามลำดับ — อย่าข้ามไป Phase 3 ก่อนที่ Phase 2 จะผ่าน (จะแยกไม่ออกว่า trigger ไม่ทำงานเพราะ webhook พัง หรือเพราะ pipeline เองพัง)

## Reference

- [JENKINS-CICD-TODO.md](JENKINS-CICD-TODO.md) — design เต็มของ pipeline stage/credential/security gate
- [KUBERNETES-DEPLOYMENT-TODO.md](KUBERNETES-DEPLOYMENT-TODO.md) — deploy target ที่ Phase 5 ต้องพึ่ง
- [jenkins/docker-compose.yml](jenkins/docker-compose.yml), [jenkins/rbac.yaml](jenkins/rbac.yaml) — infra ของ Jenkins เอง
- [Makefile](Makefile) — `make jenkins-up`/`jenkins-logs`/`jenkins-password`
