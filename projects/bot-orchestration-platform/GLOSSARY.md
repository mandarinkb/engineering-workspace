# Bot Orchestration Platform (BOP)

ระบบสั่งงาน bot ให้รันเดี่ยว รันเป็นลำดับขั้นตอน หรือรันตามตารางเวลา แบบ Control-M ขนาดเล็ก — ครอบคลุมทั้ง backend และ dashboard

## Language

### งานที่รัน

**Bot**:
หน่วยงานหนึ่งชนิดที่ระบบรู้จักและสั่งรันได้ พร้อมรายการ config ที่ต้องกรอก
_Avoid_: Task, worker

**Job**:
การสั่ง Bot หนึ่งตัวให้รันครั้งเดียวแบบเดี่ยวๆ ไม่ได้อยู่ใน Workflow
_Avoid_: Task, run (เมื่อหมายถึง Job)

**Workflow**:
ชุดของ Step ที่เรียงลำดับตายตัว รันต่อกันทีละ Step
_Avoid_: Pipeline, flow (Flow สงวนไว้สำหรับ Flow Control ใน Phase 3)

**Step**:
การรัน Bot หนึ่งครั้งในตำแหน่งหนึ่งของ Workflow — Step ไม่ใช่ Job
_Avoid_: Task, stage

**Workflow Run**:
การรัน Workflow หนึ่งครั้ง ไม่ว่าจะมาจาก Schedule หรือสั่งรันเองทันที
_Avoid_: Execution (ใช้เฉพาะเมื่อพูดถึงกลไกของ Temporal), run เฉยๆ

**Attempt**:
การลองรัน Step หนึ่งครั้ง — Step หนึ่งอาจมีหลาย Attempt ก่อนจะสำเร็จหรือล้มเหลว
_Avoid_: Try, retry (retry คือ Attempt ที่สองขึ้นไป)

**Failed Step**:
Step ที่ Attempt ครบจำนวนที่กำหนดแล้วยังไม่สำเร็จ หรือเจอข้อผิดพลาดที่ลองใหม่ก็ไม่หาย เช่น config ผิด — ทำให้ Workflow Run นั้นล้มเหลวและ Step ถัดไปไม่ถูกรัน

### ตารางเวลา

**Schedule**:
กฎที่สั่งให้ Workflow รันซ้ำตามช่วงเวลาหรือ cron expression
_Avoid_: Cron job, timer

**Missed Run**:
รอบของ Schedule ที่ถึงเวลาแล้วแต่ไม่ได้รัน เพราะระบบไม่ทำงานในขณะนั้น

**Catch-up Window**:
ระยะเวลาย้อนหลังที่ Missed Run ยังถูกรันชดเชยเมื่อระบบกลับมา — Missed Run ที่เก่ากว่านี้ถูกข้าม
_Avoid_: Grace period

**Backfill**:
การสั่งรัน Schedule ย้อนหลังในช่วงเวลาที่ผู้ใช้ระบุเอง — ต่างจาก Catch-up Window ที่เกิดอัตโนมัติ

### การเข้าถึง

**Temporal UI gateway**:
ด่านหน้า Temporal UI (APISIX) ที่ตรวจ session ของ BOP กับ `bop-api` ทุกคำขอ ผู้ที่ไม่มี session ถูกพาไปหน้า login ของ dashboard — ทำให้ Temporal UI เปิดได้เฉพาะผ่านผู้ใช้ที่ login BOP แล้ว
_Avoid_: Proxy (ไม่บอกว่าตรวจสิทธิ์), auth server

**Session**:
สถานะการ login ของผู้ใช้ BOP เก็บใน cookie `bop_session` ที่ `bop-api` ออกให้ และตรวจด้วย `GET /me`
