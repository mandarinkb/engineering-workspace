# 14: Job ที่ถูกขัดจังหวะโดย restart ต้องไม่ค้างเป็น pending/running ตลอดไป

Spec: [backend-reliability.md](../backend-reliability.md)

**What to build:** หลัง `bop-api` restart หรือ crash Job ที่ยังเป็น `pending` หรือ `running` อยู่ถูกตั้งเป็น `failed` พร้อมเหตุผลที่อ่านออก (เช่น "interrupted: bop-api restarted") และ `ended_at` เพื่อให้ตัวเลข "Job กำลังรัน", "Job ล้มเหลววันนี้" และ success rate ของ dashboard ไม่ผิดไปตลอด

**ที่มา (ผล code review ของ ticket 01-12):** ตั้งแต่ ticket 06 Job ถูกเก็บใน PostgreSQL แต่ worker pool ที่รันงานอยู่ใน process เดียวกับ API ถ้า process ตายกลางทาง goroutine ที่รัน Job ไม่ได้เขียนสถานะสุดท้ายลงฐานข้อมูล (`pool.Shutdown` ครอบคลุมเฉพาะการหยุดแบบปกติ) แถวนั้นจึงค้างเป็น `running` ตลอดไป ก่อนมี persistence restart ทำให้ Job หายไปเลย จึงเป็นปัญหาใหม่ที่เกิดจากการเก็บถาวร

**ข้อตัดสินใจ: รัน `bop-api` ตัวเดียว (ทางเลือก a)**
- การกวาดตอน start จะตั้ง Job ที่ replica อีกตัวกำลังรันอยู่เป็น `failed` ผิดๆ ถ้ามีหลายตัว จึงใช้ได้เมื่อรันตัวเดียวเท่านั้น (ทางเลือก b คือเก็บ owner กับ lease ต่อ Job ซึ่งต้องเพิ่มคอลัมน์ มี heartbeat และควรเขียน ADR ยังไม่ทำ)
- **ผู้ตัดสินคือผู้ดำเนินการ ไม่ใช่เจ้าของระบบ:** เจ้าของระบบยังไม่ได้ตอบว่าจะรันกี่ตัว แต่สั่งให้ทำโค้ดที่ค้างให้เสร็จก่อน deploy ผู้ดำเนินการจึงเลือกทางเลือก (a) ซึ่งเป็นค่าที่เรียบง่ายและตรงกับที่รันอยู่จริง ถ้าเจ้าของระบบจะรันหลายตัว ให้ตั้ง `BOP_RECONCILE_INTERRUPTED_JOBS=false` เพื่อปิดการกวาด แล้วทำทางเลือก (b)
- สวิตช์ `BOP_RECONCILE_INTERRUPTED_JOBS`: ค่าเริ่มต้นเปิด ตั้งเป็น `false` เพื่อปิด
- ข้อจำกัด "รัน `bop-api` ตัวเดียว" ต้องเขียนไว้ใน ARCHITECTURE.md และ README

**Blocked by:** None (can start immediately)

**Status:** implemented

- [x] ตอน start ก่อนรับ request ตั้ง Job ที่เป็น `pending` หรือ `running` เป็น `failed` พร้อม `error` ที่บอกว่าถูกขัดจังหวะโดย restart และมี `ended_at` พิมพ์ log บอกจำนวนที่ตั้ง
- [x] Job ที่จบแล้ว (`succeeded`, `failed`, `cancelled`) และ Job ที่สร้างหลัง start ไม่ถูกแตะ
- [x] `BOP_RECONCILE_INTERRUPTED_JOBS=false` ปิดการกวาด; ไม่ตั้ง = เปิด
- [x] repository แบบ memory ไม่ทำอะไร (ไม่มี Job ค้างข้าม restart อยู่แล้ว)
- [x] Go test กับ PostgreSQL จริง: ใส่ Job หลายสถานะ รันการกวาด แล้วตรวจแต่ละแถว (สถานะ, error, ended_at) และตรวจกรณีสวิตช์ปิด
- [x] ARCHITECTURE.md และ README ระบุข้อจำกัด "รัน bop-api ตัวเดียว" และสวิตช์
- [ ] พิสูจน์กับ stack จริงใน `bruno/run-with-restart.sh`: เริ่ม Job ที่ใช้เวลานาน ยืนยันว่ามันเป็น `running` ก่อน recreate `bop-api` แล้วตรวจว่าเป็น `failed` พร้อมเหตุผลหลังกลับมา ถ้าไม่ได้เป็น `running` ตอนนั้นต้องรายงานว่าข้ามการตรวจ (ห้ามผ่านเงียบๆ) ถ้าทำอัตโนมัติไม่ได้ ให้เขียนเหตุผลและขั้นตอนทำมือ
- [x] golangci-lint ผ่าน
- [ ] orchestrator deploy branch แล้ว `bru run --env minipc` และ `run-with-restart.sh` ผ่านทั้งชุด
