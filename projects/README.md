# Projects

โปรเจกต์จริงขนาดใหญ่ที่รวมความรู้จากหลาย [`books/`](../books/) เข้าด้วยกัน ต่างจาก [`source-code/`](../source-code/) ตรงที่นี่คือระบบที่ deploy ได้จริง ไม่ใช่แค่สนิปเป็ตตัวอย่าง

## รายการโปรเจกต์

| โปรเจกต์ | จุดโฟกัส |
|---|---|
| [monolith/](monolith/README.md) | จุดเริ่มต้นแบบ monolith ก่อนแตกเป็น microservices |
| [ecommerce/](ecommerce/README.md) | Final Project — รวมทุก Volume เข้าด้วยกัน |
| [payment/](payment/README.md) | Consistency, idempotency, Saga |
| [chat/](chat/README.md) | Realtime, WebSocket, pub/sub |

ลำดับแนะนำ: เริ่มจาก `monolith/` เพื่อฝึกพื้นฐานก่อน แล้วค่อยแยกเป็น `payment/` และ `chat/` เป็น service ย่อย ก่อนประกอบทุกอย่างเข้าด้วยกันใน `ecommerce/`
