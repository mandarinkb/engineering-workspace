# 04: gateway ใน compose ชุด dev ของโปรเจกต์

Spec: [temporal-ui-gateway.md](../temporal-ui-gateway.md) · ที่มา: code review เฟส 5

**What to build:** compose ที่ `projects/bot-orchestration-platform/docker-compose.yml` ยังเปิด Temporal UI ที่ 8233 ตรงโดยไม่ต้อง login ให้ผู้ที่เปิดชุด dev นั้นเข้า Temporal UI ได้เฉพาะผ่าน gateway เหมือน compose ที่ root (หรือถ้าตัดสินว่าไม่ใช้ชุดนั้นแล้ว ให้เอา port 8233 ออก)

**Blocked by:** 01 (ใช้ config เดียวกันจาก `docker/temporal-ui-gateway/`)

**Status:** implemented (ผู้ใช้เลือกเพิ่ม gateway ในชุด dev 2026-11-25) ตรวจบางส่วน ดู Verified

**Verified (บางส่วน):** 2026-11-25 บน Mini PC: container APISIX ชั่วคราวใช้ config เดียวกับ root พร้อม `--add-host bop-api:host-gateway` ชื่อ `bop-api` แปลงเป็น `172.17.0.1` และคำขอไม่มี cookie ได้ **302** ไป `/login` (แปลว่าเรียก `/me` ของ bop-api บนโฮสต์ได้จริง) ลบ container แล้ว **ยังไม่ได้รัน compose ชุด dev ทั้งชุด** เกณฑ์ผู้ดำเนินการข้างล่างจึงยังไม่ติ๊ก

- [x] ตัดสินวิธี: เพิ่ม gateway ในชุด dev (ใช้ config เดียวกับ root; `extra_hosts: bop-api:host-gateway` ให้ชื่อ `bop-api` ใน config ชี้ไปที่ `go run ./cmd/bop` บนโฮสต์)
- [x] README ของ bop ตรงกับพฤติกรรมจริงของชุด dev
- [ ] **ผู้ดำเนินการ:** เปิดชุด dev แล้วเปิด `:8233` ไม่ login ไม่เห็น Temporal UI
