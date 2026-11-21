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

## หลักการ

- รันตามลำดับโฟลเดอร์ request หลังใช้ค่าที่ request ก่อนหน้าเก็บไว้ (`session`, `jobId`, `workflowId`, `scheduleId`)
- `01-auth/02 Login` เก็บ cookie `bop_session` ไว้ แล้ว header ของ collection ส่งต่อให้ทุก request
- `06-require-session` ทับ header `Cookie` เป็นค่าว่างทีละ request เพื่อเช็คว่าทุก endpoint ข้อมูลตอบ 401 เมื่อไม่มี session — endpoint ข้อมูลใหม่ให้เพิ่ม request ในโฟลเดอร์นี้ด้วย
- ข้อมูลที่สร้างขึ้นต้นด้วย `bruno-` Schedule ถูกลบเองตอนจบ ส่วน Job และ Workflow Run ลบผ่าน API ไม่ได้ จึงสะสมบน stack ที่ใช้รัน
- request ที่ต้องรอให้งาน async จบใช้ `bru.sleep()` ใน pre-request script
- test เขียนตามพฤติกรรมจริงของ API ตอนนี้ — ถ้าตั้งใจเปลี่ยนพฤติกรรม ให้แก้ test ในรอบเดียวกัน
