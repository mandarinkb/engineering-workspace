# 14: Job ที่ถูกขัดจังหวะโดย restart ต้องไม่ค้างเป็น pending/running ตลอดไป

Spec: [backend-reliability.md](../backend-reliability.md)

**What to build:** หลัง `bop-api` restart หรือ crash Job ที่ยังเป็น `pending` หรือ `running` อยู่ถูกตั้งเป็น `failed` พร้อมเหตุผลที่อ่านออก (เช่น "interrupted: bop-api restarted") และ `ended_at` เพื่อให้ตัวเลข "Job กำลังรัน", "Job ล้มเหลววันนี้" และ success rate ของ dashboard ไม่ผิดไปตลอด

**ที่มา (ผล code review ของ ticket 01-12):** ตั้งแต่ ticket 06 Job ถูกเก็บใน PostgreSQL แต่ worker pool ที่รันงานอยู่ใน process เดียวกับ API ถ้า process ตายกลางทาง goroutine ที่รัน Job ไม่ได้เขียนสถานะสุดท้ายลงฐานข้อมูล (`pool.Shutdown` ครอบคลุมเฉพาะการหยุดแบบปกติ) แถวนั้นจึงค้างเป็น `running` ตลอดไป ก่อนมี persistence restart ทำให้ Job หายไปเลย จึงเป็นปัญหาใหม่ที่เกิดจากการเก็บถาวร

**ข้อตัดสินใจที่ยังเปิดอยู่ (ต้องได้คำตอบก่อนลงมือ):** จะรัน `bop-api` มากกว่า 1 ตัวพร้อมกันไหม
- การกวาดตอน start แบบตรงไปตรงมาจะตั้ง Job ที่ replica อีกตัวกำลังรันอยู่เป็น `failed` ผิดๆ ถ้ามีหลายตัว
- (a) รันตัวเดียวเสมอ: กวาดตอน start และบันทึกข้อจำกัดนี้ไว้ในเอกสาร (ผู้เขียนแนะนำ เพราะตอนนี้มีตัวเดียว และเรียบง่ายที่สุด)
- (b) รองรับหลายตัว: เก็บ owner กับ lease (หมดอายุ) ต่อ Job แล้วให้แต่ละตัวตั้ง `failed` เฉพาะ Job ที่เจ้าของหายไป ต้องเพิ่มคอลัมน์ ต้องมี heartbeat และควรเขียน ADR
- ถ้าเลือก (a) ให้เขียนข้อจำกัด "รัน `bop-api` ตัวเดียว" ไว้ใน ARCHITECTURE.md ด้วย

**Blocked by:** ข้อตัดสินใจข้างบน

**Status:** needs-decision

ร่างเกณฑ์ (ยืนยันอีกครั้งหลังตัดสินใจ):

- [ ] บันทึกข้อตัดสินใจ (a) หรือ (b) และเขียน ADR ถ้าเลือก (b)
- [ ] ตอน start Job ที่ `pending`/`running` และไม่มีเจ้าของที่ยังอยู่ ถูกตั้งเป็น `failed` พร้อม `error` ที่บอกว่าถูกขัดจังหวะและมี `ended_at`
- [ ] Job ที่จบแล้ว (`succeeded`, `failed`, `cancelled`) และ Job ที่สร้างหลัง start ไม่ถูกแตะ
- [ ] Go test กับ PostgreSQL จริง: ใส่ Job หลายสถานะ รันการกวาด แล้วตรวจแต่ละแถว; repository แบบ memory ไม่ทำอะไร
- [ ] พิสูจน์กับ stack จริง: เริ่ม Job ที่ใช้เวลานาน (เช่นชี้ไป endpoint ที่ตอบช้า) แล้ว recreate `bop-api` กลางทาง Job ต้องเป็น `failed` หลังกลับมา — ถ้าทำอัตโนมัติใน `run-with-restart.sh` ไม่ได้ ให้เขียนเหตุผลและขั้นตอนทำมือ
- [ ] golangci-lint ผ่าน
- [ ] orchestrator deploy integration branch แล้ว `bru run --env minipc` ผ่านทั้งชุด
