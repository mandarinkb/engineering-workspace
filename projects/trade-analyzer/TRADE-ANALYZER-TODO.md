# TODO: Trade Analyzer — เก็บ+วิเคราะห์กราฟระยะยาว (Design เต็ม ยังไม่ Implement)

> **ความสัมพันธ์กับเอกสารก่อนหน้า**: scope ใหญ่ขึ้นมากจาก [CHART-VISION-BOT-TODO.md](../bot-orchestration-platform/CHART-VISION-BOT-TODO.md) (ที่ออกแบบเป็นแค่ "bot 1 ตัวใน BOP") — ตอนนี้มี 9 component (browser reader, database, analysis engine, backtesting, real-time display, **custom AI model เป้าหมายระยะยาว**, risk management/position sizing, paper trading gate, operational monitoring/backup) ใหญ่เกินกว่าจะเป็นแค่ `internal/bots/X` ตัวเดียว จึงแยกเป็น **โปรเจกต์ใหม่ต่างหาก** (`projects/trade-analyzer/`) เอกสารเดิมยังอยู่เป็น decision log ไม่ลบทิ้ง แต่ถือว่าถูกแทนที่ด้วยเอกสารนี้แล้ว
>
> **ภาษา**: ไม่ใช่ Go เหมือน BOP หลัก — ใช้ **Python** (เหตุผลในหัวข้อ 2) โปรเจกต์นี้จึงเป็น process/repo แยกอิสระจาก `bot-orchestration-platform` โดยสมบูรณ์ (เชื่อมกลับเข้า BOP ทีหลังได้ผ่าน pattern ที่มีอยู่แล้ว — ดูหัวข้อ 13)

## สารบัญ

1. [Reality Check ที่ต้องอ่านก่อนเริ่ม](#1-reality-check-ที่ต้องอ่านก่อนเริ่ม)
2. [เลือกภาษา/Stack](#2-เลือกภาษาstack)
3. [สถาปัตยกรรมภาพรวม](#3-สถาปัตยกรรมภาพรวม)
4. [Component 1: Browser Data Collector](#4-component-1-browser-data-collector)
5. [Component 2: Database Storage](#5-component-2-database-storage)
6. [Component 3: Analysis Engine (เขียนเอง)](#6-component-3-analysis-engine-เขียนเอง)
7. [Component 4: Backtesting (เพิ่มเข้ามาเอง — จำเป็นก่อนเชื่อผลวิเคราะห์)](#7-component-4-backtesting-เพิ่มเข้ามาเอง--จำเป็นก่อนเชื่อผลวิเคราะห์)
8. [Component 5: Real-time Display + Signal Output](#8-component-5-real-time-display--signal-output)
9. [Component 6: Custom AI Model (เป้าหมายระยะยาว)](#9-component-6-custom-ai-model-เป้าหมายระยะยาว)
10. [Component 7: Risk Management & Position Sizing](#10-component-7-risk-management--position-sizing)
11. [Component 8: Paper Trading Gate](#11-component-8-paper-trading-gate)
12. [Component 9: Operational Monitoring & Backup](#12-component-9-operational-monitoring--backup)
13. [ความสัมพันธ์กับ BOP เดิม](#13-ความสัมพันธ์กับ-bop-เดิม)
14. [Open Questions](#open-questions)
15. [TODO Checklist](#todo-checklist)
16. [Reference](#reference)

---

## 1. Reality Check ที่ต้องอ่านก่อนเริ่ม

ขอพูดตรงๆ ตามที่ขอไว้เสมอในโปรเจกต์นี้ ("ไม่ปลอบใจเกินจริง") เพราะประโยคที่ว่า **"ถ้าเครื่องมือนี้แม่นยำ จะมาทำงานเป็น trader แทน"** เป็นการวางแผนที่มีความเสี่ยงสูงมากถ้าไม่มีข้อเท็จจริงพวกนี้กำกับไว้ก่อน:

- **ข้อมูลเปิดเผยของ broker/regulator จำนวนมาก** (forex/CFD broker ที่กำกับดูแลในหลายประเทศต้องเปิดเผยสถิตินี้ตามกฎหมาย) **รายงานตรงกันว่าบัญชี retail trader ส่วนใหญ่ (มักอยู่ในช่วง 70-90% ขึ้นไปแล้วแต่ broker/ตลาด) ขาดทุนในระยะยาว** ไม่ใช่แค่ไม่กี่เปอร์เซ็นต์ — นี่ไม่ใช่เรื่องของ "เครื่องมือดีพอหรือยัง" อย่างเดียว แต่เป็นธรรมชาติของตลาดที่แข่งกับสถาบันการเงิน/algo fund ที่มีข้อมูล+ทุน+ความเร็วเหนือกว่ามาก
- **Indicator/pattern ที่เขียนเองในเอกสารนี้ (RSI, trendline, support/resistance) เป็นสูตรสาธารณะที่ตลาดรู้กันหมดแล้ว** เหมือนที่คุยกันไปตอนคุยเรื่อง `price-watch` — ไม่มีหลักฐานว่าใช้เดี่ยวๆ แล้วเอาชนะตลาดได้สม่ำเสมอ
- **"แม่นยำ" ต้องพิสูจน์ด้วย backtest ที่เข้มงวดมาก ไม่ใช่ดูด้วยตาว่า signal ดูสมเหตุสมผล** — backtest ที่ดีต้องมี out-of-sample testing (ทดสอบกับข้อมูลที่โมเดล/กฎไม่เคยเห็นตอนออกแบบ), หักค่าธรรมเนียม/spread/slippage ให้ครบ, และทดสอบข้ามหลายช่วงตลาด (ขาขึ้น/ขาลง/sideway) — เอกสารนี้ใส่ **Component 4: Backtesting เป็น phase บังคับ** ก่อนจะไปถึง "บอกว่าควรเข้า order" เพราะไม่งั้นไม่มีทางรู้เลยว่า signal ที่เขียนขึ้นมามีค่าจริงหรือแค่ดูดีตอนมองย้อนหลังไม่กี่ครั้ง

**คำแนะนำที่ให้ตรงๆ**: ทำโปรเจกต์นี้ต่อไปได้เต็มที่ในฐานะ **เครื่องมือเรียนรู้ + portfolio ระยะยาว** (มีคุณค่าจริงไม่ว่าผลจะเทรดกำไรหรือไม่ — ฝึก data engineering, backend, algorithm design, statistics) แต่**อย่าวางแผนอนาคตการเงินโดยอิงกับสมมติฐานว่ามันจะ "แม่นยำพอจะเลี้ยงชีพได้"** จนกว่าจะผ่าน backtest ที่เข้มงวดมากๆ เป็นระยะเวลานาน (หลักปี ไม่ใช่หลักเดือน) แล้วยังต้องผ่านการ paper trade (จำลองเทรดด้วยเงินปลอมตามสัญญาณจริงแบบ real-time) อีกระยะก่อนถึงจะพอเชื่อถือได้ระดับหนึ่ง — และถึงตอนนั้นก็ยังควรมี safety net อื่นเสมอ ไม่ทิ้งงานประจำเพราะเครื่องมือตัวเดียว

ไม่ได้บอกว่าห้ามทำ — แค่ต้องตั้งความคาดหวังให้ตรงกับความเป็นจริงตั้งแต่ต้น

---

## 2. เลือกภาษา/Stack

**Python** — เหตุผล (ตรงกับที่ขอไว้ "ไม่จำกัดภาษา ขอความสะดวกเรื่อง lib"):

| งาน | Library | เหตุผล |
|---|---|---|
| Browser automation | [Playwright for Python](https://playwright.dev/python/) | ต่อเข้า browser ที่เปิดอยู่แล้วผ่าน CDP ได้เหมือน chromedp (`connect_over_cdp`) แต่ API สะดวกกว่า โดยเฉพาะเรื่อง network/WebSocket interception |
| Data analysis | pandas + numpy | มาตรฐานอุตสาหกรรม เอกสาร/ตัวอย่างเยอะที่สุดในบรรดาภาษาทั้งหมดสำหรับงานนี้ |
| Database | PostgreSQL + `psycopg2`/`asyncpg` | ใช้ตัวเดียวกับที่ roadmap หลักของ BOP วางแผนไว้อยู่แล้ว (ดู `books/05-postgresql/`) ความรู้ต่อยอดกันได้ |
| TA formula (ตัวเลือก ไม่บังคับ) | `pandas-ta` | ใช้เฉพาะสูตรคณิตศาสตร์มาตรฐาน (RSI/MACD ฯลฯ) ที่ไม่มีเหตุผลจะเขียนเองใหม่ — ส่วน **กฎ/กลยุทธ์การตัดสินใจยังเขียนเองทั้งหมด** ตรงตามที่ต้องการ |
| Backtesting loop | เขียนเอง (custom) | เริ่มจากเขียน backtest loop ง่ายๆ เองก่อน (ตรงกับที่อยากเขียนเครื่องมือวิเคราะห์เอง) ค่อยพิจารณา framework สำเร็จรูป (เช่น `backtrader`/`vectorbt`) ทีหลังถ้าโค้ดเองเริ่มซับซ้อนเกินจัดการ |
| Real-time display (เริ่มต้น) | Terminal (`rich` library) หรือ query SQL ตรงๆ | เริ่มจากง่ายที่สุดก่อน เว็บ dashboard สวยๆ เป็น phase หลังสุด (ดูหัวข้อ 8) |

**ข้อแลกเปลี่ยนที่ต้องรู้**: แยกภาษา/repo จาก BOP เดิม (Go) โดยสมบูรณ์ — ไม่ได้ผูกกับ `bot.Bot` interface เดิมเลยในช่วงแรก เชื่อมกลับเข้า BOP ได้ทีหลังถ้าต้องการ (หัวข้อ 9) แต่ไม่ใช่ข้อบังคับของ phase แรกๆ

---

## 3. สถาปัตยกรรมภาพรวม

```
[Browser ที่เปิดค้างไว้ + login แล้ว]
        │ CDP (connect_over_cdp)
        ▼
[Component 1: Data Collector] ──ดัก network/WebSocket──► ราคา/แท่งเทียนสด
        │ เขียนลง
        ▼
[Component 2: PostgreSQL] ──เก็บประวัติราคาทั้งหมด (time-series)
        │ อ่านย้อนหลัง
        ▼
[Component 3: Analysis Engine] ──คำนวณ indicator/pattern เอง (RSI, trendline, S/R)
        │
        ├──► [Component 4: Backtesting] ──ทดสอบกับข้อมูลย้อนหลัง ก่อนเชื่อ signal
        │
        ▼
[Component 5: Real-time Display] ──แสดงผล + สถานะ signal ปัจจุบัน (ไม่ auto-order)
```

**หลักการสำคัญ**: data flow เป็นเส้นตรงทางเดียว แต่ละ component ทดสอบแยกกันได้อิสระ (Data Collector ทดสอบเสร็จก่อน ไม่ต้องรอ Analysis Engine เสร็จ) — implement ทีละ component ตามลำดับ ไม่ทำพร้อมกันหลายอันตามหลักการเดิมของโปรเจกต์นี้

---

## 4. Component 1: Browser Data Collector

ต่อยอดจากสถาปัตยกรรมที่ตกลงกันไว้แล้วใน `CHART-VISION-BOT-TODO.md` แค่เปลี่ยนจาก chromedp (Go) เป็น Playwright (Python):

```python
from playwright.sync_api import sync_playwright

with sync_playwright() as p:
    browser = p.chromium.connect_over_cdp("http://localhost:9222")
    # ต่อเข้า browser ที่เปิดค้างไว้เอง (login แล้ว, เปิดกราฟที่ต้องการอยู่แล้ว)
    # ไม่ spawn browser ใหม่ เหมือนที่ตกลงกันไว้ก่อนหน้า
```

**ความแม่นยำ**: ตามที่ขอ ("ไม่จำกัดภาษาขอให้แม่นยำ") — ดักข้อมูลจาก network response/WebSocket frame ที่หน้าเว็บใช้จริง (ตัวเลขดิบ 100% ไม่ตีความจาก pixel) แม่นยำเท่ากับที่หน้าเว็บนั้นแสดงผลเป๊ะ:

- **REST response**: `page.on("response", handler)` ดักและ parse response ที่ match URL pattern ของ endpoint ราคา
- **WebSocket (สำคัญกว่าสำหรับ real-time)**: `page.on("websocket", ws_handler)` แล้ว subscribe `ws.on("framereceived", ...)` — เว็บกราฟสมัยใหม่เกือบทั้งหมด (รวม TradingView, exchange ต่างๆ) push ราคาแบบ real-time ผ่าน WebSocket ไม่ใช่ REST polling — **นี่คือกลไกหลักที่ทำให้ทำ real-time ได้จริงตาม requirement ข้อ 4**

---

## 5. Component 2: Database Storage

### Schema เริ่มต้น (PostgreSQL)

```sql
CREATE TABLE candles (
    symbol      TEXT NOT NULL,
    interval    TEXT NOT NULL,
    open_time   TIMESTAMPTZ NOT NULL,
    open        NUMERIC NOT NULL,
    high        NUMERIC NOT NULL,
    low         NUMERIC NOT NULL,
    close       NUMERIC NOT NULL,
    volume      NUMERIC,
    source      TEXT NOT NULL,        -- เว็บที่ดึงมา (เผื่อมีหลายแหล่งทีหลัง)
    ingested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (symbol, interval, open_time, source)
);

CREATE INDEX idx_candles_lookup ON candles (symbol, interval, open_time DESC);
```

- **Primary key กัน insert ซ้ำ** (`ON CONFLICT DO NOTHING`/`DO UPDATE`) — data collector อาจได้ event เดิมซ้ำจาก WebSocket reconnect ได้ ต้อง idempotent เสมอ
- **ไม่ทำ TimescaleDB ตั้งแต่แรก** (extension ของ Postgres ที่ optimize สำหรับ time-series โดยเฉพาะ) — เริ่มจาก PostgreSQL ธรรมดาไปก่อนตาม YAGNI ค่อยพิจารณาถ้าข้อมูลเยอะจนช้าจริง (ตารางเดียวไม่กี่ symbol คงไม่ถึงจุดนั้นเร็วๆ นี้)

---

## 6. Component 3: Analysis Engine (เขียนเอง)

ตรงกับที่คุยไว้ใน `CHART-VISION-BOT-TODO.md` หัวข้อ 4 ทั้งหมด — ย้ายมาทำบน pandas DataFrame ที่ query จาก PostgreSQL แทน (ได้ประวัติยาวจริง ไม่ใช่แค่ snapshot สั้นๆ แบบตอนยังไม่มี database):

- **Indicator เชิงตัวเลข**: RSI (สูตรเดียวกับที่เขียนไว้แล้วใน `internal/bots/pricewatch/pricewatch.go` — เอาสูตรมาปรับเป็น Python ได้เลย ไม่ต้องคิดใหม่), MA/EMA crossover
- **Pattern เชิงรูปทรง** (เขียนเป็น algorithm เอง ไม่มี ML):
  - Local peak/trough detection (`scipy.signal.argrelextrema` ช่วยหาได้ หรือเขียนเองด้วย loop ธรรมดา)
  - Trendline fitting ผ่านจุด peak/trough ต่อเนื่อง (linear regression พื้นฐานด้วย `numpy.polyfit`)
  - Support/Resistance level (นับความถี่ที่ราคาสะท้อนกลับใกล้ระดับเดียวกัน)
- **Output**: ทุก indicator/pattern คืนค่าเป็นตัวเลข/label ที่ชัดเจน (ไม่ใช่ "ซื้อ/ขาย" ตรงๆ) — **การแปลง indicator หลายตัวรวมกันเป็น "สัญญาณเดียว" คือส่วนที่ user ควรออกแบบเองอย่างตั้งใจ** (เช่น "RSI < 30 และราคาแตะ support level" ถึงจะนับเป็นสัญญาณที่น่าสนใจ) — ไม่ใช่สิ่งที่ควรมีสูตรสำเร็จรูปให้ เพราะนี่คือส่วนที่ต้อง backtest พิสูจน์เอง (ดูหัวข้อ 7)

---

## 7. Component 4: Backtesting (เพิ่มเข้ามาเอง — จำเป็นก่อนเชื่อผลวิเคราะห์)

**ไม่ได้อยู่ใน requirement เดิมที่ขอมา 4 ข้อ แต่เพิ่มเข้ามาเพราะจำเป็นจริง** — ไม่มีทางรู้ว่า Component 3 "แม่นยำ" แค่ไหนถ้าไม่ผ่านขั้นตอนนี้ก่อน:

1. เขียน backtest loop ง่ายๆ เอง: ไล่ข้อมูลราคาย้อนหลังทีละแท่ง (simulate ว่ากำลัง "อยู่ ณ เวลานั้น" ไม่เห็นอนาคต) เช็คว่าสัญญาณจาก Component 3 เกิดขึ้นไหม ถ้าเกิดให้จำลองว่า "เข้า order ที่ราคานี้" แล้วดูว่าถ้าถือไปอีก N แท่ง ผลลัพธ์เป็นยังไง
2. **ต้องหักค่าธรรมเนียม/spread ด้วยเสมอ** — สัญญาณที่ดูดีตอนไม่หักค่าธรรมเนียม อาจกลายเป็นขาดทุนจริงทันทีที่หักออก (คนที่เทรดจริงพลาดจุดนี้บ่อยที่สุด)
3. **แบ่งข้อมูลเป็น 2 ช่วง**: ช่วงแรกใช้ปรับจูนกฎ (in-sample), ช่วงหลังใช้ทดสอบเฉยๆ ไม่แก้กฎอีก (out-of-sample) — ถ้า performance ช่วงหลังต่างจากช่วงแรกมาก แปลว่ากฎ "overfit" กับข้อมูลเก่า ไม่ใช่ pattern จริงที่ทำซ้ำได้
4. รายงานผลเป็นตัวเลขที่เทียบได้ (win rate, average return per trade, max drawdown) ไม่ใช่แค่ดูกราฟ equity curve สวยๆ เฉยๆ
5. **ทดสอบแยกเฉพาะช่วงตลาดผันผวนหนัก/วิกฤต** ไม่ใช่ดูแค่ค่าเฉลี่ยรวมทั้งช่วง — signal ที่ดูดีตอนตลาดขาขึ้นเรียบๆ อาจพังทันทีตอนตลาดผันผวนหนัก (regime เปลี่ยน) ต้องรู้ล่วงหน้าว่า signal พังตอนไหนบ้าง ไม่ใช่มารู้ตอนใช้เงินจริง
6. **จำลอง latency ที่เกิดขึ้นจริงด้วย** — เพราะเป็นระบบ decision-support (คนกดยืนยันเอง ไม่ auto-trade) มีความหน่วงจริงระหว่าง "signal เกิดขึ้น" กับ "คนกดปุ่มจริง" (อ่านคำแนะนำ ตัดสินใจ เปิดแอป กดยืนยัน) — backtest ที่สมมติว่าเข้า order ได้ทันทีตอน signal เกิดขึ้นจะดูดีเกินจริงเสมอ ควรจำลอง delay อย่างน้อย 1-2 นาทีก่อนคำนวณราคาที่ "เข้าได้จริง"

---

## 8. Component 5: Real-time Display + Signal Output

**เริ่มจากง่ายที่สุดก่อนเสมอ** (ตาม YAGNI ของทั้งโปรเจกต์) — ไม่สร้าง web dashboard สวยๆ ตั้งแต่ต้น:

1. **ขั้นแรก**: print สถานะ indicator/signal ปัจจุบันออก terminal (หรือใช้ `rich` library ให้อ่านง่ายขึ้นหน่อย) ทุกครั้งที่มีแท่งเทียนใหม่เข้ามา
2. **ขั้นถัดไป** (หลัง backtest ยืนยันว่า signal มีความหมายแล้วเท่านั้น): ทำ dashboard จริงจัง — ตอนนั้นค่อยพิจารณาว่าจะ reuse pattern SSE ที่ `bop-dashboard` มีอยู่แล้ว (`use-job-log-stream.ts`) หรือทำแยกเป็นหน้าเว็บ Python เอง (Flask/FastAPI + WebSocket)

**เรื่อง "ควรเข้า order"**: แสดงเป็น**สถานะของสัญญาณ** ("RSI=25 (oversold), ราคาแตะ support ที่คำนวณไว้ — เข้าเงื่อนไขที่ backtest แล้วว่า [win rate X%, average return Y%]") ไม่ใช่คำสั่ง "ซื้อเลย" ตรงๆ — ให้ผู้ใช้เห็นทั้งสัญญาณและสถิติที่ backtest มาประกอบการตัดสินใจเองเสมอ (ตรงกับที่ตกลงกันไว้แต่แรกว่าไม่ auto-trade)

---

## 9. Component 6: Custom AI Model (เป้าหมายระยะยาว)

**เป้าหมายปลายทางที่ต้องการ**: ให้ระบบตัดสินใจได้ทันทีว่าควรเข้า order แบบไหน จากข้อมูลกราฟที่ได้ ด้วย AI model ที่เป็นของตัวเอง — **section นี้ต้องมาหลัง Component 4 (Backtesting) เท่านั้น ห้ามข้ามมาทำก่อน** เพราะการ evaluate ว่า model "แม่นยำ" จริงไหมต้องใช้ infra เดียวกับที่ backtest กฎ rule-based ใช้อยู่แล้วทุกประการ — ไม่มี infra evaluate ก็ไม่มีทางแยกแยะได้เลยว่า model เก่งจริงหรือแค่ overfit กับข้อมูลเก่า

**ย้ำเรื่องขอบเขตอีกครั้ง**: ต่อให้ model ตัดสินใจได้ "ทันที" ก็ยังหมายถึง**แสดงคำแนะนำให้คนกดยืนยันเอง ไม่ auto-execute order** (เหตุผลเชิงจิตวิทยาที่ควรออกแบบไว้ตั้งแต่ต้น: พอ model บอกว่า "เข้าเลย" คนมักกดตามแบบไม่ทันคิดอยู่ดี แม้จะยังมีปุ่มให้กดเองก็ตาม — UI ต้องโชว์ "เพราะอะไร" + "backtested accuracy ที่ผ่านมา" คู่กับคำแนะนำเสมอ ไม่ใช่โชว์แค่ "BUY"/"SELL" เฉยๆ ที่ตัดวิจารณญาณของคนออกไปโดยไม่ตั้งใจ)

### ทางเลือกเรียงจากง่าย → ยาก

**ทางเลือก 1 (แนะนำเริ่มจากตรงนี้) — Supervised Learning แบบคลาสสิก**: ใช้ feature ที่มีอยู่แล้วจาก Component 3 (ค่า RSI, ระยะห่างจาก support/resistance, ความชัน trendline ฯลฯ — เป็นตัวเลขล้วนๆ ที่มีอยู่ใน PostgreSQL แล้ว) เป็น input แล้ว train model แบบ classification ("ราคาน่าจะขึ้น/ลง/ไม่เปลี่ยนใน N แท่งข้างหน้า") ด้วย `scikit-learn`/`XGBoost`/`LightGBM` — ไม่ต้องมี deep learning เลยด้วยซ้ำ **และไม่ต้องแตะเรื่อง vision/ภาพเลย** เพราะข้อมูลเป็น numeric feature ที่มี pipeline พร้อมอยู่แล้วตั้งแต่ Component 2/3 — เป็นทางที่ทำได้จริงที่สุดตอนถึงจุดนี้

**ทางเลือก 2 (ยากขึ้นมาก) — Reinforcement Learning**: มอง trading เป็น sequential decision problem ให้ reward = กำไร/ขาดทุน — ฟังดูตรงกับ "ตัดสินใจได้ทันที" ที่สุด แต่**ยากในทางปฏิบัติมาก** แม้แต่ quant fund ที่ทำเรื่องนี้จริงจังก็ใช้ทีมใหญ่+ทรัพยากรมหาศาล เพราะ reward signal ในตลาดการเงิน noisy/เบาบางกว่าที่ RL ทำได้ดีในโดเมนอื่น (เกม/หุ่นยนต์) มาก ขึ้นชื่อเรื่อง unstable/overfit ง่าย — ไม่แนะนำเป็นจุดเริ่มต้น

**ทางเลือก 3 — ใช้ LLM ที่มีอยู่แล้วช่วยอ่าน feature summary** (ไม่ train เอง): ส่งตัวเลข feature ปัจจุบัน (ไม่ใช่ภาพ) ให้ LLM สรุป/ให้เหตุผลประกอบ — เร็วสุดในการเริ่มต้น แต่มีค่าใช้จ่ายต่อ call (เหมือนที่เคยพิจารณาแล้วตัดออกตอนคุยเรื่อง vision) และเป็น "ให้ model ที่คนอื่น train มาช่วยคิด" ไม่ใช่ "AI model เป็นของตัวเอง" ตามที่ต้องการเป๊ะๆ

### Reality Check เพิ่มเติมเฉพาะเรื่อง AI Model

Model ที่ train จากข้อมูลราคาย้อนหลัง **เสี่ยง overfit ง่ายกว่ากฎ rule-based ที่เขียนเองเสียอีก** เพราะ model มีความยืดหยุ่นสูงกว่ามาก (จดจำรูปแบบเฉพาะของข้อมูล training ได้ง่ายโดยไม่ได้เจอ pattern จริงที่ทำซ้ำได้ในอนาคต) — ต้องผ่าน validation ที่เข้มงวดกว่า backtest ของกฎ rule-based ด้วยซ้ำ: ทำ **walk-forward validation** (train ด้วยข้อมูลช่วงหนึ่ง ทดสอบกับช่วงถัดไปที่ไม่เคยเห็นตอน train เลื่อนหน้าต่างเวลาไปเรื่อยๆ ไม่ใช่ train/test แบ่งครั้งเดียวจบ) ก่อนถึงจะเริ่มเชื่อผลลัพธ์ได้บ้าง

---

## 10. Component 7: Risk Management & Position Sizing

**เหตุผลที่แยกเป็น component ต่างหาก**: ในทางปฏิบัติ **risk management สำคัญกว่าตัว signal เองเสียอีก** — model/กฎ rule-based แม่นยำแค่ไหนก็ไร้ประโยชน์ (หรืออันตราย) ถ้าไม่มีเรื่องนี้กำกับไว้ และใช้ได้กับทั้ง signal จาก Component 3 (rule-based) และ Component 6 (AI model) เหมือนกัน ไม่ใช่แค่ตัวใดตัวหนึ่ง

- **Position sizing** — กำหนดว่าเข้า order ครั้งละกี่ % ของทุนทั้งหมด ไม่ใช่ all-in ทุกครั้งที่ signal ขึ้น:
  - **Fixed fractional** (แนะนำเริ่มจากตรงนี้): เข้าไม้ละ X% ของทุนคงที่เสมอ — ง่ายที่สุด เข้าใจง่าย ปรับ X ได้ตามความเสี่ยงที่รับได้
  - **Fractional Kelly**: คำนวณสัดส่วนจาก win rate/avg return ที่ backtest ได้ แต่ใช้แค่เศษส่วนของค่า Kelly เต็ม (เช่น 25-50%) เพราะ Kelly เต็มรูปแบบ sensitive มากต่อความผิดพลาดของค่าประมาณ win rate — ใช้ค่าเต็มเสี่ยงเกินไปในทางปฏิบัติ
- **Stop-loss ที่กำหนดไว้ล่วงหน้าเสมอ** — ทุก signal ต้องมี "ยอมเสียแค่ไหนถ้าผิด" มาพร้อมกันตั้งแต่ตอนแสดงผล ไม่ใช่ให้คนตัดสินใจตอนขาดทุนแล้ว (อารมณ์เข้ามาแทรกตอนนั้นทำให้ตัดสินใจแย่ลงเป็นปกติ) — คำนวณจาก ATR (Average True Range) หรือระดับ support/resistance ที่มีอยู่แล้วจาก Component 3 ได้เลย ไม่ต้องคิด indicator ใหม่
- **Max exposure รวม / correlation risk** — ถ้าเปิดหลาย position พร้อมกัน (หลาย symbol) ต้องคุมไม่ให้ความเสี่ยงรวมสูงเกินไปถ้าตลาดเคลื่อนไหวพร้อมกันหมด (เช่น BTC กับ ETH มักเคลื่อนไหวไปทางเดียวกัน นับเป็นความเสี่ยงเดียวกันเกือบทั้งหมด ไม่ใช่ 2 ความเสี่ยงแยกกัน)
- **ต่อเข้า Component 5**: ทุก signal ที่แสดงต้องมี suggested position size + stop-loss level แนบมาด้วยเสมอ ไม่ใช่แค่ทิศทาง (up/down) เฉยๆ

---

## 11. Component 8: Paper Trading Gate

**เหตุผล**: backtest ที่ดีที่สุดก็ยังเป็นแค่การจำลองบนข้อมูลย้อนหลัง ไม่รับประกันอนาคต — ก่อนใช้เงินจริงแม้แต่บาทเดียว ต้องผ่านด่านนี้ก่อนเสมอ ใช้ได้กับทั้ง signal จาก Component 3/4 (rule-based) และ Component 6 (AI model)

- **Shadow mode**: หลัง backtest (Component 4 หรือ 6) ผ่านเกณฑ์แล้ว รันสร้างสัญญาณจริงแบบ real-time ต่อ **แต่ไม่ commit เงินจริงเลย** — เก็บผลลัพธ์เหมือนเทรดจริงทุกอย่าง (ขยายตาราง `predictions` จากหัวข้อ 9 ให้ log สัญญาณจาก rule-based ด้วย ไม่ใช่แค่ AI model)
- **ระยะเวลาที่แนะนำ**: ตาม Open Question ที่มีอยู่แล้ว (หลักเดือนขึ้นไป ครอบคลุมหลาย market condition — ตัดสินใจจริงตอนถึง phase นี้)
- **เกณฑ์ตัดสินใจ**: เทียบ live paper win-rate กับ backtest win-rate อย่างเข้มงวด — ถ้าต่างกันมากคือสัญญาณเตือนว่ามี bias ที่มองไม่เห็นตอน backtest (เช่น look-ahead bias ที่หลุดรอดมา, latency ที่จำลองไม่สมจริงพอ)
- **Kill switch**: ถ้า live performance (ไม่ว่าเป็น paper trade หรือหลังใช้เงินจริงแล้ว) ตกจาก backtest มากเกิน threshold ที่ตั้งไว้ ให้หยุดแสดงคำแนะนำอัตโนมัติ/แจ้งเตือนชัดเจนว่าไม่ควรเชื่อ signal ตอนนี้ — ไม่ปล่อยให้ระบบแนะนำต่อไปเรื่อยๆ ทั้งที่ performance แย่ลงชัดเจนแล้ว

---

## 12. Component 9: Operational Monitoring & Backup

**เหตุผล**: ระบบนี้ต้องรันต่อเนื่องเป็นเดือนๆ (สะสมข้อมูล backtest + paper trading period) — ถ้าไม่มีเรื่องนี้ **ข้อมูลอาจขาดหายไปโดยไม่มีใครรู้จนกว่าจะกลับมาเช็คทีหลัง** ทำให้ backtest/paper trading period ที่ผ่านมาเสียหายไปแล้วโดยไม่รู้ตัว

**Health check ของ Data Collector (Component 1)**:
- Alert ถ้าไม่มีแท่งเทียนใหม่เข้ามาตามที่คาดในช่วงเวลาที่กำหนด (เช่น ไม่มีข้อมูลเข้ามาเกิน 2 เท่าของ interval ที่ตั้งไว้)
- ตรวจจับ browser session หลุด/WebSocket ขาดการเชื่อมต่อ แล้วพยายาม reconnect อัตโนมัติ (หรืออย่างน้อย alert ให้คนมาแก้เอง — เว็บเป้าหมายเปลี่ยน internal API แบบเงียบๆ ได้ตลอดตามที่เตือนไว้ในหัวข้อ 4 เรื่อง Maintenance Cost)

**Data quality check**:
- ตรวจ gap ในข้อมูล (ช่วงเวลาที่ควรมีแท่งเทียนแต่ไม่มี)
- ตรวจค่าที่ผิดปกติ (เช่น ราคาต่างจากแท่งก่อนหน้ามากเกินไปแบบไม่สมเหตุสมผล — มักเป็น bug ตอน parse ไม่ใช่ราคาจริง)

**Backup ของ PostgreSQL**:
- ข้อมูลราคาที่เก็บสะสมมามีค่ามาก (บางแหล่งดึงย้อนหลังไม่ได้ถ้า provider ไม่เก็บ history นาน) — ต้องมีแผน backup พื้นฐาน (`pg_dump` ตามตารางเวลา อย่างน้อยที่สุด)
- backup ควรเก็บแยกจากเครื่องที่รันจริง (ไม่ใช่ backup อยู่ disk เดียวกับข้อมูลจริง ไม่งั้น disk พังเสียทั้งคู่)

---

## 13. ความสัมพันธ์กับ BOP เดิม

**ไม่บังคับต้อง integrate ตอนนี้** — Trade Analyzer พึ่งพาตัวเองได้ 100% (Python + PostgreSQL ของตัวเอง แยกจาก BOP) แต่ถ้าอยากให้ตั้ง schedule/monเนิเตอร์ผ่าน BOP dashboard เดิมทีหลัง มี pattern ที่ออกแบบไว้แล้วให้ใช้ได้เลย:

- [REMOTE-JOB-WORKER-TODO.md](../bot-orchestration-platform/REMOTE-JOB-WORKER-TODO.md) — contract สำหรับ process/ภาษาอื่นที่อยาก register Temporal Activity ของตัวเอง เข้า task queue เดียวกับ BOP โดยไม่ต้อง import โค้ด Go เลยแม้แต่บรรทัดเดียว (Python มี `temporalio` SDK รองรับอยู่แล้ว) — ตรงกับสถานการณ์นี้เป๊ะ (ภาษาต่างกัน, repo แยกกัน)

ไม่ต้องตัดสินใจตอนนี้ — ทำ Trade Analyzer ให้ใช้งานได้ด้วยตัวเองก่อน ค่อยกลับมาพิจารณา integration ทีหลัง

---

## Open Questions

1. **เว็บเป้าหมาย** — เหมือนเดิมจากเอกสารก่อนหน้า ต้องเช็ค ToS ก่อนเสมอ
2. **จะเก็บกี่ symbol/timeframe พร้อมกัน** — กระทบขนาด database และจำนวน browser tab ที่ต้องเปิดค้างพร้อมกัน
3. **Backtest framework สำเร็จรูป vs เขียนเอง** — เริ่มเขียนเองตามที่แนะนำในหัวข้อ 2 แต่ถ้าซับซ้อนขึ้นเรื่อยๆ ค่อยประเมินใหม่
4. **ระยะเวลา backtest ที่ "พอเชื่อถือได้"** — ไม่มีคำตอบตายตัว แต่ควรคุยกันอีกทีตอนถึง Component 4 จริงๆ (ทั่วไปมักพูดถึงหลักปีขึ้นไป ครอบคลุมหลาย market condition)
5. **เกณฑ์ที่ตัดสินว่า "backtest ผ่านพอจะเริ่ม Component 6 (AI model) ได้"** — ต้องตัดสินใจตอนถึง Phase 4 จริงๆ ไม่ใช่ข้ามไป train model เลยโดยยังไม่มีเกณฑ์ชัดเจนว่า rule-based baseline ดีพอหรือยัง
6. **เรื่องกฎหมาย/regulation** — ถ้าวันหนึ่งระบบขยับไปใกล้การเชื่อมต่อ broker/exchange จริง (แม้จะยัง semi-manual) ควรเช็คว่า algorithmic trading tool มีข้อกำหนดจาก ก.ล.ต. ไทยที่ต้องรู้ไหม — ยังไม่ใช่ประเด็นตอนนี้เพราะระบบยัง decision-support ล้วนๆ ไม่ได้ต่อ API ของ broker เลย แต่ควรเช็คก่อนขยับไปทางนั้นจริงๆ
7. **การเก็บ credential** — ถ้าวันหนึ่งต้องต่อ API ของ broker จริง (เช่น ดึงยอดเงินมาคำนวณ position size อัตโนมัติใน Component 7) ต้องมีที่เก็บ secret ที่ปลอดภัย ไม่ hardcode — ยังไม่ใช่ประเด็นตอนนี้ด้วยเหตุผลเดียวกับข้อ 6

## TODO Checklist

### Phase 0: Scaffold
- [ ] ตั้ง Python project (`uv`/`poetry`/`venv` ตามถนัด), ติดตั้ง `playwright`, `pandas`, `numpy`, `psycopg2`
- [ ] `playwright install chromium` (ติดตั้ง browser binary ที่ Playwright ต้องการ)

### Phase 1: Data Collector (Component 1)
- [ ] เปิด Chrome ด้วย `--remote-debugging-port=9222`
- [ ] ต่อเข้าผ่าน `connect_over_cdp` สำเร็จ (POC เปล่าๆ)
- [ ] เปิด DevTools ของเว็บเป้าหมายเอง หา endpoint/WebSocket ที่ใช้ส่งราคา
- [ ] ดักได้จริง print ราคาที่ได้ออก terminal ยืนยันว่าตรงกับที่เห็นบนหน้าเว็บ

### Phase 2: Database (Component 2)
- [ ] สร้าง PostgreSQL (`docker compose` ของ trade-analyzer เอง แยกจาก BOP)
- [ ] สร้าง schema ตามหัวข้อ 5
- [ ] ต่อ Data Collector ให้เขียนเข้า DB จริง (idempotent insert)

### Phase 3: Analysis Engine (Component 3)
- [ ] คำนวณ RSI จากข้อมูลใน DB (เทียบผลกับ `price-watch` เดิมที่คำนวณจาก API ตรงๆ ว่าตรงกันไหม — เป็นวิธี verify ที่ดี)
- [ ] Peak/trough detection
- [ ] Trendline/Support-Resistance

### Phase 4: Backtesting (Component 4) — **ห้ามข้าม**
- [ ] เขียน backtest loop พื้นฐาน
- [ ] หักค่าธรรมเนียม/spread
- [ ] แบ่ง in-sample/out-of-sample
- [ ] ทดสอบแยกเฉพาะช่วงตลาดผันผวนหนัก/วิกฤต (ไม่ใช่แค่ค่าเฉลี่ยรวม) — รู้ล่วงหน้าว่า signal พังตอนไหนบ้าง
- [ ] จำลอง latency ระหว่าง signal เกิดกับคนกดยืนยันจริง (อย่างน้อย 1-2 นาที) ก่อนคำนวณราคาที่ "เข้าได้จริง"
- [ ] สรุปผลเป็นตัวเลข (win rate, avg return, max drawdown) — **ตัดสินใจตรงนี้ว่า signal ที่เขียนมามีความหมายจริงไหม ก่อนไป Phase 5**

### Phase 5: Real-time Display (Component 5)
- [ ] Terminal output ก่อน
- [ ] (ถ้า backtest ผ่านเกณฑ์ที่ตั้งไว้) ค่อยพิจารณาทำ dashboard เต็มรูปแบบ

### Phase 6: Custom AI Model (Component 6) — **ห้ามเริ่มก่อน Phase 4 ผ่านเกณฑ์**

โมเดลที่แนะนำ: Gradient Boosting (`XGBoost`/`LightGBM`) — เหตุผลเทียบกับ RL/deep learning ดูหัวข้อ 9

### Schema เพิ่มเติมสำหรับ Component 6 (แยกจากตาราง `candles` เดิมในหัวข้อ 5)

หลัง train เสร็จมีข้อมูล 2 ชนิดที่ต้องเก็บแยกจากข้อมูลราคาดิบ — **model artifact/metadata** (บันทึกครั้งเดียวตอน train) กับ **prediction log** (บันทึกทุกครั้งที่ model ทำนายจริงแบบ real-time — คนละเรื่องกับตอน backtest):

```sql
-- เก็บ metadata ของ model แต่ละ version ที่ train ไว้ — ตัว model file จริง (.json/.ubj/.pkl)
-- เก็บแยกเป็นไฟล์บน filesystem/object storage ตาราง นี้แค่ชี้ path + เก็บบริบทของการ train
-- ครั้งนั้นไว้ ไม่งั้นย้อนกลับไปดูทีหลังไม่ได้เลยว่า prediction เก่าๆ มาจาก model รุ่นไหน
-- train ด้วย hyperparameter/feature set แบบไหน
CREATE TABLE model_registry (
    id                     SERIAL PRIMARY KEY,
    model_version          TEXT NOT NULL UNIQUE,   -- ตั้งชื่อให้อ่านแล้วรู้บริบททันที เช่น "xgb_v3_2026-08-15"
    trained_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    algorithm              TEXT NOT NULL,           -- "xgboost" / "lightgbm"
    hyperparameters        JSONB NOT NULL,          -- {"n_estimators": 200, "max_depth": 4, ...} — ใช้ JSONB เพราะ hyperparameter แต่ละ algorithm ไม่เหมือนกัน ไม่อยากแก้ schema ทุกครั้งที่ลอง algorithm ใหม่
    feature_set_version    TEXT NOT NULL,           -- ชี้ว่า train ด้วย feature ชุดไหน (feature เปลี่ยนได้ตามข้อ 6.5)
    training_data_range    TSTZRANGE NOT NULL,      -- ช่วงเวลาของข้อมูลที่ใช้ train เช่น [2020-01-01, 2024-01-01)
    backtest_win_rate      NUMERIC,                 -- ผล backtest (Component 4) ตอน train เสร็จ — ใช้เทียบกับ live performance ทีหลัง
    backtest_avg_return    NUMERIC,
    backtest_max_drawdown  NUMERIC,
    artifact_path          TEXT NOT NULL,           -- path ของไฟล์ model ที่ serialize ไว้
    is_active              BOOLEAN NOT NULL DEFAULT false -- true = model version ที่ Component 5 กำลังใช้ทำนายจริงอยู่ตอนนี้
);

-- บังคับให้มี active model ได้แค่ 1 version เสมอ กันความกำกวมว่า "ตอนนี้ระบบใช้ model ไหนทำนายอยู่จริง"
CREATE UNIQUE INDEX idx_model_registry_one_active
    ON model_registry (is_active) WHERE is_active = true;

-- เก็บทุกครั้งที่ model ทำนายจริงแบบ real-time (ไม่ใช่ตอน backtest) — actual_outcome เป็น NULL
-- ตอน insert เสมอ เพราะยังไม่รู้ผลจริงจนกว่าจะผ่านไป N แท่งตามที่นิยาม label ไว้ในข้อ 6.1
-- แล้วค่อยกลับมา UPDATE ทีหลังเมื่อรู้ผลแล้ว — ออกแบบเป็น 2 จังหวะแบบนี้เพื่อเทียบ live
-- performance กับ backtest_win_rate ใน model_registry ได้ (สัญญาณว่าถึงเวลา retrain หรือยัง)
CREATE TABLE predictions (
    id              BIGSERIAL PRIMARY KEY,
    model_version   TEXT NOT NULL REFERENCES model_registry(model_version),
    symbol          TEXT NOT NULL,
    predicted_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    features        JSONB NOT NULL,   -- snapshot ค่า feature ณ ตอนทำนาย เก็บไว้ debug/reproduce ย้อนหลังได้
    prediction      TEXT NOT NULL,    -- "up" / "down" / "flat"
    confidence      NUMERIC NOT NULL, -- ความมั่นใจของ model (0-1)
    actual_outcome  TEXT,             -- เติมทีหลังตอนรู้ผลจริงแล้วเท่านั้น (NULL = ยังรอผล)
    evaluated_at    TIMESTAMPTZ       -- เวลาที่กลับมาเติม actual_outcome
);

CREATE INDEX idx_predictions_lookup ON predictions (model_version, predicted_at DESC);
CREATE INDEX idx_predictions_pending_eval ON predictions (predicted_at) WHERE actual_outcome IS NULL;
```

**6.1 เตรียม Label (ขั้นตอนที่พลาดง่ายที่สุด)**
- [ ] นิยาม label ให้ชัดเจน: ดูราคา ณ เวลา T เทียบกับราคา ณ T+N แท่งข้างหน้า แล้ว bucket เป็น "ขึ้น/ลง/ไม่เปลี่ยน" (หรือทำเป็น regression target = % return ก็ได้)
- [ ] **ตรวจสอบ look-ahead bias ให้ละเอียด** — feature ทุกตัว ณ เวลา T ต้องคำนวณจากข้อมูลที่มีอยู่จริง ณ เวลานั้นเท่านั้น ห้ามมีข้อมูลจากอนาคตหลุดเข้าไปโดยไม่ตั้งใจ (จุดที่ทำให้ backtest ดูดีเกินจริงบ่อยที่สุด) — เขียน test เฉพาะเช็คเรื่องนี้ (เช่น สุ่มตัดข้อมูลที่ T แล้ว mock ว่า "ยังไม่รู้อนาคต" ดูว่า feature เปลี่ยนไหม)
- [ ] เช็ค class imbalance (เช่น "ขึ้น" เกิดน้อยกว่า "ลง"/"ไม่เปลี่ยน" มากไหม) ถ้า imbalance ต้องมีแผนรับมือ (resampling, class weight)

**6.2 แบ่งข้อมูลตามเวลา (ห้าม Random Split)**
- [ ] ใช้ `sklearn.model_selection.TimeSeriesSplit` แทน `train_test_split` ธรรมดาเสมอ — แบ่งตามลำดับเวลา (เช่น ปีเก่า = train, ปีล่าสุด = test) ไม่สุ่ม ป้องกัน data leakage จากอนาคตเข้าไปช่วยตอน train
- [ ] Feature engineering จาก indicator/pattern ที่มีอยู่แล้ว (Component 3) — ไม่ต้องคิด feature ใหม่ทั้งหมด ต่อยอดจากที่มี (RSI, ระยะห่าง support/resistance, ความชัน trendline)

**6.3 เทรนโมเดล**
- [ ] Train model แบบ classification ง่ายๆ ก่อน (`XGBoost`/`LightGBM`) ด้วย hyperparameter เริ่มต้นตามค่า default ของ library ก่อน ยังไม่ต้อง tune
- [ ] ตั้ง `n_estimators`/`max_depth`/`learning_rate` แบบระมัดระวัง — ยิ่งต้นเยอะ/ลึกยิ่งเสี่ยง overfit (จำ noise แทน pattern จริง)
- [ ] เปิด early stopping เทียบ train loss กับ validation loss คู่กันเสมอ — ถ้า train loss ลดต่อแต่ validation loss เริ่มขึ้น ให้หยุด train ตรงนั้น
- [ ] ใช้ regularization (`reg_alpha`/`reg_lambda`) ถ้ายัง overfit อยู่หลัง early stopping

**6.4 ประเมินผล — อย่าเชื่อแค่ Accuracy**
- [ ] Walk-forward validation (train ช่วงหนึ่ง ทดสอบช่วงถัดไปที่ไม่เคยเห็น เลื่อนหน้าต่างไปเรื่อยๆ ไม่ใช่ train/test split ครั้งเดียวจบ)
- [ ] **เอา prediction ไปวิ่งผ่าน Component 4 (Backtesting) เดียวกับที่ใช้กับ rule-based signal** — accuracy/F1-score สูงไม่ได้แปลว่ากำไรจริงเสมอไป ต้องดูผลหลังหักค่าธรรมเนียมจริง
- [ ] เทียบผลกับ rule-based baseline จาก Phase 4 — model ต้องดีกว่า baseline จริงถึงจะถือว่าคุ้มความซับซ้อนที่เพิ่มขึ้น (ถ้าไม่ดีกว่า ใช้ rule-based ต่อไปดีกว่า ไม่มีเหตุผลเพิ่มความซับซ้อน)

**6.5 วนซ้ำ (ถ้าผลยังไม่ดีพอ)**
- [ ] ลองปรับ/เพิ่ม feature ก่อนเสมอ (feature ที่ดีสำคัญกว่า model ซับซ้อน) — **ไม่ใช่**เพิ่มจำนวนต้น/ทำโมเดลซับซ้อนขึ้นเป็นทางแก้แรก (มักทำให้ overfit หนักขึ้นเฉยๆ)
- [ ] ลอง label definition แบบอื่น (เปลี่ยนจำนวนแท่ง N ข้างหน้า, เปลี่ยน threshold)
- [ ] ยอมรับเพดานที่เป็นไปได้: ถ้า feature ที่มีไม่มี "สัญญาณ" ทำนายอนาคตได้จริง ต่อให้ tune ยังไงก็มีเพดานที่ไปต่อไม่ได้ — ต้องกลับไปคิด feature ใหม่ ไม่ใช่ tune model ต่อไปเรื่อยๆ

**6.6 ต่อเข้าระบบ**
- [ ] สร้างตาราง `model_registry`/`predictions` ตาม schema ด้านบน + insert แถวใหม่เข้า `model_registry` ทุกครั้งที่ train เสร็จ (ตั้ง `is_active` เองด้วยมือก่อน ยังไม่ต้อง auto-promote)
- [ ] ต่อเข้า Component 5 (แสดงคำแนะนำ + เหตุผล + backtested accuracy คู่กันเสมอ ไม่โชว์แค่ BUY/SELL เปล่าๆ — ดูเหตุผลเชิงจิตวิทยาในหัวข้อ 9) — ทุกครั้งที่ทำนายจริง ต้อง insert แถวใหม่เข้า `predictions` ด้วยเสมอ
- [ ] เขียน job แยกต่างหาก (รันเป็น schedule ก็ได้) ไล่ query `predictions` ที่ยังไม่มี `actual_outcome` แล้วครบ N แท่งตามที่นิยามไว้ตอน 6.1 แล้ว → เติมผลจริงกลับเข้าไป
- [ ] สรุป live win-rate จาก `predictions` ที่ evaluate แล้วเทียบกับ `backtest_win_rate` ใน `model_registry` เป็นระยะ — performance live ตกจาก backtest มากคือสัญญาณว่าถึงเวลา retrain (model drift)

### Phase 7: Risk Management & Position Sizing (Component 7)
- [ ] เลือกวิธี position sizing เริ่มต้น (แนะนำ fixed fractional ก่อน ง่ายสุด)
- [ ] เขียน logic คำนวณ stop-loss suggestion (ATR หรือระดับ support/resistance จาก Component 3)
- [ ] เพิ่ม check max exposure รวมก่อนแสดง signal ใหม่ (เกิน limit ที่ตั้งไว้ → ไม่แสดง/แสดงเตือน)
- [ ] อัปเดต backtest loop ที่ Component 4 ให้ include position sizing/stop-loss ด้วย ไม่ใช่แค่ทิศทางถูก/ผิดเฉยๆ (equity curve ต้องคำนวณจาก sizing จริง)
- [ ] ต่อเข้า Component 5: ทุก signal แสดง suggested position size + stop-loss คู่กับทิศทางเสมอ

### Phase 8: Paper Trading Gate (Component 8) — **ห้ามข้ามก่อนใช้เงินจริง**
- [ ] กำหนดระยะเวลา paper trading ขั้นต่ำก่อนเริ่ม (ตัดสินใจจริงตอนถึง phase นี้)
- [ ] ขยายตาราง `predictions` (จากหัวข้อ 9) ให้ log สัญญาณจาก rule-based (Component 3/4) ด้วย ไม่ใช่แค่ AI model
- [ ] เขียน report เปรียบเทียบ live win-rate กับ backtest win-rate เป็นระยะ (weekly/monthly)
- [ ] กำหนด kill-switch threshold ที่ชัดเจน (ตัวเลขเท่าไหร่ ติดต่อกันกี่ครั้ง)
- [ ] **เกณฑ์ตัดสินใจก่อนใช้เงินจริง**: ต้องผ่านทั้ง backtest (Phase 4/6) และ paper trading period เต็มโดยไม่มี kill-switch trigger

### Phase 9: Operational Monitoring & Backup (Component 9) — ทำคู่ขนานได้ตั้งแต่ Phase 1 (ไม่ต้องรอ Phase 8)
- [ ] เขียน health check script เช็คว่าแท่งเทียนใหม่เข้ามาตามที่คาดไหม + alert แบบง่ายๆ ก่อน (เช่น log/แจ้งเตือนพื้นฐาน ไม่ต้องซับซ้อน)
- [ ] เขียน reconnect logic ให้ Data Collector (Component 1) เมื่อ browser session/WebSocket หลุด
- [ ] เขียน data quality check ตรวจ gap/ค่าผิดปกติในข้อมูลเป็นระยะ
- [ ] ตั้ง `pg_dump` ตามตารางเวลา (เช่น daily) + เก็บ backup แยกที่อื่น ไม่ใช่ disk เดียวกับ production data

## Reference

- [../bot-orchestration-platform/CHART-VISION-BOT-TODO.md](../bot-orchestration-platform/CHART-VISION-BOT-TODO.md) — เอกสารเดิมที่ถูกแทนที่ (เก็บไว้เป็น decision log)
- [../bot-orchestration-platform/internal/bots/pricewatch/pricewatch.go](../bot-orchestration-platform/internal/bots/pricewatch/pricewatch.go) — สูตร RSI ต้นแบบ (ย้ายมา Python ได้ตรงๆ)
- [../bot-orchestration-platform/REMOTE-JOB-WORKER-TODO.md](../bot-orchestration-platform/REMOTE-JOB-WORKER-TODO.md) — ทางเชื่อมกลับเข้า BOP ทีหลัง (ไม่บังคับตอนนี้)
- [Playwright for Python](https://playwright.dev/python/) — browser automation
