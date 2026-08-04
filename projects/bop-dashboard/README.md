# BOP Dashboard

Next.js dashboard ที่ควบคุม [Bot Orchestration Platform (BOP)](../bot-orchestration-platform/README.md) — implement ตาม [PHASE1-DASHBOARD-TODO.md](PHASE1-DASHBOARD-TODO.md) (roadmap Phase 1 ของ [FULLSTACK-INFRA-ROADMAP.md](../../FULLSTACK-INFRA-ROADMAP.md)) แยก repo/deploy ออกจาก backend (Go) โดยตั้งใจ ดูเหตุผลที่ PHASE1-DASHBOARD-TODO.md หัวข้อ "ตัดสินใจไว้แล้ว"

## Prerequisite

- **Node.js >= 22** (ดู `.nvmrc`) — เครื่องที่เขียนโค้ดนี้ตอนแรกมี Node v19.9.0 ซึ่งเก่าเกินไปสำหรับ Next.js 15/Vitest 4/Tailwind v4 (native binding ของ `@tailwindcss/oxide` และ `jsdom`/`undici` ต้องการ Node ใหม่กว่านี้จริงๆ ไม่ใช่แค่ policy) ถ้าเครื่องคุณมี Node เก่ากว่านี้ ติดตั้ง [nvm](https://github.com/nvm-sh/nvm) แล้วรัน `nvm install` (อ่านค่าจาก `.nvmrc` อัตโนมัติ) ก่อนเริ่ม
- Backend (`../bot-orchestration-platform`) ต้องรันอยู่คู่กันเสมอ — ดู [backend README](../bot-orchestration-platform/README.md) หัวข้อ "รันยังไง" (ต้องมีทั้ง `docker compose up -d`, `go run ./cmd/bop-worker`, และ `go run ./cmd/bop`)

## รันยังไง

```bash
npm install
cp .env.example .env.local   # ปรับ NEXT_PUBLIC_API_URL ถ้า backend ไม่ได้รันที่ localhost:8080

npm run dev          # dev server ที่ http://localhost:3000
npm run typecheck    # tsc --noEmit
npm run lint         # eslint
npm run test          # vitest run
npm run build         # production build (รัน type-check + lint ในตัวด้วย)
```

เข้า `http://localhost:3000` แล้วจะถูก redirect ไป `/login` อัตโนมัติถ้ายัง login ไม่อยู่ (ดู `src/middleware.ts`) — credential เริ่มต้นตอนนี้ hard-code ไว้ที่ฝั่ง backend คือ `admin` / `changeme` (ดู `internal/api/http/handler.go` — **ต้องเปลี่ยนก่อน deploy จริงเสมอ**)

## โครงสร้างโปรเจกต์

Feature-based structure ดูคำอธิบายเหตุผลและ diagram เต็มๆ ที่ [PHASE1-DASHBOARD-TODO.md](PHASE1-DASHBOARD-TODO.md#โครงสร้างโปรเจกต์ที่แนะนำ-feature-based) สรุปสั้นๆ:

```
src/
├── app/            routing/layout เท่านั้น (ไม่มี business logic)
├── features/       1 feature = 1 โฟลเดอร์ (bots, jobs, schedules, workflow-runs, auth)
├── components/ui   UI component กลาง ไม่ผูก feature ไหน (StatusBadge ฯลฯ)
├── components/layout  NavBar
├── lib/            api-client, query-client, sse-client (infrastructure กลาง)
├── types/          type กลางข้าม feature (ApiError)
└── middleware.ts   protected route (UX shortcut — security ตัวจริงอยู่ที่ backend)
```

**ข้อยกเว้นกติกา import ที่ตั้งใจ**: ปกติ `features/X` ห้าม import จาก `features/Y` ตรงๆ แต่ `features/schedules` import `useBots`/`ConfigField` จาก `features/bots` เพราะ "bot catalog" เป็นข้อมูลอ้างอิงที่ทั้งสอง feature ต้องใช้จริง (เลือก bot ต่อ step ของ schedule ก็ต้องรู้ config schema เหมือนตอนสร้าง job เดี่ยว) — เหตุผลเดียวกับที่ `NavBar` import จาก `features/auth` ดู comment เต็มที่ `schedule-form.tsx`

## Feature: Schedules (จัดการ cron job แบบ Control-M)

CRUD + manual trigger เต็มรูปแบบสำหรับ Temporal Schedule ผ่าน `features/schedules/` — สร้าง/แก้ไข/ลบ, pause/unpause, **Run Now** (trigger ทันที นอกตารางเวลา), backfill (rerun ย้อนหลัง) ดูรายละเอียด endpoint ที่ backend เรียกที่ [backend README](../bot-orchestration-platform/README.md) หัวข้อ "จัดการ Schedule" — ฟอร์มสร้าง/แก้ไขรองรับหลาย step ต่อ schedule (แต่ละ step เลือก bot + กรอก config แบบ dynamic เหมือน `bot-form.tsx`) และเลือกได้ทั้ง interval (ทุก N วินาที) หรือ cron expression (รองรับ "เฉพาะวันทำการ" ฯลฯ ผ่าน Temporal เอง)

## สถานะ (เทียบกับ checklist เต็มใน PHASE1-DASHBOARD-TODO.md)

ทำแล้ว: 1.1-1.9 ครบทุกข้อ (bot list + dynamic form, job history + filter + pagination, live job monitor ผ่าน SSE, workflow runs, auth + protected route, styling, Vitest + RTL test) บวก feature Schedules เพิ่มเติมนอก roadmap เดิม (ดูหัวข้อด้านบน) — `npm run typecheck`/`lint`/`test`/`build` ผ่านหมด และทดสอบ `npm run dev` จริงแล้วว่า redirect ไป `/login` ถูกต้องทุก route รวม `/schedules/*`

**ยังไม่ verify แบบเต็ม end-to-end กับ backend จริง** (สร้าง bot job จริง ดู live log ไหลจริงผ่าน SSE, ดู workflow run จริงจาก Temporal, สร้าง/trigger schedule จริง) เพราะ session ที่เขียนโค้ดนี้ติด Docker permission ในเครื่อง (`permission denied` ที่ `/var/run/docker.sock`) ทำให้รัน Temporal server ไม่ได้ — ต้องให้ผู้ใช้ทดสอบเองตามขั้นตอนใน backend README

**ยังไม่ทำ**: deploy ขึ้น Vercel จริง (ข้อ 1.9 ส่วนหลัง) เพราะต้องใช้ account/สิทธิ์ของเจ้าของโปรเจกต์
