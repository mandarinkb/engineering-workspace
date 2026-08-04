# TODO: Next.js Dashboard สำหรับควบคุม BOP (Roadmap Phase 1)

> **สถานะ: implement ครบ 1.1-1.9 แล้ว** — `npm run typecheck`/`lint`/`test`/`build` ผ่านหมด ทดสอบ `npm run dev` จริงแล้วว่า `/` redirect ไป `/login` ถูกต้องตามที่ middleware ควรทำ backend prerequisite ทั้งหมด (ข้อ 1-5 ด้านล่าง) implement ครบและมี `go test -race` (รวม `httptest` smoke test เต็มของ `internal/api/http`) คุมไว้แล้วเช่นกัน
>
> **ยังไม่ verify**: end-to-end จริงกับ Temporal server (สร้าง job จริงผ่าน dashboard, ดู log ไหลจริงผ่าน SSE, ดู workflow run จริง) เพราะ session ที่ implement ติด Docker permission ในเครื่อง (`permission denied` ที่ `/var/run/docker.sock`) เหมือนที่เจอตอนทำ Temporal migration — ต้องให้ผู้ใช้ทดสอบเองตาม [backend README](../bot-orchestration-platform/README.md) หัวข้อ "รันยังไง" คู่กับ [dashboard README](README.md)
>
> **สิ่งที่ค้นพบเพิ่มระหว่าง implement (ไม่ได้คาดไว้ตอนวางแผน)**:
> - **ต้อง Node.js >= 22** จริงๆ ไม่ใช่แค่ >= 20 — เครื่องที่พัฒนามี Node v19.9.0 ซึ่งเก่าเกินสำหรับ `@tailwindcss/oxide` (native binding), Vitest 4, และ `jsdom`/`undici` (`webidl.util.markAsUncloneable is not a function` บน Node 20) ติดตั้ง Node 22 ผ่าน `nvm` แล้วทุกอย่างผ่านสะอาด ไม่มี engine warning เลย — ดู `.nvmrc`
> - **Backend Prerequisite ข้อ 4 (Logger) เข้าใจผิดตอนวางแผน**: `logger.Logger`/`internal/logger/memory` มี persist + per-jobID broadcast (`Subscribe`) ให้ครบอยู่แล้วตั้งแต่ Phase 0 (ไม่ได้เขียนตอนนี้) สิ่งที่ต้องเพิ่มจริงๆ มีแค่ SSE HTTP handler (`GET /jobs/{id}/logs/stream`) ที่เรียก `Subscribe`/`History` ที่มีอยู่แล้วเท่านั้น — งานเบากว่าที่วางแผนไว้มาก
> - SSE endpoint ออกแบบให้ **ส่ง backfill (log เก่า) เป็นส่วนหนึ่งของ stream เดียวกันเลย** (ไม่ใช่แยก REST call ต่างหากแล้วค่อยเปิด SSE) เพื่อไม่ให้ log ซ้ำ — ดูผลกระทบต่อ `use-job-log-stream.ts` ในหัวข้อ TODO Checklist ข้อ 1.5 ด้านล่าง (ต่างจากแผนเดิมที่เขียนไว้เล็กน้อย)

## บริบท

โปรเจกต์นี้คือ **Bot Orchestration Platform (BOP)** — ดูภาพรวมทั้งหมดที่ [FULLSTACK-INFRA-ROADMAP.md](../../FULLSTACK-INFRA-ROADMAP.md) Backend (Go) เสร็จถึง Phase 0 + Phase 2.5 (Temporal migration) แล้ว ดู [projects/bot-orchestration-platform/README.md](../bot-orchestration-platform/README.md) — มี HTTP API พร้อมใช้งานเต็มที่ที่ `:8080`

**งานนี้คือ Phase 1 เต็มรูปแบบ**: สร้าง Next.js dashboard ที่คุยกับ Go API ชุดนี้ ตามรายละเอียดที่ [FULLSTACK-INFRA-ROADMAP.md บรรทัด 48-119](../../FULLSTACK-INFRA-ROADMAP.md#phase-1--react--nextjs-สร้าง-dashboard-ควบคุม-bop) — เอกสารนี้ไม่ได้แทนที่ roadmap นั้น แค่ขยายความให้ลงมือทำได้จริง (โครงสร้างโค้ด, gap ของ backend ที่ต้องเติมก่อน, type ที่ต้อง mirror)

**เป้าหมายของ phase (คัดจาก roadmap)**: dashboard ที่ backend dev ทั่วไปสร้างไม่ได้ง่ายๆ (real-time, dynamic form) ไม่ใช่แค่ CRUD ธรรมดา — จุดขายที่สุดคือหน้า **Live Job Monitor** (ข้อ 1.5) ที่เห็น log ของ bot ที่กำลังรันอยู่แบบ real-time

## อ่านก่อนเริ่ม

1. [FULLSTACK-INFRA-ROADMAP.md บรรทัด 48-119](../../FULLSTACK-INFRA-ROADMAP.md) — Phase 1 เต็ม แบ่งเป็นข้อ 1.1-1.9 พร้อม checklist ทฤษฎีที่ต้องเรียน (เอกสารนี้จะอ้างอิงเลขข้อเดียวกัน)
2. [projects/bot-orchestration-platform/README.md](../bot-orchestration-platform/README.md) หัวข้อ "รันยังไง" — ต้องรัน backend ครบ 3 อย่างคู่กันตลอดการพัฒนา dashboard นี้: `docker compose up -d` (Temporal), `go run ./cmd/bop-worker`, `go run ./cmd/bop`
3. [internal/api/http/handler.go](../bot-orchestration-platform/internal/api/http/handler.go) — endpoint contract ตัวจริงของ backend ตอนนี้ (อย่าเดา ให้เปิดอ่านโค้ดจริงเสมอ เพราะเอกสารนี้อาจตกยุคถ้า backend เปลี่ยนทีหลัง)
4. [internal/job/job.go](../bot-orchestration-platform/internal/job/job.go), [internal/logger/logger.go](../bot-orchestration-platform/internal/logger/logger.go), [internal/workflow/workflow.go](../bot-orchestration-platform/internal/workflow/workflow.go) — struct ที่ต้อง mirror เป็น TypeScript type (ดูหัวข้อ "Type ที่ต้อง mirror" ด้านล่าง)
5. Books อ้างอิง: [books/09-authentication/](../../books/09-authentication/README.md) (JWT/OAuth สำหรับข้อ 1.8), [books/06-redis/02-pubsub-and-streams.md](../../books/06-redis/02-pubsub-and-streams.md) (บริบทตอน scale ข้อ 1.5 — ดูหมายเหตุ Redis Pub/Sub ใน roadmap)

## ตัดสินใจไว้แล้ว (คุยกับเจ้าของ repo ก่อนเขียนเอกสารนี้ — ไม่ต้องถามซ้ำ)

1. **ตำแหน่งโปรเจกต์: `projects/bop-dashboard/` แยก top-level ออกจาก backend** (`projects/bot-orchestration-platform/`) ไม่ใช่ monorepo เดียวกัน — เหตุผล: แยก deploy/CI ได้อิสระจากกัน (dashboard deploy Vercel ตาม roadmap 1.9, backend deploy K8s ตาม Phase 2.1 ทีหลัง) **ผลที่ตามมา**: ต้องเปิด CORS ที่ฝั่ง Go API ให้ origin ของ dashboard เรียกได้ (ดู Backend Prerequisite ข้อ 3 ด้านล่าง — ยังไม่มีตอนนี้)
2. **Real-time channel: Server-Sent Events (SSE) ไม่ใช่ WebSocket** — log stream เป็น one-way (server→client เท่านั้น dashboard ไม่เคยต้องส่งอะไรกลับผ่าน channel นี้เลย) SSE เป็น HTTP ธรรมดา (`Content-Type: text/event-stream`) เบราว์เซอร์มี auto-reconnect ในตัวผ่าน `EventSource` โดยไม่ต้องเขียน reconnect logic เอง ฝั่ง Go เขียนง่ายกว่า WebSocket มาก (ไม่ต้องพึ่ง library ภายนอกอย่าง `gorilla/websocket` เพื่อ handshake/ping-pong เอง) — **trade-off ที่ยอมรับ**: SSE ส่งได้ทางเดียว ถ้าอนาคตต้องการส่งคำสั่งกลับผ่าน channel เดียวกัน (เช่น "pause stream") ต้องเปิด endpoint แยกต่างหาก
3. **Package manager: npm** (ไม่ใช่ pnpm ตามที่เสนอไว้ตอนแรก) — เครื่องที่ implement จริงมี npm อยู่แล้ว ไม่ติดตั้ง global tool เพิ่มโดยไม่จำเป็น (เอกสารตอนวางแผนบอกไว้แล้วว่า pnpm ไม่ใช่ hard requirement)
4. **โครงสร้างโค้ด: feature-based** ไม่ใช่ type-based (`components/`, `hooks/`, `services/` แยกตามประเภทไฟล์แบบเดิม) — ดูเหตุผลเต็มๆ ที่หัวข้อ "โครงสร้างโปรเจกต์" ด้านล่าง

## Backend Prerequisite (Go API ที่ dashboard ต้องใช้) — ทำครบแล้วทั้ง 5 ข้อ

เช็คจาก `internal/api/http/handler.go` ตอนวางแผน (route ที่มีตอนนั้น: `POST /bots/{name}/jobs`, `GET /jobs`, `GET /jobs/{id}`, `GET /jobs/{id}/logs`, `DELETE /jobs/{id}`, `GET /workflow-runs`, `GET /workflow-runs/{id}`) พบว่าไม่พอสำหรับ Phase 1 — เพิ่มครบแล้วทั้งหมด:

1. [x] **`GET /bots`** — เพิ่ม `worker.Pool.List() []bot.Bot` (คืน `bot.Bot` เต็มๆ ไม่ใช่แค่ชื่อ ตามที่ Open Question ข้อ 1 ตัดสินใจ) แล้ว handler แปลงเป็น `[]{name, config_schema}` (`botSummary` ใน handler.go)
2. [x] **Config schema ต่อ bot** — เพิ่ม `ConfigField`/`ConfigFieldType` และ method `ConfigSchema() []ConfigField` เข้า `bot.Bot` interface ตรงๆ (Open Question ข้อ 1 ตัดสินใจ: backend-declared ไม่ใช่ static mapping ฝั่ง frontend เพราะสอดคล้องกับหลักการ pluggable ของทั้งระบบมากกว่า) `httpstatus.Bot` ประกาศ field `url` (required) เป็นตัวอย่างเดียวที่มีตอนนี้
3. [x] **CORS middleware** — `dashboardOrigin` constant + header ใน `Server.ServeHTTP` (`internal/api/http/handler.go`) รองรับ preflight `OPTIONS` และ `Access-Control-Allow-Credentials: true` (จำเป็นเพราะ auth ใช้ cookie ข้าม origin)
4. [x] **SSE endpoint** `GET /jobs/{id}/logs/stream` — **ไม่ต้อง "redesign" Logger เลยอย่างที่คิดตอนวางแผน** `logger.Logger`/`internal/logger/memory` มี `Subscribe`/`History` (persist + per-jobID broadcast) ครบมาตั้งแต่ Phase 0 แล้ว handler แค่เรียก `Subscribe` ก่อน `History` (ลำดับสำคัญ — กัน log หายระหว่างสองคำเรียก ดู comment เต็มที่ `handleStreamJobLogs`) แล้ว stream ทั้ง backfill และ live update ผ่าน connection เดียวกัน
5. [x] **Auth** — `internal/auth/jwt.go` (JWT HS256 เขียนเองด้วย stdlib ล้วนๆ ไม่เพิ่ม dependency) + `POST /login`/`POST /logout`/`GET /me` ใน handler.go ออก/ล้าง httpOnly cookie

ทั้งหมดมี test คุมไว้ที่ `internal/auth/jwt_test.go` (round-trip, tampered payload, expired, wrong secret) และ `internal/api/http/handler_test.go` (bots list, job create+read+logs, auth flow เต็ม, CORS header) — รันผ่าน `httptest` ล้วนๆ ไม่ต้องมี Temporal/Docker เลย

## โครงสร้างโปรเจกต์ที่แนะนำ (feature-based)

**ทำไมเลือก feature-based แทน type-based**: โครงสร้างแบบเดิมที่แยกโฟลเดอร์ตาม "ประเภทไฟล์" (`components/`, `hooks/`, `services/`, `types/` แยกกันหมด) ใช้ได้ดีตอนแอปเล็ก แต่พอ feature เพิ่มขึ้น (bots, jobs, live-monitor, workflow-runs, auth, และ Phase 3 enterprise feature ในอนาคตอย่าง approval-gate/gantt-view) ทุกโฟลเดอร์จะเต็มไปด้วยไฟล์จากหลาย feature ปนกัน หา "ทุกอย่างของ feature X" ทีต้องกระโดดข้าม 4-5 โฟลเดอร์ — **feature-based** จัดกลุ่มตาม "feature ทำอะไร" แทน (ทุกไฟล์ของ jobs feature อยู่ใต้ `features/jobs/` ที่เดียว) ลบ feature ทิ้งก็ลบทั้งโฟลเดอร์ได้เลย ไม่ต้องไล่หาไฟล์กระจัดกระจาย — เป็น pattern มาตรฐานของ React/Next.js enterprise repo จริง (รู้จักในชื่อ "bulletproof-react" pattern)

```
projects/bop-dashboard/
├── src/
│   ├── app/                        # Next.js App Router — เก็บแค่ "routing + layout" เท่านั้น
│   │   │                           # page.tsx ในนี้ต้อง "บาง" ที่สุด แค่ import component จาก
│   │   │                           # features/ มาวางแล้ว pass params เข้าไป ห้ามมี business
│   │   │                           # logic อยู่ในไฟล์ใต้ app/ เลยแม้แต่บรรทัดเดียว — เหตุผล:
│   │   │                           # แยก "URL structure" ออกจาก "feature logic" เพื่อย้าย route
│   │   │                           # (เช่น เปลี่ยน URL จาก /jobs เป็น /monitoring/jobs) ได้โดย
│   │   │                           # ไม่ต้องแตะ logic เลย
│   │   ├── layout.tsx              # root layout (font, providers ระดับ global เช่น QueryClientProvider)
│   │   ├── (auth)/
│   │   │   └── login/page.tsx      # หน้า login (ไม่มี nav bar ของ dashboard)
│   │   └── (dashboard)/
│   │       ├── layout.tsx          # nav bar ที่ใช้ร่วมทุกหน้าใน dashboard (ข้อ 1.6)
│   │       ├── bots/
│   │       │   └── page.tsx        # ข้อ 1.2 — แค่ import <BotListPage/> จาก features/bots
│   │       ├── jobs/
│   │       │   ├── page.tsx        # ข้อ 1.4 — job history table
│   │       │   └── [id]/
│   │       │       └── page.tsx    # ข้อ 1.5 — live job monitor (Server Component โหลด log
│   │       │                       # เก่าก่อน ส่งต่อให้ Client Component เปิด SSE ต่อ — ดูข้อ 1.6)
│   │       └── workflow-runs/
│   │           ├── page.tsx
│   │           └── [id]/page.tsx
│   │
│   ├── features/                   # หัวใจของโครงสร้างนี้ — 1 feature = 1 โฟลเดอร์ ครบทุกอย่าง
│   │   │                           # ของ feature นั้นอยู่ที่เดียว (component, hook, api call, type,
│   │   │                           # zod schema) ห้าม feature หนึ่ง import จากอีก feature หนึ่ง
│   │   │                           # ตรงๆ ถ้าต้องแชร์อะไรข้าม feature ให้ยกขึ้นไปที่ src/components
│   │   │                           # หรือ src/lib แทน (กัน dependency พันกันไปมาจนแยกไม่ออก)
│   │   ├── bots/
│   │   │   ├── api/
│   │   │   │   └── use-bots.ts     # TanStack Query hook: useQuery(['bots'], ...) เรียก GET /bots
│   │   │   ├── components/
│   │   │   │   ├── bot-card.tsx    # ข้อ 1.2 — <BotCard/>
│   │   │   │   ├── bot-list.tsx    # ข้อ 1.2 — <BotList/> + filter/search (useState)
│   │   │   │   └── bot-form.tsx    # ข้อ 1.3 — dynamic form ตาม config schema ของ bot ที่เลือก
│   │   │   ├── schemas/
│   │   │   │   └── bot-config.schema.ts  # zod schema, mapping กับ config schema จาก backend
│   │   │   └── types/
│   │   │       └── index.ts        # Bot, ConfigField (mirror จาก Go — ดูหัวข้อถัดไป)
│   │   │
│   │   ├── jobs/
│   │   │   ├── api/
│   │   │   │   ├── use-jobs.ts           # ข้อ 1.4 — list + filter (bot/status/ช่วงเวลา)
│   │   │   │   ├── use-job.ts            # GET /jobs/{id}
│   │   │   │   └── use-job-log-stream.ts # ข้อ 1.5 — hook ที่เปิด EventSource (SSE) เอง เก็บ
│   │   │   │                             # state connecting/connected/disconnected + log line
│   │   │   │                             # ที่ไหลเข้ามา ให้ component ดึงไปแสดงตรงๆ
│   │   │   ├── components/
│   │   │   │   ├── job-table.tsx         # ข้อ 1.4 — table + pagination + status badge
│   │   │   │   ├── job-filter.tsx        # sync กับ URL query param
│   │   │   │   └── job-log-viewer.tsx    # ข้อ 1.5 — auto-scroll, badge บอกสถานะ connection
│   │   │   └── types/
│   │   │       └── index.ts              # Job, JobStatus, LogLine (mirror จาก Go)
│   │   │
│   │   ├── workflow-runs/
│   │   │   ├── api/
│   │   │   │   └── use-workflow-runs.ts
│   │   │   ├── components/
│   │   │   │   ├── workflow-run-table.tsx
│   │   │   │   └── step-result-list.tsx  # แสดง StepResult ทีละ step ของ 1 run
│   │   │   └── types/
│   │   │       └── index.ts              # WorkflowRunSummary, WorkflowRunDetail, StepResult
│   │   │
│   │   └── auth/
│   │       ├── api/
│   │       │   └── use-login.ts          # ข้อ 1.8
│   │       ├── components/
│   │       │   └── login-form.tsx
│   │       └── hooks/
│   │           └── use-auth.ts           # อ่าน session/user จาก context
│   │
│   ├── components/                 # UI component ที่ใช้ "ข้าม feature" เท่านั้น (shared, ไม่ผูก
│   │   │                           # business logic ของ feature ไหนเลย — ถ้า component ไหนรู้จัก
│   │   │                           # concept ของ Job/Bot ให้ย้ายกลับไปอยู่ใน features/ แทน)
│   │   ├── ui/                     # Button, StatusBadge, Table, Modal ฯลฯ (ข้อ 1.7 — Tailwind)
│   │   └── layout/
│   │       └── nav-bar.tsx
│   │
│   ├── lib/                        # Infrastructure กลางที่ไม่ผูก feature ไหนเป็นพิเศษ
│   │   ├── api-client.ts           # fetch wrapper กลาง: base URL (NEXT_PUBLIC_API_URL),
│   │   │                           # แนบ auth header, แปลง error response เป็น รูปแบบเดียวกัน
│   │   │                           # ทุก feature/api/*.ts เรียกผ่านตัวนี้ ไม่ยิง fetch() ตรงๆ เอง
│   │   ├── query-client.ts         # TanStack QueryClient config กลาง (retry, staleTime default)
│   │   └── sse-client.ts           # helper เปิด/ปิด EventSource + reconnect กลาง ให้
│   │                               # use-job-log-stream.ts เรียกใช้ (ไม่เขียน EventSource ตรงๆ
│   │                               # กระจายอยู่หลายที่)
│   │
│   ├── hooks/                      # hook กลางที่ไม่ผูก feature ไหน (เช่น useDebounce, useMediaQuery)
│   ├── types/
│   │   └── api.ts                  # type กลางที่ใช้ข้าม feature เช่น ApiError, Paginated<T>
│   └── middleware.ts               # ข้อ 1.8 — protected route ผ่าน Next.js middleware (เช็ค
│                                   # httpOnly cookie ก่อนปล่อยเข้า (dashboard) route group)
│
├── public/
├── .env.local.example              # NEXT_PUBLIC_API_URL=http://localhost:8080 เป็นต้น
├── package.json
├── tsconfig.json
├── vitest.config.ts                # ข้อ 1.9
└── tailwind.config.ts              # ข้อ 1.7
```

**กติกาการ import ที่ต้องรักษาไว้ตลอด** (ถ้าไม่รักษา โครงสร้างนี้จะพังกลับไปเป็น spaghetti เหมือนเดิม):
- `app/` import จาก `features/` ได้อย่างเดียว (ห้ามมี logic เอง)
- `features/X/` import จาก `components/`, `lib/`, `hooks/`, `types/` ได้ แต่ **ห้าม** import จาก `features/Y/` อื่นตรงๆ
- `components/`, `lib/`, `hooks/` **ห้าม** import อะไรจาก `features/` เลย (ทิศทางเดียว: กลางไหลเข้าหา feature ไม่ใช่ feature ไหลเข้าหากลาง)

## Type ฝั่ง TypeScript ที่ต้อง mirror จาก Go struct

**อัปเดต**: Open Question เรื่อง JSON casing (ด้านล่าง) ตัดสินใจแล้วว่า **normalize ทั้ง API เป็น snake_case สม่ำเสมอกันหมด** — เพิ่ม json tag ให้ `job.Job`/`logger.Line` แล้ว (เดิมไม่มี tag เลย ทำให้ Go เข้ารหัสเป็น PascalCase ตาม field name ตรงๆ) ตอนนี้ทั้ง API (ทั้งกลุ่ม `/jobs` และ `/workflow-runs`) เป็น snake_case เหมือนกันหมดแล้ว ไม่มีจุดไหนผสม casing อีกต่อไป

Type ที่ implement จริง (ตรงกับโค้ด ณ ตอนเขียนเอกสารนี้ — เช็ค commit ล่าสุดของ backend ก่อนเชื่อ 100% เสมอ):

```typescript
// features/jobs/types/index.ts
// mirror จาก internal/job/job.go (Job struct)
export type JobStatus = "pending" | "running" | "succeeded" | "failed" | "cancelled";

export interface Job {
  id: string;
  bot_name: string;
  config: Record<string, string>;
  status: JobStatus;
  summary?: string;
  error?: string;
  created_at: string; // ISO timestamp string
  started_at?: string;
  ended_at?: string;
}

// mirror จาก internal/logger/logger.go (Line struct)
export type LogLevel = "info" | "warn" | "error";

export interface LogLine {
  job_id: string;
  time: string;
  level: LogLevel;
  message: string;
}

// features/workflow-runs/types/index.ts
// mirror จาก internal/api/http/handler.go (workflowRunSummary/workflowRunDetail)
// และ internal/workflow/workflow.go (StepResult) — snake_case เหมือนกับทุก type อื่น
export interface WorkflowRunSummary {
  workflow_id: string;
  run_id: string;
  workflow_name: string;
  status: string; // ค่าจริงคือ Temporal enum string เช่น "WORKFLOW_EXECUTION_STATUS_RUNNING"
                   // — แนะนำ map เป็น label ที่อ่านง่ายกว่านี้ในชั้น UI ไม่ใช่โชว์ string ดิบ
  started_at: string;
  ended_at?: string;
}

export interface StepResult {
  step_index: number;
  bot_name: string;
  summary?: string; // มีค่าก็ต่อเมื่อ step นี้สำเร็จ
  error?: string;   // มีค่าก็ต่อเมื่อ step นี้ล้มเหลว
}

export interface WorkflowRunDetail extends WorkflowRunSummary {
  steps: StepResult[];
}
```

## TODO Checklist (ผูกกับโครงสร้างด้านบน — เลขข้อตรงกับ roadmap) — ครบทุกข้อแล้ว

### 1.1 พื้นฐานก่อนแตะ React
- [x] ตั้งโปรเจกต์: `create-next-app@15` (App Router, TypeScript, Tailwind v4, ESLint, `src/`) — ใช้ Next 15 ไม่ใช่ 16 ล่าสุด เพราะ 16 ต้องการ Node >=20.9 ตรงๆ ตอนที่ยังใช้ Node 19 อยู่ (ภายหลังอัปเกรดเป็น Node 22 แล้ว แต่ไม่ได้ bump Next เป็น 16 ตาม เพราะไม่มีเหตุผลจำเป็นต้องใช้ฟีเจอร์ใหม่)
- [x] เขียน type ทั้งหมดใน `features/*/types/index.ts` + `src/types/api.ts` (ApiError กลาง)

### 1.2 React Core ผ่านหน้า Bot List
- [x] `features/bots/api/use-bots.ts`, `features/bots/components/{bot-card,bot-list}.tsx`
- [x] Empty state, list rendering พร้อม `key={bot.name}`, search filter ด้วย `useState`

### 1.3 Dynamic Form
- [x] `features/bots/schemas/bot-config.schema.ts` — `buildConfigSchema(fields)` สร้าง zod schema แบบ dynamic ตาม `config_schema` จาก `GET /bots`
- [x] `features/bots/components/bot-form.tsx` — react-hook-form + zodResolver, field เปลี่ยนตาม bot ที่เลือก, `key={bot.name}` บังคับ remount ทั้ง form ตอนเปลี่ยน bot (กัน form state เก่าปนกับ schema ใหม่)

### 1.4 หน้า Job History
- [x] `features/jobs/api/use-jobs.ts` (TanStack Query) เรียก `GET /jobs`
- [x] `features/jobs/components/{job-table,job-filter}.tsx` — filter (bot/status) sync กับ URL query param ผ่าน `useSearchParams`, pagination ฝั่ง client (`useState`)

### 1.5 Live Job Monitor
- [x] Backend: `GET /jobs/{id}/logs/stream` (SSE) — ดู Backend Prerequisite ข้อ 4 ด้านบน (งานเบากว่าที่วางแผนไว้ เพราะ Logger persist+broadcast มีอยู่แล้ว)
- [x] `src/lib/sse-client.ts` — helper เปิด `EventSource` กลาง พร้อม `SSEStatus` (connecting/connected/disconnected)
- [x] `features/jobs/api/use-job-log-stream.ts` — **ต่างจากแผนเดิม**: ไม่แยกยิง REST backfill ต่างหากอีกแล้ว เพราะ SSE endpoint ส่ง backfill มาเป็นส่วนหนึ่งของ stream เดียวกันเอง (ดู comment เต็มในไฟล์) เปิด SSE อย่างเดียวก็ได้ทั้ง backfill และ live update ครบ
- [x] `features/jobs/components/job-log-viewer.tsx` — auto-scroll (`scrollIntoView`), badge สถานะ connecting/connected/disconnected คู่กับ `JobStatusBadge`

### 1.6 Next.js เฉพาะทาง
- [x] `app/(dashboard)/layout.tsx` — nav bar ร่วม (`NavBar`)
- [x] แยก Server/Client Component: `page.tsx` ทุกไฟล์เป็น Server Component บาง (แค่ import feature component), ส่วน interactive (form, SSE, useSearchParams) เป็น Client Component — `jobs/page.tsx` ต้องห่อ `<Suspense>` รอบ component ที่ใช้ `useSearchParams` ด้วย ไม่งั้น Next.js 15 build error
- [x] `.env.example` (ไม่ใช่ `.env.local.example` ตามที่ร่างไว้ตอนแรก — เปลี่ยนชื่อให้ตรงกับ pattern ที่ root `.gitignore` ทั้ง workspace whitelist ไว้แล้ว ไม่งั้นไฟล์นี้จะโดน `.gitignore` กลืนหายไปเงียบๆ)

### 1.7 Styling
- [x] `components/ui/status-badge.tsx` (generic, ไม่รู้จัก feature ไหนเลย) + `features/jobs/components/job-status-badge.tsx` และ `features/workflow-runs/components/workflow-status-badge.tsx` (map status ของแต่ละ feature เป็น tone/label แล้วส่งเข้า StatusBadge กลาง)

### 1.8 Authentication
- [x] Backend: `internal/auth/jwt.go` + `POST /login`/`POST /logout`/`GET /me`
- [x] `features/auth/` — login form (react-hook-form + zod), `useAuth`/`useMe`/`useLogin`/`useLogout` hooks
- [x] `src/middleware.ts` — protected route แบบ cookie-presence check (ไม่ verify signature ที่ edge — ดู comment เต็มในไฟล์ว่าทำไม และ security boundary ตัวจริงอยู่ที่ไหน)

### 1.9 Testing + Deploy
- [x] Vitest + React Testing Library — `bot-config.schema.test.ts` (validation logic), `job-status-badge.test.tsx` (สีถูกต้องตาม status ทุกค่า)
- [ ] **ยังไม่ทำ**: Deploy ขึ้น Vercel จริง — ต้องใช้ account ของเจ้าของโปรเจกต์ ไม่ใช่สิ่งที่ session implement ทำได้เอง

## Open Questions — ตัดสินใจแล้วทั้งหมด (เก็บไว้เป็น decision log)

1. **Config schema ของ bot** — **backend-declared**: เพิ่ม `ConfigField`/`ConfigFieldType` + method `Bot.ConfigSchema() []ConfigField` เข้า interface ตรงๆ (ไม่ใช่ static mapping ฝั่ง frontend) เพราะสอดคล้องกับหลักการ "pluggable, ไม่ special-case ฝั่ง frontend ต่อ bot" ที่ยึดมาตลอดทั้งโปรเจกต์ และไม่มีความเสี่ยง frontend/backend drift แบบที่ static mapping จะมี
2. **JSON casing normalization** — **ทำ**: เพิ่ม json tag ให้ `job.Job`/`logger.Line` เป็น snake_case ครบแล้ว ทั้ง API สม่ำเสมอกันหมด ไม่มีจุดผสม casing อีกต่อไป (README ของ backend อัปเดต curl example ตามแล้ว)
3. **Auth**: **custom JWT ล้วน** ตามทางหลักของ roadmap — ยังไม่ทำ NextAuth.js/OAuth เพราะยังไม่มี use case จริงที่ต้องการ (YAGNI) เพิ่มทีหลังได้ถ้าจำเป็น
4. **Mock ตอน test**: ไม่ได้ใช้ MSW — test ที่เขียนจริง (`bot-config.schema.test.ts`, `job-status-badge.test.tsx`) เป็น pure unit/component test ที่ไม่ต้อง mock network เลย (test logic/render ล้วนๆ) ยังไม่มี test ที่ต้อง mock API call จริงจัง ถ้าเพิ่ม integration test ทีหลัง (เช่น test `bot-list.tsx` เต็มหน้า) ค่อยกลับมาตัดสินใจ MSW vs `vi.mock` ตอนนั้น

## Reference

- [FULLSTACK-INFRA-ROADMAP.md](../../FULLSTACK-INFRA-ROADMAP.md) — roadmap เต็มของ BOP (Phase 1 อยู่บรรทัด 48-119)
- [projects/bot-orchestration-platform/README.md](../bot-orchestration-platform/README.md) — วิธีรัน backend คู่กับ dashboard นี้
- [projects/bot-orchestration-platform/internal/api/http/handler.go](../bot-orchestration-platform/internal/api/http/handler.go) — endpoint contract ตัวจริง (รวม `handleListBots`, `handleStreamJobLogs`, `handleLogin`/`handleLogout`/`handleMe`)
- [projects/bot-orchestration-platform/internal/auth/jwt.go](../bot-orchestration-platform/internal/auth/jwt.go) — JWT implementation เต็ม
- [books/09-authentication/](../../books/09-authentication/README.md) — JWT/OAuth
- [books/06-redis/02-pubsub-and-streams.md](../../books/06-redis/02-pubsub-and-streams.md) — บริบทตอน scale SSE เป็นหลาย replica (Redis Pub/Sub)
- Bulletproof React (feature-based structure pattern) — แนวทางที่ใช้เป็นต้นแบบของโครงสร้างในเอกสารนี้
- โค้ดจริงหลัง implement — `README.md` ของ `bop-dashboard` เอง สรุปวิธีรัน/โครงสร้างสั้นๆ, `.nvmrc` (Node 22)
