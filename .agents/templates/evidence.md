# หลักฐาน ticket <fill: NN>: <fill: ชื่อ ticket>

- **Ticket:** <fill: path ของ ticket>
- **รอบทั้งหมด:** <fill: n>
- **โค้ดเปลี่ยนหลัง qa รันรอบสุดท้ายไหม:** <fill: "ไม่มี" หรือ "มี: <ไฟล์> → รันใหม่ที่ r<n> / ไม่ได้ทดสอบส่วนนี้"> (เทียบ `git status --short` + `git diff --stat` กับ `r<n>/git-status.txt`)
- **ระดับความเสี่ยงและผู้ตรวจโค้ด:** <fill: "เสี่ยง: เกณฑ์ข้อ n → spec-reviewer" หรือ "ปกติ → lead" และรอบที่เลื่อนเป็นเสี่ยงถ้ามี>
- **lead build/test:** <fill: `go build ./...` + `go test ./...` exit code และ path ของ log>

lead เขียนหน้านี้จากไฟล์ใน `r*/` เท่านั้น ทุกช่อง "หลักฐาน" ต้องชี้ไฟล์ที่มีอยู่จริง ข้อความในหน้านี้เป็นสารบัญ ผลดิบอยู่ใน log

## ทีละ AC

| AC | ทดสอบอย่างไร | ผล | หลักฐาน | ประวัติรอบ |
|---|---|---|---|---|
| <fill: AC-1 …> | <fill: Bruno request / test / การไล่โค้ด> | <fill: PASS / FAIL / UNVERIFIED> | <fill: r<n>/bruno.log `<request>`, r<n>/spec-review.md หรือ r<n>/lead-review.md> | <fill: เช่น "r1 FAIL: คืน totalPoints → r2 PASS"> |

## Test ของ backend

<fill: test ที่เห็นแดงก่อนแก้ / ไม่เคยเห็นแดง / mutation ที่ test จับได้และจับไม่ได้ → ลิงก์ `r<n>/backend-tests.md`> · <fill: ticket ปกติ: mutation ที่ lead รันซ้ำ 1 จุดและผล → `r<n>/lead-review.md`, `r<n>/lead-mutation.log` · ticket เสี่ยง: "ไม่ใช้">

## สิ่งที่ไม่ได้ทดสอบ

<fill: UNVERIFIED และ gap ทั้งหมด พร้อมสิ่งที่ต้องใช้ หรือ "ไม่มี">

## สุ่มตรวจเอง

<fill: 2–3 เคสที่แนะนำให้ลอง พร้อมเหตุผล และลิงก์ส่วน "ลองเอง" ใน `r<n>/qa.md`>

## ไฟล์ทั้งหมด

<fill: รายการไฟล์ใน `r1/` … `r<n>/`> · `log.tsv` (การตัดสินใจทุกจุด) · `progress.md` (สถานะล่าสุด) · `briefs/` (คำสั่งที่ lead ส่งให้ subagent ทุกครั้ง)

อ่านเสร็จแล้วลบโฟลเดอร์ `<ticket>/` ได้ (อยู่ใน `reviews/` ที่ถูก gitignore)
