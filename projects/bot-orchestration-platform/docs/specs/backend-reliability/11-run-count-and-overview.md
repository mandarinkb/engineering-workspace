# 11: endpoint นับจำนวนและ Overview ที่ถูกต้องเมื่อข้อมูลเยอะ

Spec: [backend-reliability.md](../backend-reliability.md)

**What to build:** ตัวเลขในหน้า Overview ถูกต้องไม่ว่าจะมี Workflow Run กี่ตัว: จำนวนที่กำลังรันมาจากการนับฝั่ง server และรายการล้มเหลวล่าสุดมาจากการกรองฝั่ง server

**Blocked by:** 09 (แบ่งหน้าและ filter สถานะของ Workflow Run)

**Status:** ready-for-agent

- [ ] `GET /workflow-runs/count?status=` คืน `{ "count": n }`
- [ ] seam A: count ตรงกับจำนวนจริงเมื่อมี run มากกว่า 1 หน้า
- [ ] Overview: "Workflow กำลังรัน" มาจาก count และ "ล้มเหลวล่าสุด" มาจาก `status=failed&page_size=5`
- [ ] golangci-lint, vitest และ eslint ผ่าน; deploy ขึ้น Mini PC แล้วทดสอบผ่าน browser
