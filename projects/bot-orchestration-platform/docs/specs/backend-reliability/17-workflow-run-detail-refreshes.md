# 17: หน้ารายละเอียด Workflow Run ต้องอัปเดตเองจนกว่า run จะจบ

Spec: [backend-reliability.md](../backend-reliability.md)

**What to build:** เปิด `/workflow-runs/{id}` ขณะ run ยังรันอยู่ แล้วป้ายสถานะและ Pipeline เปลี่ยนเองเมื่อ run จบ (succeeded, failed, cancelled, terminated, timed out) โดยไม่ต้อง reload หน้า Job ทำแบบนี้อยู่แล้ว (ดึงข้อมูลใหม่ทุก 2 วินาทีจนกว่าจะจบ) ส่วนหน้านี้ไม่ทำ

**ที่มา (พบตอนไล่ทดสอบ dashboard ด้วย browser):** สั่งรัน workflow ที่ยิง URL ที่ไม่มีอยู่ พอหน้ารายละเอียดเปิดขึ้นมา run ยังเป็น `running` แล้วหน้าค้างที่ "running" และ "กำลังทำงาน..." เกิน 90 วินาที ทั้งที่ Temporal แสดง Failed แล้ว และหน้ารายการกับ Overview ก็แสดง failed ถูกต้อง สาเหตุคือ `useWorkflowRun` ไม่มี `refetchInterval`

**Blocked by:** None (can start immediately)

**Status:** done

**Verified:** deploy commit 5beb18b ขึ้น Mini PC แล้ว ตรวจด้วย Playwright: run ที่ค้างอยู่ (ชี้ไปที่อยู่ที่ไม่ตอบ ลอง 1 ครั้ง) แสดง `running` แล้วเปลี่ยนเป็น `failed` เองหลัง 30.9 วินาทีในหน้าเดิมโดยไม่ reload (ยืนยันด้วยตัวแปรที่ฝังไว้ในหน้า); vitest 63 ตัว, eslint, `next build` ผ่าน — 2026-11-21

- [x] ดึงข้อมูลของ run ใหม่ทุก 2 วินาที **เฉพาะ**เมื่อสถานะยังเป็น running และหยุดเมื่อจบ ตัดสินจากสถานะที่ผ่าน `normalizeRunStatus` (ไม่เทียบสตริงดิบ) ไม่ดึงซ้ำตอนยังโหลดไม่เสร็จหรือเกิด error
- [x] vitest: ฟังก์ชันที่ตัดสินว่าต้องดึงซ้ำหรือไม่ — running ได้ผลจริง สถานะที่จบแล้วทุกแบบ สถานะที่ไม่รู้จัก และไม่มีข้อมูล ได้ผลเท็จ
- [x] พิสูจน์กับ stack จริงด้วย Playwright: สั่งรัน workflow ที่ล้มเหลว (ตั้งลอง 2 ครั้ง) แล้วหน้ารายละเอียดเปลี่ยนจาก running เป็น failed เอง ไม่ reload ภายใน 30 วินาที
- [x] vitest, eslint และ `next build` ผ่าน
- [x] เอาหมายเหตุ "ข้อควรระวัง" เรื่องนี้ออกจาก `projects/bop-dashboard/UI-GUIDE.md` และแก้ตารางช่วงรีเฟรชอัตโนมัติ
- [x] orchestrator deploy branch แล้ว `bru run --env minipc` ผ่านทั้งชุด
