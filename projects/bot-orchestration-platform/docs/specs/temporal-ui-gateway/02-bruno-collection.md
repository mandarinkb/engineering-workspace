# 02: Bruno collection ของ Temporal UI gateway

Spec: [temporal-ui-gateway.md](../temporal-ui-gateway.md)

**What to build:** ผู้ดำเนินการรันพฤติกรรมของ gateway แบบ black-box ด้วย `bru run --env minipc` ครอบทุกเคสของการตรวจ session ที่ `:8233`

**Blocked by:** 01

**Status:** implemented (ทำใน ticket 01 แล้ว; รอเกณฑ์ของผู้ดำเนินการ `bru run`)

- [x] request ใน `bruno/TemporalUiGateway/` ตาม README ของ bruno: `seq` ไม่ซ้ำ, ไม่มี cookie → 302 + `Location` ลงท้าย `/login`, cookie ปลอม → 302, login แล้ว → 200
- [x] request "ไม่มี cookie" วางหลัง logout ของโฟลเดอร์ (cookie jar ของ Bruno CLI แนบ cookie ให้ทุก request) และ Bruno ไม่ตาม redirect (assert สถานะ 302 ได้จริง)
- [x] assertion ตรวจ status และ header `Location` ไม่ใช่แค่ status; มี `docs` block ระบุเกณฑ์ที่ครอบ
- [ ] **ผู้ดำเนินการ:** `bru run --env minipc` ผ่านทั้งชุด และ `bruno/run-with-restart.sh` ยังผ่าน
