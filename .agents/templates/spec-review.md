# Spec review: <ticket> รอบ <n>

ใช้ทั้ง `r<n>/spec-review.md` (ticket เสี่ยง) และ `r<n>/lead-review.md` (ticket ปกติ)

- **Ticket:** <path ของ ticket>
- **Revision:** <commit SHA / ช่วง commit / ชื่อไฟล์ diff ที่ตรวจ>
- **Verdict:** <PASS | PASS (มี AC ตรวจไม่ได้ n ข้อ: AC-x) | FAIL> — FAIL เฉพาะเมื่อมี AC "ไม่ตรง"; AC ที่ตรวจไม่ได้ใน session นี้ไม่ทำให้เป็น FAIL แต่ต้องระบุ
- **Bruno coverage:** <PASS | FAIL | n/a (ไม่มี Bruno ใน change นี้)>
- **Reviewer:** <spec-reviewer | lead>

## ทีละ criterion

| AC | ticket / spec กำหนด | โค้ดทำ (file:line) | ตรง? |
|---|---|---|---|
| AC-1 | | | ตรง / ไม่ตรง / ตรวจไม่ได้ใน session นี้ (ระบุสิ่งที่ต้องใช้) |

## คุณภาพ test / Bruno

| ไฟล์ | assert อะไร | จะแดงไหมถ้า subject คืนค่าว่างหรือพฤติกรรมเก่า | หมายเหตุ |
|---|---|---|---|

## นอก ticket

<สิ่งที่ change ทำเกินหรือขาดจาก ticket, ปัญหาความปลอดภัย, ไฟล์นอก scope หรือ "ไม่มี">

## Fix items

<ใช้รูปแบบ templates/fix-task.md ต่อข้อ หรือ "ไม่มี">

## เรื่องที่ต้องให้ user ตัดสิน

<หรือ "ไม่มี">

## ตรวจไม่ได้ (UNVERIFIED)

<AC ที่ "ตรวจไม่ได้ใน session นี้" พร้อมสิ่งที่ต้องใช้ (env จริง, deploy, ข้อมูล, คนทดสอบมือถือ ฯลฯ) และอย่างอื่นที่ไม่ได้ตรวจ>

