# บทที่ 3: Git

## 3.1 Commit

### Commit Object

ทุก commit ใน Git คือ object ที่เก็บ:

- **Snapshot** ของ tree (โครงสร้างไฟล์ทั้งหมด ณ จุดนั้น — ไม่ใช่แค่ diff!)
- **Parent commit** (หนึ่งตัวสำหรับ commit ปกติ, สองตัวสำหรับ merge commit, ไม่มีเลยสำหรับ commit แรกสุด)
- Author, committer, timestamp
- Commit message

แต่ละ commit ถูกระบุด้วย **hash** (SHA-1 หรือ SHA-256 ใน Git รุ่นใหม่) ที่คำนวณจากเนื้อหาทั้งหมดข้างต้น — เปลี่ยนแม้แต่ตัวอักษรเดียวในไฟล์ หรือแก้ commit message ก็ได้ hash ใหม่ทันที นี่คือเหตุผลที่การ "แก้ไข commit เก่า" (เช่น `rebase`, `amend`) จะเปลี่ยน hash ของ commit นั้นและทุก commit ที่ตามหลังมันเสมอ

### เขียน Commit Message ที่ดี

```
สั้น กระชับ ใช้ imperative mood: "Fix" ไม่ใช่ "Fixed" หรือ "Fixes"

อธิบาย "ทำไม" ไม่ใช่ "ทำอะไร" (โค้ด diff บอกอยู่แล้วว่าทำอะไร)
เช่น "Add retry to payment API call" ดีกว่า "Update payment.go"
```

Commit message ที่ดีช่วยให้ `git log` และ `git blame` มีประโยชน์จริงตอนย้อนกลับมาดูภายหลัง (โดยเฉพาะตอน debug production issue)

## 3.2 Branch

**Branch** คือ pointer แบบเบาๆ (แค่ไฟล์เล็กๆ เก็บ hash ของ commit ล่าสุด) ที่ชี้ไปยัง commit หนึ่ง — **ไม่ใช่** การ copy โค้ดทั้งหมด นี่คือเหตุผลที่การสร้าง branch ใน Git เร็วมากแทบจะทันที (ต่างจากระบบ version control รุ่นเก่าบางตัวที่ branch แพง)

### HEAD

**HEAD** คือ pointer พิเศษที่บอกว่า "ตอนนี้กำลังอยู่ที่ branch/commit ไหน" ปกติ HEAD จะชี้ไปที่ branch (เช่น `refs/heads/main`) ซึ่งชี้ไปที่ commit อีกที (HEAD → main → commit abc123)

### Detached HEAD

ถ้า checkout ไปที่ commit hash ตรงๆ (ไม่ใช่ชื่อ branch) เช่น `git checkout abc123` HEAD จะชี้ไปที่ commit นั้นโดยตรง ไม่ผ่าน branch ใดๆ เรียกว่า **detached HEAD** — ถ้า commit เพิ่มในสถานะนี้แล้วสลับไป branch อื่นโดยไม่สร้าง branch ใหม่มาเก็บไว้ก่อน commit เหล่านั้นจะ**หาไม่เจอ**ได้ง่าย (แม้จะยังไม่ถูกลบจริงจนกว่า garbage collection จะทำงาน) วิธีป้องกัน: ถ้าจะ commit ต่อจาก detached HEAD ให้สร้าง branch ใหม่ก่อนเสมอด้วย `git switch -c new-branch-name`

## 3.3 Merge

### Fast-Forward Merge

ถ้า branch ปลายทาง (เช่น `main`) ไม่มี commit ใหม่เพิ่มเข้ามาเลยตั้งแต่แยก branch ออกไป Git แค่ "เลื่อน pointer" ของ `main` ไปยัง commit ล่าสุดของ branch ที่จะ merge — ไม่มี merge commit ใหม่เกิดขึ้น:

```
ก่อน:  main ──A──B
                   \
       feature      C──D

หลัง:  main ──A──B──C──D   (fast-forward, main แค่เลื่อน pointer)
```

### 3-Way Merge

ถ้าทั้งสอง branch มี commit ใหม่แยกกันไปคนละทาง Git จะสร้าง **merge commit** ใหม่ที่มี parent สองตัว โดยเทียบ 3 จุด: common ancestor, ปลาย branch หนึ่ง, ปลายอีก branch หนึ่ง:

```
main    ──A──B────────E (merge commit)
              \       /
feature        C──D──
```

### Merge Conflict

เกิดเมื่อทั้งสอง branch แก้ไข**บรรทัดเดียวกัน**ของไฟล์เดียวกันต่างกัน Git ไม่รู้ว่าจะเลือกฝั่งไหน จึงหยุดรอให้คนตัดสินใจเอง โดยแทรก marker ในไฟล์:

```
<<<<<<< HEAD
โค้ดจาก branch ปัจจุบัน
=======
โค้ดจาก branch ที่กำลัง merge เข้ามา
>>>>>>> feature-branch
```

ต้องแก้ไขไฟล์ให้เหลือโค้ดที่ถูกต้อง ลบ marker ออก แล้ว `git add` + `git commit` เพื่อจบการ merge

## 3.4 Rebase

**Rebase** คือการ "ย้ายฐาน" ของ branch ไปตั้งอยู่บน commit ใหม่ล่าสุดของอีก branch โดยเล่น (replay) commit ของ branch ตัวเองซ้ำใหม่ทีละตัวบนฐานใหม่นั้น

```
ก่อน:  main    ──A──B──E
                     \
       feature        C──D

หลัง rebase feature onto main:
       main    ──A──B──E
                        \
       feature           C'──D'   (hash ใหม่ทั้งหมด เพราะ parent เปลี่ยน)
```

### Rebase vs Merge

- **Merge** — เก็บประวัติจริงไว้ครบ (รู้ว่า branch แยก/รวมกันตอนไหน) แต่ history อาจดูรกถ้า merge บ่อย
- **Rebase** — ได้ history เส้นตรง สวยงาม อ่านง่าย แต่**เปลี่ยน hash ของทุก commit ที่ replay ใหม่**

### Interactive Rebase

`git rebase -i` เปิดให้แก้ไข commit history ก่อนหน้าได้อย่างละเอียด:

```
pick   a1b2c3 Add login endpoint
squash d4e5f6 Fix typo in login       ← รวมเข้ากับ commit ก่อนหน้า
reword g7h8i9 Add password hashing    ← แก้ commit message
drop   j1k2l3 Debug print (ลืมลบ)     ← ลบ commit นี้ทิ้งไปเลย
```

มีประโยชน์มากก่อน merge feature branch เข้า main — ทำ commit history ให้สะอาด อ่านง่าย ก่อนแชร์ให้คนอื่นเห็น

### กฎทองของ Rebase

**อย่า rebase branch ที่คนอื่นกำลังใช้งานร่วมกันอยู่ (shared/public branch)** เพราะ rebase เปลี่ยน hash ของทุก commit ที่ replay ใหม่ทั้งหมด ถ้ามีคนอื่น pull branch เดิมไปแล้ว ประวัติจะขัดแย้งกันทันที (คนละ hash สำหรับ commit ที่ "เนื้อหาเดียวกัน") นำไปสู่ merge conflict ที่ยุ่งเหยิงเมื่อพยายาม sync กัน — rebase ควรใช้กับ branch ส่วนตัวที่ยังไม่ push หรือ push ไปแล้วแต่รู้แน่ชัดว่าไม่มีใครอื่นดึงไปใช้

## 3.5 Cherry-pick

**Cherry-pick** คือการหยิบ commit เดี่ยวๆ จาก branch หนึ่ง มา apply ซ้ำบน branch ปัจจุบัน โดยไม่ต้อง merge ทั้ง branch:

```bash
git cherry-pick a1b2c3
```

ใช้บ่อยเมื่อต้องการแค่ 1 commit เฉพาะจาก branch อื่น (เช่น bug fix ที่เกิดใน feature branch แต่ต้องรีบเอาไป apply บน production hotfix branch ด้วย โดยไม่อยากดึง feature อื่นที่ยังไม่เสร็จตามไปด้วย) — เหมือน rebase ตรงที่ commit ที่ได้จะมี **hash ใหม่** (เพราะ parent ต่างไปจากต้นฉบับ) แม้เนื้อหา diff จะเหมือนเดิม

## 3.6 Git Flow

**Git Flow** คือ branching model ที่กำหนด branch หลักหลายประเภท พร้อมกฎว่า branch ไหนคุยกับ branch ไหนได้:

```
main       ─────────────●───────────●──────   (production เท่านั้น, tag version ที่นี่)
                          \         /
release/1.2 ─────────●────●───────    (เตรียม release, bug fix เล็กน้อยก่อนออก)
                       \  /
develop    ──●───●──●───●────●──●──   (รวม feature ที่เสร็จแล้ว)
              \ /        \  /
feature/a      ●──●        (งานแต่ละ feature แยก branch ของตัวเอง)
feature/b               ●──●
```

- **main** — โค้ดที่ deploy จริงเท่านั้น
- **develop** — โค้ดล่าสุดที่รวม feature เสร็จแล้ว รอเตรียม release
- **feature/\*** — งานแต่ละ feature แยกจากกัน
- **release/\*** — เตรียมความพร้อมก่อนออก version ใหม่ (bug fix เล็กน้อย, ไม่รับ feature ใหม่แล้ว)
- **hotfix/\*** — แก้ปัญหาด่วนบน production โดยไม่ต้องรอรอบ release ปกติ

### Git Flow vs Trunk-Based Development

Git Flow เหมาะกับระบบที่มีรอบ release ชัดเจน เป็นช่วงๆ (เช่น software ที่ผู้ใช้ต้อง update เอง) แต่สำหรับระบบที่ deploy บ่อยๆ (หลายครั้งต่อวัน) แบบที่ผูกกับ [Volume 16 — CI/CD](../16-cicd/README.md) และแนวคิด GitOps มักนิยม **Trunk-Based Development** แทน — ทุกคน merge เข้า `main` บ่อยๆ (feature branch อายุสั้นมาก เป็นชั่วโมง/วัน ไม่ใช่สัปดาห์) ควบคุมความเสี่ยงด้วย feature flag แทนการแยก branch ค้างไว้นานๆ ซึ่งลดปัญหา "merge ครั้งใหญ่แล้ว conflict มหาศาล" ที่มักเกิดกับ branch ที่แยกออกไปนาน

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| Commit | เก็บ snapshot ไม่ใช่ diff, hash เปลี่ยนถ้าเนื้อหา/parent เปลี่ยน |
| Branch | pointer เบาๆ ชี้ commit ไม่ใช่การ copy โค้ด |
| Merge | fast-forward (ไม่มี commit ใหม่) vs 3-way (สร้าง merge commit) |
| Rebase | history เส้นตรงสวยงาม แต่เปลี่ยน hash — ห้ามใช้กับ shared branch |
| Cherry-pick | หยิบ commit เดี่ยวข้าม branch โดยไม่ merge ทั้งหมด |
| Git Flow vs Trunk-based | Git Flow เหมาะ release เป็นรอบ, trunk-based เหมาะ deploy บ่อยแบบ CI/CD |

## คำถามทบทวน

1. ทำไมการ rebase ถึงเปลี่ยน hash ของ commit ทุกตัวที่ replay ใหม่ ทั้งที่เนื้อหา diff เหมือนเดิม?
2. อธิบายว่าทำไมไม่ควร rebase branch ที่แชร์กับคนอื่นอยู่แล้ว จะเกิดอะไรขึ้นถ้าฝืนทำ?
3. Fast-forward merge กับ 3-way merge ต่างกันตรงไหน เกิดขึ้นในสถานการณ์ไหนคนละแบบ?
4. Cherry-pick กับ merge ต่างกันอย่างไรในแง่ผลลัพธ์ที่ได้บน branch ปัจจุบัน?
5. ทำไม trunk-based development ถึงเหมาะกับระบบที่ deploy บ่อยมากกว่า Git Flow?

---

ก่อนหน้า: [บทที่ 2 — Shell Scripting](02-shell-scripting.md) | กลับสู่ [สารบัญ Volume 3](README.md)
