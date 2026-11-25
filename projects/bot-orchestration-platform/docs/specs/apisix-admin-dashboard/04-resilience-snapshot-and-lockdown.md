# 04: snapshot, etcd ล่ม และ Admin API เข้าไม่ได้จาก LAN

Spec: [apisix-admin-dashboard.md](../apisix-admin-dashboard.md) · ADR 0003 กลไกข้อ 2 และ 4

**What to build:** พิสูจน์ว่ากลไกป้องกันของ ADR 0003 ทำงานจริงกับ gateway โหมด etcd (ยังบนพอร์ตชั่วคราว) ก่อนย้ายของจริง: กู้ route จาก snapshot ได้, เมื่อ etcd ล่ม gateway ไม่ปล่อยผ่าน, และ Admin API ต่อไม่ได้จากเครื่องอื่นใน LAN

**Blocked by:** 01

**Status:** implemented (สคริปต์เขียนแล้วแต่ยังไม่เคยรัน; การทดสอบบน Mini PC ต้องได้รับอนุมัติ: สร้าง `~/Workspace/apisix-snapshots/` และหยุด `etcd` ของ stack ทดลองชั่วคราว)

- [x] `docker/temporal-ui-gateway/snapshot.sh` (เขียนแล้ว ยังไม่เคยรันจริง): `etcdctl snapshot save` ผ่าน `docker exec` ลง `~/Workspace/apisix-snapshots/` ตั้งชื่อไฟล์ด้วยเวลา เก็บ 5 ไฟล์ล่าสุด ลบเฉพาะไฟล์ในไดเรกทอรีนี้ที่ตรงรูปแบบชื่อของสคริปต์; `seed.sh` เรียก snapshot ก่อนเขียนทุกครั้ง (ถ้า snapshot ล้ม ต้องไม่ seed)
- [ ] ขั้นตอนกู้คืนจาก snapshot เขียนเป็นไฟล์คำสั่งใน README (ภาษาไทย) และทดสอบแล้วบน stack ทดลอง
- [ ] **ผู้ดำเนินการ:** หยุด `etcd` ของ stack ทดลองแล้ว `:18233` (ไม่มี cookie) ไม่ตอบ 200 จากหน้า Temporal UI (ผลที่ได้: บันทึกพฤติกรรมจริง ถ้า gateway ยังตอบตาม route ที่โหลดไว้ในหน่วยความจำก็ต้องยังเป็น 302 ไม่ใช่ 200) แล้วเปิดกลับและทุกอย่างกลับมา
- [ ] **ผู้ดำเนินการ:** จาก Mac `curl -m 5 http://192.168.1.46:9180/apisix/admin/routes` ต้องต่อไม่ได้ (connection refused หรือ timeout) และจาก Mini PC เอง ต่อได้ด้วย key
- [ ] **ผู้ดำเนินการ:** ลบ route ใน etcd ของ stack ทดลองแล้วกู้จาก snapshot ล่าสุดได้ route กลับมา และ `:18233` กลับเป็น 302
- [ ] บันทึกผลที่ได้ (โดยเฉพาะพฤติกรรมตอน etcd ล่ม) กลับเข้า ADR 0003 หัวข้อ "ยังไม่พิสูจน์" โดยเปลี่ยนเป็น "พิสูจน์แล้ว" หรือแก้การตัดสินใจถ้าผลไม่ตรงที่คาด
