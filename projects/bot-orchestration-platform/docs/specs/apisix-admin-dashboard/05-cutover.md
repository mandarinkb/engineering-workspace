# 05: ย้ายจริง: สลับ :8233 ไปที่ gateway โหมด etcd

Spec: [apisix-admin-dashboard.md](../apisix-admin-dashboard.md) · ADR 0003 กลไกข้อ 5

**What to build:** สลับพอร์ต `8233` จาก gateway โหมดไฟล์ไปที่ gateway โหมด etcd และเลิกใช้ gateway เดิมใน compose ที่ root ผู้ใช้ Temporal UI ต้องไม่เห็นความต่าง (302 เมื่อไม่ login, 503 เมื่อถาม `/me` ไม่ได้, เข้าได้เมื่อ login) ชุด dev ไม่เปลี่ยน

**Blocked by:** 01, 02, 03, 04

**Status:** done

**Verified:** 2026-11-25 บน Mini PC, branch `feat/apisix-etcd-cutover` (commit 4bb5692) — override เพิ่ม `etcd: restart: unless-stopped` (สำรองเป็น `.bak` ไว้ก่อนแก้ และลบทิ้งแล้วตามที่ผู้ใช้สั่งหลัง merge); dry-run แสดงว่าแตะแค่ `etcd` (restart policy เปลี่ยน) กับ `temporal-ui-gateway` ส่วน container อื่นแค่ Running; ย้ายจริงแล้ว: `seed.sh` เห็น "ยืนยัน: 302" ที่ `:8233`, `export.sh` ไม่มี drift, `bru run --env minipc` 96 request / 112 test ผ่าน, Playwright 9 ข้อผ่าน, fail closed (หยุด `bop-api`) ได้ 503 ทั้งไม่มี cookie และ cookie ปลอม แล้วกลับเป็น 302 หลังเปิด `bop-api`, Admin API จาก Mac ต่อไม่ได้ (000), dashboard ผ่าน tunnel เห็น route `temporal-ui`; ตัวลอง `gateway-next` ถูกลบและพอร์ต 18233 ว่าง

**เหตุการณ์ระหว่างย้าย (ต้องจำ):** Temporal UI ที่ `:8233` ดับราว 25 วินาที (แผนบอก 5–15) เพราะ `up -d` สร้าง gateway ใหม่ไม่ได้: พอร์ต `127.0.0.1:9180` ยังถูก container ลอง `gateway-next` ถือไว้ ผมลบตัวลองแล้วสั่ง `docker start` ตัวใหม่เอง ครั้งหน้าถ้ามี stack ลอง ที่ publish พอร์ตเดียวกัน ให้ลบ/หยุดมันก่อนสลับ

**ทางย้อนกลับ:** `ssh minipc 'cd ~/Workspace/engineering-workspace && git switch main-or-commit-ก่อนหน้า && docker-compose --env-file ... -f docker-compose.yml -f ../bop-lan.override.yml up -d'` (gateway โหมดไฟล์กลับมาที่ `8233`; ถ้าไม่ใช้ etcd แล้ว `stop`/`rm` เฉพาะตัวนั้น ข้อมูลอยู่ใน volume `engineering-workspace_etcd-data`) และลบสองบรรทัด `etcd:` / `restart: unless-stopped` ออกจาก `~/Workspace/bop-lan.override.yml` (ไฟล์ `.bak` ถูกลบแล้ว) **dry-run ทางย้อนกลับแล้ว (2026-11-25):** ใช้ compose เก่า (commit 57f8d17) กับ override ที่ไม่มี `etcd` → แตะแค่ `temporal-ui-gateway` (Recreate) container อื่นแค่ Running และ `etcd` กลายเป็น orphan (คำเตือน ไม่ถูกลบ ต้อง `stop`/`rm` เอง)

- [x] `docker-compose.yml` ที่ root: gateway โหมดไฟล์เดิมถูกแทนที่ด้วยตัวโหมด etcd ที่ publish `8233:9080` (ไม่ใช่ `18233`) ชื่อ service ที่ override ใน `bop-lan.override.yml` ตรงกัน (`restart: unless-stopped`) และ Admin API ยัง bind `127.0.0.1:9180` เท่านั้น
- [x] ไม่ลบ `config.yaml`/`apisix.yaml` ของโหมดไฟล์ เพราะชุด dev ยังใช้
- [x] เอกสารอัปเดต: ADR 0002 (หมายเหตุ), `README` ของ bop และ `environment.md` ของ skill `grill-to-deploy` ที่พูดถึงพอร์ต/วิธีรัน gateway; spec ต่างๆ เปลี่ยนสถานะตามจริง
- [x] **ผู้ดำเนินการ:** แผน deploy ได้รับอนุมัติ, snapshot ก่อนสลับ, `seed.sh` ผ่าน (302), gateway เดิมถูกแทนที่แล้วโดยไม่แตะ container อื่น
- [x] **ผู้ดำเนินการ:** regression ทั้งชุดที่ `:8233`: `bru run --env minipc` ผ่านทั้งชุด, Playwright ชุดเดิม 9 ข้อผ่าน, fail closed (หยุด `bop-api` ชั่วคราว ได้ 503) และตรวจว่า Admin API ต่อไม่ได้จาก Mac
- [x] **ผู้ดำเนินการ:** ย้อนกลับได้จริง: บันทึกคำสั่งย้อน (`git switch` ไป commit ก่อนหน้า + `up -d`) ไว้ใน ticket และทดสอบอย่างน้อยด้วย dry-run — dry-run แล้ว ดูหมายเหตุทางย้อนกลับด้านบน (ยังไม่ได้ย้อนกลับจริง)
- [x] **ผู้ดำเนินการ:** บันทึกต้นทุนการทำงานท้ายไฟล์ spec ตามเฟส 8 ของ `grill-to-deploy`
