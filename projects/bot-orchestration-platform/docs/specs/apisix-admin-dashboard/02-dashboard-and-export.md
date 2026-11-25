# 02: dashboard ผ่าน tunnel และ export/drift

Spec: [apisix-admin-dashboard.md](../apisix-admin-dashboard.md)

**What to build:** ผู้เรียนเปิด dashboard จาก Mac ผ่าน `ssh -L` เห็น route และแก้ได้ และผู้ดูแลรู้ว่าสิ่งที่แก้ใน etcd ต่างจาก git ตรงไหน (`export.sh`)

**Blocked by:** 01

**Status:** implemented (โค้ดและ test ในเครื่องผ่านแล้ว รอเกณฑ์ผู้ดำเนินการบน Mini PC)

- [x] (ทำจริงด้วย `export.sh` + `drift.py` เทียบแบบ subset เพราะเซิร์ฟเวอร์เติมค่า default ให้ route; plugin และ node ต้องตรงทุกคีย์) `docker/temporal-ui-gateway/export.sh`: `GET /apisix/admin/routes` เขียน JSON จัดรูปแบบคงที่ (เรียงคีย์) ลงโฟลเดอร์ชั่วคราว แล้ว `diff` กับ `routes/` พิมพ์ผลต่าง ไม่แก้ไฟล์ใน repo ไม่พิมพ์ admin key
- [x] (test ด้วยไฟล์จำลองผลของ Admin API: `tests/test-drift.sh` 9 เคสผ่าน; รูปแบบผลลัพธ์จริงของ Admin API ต้องยืนยันบน Mini PC) export หลัง seed ได้ผลต่างว่างสำหรับ route ที่ git ดูแล; เพิ่ม route ทดลองใน etcd แล้ว export ต้องรายงานว่าเป็น drift (ผู้ทำเขียน test ของสคริปต์ด้วย shell แบบ black-box หรือบันทึกวิธีทดสอบมือ)
- [x] README ของ `docker/temporal-ui-gateway/` (ภาษาไทย): วิธีเปิด tunnel (`ssh -L 19180:127.0.0.1:9180 minipc`), อ่าน admin key จากที่ตั้งไว้, สิ่งที่ไม่ควรทำใน UI (แก้ route ที่ git ดูแล), และวิธี export/commit กลับ
- [ ] **ผู้ดำเนินการ (Playwright):** เปิด `http://127.0.0.1:19180/ui` ผ่าน tunnel กรอก admin key (อ่านจากไฟล์บน Mini PC ไม่พิมพ์ลงบันทึก) เห็นรายการ route รวม `temporal-ui`; สร้างและลบ route ทดลองผ่าน UI ได้
- [ ] **ผู้ดำเนินการ:** หลังสร้าง route ทดลองผ่าน UI `export.sh` รายงาน drift และหลังลบแล้ว drift หาย
