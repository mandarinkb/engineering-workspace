# BOP API — Bruno collection

ทดสอบ HTTP API ของ bop-api แบบ black-box กับ stack ที่รันอยู่จริง (seam A ใน [spec](../docs/specs/backend-reliability.md))

## ติดตั้ง

- แอป: `brew install --cask bruno` แล้วเปิดโฟลเดอร์นี้ด้วย Open Collection
- CLI: `npm install -g @usebruno/cli@4.3.0`

## รัน

```bash
cd projects/bot-orchestration-platform/bruno
bru run --env minipc     # stack บน Mini PC (http://192.168.1.46:8080)
bru run --env local      # stack บนเครื่องตัวเอง (http://localhost:8080)
bru run 05-schedules --env minipc   # เฉพาะโฟลเดอร์เดียว (ต้อง login ก่อน: รัน 01-auth คู่กัน)
```

### ทดสอบว่า Job อยู่รอดหลัง restart

Bruno สั่ง restart container เองไม่ได้ `run-with-restart.sh` จึงรันชุดหลัก → recreate bop-api ด้วยคำสั่งใน `RESTART_CMD` → รันโฟลเดอร์ `after-restart` โดยส่ง `runMarker` เดียวกันให้ทั้งสองรอบ (`after-restart` หา Job จากรอบก่อนด้วย `bruno-run=<runMarker>` ใน url ของ config)

```bash
RESTART_CMD="ssh minipc 'cd <โฟลเดอร์ที่ deploy> && /usr/libexec/docker/cli-plugins/docker-compose up -d --force-recreate --no-deps bop-api'" \
  ./run-with-restart.sh minipc
```

`bru run` ทั้งชุดตามปกติก็รัน `after-restart` ด้วย (ต่อท้ายชุดหลักโดยไม่มี restart คั่น) และผ่านเหมือนกัน — การพิสูจน์ persistence จริงต้องใช้สคริปต์นี้

## หลักการ

- รันตามลำดับโฟลเดอร์ request หลังใช้ค่าที่ request ก่อนหน้าเก็บไว้ (`session`, `jobId`, `workflowId`, `scheduleId`)
- `01-auth/02 Login` เก็บ cookie `bop_session` ไว้ แล้ว header ของ collection ส่งต่อให้ทุก request
- `06-require-session` ทับ header `Cookie` เป็นค่าว่างทีละ request เพื่อเช็คว่าทุก endpoint ข้อมูลตอบ 401 เมื่อไม่มี session — endpoint ข้อมูลใหม่ให้เพิ่ม request ในโฟลเดอร์นี้ด้วย
- ข้อมูลที่สร้างขึ้นต้นด้วย `bruno-` Schedule ถูกลบเองตอนจบ ส่วน Job และ Workflow Run ลบผ่าน API ไม่ได้ จึงสะสมบน stack ที่ใช้รัน
- request ที่ต้องรอให้งาน async จบใช้ `bru.sleep()` ใน pre-request script
- test เขียนตามพฤติกรรมจริงของ API ตอนนี้ — ถ้าตั้งใจเปลี่ยนพฤติกรรม ให้แก้ test ในรอบเดียวกัน
