# .agents — role ผู้ตรวจอิสระ (ระยะ A: ไม่มี hook)

ติดตั้งบางส่วนจากทีมต้นแบบ `claude code skill/agent-team-for-work` — **ไม่มี** `settings.json`, hook, ระบบ job/task หรือ commit gate ที่นี่ ใช้แค่ role ที่ปรับให้ใช้ ticket ใน `.scratch/` เป็น spec

## ที่ติดตั้ง
| ไฟล์ | ใช้ทำอะไร |
|---|---|
| `.claude/agents/analyst.md` | หาหลักฐานแบบอ่านอย่างเดียว (โค้ด, repo พี่น้อง, DB ผ่าน MCP read-only) ไม่แก้โค้ด |
| `.claude/agents/spec-reviewer.md` | ตรวจ change เทียบ ticket ทีละ AC พร้อม `file:line` ไม่แก้โค้ด ใช้เฉพาะ ticket เสี่ยง ticket ปกติให้ lead ตรวจเอง (ดู "ระดับความเสี่ยงและผู้ตรวจโค้ด" ใน `ticket.md`) |
| `.claude/agents/backend.md` | implement ticket (Go, test-first) เฉพาะโค้ดใน `public/` `private/` `utility/` ไม่แตะ `bruno/` ไม่ commit |
| `.claude/agents/qa.md` | เขียน Bruno จาก ticket (ไม่ดูโค้ด) ใต้ `bruno/Regression/<Folder>/` และรันเฉพาะโฟลเดอร์นั้นเมื่อสั่ง |
| `.agents/templates/` | `brief.md`, `evidence.md`, `spec-review.md`, `fix-task.md`, `decision-request.md`, `db-check.md` |

## ขั้นตอนทำงานเป็นทีม
ดู `.agents/workflows/ticket.md` — lead + backend + qa (+ spec-reviewer เฉพาะ ticket เสี่ยง) ทำ implement → Bruno → รัน local → แก้วน → อัปเดต ticket (commit เมื่อผู้ใช้สั่ง)

## เรียกใช้ (ตัวอย่าง)
- "ให้ `spec-reviewer` ตรวจ commit `<sha>` เทียบ `.scratch/<slug>/issues/NN-....md` เขียนผลที่ `.scratch/<slug>/reviews/NN-r1.md`"
- "ให้ `qa` เขียน Bruno จาก ticket NN โดยไม่อ่านโค้ด ลงโฟลเดอร์ `bruno/Regression/<Folder>/` แล้วรายงานช่องโหว่"
- "ให้ `analyst` หาว่าใครเรียก `<endpoint>` และอ่าน field ไหน เขียนที่ `.scratch/<slug>/investigation/NN.md`"

## กติกา
- ผู้ตรวจ ≠ ผู้เขียน: อย่าให้ตัวที่เขียนโค้ดตรวจ/ทดสอบงานตัวเอง
- ทุก role ห้าม commit / push และรายงานเป็นภาษาไทย · ของที่เขียนได้มีแค่ path ที่ผู้สั่งกำหนด
- ทุก role ใช้ `model: sonnet` ไปก่อน (ตั้งค่าที่ `model:` ในหัวไฟล์ role) — ถ้าผู้ตรวจพลาดงานเสี่ยงสูง (เช่น ช่องโหว่ข้อมูลลับ) ค่อยพิจารณาเปลี่ยน role นั้นเป็น `opus` ซึ่งแพงกว่ามาก
- ภาษาเริ่มต้นของทุก role คือไทย: คำถาม คำตอบ ข้อความถึง lead และไฟล์รายงาน (path, คำสั่ง, โค้ด, ชื่อ field คงเดิม) — ผู้ใช้สั่งภาษาอื่นได้
- ไม่ใช้ teammate (Agent Teams): ทุก role เป็น subagent ที่ lead spawn ทีละขั้นด้วยเครื่องมือ Agent พร้อม brief (`.agents/templates/brief.md`) **ไม่ตั้ง `name`** ทำขั้นเดียว รายงาน lead แล้วจบ ไม่มีการคุยกันเองระหว่าง role ทุกการส่งต่อผ่าน lead (ดู "กลไก" ใน `.agents/workflows/ticket.md`)
- subagent spawn ต่อไม่ได้: ทุก role มี `Agent` ใน `disallowedTools` มีแต่ lead ที่ spawn ได้
- brief เป็นไฟล์ใน `reviews/<ticket>/briefs/` lead ตรวจว่ากรอกครบด้วย `grep` ก่อน spawn และไม่แก้หลัง spawn

## ยังไม่ได้ทำ / ยังไม่ได้ตรวจ
- hook (`guard_git`, `guard_sql`, ...) ไม่ได้ติดตั้ง · `guard_git.py` ของต้นแบบ crash กับ `git commit -F -` ต้องแก้ก่อนใช้
- ไม่ได้ลิงก์ skill ของทีม (`grilling`, `domain-modeling`, `grill-with-docs`) — ซ้ำกับ plugin ที่ใช้อยู่
- ยังไม่รู้ว่า Claude Code ค้นเจอ role ใน session ใหม่ครบทั้งสามตัวหรือไม่ (ตรวจด้วย `/agents`)
