# แผน: ปรับ role ของ subagent ให้เข้ากับ `grill-to-deploy` (งาน bop)

วันที่: 2026-11-23 · สถานะ: **แผน ยังไม่ได้ลงมือแก้ไฟล์ใดนอกจากไฟล์นี้** · ลบไฟล์นี้ได้เมื่อทำเสร็จ

## สถานะ (อัปเดต 2026-11-23)

เลือกแนว **"ทำน้อยที่สุดก่อน"** แล้ว: ทำและทดสอบกับ Mini PC แล้วคือ `verify.sh` (ข้อ 3.1), การชี้จาก `environment.md`/`implementer-guide.md`, และบรรทัดสรุปต้นทุนเฟส 8 (ข้อ 3.4) **ยังไม่ทำ:** `bop-implementer` (3.2), `bop-verifier` (3.3), และการจัดการไฟล์ role ของ icrm-share (3.5) รอข้อมูลต้นทุนจาก feature จริง 3–5 ตัวก่อนตัดสิน (ข้อ 5)

## 0. เป้าหมาย และสิ่งที่ตรวจแล้ว / ยังไม่ได้ตรวจ

เป้าหมาย: ให้ feature จบด้วย workflow ของ `grill-to-deploy` โดยใช้ subagent ที่มี role ถาวร ประหยัดโทเค็นของ lead และลดงานที่ต้องทำซ้ำทุก feature

**อ่านแล้ว:** `SKILL.md` ทุกเฟส, `environment.md`, `testing.md`, `implementer-guide.md`, `traps.md` (ส่วนต้น), `.agents/workflows/ticket.md`, ไฟล์ role ใน `.claude/agents/` (ส่วนต้น)
**ยังไม่ได้อ่าน:** `ARCHITECTURE.md` ของ bop และ dashboard, README ของ bruno, `traps.md` ส่วนท้าย
**ยังไม่ได้ทดสอบ:** Playwright MCP ใช้ได้จากใน subagent หรือไม่, frontmatter ของ agent กำหนด worktree ได้หรือไม่ (แผนนี้ให้ lead ส่ง `isolation: "worktree"` ตอน spawn เหมือนที่ skill ทำอยู่แล้ว)

## 1. ความจริงที่ต้องยอมรับก่อน (กันคาดหวังเกินจริง)

- การย้ายคู่มือจาก scratchpad เป็นไฟล์ role ถาวร **ไม่ได้ลดโทเค็นของ implementer มาก** เพราะ role ไฟล์ก็ถูกโหลดเข้า context ทุก spawn อยู่ดี (ขนาดใกล้เคียง `implementer-guide.md` 6.3 KB)
- สิ่งที่ได้จริง: lead ไม่ต้องคัดลอก/เติมแม่แบบทุก feature, บังคับ `disallowedTools: Agent` และ `model`, ค้นเจอผ่าน `/agents`, และทุก feature ใช้กฎชุดเดียวกัน
- ที่ประหยัดโทเค็นได้มากกว่าคือ **สคริปต์ตรวจ (ข้อ 3.1)** เพราะ output ของ `go test`/`lint`/`next build` ยาวมาก และ **verifier (ข้อ 3.3)** เพราะ log ของ Bruno/Playwright ไม่ต้องเข้า context ของ lead
- ตัวเลขการประหยัดยังไม่มี ต้องวัดจากการรันจริงหนึ่งรอบ (ข้อ 5)

## 2. จะยกอะไรมาจาก icrm-share และจะไม่ยกอะไร

| ยกมา | เหตุผล |
|---|---|
| หลักผู้ตรวจ ≠ ผู้เขียน, `disallowedTools: Agent` ทุก role | ตรงกับเฟส 4–7 อยู่แล้ว |
| ความคิด "verifier ไม่แก้โค้ด" (qa) | ตรงกับ playbook และเฟส 7 |
| สรุปต้นทุนหนึ่งบรรทัดต่อ ticket (spawn, รอบแก้, สาเหตุที่วน) | วัดว่าอะไรแพงจริง ลงเฟส 8 |
| รายงานอ้าง path ไม่แปะ output, ผลดิบลงไฟล์ | ตรงกับ `traps.md` ข้อ "อ่านผลทุกบรรทัด" |

| **ไม่ยกมา** | เหตุผล |
|---|---|
| `ticket.md` 40 KB ทั้งก้อน, `log.tsv`, `progress.md`, `briefs/` ทีละไฟล์ | หนักเกินสำหรับงานของ skill นี้ ที่ prompt ชี้ guide + ticket + spec อยู่แล้ว และ Status ใน ticket คือบันทึกความคืบหน้า |
| `digest.md` | ticket ของ `to-tickets` มีเกณฑ์ตรวจได้ทีละข้อแล้ว ใช้ไอดีรูป `T<NN>-<n>` ในรายงานแทน |
| `spec-reviewer` | เฟส 5 ใช้ `code-review` ซึ่งตรวจ Standards + Spec ขนานกันอยู่แล้ว ซ้ำกัน |
| `backend` (Go ของ icrm-share), `qa` (Bruno + MySQL read-only MCP) | เส้นทางไฟล์, ภาษา, ฐานข้อมูลต่างกัน |

## 3. สิ่งที่จะสร้าง / แก้ (เรียงตามลำดับทำ)

### 3.1 สคริปต์ตรวจ `verify.sh` (ทำก่อน เพราะประหยัดโทเค็นสุดและเสี่ยงต่ำ)

- **ที่:** `.claude/skills/grill-to-deploy/verify.sh` (อยู่กับ skill ให้ skill ครบในตัว)
- **ทำอะไร:** รวมคำสั่งจาก `environment.md` (Go: vet+test+lint+fmt, dashboard: vitest+eslint+build) โหมด `go` | `dashboard` | `all`; รวมลูปรอ "container ตรวจงาน ≥ 3" ไว้ในสคริปต์ด้วย เพื่อไม่ต้องให้ subagent เขียนเอง
- **เอาต์พุต:** สรุปไม่เกิน ~20 บรรทัด (ขั้นไหน OK/FAIL, จำนวน test, 30 บรรทัดสุดท้ายเมื่อ FAIL) ผลเต็มลงไฟล์ที่ `--log <path>` และ **exit code ไม่ใช่ 0 เมื่อมีคำว่า FAIL / `×` / `build failed`** (ตรงกับ `traps.md` ข้อ "exit 0 ไม่ใช่หลักฐาน")
- **ข้อควรระวัง:** ต้องใช้ ssh + docker บน Mini PC ซึ่งตามกฎส่วนตัวต้องบอกแผนและขออนุญาตก่อนสร้าง container (ดูข้อ 4)
- **เสร็จเมื่อ:** รันบน `main` ปัจจุบันแล้วได้ OK ทั้งสองโหมด และลองทำให้ test ล้มหนึ่งตัวแล้วสคริปต์ exit ไม่เป็น 0 (พิสูจน์ว่าล้มได้จริง)

### 3.2 role `bop-implementer` (backend + dashboard)

- **ที่:** `.claude/agents/bop-implementer.md`
- **frontmatter:** `disallowedTools: Agent`, `model: sonnet` (ตามที่ทีมใช้ ค่อยปรับตามข้อมูลต้นทุน)
- **เนื้อหา:** ย้ายจาก `implementer-guide.md` ส่วนที่ไม่เปลี่ยนตาม feature: ตัวชี้ (`projects/bot-orchestration-platform` Go module `bop`, `projects/bop-dashboard` Next.js FSD, `GLOSSARY.md`, ADR, `ARCHITECTURE.md`), ลำดับงาน (tdd red-first, uber-go, คอมเมนต์ไทย, Conventional Commits scope `bop`/`dashboard`), กฎเหล็ก (ห้าม deploy/push/รัน Bruno กับ Mini PC, ห้าม sudo, ห้ามแตะ container เดิม, ห้ามแก้ GLOSSARY/ADR), เงื่อนไขหยุดแล้วรายงาน, ส่วน Bruno, การตรวจโดยเรียก `verify.sh` แทนคำสั่งยาว, รูปแบบรายงานไม่เกิน 150 คำ
- **สิ่งที่ยังส่งใน prompt ตอน spawn (3–4 บรรทัด):** path ของ spec, path ของ ticket, integration branch, บรรทัด `Co-Authored-By` ของรอบนั้น, ข้อห้ามเฉพาะ ticket
- **หนึ่ง role หรือสอง:** แนะนำหนึ่ง role ก่อน (ticket ระบุขอบเขต `bop` หรือ `dashboard`) แยกเป็นสองเมื่อไฟล์เกิน ~8 KB
- **แก้ `SKILL.md` เฟส 4:** เปลี่ยน "คัดลอก `implementer-guide.md` … เติมค่า" เป็น "spawn `bop-implementer` พร้อม `isolation: "worktree"`" และ **ลบ `implementer-guide.md`** เพื่อไม่ให้มีแหล่งความจริงสองที่ (เก็บส่วนที่ยัง generic ไว้ใน role เท่านั้น)
- **เสร็จเมื่อ:** `/agents` เห็น role นี้ และ spawn ทดสอบให้รายงานกฎเหล็กของตัวเองกลับมาตรงกับไฟล์

### 3.3 role `bop-verifier` (เฟส 7)

- **ที่:** `.claude/agents/bop-verifier.md`, `disallowedTools: Edit, NotebookEdit, Agent` (ไม่แก้โค้ด ตามแบบ analyst/spec-reviewer ที่มีอยู่)
- **ทำอะไร:** รัน `bru run --env minipc` และ `run-with-restart.sh`, ตรวจ UI ด้วย Playwright ตาม `testing.md` (กันผ่านปลอม: รอเห็นสถานะเริ่มต้น, marker กัน reload, ✗ ต้องรันซ้ำแยกว่าผิดที่สคริปต์หรือระบบ), ลบข้อมูลทดสอบด้วย ID ตรงตัว/prefix พร้อมนับก่อนหลัง
- **ข้อจำกัด:** ทีละตัวเท่านั้น (Mini PC ใช้ร่วมกับ stack จริง), เอาต์พุตดิบลงไฟล์ใน scratchpad, รายงานเป็นตาราง `T<NN>-<n> | ✓/✗/ไม่พิสูจน์ | หลักฐาน (path)` และแยก "เห็นจริง" ออกจาก "มาจากความรู้ทั่วไป"
- **ไม่ทำ:** ไม่ติ๊กเกณฑ์ใน ticket และไม่เขียนบรรทัด `Verified` ผู้ดำเนินการ (lead) ทำเองจากรายงาน ตามเฟส 8
- **แก้ `SKILL.md` เฟส 7:** ให้ lead spawn `bop-verifier` แล้วอ่านเฉพาะรายงาน
- **ความเสี่ยงที่ต้องทดสอบก่อนเชื่อ:** subagent เข้าถึง Playwright MCP ได้หรือไม่ ถ้าไม่ ให้ใช้ไลบรารีใน scratchpad ตามที่ `testing.md` กำหนดไว้แล้ว
- **เสร็จเมื่อ:** รันกับ feature จริงหนึ่งรอบ และ lead ไม่ต้องอ่าน log ดิบของ Bruno/Playwright

### 3.4 สรุปต้นทุนต่อ feature (เฟส 8)

เพิ่มหนึ่งบรรทัดใน `SKILL.md` เฟส 8: lead เขียนสรุปลงไฟล์ spec หรือ PR body ว่า จำนวน spawn, ticket ที่วนแก้กี่รอบและเพราะอะไร, ขั้นที่หนักสุด นับจาก git log และจำนวน subagent ที่เปิดจริง ไม่ใช้ความจำ

### 3.5 ไฟล์ role ของ icrm-share ที่อยู่ใน `.claude/agents/` ตอนนี้

ตัดสินใจก่อนทำข้อ 3.2 (ดู "คำถามที่ต้องตอบ" ข้อ 1) ตัวเลือก:
- **A (แนะนำ):** คงไว้ที่เดิม และเติมต้น `description` ว่า "[icrm-share เท่านั้น ห้ามใช้กับ bop]" ผลกระทบน้อย ไม่ย้ายไฟล์ แต่ role ที่ทำงานคนละโปรเจกต์ยังอยู่ปนกัน
- **B:** ย้ายไป `.agents/icrm-share/` (เป็นเอกสารอ้างอิง ไม่ถูกโหลดเป็น agent) ต้องแก้ path ที่ `.agents/README.md` และ `ticket.md` อ้างอยู่
- **C:** เปลี่ยนชื่อไฟล์เป็น `icrm-*` (ต้องแก้ `name:` และทุกจุดที่อ้างชื่อ `backend`/`qa`)

## 4. เรื่องสิทธิ์ที่ต้องตกลงก่อนเริ่ม

กฎส่วนตัวของผู้ใช้: ก่อนสร้างอะไรบน Mini PC (container, ไดเรกทอรี, แพ็กเกจ, พอร์ต) ต้องบอกแผนและได้ yes ยกเว้นแผนนั้นเคยอนุมัติแล้ว implementer และ verifier เป็น subagent ถามผู้ใช้เองไม่ได้ จึงต้อง:
- **ต้นเฟส 4:** lead บอกแผนครั้งเดียว ("implementer แต่ละตัวรัน container ชั่วคราวได้ทีละหนึ่ง ตามชุดคำสั่งใน `verify.sh` และลบเองเมื่อจบ ไม่แตะของเดิม") แล้วรอ yes ครั้งเดียวต่อ feature
- **ต้นเฟส 7:** แผนของ verifier (Bruno + Playwright + ลบข้อมูลทดสอบ) ขอ yes ครั้งเดียวเช่นกัน
- ในไฟล์ role ระบุว่าแผนที่อนุมัติอยู่ที่ prompt ของ lead ถ้า prompt ไม่ระบุ ให้หยุดแล้วถาม

## 5. ทดลองก่อนเชื่อ (ขั้นตอนตรวจ)

1. ทำ 3.1 → ตรวจตามเกณฑ์เสร็จของข้อนั้น
2. ทำ 3.2 → spawn ทดสอบให้รายงานกฎ แล้วลอง ticket เล็กหนึ่งใบใน feature จริง (ไม่ใช่งานสาธิต)
3. ทำ 3.3 เมื่อถึงเฟส 7 ของ feature นั้น
4. หลังจบ feature: ดูสรุปต้นทุน (3.4) เทียบกับของเดิมที่จำได้ ถ้าไม่ดีขึ้น ให้ย้อนกลับ (ไฟล์ทั้งหมดอยู่ใน git ย้อนได้)
5. ตัดสินใจข้อต่อไป (แยก role, ปรับโมเดล, เพิ่มสคริปต์) จากข้อมูลนั้น ไม่เดา

## 6. ไฟล์ที่จะแตะ (สรุป)

| ไฟล์ | การกระทำ |
|---|---|
| `.claude/skills/grill-to-deploy/verify.sh` | เพิ่ม |
| `.claude/agents/bop-implementer.md` | เพิ่ม |
| `.claude/agents/bop-verifier.md` | เพิ่ม |
| `.claude/skills/grill-to-deploy/SKILL.md` | แก้เฟส 4, 7, 8 (ไม่กี่บรรทัด) |
| `.claude/skills/grill-to-deploy/implementer-guide.md` | ลบ (หลัง role ใช้งานได้แล้ว) |
| `.claude/skills/grill-to-deploy/environment.md` | ชี้ไป `verify.sh` แทนคำสั่งยาว (คงคำสั่งเดิมเป็นอ้างอิงสั้นๆ) |
| `.claude/agents/{analyst,backend,qa,spec-reviewer}.md` | ตามตัวเลือก 3.5 |
| `.agents/README.md`, `.agents/workflows/ticket.md` | ตามตัวเลือก 3.5 (เฉพาะ B และ C) |

## 7. คำถามที่ต้องตอบก่อนลงมือ

1. ไฟล์ role ของ icrm-share ใน `.claude/agents/` ตอนนี้ จะเลือก A, B หรือ C (ข้อ 3.5)
2. ตกลงให้ `verify.sh` อยู่ใน skill หรือจะให้อยู่ที่ `scripts/` ของ root
3. เริ่มด้วย implementer หนึ่ง role (แนะนำ) หรือแยก backend/dashboard ตั้งแต่ต้น
4. ให้ลองกับ feature จริงชิ้นไหนเป็นตัวทดลอง
