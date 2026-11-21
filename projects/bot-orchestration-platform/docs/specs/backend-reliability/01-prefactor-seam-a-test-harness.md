# 01: Prefactor: test harness ของ seam A

Spec: [backend-reliability.md](../backend-reliability.md)

**What to build:** นักพัฒนารัน HTTP test ของ bop-api กับ Temporal dev server และ PostgreSQL จริง (ใน container บน Mini PC เพราะ Mac ไม่มี Docker) ได้ด้วยคำสั่งเดียว เป็นฐานให้ ticket ที่ต้องพิสูจน์พฤติกรรมกับ Temporal/PostgreSQL จริง — HTTP test เดิมสร้าง server ด้วย Temporal client เป็น nil จึงทดสอบส่วนนี้ไม่ได้

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] มีคำสั่งเดียวที่ยก Temporal dev server + PostgreSQL ชั่วคราว รัน test ชุด seam A แล้วเก็บกวาดทุกอย่างเมื่อจบ ไม่ทิ้ง container/volume/network ค้าง
- [ ] ไม่แตะ container, volume หรือ compose project เดิมบน Mini PC
- [ ] มี test ตัวอย่าง 1 ตัวที่ผ่าน: สร้าง Schedule ผ่าน HTTP API แล้วอ่านรายละเอียดกลับมาได้
- [ ] test ชุดเดิม (`go test ./...` ที่ไม่ต้องใช้ infra) ยังรันได้แยกโดยไม่ต้องมี container
- [ ] README อธิบายวิธีรัน
- [ ] golangci-lint ผ่าน
