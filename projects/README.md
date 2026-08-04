# Projects

โปรเจกต์จริงขนาดใหญ่ที่รวมความรู้จากหลาย [`books/`](../books/) เข้าด้วยกัน ต่างจาก [`source-code/`](../source-code/) ตรงที่นี่คือระบบที่ deploy ได้จริง ไม่ใช่แค่สนิปเป็ตตัวอย่าง

## รายการโปรเจกต์

| โปรเจกต์ | จุดโฟกัส |
|---|---|
| [bot-orchestration-platform/](bot-orchestration-platform/README.md) | **Flagship project ที่ใช้งานจริง** — ดูรายละเอียดที่ [FULLSTACK-INFRA-ROADMAP.md](../FULLSTACK-INFRA-ROADMAP.md) |
| [monolith/](monolith/README.md) | จุดเริ่มต้นแบบ monolith ก่อนแตกเป็น microservices (แบบฝึกหัดสำรอง) |
| [ecommerce/](ecommerce/README.md) | Final Project เดิม — รวมทุก Volume เข้าด้วยกัน (แบบฝึกหัดสำรอง) |
| [payment/](payment/README.md) | Consistency, idempotency, Saga (แบบฝึกหัดสำรอง) |
| [chat/](chat/README.md) | Realtime, WebSocket, pub/sub (แบบฝึกหัดสำรอง) |

**โปรเจกต์ที่กำลังพัฒนาจริงคือ `bot-orchestration-platform/`** ตาม roadmap เฉพาะที่ผูกกับเป้าหมาย layoff-prep — ส่วน `monolith/`, `ecommerce/`, `payment/`, `chat/` เป็นแนวคิดตั้งต้นที่ยังใช้เป็นแบบฝึกหัดแยกได้ถ้าอยากลองแพทเทิร์นเฉพาะทาง (เช่น payment consistency, realtime chat) แต่ไม่ใช่ทางหลักอีกต่อไป
