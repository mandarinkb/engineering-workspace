# Trade Analyzer — สรุปความต้องการ (สำหรับเปิด session ใหม่)

> สรุปสั้นๆ ไว้ให้เข้าใจเป้าหมายเร็ว — รายละเอียดเชิงเทคนิคทั้งหมดอยู่ที่ [TRADE-ANALYZER-TODO.md](TRADE-ANALYZER-TODO.md) (phase/checklist เต็ม)

## เป้าหมาย

โปรเจกต์ระยะยาว (ไม่ใช่ short-term side project) — เครื่องมือช่วยวิเคราะห์กราฟหุ้น/ทอง/คริปโต เจ้าของ repo บอกไว้ตรงๆ ว่า **ถ้าเครื่องมือนี้แม่นยำพอในอนาคต อยากใช้แทนอาชีพ trader** (เผื่อวันที่ทำงานประจำไม่ได้แล้ว)

## Requirement หลัก 4 ข้อ (คำเดิมของเจ้าของ repo)

1. Bot อ่านค่ากราฟแท่งเทียนผ่าน browser ได้ — ไม่จำกัดภาษาว่าต้องเป็น Go เอาภาษาที่ lib สะดวก ขอความแม่นยำเป็นหลัก
2. อ่านข้อมูลได้แล้วเก็บลง database ไว้วิเคราะห์
3. เขียนเครื่องมือวิเคราะห์ขึ้นมาเอง (**ไม่ใช้ AI vision API ให้คนอื่นวิเคราะห์แทน**)
4. แสดงผลวิเคราะห์บอกว่าตอนนี้ควรเข้า order ไหม — real-time ยิ่งดี — **ไม่ auto-trade เอง คนตัดสินใจเอง**

**เป้าหมายเพิ่มภายหลัง**: อยากมี AI model เป็นของตัวเอง ตัดสินใจได้ทันทีว่าควรเข้า order แบบไหน (ยัง "แนะนำ" ไม่ใช่ "สั่งเอง" เหมือนเดิม)

## การตัดสินใจสำคัญที่ทำไปแล้ว (ห้ามเสนอย้อนกลับโดยไม่มีเหตุผลใหม่)

- **แยกเป็นโปรเจกต์ใหม่** `projects/trade-analyzer/` ไม่ใช่ bot ใน BOP (Go) เดิม — เพราะ scope ใหญ่เกิน
- **ภาษา: Python** (Playwright, pandas/numpy, PostgreSQL)
- **Browser**: ต่อเข้า Chrome ที่เปิดค้างไว้เอง (login แล้ว) ผ่าน CDP — **ไม่ spawn browser ใหม่**
- **วิธีอ่านข้อมูล**: ดัก network response/WebSocket ที่หน้าเว็บโหลดเอง (ตัวเลขดิบแม่นยำ 100%) — **ไม่ใช้ AI vision อ่านภาพ pixel** (ลองแล้วตัดออกเพราะมี cost + ไม่แม่นยำเท่า)
- **Analysis**: เขียนเอง — RSI (สูตรเดียวกับ `internal/bots/pricewatch/` ของ BOP), peak/trough detection, trendline, support/resistance
- **Backtesting เป็น phase บังคับ** ก่อนเชื่อ signal ใดๆ (เพิ่มเข้ามาเพราะจำเป็น ไม่ใช่ requirement เดิม)
- **AI model (ถ้าทำ)**: ต้องมาหลัง backtesting ผ่านเกณฑ์แล้วเท่านั้น — แนะนำ Gradient Boosting (XGBoost/LightGBM) ไม่ใช่ Reinforcement Learning (ยากเกินไปสำหรับ solo project) ไม่ใช่ LLM สำเร็จรูป (ไม่ใช่ "ของตัวเอง")
- **Risk management/position sizing (Component 7)**, **paper trading gate (Component 8)**, **operational monitoring/backup (Component 9)** — เพิ่มเข้ามาเพราะจำเป็น ไม่ใช่ requirement เดิม เช่นเดียวกับ backtesting — ห้ามใช้เงินจริงก่อนผ่าน paper trading period เต็มโดยไม่มี kill-switch trigger
- **ไม่ auto-trade ทุกกรณี** — ทุก component (rule-based หรือ AI model) แสดงคำแนะนำ+เหตุผล+สถิติให้คนกดยืนยันเองเสมอ

## Reality Check ที่คุยกันไว้แล้ว (อย่าอธิบายซ้ำ แต่ต้องยึดหลักนี้เสมอ)

Retail trader ส่วนใหญ่ขาดทุนระยะยาว (สถิติเปิดเผยจาก broker หลายแห่ง มักอยู่ 70-90%+) — โปรเจกต์นี้มีคุณค่าแน่นอนในฐานะเครื่องมือเรียนรู้/portfolio ไม่ว่าผลจะเทรดกำไรหรือไม่ แต่**ห้ามวางแผนอนาคตการเงินโดยอิงว่ามันจะแม่นยำพอเลี้ยงชีพ** จนกว่าจะผ่าน backtest เข้มงวดระดับปี + paper trade จริง

## เอกสารที่เกี่ยวข้อง

- [TRADE-ANALYZER-TODO.md](TRADE-ANALYZER-TODO.md) — แผนเทคนิคเต็ม 9 component, phase checklist ละเอียด
- [../bot-orchestration-platform/CHART-VISION-BOT-TODO.md](../bot-orchestration-platform/CHART-VISION-BOT-TODO.md) — เอกสารเวอร์ชันแรก (ถูกแทนที่แล้ว เก็บไว้เป็น decision log)
- [../bot-orchestration-platform/internal/bots/pricewatch/](../bot-orchestration-platform/internal/bots/pricewatch/pricewatch.go) — bot ตัวแรกใน BOP เดิม (Go) ที่จุดประกายเรื่องนี้ทั้งหมด
