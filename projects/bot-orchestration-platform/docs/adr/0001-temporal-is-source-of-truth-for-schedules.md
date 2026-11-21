# Temporal เป็นแหล่งข้อมูลจริงเพียงที่เดียวของ Schedule

Schedule ทั้งตารางเวลา สถานะ pause และ steps ของ Workflow อยู่ใน Temporal Schedule อยู่แล้ว (steps อยู่ใน action args) เราจึงอ่าน steps จาก Temporal ตรงๆ และเลิกเก็บ Schedule ไว้ใน repository ของ BOP เอง แม้จะเพิ่ม PostgreSQL แล้วก็ตาม — PostgreSQL เก็บเฉพาะ Job และ log

## Considered Options

- **เก็บ Schedule ไว้ใน PostgreSQL คู่กับ Temporal** (แบบเดิมที่เก็บใน memory): ข้อมูล 2 ชุดไม่ตรงกันได้ เช่นลบ schedule ผ่าน Temporal UI แล้วข้อมูลยังค้างใน DB หรือ schedule ที่ `bop-worker` สร้างตรงผ่าน Temporal client ไม่มีข้อมูลใน DB เลยจึงไม่เห็น steps — ต้องเขียนโค้ด sync เพิ่มโดยไม่ได้อะไรที่ Temporal ให้ไม่ได้

## Consequences

- อ่านรายละเอียด Schedule ต้องถอดรหัส action args จาก Temporal ทุกครั้ง ข้อมูลที่ Temporal ไม่มีที่เก็บ (เช่นวันที่สร้างตามมุมของ BOP) จะไม่มี
- ถ้าวันหนึ่งต้องการข้อมูลของ Schedule ที่ Temporal เก็บไม่ได้ ให้เก็บแยกเป็นตารางเสริมที่อ้างด้วย schedule ID ไม่ใช่ย้าย steps กลับมาเก็บซ้ำ
