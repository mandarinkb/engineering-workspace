# 05: ปิดช่อง gRPC 7233 ที่ข้าม gateway

Spec: [temporal-ui-gateway.md](../temporal-ui-gateway.md) · ADR: [0002](../../adr/0002-temporal-ui-behind-apisix-gateway.md) · ที่มา: Out of Scope ของ spec และการคุยเรื่อง prod

**What to build:** ตอนนี้ `docker-compose.yml` (root และชุด dev) publish พอร์ต `7233:7233` ของ Temporal server ให้ทุกเครื่องในเครือข่าย ใครต่อ gRPC ได้ก็สั่ง start/terminate/delete workflow ได้โดยไม่ผ่าน Temporal UI gateway เลย ทำให้การปิด Temporal UI ด้วย session ของ BOP ไร้ผลในทางปฏิบัติ งานนี้ทำให้ gRPC ของ Temporal เข้าถึงได้เฉพาะ client ที่ตั้งใจให้เข้า

**ข้อเท็จจริงจากโค้ด (ตรวจแล้ว)**

- `bop-api` และ `bop-worker` ใน compose ที่ root ต่อ `temporal:7233` ผ่านเครือข่ายภายใน (`TEMPORAL_HOST_PORT: temporal:7233`) **ไม่ต้องพึ่งพอร์ตที่ publish**
- ชุด dev: `go run ./cmd/bop` และ `go run ./cmd/bop-worker` บนโฮสต์ต่อ `localhost:7233` (ค่า default ใน `cmd/*/main.go`) จึงต้องการพอร์ตนี้บนโฮสต์ แต่เฉพาะจาก `localhost`
- `projects/remote-job-worker` ต่อ `localhost:7233` (hardcode) และ `REMOTE-JOB-WORKER-TODO.md` ออกแบบให้ worker รันที่อื่นแล้วต่อมาที่ Temporal server จริง ถ้าเป็นเครื่องอื่นในเครือข่าย การปิดพอร์ตจะทำให้มันต่อไม่ได้
- ยังไม่ทราบว่าโปรเจกต์อื่นบน Mini PC (เช่น `go-temporal`) ต่อ `192.168.1.46:7233` อยู่หรือไม่ (ตรวจ `ss` ตอนนี้ไม่เห็นการเชื่อมต่อ แต่ไม่ใช่หลักฐานว่าไม่มีใช้)
- Temporal server ไม่มีการยืนยันตัวตนเลยในค่าที่ตั้งอยู่ (ไม่มี mTLS, ไม่มี authorizer)

**ทางเลือก (ต้องให้ผู้ใช้ตัดสิน)**

| ทาง | ทำอย่างไร | ผลดี | ผลเสีย |
|---|---|---|---|
| A (แนะนำเป็นก้าวแรก) | publish เป็น `127.0.0.1:7233:7233` ในทั้งสอง compose | ปิดการเข้าถึงจากเครื่องอื่นทันที `go run` บนโฮสต์ยังใช้ได้ ไม่แก้โค้ด | remote worker บนเครื่องอื่นต่อไม่ได้ |
| B | เลิก publish 7233 เลย | ปิดสนิทที่สุด | ชุด dev ที่ `go run` บนโฮสต์พัง ต้องเปลี่ยนวิธีรัน |
| C | เปิด gRPC ผ่าน APISIX (route gRPC + `key-auth` หรือ `jwt-auth`) | ได้ gateway ตัวเดียวคุมทั้ง UI และ gRPC และได้เรียน plugin ยืนยันตัวตน | Temporal Go SDK ต้องส่ง header ยืนยันตัวตน ต้องแก้ client (`bop-worker`, `remote-job-worker`) และต้องพิสูจน์ว่า gRPC ผ่าน APISIX ได้ |
| D | เปิดเฉพาะ IP ที่อนุญาตด้วย firewall (ต้องใช้ `sudo` ซึ่งต้องขอรหัสผ่านจากผู้ใช้) | ไม่แก้โค้ด | แก้ผ่าน host ไม่ใช่ใน repo, ดูแลยาก |

**Blocked by:** None · ต้องการคำตอบจากผู้ใช้ก่อน: ใช้ทางไหน และ `remote-job-worker` จะรันที่ไหน (เครื่องเดียวกัน หรือเครื่องอื่น)

**Status:** needs-decision

- [ ] ผู้ใช้เลือกทาง (A/B/C/D) และบอกที่รัน `remote-job-worker`
- [ ] ตรวจก่อนแก้ว่าไม่มี client อื่นใช้ `192.168.1.46:7233` อยู่ (เช่นสังเกต `ss` หลายช่วงเวลา หรือถามผู้ใช้เรื่อง `go-temporal`)
- [ ] แก้ compose ตามทางที่เลือก (ถ้าทาง A: `127.0.0.1:7233:7233` ทั้ง root และชุด dev พร้อมคอมเมนต์ไทยบอกเหตุผล) และอัปเดต README/`REMOTE-JOB-WORKER-TODO.md` ที่พูดถึง `localhost:7233`
- [ ] **ผู้ดำเนินการ:** deploy แล้วจากเครื่องอื่นในเครือข่าย (เช่น Mac) `nc -vz 192.168.1.46 7233` ต้องต่อไม่ได้ (ก่อนแก้ต่อได้) และบน Mini PC เอง `nc -vz 127.0.0.1 7233` ยังต่อได้ (ถ้าทาง A)
- [ ] **ผู้ดำเนินการ:** หลัง deploy `bru run --env minipc` ผ่านทั้งชุด และ log ของ `bop-worker` ไม่มี error ต่อ Temporal (Temporal UI ต่อ `temporal:7233` ภายใน ไม่กระทบ)
- [ ] **ผู้ดำเนินการ:** deploy นี้ recreate container `temporal` ซึ่งเป็นของผู้ใช้ (publish พอร์ตเปลี่ยน) ต้องขออนุมัติก่อนทำและประเมินผลต่อ workflow ที่กำลังรัน
