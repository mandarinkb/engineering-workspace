# 04: gateway ใน compose ชุด dev ของโปรเจกต์

Spec: [temporal-ui-gateway.md](../temporal-ui-gateway.md) · ที่มา: code review เฟส 5

**What to build:** compose ที่ `projects/bot-orchestration-platform/docker-compose.yml` ยังเปิด Temporal UI ที่ 8233 ตรงโดยไม่ต้อง login ให้ผู้ที่เปิดชุด dev นั้นเข้า Temporal UI ได้เฉพาะผ่าน gateway เหมือน compose ที่ root (หรือถ้าตัดสินว่าไม่ใช้ชุดนั้นแล้ว ให้เอา port 8233 ออก)

**Blocked by:** 01 (ใช้ config เดียวกันจาก `docker/temporal-ui-gateway/`)

**Status:** done

**Verified:** 2026-11-25 บน Mini PC, branch `feat/gateway-in-dev-compose` (commit b0c7e51) รัน compose ชุด dev ทั้งชุดเป็น project แยก `t04dev` (ย้ายพอร์ตเป็น 17233/18233 ด้วย override ชั่วคราว): container ทั้ง 4 ตัวขึ้น; `:18233` ไม่มี cookie ได้ **302** ไป `/login`; login แล้ว `/` และ `/api/v1/namespaces` ได้ 200 และได้ JSON จริง ของ Temporal ในชุด dev; Playwright ใน Chrome 4 ข้อผ่าน (ไม่ login ถูกพาไป login, login แล้วเข้าได้, title "Workflows | temporal-system", `/api/` 8 คำขอ 2xx) ลบ project, volume และไดเรกทอรีชั่วคราวแล้ว **ยังไม่ได้ทดสอบบน Docker Desktop (Mac)** และผู้ทดสอบใช้ `bop-api` บน Mini PC เป็นโฮสต์แทน `go run` บนเครื่องตัวเอง

- [x] ตัดสินวิธี: เพิ่ม gateway ในชุด dev (ใช้ config เดียวกับ root; `extra_hosts: bop-api:host-gateway` ให้ชื่อ `bop-api` ใน config ชี้ไปที่ `go run ./cmd/bop` บนโฮสต์)
- [x] README ของ bop ตรงกับพฤติกรรมจริงของชุด dev
- [x] **ผู้ดำเนินการ:** เปิดชุด dev แล้วเปิด `:8233` ไม่ login ไม่เห็น Temporal UI — ผลจริงบนพอร์ต 18233 (ไม่ใช่ 8233 เพราะชน stack จริง)
