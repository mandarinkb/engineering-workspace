# BOP Dashboard Architecture

เอกสารนี้อธิบายโครงสร้างของ BOP Dashboard (Next.js 15 App Router) ซึ่งจัดตาม **Feature-Sliced Design (FSD)** ว่าแต่ละชั้นมีหน้าที่อะไร import อะไรได้บ้าง และโค้ดใหม่ควรไปอยู่ตรงไหน

## ภาพรวม

FSD แบ่งโค้ดเป็น **ชั้น (layer)** เรียงจากบนลงล่าง แต่ละชั้นแบ่งเป็น **slice** ตามเรื่องทางธุรกิจ และแต่ละ slice แบ่งเป็น **segment** ตามหน้าที่ทางเทคนิค

```
app        ← providers ระดับแอป
pages      ← 1 หน้า = 1 slice
widgets    ← บล็อก UI ใหญ่ที่ประกอบหลายชั้น
features   ← action ของ user ที่มีผลทางธุรกิจ
entities   ← business entity (ข้อมูล + การดึงข้อมูล + การแสดงผลพื้นฐาน)
shared     ← ของกลางที่ไม่รู้จัก business เลย
```

**กฎ 3 ข้อ**
1. **import ลงล่างได้อย่างเดียว:** ชั้นบนใช้ชั้นล่างได้ ชั้นล่างห้าม import ชั้นบน
2. **slice ในชั้นเดียวกันห้าม import กันเอง:** เช่น `features/create-job` ห้าม import `features/cancel-job` ยกเว้นชั้น `shared` ซึ่งไม่มี slice
3. **นอก slice ต้อง import ผ่าน public API:** ใช้ `@/entities/job` ห้าม `@/entities/job/api/use-jobs` ภายใน slice เดียวกันให้ใช้ relative import (`../model/types`)

ESLint บังคับทั้ง 3 ข้อผ่าน `no-restricted-imports` ใน `eslint.config.mjs` ซึ่งเป็นกฎที่มีใน ESLint อยู่แล้ว ไม่ต้องติดตั้ง plugin เพิ่ม

## โครงสร้างโฟลเดอร์

```
app/                          Next.js App Router (routing อย่างเดียว)
  layout.tsx                  root layout ห่อ <Providers> จาก src/app
  page.tsx                    redirect "/" ไป /jobs
  globals.css, favicon.ico
  (auth)/login/page.tsx       export { LoginPage as default } from "@/pages/login"
  (dashboard)/layout.tsx      ใส่ NavBar จาก @/widgets/nav-bar
  (dashboard)/**/page.tsx     re-export จาก src/pages ทุกไฟล์
middleware.ts                 protected route (UX shortcut — security ตัวจริงอยู่ที่ backend)
pages/README.md               โฟลเดอร์ว่างที่ตั้งใจไว้ (ดูหัวข้อ "ทำไม routing อยู่ที่ app/ ระดับ root")

src/
  app/providers/              Providers: React Query, theme, toast, WatchedJobsProvider

  pages/
    login, bots, jobs, job-detail, schedules, schedule-new,
    schedule-detail, schedule-edit, workflow-runs, workflow-run-detail

  widgets/
    nav-bar                   เมนู + user ที่ login + logout + theme
    bot-list                  รายการ bot + ฟอร์มสั่งรัน job
    job-table, job-stats      ตาราง job + การ์ดสรุป
    job-log-viewer            log แบบ real-time (SSE) + เฝ้าดูสถานะ job
    schedule-table, schedule-detail, schedule-edit
    workflow-run-table, workflow-run-detail

  features/
    auth                      login, logout
    create-job                ฟอร์มสั่งรัน bot (BotForm, useCreateJob)
    cancel-job                ยกเลิก job ที่กำลังรัน
    filter-jobs               filter ตาราง job ผ่าน URL query param
    watch-job                 แจ้งเตือนเมื่อ job ที่เปิดดูเปลี่ยนสถานะ
    schedule-form             สร้าง/แก้ schedule
    manage-schedule           pause, unpause, trigger, backfill, delete

  entities/
    bot                       Bot, ConfigField, useBots, BotCard, buildConfigSchema
    job                       Job, LogLine, useJobs, useJob, useJobLogStream, JobStatusBadge,
                              computeJobStats, filterLogLines
    schedule                  ScheduleSummary/Detail, useSchedules, useSchedule, ScheduleStatusBadge
    workflow-run              WorkflowRunSummary/Detail, useWorkflowRuns, useWorkflowRun,
                              StepPipelineView, StepResultList, WorkflowStatusBadge
    user                      User, useMe, useAuth

  shared/
    api/                      apiClient, connectSSE, createQueryClient, ApiError
    ui/                       StatusBadge, StatCard, Toast, ToastProvider
    lib/theme/                ThemeProvider, useTheme
```

## Segment ภายใน slice

| Segment | ใส่อะไร |
|---|---|
| `ui/` | React component |
| `api/` | React Query hook และการเรียก backend (ผ่าน `@/shared/api` เสมอ) |
| `model/` | type, zod schema, context/provider, hook ที่จัดการ state |
| `lib/` | pure function ของ slice นั้น (เขียน unit test ได้ง่าย) |
| `index.ts` | public API ของ slice — re-export สิ่งที่ slice อื่นใช้ได้ |

test วางข้างไฟล์ที่ทดสอบ (`*.test.ts`) และ `index.ts` ไม่ export ไฟล์ test

## ตารางการ import

| จาก \ ไป | app | pages | widgets | features | entities | shared |
|---|:-:|:-:|:-:|:-:|:-:|:-:|
| `app/` (Next routing) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `src/app` | — | ✅ | ✅ | ✅ | ✅ | ✅ |
| pages | ❌ | ❌ ข้าม slice | ✅ | ✅ | ✅ | ✅ |
| widgets | ❌ | ❌ | ❌ ข้าม slice | ✅ | ✅ | ✅ |
| features | ❌ | ❌ | ❌ | ❌ ข้าม slice | ✅ | ✅ |
| entities | ❌ | ❌ | ❌ | ❌ | ❌ ข้าม slice | ✅ |
| shared | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ |

## ทำไม routing อยู่ที่ `app/` ระดับ root

ชั้น `app` และ `pages` ของ FSD ชื่อซ้ำกับโฟลเดอร์พิเศษของ Next.js จึงใช้วิธีที่เอกสาร FSD แนะนำสำหรับ Next.js:

- **`app/` ที่ root คือ App Router ของ Next.js:** เมื่อมีโฟลเดอร์นี้ Next.js จะไม่สแกน `src/app` และ `src/pages` เป็น route
- **`pages/README.md` ที่ root คือโฟลเดอร์ว่างที่ตั้งใจไว้:** กันไม่ให้ Next.js ตีความ `src/pages` เป็น Pages Router ห้ามลบ
- **ไฟล์ `page.tsx` ใน `app/` ทำแค่ re-export:** เช่น `export { JobsPage as default } from "@/pages/jobs"` เนื้อหาจริงของหน้าอยู่ที่ `src/pages`
- **`layout.tsx` และ `middleware.ts` อยู่กับ routing:** เพราะเป็นไฟล์ convention ของ Next.js

## ตัวอย่าง: หน้า `/jobs`

1. `app/(dashboard)/jobs/page.tsx` re-export `JobsPage` จาก `@/pages/jobs`
2. `pages/jobs` ประกอบ `JobStats`, `JobFilter` และ `JobTable`
3. `widgets/job-table` ใช้ `useJobs` และ `JobStatusBadge` จาก `entities/job` และใช้ `useCancelJob` จาก `features/cancel-job`
4. `features/filter-jobs` เขียน filter ลง URL query param ส่วน `widgets/job-table` อ่านจาก URL เดียวกัน สองตัวนี้จึงไม่ต้อง import กันเอง
5. `entities/job` เรียก backend ผ่าน `apiClient` จาก `shared/api`

## เพิ่มของใหม่ต้องไปไว้ที่ไหน

| อยากเพิ่ม | ที่อยู่ |
|---|---|
| หน้าใหม่ | slice ใหม่ใน `src/pages/<name>` + `app/<route>/page.tsx` ที่ re-export |
| type หรือ API ของข้อมูลชนิดใหม่จาก backend | `src/entities/<name>` (`model/types.ts`, `api/use-*.ts`) |
| ปุ่มหรือฟอร์มที่ user กดแล้วเปลี่ยนข้อมูล | `src/features/<verb-noun>` เช่น `approve-run` |
| ส่วนของหน้าที่ประกอบ entity + feature หลายตัว | `src/widgets/<name>` |
| component UI ที่ไม่รู้จัก business | `src/shared/ui` |
| การตั้งค่าการเรียก backend หรือ util ทั่วไป | `src/shared/api` หรือ `src/shared/lib/<name>` |
| provider ระดับแอป | `src/app/providers` |

**ถ้าสอง feature ต้องใช้ของเดียวกัน:** ย้ายของนั้นลงไปชั้น `entities` หรือ `shared` ไม่ใช่ให้ feature import กันเอง ถ้าสอง entity ต้องอ้างถึงกัน (เช่น schedule อ้าง bot) ให้ใช้ type ในชั้นบนประกอบ หรือใช้รูปแบบ `@x` ของ FSD เฉพาะเมื่อจำเป็นจริง

## สิ่งที่ห้ามเปลี่ยนโดยไม่แก้ backend

- **type ใน `entities/*/model/types.ts` ต้องตรงกับ JSON ของ BOP API:** ใช้ snake_case ตาม backend ดูข้อห้ามฝั่ง backend ที่ `projects/bot-orchestration-platform/ARCHITECTURE.md`
- **query key ของ React Query:** เช่น `["jobs"]` ถูกใช้ร่วมกันหลาย widget เพื่อให้ dedupe request และ invalidate cache ได้ถูกที่

## ประวัติ

- เดิมใช้ feature-based (`src/features/{auth,bots,jobs,schedules,workflow-runs}` + `src/components`, `src/lib`, `src/types`) โดยมีข้อยกเว้น `schedules → bots` และ `NavBar → features/auth`
- ย้ายมาเป็น FSD บน branch `refactor/clean-architecture` (2026-10-10) ด้วย `git mv` จึงตามประวัติเดิมได้ด้วย `git log --follow` และไม่มีข้อยกเว้นเรื่องการ import อีกต่อไป
- path ใน `PHASE1-DASHBOARD-TODO.md` และ `PHASE3-DASHBOARD-TODO.md` เป็นโครงสร้างก่อนย้าย เพราะเป็นบันทึกตามเวลาที่เขียน
