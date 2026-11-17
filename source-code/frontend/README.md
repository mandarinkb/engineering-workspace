# Source Code: Frontend

โค้ดฝึกตาม [Volume 18 — Fullstack](../../books/18-fullstack-frontend/README.md) ([แผนรายวัน](../../books/18-fullstack-frontend/00-daily-plan.md)) และหน้าจอ demo ประกอบ backend ใน [projects/](../../projects/)

## โครงสร้าง (สร้างทีละโฟลเดอร์ตามแผน)

```text
source-code/frontend/
├── NOTES.md          บันทึกสรุปท้ายแต่ละคืน (template อยู่ท้ายแผนรายวัน)
├── w01-resume/       สัปดาห์ 1 — HTML/CSS ล้วน
├── w01-js/           สัปดาห์ 1 — สคริปต์ JS ฝึก (รันด้วย node)
├── w02-async/        สัปดาห์ 2 — fetch, Promise
├── w02-vanilla/      สัปดาห์ 2 — Vite vanilla-ts
├── tasks-api/        สัปดาห์ 2 → 7 — Go backend ที่ใช้ตลอดหลักสูตร (in-memory → Postgres + auth)
├── react-tasks/      สัปดาห์ 3–5 — Vite + React + TanStack Query + react-hook-form
├── next-tasks/       สัปดาห์ 6–8 — Next.js (portfolio project)
└── docker-compose.yml  สัปดาห์ 7 — db + api + web
```

## Prerequisite

- Node.js 22+ (เครื่องนี้มี v26 แล้ว)
- Go (บน Mac) สำหรับ `tasks-api`
- Docker — Mac เครื่องนี้ไม่มี ใช้ Mini PC ตอนสัปดาห์ 7
