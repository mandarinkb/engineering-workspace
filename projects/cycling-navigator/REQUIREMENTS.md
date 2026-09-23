# Cycling Navigator — สรุปความต้องการ (สำหรับเปิด session ใหม่)

> สรุปสั้นๆ ไว้ให้เข้าใจเป้าหมายเร็ว — รายละเอียดเชิงเทคนิคทั้งหมด (Phase 0 analysis เต็ม + roadmap) อยู่ที่ [CYCLING-NAVIGATOR-TODO.md](CYCLING-NAVIGATOR-TODO.md)

## เป้าหมาย

แอพมือถือ (Flutter, iOS+Android) สำหรับนักปั่นจักรยาน — GPS navigation + route planning ที่ทำงาน **แบบ offline-first เต็มรูปแบบ ไม่มี backend server** ตั้งแต่ v1 (ไม่มี REST/GraphQL API ของตัวเอง, ไม่มี cloud DB, ไม่มี user account)

## ความสัมพันธ์กับกลยุทธ์ layoff-prep

**เป็นโปรเจกต์คู่ขนานแยกเรื่องจาก [BOP](../bot-orchestration-platform/) — ไม่ใช่ตัวแทนเป้าหมาย fullstack (React/Next.js) เดิม** เจ้าของ repo ยืนยันไว้ตรงๆ ว่าเกี่ยวกับ layoff-prep แต่เป็นคนละ track เพราะเป็นความสนใจส่วนตัว (มี use case จักรยานจริง) ไม่ใช่ตัวขาย fullstack หลัก

**ข้อตกลงเรื่อง priority**: ทำแบบ **time-boxed** (เช่น เวลาว่าง/weekend) ห้ามแซง priority ของ BOP ที่ยังค้างอยู่ (Postgres persistence, ทำ Jenkins pipeline ให้รันจบจริง) — ดู `CLAUDE.md` หลักการ "ผูกกับโปรเจกต์เดียวที่โตขึ้นเรื่อยๆ"

## Source ของ requirement

Requirement เต็มมาจาก prompt ยาวที่เจ้าของ repo ส่งมาเอง (system prompt แบบ senior mobile architect spec 49 หัวข้อ) — สรุปสาระสำคัญที่ล็อกไว้แล้ว:

- **Stack**: Flutter/Dart, Riverpod, MapLibre (ไม่ผูก Google Maps ตายตัว), OpenStreetMap เป็น map data, SQLite-based local DB
- **ห้าม**: backend/API/cloud DB/auth server ของตัวเองในระยะแรก (ออกแบบ interface ให้เพิ่ม remote implementation ได้ทีหลังเท่านั้น)
- ต้องทำงานได้ **100% ใน airplane mode** หลัง onboarding/ดาวน์โหลดแผนที่ครั้งแรก — นี่คือ acceptance criteria หลักของ MVP
- Routing ต้องคำนวณ **on-device** ไม่ใช่ยิง routing API ระยะไกล
- Phase 0 (architecture/analysis) ต้องเสร็จและ **รอ approval ก่อนเขียนโค้ด Phase 1**

## การตัดสินใจสำคัญที่ทำไปแล้วใน Phase 0 (ห้ามเสนอย้อนกลับโดยไม่มีเหตุผลใหม่)

- **Local DB: Drift** (ไม่ใช่ Isar/Hive — ผู้เขียนเดิมเลิกดูแลแล้ว เหลือ community maintain เท่านั้น เสี่ยงระยะยาว)
- **Map tile format: PMTiles** ผ่าน official `maplibre_gl` plugin (รองรับ PMTiles + offline region ในตัว ไม่ต้องเขียน tile server เอง)
- **Routing engine primary candidate: Valhalla ผ่าน Dart FFI** — แต่**ต้องผ่าน feasibility spike ก่อน** (Phase 6a) เพราะไม่มี official mobile build artifact ยัง (มี GitHub discussion เปิดอยู่ ยังไม่ปิด ณ ตอนที่เขียนเอกสารนี้) — นี่คือความเสี่ยงทางเทคนิคที่สูงสุดของทั้งโปรเจกต์
- **Fallback routing ถ้า spike ล้มเหลว: GraphHopper บน Android เท่านั้น** (iOS support เป็น community fork ไม่ official — ยอมรับว่า iOS อาจต้องปล่อย routing ทีหลัง)
- **ตัดสิทธิ์ BRouter ตั้งแต่แรก** — ไม่มี iOS เลย (Android-only, architecture เป็น AIDL service ที่พึ่งแอปอื่น) ไม่ตรง requirement ที่ต้องมีทั้ง 2 platform

## Reality Check ที่คุยกันไว้แล้ว (อย่าอธิบายซ้ำ แต่ต้องยึดหลักนี้เสมอ)

- **Offline routing ข้ามแพลตฟอร์ม (iOS+Android) พร้อมกันคือจุดเสี่ยงสุดของโปรเจกต์** — ทีมใหญ่ (OsmAnd, Organic Maps, Komoot) ใช้เวลาหลายปีสร้างเอง solo dev ควรยอมรับตั้งแต่ต้นว่า MVP ของ routing ต้อง **จำกัดพื้นที่ทดสอบเล็กมาก** (เช่นแค่กรุงเทพฯ) ไม่ใช่ทั่วประเทศ/ทั่วโลกทันที
- **Map tiles กับ routing graph เป็นข้อมูลคนละชุด** ต้องมี pipeline สร้างสองอันแยกกัน (PMTiles สำหรับแผนที่, Valhalla tile set สำหรับ routing) ก่อนจะใช้งานได้จริงในพื้นที่ไหนก็ตาม — ไม่ใช่แค่ "ดาวน์โหลดแผนที่แล้วปั่นได้เลย"
- **iOS background location** จะโดน Apple review ตรวจเข้มเรื่อง justification — แผน v1 ควรเริ่มจาก foreground tracking ก่อน แล้วค่อยขอ background ทีหลังเมื่อมี use case ชัดเจนพอ

## เอกสารที่เกี่ยวข้อง

- [CYCLING-NAVIGATOR-TODO.md](CYCLING-NAVIGATOR-TODO.md) — Phase 0 deliverable เต็ม (stack, architecture diagram, routing engine comparison, offline map strategy, storage architecture, schema, interfaces, roadmap phase 0-12, technical risks, licensing) + open questions ที่รอคำตอบก่อนเข้า Phase 1
