# 03: gen-yaml สำหรับชุด dev

Spec: [apisix-admin-dashboard.md](../apisix-admin-dashboard.md) · Decisions ข้อ 1

**What to build:** ชุด dev ของโปรเจกต์ยังใช้ gateway โหมดไฟล์ (`apisix.yaml`) แต่ route ตรวจ session ตอนนี้มีแหล่งจริงเป็นไฟล์ JSON ใน `routes/` งานนี้ทำให้ `apisix.yaml` ถูกสร้างจาก JSON อัตโนมัติ และมีขั้นตรวจว่าไฟล์ที่ commit ตรงกับที่สร้างใหม่ ไม่ให้สองรูปแบบเพี้ยนกัน

**Blocked by:** 01

**Status:** implemented (โค้ดและ test ในเครื่องผ่านแล้ว รอเกณฑ์ผู้ดำเนินการบน Mini PC)

- [x] `docker/temporal-ui-gateway/gen-yaml.sh`: อ่าน `routes/*.json` เขียน `apisix.yaml` (ลงท้าย `#END`, คอมเมนต์หัวไฟล์ภาษาไทยบอกว่า "สร้างจาก routes/*.json ห้ามแก้ด้วยมือ") Lua ต้องเป็นข้อความเดียวกับใน JSON ทุกตัวอักษร
- [x] โหมดตรวจ (`gen-yaml.sh --check`): สร้างลงไฟล์ชั่วคราวแล้วเทียบกับ `apisix.yaml` ที่ commit; ต่างกันต้อง exit ≠ 0 และพิมพ์ผลต่าง; test ของสคริปต์: แก้ JSON ไป 1 อักขระต้องล้ม และคืนค่าแล้วต้องผ่าน
- [x] `apisix.yaml` ที่สร้างใหม่ทำงานเหมือนของเดิม (ถ้า route เดิมไม่เปลี่ยน ผลต่างควรเป็นแค่คอมเมนต์หัวไฟล์) ชุด dev ไม่ต้องแก้ compose
- [x] README ของ `docker/temporal-ui-gateway/` บอกให้รัน `gen-yaml.sh` หลังแก้ JSON และ `--check` ก่อน commit
- [ ] **ผู้ดำเนินการ:** รันการทดสอบชุด dev แบบแยก project บน Mini PC (วิธีเดียวกับที่ทำกับ ticket 04 ของ spec เดิม: project ชั่วคราว พอร์ตสำรอง ลบหลังจบ ขออนุมัติแผนก่อน) แล้ว `:<พอร์ตสำรอง>` ไม่มี cookie ได้ 302 และ login แล้วเข้า Temporal UI ได้
