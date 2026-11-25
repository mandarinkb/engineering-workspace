# Spec: Temporal UI gateway แบบ etcd + Admin API + dashboard

> **เลิกใช้แล้ว ([ADR 0004](../adr/0004-remove-apisix-gateway.md)):** spec นี้เก็บไว้เป็นประวัติ ไฟล์ที่อ้างถึงอยู่ที่ git tag `archive/apisix-gateway`

> สถานะ: implemented และ deploy บน Mini PC แล้ว (2026-11-25) · ศัพท์ตาม [GLOSSARY.md](../../GLOSSARY.md) · การตัดสินใจ: [ADR 0003](../adr/0003-apisix-gateway-with-etcd-and-dashboard.md) (แทนที่ข้อ "โหมดไฟล์" ของ [ADR 0002](../adr/0002-temporal-ui-behind-apisix-gateway.md)) · ต่อยอดจาก [temporal-ui-gateway.md](temporal-ui-gateway.md)

## Problem Statement

**Temporal UI gateway** ตอนนี้เป็น APISIX standalone โหมดไฟล์ ไม่มี Admin API และไม่มี dashboard ผู้ใช้ต้องการเรียนรู้ APISIX ผ่าน dashboard กับ gateway ตัวจริง (ไม่ใช่ lab แยก) การแก้ route ตอนนี้ทำได้แค่แก้ `apisix.yaml` แล้ว restart

## Solution

เปลี่ยน gateway ตัวจริงที่ `:8233` เป็นโหมด traditional (`config_provider: etcd`) เปิด Admin API และ dashboard ในตัว (`enable_admin_ui: true`) โดยคงพฤติกรรมของ gateway ตามเดิม (ตรวจ `/me` → 200 ส่งต่อ, 401 → 302 ไป login, อย่างอื่น/ถามไม่ได้ → 503, ตัด Cookie) และเติมกลไกป้องกัน 5 ข้อของ ADR 0003:

```text
git: docker/temporal-ui-gateway/routes/*.json  ──seed.sh──▶  etcd  ◀──dashboard (ผ่าน ssh -L, 127.0.0.1:9180)
                                                 │ ▲
                              export.sh ◀────────┘ └── drift report (route ใน etcd ที่ไม่ตรงกับ git)

เบราว์เซอร์ → :8233 gateway (อ่าน route จาก etcd) → ตรวจ /me → temporal-ui:8080
```

## User Stories

1. ในฐานะผู้เรียน ฉันเปิด dashboard ผ่าน SSH tunnel เห็นรายการ route จริง และสร้าง/แก้ route ทดลองได้
2. ในฐานะผู้ดูแล ฉันรัน seed หลัง deploy ทุกครั้งแล้ว route ที่ตรวจ session ถูกเขียนลง etcd จาก git โดยไม่ต้องคลิก
3. ในฐานะผู้ดูแล ฉันรัน export แล้วเห็นว่า route ใน etcd ต่างจาก git ตรงไหน เพื่อ commit สิ่งที่แก้ผ่าน UI หรือยอมให้ถูกเขียนทับ
4. ในฐานะผู้ดูแล หลังแก้ route ผ่าน UI ฉันรันชุดทดสอบเดิมแล้วรู้ทันทีถ้า `:8233` เปิดโล่งโดยไม่ login
5. ในฐานะผู้ดูแล Admin API และ dashboard เข้าได้จากเครื่อง Mini PC เองเท่านั้น (และจาก Mac ผ่าน tunnel) ไม่เข้าได้จากเครื่องอื่นใน LAN
6. ในฐานะผู้ดูแล ฉันกู้คืน route ได้จาก snapshot ของ etcd ถ้าข้อมูลเสียหาย
7. ในฐานะผู้ใช้ Temporal UI ฉันไม่เห็นความแตกต่างจากเดิม (ยังต้อง login BOP, ยังได้ 302/503 ตามเดิม) ระหว่างและหลังการย้าย

## Implementation Decisions

- **service ใหม่ `etcd`** ใน `docker-compose.yml` ที่ root image pin เวอร์ชัน (ตรวจ tag ด้วย `docker buildx imagetools inspect` ก่อน) ข้อมูลใน named volume ไม่ publish พอร์ตออกนอก network
- **`temporal-ui-gateway`:** `role: traditional`, `config_provider: etcd`, `etcd.host` ชี้ `etcd`, `deployment.admin` ตั้ง `enable_admin_ui: true`, `admin_key_required: true`, admin key อ่านจาก env (`APISIX_ADMIN_KEY`, ตั้งบน Mini PC นอก repo ห้าม commit; compose ต้องไม่ start ถ้าไม่ตั้ง), `allow_admin` จำกัดแคบ, publish `127.0.0.1:9180:9180` เท่านั้น ส่วน `:8233` ยังเป็น node_listen เดิม
- **ไฟล์ config ของโหมดใหม่แยกจากของเดิม** (`config-etcd.yaml`) ชุด dev (`projects/bot-orchestration-platform/docker-compose.yml`) mount `config.yaml` กับ `apisix.yaml` ร่วมกับ root ในปัจจุบัน ถ้าแก้สองไฟล์นั้นเป็นโหมด etcd ชุด dev จะพัง ชุด dev **คงโหมดไฟล์เดิมไว้** (ไม่อยู่ในขอบเขต) โดย `apisix.yaml` ถูกสร้างจาก JSON ตาม Decisions ข้อ 1
- **route เก็บเป็น JSON ของ Admin API** ใน `docker/temporal-ui-gateway/routes/` (route id คงที่ เช่น `temporal-ui`) Lua เป็น string ใน JSON
- **`seed.sh`:** รันบน Mini PC หลัง gateway พร้อม อ่าน admin key จาก env, `PUT /apisix/admin/routes/{id}` ต่อไฟล์ (idempotent), ไม่ลบ route อื่นใน etcd (เพื่อให้ทดลองผ่าน UI ได้) แต่พิมพ์รายการ route ที่ไม่ได้มาจาก git เป็น **drift** แล้วตรวจท้ายสุดว่า `GET :8233/` ไม่มี cookie ได้ **302** ไป `/login` ถ้าไม่ใช่ **ล้มด้วย exit ไม่ใช่ 0** (สอดคล้องหลัก "exit code ไม่ใช่หลักฐาน": ต้องเห็นตัวยืนยัน 302 จริง)
- **`export.sh`:** `GET /apisix/admin/routes` เขียนเป็น JSON ที่จัดรูปแบบคงที่ลงโฟลเดอร์ชั่วคราว แล้วเทียบกับ `routes/` ด้วย `diff` พิมพ์ผลต่าง ไม่แก้ไฟล์ใน repo เอง (ผู้ใช้เป็นคนคัดลอกกลับถ้าต้องการ)
- **snapshot:** ก่อน seed ทุกครั้ง `etcdctl snapshot save` ผ่าน `docker exec` ลงไดเรกทอรีนอก repo บน Mini PC (ขออนุมัติการสร้างไดเรกทอรีตอน deploy) เก็บ 5 ไฟล์ล่าสุด
- **ย้ายสองขั้น:** (1) ยก gateway ใหม่ข้างๆ ที่ `:18233` ด้วย route เดียวกัน รันชุดทดสอบให้ผ่านทั้งหมด (2) สลับ `:8233` ไปที่ตัวใหม่ ลบ gateway แบบไฟล์ ย้อนกลับ: `git switch` ไป commit เดิมแล้ว `up -d`
- ไม่แก้ `bop-api`, dashboard ของ BOP และ Temporal UI

## Testing Decisions

seam ที่เสนอ (ผู้ใช้ให้ไปต่อจากข้อเสนอนี้ ถ้าไม่เห็นด้วยให้แก้ก่อนเริ่ม ticket):

1. **Bruno black-box กับ `:8233`** (โฟลเดอร์ `TemporalUiGateway` เดิม) ต้องผ่านเท่าเดิมหลังย้าย และเพิ่มเคสที่ทำได้: คำขอที่ gateway ตอบ 503 เมื่อ `bop-api` ถูกหยุด (ถ้าทำใน Bruno ไม่ได้ ใช้ตรวจมือตามเดิมและบันทึกเหตุผล)
2. **Playwright ในเบราว์เซอร์จริง** (ชุดเดิม 9 ข้อ) ต้องผ่านหลังย้ายและหลังแก้ route ทดลองผ่าน UI แล้วลบทิ้ง และเพิ่ม: เปิด dashboard ผ่าน SSH tunnel (`127.0.0.1:19180` หรือพอร์ตที่ตั้ง) กรอก admin key (อ่านจากไฟล์บน Mini PC ไม่พิมพ์ลงบันทึก) เห็นรายการ route
3. **ชั้นสคริปต์ seed/export:** รัน seed แล้ว export ต้อง `diff` ว่างสำหรับ route ที่ git ดูแล; ทดสอบว่า seed ล้ม (exit ≠ 0) เมื่อ route ที่ถูก seed ทำให้ `:8233` ไม่ตอบ 302 (ตรวจด้วย route ทดลองที่ทำให้เปิดโล่ง แล้วลบทิ้ง)
4. **ตรวจมือ:** `curl` Admin API จากเครื่องอื่นใน LAN (เช่น Mac) ที่ `192.168.1.46:9180` ต้องต่อไม่ได้; หยุด `etcd` ชั่วคราว (ขออนุมัติ) แล้วดูว่า `:8233` ไม่ปล่อยผ่าน; กู้ route จาก snapshot ได้จริงบน stack ทดลอง

## Out of Scope

- TLS สำหรับ Admin API/dashboard, การแยกสิทธิ์หลายผู้ใช้ใน dashboard
- ย้ายชุด dev ของโปรเจกต์ไปโหมด etcd (ยังคงโหมดไฟล์)
- ย้าย `bop-api`/dashboard ของ BOP มาอยู่หลัง APISIX และเรื่อง gRPC 7233 ผ่าน gateway
- etcd แบบ cluster หรือ HA
- การทำให้ dashboard ของ APISIX เป็นของเก็บ route จริงแทน git (ตัดสินแล้วใน ADR 0003 ว่า git เป็นหลัก)

## Decisions จากคำถามเปิด (ผู้ใช้ตอบรับคำแนะนำ 2026-11-25)

1. **route ซ้ำสองรูปแบบ:** ไฟล์ JSON ใน `routes/` เป็นแหล่งเดียว สคริปต์ `gen-yaml.sh` สร้าง `apisix.yaml` ของชุด dev จาก JSON นั้น (ตัว Lua เป็นข้อความเดียวกัน) และมีขั้นตรวจว่าไฟล์ที่ commit ตรงกับผลที่สร้างใหม่ (ล้มถ้าต่าง) ห้ามแก้ `apisix.yaml` ด้วยมือ
2. **ที่เก็บ snapshot ของ etcd:** `~/Workspace/apisix-snapshots/` บน Mini PC (นอก repo) เก็บ 5 ไฟล์ล่าสุด ต้องขออนุมัติสร้างไดเรกทอรีตอน deploy

## Further Notes

- **ความเสี่ยงหลัก:** Admin API แก้ gateway ที่ปกป้อง Temporal UI ได้ทั้งตัว ความปลอดภัยพึ่งการจำกัดการเข้าถึง (ข้อ 2 ของ ADR 0003) และชุดทดสอบ (ข้อ 3) ทั้งคู่ต้องผ่านก่อนถือว่าเสร็จ
- **ยังไม่พิสูจน์:** ฟีเจอร์ของ dashboard ในตัว 3.21 ครบแค่ไหน, พฤติกรรมเมื่อ etcd ล่ม, วิธี inject admin key เข้า `config.yaml` (รูปแบบอ้างตัวแปร env), และ `allow_admin` เมื่อคำขอผ่านการ publish ของ Docker (ที่อยู่ต้นทางที่ APISIX เห็นอาจเป็น IP ของ bridge ไม่ใช่ 127.0.0.1) ต้องทดสอบในขั้นย้ายสองขั้นบนพอร์ตชั่วคราวก่อนสลับ

## ต้นทุนการทำงาน

tickets=5 · subagents=0 (ผู้ช่วยหลักทำเองทั้งหมดตามที่ผู้ใช้อนุมัติ เพราะแต่ละ ticket เล็กและต้องทดสอบบน Mini PC) · รอบแก้=2 (restore.sh ล้มเพราะ image etcd ไม่มี shell, และลำดับย้ายจริงพลาดเรื่องพอร์ต 9180 ทำให้ Temporal UI ดับ ~25 วินาที) · verify.sh ไม่ได้ใช้ (ไม่มีโค้ด Go/TS; test ของสคริปต์รันในเครื่องด้วย `tests/test-drift.sh`, `tests/test-gen-yaml.sh`) · หนักสุด: ทดสอบส่วน A/B บน Mini PC รวม Playwright ราว 10 รอบ และการเดารูปแบบ Admin API/entrypoint ที่ต้องแก้ตอนรันจริงหนึ่งครั้ง
