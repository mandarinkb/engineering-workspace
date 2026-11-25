# ถอด APISIX gateway ออก เปิด Temporal UI ตรงที่พอร์ต 8233

ADR 0002 และ 0003 วาง APISIX (พร้อม etcd) ไว้หน้า Temporal UI เพื่อให้เข้าได้เฉพาะผู้ที่ login BOP เราถอดออกทั้งชุด Temporal UI เปิดตรงที่ `:8233` ไม่มีด่านตรวจ login ADR นี้แทนที่ ADR 0002 และ 0003 ทั้งหมด

## เหตุผล

- gateway ทำงานเดียวคือกัน Temporal UI แต่ต้องแลกกับ container เพิ่ม 2 ตัว (`temporal-ui-gateway`, `etcd`), route เดียว, และสคริปต์ดูแลอีกชุด (seed, snapshot, restore, export, drift) เกินความจำเป็นของระบบที่ยังมีบริการไม่กี่ตัว
- `bop-api` และ dashboard ไม่เคยอยู่หลัง gateway จึงยังไม่ได้ใช้จุดเด่นของ API gateway เลย (rate limit กลาง, auth กลาง, routing หลายบริการ)
- การเรียนรู้ APISIX ยังทำได้ที่ `labs/apisix` และกลับมาใช้จริงทีหลังเมื่อมีหลายบริการ

## ผลที่ตามมา

- **Temporal UI เปิดโล่งในเครือข่าย:** ใครเข้า `http://<host>:8233` ได้ก็ดู แก้ Terminate/Delete workflow ได้ ยอมรับได้เฉพาะเครือข่ายบ้านที่ไว้ใจ ห้ามเปิดออกอินเทอร์เน็ต
- พอร์ต gRPC `7233` ยังผูกที่ `127.0.0.1` เหมือนเดิม
- ถอด service `temporal-ui-gateway`, `etcd` และ volume `etcd-data` ออกจาก compose ทั้งสองไฟล์ ลบ `docker/temporal-ui-gateway/` และชุด Bruno `TemporalUiGateway`
- งานเดิมเก็บไว้ที่ git tag `archive/apisix-gateway` (ADR, route, สคริปต์, spec) ถ้าจะกลับมาใช้ให้ checkout จาก tag นั้น
- บน Mini PC: ไฟล์ override `~/Workspace/bop-lan.override.yml` (นอก repo) ยังมี `etcd` และ `temporal-ui-gateway` ต้องลบสองบล็อกนี้ออกก่อน deploy ไม่งั้น compose error ("service has neither an image nor a build context") และ `APISIX_ADMIN_KEY` ใน `~/Workspace/apisix-admin.env` ใช้ไม่ได้แล้ว
