# แม่แบบคู่มือสำหรับ implementer subagent

คัดลอกไฟล์นี้ไปไว้ใน scratchpad นอก repo เติมค่า `<...>` แล้วให้ทุก subagent อ่านเป็นอย่างแรก ใน prompt ของแต่ละตัวชี้ไปที่คู่มือ, ticket ของมัน และ spec — ไม่ต้องคัดลอกเนื้อหาซ้ำ ข้อกำหนดเฉพาะ ticket (สิ่งที่ห้ามแตะ จุดที่ต้องหยุดถาม) ใส่ใน prompt

---

# Implementer guide — <SPEC>

อ่านให้ครบก่อนเริ่ม

## ตัวชี้

- worktree ของคุณคือ repo (engineering-workspace) backend: `projects/bot-orchestration-platform` (Go module `bop`) dashboard: `projects/bop-dashboard` (Next.js, Feature-Sliced Design — อ่าน `ARCHITECTURE.md` ของมัน ESLint บังคับกฎชั้น)
- spec: `<path ของ spec>` · ticket ของคุณ: `<path โฟลเดอร์ ticket>/<NN>-*.md`
- ศัพท์: `GLOSSARY.md` ใช้ศัพท์เหล่านี้ในชื่อ คอมเมนต์ และข้อความ commit · ADR: `docs/adr/`
- สถาปัตยกรรม backend และกฎ import: `ARCHITECTURE.md` (บังคับด้วย depguard ใน `.golangci.yml`)
- Bruno collection: `projects/bot-orchestration-platform/bruno/` (อ่าน `README.md` ของมัน)
- integration branch: `<INTEGRATION_BRANCH>`

## ลำดับงาน

1. ยืนยันว่า worktree อยู่บน `<INTEGRATION_BRANCH>` (`git merge-base --is-ancestor <INTEGRATION_BRANCH> HEAD`) ถ้าไม่ ให้ `git reset --hard <INTEGRATION_BRANCH>` (ตอนนั้น branch ของคุณยังไม่มีงาน)
2. เรียก skill `tdd` แล้วสร้าง ticket แบบ test-first **เห็น test ล้ม (red) ก่อนแก้โค้ดทุกครั้ง** ถ้าเป็น Go เรียก `uber-go` ด้วย
3. คอมเมนต์ในโค้ดตามของเดิมรอบข้าง: ภาษาไทยอธิบายว่า *ทำไม* ศัพท์เทคนิคเป็นอังกฤษ
4. commit บน branch ตัวเองแบบ Conventional Commits scope `bop` สำหรับ backend และ `dashboard` สำหรับ dashboard ท้ายข้อความใส่บรรทัดที่ผู้ดำเนินการสั่งให้ใช้ (`Co-Authored-By: ...`)
5. ก่อนรายงาน: `git merge <INTEGRATION_BRANCH>` เข้า branch ตัวเอง แก้ conflict แล้วตรวจซ้ำ
6. แก้ไฟล์ ticket: ติ๊กเกณฑ์ที่ผ่าน ปล่อยเกณฑ์ของผู้ดำเนินการ (deploy, `bru run --env minipc`) ไม่ติ๊ก ตั้ง `**Status:** implemented`
7. รายงานไม่เกิน 150 คำ: ชื่อ branch, รายการ commit, ตรวจอะไรและตรวจอย่างไร, การตัดสินใจที่คุณทำเอง, เรื่องที่ต้องให้ผู้ใช้ตัดสิน ไม่แปะ diff

## การตรวจ (Mac ไม่มี Go และ Docker)

รันด้วย `.claude/skills/grill-to-deploy/verify.sh <go|dashboard|all> --log <โฟลเดอร์ใน scratchpad>` (รอ container ว่างให้เอง สรุปผลสั้นๆ และเก็บผลเต็มใน log รายงานโดยอ้าง path ของ log ไม่แปะผลเต็ม; ผ่านก็ต่อเมื่อสคริปต์พิมพ์ "ผ่านทุกขั้นที่รัน") ส่วน test ที่ใช้ PostgreSQL จริงไม่อยู่ในสคริปต์ ให้ทำตาม `environment.md` · ทุกอย่างรันใน container ชั่วคราวบน Mini PC ตามคำสั่งใน `environment.md` ของ skill `grill-to-deploy` (Go + lint + format, dashboard, PostgreSQL ชั่วคราว) ภาระ: หนึ่ง container ต่อครั้ง และรอเมื่อมี container ตรวจงานตั้งแต่ 3 ตัว ลบทุกอย่างที่สร้าง (container, network) แม้ล้ม รวม lint และ test Go ไว้ใน container เดียวได้

## กฎเหล็ก

- **ไม่ deploy ไม่ push และไม่รัน Bruno กับ Mini PC** ผู้ดำเนินการ deploy branch และรัน `bru run --env minipc` หลัง merge
- บน Mini PC ห้ามหยุด ลบ หรือแก้ container, volume, image, network หรือ compose project ที่มีอยู่เดิม ห้ามแตะ `~/Workspace/engineering-workspace` สร้างได้เฉพาะ container และ network ชั่วคราวที่ลบเองทั้งหมด
- ห้ามใช้ `sudo`
- ห้ามแก้ `GLOSSARY.md` หรือ ADR ถ้า ticket ต้องการศัพท์หรือการตัดสินใจใหม่ ให้รายงาน
- **หยุดและรายงาน** (ไม่เดา) เมื่อ: ข้อกำหนดของ ticket ขัดกับโค้ดจริง, ต้องตัดสินใจที่ spec กับ ticket ไม่ได้ตอบ, หรือเงื่อนไขหยุดเฉพาะ ticket ที่ผู้ดำเนินการเขียนไว้ใน prompt เป็นจริง

## Bruno

เพิ่ม request ใน `bruno/` ตาม README ของมัน: `seq` เรียงไม่ซ้ำในโฟลเดอร์, ข้อมูลที่สร้างขึ้นต้นด้วย `bruno-`, ลบสิ่งที่ API ให้ลบ, รอเหตุการณ์ async ด้วย `await bru.sleep(ms)` เขียน test ตามพฤติกรรมจริง request ที่ต้อง "ไม่มี cookie" ต้องวางหลัง request logout ของโฟลเดอร์ (cookie jar ของ Bruno CLI จะแนบ cookie ให้ทุก request) คุณรันกับ Mini PC ไม่ได้ จึงอ่านไฟล์เดิมให้ละเอียดแล้วเทียบรูปแบบ
