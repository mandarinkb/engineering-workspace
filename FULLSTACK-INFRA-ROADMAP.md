# Fullstack + Infra Roadmap (Golang → React/Next.js → Infra)

## โปรเจกต์หลัก: Bot Orchestration Platform (BOP)

roadmap นี้ผูกทุกหัวข้อเข้ากับการสร้างโปรเจกต์เดียวต่อเนื่อง: **แพลตฟอร์มควบคุม/สั่งงาน bot แบบ workflow (เรียกย่อว่า BOP)** — ระบบที่ให้ผู้ใช้สร้าง/config bot (scraping, price monitoring, notification, automation ทั่วไป), สั่งรัน/schedule, ดูสถานะ/log แบบ real-time ผ่าน dashboard

**ทำไมใช้โปรเจกต์นี้เป็นแกน**: เป็นสิ่งที่คุณสนใจอยู่แล้ว (โอกาสทำจนจบสูงกว่า tutorial ทั่วไป) และ domain นี้ mapping ตรงกับหัวข้อ infra ที่ต้องเรียนแบบไม่ต้องนึกโจทย์สมมติ — "รัน task หลายตัวพร้อมกัน, ต้อง retry ได้, ต้องมี step ที่ compensate กันได้, ต้อง monitor ว่า task ไหนพัง" คือโจทย์ตรงตัวของ Kafka, Temporal, Observability อยู่แล้ว

**ข้อควรระวังเชิงสถาปัตยกรรม**: ออกแบบ BOP เป็น **engine ทั่วไปที่รองรับ bot หลายประเภทแบบ pluggable** (scrape ข้อมูลสาธารณะ, monitor ราคา, แจ้งเตือน, automation ทั่วไป) อย่า hard-code ให้ผูกกับ use case เดียวแบบ "จองสินค้าเฉพาะเว็บ" — เหตุผลทางเทคนิคล้วนๆ: engine ทั่วไป demo ได้กว้างกว่า, เป็น system design ที่ลึกกว่า, และไม่ไปเสี่ยงชน ToS ของเว็บเป้าหมายจนกลายเป็นจุดอ่อนตอนอธิบายในสัมภาษณ์แทนที่จะเป็นจุดแข็ง

**Positioning ที่แนะนำ**: วาง BOP เป็น "lightweight cloud-native workload automation platform" — แนวเดียวกับ **Control-M** (BMC), **Apache Airflow**, หรือ workflow orchestration tool ระดับ enterprise แต่สร้างด้วย stack สมัยใหม่ (Temporal/K8s/Kafka) นี่คือ narrative ที่ทรงพลังสำหรับตลาดไทยโดยเฉพาะ เพราะแบงก์/ประกัน/telco ไทยใช้ Control-M กันแพร่หลาย และกำลังอยู่ในช่วง modernize/ลดต้นทุน license — คนที่โชว์ของจริงว่าเข้าใจ concept เดียวกัน (job dependency, calendar scheduling, SLA, rerun) แต่สร้างด้วยเครื่องมือ cloud-native คือจุดขายที่ตรงใจ hiring manager กลุ่มนี้มาก รายละเอียดฟีเจอร์แนว Control-M อยู่ใน [Phase 3](#phase-3--workload-automation-depth-แรงบันดาลใจจาก-control-m)

หลักการของไฟล์นี้: เอาเฉพาะหัวข้อที่ **จำเป็นต่อการใช้งานจริง** พร้อมงานที่ต้องสร้างจริงใน BOP กำกับทุกหัวข้อ — ถ้าอยากได้เนื้อหาลึกแบบ textbook ดูที่ [books/](books/README.md) และ [ROADMAP.md](ROADMAP.md) ประกอบ

---

## Phase 0 — Golang: สร้าง Bot Execution Engine

เป้าหมาย: ไม่ใช่เรียนใหม่ แต่พิสูจน์ว่าฐาน Go แน่นจริง ผ่านการสร้าง engine รันจริง (อ่านลึกที่ [books/04-golang/](books/04-golang/README.md))

### 0.1 ออกแบบ Core Interface

- [ ] นิยาม `Bot` interface (`Run(ctx context.Context) (Result, error)`, `Name() string`, `Config() BotConfig`) — ฝึก interface + implicit implementation ให้เป็นธรรมชาติ
- [ ] เขียน bot ตัวแรกที่ implement interface นี้ (เช่น price-checker bot ใช้ `colly` scrape ราคาสินค้าจากหน้าเว็บสาธารณะ)
- [ ] เขียน bot ตัวที่สองที่ต้องใช้ headless browser (`chromedp`) สำหรับเว็บที่ render ด้วย JS — พิสูจน์ว่า interface เดิมรองรับ implementation ที่ต่างกันได้จริงโดยไม่แก้ core

### 0.2 Worker Pool ด้วย Concurrency

- [ ] สร้าง worker pool ที่ดึงงานจาก queue (เริ่มจาก channel ธรรมดาก่อน) แล้วรัน bot พร้อมกันได้ N ตัว จำกัดด้วย semaphore/buffered channel — ฝึก goroutine + channel จาก [books/04-golang/02-concurrency-and-context.md](books/04-golang/02-concurrency-and-context.md) ตรงๆ
- [ ] ทดสอบด้วย `-race` ว่า worker pool ไม่มี race condition ตอนหลาย worker เขียน result พร้อมกัน
- [ ] ใช้ `context.WithTimeout` ให้แต่ละ job มี timeout ของตัวเอง และ `context.WithCancel` ให้ผู้ใช้กด "หยุด job" กลางทางได้จริง

### 0.3 Error Handling ที่แยกแยะได้

- [ ] ออกแบบ error type แยก retryable (เช่น network timeout ชั่วคราว) กับ non-retryable (เช่น เว็บเป้าหมายเปลี่ยน HTML structure จน selector หาไม่เจอ) ด้วย `errors.Is`/`errors.As`
- [ ] เก็บ error message ที่มีประโยชน์พอจะ debug ทีหลังได้ (ไม่ใช่แค่ "failed")

### 0.4 Persistence และ Testing

- [ ] Repository pattern: `JobRepository` interface + PostgreSQL implementation เก็บประวัติการรัน (start time, end time, status, result, error)
- [ ] Table-driven test สำหรับ logic การ parse ข้อมูลที่ scrape มา, mock HTTP response ด้วย `httptest.Server` แทนการยิงเว็บจริงตอน test
- [ ] CLI เล็กๆ (`go run . run --bot=price-checker`) สั่งรัน bot ได้ก่อนที่จะมี UI — พิสูจน์ core engine ทำงานได้จริงก่อนเพิ่มความซับซ้อน

**เป้าหมายเมื่อจบ phase**: มี Go engine ที่รัน bot อย่างน้อย 2 ประเภทพร้อมกันได้, มี test ผ่าน, เก็บประวัติลง DB ได้ — สั่งงานผ่าน CLI ได้จริงแม้ยังไม่มี UI

---

## Phase 1 — React + Next.js: สร้าง Dashboard ควบคุม BOP

เป้าหมาย: dashboard ที่ backend dev ทั่วไปสร้างไม่ได้ง่ายๆ (real-time, form ซับซ้อน) ไม่ใช่แค่ CRUD ธรรมดา

### 1.1 พื้นฐานก่อนแตะ React

- [ ] JavaScript ES6+ ที่ใช้บ่อย: arrow function, destructuring, spread/rest, template literal, `async/await`, module import/export
- [ ] **TypeScript ตั้งแต่ต้น** (ไม่ใช่เรียน JS ล้วนก่อน) — นิยาม type ของ `Bot`, `Job`, `JobStatus` ให้ตรงกับ struct ฝั่ง Go เพื่อความคุ้นเคย (mental model เดียวกับ struct ที่ใช้อยู่แล้ว)

### 1.2 React Core ผ่านหน้า Bot List

- [ ] Component, JSX, props — สร้าง `<BotCard />`, `<BotList />` แสดงรายการ bot ที่มี
- [ ] `useState` — state ของ filter/search ในหน้า list
- [ ] `useEffect` — โหลดรายการ bot ตอน mount, cleanup ตอน unmount
- [ ] Conditional rendering + list rendering + `key` prop — แสดง empty state ถ้ายังไม่มี bot, render list พร้อม key ที่ถูกต้อง
- [ ] `useRef`, `useContext` — เก็บ reference ของ modal, แชร์ user session ผ่าน context

### 1.3 Dynamic Form: สร้าง/แก้ไข Bot Config

- [ ] **react-hook-form + zod** — form สร้าง bot ที่ field เปลี่ยนตามประเภท bot ที่เลือก (price-checker ต้องการ URL + selector, notification bot ต้องการ webhook URL) — โจทย์ dynamic form ที่ยากกว่า tutorial ทั่วไป
- [ ] Validation ฝั่ง client ด้วย zod schema ที่ mapping กับ validation ฝั่ง Go (ความคิดคล้าย struct tag validation ที่คุ้นเคย)
- [ ] Controlled input ทั้งหมด, error message ที่ user เข้าใจได้

### 1.4 หน้า Job History

- [ ] Table แสดงประวัติการรัน (status badge, เวลาเริ่ม/จบ, ผลลัพธ์), pagination
- [ ] **TanStack Query** — โหลด/cache ข้อมูล job history, retry อัตโนมัติถ้า request fail (mental model ตรงกับ cache-aside ที่เคยอ่านใน [books/06-redis/01-caching-and-ttl.md](books/06-redis/01-caching-and-ttl.md))
- [ ] Filter ตาม bot/status/ช่วงเวลา — ฝึก sync state ระหว่าง URL query param กับ UI filter

### 1.5 หน้า Live Job Monitor (จุดที่ยากที่สุดและมีค่าที่สุดของ phase นี้)

**ฝั่ง Go API — ออกแบบ log pipeline ก่อนเขียน WebSocket handler**:

- [ ] นิยาม `Logger` interface กลาง (`Log(jobID, level, message string)`) ที่ bot ทุกตัวเรียกใช้แทนการ `fmt.Println` ตรงๆ — ทำ 2 อย่างเสมอทุกครั้งที่เรียก: (1) insert log line ลง PostgreSQL table `job_logs` ทันที (2) broadcast เข้า in-memory pub/sub ที่แยกตาม `jobID`
- [ ] **Persist ก่อนเสมอ แล้วค่อย stream** — ห้ามให้ log วิ่งเข้า WebSocket ตรงๆ โดยไม่เก็บ DB เพราะ WebSocket ปิด connection ปุ๊บข้อมูลหายทันที ต้องมีที่เก็บถาวรให้กลับมาดูย้อนหลังได้เสมอ (ต่อยอดจาก `JobRepository` ที่ทำไว้ใน 0.4 แล้ว)
- [ ] **Broadcast แบบ per-jobID** (ไม่ใช่ channel เดียวทั้งระบบ) — ป้องกันไม่ให้ client ที่ดู job A เห็น log ของ job B ปนกัน ออกแบบเป็น map ของ `jobID → []chan LogLine` หรือใช้ broadcaster pattern เล็กๆ เอง
- [ ] **REST endpoint สำหรับ backfill**: `GET /jobs/{id}/logs` ดึง log ทั้งหมดที่มีอยู่แล้วจาก DB ก่อน แล้วค่อยเปิด WebSocket ต่อรับเฉพาะ line ใหม่ที่เกิดหลังจากนั้น — แก้ปัญหา "ปิดเบราว์เซอร์แล้วเปิดใหม่ระหว่าง job กำลังรัน" ไม่ให้ log ที่พลาดไปหายไปเลย

**ฝั่ง Next.js**:

- [ ] **WebSocket หรือ Server-Sent Events (SSE)** — เชื่อมต่อกับ Go backend เพื่อดู log/สถานะ bot ที่กำลังรันแบบ real-time ไม่ใช่ polling — ทักษะนี้ portfolio ทั่วไปมักไม่มีเพราะ tutorial ส่วนใหญ่สอนแค่ CRUD
- [ ] จัดการ connection lifecycle (reconnect ถ้าหลุด, cleanup ตอนออกจากหน้า) ด้วย `useEffect`
- [ ] แสดง log แบบ stream (auto-scroll, บอกสถานะ connecting/connected/disconnected ให้ user เห็น)
- [ ] โหลด log เก่าจาก REST endpoint ก่อน render แล้วค่อย append line ใหม่ที่มาจาก WebSocket ต่อท้าย (ไม่ใช่รอ WebSocket อย่างเดียวตั้งแต่แรก)

> **หมายเหตุสำหรับตอน scale ขึ้น K8s (Phase 2.1)**: in-memory pub/sub ใช้ได้ตราบใดที่ API ยังมี instance เดียว พอ scale เป็นหลาย replica, worker อาจ report กลับมาที่ replica คนละตัวกับที่ user เปิด WebSocket ค้างอยู่ — ตอนนั้นต้องสลับ broadcaster จาก in-memory เป็น **Redis Pub/Sub** ([books/06-redis/02-pubsub-and-streams.md](books/06-redis/02-pubsub-and-streams.md)) แทน ออกแบบ broadcaster เป็น interface แยกไว้ตั้งแต่ต้นเพื่อสลับ implementation ทีหลังได้โดยไม่แก้โค้ดส่วนอื่น

### 1.6 Next.js เฉพาะทาง

- [ ] **App Router** — `layout.tsx` สำหรับ nav bar ที่ใช้ร่วมทุกหน้า, `page.tsx` แยกตามฟีเจอร์ (`/bots`, `/jobs`, `/jobs/[id]`)
- [ ] **Server Component vs Client Component** — หน้า list ที่ดึงข้อมูลแรกใช้ Server Component (เร็วกว่า), ส่วนที่ต้อง interactive (form, WebSocket) ต้องเป็น Client Component (`"use client"`) — เข้าใจว่าทำไมแยกแบบนี้
- [ ] Route Handler — เขียน endpoint เล็กๆ ใน Next.js เอง (เช่น proxy บาง request ไป Go API พร้อมแนบ auth header)
- [ ] Environment variable (`NEXT_PUBLIC_API_URL` vs server-only secret)

### 1.7 Styling

- [ ] **Tailwind CSS** — status badge สี (running=เหลือง, success=เขียว, failed=แดง), layout ของ dashboard

### 1.8 Authentication

- [ ] จัดการ JWT ที่ Go API ออกให้ เก็บใน httpOnly cookie (เชื่อมกับ [books/09-authentication/01-session-and-jwt.md](books/09-authentication/01-session-and-jwt.md))
- [ ] Protected route — ต้อง login ก่อนเห็น dashboard
- [ ] (Optional) NextAuth.js ถ้าอยาก login ผ่าน GitHub OAuth — เชื่อมกับ [books/09-authentication/02-oauth2-and-oidc.md](books/09-authentication/02-oauth2-and-oidc.md)

### 1.9 Testing + Deploy

- [ ] Vitest + React Testing Library — test form validation logic, test ว่า status badge render สีถูกต้องตาม status
- [ ] Deploy ขึ้น Vercel ก่อนเพื่อ demo เร็ว แล้วค่อย containerize ด้วย Docker ต่อใน Phase 2

**เป้าหมายเมื่อจบ phase**: dashboard ที่ login ได้, สร้าง/แก้ bot config ได้, ดู job history ได้, และดู log แบบ real-time ของ job ที่กำลังรันอยู่ได้จริง

---

## Phase 2 — Infra: ยกระดับ BOP ให้เป็น Production-Grade

เนื้อหาลึกอยู่ใน [books/07](books/07-docker/README.md) ถึง [books/17](books/17-system-design/README.md) — ที่นี่คือ**งานที่ต้องอัปเกรด BOP ให้ตรงกับแต่ละหัวข้อ** เรียงตาม priority ต่อการหางาน ไม่ใช่ตามเลขบทเดิม

### 2.1 Containerize + Deploy พื้นฐาน

- [ ] [Docker](books/07-docker/README.md) — เขียน Dockerfile แยกให้ Go API, Go worker, และ Next.js dashboard (multi-stage build ให้ image เล็ก)
- [ ] [Kubernetes](books/08-kubernetes/README.md) — deploy 3 ส่วนเป็น Deployment แยกกัน, ตั้ง Service เชื่อมกัน, Ingress ให้ dashboard เข้าถึงจากภายนอกได้
- [ ] [CI/CD](books/16-cicd/README.md) — pipeline build image ทั้ง 3 ส่วน, push ขึ้น registry, deploy อัตโนมัติเมื่อ merge

**เช็คพอยต์**: push โค้ดแล้ว BOP ทั้งระบบ (API + worker + dashboard) deploy ขึ้น K8s เองอัตโนมัติ ไม่ deploy มือ

### 2.2 Observability: มองเห็นว่า Bot ไหนพังเพราะอะไร

- [ ] [Prometheus + Grafana](books/14-observability/01-metrics.md) — เก็บ metric: job success/fail count, job duration histogram, worker pool utilization
- [ ] **[Fluent Bit + OpenSearch](books/14-observability/02-logging.md) — ย้าย log storage ของ job logs จาก PostgreSQL (ที่ทำไว้ชั่วคราวใน 1.5) มาที่นี่จริงจัง**: Go เขียน log line ออกเป็น structured output ตามปกติ (ไม่ยิง OpenSearch API ตรงจากโค้ด Go) → Fluent Bit อ่าน/parse/ส่งต่อเข้า OpenSearch → ค้นหาย้อนหลังได้แบบ full-text (เช่น "หา job ทั้งหมดที่ error message มีคำว่า selector not found") พร้อมตั้ง index lifecycle ลบ log เก่าอัตโนมัติ — **ข้อสำคัญ**: ย้ายเฉพาะเนื้อหา log text เท่านั้น ส่วน job metadata/status ยังอยู่ PostgreSQL เหมือนเดิม (ต้องการ transactional consistency ที่ OpenSearch ให้ไม่ได้) เพราะออกแบบผ่าน `Logger` interface ไว้แล้วตั้งแต่ 1.5 การสลับ storage backend นี้ไม่ต้องแก้โค้ด bot/API เลย
- [ ] [OpenTelemetry + Jaeger](books/14-observability/03-tracing.md) — trace request ตั้งแต่ dashboard กดสั่งรัน → API รับ → worker execute → เห็นทั้งเส้นทางว่าช้าตรงไหน
- [ ] ตั้ง alert: ถ้า bot ตัวใดตัวหนึ่ง fail ติดกันเกิน N ครั้ง (สัญญาณคลาสสิกว่าเว็บเป้าหมายเปลี่ยน HTML structure) ให้แจ้งเตือนอัตโนมัติ
- [ ] ย้าย broadcaster ของ live log (จาก 1.5) จาก in-memory ไปเป็น Redis Pub/Sub ถ้า API ถูก scale เป็นหลาย replica แล้ว (WebSocket แต่ละ client อาจต่อกับ replica คนละตัวกับที่ worker report เข้ามา)

### 2.3 Auth และ Gateway ระดับ Production

- [ ] [Authentication](books/09-authentication/README.md) — ทำ JWT + refresh token ให้ถูกต้อง, เพิ่ม RBAC (role `admin` แก้ bot ได้ทุกตัว, role `viewer` ดูได้อย่างเดียว)
- [ ] [APISIX](books/10-apisix/README.md) — วาง gateway หน้า Go API, ทำ rate limit ต่อ user (ป้องกันคนสั่งรัน bot ถี่เกินไปจน worker ล้น)

### 2.4 Data Layer ที่รับสเกลได้จริง

- [ ] [PostgreSQL — Transaction/MVCC](books/05-postgresql/03-transactions-and-mvcc.md) — ป้องกัน race condition ตอนหลาย worker อัปเดต job status พร้อมกัน
- [ ] [PostgreSQL — Partitioning](books/05-postgresql/05-partitioning-and-replication.md) — job history โตเร็วมาก ต้อง partition ตามเวลา (รายเดือน) ไม่งั้น query ช้าลงเรื่อยๆ
- [ ] [Redis](books/06-redis/README.md) — cache summary stat ของหน้า dashboard, ทำ rate limiter (token bucket) จำกัดความถี่ที่แต่ละ bot ยิง request หาเว็บเป้าหมาย ป้องกันโดนบล็อก IP

### 2.5 Queue และ Workflow Orchestration (จุดที่ domain นี้ mapping ตรงที่สุด)

- [ ] [Microservices — Retry/Circuit Breaker](books/11-microservices/04-saga-and-resilience.md) — ใส่ retry with backoff ตอนยิง request หาเว็บเป้าหมาย, circuit breaker ตัดการยิงถ้าเว็บเป้าหมายล่มต่อเนื่อง
- [ ] [Kafka](books/12-kafka/README.md) — แทนที่ channel/Redis queue เดิมด้วย Kafka topic สำหรับ dispatch job ให้ worker หลายตัว รองรับ scale ออกแนวนอน, ทำ DLQ สำหรับ job ที่ fail ซ้ำๆ ให้คนมาดูทีหลังแทนที่จะหายไปเงียบๆ
- [x] **[Temporal](books/13-temporal/README.md) — จุดที่คุ้มค่าที่สุดของทั้ง roadmap** เขียน bot execution flow ใหม่เป็น Temporal workflow แล้ว: แต่ละ step เป็น Activity ที่มี retry policy ของตัวเอง (`NonRetryableErrorTypes` แยก config error ที่ retry ไปก็ไม่มีประโยชน์ออกจาก error ที่ retry แล้วอาจสำเร็จ) — `internal/workflow` (เดิมเป็น lightweight version + `internal/scheduler`) ย้ายไปใช้ Temporal จริงทั้งหมดแล้ว, `internal/scheduler` ถูกลบทิ้งเพราะใช้ Temporal Schedules แทน ดู decision log ที่ [projects/bot-orchestration-platform/TEMPORAL-MIGRATION-TODO.md](projects/bot-orchestration-platform/TEMPORAL-MIGRATION-TODO.md)
  - [ ] **ยังไม่ทำ**: compensation logic (ตัวอย่าง "notify fail หลัง save result สำเร็จ" ที่ [books/13-temporal/02-retry-and-compensation.md](books/13-temporal/02-retry-and-compensation.md)) — รอ bot ที่มี side effect จริงจังก่อน ตอนนี้มีแค่ `http-status-checker` ที่ไม่มีอะไรต้อง rollback
  - [ ] **ยังไม่ verify**: รัน end-to-end จริงกับ Temporal server ในเครื่อง (build/test -race ผ่านหมด แต่ยังไม่เคย `docker compose up -d` สำเร็จเพราะติด Docker permission ตอน implement)
- [ ] [Kubernetes CronJob](books/08-kubernetes/04-scaling-and-control-plane.md) — ให้ bot บางตัวรันตาม schedule (ทุกชั่วโมง) แทนที่จะสั่งมือ, ตั้ง HPA scale worker ตาม queue depth

**ฟีเจอร์พื้นฐานแนว Control-M ที่เพิ่มเข้ามาใน step นี้** (ยังอยู่ในระดับ "infra ต้องมี" ก่อนจะไปฟีเจอร์ขั้นสูงใน Phase 3):

- [ ] **Job Dependency (DAG)** — job B รันได้ก็ต่อเมื่อ job A สำเร็จแล้ว ใช้ Temporal child workflow หรือ `workflow.Await` รอผลลัพธ์ของ workflow อื่นก่อนเริ่ม
- [ ] **Calendar-based Scheduling** — ไม่ใช่แค่ cron ธรรมดา แต่รองรับกฎแบบ "ข้ามวันหยุด", "รันเฉพาะวันทำการสุดท้ายของเดือน" — เขียน logic คำนวณวันที่เอง (โจทย์ scheduling ที่มีความลึกน่าสนใจ)
- [ ] **SLA Monitoring + Alert** — ถ้า job ไม่เสร็จภายในเวลาที่คาด (เช่น เกิน 2 เท่าของ duration เฉลี่ยที่ผ่านมา) ยิง alert ทันที ต่อยอดจาก metric ที่เก็บไว้แล้วใน 2.2
- [ ] **Rerun เฉพาะ Node ที่ Fail** — chain ยาวๆ ถ้า fail กลางทาง ต้อง rerun แค่ node นั้นได้โดยไม่รันทั้ง chain ใหม่ (Temporal รองรับผ่านการ query event history หา checkpoint ล่าสุดที่สำเร็จ)
- [ ] **Audit Trail** — บันทึกว่าใครสั่ง/แก้ config job ไหนเมื่อไหร่ ต่อยอดจาก RBAC ที่วางไว้ใน 2.3

### 2.6 System Design: อธิบายทั้งระบบเป็นคำพูดได้

- [ ] [CAP / PACELC](books/17-system-design/01-cap-and-consistency.md) — อธิบายได้ว่า job status ที่แสดงบน dashboard ยอมรับ eventual consistency ได้แค่ไหน (เช่น real-time log ผ่าน WebSocket อาจ deliver ช้ากว่า DB จริงเล็กน้อย ยอมรับได้ไหม)
- [ ] [Sharding/Replication/Distributed Cache](books/17-system-design/02-scaling-data.md) — ถ้า BOP ต้องรองรับ bot นับหมื่นตัว/job นับล้าน record ต่อวัน จะ shard job history อย่างไร
- [ ] [HA/DR](books/17-system-design/03-cdn-ha-dr.md) — ถ้า worker node ตายกลางทางที่ bot กำลังรันอยู่ ระบบ recover เองได้แค่ไหน (เชื่อมกับ durable execution ของ Temporal ที่เพิ่งทำในข้อ 2.5)

### 2.7 Security (ที่จำเป็นเพราะ Bot ยิง Request ตาม URL ที่ผู้ใช้กำหนด)

- [ ] [OWASP Top 10](books/15-security/01-owasp-top-10.md) — โฟกัส **SSRF (Server-Side Request Forgery)** เป็นพิเศษ: BOP รับ URL จาก user แล้วให้ backend ไปยิง request ตาม — ถ้าไม่ validate ดี user อาจใส่ URL ชี้กลับเข้า internal network (เช่น `http://localhost:6379` ไปแตะ Redis เอง) ต้องทำ URL allowlist/validation ป้องกัน เป็นบทเรียน security ที่ตรงกับสถาปัตยกรรมของ BOP เป๊ะๆ ไม่ใช่ทฤษฎีลอยๆ
- [ ] [Vault/Secrets](books/15-security/02-vault-secrets-network-policy.md) — เก็บ credential ที่ bot อาจต้องใช้ (เช่น API key ของบริการภายนอก) ผ่าน Vault แทน hardcode

**เป้าหมายเมื่อจบ phase**: BOP เวอร์ชันเต็ม — dashboard (Next.js) + API (Go) + worker ที่รันผ่าน Temporal workflow + Kafka queue + PostgreSQL (partitioned) + Redis cache + APISIX gateway + auth/RBAC + observability ครบ deploy บน K8s ผ่าน CI/CD พร้อม SSRF protection — อธิบาย trade-off ทุกจุดในสัมภาษณ์ได้จากของจริงที่สร้างเอง ไม่ใช่ท่องทฤษฎี

---

## Phase 3 — Workload Automation Depth (แรงบันดาลใจจาก Control-M)

Phase นี้ต่างจาก Phase 2 ตรงที่ไม่ใช่การเรียนเทคโนโลยีใหม่ แต่เป็นการ**เอา infra ที่มีอยู่แล้วมาสร้างฟีเจอร์ระดับ enterprise ทับลงไป** — นี่คือ phase ที่ทำให้ BOP ไม่ใช่แค่ "bot runner" ธรรมดา แต่กลายเป็นของที่เทียบเคียง concept กับ Control-M/Airflow ได้จริง เหมาะทำหลังจาก Phase 2 เสร็จแล้วและมีเวลาเหลือ เพราะเป็นตัวสร้างความต่างจากคนอื่นในตลาดชัดเจนที่สุด

### 3.1 Condition-Based Job Triggering (แทน DAG ตายตัว)

Control-M ไม่ได้ผูก job ต่อ job แบบ parent-child ตรงๆ แต่ใช้ **condition**: job หนึ่งจบแล้ว "set" condition (เช่น `FILE_READY`), job อื่นที่ "wait for" condition นั้นถึงจะเริ่ม — โมเดลนี้ยืดหยุ่นกว่า DAG เพราะเป็น many-to-many ได้ (หลาย job รอ condition เดียวกัน, หนึ่ง job รอหลาย condition พร้อมกัน)

- [ ] สร้าง condition store (Postgres/Redis) ที่ job ไป "set" condition หลังจบงาน
- [ ] job อื่น subscribe/wait condition ก่อนเริ่ม — ผูกกับ Temporal Signal pattern

### 3.2 Resource Pools (จำกัดการใช้ Shared Resource พร้อมกัน)

Control-M มี "quantitative resource" จำกัดว่ากี่ job พร้อมกันที่แตะ resource เดียวกันได้ (เช่น database connection, เว็บเป้าหมายเดียวกัน) ป้องกัน overload ปลายทาง

- [ ] Reframe rate limiter ที่ทำไว้แล้วใน 2.4 ให้เป็น "resource pool" ที่ config ได้ต่อ target (เช่น "เว็บ X รับพร้อมกันได้ไม่เกิน 3 job")
- [ ] semaphore ผูกกับชื่อ resource แทนที่จะเป็น global limit เดียวทั้งระบบ

### 3.3 Flow Control (จัดกลุ่ม Job เป็น Flow ควบคุมพร้อมกัน)

Control-M มี "SMART Folder" จัดกลุ่ม job หลายตัวเป็น flow เดียว แล้ว hold/resume/kill ทั้ง flow พร้อมกันได้

- [ ] ออกแบบ data model: `Flow` มีหลาย `Job`, สถานะของ Flow derive จากสถานะ job ย่อยทั้งหมด
- [ ] UI: ปุ่ม pause/resume/kill ทั้ง flow ในหน้า dashboard — ต่อยอด state management ฝั่ง frontend ที่ทำไว้ใน Phase 1

### 3.4 Approval Gate (Human-in-the-Loop)

Control-M รองรับ job ที่ต้องรอ manual confirmation ก่อนรันต่อ (เช่น "ยืนยันก่อน deploy เข้า production") — ฟีเจอร์นี้เป็น Temporal pattern ขั้นสูงที่ portfolio ทั่วไปไม่มี คุ้มค่ามากที่จะทำ

- [ ] ใช้ Temporal Signal (`workflow.GetSignalChannel`) หยุด workflow รอสัญญาณจาก user ก่อนไปขั้นตอนถัดไป
- [ ] Frontend: ปุ่ม "Approve"/"Reject" ในหน้า job monitor ที่ยิง signal กลับไปยัง workflow ที่ค้างรออยู่แบบ real-time

### 3.5 Gantt / Timeline Visualization

Control-M มี Forecast view แสดง timeline ของ job ที่จะรัน/รันไปแล้วแบบ Gantt chart

- [ ] Frontend: แสดง timeline ของ job ตามเวลาที่ schedule ไว้ ซ้อนกับ job ที่รันจริง (ใช้ library สำเร็จรูปหรือสร้างเองด้วย CSS grid ก็ได้) — โจทย์ data visualization ที่ยากกว่า table ธรรมดา ฝึก frontend เพิ่มอีกระดับจาก Phase 1

### 3.6 Alert Triage / Ownership

Control-M มี Alert console ที่ operator กด acknowledge, assign เจ้าของดูแล, ปิด alert เมื่อแก้เสร็จ

- [ ] ต่อยอดจาก alert ที่ทำไว้ใน 2.2 ให้มี state machine: `new → acknowledged → resolved` พร้อมบันทึกว่าใคร handle
- [ ] หน้า UI สำหรับ operator ดู alert ที่ยังไม่ถูก acknowledge ทั้งหมด

### 3.7 Job Definition Versioning + Environment Promotion

Control-M แยก environment dev/test/prod และ promote job definition ข้าม environment ได้อย่างมีการควบคุม

- [ ] เก็บ version history ของ bot config ทุกครั้งที่แก้ไข (ดู diff ย้อนหลังได้)
- [ ] ทำ "promote" config จาก staging ไป production ผ่าน approval flow — แนวคิดเดียวกับ GitOps ที่อธิบายไว้ใน [books/16-cicd/03-gitops.md](books/16-cicd/03-gitops.md) แค่เปลี่ยนจาก "Git เป็น source of truth ของ K8s manifest" เป็น "DB/Git เป็น source of truth ของ job definition"

**เป้าหมายเมื่อจบ phase**: BOP มีฟีเจอร์ที่เทียบเคียง concept หลักของ enterprise workload automation tool ได้จริง — พอจะยืนอธิบายต่อหน้า hiring manager ที่ใช้ Control-M อยู่ทุกวันได้ว่า "ผมเข้าใจปัญหาที่เครื่องมือแบบนี้แก้ และสร้าง alternative แบบ cloud-native ขึ้นมาเองได้"

---

## สรุปลำดับภาพรวม

```
Phase 0: Golang Bot Engine              ──► worker pool รัน bot ได้จริง ผ่าน CLI
Phase 1: React/Next.js Dashboard        ──► control BOP ผ่าน UI, real-time log ผ่าน WebSocket
Phase 2.1: Docker + K8s + CI/CD         ──► BOP deploy อัตโนมัติทั้งระบบ
Phase 2.2: Observability                ──► รู้ทันทีว่า bot ไหนพังเพราะอะไร
Phase 2.3: Auth + APISIX                ──► RBAC + rate limit ระดับ production
Phase 2.4: PostgreSQL scale + Redis     ──► รองรับ job history ปริมาณมาก
Phase 2.5: Kafka + Temporal + K8s CronJob ──► queue + workflow orchestration จริง + DAG/Calendar/SLA/Rerun/Audit พื้นฐาน
Phase 2.6: System Design                ──► อธิบาย trade-off ทั้งระบบเป็นคำพูดได้
Phase 2.7: Security (SSRF focus)        ──► ปิดช่องโหว่ที่มาจากสถาปัตยกรรมของ BOP เอง
Phase 3: Workload Automation Depth      ──► Condition trigger, Resource pool, Flow control,
         (แรงบันดาลใจจาก Control-M)         Approval gate, Gantt view, Alert triage, Config versioning
```

อย่าข้าม phase หรือเรียนขนานกันหลาย phase พร้อมกัน — ทำทีละ phase ให้ BOP โตขึ้นเรื่อยๆ Phase 0-2 คือ "ต้องมี" สำหรับ portfolio ที่ใช้งานได้จริง ส่วน Phase 3 คือ "ยิ่งมียิ่งเด่น" ทำเมื่อ Phase 2 นิ่งแล้วและมีเวลาเหลือ ตอนจบจะได้ระบบเดียวที่เล่าเรื่องได้ครบทุกมิติ (frontend, backend, infra, security, product depth) ดีกว่าความรู้กระจัดกระจายที่ไม่เชื่อมกัน
