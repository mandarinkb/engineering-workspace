# Brief: <fill: role> · ticket <fill: NN> · รอบ <fill: n>

- **Role:** <fill: backend | qa | spec-reviewer | analyst>
- **Step:** <fill: ขั้นใน `.agents/workflows/ticket.md` เช่น "2 implement", "2 write Bruno", "3 contract", "4 run", "4 review", "5 fix">
- **Ticket:** <fill: path ของ ticket>
- **Round:** <fill: รอบที่ ต่อประเด็น>
- **Evidence folder:** <fill: `.scratch/<feature-slug>/reviews/<ticket>/r<n>/`>
- **Revision / change:** <fill: สิ่งที่ต้องตรวจ เช่น diff เทียบ `git log --oneline -1` ตอนเริ่ม, รายการไฟล์ หรือไฟล์ diff>

## อ่านก่อน
<fill: path ที่ต้องอ่าน เช่น AGENTS.md, ticket, spec section, review / fix item รอบก่อน>

## Scope
<fill: backend: รายการไฟล์ที่ผู้ใช้ยืนยัน · qa: โฟลเดอร์ `bruno/Regression/<Folder>/` · reviewer/analyst: path ของไฟล์ผลลัพธ์>

## Contract / ข้อเท็จจริงที่ lead ตรวจแล้ว
<fill: พร้อม `file:line` (กติกาประสานงานข้อ 8) หรือ "ไม่มี">

## คำตัดสินของผู้ใช้
<fill: คำต่อคำ เฉพาะคำตัดสินล่าสุด ถ้าแทนของเดิมให้เขียน "แทนที่คำตัดสิน <เรื่อง> ก่อนหน้า" หรือ "ไม่มี">

## ห้ามในขั้นนี้
<fill: เช่น "ห้ามอ่านโค้ดของ backend", "ห้ามรัน server / bru" (ทุกขั้นของ qa ยกเว้น Step 4 run) หรือ "ไม่มีเพิ่มจาก role">

## รายงานกลับ
<fill: สิ่งที่ต้องมีในรายงาน เช่น verdict, path ไฟล์ผลลัพธ์, fix items, decision request, contract ที่สร้าง, สิ่งที่ตรวจไม่ได้>

subagent จบหลังส่งรายงานนี้ ห้ามรอข้อความเพิ่ม ห้ามส่งงานต่อให้ role อื่นเอง และห้าม spawn subagent ต่อ ถ้างานใหญ่เกินไป ให้เสนอวิธีแบ่งงานในรายงาน

<!-- สำหรับ lead: บันทึกไฟล์นี้ที่ reviews/<ticket>/briefs/<NN>-<role>-step<ขั้น>.md แล้วตรวจว่าไม่เหลือช่องที่ยังไม่กรอกก่อน spawn (วิธีตรวจอยู่ในหัวข้อ "กลไก" ของ .agents/workflows/ticket.md) ห้ามแก้ไฟล์หลัง spawn และห้ามใส่ค่า secret -->
