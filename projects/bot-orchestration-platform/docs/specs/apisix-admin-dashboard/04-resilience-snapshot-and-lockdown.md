# 04: snapshot, etcd ล่ม และ Admin API เข้าไม่ได้จาก LAN

Spec: [apisix-admin-dashboard.md](../apisix-admin-dashboard.md) · ADR 0003 กลไกข้อ 2 และ 4

**What to build:** พิสูจน์ว่ากลไกป้องกันของ ADR 0003 ทำงานจริงกับ gateway โหมด etcd (ยังบนพอร์ตชั่วคราว) ก่อนย้ายของจริง: กู้ route จาก snapshot ได้, เมื่อ etcd ล่ม gateway ไม่ปล่อยผ่าน, และ Admin API ต่อไม่ได้จากเครื่องอื่นใน LAN

**Blocked by:** 01

**Status:** done

**Verified:** 2026-11-25 บน Mini PC, branch `feat/apisix-etcd-gateway` (commit c7ff5de) — snapshot.sh ใช้ได้ (36–57 KB, สิทธิ์ 600); **restore.sh ล้มครั้งแรกเพราะ image etcd ไม่มี `/bin/sh`** (ล้มก่อนแตะข้อมูล) แก้เป็นสองขั้น (etcdutl + container ที่มี bash) แล้วลบ route → กู้จาก snapshot → route กลับ, export ไม่มี drift, `:18233` 302; ขณะ route ถูกลบ `:18233` ตอบ **404** (ไม่ปล่อยผ่าน); **etcd ล่ม 33 วินาที:** ไม่มี cookie ยัง 302, cookie ผิด 302, login แล้ว 200 (gateway ใช้ route ในหน่วยความจำต่อ, log มี `has no healthy etcd endpoint available`) และกลับเป็นปกติหลังเปิด etcd; Admin API จาก Mac `192.168.1.46:9180` ต่อไม่ได้ (000) แต่ผ่าน tunnel ต้องมี key (ไม่ใส่ 401, ผิด 401, ถูก 200) **ยังไม่ได้ตรวจ:** gateway restart ขณะ etcd ล่ม (ขึ้นไม่ได้หรือไม่ ควรเป็น fail closed) และ etcd ล่มนานกว่า 33 วินาที

- [x] `docker/temporal-ui-gateway/snapshot.sh` (เขียนแล้ว ยังไม่เคยรันจริง): `etcdctl snapshot save` ผ่าน `docker exec` ลง `~/Workspace/apisix-snapshots/` ตั้งชื่อไฟล์ด้วยเวลา เก็บ 5 ไฟล์ล่าสุด ลบเฉพาะไฟล์ในไดเรกทอรีนี้ที่ตรงรูปแบบชื่อของสคริปต์; `seed.sh` เรียก snapshot ก่อนเขียนทุกครั้ง (ถ้า snapshot ล้ม ต้องไม่ seed)
- [x] ขั้นตอนกู้คืนจาก snapshot เขียนเป็นไฟล์คำสั่งใน README (ภาษาไทย) และทดสอบแล้วบน stack ทดลอง — ผลจริง: ต้องแก้ restore.sh (ดูหมายเหตุ Status)
- [x] **ผู้ดำเนินการ:** หยุด `etcd` ของ stack ทดลองแล้ว `:18233` (ไม่มี cookie) ไม่ตอบ 200 จากหน้า Temporal UI (ผลที่ได้: บันทึกพฤติกรรมจริง ถ้า gateway ยังตอบตาม route ที่โหลดไว้ในหน่วยความจำก็ต้องยังเป็น 302 ไม่ใช่ 200) แล้วเปิดกลับและทุกอย่างกลับมา
- [x] **ผู้ดำเนินการ:** จาก Mac `curl -m 5 http://192.168.1.46:9180/apisix/admin/routes` ต้องต่อไม่ได้ (connection refused หรือ timeout) และจาก Mini PC เอง ต่อได้ด้วย key
- [x] **ผู้ดำเนินการ:** ลบ route ใน etcd ของ stack ทดลองแล้วกู้จาก snapshot ล่าสุดได้ route กลับมา และ `:18233` กลับเป็น 302
- [x] บันทึกผลที่ได้ (โดยเฉพาะพฤติกรรมตอน etcd ล่ม) กลับเข้า ADR 0003 หัวข้อ "ยังไม่พิสูจน์" โดยเปลี่ยนเป็น "พิสูจน์แล้ว" หรือแก้การตัดสินใจถ้าผลไม่ตรงที่คาด
