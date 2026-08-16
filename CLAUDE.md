# บริบทของ Workspace นี้

ไฟล์นี้สรุปบริบทเจ้าของ repo และเป้าหมาย เพื่อให้ session ใหม่เข้าใจสถานการณ์ได้ทันทีโดยไม่ต้องอธิบายซ้ำ

## เจ้าของ repo คือใคร

- อายุ 39 ปี ทำงานสาย backend มาประมาณ 4 ปีกว่า เขียน **Golang เป็นหลัก**
- อยู่ในตลาดงานไทย
- ตอนนี้ยังมีงานทำอยู่ แต่กำลัง **เตรียมตัวเชิงรุกเผื่อโดน layoff** — เป้าหมายคือให้หางานใหม่ได้ไม่ยากถ้าเกิดเหตุการณ์นั้นขึ้นจริง

## เป้าหมายการเรียนรู้

1. **ขยายเป็น fullstack** — เรียน React/Next.js เพิ่มจาก backend เดิม เพื่อผ่าน filter ตำแหน่งงานได้กว้างขึ้น (ไม่ได้ตั้งใจเก่งเท่า frontend specialist)
2. **ขยายเป็น infra ลึก** — ออกแบบระบบรองรับ scale ระดับหลักล้าน request ได้ เพื่อเป็นจุดขายที่ทำให้ต่างจาก backend ทั่วไป และเป็นจุดที่อายุ 39 กลายเป็นข้อได้เปรียบ (senior judgment) แทนข้อเสีย

**หลักการที่ตกลงกันไว้**: portfolio ที่ deploy ได้จริง สำคัญกว่าจำนวนความรู้ในหัว — เรียนแบบผูกกับโปรเจกต์เดียวที่โตขึ้นเรื่อยๆ ไม่ใช่กระจายไปเรียนหลายเรื่องแยกกัน

## โปรเจกต์หลัก (Flagship Project): Bot Orchestration Platform (BOP)

แพลตฟอร์มควบคุม/สั่งงาน bot แบบ workflow — มาจากความสนใจส่วนตัวเรื่อง web scraping/bot automation จริงของเจ้าของ (ไม่ใช่โจทย์สมมติ จึงมีโอกาสทำจนจบสูงกว่า)

- **Positioning**: วางเป็น "lightweight cloud-native workload automation platform" แนวเดียวกับ **Control-M** (BMC) / Apache Airflow แต่สร้างด้วย stack cloud-native สมัยใหม่ — narrative นี้ตรงใจตลาดไทยเป็นพิเศษเพราะแบงก์/ประกัน/telco ไทยใช้ Control-M กันแพร่หลายและกำลัง modernize
- **ข้อจำกัดเชิงออกแบบที่ตั้งใจไว้**: ต้องเป็น engine ทั่วไปที่รองรับ bot หลายประเภทแบบ pluggable (scrape ข้อมูลสาธารณะ, monitor ราคา, แจ้งเตือน) **ห้าม hard-code ผูกกับ bot จองสินค้า/ตั๋วแบบเฉพาะเจาะจง** — เหตุผลคือ engine ทั่วไปเป็น system design ที่ลึกกว่าและไม่เสี่ยงชน ToS ของเว็บเป้าหมายจนกลายเป็นจุดลบตอนสัมภาษณ์
- Roadmap เต็มของโปรเจกต์นี้ (แบ่งเป็น Phase 0-3 พร้อม checklist ละเอียดทุกหัวข้อ) อยู่ที่ **[FULLSTACK-INFRA-ROADMAP.md](FULLSTACK-INFRA-ROADMAP.md)** — นี่คือไฟล์หลักที่ใช้ track ความคืบหน้าจริง (ต่างจาก `ROADMAP.md` ที่เป็นลำดับการอ่าน `books/` ทั้งหมดแบบ textbook)

สรุป phase ของ BOP:
- Phase 0: Golang bot execution engine (worker pool, interface design)
- Phase 1: React/Next.js dashboard (รวม WebSocket real-time log — จุดที่ยากกว่า CRUD ทั่วไป)
- Phase 2: Infra (Docker/K8s/CI-CD → Observability → Auth/APISIX → DB scale/Redis → Kafka+Temporal → System Design → Security/SSRF)
- Phase 3: ฟีเจอร์ระดับ enterprise แรงบันดาลใจจาก Control-M (condition-based trigger, resource pool, flow control, approval gate ผ่าน Temporal signal, Gantt view, alert triage, config versioning) — ทำเมื่อ Phase 0-2 นิ่งแล้ว ไม่ใช่สิ่งที่ต้องมีก่อน

## กลยุทธ์เตรียมตัวเผื่อ layoff (นอกเหนือจากสกิลทางเทคนิค)

ประเด็นเหล่านี้คุยไว้แล้วและเจ้าของ repo รับทราบ ไม่ต้องอธิบายซ้ำถ้าไม่ถูกถาม:

- **ภาษาอังกฤษ** — เปิดตลาดบริษัท foreign-owned/remote ที่จ่ายดีกว่าและสนใจอายุน้อยกว่ามาก
- **Financial runway** 6-12 เดือน — ลดโอกาสตัดสินใจหางานแบบเร่งรีบ/กดราคาตัวเอง
- **Open source contribution** — น่าเชื่อถือกว่า personal project เฉยๆ เพราะมี code review จากคนอื่นยืนยัน
- **อย่าเป็น "รู้ทุกอย่างแต่ไม่มีจุดขายเดียว"** — backend/infra ต้องเป็นแกนหลักเสมอ frontend เป็นแค่ส่วนเสริมให้ครบ ไม่ใช่กลับกัน
- **เตรียม behavioral/leadership story** (STAR method) ไม่ใช่แค่เตรียม technical
- **เริ่มสร้าง network/visibility ตอนนี้** ก่อนเดือดร้อนจริง (บล็อก, LinkedIn, ตอบคำถามใน community)
- **ความเป็นจริงเรื่อง age discrimination ในตลาดไทย**: ประกาศงานที่กรอง "อายุไม่เกิน 35" ส่วนใหญ่เป็น HR pre-filter สำหรับตำแหน่ง junior-mid ทั่วไป ผ่าน job board/ATS — แต่ตำแหน่ง senior/staff/specialist และการเข้าตลาดผ่าน referral/network มักไม่ติดข้อจำกัดนี้ กลยุทธ์คือทำให้ตัวเอง "หายากพอ" (infra/system-design เฉพาะทาง) และเข้าตลาดผ่านช่องทางที่ข้ามการกรองด้วยอายุ

## หมายเหตุเรื่อง License ของ Stack ที่ใช้

ส่วนใหญ่ permissive เต็มรูปแบบ (Apache 2.0/MIT/BSD) ใช้เชิงพาณิชย์ได้ไม่มีเงื่อนไข — ยกเว้นบางตัวที่เปลี่ยน license ไปแล้วมีเงื่อนไขเพิ่ม (แต่ใช้ภายในบริษัทเองแบบปกติยังไม่มีปัญหา):

- **Redis** (7.4+) — RSAL/SSPL ไม่ใช่ OSI open source แล้ว ถ้าอยากชัวร์ 100% ใช้ **Valkey** แทน (fork BSD, ดูแลโดย Linux Foundation)
- **Grafana** — AGPLv3 (copyleft แต่ยังนับเป็น OSI open source) มีปัญหาเฉพาะถ้าแก้ source แล้วเอาไปให้บริการต่อ
- **HashiCorp Vault** — BUSL ตั้งแต่ 2023 ไม่ใช่ OSI open source แล้ว ทางเลือกที่ยัง MPL 2.0 คือ **OpenBao**
- **Docker Desktop** — เสียเงินถ้าบริษัทใหญ่ (พนักงาน>250/รายได้>$10M) แต่ Docker Engine/CLI ที่ใช้ตอน production ยังฟรี Apache 2.0

## โครงสร้าง repo นี้

- `books/` — หลักสูตรทฤษฎีเต็ม 17 volume (เขียนจบครบแล้ว ภาษาไทย) ใช้เป็นเนื้อหาอ้างอิงเชิงลึก
- `ROADMAP.md` — ลำดับการอ่าน `books/` ทั้งหมดแบบ textbook (ครบทุกหัวข้อ ไม่ได้ priority ตามเป้าหมายส่วนตัว)
- `FULLSTACK-INFRA-ROADMAP.md` — **ไฟล์หลักที่ใช้งานจริง** roadmap เฉพาะที่ priority ตามเป้าหมาย layoff-prep ผูกกับโปรเจกต์ BOP ทุกหัวข้อมีงานที่ต้องสร้างจริงกำกับ
- `projects/`, `labs/`, `source-code/` — ยังเป็น scaffold เปล่าเป็นส่วนใหญ่ รอถูกเติมเนื้อหาจริงตอนลงมือทำ BOP

## Coding Convention สำหรับโค้ดในโปรเจกต์นี้ (override default behavior)

- **ทุกโค้ดที่เขียนใน `projects/bot-orchestration-platform/` (และโค้ดอื่นในโปรเจกต์ BOP ต่อจากนี้) ต้องมี comment ภาษาไทยอธิบายการทำงานแบบละเอียด** — นี่คือการ override หลักการ "default ไม่ใส่ comment / อธิบายแค่ WHY" ที่ปกติควรทำ เพราะ repo นี้คือโปรเจกต์เรียนรู้ของเจ้าของ comment ที่อธิบายกลไกการทำงานทีละขั้นตอนมีประโยชน์ตอนกลับมาทบทวนทีหลัง ไม่ใช่ overhead ที่ควรเลี่ยง
- อธิบายทั้ง **"ทำอะไร"** และ **"ทำงานยังไง"** ไม่ใช่แค่ "ทำไมถึงทำแบบนี้" — ต่างจาก convention ทั่วไปที่มักบอกให้ comment แค่ WHY เพราะโค้ดที่ตั้งชื่อดีบอก WHAT อยู่แล้ว แต่เจ้าของ repo ต้องการใช้ comment เป็นเครื่องมือเรียนรู้ ภาษา/กลไกที่ยังไม่คุ้นเคย (เช่น semaphore pattern, mutex, context cancellation) ควรอธิบายละเอียดแม้จะดูเหมือน "อธิบาย WHAT" ก็ตาม
- ใช้กับไฟล์โค้ดที่เขียนขึ้นใหม่ทุกไฟล์ในโปรเจกต์นี้ ไม่ใช่แค่ตอนถูกขอ — เป็นค่า default ของการทำงานในโปรเจกต์นี้ตั้งแต่นี้ไป

## แนวทางเวลาช่วยเหลือ

- เช็ค `FULLSTACK-INFRA-ROADMAP.md` ก่อนเสมอเพื่อดูว่าอยู่ phase ไหน มี checklist อะไรที่ยังไม่ทำ
- ให้คำแนะนำที่ actionable/ลงมือทำได้จริง มากกว่าทฤษฎีเพิ่ม (ทฤษฎีมีครบใน `books/` อยู่แล้ว)
- ให้ feedback แบบตรงไปตรงมา ไม่ปลอบใจเกินจริง — เจ้าของ repo ขอไว้ชัดเจนว่าต้องการความเห็นที่ calibrated ไม่ใช่ cheerleading
