# คู่มือหน้า UI: BOP Dashboard และ Temporal UI

ว่าแต่ละเมนูและแต่ละปุ่มทำอะไร เขียนจากโค้ดของ dashboard (branch `feat/backend-reliability`, commit `4a38f0c`) และจากการเปิดหน้าจริงของทั้งสองระบบบน Mini PC เมื่อ 2026-11-21

| ระบบ | ที่อยู่ | เวอร์ชัน |
|---|---|---|
| BOP Dashboard | http://192.168.1.46:3000 | โค้ดใน `projects/bop-dashboard` |
| Temporal UI | http://192.168.1.46:8233 | 2.58.1 |

**วิธีอ่านส่วน Temporal UI:** ผมไม่ได้กดปุ่มทำงานของ Temporal UI เลย (ไม่กด Terminate, Delete, Reset, Pause, Start ฯลฯ) ทำแค่เปิดหน้า เปิดเมนูหรือกล่องข้อความแล้วปิด `✅` = เห็นปุ่มหรือข้อความนี้ในหน้าจริง ส่วนคำอธิบายว่าปุ่มทำอะไร มาจากชื่อปุ่ม ข้อความในกล่อง และความรู้ทั่วไปของ Temporal ไม่ได้มาจากการกดดูผล · `📘` = ไม่มีหลักฐานจากหน้าจริงนอกจากว่ามีปุ่มนั้น คำอธิบายมาจากความรู้ทั่วไปล้วนๆ

---

# ส่วนที่ 1 — BOP Dashboard

## แผนผังหน้า

```text
/login                         เข้าสู่ระบบ
/overview                      หน้าแรกหลัง login (สรุปภาพรวม)
/bots                          รายการ Bot และสั่งรัน Job
/jobs                          ประวัติ Job
  └─ /jobs/{id}                รายละเอียดและ log ของ Job
/schedules                     รายการ Schedule
  ├─ /schedules/new            สร้าง Schedule
  ├─ /schedules/{id}           รายละเอียดและปุ่มจัดการ
  └─ /schedules/{id}/edit      แก้ไข Schedule
/workflow-runs                 ประวัติ Workflow Run
  ├─ /workflow-runs/new        สั่งรัน Workflow ครั้งเดียว
  └─ /workflow-runs/{id}       รายละเอียดของ run
```

- เข้า `/` จะถูกพาไป `/overview` ถ้ายังไม่ login จะถูกพาไป `/login`
- ทุกหน้าต้อง login ยกเว้น `/login` และเมื่อ session หมดอายุ (API ตอบ 401) จะถูกพากลับ `/login` เอง
- **Job** กับ **Workflow Run** ต่างกัน: Job คือการสั่ง Bot ตัวเดียวรันครั้งเดียว เก็บใน PostgreSQL ของ BOP ไม่ผ่าน Temporal ส่วน Workflow Run คือการรัน Workflow ที่มีหนึ่งหรือหลาย Step เรียงกัน รันผ่าน Temporal จึงไปดูต่อใน Temporal UI ได้

## 1.1 แถบเมนูซ้าย (Sidebar) — ทุกหน้าหลัง login

| ปุ่ม / ส่วน | หน้าที่ |
|---|---|
| `BOP` + ปุ่ม `‹` / `☰` | พับเมนูให้เหลือแค่ไอคอน (กว้าง 64px) หรือกางกลับ (224px) จำสถานะไว้ใน cookie `bop_sidebar` นาน 1 ปี ตอนพับ เอาเมาส์ชี้ไอคอนจะเห็นชื่อ |
| Overview | หน้าสรุปภาพรวม |
| Bots | รายการ Bot และสั่งรัน Job |
| Jobs | ประวัติ Job |
| Schedules | รายการ Schedule |
| Workflow Runs | ประวัติ Workflow Run |
| Temporal UI ↗ | เปิด Temporal UI ในแท็บใหม่ (ที่อยู่ตั้งตอน build ผ่าน `NEXT_PUBLIC_TEMPORAL_UI_URL`) |
| จุดสี + ข้อความสถานะ | เรียก `/readyz` ทุก 15 วินาที: **เขียว** "ระบบปกติ" · **แดง** "ระบบไม่พร้อม (Temporal/ฐานข้อมูล)" (API ตอบแต่ต่อ Temporal หรือฐานข้อมูลไม่ได้) · **แดง** "เรียก API ไม่ติด" · **เทา** "กำลังเช็คระบบ..." |
| ชื่อ user (วงกลมตัวอักษรแรก) | แสดง user ที่ login อยู่ |
| ธีมมืด / ธีมสว่าง | สลับโหมดมืด/สว่าง จำค่าไว้ในเบราว์เซอร์ |
| ออกจากระบบ | ล้าง session แล้วกลับ `/login` |

เมนูของหน้าที่เปิดอยู่จะถูกไฮไลต์

## 1.2 Login (`/login`)

| ส่วน | หน้าที่ |
|---|---|
| Username / Password | ช่องกรอก ถ้าเว้นว่างจะขึ้น "กรอก username" / "กรอก password" |
| เข้าสู่ระบบ | ส่งไปตรวจที่ backend ถ้าผิดขึ้น `invalid credentials` ถ้าถูกพาไป `/overview` |

ถ้า login ค้างอยู่แล้วเปิด `/login` จะถูกพาไป `/overview` ทันที ค่าเริ่มต้นของระบบ dev คือ `admin` / `changeme` (ตั้งใน compose ห้ามใช้จริง)

## 1.3 Overview (`/overview`)

ข้อมูลรีเฟรชเองทุก 10 วินาที

| ส่วน | ความหมาย |
|---|---|
| การ์ด **Job กำลังรัน** | จำนวน Job สถานะ running (เหลืองเมื่อมากกว่า 0) |
| การ์ด **Job ล้มเหลววันนี้** | Job ที่ failed และสร้างวันนี้ (แดงเมื่อมากกว่า 0) |
| การ์ด **Success rate 7 วัน** | สัดส่วน Job ที่สำเร็จ จากเฉพาะ Job ที่จบแล้วในรอบ 7 วัน เขียวถ้าตั้งแต่ 80% แดงถ้าต่ำกว่า `-` = ยังไม่มีข้อมูล |
| การ์ด **Workflow กำลังรัน** | จำนวน Workflow Run ที่ running (นับจาก server) |
| กล่อง **Schedule ที่จะรันถัดไป** | Schedule ที่ active เรียงตามรอบถัดไป คลิกแถวเข้าหน้ารายละเอียด `ดูทั้งหมด` ไป `/schedules` |
| กล่อง **Workflow run ที่ล้มเหลวล่าสุด** | 5 run ล่าสุดที่ failed / terminated / timed out คลิกแถวเข้าหน้ารายละเอียด `ดูทั้งหมด` ไป `/workflow-runs` |

ถ้าดึงข้อมูลบางส่วนไม่สำเร็จ จะมีแถบแดงบอกว่าตัวเลขอาจไม่ครบ

## 1.4 Bots (`/bots`)

| ส่วน | หน้าที่ |
|---|---|
| ช่อง `ค้นหา bot...` | กรองรายการ Bot ตามชื่อ |
| การ์ด Bot (ชื่อ + "N config fields") | คลิกเพื่อเปิดฟอร์มสั่งรัน Bot นั้น (เปิดทีละตัว) |
| ฟอร์ม **รัน {ชื่อ bot}** | ช่องกรอกตาม config ของ Bot (เช่น `http-status-checker` มี URL ที่ต้องกรอก `external-job` มี Job Code และ Extra Arguments) ช่องที่มี `*` บังคับกรอก |
| ปุ่ม **รัน {ชื่อ bot}** | สร้าง Job ใหม่แล้วพาไป `/jobs/{id}` |

## 1.5 Jobs (`/jobs`)

รีเฟรชเองทุก 10 วินาที

| ส่วน | หน้าที่ |
|---|---|
| การ์ด Job ทั้งหมด / Success Rate / วันนี้ / ล้มเหลวทั้งหมด | สรุปตัวเลขจากรายการ Job |
| ช่อง `กรองตามชื่อ bot...` | กรองตามชื่อ Bot |
| ตัวเลือกสถานะ | ทุกสถานะ · pending · running · succeeded · failed · cancelled |
| วันที่ **จาก / ถึง** | กรองตามวันที่สร้าง |
| (ตัวกรองทั้งหมด) | เก็บไว้ใน URL จึงแชร์ลิงก์หรือ reload แล้วกรองเหมือนเดิม |
| ตาราง Bot · Status · Summary · เริ่มเมื่อ | คลิกชื่อ Bot เพื่อเข้าหน้ารายละเอียดของ Job |
| ช่องติ๊กหน้าแถว | เลือก Job เพื่อยกเลิกทีละหลายตัว Job ที่ยกเลิกไม่ได้ (ไม่ใช่ pending หรือ running) ช่องจะแสดงแต่ถูกปิดไว้ |
| ช่องติ๊กหัวตาราง | เลือกทุก Job ที่ยกเลิกได้ในหน้านี้ |
| `เลือกไว้ N งาน` + **ยกเลิกที่เลือก** | สั่งยกเลิก Job ที่เลือกทั้งหมด (ใน dashboard ปุ่มนี้คือที่เดียวที่ยกเลิก Job ได้ หน้ารายละเอียดไม่มีปุ่มยกเลิก) |
| **ก่อนหน้า / ถัดไป** + `หน้า X / Y` | แบ่งหน้า หน้าละ 20 แถว |

## 1.6 Job detail (`/jobs/{id}`)

หน้านี้ดึงสถานะใหม่ทุก 2 วินาทีขณะ Job ยัง pending หรือ running

| ส่วน | หน้าที่ |
|---|---|
| สถานะของ Job | pending / running / succeeded / failed / cancelled พร้อม summary หรือ error |
| ป้ายการเชื่อมต่อ | "กำลังเชื่อมต่อ..." · "เชื่อมต่อแล้ว (real-time)" · "หลุดการเชื่อมต่อ กำลังลองเชื่อมใหม่..." (log ไหลเข้ามาเองแบบ SSE) |
| ปุ่ม **ทั้งหมด / Info / Warn / Error** | กรอง log ตามระดับ |
| ช่อง `ค้นหาใน log...` | ค้นข้อความใน log ที่โหลดมาแล้ว |
| ช่องติ๊ก **Auto-scroll** | เปิด = เลื่อนตาม log ใหม่อัตโนมัติ ปิดเมื่ออยากอ่าน log เก่าโดยไม่ให้จอเลื่อนหนี |
| **ดาวน์โหลด .txt** | ดาวน์โหลด log ที่เห็นอยู่เป็นไฟล์ข้อความ |

log ของ Job เก็บใน PostgreSQL นาน 30 วัน หลังจากนั้นเหลือแค่สถานะและ summary

## 1.7 Schedules (`/schedules`)

| ส่วน | หน้าที่ |
|---|---|
| **+ สร้าง schedule ใหม่** | ไปฟอร์มสร้าง (`/schedules/new`) |
| ตาราง ID · Workflow · สถานะ · ตารางเวลา · รอบถัดไป · การจัดการ | คลิก ID เข้าหน้ารายละเอียด ตารางเวลาแสดง "ทุก N วินาที" หรือ cron |
| ปุ่ม **Run Now** (ท้ายแถว) | สั่งรันทันที นอกตารางเวลาปกติ |
| ปุ่ม **Pause / Unpause** (ท้ายแถว) | หยุดหรือเปิด Schedule ต่อ |

## 1.8 ฟอร์มสร้างและแก้ไข Schedule

| ช่อง / ปุ่ม | หน้าที่ |
|---|---|
| **ID** | ชื่ออ้างอิงของ Schedule ตั้งตอนสร้างเท่านั้น แก้ไขภายหลังไม่ได้ (ถ้าจะเปลี่ยนต้องลบแล้วสร้างใหม่) |
| **Workflow Name** | ชื่อของ workflow ใช้เป็นคำนำหน้า ID ของ run ที่ Schedule สั่ง |
| **ตารางเวลา** | เลือก `ทุก N วินาที` (กรอกจำนวนวินาที) หรือ `Cron expression` (เช่น `0 9 * * MON-FRI`) |
| **Overlap Policy** | ถ้ารอบใหม่มาถึงตอนรอบเก่ายังรันไม่จบ: `skip` ข้ามรอบใหม่ · `buffer_one` เก็บไว้ 1 รอบแล้วรันตามหลัง · `buffer_all` เก็บทุกรอบ · `cancel_other` ยกเลิกรอบเก่า · `terminate_other` หยุดรอบเก่าทันที · `allow_all` รันซ้อนกันได้ |
| **Catch-up Window (วินาที)** | ถ้าระบบหยุดไปแล้วพลาดรอบไว้ จะรันชดเชยเฉพาะรอบที่พลาดไม่เกินช่วงนี้ ค่าเริ่มต้น 3600 (1 ชั่วโมง) ต่ำสุด 10 สูงสุด 31,536,000 (1 ปี) |
| **Steps** | ขั้นตอนที่จะรัน (ดูด้านล่าง) |
| **สร้าง schedule** / **บันทึกการแก้ไข** | บันทึก แล้วไปหน้ารายละเอียดของ Schedule |

**Steps editor** (ใช้ร่วมกับฟอร์ม Run workflow):

| ช่อง / ปุ่ม | หน้าที่ |
|---|---|
| เลือก bot | Bot ของ step นี้ เปลี่ยน Bot จะล้างค่า config เดิมของ step |
| ช่อง config ของ Bot | ขึ้นตามที่ Bot ต้องการ (`*` = บังคับ) |
| **จำนวนครั้งที่ลองสูงสุด** | ลอง step นี้ได้กี่ครั้งก่อนนับว่าล้มเหลว เว้นว่าง = 3 ต้องเป็นจำนวนเต็มตั้งแต่ 1 |
| **รันที่ไหน** | `Local Bot` รันใน worker ของ BOP · `Remote Task Queue` ส่งไปรันที่ worker อื่นผ่าน Task Queue ที่กรอก · `Kubernetes Job` สั่งสร้าง K8s Job (ต้องกรอก Image บังคับ Args คั่นด้วยจุลภาค Namespace ไม่บังคับ) |
| **ลบ step นี้** | ลบ step (มีเมื่อมีมากกว่า 1 step) |
| **+ เพิ่ม step** | เพิ่ม step ต่อท้าย steps รันเรียงตามลำดับ ถ้า step ใดล้มเหลว step หลังจากนั้นไม่ถูกรัน |

> **ข้อควรระวัง (ticket 13 — ยังไม่แก้):** ฟอร์ม **แก้ไข** เติมช่อง Workflow Name เป็น `Execute` ทุกครั้ง แทนชื่อที่ตั้งไว้ตอนสร้าง ถ้ากดบันทึกโดยไม่แก้ชื่อ ชื่อ workflow ของ Schedule จะถูกเปลี่ยนเป็น `Execute` และ run ถัดไปจะมี ID ขึ้นต้น `Execute-` (ยืนยันแล้วด้วยการลองจริง) ระหว่างรอแก้ ให้พิมพ์ชื่อเดิมกลับก่อนบันทึก

## 1.9 Schedule detail (`/schedules/{id}`)

หัวหน้าแสดง ID, สถานะ (active/paused), Workflow, Overlap และ Catch-up Window

| ปุ่ม / ส่วน | หน้าที่ |
|---|---|
| **Run Now** | สั่งรันทันที 1 ครั้งนอกตารางเวลา ไม่กระทบรอบปกติ |
| **Pause** / **Unpause** | หยุด/เปิด Schedule ต่อ (จะแสดงอย่างใดอย่างหนึ่งตามสถานะ) |
| **แก้ไข** | ไปฟอร์มแก้ไข |
| **Backfill** | เปิดช่องกรอก `เริ่ม` และ `สิ้นสุด` แล้วกด **ยืนยัน backfill** เพื่อรัน Schedule ย้อนหลังตามช่วงเวลานั้น เหมือนเวลานั้นเพิ่งผ่านไปตอนนี้ (เทียบเท่า Rerun) ไม่ได้ถูกจำกัดด้วย Catch-up Window |
| **ลบ** | ลบ Schedule ถาวร มีกล่องยืนยันก่อน ย้อนกลับไม่ได้ |
| **Steps** | steps ของ Schedule (อ่านจาก Temporal) ถ้าเป็น Schedule ที่สร้างนอก BOP (เช่นจาก Temporal UI) อาจไม่มีข้อมูลนี้ |
| **ประวัติการรันล่าสุด** | เวลาที่รันจริงและ workflow ID ของแต่ละรอบ |

## 1.10 Workflow Runs (`/workflow-runs`)

รีเฟรชเองทุก 10 วินาที

| ส่วน | หน้าที่ |
|---|---|
| **+ Run workflow** | ไปฟอร์มสั่งรันครั้งเดียว |
| ตัวกรอง **กรองตาม Schedule** | เลือก Schedule เพื่อดูเฉพาะ run ที่ Schedule นั้นสั่ง `ทุก Schedule (รวมที่สั่งเอง)` = ไม่กรอง |
| ตัวกรอง **กรองตามสถานะ** | running · succeeded · failed / terminated / timed out · cancelled |
| `N run ที่โหลดแล้ว` | จำนวนแถวที่โหลดมา และต่อท้ายว่า "(ครบแล้ว)" เมื่อไม่มีเหลือ |
| ตาราง Workflow ID · Schedule · Status · เริ่มเมื่อ | คลิก Workflow ID เข้าหน้ารายละเอียด คอลัมน์ Schedule เป็น "สั่งเอง" ถ้า run ไม่ได้มาจาก Schedule |
| **โหลดเพิ่ม** | โหลด run หน้าถัดไปอีก 50 ตัว (ตัวกรองทำที่ server) |

## 1.11 Run workflow (`/workflow-runs/new`)

สั่งรัน steps ทันทีครั้งเดียว โดยไม่ต้องสร้าง Schedule

| ช่อง / ปุ่ม | หน้าที่ |
|---|---|
| **Workflow Name** | ใช้เป็นคำนำหน้า workflow ID ของ run (เช่น `manual-healthcheck`) จะได้ค้นเจอในหน้า Workflow Runs |
| **Steps** | steps editor เหมือนหัวข้อ 1.8 |
| **รันเลย** | เริ่ม run แล้วพาไปหน้ารายละเอียดของ run นั้น |

## 1.12 Workflow Run detail (`/workflow-runs/{id}`)

| ปุ่ม / ส่วน | หน้าที่ |
|---|---|
| ป้ายสถานะ + "เริ่มเมื่อ" | สถานะของ run (running, succeeded, failed, terminated, timed out, cancelled) |
| **เปิดใน Temporal ↗** | เปิดประวัติ event ของ run นี้ใน Temporal UI ในแท็บใหม่ |
| **Pipeline** | ผังแนวนอน แต่ละกล่องคือ 1 step (ลำดับ, สถานะ, ชื่อ Bot, summary หรือ error) ถ้ายังรันอยู่จะมีกล่อง "กำลังทำงาน..." ต่อท้าย |
| **List** | แสดงผลของแต่ละ step เป็นรายการ อ่านง่ายกว่าเมื่อ summary หรือ error ยาว |

> **ข้อควรระวัง (ยังไม่มี ticket):** หน้านี้ไม่ดึงข้อมูลใหม่เอง ถ้าเปิดตอน run ยังรันอยู่ สถานะจะค้างที่ "running" แม้ run จบแล้ว ต้อง reload หน้าเอง (หน้า Job ดึงใหม่ทุก 2 วินาที แต่หน้านี้ไม่) ยังไม่ได้เปิดเป็น ticket

## 1.13 ช่วงรีเฟรชอัตโนมัติ

| หน้า | ทุกกี่วินาที |
|---|---|
| Overview, Jobs, Schedules, Workflow Runs (รายการ) | 10 (หยุดตอนแท็บไม่ได้เปิดอยู่) |
| Job detail | 2 (เฉพาะตอน Job ยังไม่จบ) |
| ไฟสถานะระบบใน sidebar | 15 |
| Workflow Run detail | ไม่รีเฟรช |

---

# ส่วนที่ 2 — Temporal UI (2.58.1)

Temporal คือระบบที่ BOP ใช้สั่งและเก็บประวัติ Workflow Run กับ Schedule ใช้ดูรายละเอียดระดับลึกกว่า BOP เช่น event ทุกตัวของ run และจำนวนครั้งที่ลอง Activity ที่อยู่ของ BOP ใน Temporal: namespace `default`, task queue `bop-workflows`, workflow type `Execute`

## 2.1 แถบด้านบนและเมนูซ้าย ✅

| ปุ่ม / ส่วน | หน้าที่ |
|---|---|
| Collapse Navigation | พับเมนูซ้าย |
| `local` (ใกล้ "Select a Namespace") | แสดงชื่อ namespace/environment ที่กำลังดู ตอนลองกดไม่ตอบสนอง จึงไม่ยืนยันว่าเป็น dropdown |
| Configure Codec Server | กล่องตั้งค่าว่าจะถอดรหัส payload ผ่าน Codec Server อย่างไร ตัวเลือก: ใช้ค่าระดับ cluster ("Use Cluster-level setting, where available.") หรือใช้ค่าในเบราว์เซอร์และไม่สนค่า cluster ("Use my browser setting and ignore Cluster-level setting.") มี Cancel / Apply |
| Open news feed | เปิดแผงข่าว มีตัวเลือก "Automatically fetch news" และปุ่ม Close |
| user-menu (Anonymous Tardigrade) | เมนู user ไม่ได้ผูกกับบัญชีจริง (anonymous) มีลิงก์ Slack Community, Community Forum, Change Log |
| System Default / Day / Night | สลับธีม |
| **Namespaces** | `/namespaces` รายการ namespace (ระบบนี้มี `default` ตัวเดียว) เลือกจำนวนต่อหน้าได้ 100/250/500/1000 |
| **Workflows** | รายการ Workflow ทั้งหมด (หัวข้อ 2.2) |
| **Schedules** | รายการ Schedule (หัวข้อ 2.5) |
| **Agents** | หน้าแนะนำฟีเจอร์ Temporal สำหรับ AI agent: "Agent Harness" (EXPERIMENTAL) และ "MCP Server" (PRE-RELEASE) พร้อมลิงก์ GitHub ไม่มีปุ่มใช้งานและไม่เกี่ยวกับ BOP |
| **Batch** | รายการงาน batch (หัวข้อ 2.6) |
| **Workers** | รายการ worker (หัวข้อ 2.7) |
| **Nexus** | หน้า "NEXUS ENDPOINTS / Get Started" อธิบาย Temporal Nexus ซึ่งเชื่อม Temporal Application ข้ามทีมหรือ namespace ระบบนี้ไม่ได้ตั้ง endpoint ไว้ |
| **Archive** | ขึ้นว่า "This namespace is currently not enabled for archival" พร้อมคำสั่งเปิดการเก็บถาวร (archival) ระบบนี้ยังไม่เปิด |
| **Import** | อัปโหลดไฟล์ JSON ของ event history (รูปแบบ `{"events": HistoryEvent[]}` หรือ `HistoryEvent[]`) เพื่อเปิดดูใน UI |
| Docs / Feedback | ลิงก์ออกไป docs.temporal.io และหน้าแจ้งปัญหาบน GitHub ของ Temporal UI |

## 2.2 Workflows (`/namespaces/default/workflows`) ✅

หัวหน้าแสดงจำนวน workflow และสรุปตามสถานะ (เช่น "684 COMPLETED")

| ปุ่ม / ส่วน | หน้าที่ |
|---|---|
| Refresh | โหลดรายการใหม่ |
| **Start Workflow** | ไปฟอร์มเริ่ม workflow ใหม่ (หัวข้อ 2.4) |
| **All / Running / Parent / Scheduled / Today / Last 1h** | ตัวกรองด่วน: ทั้งหมด · `ExecutionStatus="Running"` · ไม่มี parent (`ParentWorkflowId is null`) · มาจาก Schedule (`TemporalScheduledById is not null`) · เริ่มตั้งแต่เที่ยงคืนวันนี้ · เริ่มภายใน 1 ชั่วโมงล่าสุด |
| Add Filter | เปิดรายการ search attribute ให้เลือกมาเป็นเงื่อนไข เช่น ExecutionStatus, WorkflowId, WorkflowType, RunId, StartTime, CloseTime, TaskQueue, HistoryLength และ attribute กำหนดเอง |
| Status / Workflow ID / Run ID / Type / Start / End | ตัวกรองรายคอลัมน์ ใช้ค้นตามค่านั้นโดยตรง |
| toggle-manual-query | สลับไปพิมพ์ query เอง แทนการเลือกทีละตัว ตัวอย่างที่ใช้ได้: `WorkflowId STARTS_WITH "bruno-"` และ `TemporalScheduledById = "check-example-com-schedule"` |
| Saved Views `0 / 50` | บันทึกชุดตัวกรองไว้ใช้ซ้ำ สูงสุด 50 ชุด ตอนนี้ขึ้น "No saved views yet" |
| ช่องติ๊กหน้าแถว / "Select all Workflows" | เลือก workflow เพื่อทำงานเป็นชุด |
| แถบเมื่อเลือกแถว: **Request Cancellation / Reset / Terminate** | ทำกับทุก workflow ที่เลือก: ขอให้ยกเลิก (ปิดไว้ถ้า workflow จบแล้ว) · ย้อนเริ่มใหม่จากจุดหนึ่ง · หยุดทันที 📘 งานชุดนี้จะไปโผล่ในหน้า Batch |
| Start Workflow Like This One | เปิดฟอร์มเริ่ม workflow โดยเติม Workflow ID, Task Queue และ Type จากแถวที่เลือกให้ |
| Comfortable | ปรับความหนาแน่นของตาราง |
| Download JSON | ส่งออกรายการที่เห็น 📘 |
| Configure Columns | จัดคอลัมน์: เลื่อนขึ้น/ลง ลบ หรือเพิ่ม: History Size, History Length, Execution Time, Execution Duration, State Transitions, Task Queue, Scheduled By ID, Scheduled Start Time, Deployment, Deployment Version, Versioning Behavior |
| Maximize | ขยายตารางเต็มจอ 📘 |
| Previous / Next | เลื่อนหน้า |

คลิก Workflow ID เพื่อเข้าหน้ารายละเอียด

## 2.3 Workflow detail ✅

หัวหน้าแสดงสถานะ (เช่น COMPLETED), Workflow ID, เวลาเริ่ม/จบ, Duration, Run ID, Workflow Type, Task Queue, **Scheduled By** (ชื่อ Schedule ที่สั่ง ถ้ามี), History Size, State Transitions และ SDK ที่ใช้

| ปุ่ม / แท็บ | หน้าที่ |
|---|---|
| Reset | เปิดกล่อง "Reset Workflow": เลือก event ที่จะย้อนไปเริ่มใหม่ (Choose an Event to reset to) และตัวเลือกว่าจะตัด Signal หรือ Update ที่เกิดหลังจุดนั้นไหม จะสร้าง run ใหม่จากจุดนั้น |
| More Actions | สำหรับ workflow ที่จบแล้ว มีแค่ "Start Workflow Like This One" 📘 workflow ที่ยังรันอยู่จะมีเมนูเพิ่ม เช่น ยกเลิกและ Terminate ผมไม่ได้เห็นเพราะ run ของ BOP จบเร็วเกินไป |
| Preview / Maximize Input, Preview / Maximize Result | ดู input และ result ของ workflow (แบบย่อหรือเต็มจอ) |
| **Timeline** | เส้นเวลาของ activity แต่ละตัว (ใน BOP เห็นแท่ง `RunStep` คือ 1 Step) มีปุ่ม Show idle time และ Show time segment |
| **Event History (N)** | รายการ event ทุกตัวของ run ตัวควบคุม: **All / Compact / JSON** (รูปแบบการแสดง) · **Descending** (สลับลำดับ) · **Filter** · **Auto Refresh** (ปิดไว้ถ้า workflow จบแล้ว) · **Download** (ดาวน์โหลดประวัติ) · **Expand details** (ขยายรายละเอียดแต่ละ event) |
| Relationships (N) | ความสัมพันธ์ parent / child workflow 📘 |
| Workers | worker ที่รับงานของ task queue นี้ |
| Pending Activities (N) | activity ที่ยังรันหรือรอลองใหม่ (ถ้ามีจะเห็นจำนวน Attempt) |
| Call Stack | ขอ stack trace ของ workflow ที่กำลังรัน 📘 |
| Queries | ส่ง query ไปอ่านสถานะ workflow ของ BOP มี query ชื่อ `steps` ซึ่งคืนผลของแต่ละ step |
| User Metadata | ข้อความสรุปที่ผู้ใช้ตั้งไว้กับ workflow 📘 |
| Search Attributes | ค่า search attribute ของ run (เช่น TemporalScheduledById) |
| Memo | ข้อมูล memo ที่แนบกับ run |

**วิธีดู run ที่ล้มเหลวของ BOP:** เปิดที่แท็บ Event History มองหา `Activity Task Failed` และ `Workflow Execution Failed` Event ท้ายสุดของ activity บอกหมายเลข Attempt และ `retryState` เช่น `RETRY_STATE_MAXIMUM_ATTEMPTS_REACHED` แปลว่าลองครบ `max_attempts` แล้ว

## 2.4 Start Workflow ✅

| ช่อง / ปุ่ม | หน้าที่ |
|---|---|
| Workflow ID * + **Random UUID** | ID ของ workflow สุ่มใหม่ให้ด้วยปุ่ม Random UUID |
| Task Queue * | task queue ที่จะส่งงานไป ใต้ช่องแสดงว่า "Task Queue is Active · 1 Worker" และปุ่ม **View Task Queue** |
| Workflow Type * | ชื่อ type ของ workflow (ของ BOP คือ `Execute`) |
| Input Data (**Add / Edit / Preview**) + encoding | ใส่ input เป็น JSON เลือก encoding ได้ |
| More options | ตัวเลือกเพิ่ม (timeout ฯลฯ) 📘 |
| **Start Workflow** | เริ่ม workflow จริง |

> ถ้าเริ่ม `Execute` เองจากหน้านี้ ต้องใส่ input ให้ตรงรูปแบบของ BOP (`{"name": ..., "steps": [...]}`) ไม่เช่นนั้น workflow จะล้มเหลว ปกติสั่งผ่านหน้า **Run workflow** ของ BOP จะปลอดภัยกว่า

## 2.5 Schedules ✅

**รายการ (`/namespaces/default/schedules`)**

| ปุ่ม / ส่วน | หน้าที่ |
|---|---|
| Create Schedule | ไปฟอร์มสร้าง Schedule |
| All / Paused | กรองตามสถานะ |
| Add Filter, Saved Views, Filter by Schedule ID | กรองและบันทึกชุดตัวกรอง |
| คอลัมน์ Status · Schedule ID · Workflow Type · Recent Runs · Upcoming Runs · Schedule Spec | ข้อมูลของแต่ละ Schedule คลิกชื่อเพื่อเข้าหน้ารายละเอียด |

**ฟอร์ม Create Schedule:** Schedule Name * · Schedule Start Date / End Date (Never / On / After) · Timezone * · Jitter · Schedule Spec Type * (Cron Shortcuts: Every minute / Every hour / Daily / Every Saturday / Monthly…) · Cron expression * · `+ Add another schedule spec` · Edit Schedule Policies · Workflow Type * · Workflow ID · Task Queue * · encoding (json/plain, json/protobuf) · Task / Run / Execution Timeout · search attribute · ปุ่ม **Create Schedule**

> Schedule ที่สร้างจาก Temporal UI ไม่มี steps ในรูปแบบที่ BOP อ่านได้ ในหน้ารายละเอียดของ BOP จะเห็น steps ว่าง ถ้าจะใช้กับ BOP ให้สร้างจากหน้า Schedules ของ BOP

**หน้ารายละเอียด Schedule**

| ปุ่ม / ส่วน | หน้าที่ |
|---|---|
| สถานะ (RUNNING / PAUSED) | สถานะปัจจุบัน |
| **Pause** | หยุด Schedule ต้องกรอกช่อง **Reason** (เหตุผล) แล้วยืนยัน (ผมกดเปิดกล่องแล้วปิดโดยไม่ยืนยัน Schedule ยัง RUNNING) |
| **Schedule Actions** | เมนู **Trigger** (รันทันที) · **Backfill** (รันย้อนหลังช่วงที่กำหนด) · **Edit** (แก้ไข) · **Delete** (ลบถาวร) |
| Workflow Type / Created / Last Updated / Total Workflows | ข้อมูลของ Schedule และจำนวน workflow ที่รันไปแล้ว |
| View all N runs | ไปหน้ารายการ workflow ที่กรองเฉพาะ run ของ Schedule นี้ |
| **5 Recent Runs / 5 Upcoming Runs** | 5 run ล่าสุดที่รันไป และ 5 รอบถัดไป |
| View Full Spec / Hide Full Spec | แสดงหรือซ่อนรายละเอียดตารางเวลาทั้งหมด |
| Advanced Settings | นโยบายของ Schedule เช่น Overlap Policy, Catchup Window, Pause On Failure 📘 |
| Custom Search Attributes | search attribute ที่กำหนดเอง (ตอนนี้ 0) |

## 2.6 Batch (`/namespaces/default/batch-operations`) ✅

รายการงานที่ทำกับ workflow หลายตัวพร้อมกัน เช่นการเลือกหลายแถวแล้ว Terminate หรือสั่งลบ ตารางมีคอลัมน์ Status · Job ID · Operation Type · Start Time · Close Time ตัวอย่างในระบบนี้คืองานลบข้อมูลทดสอบของ BOP ที่สั่งผ่าน CLI

## 2.7 Workers และ Task Queue ✅

- **Workers (`/namespaces/default/workers`):** มีแท็บ **Instances** และ **Deployments** กับปุ่ม **Create Worker Deployment** และ Refresh ในระบบนี้หน้า Instances ไม่แสดง worker ของ BOP ผมสงสัยว่าเป็นเพราะ Temporal server เวอร์ชันนี้ไม่รองรับ worker heartbeat (log ของ worker เขียนไว้ว่า "server version does not support it") แต่ยังไม่ได้ตรวจเนื้อหาหน้านี้เต็มๆ
- **Task Queue (`/namespaces/default/task-queues/bop-workflows`):** แสดง "Workers 1" และตาราง poller: ID · Build ID · Deployment · Last Accessed · Workflow Task Handler · Activity Handler · Nexus Handler ใช้ยืนยันว่า worker ของ BOP กำลังรอรับงานอยู่

## 2.8 BOP เทียบกับ Temporal

| ใน BOP | ใน Temporal UI |
|---|---|
| Workflow Run | Workflow ใน namespace `default` มี Workflow Type `Execute` และ Task Queue `bop-workflows` |
| Workflow ID ของ run | คอลัมน์ Workflow ID ใช้ค้นใน Workflows ได้ |
| Step ใน Workflow Run | Activity ชื่อ `RunStep` (บน Timeline และใน Event History) |
| Attempt / จำนวนครั้งที่ลอง | หมายเลข Attempt ของ activity ใน Event History และ Pending Activities |
| Schedule | หน้า Schedules ใช้ Schedule ID เดียวกัน |
| Catch-up Window / Overlap Policy | ใน Advanced Settings ของหน้ารายละเอียด Schedule |
| run ที่มาจาก Schedule | ช่อง **Scheduled By** ในหน้ารายละเอียด และ search attribute `TemporalScheduledById` |
| ปุ่ม "เปิดใน Temporal ↗" | ลิงก์ไปหน้า `/namespaces/default/workflows/{id}/{runId}/history` |
| Job (สั่ง Bot ตัวเดียว) | **ไม่มี** ใน Temporal เลย เพราะ Job ไม่ผ่าน Workflow เก็บใน PostgreSQL ของ BOP |

## 2.9 ข้อควรระวังเวลาแก้ใน Temporal UI

BOP อ่านข้อมูลของ Schedule และ Workflow Run จาก Temporal โดยตรง การแก้ใน Temporal UI จึงน่าจะส่งผลกับ BOP ทันที (ข้อต่อไปนี้อนุมานจากการที่ BOP อ่านจาก Temporal ผมไม่ได้ลองทีละข้อ เพราะต้องทำลายข้อมูล):

- **ลบ Schedule** ใน Temporal (Schedule Actions → Delete) → หายจากหน้า Schedules ของ BOP ทันที
- **ลบ workflow** → run นั้นหายจากหน้า Workflow Runs ของ BOP และจากการนับใน Overview
- **Terminate / Request Cancellation** → สถานะ run เปลี่ยนเป็น terminated/cancelled ใน BOP
- **Pause / Edit Schedule** ใน Temporal → ไปเปลี่ยนสถานะหรือการตั้งค่าที่ BOP แสดง ถ้าแก้ Edit อาจทำให้ steps ไม่อยู่ในรูปแบบที่ BOP อ่านได้
- ปุ่มที่ทำลายข้อมูลทั้งหมด (Terminate, Delete, Reset) ย้อนกลับไม่ได้ ควรทำเมื่อรู้ว่าทำอะไรอยู่
