# TODO: Bot ดึงข้อมูลกราฟผ่าน Browser + วิเคราะห์เอง (ไม่พึ่ง AI Vision API) — Design เต็ม ยังไม่ Implement

> ⚠️ **เอกสารนี้ถูกแทนที่แล้ว** — scope ขยายใหญ่ขึ้นมาก (เพิ่ม database + backtesting + real-time display) จนใหญ่เกินกว่าจะเป็นแค่ bot 1 ตัวใน BOP ย้ายไปเป็นโปรเจกต์แยกต่างหากที่ [projects/trade-analyzer/TRADE-ANALYZER-TODO.md](../trade-analyzer/TRADE-ANALYZER-TODO.md) แล้ว — เอกสารนี้เก็บไว้เป็น **decision log** ของแนวคิดที่นำไปต่อยอด (เหตุผลที่เลือก network interception แทน pixel/vision) ไม่ใช่แผนที่ใช้งานจริงอีกต่อไป
>
> **สถานะ**: design doc เขียนไว้ล่วงหน้า ยังไม่เริ่ม implement — ต่อยอดจาก `price-watch` bot ([internal/bots/pricewatch/](internal/bots/pricewatch/pricewatch.go)) ที่ implement ไปแล้วก่อนหน้า **ไม่ได้แทนที่กัน**
>
> **เปลี่ยนทิศทางจากร่างแรก**: เวอร์ชันแรกของเอกสารนี้ออกแบบให้ถ่าย screenshot แล้วส่งให้ vision-capable LLM วิเคราะห์ (ทางเลือก A เดิม) — **ตัดสินใจเปลี่ยนแล้ว**: ดึงข้อมูลราคาดิบผ่าน browser โดยตรง (ไม่ตีความจากภาพ) แล้วเขียน logic วิเคราะห์เองทั้งหมด **ไม่พึ่ง AI API ภายนอกเลย** — ตัด section เรื่อง cost ของ vision API ออกเพราะไม่เกี่ยวแล้ว
>
> **ขอบเขตที่ยังคงเดิมจาก `price-watch`**: **ไม่ auto-trade** เป็นเครื่องมือช่วยวิเคราะห์เท่านั้น คนตัดสินใจเทรดเอง

## สารบัญ

1. [ทำไมยังต้องผ่าน Browser ทั้งที่ไม่ใช้ Vision แล้ว](#1-ทำไมยังต้องผ่าน-browser-ทั้งที่ไม่ใช้-vision-แล้ว)
2. [สถาปัตยกรรม: ต่อเข้า Browser ที่เปิดอยู่แล้ว](#2-สถาปัตยกรรม-ต่อเข้า-browser-ที่เปิดอยู่แล้ว)
3. [ดึงข้อมูลราคาจริงผ่าน Network Interception](#3-ดึงข้อมูลราคาจริงผ่าน-network-interception)
4. [เขียน Analysis Logic เอง](#4-เขียน-analysis-logic-เอง)
5. [Maintenance Cost ที่ต้องรู้ก่อนเริ่ม (แทนที่ vision API cost เดิม)](#5-maintenance-cost-ที่ต้องรู้ก่อนเริ่ม-แทนที่-vision-api-cost-เดิม)
6. [Open Questions](#open-questions)
7. [TODO Checklist](#todo-checklist)

---

## 1. ทำไมยังต้องผ่าน Browser ทั้งที่ไม่ใช้ Vision แล้ว

คำถามตรงๆ ที่ต้องตอบก่อน: ถ้าแค่อยากได้ตัวเลขราคาไปคำนวณเอง ทำไมไม่ใช้ `price-watch` (ยิง Binance API ตรงๆ) ไปเลย ต้องผ่าน browser ให้ยุ่งยากทำไม?

**คำตอบ**: `price-watch` ใช้ได้ดีเฉพาะ **crypto ที่มี exchange (Binance) เปิด public API ให้ฟรี** — แต่ผู้ใช้สนใจทั้ง **หุ้นและทองคำด้วย** ซึ่งส่วนใหญ่**ไม่มี public API ฟรีที่เรียกตรงๆ ได้ง่ายแบบ Binance** (ราคาทอง/หุ้นไทยมักอยู่หลัง provider ที่ต้องสมัคร/จ่ายเงิน หรือแสดงผลผ่านหน้าเว็บเท่านั้น) — การผ่าน browser คือทางเลือกที่ใช้ได้กับ**ทุกเว็บที่แสดงกราฟได้ ไม่ว่าจะมี public API หรือไม่** เพราะดึงข้อมูลจากสิ่งที่หน้าเว็บโหลดมาแสดงอยู่แล้ว

**สรุปขอบเขต**: bot นี้เกิดมาเพื่อรองรับ**สินทรัพย์ที่ไม่มี API ฟรีให้เรียกตรงๆ** (หุ้น/ทอง) — สำหรับ crypto ที่มี API ฟรีอยู่แล้ว ยังแนะนำใช้ `price-watch` ต่อไป (เร็วกว่า/เบากว่า/แม่นยำกว่า ไม่มีเหตุผลเปลี่ยน)

---

## 2. สถาปัตยกรรม: ต่อเข้า Browser ที่เปิดอยู่แล้ว

เหมือนร่างเดิม ไม่เปลี่ยน — ต่อเข้า Chrome ที่เปิดค้างไว้เองผ่าน Chrome DevTools Protocol (CDP) ด้วย [chromedp](https://github.com/chromedp/chromedp) แทนที่จะ spawn headless browser ใหม่ทุกครั้ง (เสีย state ที่มีค่า เช่น login, indicator/drawing ที่ตั้งไว้แล้ว):

```go
// เปิด Chrome เองก่อนด้วย remote debugging port (ทำครั้งเดียว ไม่ใช่ทุก job):
//   google-chrome --remote-debugging-port=9222
allocCtx, cancel := chromedp.NewRemoteAllocator(ctx, "http://localhost:9222")
```

---

## 3. ดึงข้อมูลราคาจริงผ่าน Network Interception

แทนที่จะถ่าย screenshot แล้วตีความภาพ — ดัก network response ที่หน้าเว็บโหลดเข้ามาเองตอนแสดงกราฟ (เว็บ chart แทบทุกที่ดึงราคาจริงผ่าน API ภายในของตัวเองอยู่ดี ซ่อนอยู่หลัง JS แค่ไม่ได้เปิดให้เรียกตรงๆ จากภายนอก) — chromedp ทำผ่าน CDP's Network domain ได้:

1. `network.Enable()` เปิดการดักฟัง network event ของ tab ที่ต่ออยู่
2. Subscribe event `network.EventResponseReceived` แล้วกรองเฉพาะ URL ที่ตรงกับ pattern ของ APIภายในเว็บนั้น (ต้องเปิด DevTools ดูเองก่อนว่าเว็บเป้าหมายเรียก endpoint ไหนตอนโหลดกราฟ — ขั้นตอนนี้ต้องทำต่อเว็บทีละที่ เพราะแต่ละเว็บ endpoint ไม่เหมือนกัน)
3. เรียก `network.GetResponseBody` ดึงเนื้อหา response ออกมา parse เป็นราคา/แท่งเทียน

**ข้อดีเทียบกับอ่าน pixel**: ได้ตัวเลขแม่นยำ 100% ไม่ต้องตีความจากภาพ ไม่ต้องกังวลเรื่อง theme สี/zoom/resolution ของกราฟเลย

**ข้อเสีย**: เปราะบางกว่า public API ตรงๆ (เว็บเปลี่ยน internal API endpoint เมื่อไหร่ก็พังทันที) — ดูหัวข้อ 5

---

## 4. เขียน Analysis Logic เอง

เมื่อมีราคาดิบแล้ว (เหมือนที่ `price-watch` มีอยู่แล้ว) เขียน logic วิเคราะห์เพิ่มเองได้ 2 ระดับ:

**ระดับที่ทำแล้ว (ใน `price-watch`)**: indicator เชิงตัวเลขล้วนๆ เช่น RSI — สูตรตายตัว คำนวณตรงไปตรงมา

**ระดับถัดไปที่น่าสนใจกว่า**: **pattern detection เชิงรูปทรง** (สิ่งที่ตัวเลข RSI เดี่ยวๆ จับไม่ได้) เขียนเป็น classic algorithm ได้โดยไม่ต้องมี ML/AI เลย เช่น:

- **หา local peak/trough** จากชุดราคา (จุดที่ราคาสูง/ต่ำกว่าเพื่อนบ้านทั้งสองข้าง) — พื้นฐานของการหา pattern แบบ double-top/double-bottom
- **Trendline detection** — fit เส้นตรงผ่านชุดจุด peak หรือ trough ต่อเนื่องกัน (linear regression พื้นฐาน) เพื่อหาแนวโน้ม
- **Support/Resistance level** — หาราคาที่ราคาสะท้อนกลับ (bounce) ซ้ำๆ หลายครั้งในอดีต (นับความถี่ที่ราคาเข้าใกล้ระดับหนึ่งแล้วเปลี่ยนทิศทาง)

ทั้งหมดนี้เป็น **algorithm ธรรมดาที่เขียนเองได้ 100% ไม่ต้อง train/พึ่ง model ภายนอกเลย** — ตรงกับที่ต้องการ ("การวิเคราะห์เขียนขึ้นมาเองด้วย")

---

## 5. Maintenance Cost ที่ต้องรู้ก่อนเริ่ม (แทนที่ Vision API Cost เดิม)

ไม่มีค่าใช้จ่ายเป็นเงินแล้ว (ตัด vision API ออกไปแล้ว) — แต่มี**ค่าใช้จ่ายเป็นเวลาดูแลแทน** ที่ต้องรู้ไว้ก่อนเริ่ม:

- **เว็บเปลี่ยน internal API endpoint/response format เมื่อไหร่ก็พังทันที** — ต่างจาก public API ที่มี versioning/backward compatibility ให้ (แบบ Binance) internal API ของเว็บทั่วไปไม่มีสัญญาเรื่องความเสถียรให้เลย เปลี่ยนได้ตลอดโดยไม่แจ้งล่วงหน้า
- ต้องเปิด DevTools เองทุกครั้งที่เพิ่มเว็บ target ใหม่ เพื่อหา endpoint ที่ถูกต้อง (ไม่มีทางอัตโนมัติ)

---

## Open Questions

1. **เว็บ chart ไหนสำหรับหุ้น/ทอง** — ต้องเช็ค Terms of Service ก่อนเสมอ (ตรงกับหลักการเดิมของโปรเจกต์ใน `CLAUDE.md`: "ไม่เสี่ยงชน ToS ของเว็บเป้าหมาย") เลือกเว็บที่มั่นใจเรื่องนี้ก่อนเริ่ม implement
2. **Endpoint ภายในของเว็บเป้าหมายคืออะไร** — ต้องเปิด DevTools เองสำรวจก่อนเขียนโค้ด (เอกสารนี้บอกกลไกทั่วไปได้ แต่ endpoint จริงต้องหาเองต่อเว็บ)
3. **เก็บข้อมูลราคาที่ดึงมาไว้ยาวไหม** (สำหรับคำนวณ trendline/support-resistance ที่ต้องใช้ประวัติย้อนหลัง) — กระทบว่าต้องมี storage เพิ่มหรือไม่ (ต่างจาก `price-watch` ที่ดึงสดทุกครั้งพอสำหรับ RSI แบบ snapshot)

## TODO Checklist

### Phase 1: Browser Connection POC
- [ ] เปิด Chrome ด้วย `--remote-debugging-port=9222`
- [ ] `go get github.com/chromedp/chromedp`
- [ ] ทดสอบต่อเข้า browser ที่เปิดอยู่ผ่าน `chromedp.NewRemoteAllocator` ให้สำเร็จก่อน (proof of concept เปล่าๆ ยังไม่ต้องดึงราคา)

### Phase 2: หา + ดัก Internal API ของเว็บเป้าหมาย
- [ ] ตัดสินใจ Open Question ข้อ 1 (เว็บไหน + เช็ค ToS แล้ว)
- [ ] เปิด DevTools ของเว็บนั้นเอง สำรวจหา network request ที่โหลดข้อมูลกราฟ (ดู Open Question ข้อ 2)
- [ ] เขียนโค้ด `network.Enable()` + subscribe `network.EventResponseReceived` + `network.GetResponseBody` ดึงข้อมูลออกมาให้ได้จริง

### Phase 3: เขียน Analysis Logic เอง
- [ ] เริ่มจาก local peak/trough detection ก่อน (ง่ายสุด เป็นฐานของอย่างอื่น)
- [ ] ต่อยอดเป็น trendline detection / support-resistance
- [ ] เขียน test แบบ table-driven เหมือน `computeRSI`/`interpretRSI` ใน `price-watch` (ข้อดีของ logic แบบนี้คือ test ได้ง่ายมาก เพราะเป็น pure function ล้วนๆ ไม่ต้องพึ่ง network เลยตอน test)

### Phase 4: ห่อเป็น Bot ตาม Pattern เดิม
- [ ] สร้าง `internal/bots/chartreader/` (ตั้งชื่อใหม่ให้ตรงกับที่ทำจริง — ไม่ใช้คำว่า "vision" แล้วเพราะไม่มี vision component อีกต่อไป) implement `bot.Bot` interface
- [ ] `ConfigSchema()`: `chrome_debug_url` (ที่อยู่ remote debugging endpoint)
- [ ] `Run()`: ต่อ browser → ดัก network → parse ราคา → รัน analysis logic → คืน `Result{Summary, Data}`

## Reference

- [internal/bots/pricewatch/pricewatch.go](internal/bots/pricewatch/pricewatch.go) — ต้นแบบโครง Bot + ตัวอย่าง indicator เชิงตัวเลข (RSI) ที่ทดสอบได้ง่าย
- [internal/bot/bot.go](internal/bot/bot.go) — `Bot` interface ที่ต้อง implement
- [chromedp](https://github.com/chromedp/chromedp) — library หลักที่ใช้ต่อเข้า browser + ดัก network
- `CLAUDE.md` (root ของ workspace) — หลักการ "ไม่เสี่ยงชน ToS" ที่ต้องใช้ตัดสินใจ Open Question ข้อ 1
