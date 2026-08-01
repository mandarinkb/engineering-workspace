# บทที่ 1: Workflow และ Activity

## 1.1 Workflow

### Workflow คือ Deterministic Code ที่ Describe ลำดับขั้นตอน

**Workflow** ใน Temporal คือโค้ดปกติ (เขียนเป็นภาษา Go เหมือนโค้ดทั่วไป — เชื่อมกับ [Volume 4 — Golang](../04-golang/README.md)) ที่อธิบายลำดับขั้นตอนของ business process หนึ่งชุด เช่น "รับ order → เก็บเงิน → หักสต็อก → จัดส่ง" — สิ่งที่ทำให้ workflow ต่างจากฟังก์ชันทั่วไปอย่างสิ้นเชิงคือ Temporal รับประกันว่า**โค้ดนี้จะรันจนจบเสมอ ไม่ว่าจะใช้เวลานานแค่ไหน และไม่ว่า process ที่รันมันจะล่มไปกี่ครั้งระหว่างทาง** — คุณสมบัตินี้เรียกว่า **durable execution**

ตัวอย่าง workflow ง่ายๆ:

```go
func OrderWorkflow(ctx workflow.Context, orderID string) error {
    // ขั้นตอนที่ 1: เก็บเงิน
    err := workflow.ExecuteActivity(ctx, ChargePayment, orderID).Get(ctx, nil)
    if err != nil {
        return err
    }

    // ขั้นตอนที่ 2: หักสต็อก
    err = workflow.ExecuteActivity(ctx, ReserveInventory, orderID).Get(ctx, nil)
    if err != nil {
        // compensate: คืนเงินที่เก็บไปแล้ว
        workflow.ExecuteActivity(ctx, RefundPayment, orderID).Get(ctx, nil)
        return err
    }

    // ขั้นตอนที่ 3: จัดส่ง
    return workflow.ExecuteActivity(ctx, ShipOrder, orderID).Get(ctx, nil)
}
```

โค้ดนี้อ่านเหมือนฟังก์ชันธรรมดาที่เรียงลำดับ if-else ปกติ — ความมหัศจรรย์ของ Temporal คือทำให้โค้ดที่**ดูเหมือน**รันแบบ synchronous ธรรมดานี้ ทนต่อการที่ worker (process ที่รัน workflow) ล่มกลางคันได้จริง โดยที่คนเขียนโค้ดไม่ต้องจัดการเรื่อง state persistence เอง

### Event History และ Durable Execution: กลไกที่สำคัญที่สุด

คำถามที่ต้องตอบให้ชัดเจนคือ: ถ้า worker ที่กำลังรัน `OrderWorkflow` ล่มไปตอนกำลังรอผลลัพธ์จาก `ReserveInventory` แล้ว worker ตัวใหม่ (หรือตัวเดิมที่ restart) จะรู้ได้อย่างไรว่า "ตอนนี้ทำถึงขั้นตอนไหนแล้ว, `ChargePayment` สำเร็จไปแล้วหรือยัง"

คำตอบคือ Temporal ไม่ได้เก็บ "state ปัจจุบัน" ของ workflow ไว้ตรงๆ แต่เก็บ **event history** — ลำดับเหตุการณ์ทั้งหมดที่เกิดขึ้นกับ workflow นี้แบบ append-only (แนวคิดเดียวกับ event sourcing ที่อธิบายใน [Volume 11 — Event Sourcing](../11-microservices/03-data-patterns.md)) ทุกครั้งที่ workflow ทำอะไรสักอย่างที่มีผลต่อ state (เรียก activity, ตัดสินใจ branch ไหน, ได้ผลลัพธ์กลับมา) Temporal จะบันทึกเป็น event ต่อท้าย history นี้เสมอ:

```
Event History ของ OrderWorkflow(orderID="1001"):
  1. WorkflowExecutionStarted
  2. ActivityTaskScheduled (ChargePayment)
  3. ActivityTaskCompleted (ChargePayment, result=success)
  4. ActivityTaskScheduled (ReserveInventory)
  ── worker ล่มตรงนี้ ระหว่างรอผล ReserveInventory ──
```

เมื่อ worker ตัวใหม่รับหน้าที่ต่อ Temporal จะสั่งให้ worker **replay** event history ทั้งหมดตั้งแต่ต้น — คือรันโค้ด workflow ใหม่ตั้งแต่บรรทัดแรก แต่แทนที่จะเรียก activity จริงซ้ำ (ซึ่งจะเก็บเงินซ้ำ!) Temporal จะ**คืนผลลัพธ์ที่บันทึกไว้ใน history ทันที**สำหรับทุก activity ที่เคยเสร็จไปแล้ว (event 2-3 ในตัวอย่างข้างบน) แล้วเดินหน้าโค้ดต่อไปจนกระทั่งถึงจุดที่ยังไม่มีผลลัพธ์ใน history (event 4 ที่ยังรอ `ReserveInventory` อยู่) จากจุดนั้น worker ตัวใหม่จะรันต่อจริงๆ (เรียก `ReserveInventory` จริง)

```
Replay:
  บรรทัดที่ 1 (ChargePayment)      ──► เจอผลลัพธ์ใน history แล้ว (event 3) ──► ใช้ผลลัพธ์เดิม ไม่เรียกซ้ำ
  บรรทัดที่ 2 (ReserveInventory)   ──► ไม่มีผลลัพธ์ใน history ──► รันจริง เรียก activity จริง
```

นี่คือกลไกที่ทำให้ workflow "จำ" สถานะของตัวเองได้แม้ worker จะ crash/restart กี่ครั้งก็ตาม — **in-memory state ของ workflow ไม่เคยถูกเก็บโดยตรงเลย มันถูก "สร้างขึ้นใหม่" ทุกครั้งผ่านการ replay event history** ซึ่งเป็นข้อมูลเดียวที่ Temporal server เก็บไว้อย่างถาวรจริงๆ

### Determinism Requirement: ทำไมห้าม `time.Now()` และ Random ตรงๆ

กลไก replay ข้างต้นทำงานได้ถูกต้องก็ต่อเมื่อ**โค้ด workflow ต้อง deterministic เท่านั้น** — หมายความว่า รันโค้ดเดิมซ้ำกี่ครั้งด้วย input เดิม ต้องได้ผลลัพธ์ (และลำดับการเรียก activity) เหมือนเดิมทุกครั้ง ถ้าโค้ด workflow เรียก `time.Now()` ตรงๆ ผลลัพธ์ตอน replay จะ**ไม่เท่ากับ**ตอนรันจริงครั้งแรก (เพราะเวลาผ่านไปแล้ว) ทำให้ branch การตัดสินใจของโค้ดอาจเบี่ยงไปคนละทางกับที่เคยเกิดขึ้นจริง — history ที่บันทึกไว้จะไม่ตรงกับสิ่งที่โค้ด replay สร้างขึ้นมาใหม่ ทำให้ Temporal ตรวจพบ **non-determinism error** และ workflow ล้มเหลว

สิ่งที่ห้ามทำโดยตรงในโค้ด workflow และเหตุผล:

| ห้ามทำ | เพราะอะไร | Temporal ให้ใช้แทน |
|---|---|---|
| `time.Now()` | ค่าต่างกันทุกครั้งที่ replay | `workflow.Now(ctx)` — อ่านค่าจาก event history แทนนาฬิกาจริง |
| `time.Sleep()` | ไม่ persist การรอข้าม worker restart | `workflow.Sleep(ctx, duration)` — Temporal จัดการ timer แบบ durable |
| สุ่มเลข (`rand.Int()`) | ค่าต่างกันทุกครั้งที่ replay | `workflow.SideEffect(ctx, ...)` — บันทึกผลลัพธ์สุ่มลง history ครั้งแรก แล้วใช้ค่าเดิมตอน replay |
| เรียก API/DB ตรงๆ ในโค้ด workflow | side effect ที่ไม่ควรเกิดซ้ำตอน replay และไม่ persist ผลลัพธ์ | ต้องทำผ่าน **Activity** เท่านั้น (ดูหัวข้อถัดไป) |
| Goroutine ปกติ ของภาษา / global mutable state | Temporal ควบคุม scheduling ของ concurrency ภายใน workflow เอง เพื่อให้ replay ได้ผลเหมือนเดิม | `workflow.Go(ctx, ...)` — coroutine ที่ Temporal SDK จัดการ |

กฎทองคือ: **workflow code ห้ามมี side effect ใดๆ โดยตรง** ทุกอย่างที่แตะโลกภายนอก (เวลาจริง, สุ่มเลข, เรียก API, เขียน DB) ต้องผ่านกลไกที่ Temporal SDK เตรียมไว้ให้ ซึ่งจะบันทึกผลลัพธ์ลง event history เพื่อให้ replay ได้ค่าเดิมเสมอ

## 1.2 Activity

### Activity คือที่ที่ Side Effect อยู่ได้

**Activity** คือหน่วยงานที่**แยกออกจาก workflow logic โดยสิ้นเชิง** และเป็นที่เดียวที่โค้ดสามารถมี side effect ได้อย่างอิสระ — เรียก external API, อ่าน/เขียนฐานข้อมูล, ส่งอีเมล, เรียก payment gateway ทุกอย่างที่ "แตะโลกภายนอก" ต้องอยู่ใน activity ไม่ใช่ในตัว workflow เอง:

```go
func ChargePayment(ctx context.Context, orderID string) error {
    // side effect เต็มรูปแบบ: เรียก payment gateway จริง
    return paymentClient.Charge(ctx, orderID)
}
```

Activity **ไม่ต้อง deterministic** เพราะ Temporal ไม่ replay ตัว activity เอง — สิ่งที่ replay คือ**ผลลัพธ์**ของ activity ที่บันทึกไว้ใน event history (ตามที่อธิบายไปข้างต้น) ตัว activity function เองรันแค่ครั้งเดียวจริงๆ ต่อการเรียกหนึ่งครั้งที่สำเร็จ (หรือหลายครั้งถ้ามี retry — ดูรายละเอียดใน [บทที่ 2](02-retry-and-compensation.md))

การแยกส่วนนี้ชัดเจนคือหัวใจของการออกแบบระบบด้วย Temporal: **workflow คือ "แผนงาน" ที่บอกลำดับและเงื่อนไข ส่วน activity คือ "การกระทำจริง" ที่ไปแตะโลกภายนอก** แยกความรับผิดชอบสองอย่างนี้ออกจากกันทำให้ทั้งสองฝั่งทดสอบและปรับเปลี่ยนได้อย่างอิสระ — เปลี่ยน implementation ภายใน activity (เช่น เปลี่ยน payment gateway) โดยไม่กระทบ logic ของ workflow เลย

### Activity Timeout Types

Activity แต่ละตัวต้องกำหนด timeout เพื่อไม่ให้ workflow รอค้างตลอดไปถ้า activity มีปัญหา — Temporal มี timeout หลายแบบที่ควบคุมคนละมิติ:

| Timeout | ควบคุมอะไร |
|---|---|
| **ScheduleToStart** | เวลาตั้งแต่ activity ถูกสั่งให้ทำงาน จนถึงตอนที่ worker เริ่มหยิบไปทำงานจริง — timeout นี้สูงมักบ่งชี้ว่า worker pool ไม่พอ (ไม่มี worker ว่างมารับงาน) |
| **StartToClose** | เวลาตั้งแต่ worker เริ่มรัน activity จนถึงรันเสร็จ (หรือ error) — นี่คือ timeout ของ "งานจริง" ที่ activity ทำ กำหนดตามลักษณะงาน (เรียก API เร็วอาจตั้ง 10 วินาที, ประมวลผลไฟล์ใหญ่อาจตั้ง 30 นาที) |
| **ScheduleToClose** | เวลารวมทั้งหมดตั้งแต่ schedule จนถึง close สำเร็จ (รวมทั้งเวลารอคิวและเวลารันจริง รวมถึงเวลาที่ retry ไปด้วยหลายรอบ) — เป็นเพดานบนสุดของทั้งกระบวนการ |
| **Heartbeat timeout** | สำหรับ activity ที่ใช้เวลานาน (ดูหัวข้อถัดไป) |

### Heartbeat สำหรับ Activity ที่ใช้เวลานาน

Activity บางตัวใช้เวลานานมาก (เช่น ประมวลผลไฟล์ขนาดใหญ่ที่ใช้เวลาเป็นชั่วโมง) — ถ้าตั้ง `StartToClose` timeout ยาวมากเฉยๆ (เช่น 2 ชั่วโมง) ปัญหาคือ Temporal จะไม่รู้เลยว่า activity ที่กำลังรันอยู่ **ยังทำงานอยู่จริง (แค่ช้า) หรือ worker ที่รันมันตายไปแล้วเงียบๆ** ต้องรอจนครบ 2 ชั่วโมงเต็มถึงจะรู้ว่ามีปัญหา

**Heartbeat** แก้ปัญหานี้ — activity ที่ใช้เวลานานจะส่งสัญญาณ "ฉันยังทำงานอยู่" กลับไปยัง Temporal server เป็นระยะระหว่างทำงาน (เช่น ทุกครั้งที่ประมวลผลไฟล์เสร็จไป 1%):

```go
func ProcessLargeFile(ctx context.Context, fileID string) error {
    for i, chunk := range chunks {
        processChunk(chunk)
        activity.RecordHeartbeat(ctx, i) // บอก progress ปัจจุบัน
    }
    return nil
}
```

ถ้า Temporal ไม่ได้รับ heartbeat ภายในเวลาที่กำหนด (`HeartbeatTimeout`) จะถือว่า worker ที่รัน activity นี้ตายไปแล้ว แล้วสั่งให้ activity นี้ **retry บน worker ตัวอื่น**ทันที โดยไม่ต้องรอจนครบ `StartToClose` timeout เต็มจำนวน — นอกจากนี้ heartbeat ยังพก **progress data** ไปด้วยได้ (เช่น "ประมวลผลไปถึง chunk ที่เท่าไหร่แล้ว") ทำให้ตอน retry activity สามารถ**เริ่มต่อจากจุดที่ค้างไว้**แทนที่จะเริ่มใหม่ตั้งแต่ต้น ประหยัดเวลาไปได้มากสำหรับงานที่ใช้เวลานาน

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| Workflow | โค้ด deterministic ที่ describe ลำดับขั้นตอน business process |
| Durable execution | workflow รันจนจบเสมอแม้ worker ล่มกลางคัน เพราะ state สร้างใหม่ได้จาก event history |
| Event History | Temporal เก็บลำดับเหตุการณ์แบบ append-only ไม่ใช่ state ปัจจุบันตรงๆ |
| Replay | worker ใหม่รันโค้ดตั้งแต่ต้น แต่ใช้ผลลัพธ์เดิมจาก history แทนการเรียก activity ซ้ำ |
| Determinism requirement | ห้าม `time.Now()`/random/side effect ตรงๆ ในโค้ด workflow เพราะทำให้ replay ได้ผลไม่ตรงกับของเดิม |
| Activity | หน่วยงานที่มี side effect (API, DB) แยกจาก workflow logic โดยสิ้นเชิง |
| Timeout types | ScheduleToStart, StartToClose, ScheduleToClose ควบคุมคนละช่วงของ activity lifecycle |
| Heartbeat | activity ที่ใช้เวลานานส่งสัญญาณ+progress เป็นระยะ ตรวจจับ worker ตายได้เร็ว และ resume จากจุดค้างได้ |

## คำถามทบทวน

1. อธิบายกลไก replay ทีละขั้นตอน: ถ้า worker ล่มระหว่างรอผล activity ตัวที่สาม worker ใหม่จะรู้ได้อย่างไรว่าสอง activity แรกทำสำเร็จไปแล้ว
2. ทำไมการเรียก `time.Now()` ตรงๆ ในโค้ด workflow ถึงทำให้เกิด non-determinism error? `workflow.Now(ctx)` แก้ปัญหานี้อย่างไร
3. ทำไม side effect (เรียก API, เขียน DB) ต้องอยู่ใน activity เท่านั้น ไม่ใช่ในตัว workflow โดยตรง
4. เปรียบเทียบ `StartToClose` กับ `ScheduleToClose` timeout ว่าควบคุมช่วงเวลาต่างกันอย่างไร
5. อธิบายว่า heartbeat ช่วยแก้ปัญหาอะไรสำหรับ activity ที่ใช้เวลานาน และ progress data ที่แนบไปกับ heartbeat มีประโยชน์อย่างไรตอน retry

---

ถัดไป: [บทที่ 2 — Retry และ Compensation](02-retry-and-compensation.md)
