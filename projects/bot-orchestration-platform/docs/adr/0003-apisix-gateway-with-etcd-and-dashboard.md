# Temporal UI gateway ใช้ APISIX โหมด etcd พร้อม Admin API และ dashboard

ADR 0002 ให้ gateway รันแบบ standalone อ่าน route จากไฟล์ `apisix.yaml` (ไม่มี etcd ไม่มี Admin API) เราเปลี่ยนเป็น **โหมด traditional (`config_provider: etcd`)** พร้อม Admin API และ dashboard ในตัว (`enable_admin_ui: true`) บน gateway ตัวจริงที่พอร์ต 8233 เพื่อให้ผู้ใช้เรียนรู้ APISIX ผ่าน dashboard กับ gateway ที่ใช้งานจริง ข้อตัดสินใจเรื่อง "gateway ตรวจ session ด้วย `/me` แล้ว redirect/503" ของ ADR 0002 ยังเหมือนเดิมทุกประการ เปลี่ยนเฉพาะ **วิธีเก็บและแก้ route**

ADR นี้แทนที่เฉพาะข้อ "โหมดไฟล์ ไม่มี etcd" ของ [ADR 0002](0002-temporal-ui-behind-apisix-gateway.md)

## Considered Options

- **คงโหมดไฟล์ (ADR 0002)**: route อยู่ใน git review ได้ ไม่มี Admin API ที่แก้ gateway ได้ ปลอดภัยที่สุด แต่ไม่มี dashboard ให้เรียนรู้
- **ยก lab แยกเป็น compose profile (ทาง C)**: เรียนรู้ผ่าน dashboard โดยไม่แตะ gateway จริง เสี่ยงต่ำสุดและแนะนำโดยผู้ช่วย แต่ผู้ใช้เลือกไม่ใช้ เพราะต้องการเล่นกับ gateway ของจริง
- **โหมด etcd + dashboard บน gateway จริง (ที่เลือก)**: ได้ dashboard กับของจริงตัวเดียว ไม่มีของซ้ำ แลกกับความเสี่ยงด้านความปลอดภัยและการสูญเสีย review ผ่าน git ซึ่งต้องปิดด้วยกลไกด้านล่าง

## กลไกที่ต้องมีควบคู่ (เงื่อนไขของการตัดสินใจนี้)

1. **แหล่งความจริงของ route อยู่ใน git:** route เก็บเป็น JSON ของ Admin API ใน repo (`docker/temporal-ui-gateway/`) และมีสคริปต์ seed ที่ PUT เข้า etcd ทุกครั้งที่ deploy พร้อมสคริปต์ export ดึง route ปัจจุบันจาก etcd มาเทียบ (`git diff`) ค่าที่แก้ผ่าน UI ต้องถูก export กลับมา commit ไม่งั้นถูกเขียนทับตอน seed ครั้งถัดไป ไฟล์ `apisix.yaml` เดิมเลิกใช้เพราะโหมด etcd ไม่อ่านไฟล์นี้
2. **Admin API และ dashboard เข้าได้จาก localhost เท่านั้น:** bind `127.0.0.1:9180` บน Mini PC เข้าจากเครื่องอื่นผ่าน `ssh -L` ตั้ง `allow_admin` ให้แคบ ใช้ admin key สุ่มที่ไม่ commit (อ่านจากตัวแปร env) และ `admin_key_required: true` ห้ามใช้ key default
3. **ชุดทดสอบเดิมเป็น regression:** หลังแก้ route ผ่าน UI ต้องรัน Bruno `TemporalUiGateway` กับ Playwright และสคริปต์ seed ตรวจว่าไม่มี cookie ที่ `:8233` ได้ 302 ถ้าไม่ใช่ให้ล้มดังๆ เพิ่มเคส fail closed (503) ใน Bruno ถ้าทำได้
4. **etcd มี named volume และ snapshot:** ก่อน seed ทุกครั้ง `etcdctl snapshot save` ลงไดเรกทอรีนอก repo บน Mini PC และต้องทดสอบว่าเมื่อ etcd ล่มแล้ว gateway ไม่ปล่อยผ่าน (ยังไม่ได้ตรวจ ดูหัวข้อ "ยังไม่พิสูจน์")
5. **ย้ายสองขั้น:** ยกโหมดใหม่ขึ้นข้างๆ ที่พอร์ตชั่วคราว (เช่น 18233) ใช้ route เดียวกันและรันชุดทดสอบให้ผ่านก่อน แล้วจึงสลับพอร์ต 8233 ไปที่ตัวใหม่ และลบ gateway แบบไฟล์ ย้อนกลับด้วย `git switch` ไป commit เดิมแล้ว `up -d`

## Consequences

- **ความปลอดภัยแย่ลงกว่า ADR 0002 โดยตั้งใจ:** Admin API แก้ gateway ได้ทั้งตัว รวมถึงปิดการตรวจ session ของ Temporal UI ได้ ผู้ที่เข้า Admin API ได้ = ข้ามการป้องกันทั้งหมด จึงต้องคุมการเข้าถึงตามกลไกข้อ 2 อย่างเคร่งครัด
- gateway พึ่ง etcd ซึ่งเป็น service ที่มีสถานะเพิ่มอีกตัว (RAM เพิ่มราวหลักสิบ MB และต้องดูแล backup)
- การแก้ route ผ่าน UI ไม่ผ่าน PR review ความถูกต้องพึ่ง export + regression (กลไกข้อ 1 และ 3) ไม่ใช่ git โดยอัตโนมัติ
- ต้องรัน seed ตอน deploy และ route อยู่ในรูป JSON ของ Admin API (Lua เป็น string ใน JSON) อ่านยากกว่า YAML เดิม
- ADR 0002 ยังถูกต้องในส่วน: ตรวจ `/me`, redirect ด้วย `$host`, ตัด Cookie ด้วย `serverless-post-function`, ลำดับเฟส plugin (rewrite ก่อน access), fail closed
- ถ้าวันหนึ่งต้องกลับไปโหมดไฟล์ ให้ export route จาก etcd มาเขียนเป็น `apisix.yaml` ได้ (กลไกข้อ 1 ทำให้ทั้งสองรูปแบบแปลงกันได้)

## พิสูจน์แล้ว (ticket 01, 2026-11-25)

- tag ของ etcd: `quay.io/coreos/etcd:v3.5.21` มีจริงและใช้ได้
- `deployment.etcd`, `admin_key`, `enable_admin_ui` ใน config ของ 3.21 ถูกต้อง: gateway เริ่มได้, `/ui` ตอบ 200, Admin API ใช้ key บังคับจริง
- การใส่ admin key จาก env ผ่าน entrypoint (sed ลง config ตอนเริ่ม) ใช้ได้ และ key ไม่รั่วใน log
- `allow_admin: 172.16.0.0/12` พอสำหรับคำขอที่ผ่านพอร์ตที่ publish ของ Docker (เข้าผ่าน tunnel ได้) และจาก LAN ต่อไม่ได้เพราะ bind `127.0.0.1`
- Lua เก็บเป็น JSON ผ่าน Admin API ใช้ได้ (ผล 302 และ Bruno/Playwright ผ่านเหมือน gateway แบบไฟล์)
- **etcd ล่ม (ticket 04):** gateway ยังตอบด้วย route ในหน่วยความจำ ไม่ปล่อยผ่าน (ไม่มี cookie → 302, login แล้ว → 200) ทดสอบ 33 วินาที และกลับเป็นปกติเมื่อ etcd กลับมา; route ที่ถูกลบออกจาก etcd ทำให้ `:<พอร์ต>` ตอบ 404 ไม่ใช่เปิดโล่ง
- **snapshot และ restore (ticket 04):** กู้ route ที่ถูกลบกลับมาได้จริง (image etcd ไม่มี shell จึงต้องกู้ด้วย etcdutl + container ที่มี bash)
- **drift (ticket 02):** `export.sh`/`seed.sh` อ่านผลจริงของ Admin API ได้ จับ route ใหม่และ plugin ที่เพิ่มใน route ที่ git ดูแลได้ และ seed เขียนทับกลับได้
- **dashboard (ticket 02):** ใส่ admin key ในหน้า Settings แล้วเห็นรายการ route ได้ผ่าน SSH tunnel
- **gateway restart ขณะ etcd ล่ม (ticket 04):** gateway **หยุด (exit 1) และพอร์ตปิด** ไม่ปล่อยผ่าน (fail closed) log: `all etcd nodes are unavailable` เมื่อ etcd กลับมาแล้วสั่ง start gateway ใหม่ก็กลับเป็น 302 ปกติ (stack ลองไม่มี restart policy จึงต้องสั่งเอง ตัวจริงมี `restart: on-failure` และ override `unless-stopped`)
- **ย้ายจริงที่ `:8233` (ticket 05):** ชุดทดสอบเดิมผ่านทั้งหมดบน gateway โหมด etcd (Bruno 96/112, Playwright 9 ข้อ, fail closed 503, Admin API ต่อไม่ได้จาก LAN) ผู้ใช้ Temporal UI เห็น 302/200/503 เหมือนเดิม ราคาที่จ่ายคือ Temporal UI ดับราว 25 วินาทีตอนสลับ (ดู ticket 05)

## ยังไม่พิสูจน์

- ฟีเจอร์ของ dashboard ครบแค่ไหนเมื่อสร้าง/แก้ route และ plugin ด้วยปุ่ม (ยังไม่ได้ขับ UI สร้าง/ลบเอง ผู้เรียนจะลองเอง)
- etcd ล่มนานกว่า 33 วินาที และการที่ gateway ตัวจริงกลับมาเองได้หลัง reboot เครื่อง (ตั้ง `unless-stopped` ให้ทั้ง etcd และ gateway ใน override แล้ว แต่ยังไม่ได้ reboot ทดสอบ)
- Docker บน Mac (ไม่มี)
