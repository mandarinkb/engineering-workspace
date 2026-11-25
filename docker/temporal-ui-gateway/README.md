# Temporal UI gateway (APISIX)

ด่านหน้า Temporal UI ที่ให้เฉพาะผู้ที่ login BOP เข้าได้ ตัดสินใจไว้ใน [ADR 0002](../../projects/bot-orchestration-platform/docs/adr/0002-temporal-ui-behind-apisix-gateway.md) และ [ADR 0003](../../projects/bot-orchestration-platform/docs/adr/0003-apisix-gateway-with-etcd-and-dashboard.md) (โหมด etcd + dashboard) ดู spec ที่ `projects/bot-orchestration-platform/docs/specs/apisix-admin-dashboard.md`

## ไฟล์

| ไฟล์ | หน้าที่ |
|---|---|
| `routes/*.json` | **แหล่งจริงของ route** (รูปแบบ Admin API) ชื่อไฟล์ = route id |
| `config-etcd.yaml` | config ของ gateway โหมด etcd (template: key มาจากตัวแปร env) |
| `seed.sh` | สำรอง etcd แล้วเขียน `routes/*.json` ลง etcd รายงาน drift และตรวจว่าไม่ login ได้ 302 |
| `export.sh` | เทียบ route ใน etcd กับ git (ผลต่างบอกว่าอะไรแก้ผ่าน dashboard) ไม่แก้ไฟล์ใน repo |
| `snapshot.sh` / `restore.sh` | สำรอง/กู้ etcd (`~/Workspace/apisix-snapshots/` เก็บ 5 ไฟล์ล่าสุด) |
| `gen-yaml.sh` | สร้าง `apisix.yaml` ของชุด dev (โหมดไฟล์) จาก `routes/*.json` |
| `config.yaml`, `apisix.yaml` | โหมดไฟล์ ใช้โดยชุด dev ของโปรเจกต์ — **ห้ามแก้ `apisix.yaml` ด้วยมือ** |
| `tests/` | `test-drift.sh`, `test-gen-yaml.sh` รันได้ในเครื่องโดยไม่ต้องใช้ Docker |

## แก้ route อย่างไร

1. แก้ `routes/<id>.json` แล้วรัน `./gen-yaml.sh` (ให้ `apisix.yaml` ของชุด dev ตรงกัน) และ `./tests/test-gen-yaml.sh`
2. deploy แล้วรัน `seed.sh` บน Mini PC (เห็น "ยืนยัน: 302" = ผ่าน)
3. ถ้าลองแก้ผ่าน dashboard: รัน `export.sh` เพื่อดู drift ถ้าอยากเก็บสิ่งที่แก้ ให้คัดลอกไฟล์จากโฟลเดอร์ที่ `export.sh` บอกกลับมาแก้ใน `routes/` (ลบฟิลด์ที่เซิร์ฟเวอร์เติมเองที่ไม่ต้องการ) แล้ว commit ไม่งั้น route ที่ git ดูแลจะถูกเขียนทับตอน seed ครั้งถัดไป
4. หลังแก้ route ผ่าน UI ทุกครั้ง ให้รัน Bruno โฟลเดอร์ `TemporalUiGateway` และ Playwright ชุด gateway เพราะ route ผิดอาจทำให้ Temporal UI เปิดโล่ง

## เปิด dashboard

Admin API และ dashboard เข้าได้จากเครื่อง Mini PC เองเท่านั้น (`127.0.0.1:9180`) จาก Mac ใช้ SSH tunnel:

```bash
ssh -L 19180:127.0.0.1:9180 minipc
# เปิด http://127.0.0.1:19180/ui/ แล้วใส่ admin key
ssh minipc 'sed -n "s/^APISIX_ADMIN_KEY=//p" ~/Workspace/apisix-admin.env'   # อ่าน key (อย่าวางลงแชตหรือ commit)
```

key อยู่ในไฟล์ `~/Workspace/apisix-admin.env` (สิทธิ์ 600 นอก repo) Admin API แก้ gateway ได้ทั้งตัว รวมถึงปิดการตรวจ session ของ Temporal UI จึงห้ามเปิดพอร์ต 9180 ออก LAN

## สำรองและกู้คืน

```bash
cd docker/temporal-ui-gateway
./snapshot.sh                                  # seed.sh เรียกให้อัตโนมัติก่อนเขียน
./restore.sh ~/Workspace/apisix-snapshots/etcd-<เวลา>.db --yes   # ทำลายข้อมูล etcd ปัจจุบัน
```

หลังกู้ รัน `export.sh` เพื่อดูว่า route ตรงกับที่ต้องการ แล้ว `seed.sh` ถ้าอยากให้ตรงกับ git
