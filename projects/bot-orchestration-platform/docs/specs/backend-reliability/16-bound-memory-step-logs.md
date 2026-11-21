# 16: log ของ Step ใน memory ของ bop-worker ต้องมีขีดจำกัด

Spec: [backend-reliability.md](../backend-reliability.md)

**ที่มา (ผล code review ของ ticket 01-12):** เมื่อ `bop-worker` ไม่ได้ตั้ง `BOP_DATABASE_URL` มันใช้ logger แบบ memory ซึ่งเก็บ log ของ Step ใน map ที่ไม่มีการลบหรือจำกัดขนาด Schedule ที่รันทุกนาทีเพิ่มหลายบรรทัดต่อ Attempt ต่อ run ราว 1,440 run ต่อวัน หน่วยความจำจึงโตจนกว่า process จะถูกฆ่า และ log เหล่านี้ API อ่านไม่ได้อยู่แล้ว เพราะ `bop-api` เป็นคนละ process ที่มี logger เป็นของตัวเอง

**Status:** deferred — ตัดสินใจร่วมกับผู้ใช้ให้เลื่อนไปก่อน เกิดเฉพาะโหมดไม่มีฐานข้อมูล compose ที่ root ตั้งค่านี้ไว้เสมอ จึงไม่กระทบ Mini PC ยังไม่ได้ตรวจเทียบว่าก่อน ticket 08 log ของ Step ใน worker โตแบบเดียวกันหรือไม่ (ผู้เขียนสงสัยว่าเป็นอย่างนั้น จึงอาจไม่ใช่ปัญหาใหม่)

**กลับมาทำเมื่อ:** มีคนรัน `bop-worker` โดยไม่ตั้งฐานข้อมูลเป็นเวลานาน (เช่นรันแบบ `go run` ค้างไว้ข้ามวัน) หรือเมื่อมีใครเจอ worker ถูกฆ่าเพราะ memory

**ทางเลือก:** ในโหมด memory ของ worker ไม่เก็บ log ของ Step เลย (API อ่านไม่ได้อยู่แล้ว) หรือจำกัดจำนวนด้วย ring buffer ของ workflow ล่าสุด เช่น 1,000 รายการ
