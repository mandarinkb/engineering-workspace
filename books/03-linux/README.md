# Volume 3 — Linux & Git

เครื่องมือประจำวันของ backend/infra engineer ทุกคน ไม่ว่าจะ deploy, debug production, หรือทำงานร่วมกับทีม ก็ต้องผ่าน Linux shell และ Git เกือบตลอดเวลา หมวดนี้รวม **Linux**, **Shell**, และ **Git** เข้าด้วยกัน เพราะเป็นทักษะพื้นฐานระดับ "เครื่องมือทำมาหากิน" ที่ใช้คู่กันเสมอ

## เป้าหมายการเรียนรู้

- ใช้คำสั่ง Linux พื้นฐานสืบสวนปัญหาระบบ (process กิน CPU สูง, disk เต็ม, service ล่ม) ได้ด้วยตัวเอง
- เขียน shell script/one-liner เพื่อ automate งาน วิเคราะห์ log ได้
- ใช้ Git ในการทำงานเป็นทีมได้อย่างถูกต้อง รวมถึงการแก้ปัญหา merge conflict และ rebase

## สารบัญ

### [บทที่ 1 — Linux Fundamentals](01-linux-fundamentals.md)

- [ ] **Filesystem** — FHS (Filesystem Hierarchy Standard), mount point, hard link vs symbolic link
- [ ] **inode** — inode คืออะไร, ต่างจาก filename อย่างไร, ทำไม `rm` ไฟล์ที่ process เปิดอยู่แล้ว disk space ไม่ว่าง
- [ ] **systemd** — unit file, service management (`systemctl start/enable/status`), target
- [ ] **journalctl** — อ่าน log จาก systemd journal, filter ตาม unit/เวลา
- [ ] **ps** — ดู process, flag ที่ใช้บ่อย (`ps aux`, `ps -ef`), อ่านคอลัมน์ (PID, PPID, STAT, %CPU, %MEM)
- [ ] **top** — real-time monitoring, load average คืออะไร อ่านอย่างไร
- [ ] **ss** — ดู socket/connection แทน `netstat` รุ่นเก่า, ดู listening port, established connection

### [บทที่ 2 — Shell Scripting](02-shell-scripting.md)

- [ ] **bash** — variable, condition, loop, function, exit code, pipe (`|`), redirection (`>`, `>>`, `2>&1`)
- [ ] **grep** — pattern matching, regex พื้นฐาน, flag ที่ใช้บ่อย (`-i`, `-v`, `-r`, `-E`)
- [ ] **awk** — field processing, ใช้สรุป/แปลง log แบบ column-based
- [ ] **sed** — stream editor, find & replace, ใช้แก้ไฟล์แบบ non-interactive
- [ ] **curl** — ทดสอบ HTTP API, ดู header, ส่ง POST/JSON, debug ด้วย `-v`
- [ ] **jq** — parse/filter JSON จาก command line ประกอบกับ API response

### [บทที่ 3 — Git](03-git.md)

- [ ] **Commit** — commit object, การเขียน commit message ที่ดี
- [ ] **Branch** — branching model, HEAD, detached HEAD
- [ ] **Merge** — fast-forward vs 3-way merge, merge conflict
- [ ] **Rebase** — ความต่างจาก merge, interactive rebase, เมื่อไหร่ควร/ไม่ควร rebase (โดยเฉพาะ branch ที่แชร์กับคนอื่น)
- [ ] **Cherry-pick** — หยิบ commit เดี่ยวข้ามสาขา
- [ ] **Git Flow** — feature/release/hotfix branch, เทียบกับ trunk-based development ที่ใช้คู่กับ [Volume 16 — CI/CD](../16-cicd/README.md)

## Lab ที่เกี่ยวข้อง

ฝึกปฏิบัติได้ที่ [labs/](../../labs/README.md) และ [scripts/](../../scripts/README.md) สำหรับ automation script

## แหล่งอ้างอิงแนะนำ

ดู [resources/](../../resources/README.md)
