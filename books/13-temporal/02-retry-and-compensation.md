# บทที่ 2: Retry และ Compensation

## 2.1 Retry Policy

### กำหนด Retry Policy ให้ Activity

Activity ทุกตัวสามารถผูก **Retry Policy** เข้าไปได้ตอนเรียก เพื่อบอก Temporal ว่าถ้า activity ล้มเหลว ควรลองใหม่อย่างไร:

```go
retryPolicy := &temporal.RetryPolicy{
    InitialInterval:    time.Second,        // รอ 1 วินาทีก่อน retry ครั้งแรก
    BackoffCoefficient: 2.0,                // แต่ละครั้งรอนานขึ้น 2 เท่า (exponential backoff)
    MaximumInterval:    time.Minute,        // แต่ interval สูงสุดไม่เกิน 1 นาที
    MaximumAttempts:    5,                  // ลองรวมทั้งหมดไม่เกิน 5 ครั้ง
    NonRetryableErrorTypes: []string{"InvalidCardError"}, // error บางชนิดไม่ควร retry เลย
}

opts := workflow.ActivityOptions{
    StartToCloseTimeout: 10 * time.Second,
    RetryPolicy:         retryPolicy,
}
ctx = workflow.WithActivityOptions(ctx, opts)
err := workflow.ExecuteActivity(ctx, ChargePayment, orderID).Get(ctx, nil)
```

พารามิเตอร์แต่ละตัวควบคุมคนละมิติ:

| พารามิเตอร์ | ความหมาย |
|---|---|
| `InitialInterval` | ระยะเวลารอก่อน retry ครั้งแรก |
| `BackoffCoefficient` | อัตราการเพิ่มระยะเวลารอในแต่ละครั้งถัดไป (exponential backoff — เชื่อมกับแนวคิดเดียวกันใน [Volume 11 — Retry](../11-microservices/04-saga-and-resilience.md)) |
| `MaximumInterval` | เพดานบนของระยะเวลารอ ไม่ให้ backoff โตไม่จำกัด |
| `MaximumAttempts` | จำนวนครั้งสูงสุดที่ retry (0 = ไม่จำกัด) |
| `NonRetryableErrorTypes` | error บางประเภทไม่มีประโยชน์ที่จะ retry เลย (เช่น "บัตรเครดิตถูกปฏิเสธ" — ลองใหม่กี่ครั้งก็ผลลัพธ์เดิม) ควรโยน error กลับไปให้ workflow จัดการ (เช่น compensate) ทันทีแทนที่จะเสียเวลา retry |

### ทำไม Retry ของ Temporal ต่างจาก Retry Loop ที่เขียนเอง

ลองเทียบกับการเขียน retry loop มือเปล่าในโค้ดทั่วไป:

```go
// Retry loop แบบเขียนเอง — เปราะบาง
for attempt := 0; attempt < 5; attempt++ {
    err := chargePayment(orderID)
    if err == nil {
        break
    }
    time.Sleep(time.Duration(attempt) * time.Second)
}
```

โค้ดแบบนี้มีปัญหาที่มองไม่เห็นจนกว่าจะเกิดจริง: **ถ้า process ที่รัน loop นี้ crash ระหว่างกลาง (เช่น ตอน attempt ที่ 3 กำลังรออยู่) ความคืบหน้าทั้งหมดหายไปทันที** process ใหม่ที่มารันต่อไม่มีทางรู้เลยว่าเพิ่ง retry ไปกี่ครั้งแล้ว ต้องเริ่มนับ attempt ใหม่ตั้งแต่ 0 — หรือแย่กว่านั้นคือไม่รู้ด้วยซ้ำว่า `chargePayment` ครั้งล่าสุดสำเร็จไปแล้วหรือยังก่อนที่ process จะตาย (เชื่อมกับปัญหา idempotency ใน [Volume 11](../11-microservices/05-idempotency-and-tracing.md))

Retry policy ของ Temporal ต่างออกไปโดยพื้นฐาน: **จำนวนครั้งที่ retry ไปแล้วและระยะเวลาที่ต้องรอครั้งถัดไป ถูกบันทึกเป็นส่วนหนึ่งของ event history** (บทที่ 1) เช่นเดียวกับทุกอย่างที่เกิดกับ workflow ถ้า worker ตายระหว่างรอ retry ครั้งที่ 3 worker ตัวใหม่ที่มารับช่วงต่อจะ replay history แล้วรู้ทันทีว่า "ตอนนี้อยู่ที่ attempt 3 แล้ว รออีก 4 วินาทีถึงจะ retry ครั้งถัดไป" — ความคืบหน้าของ retry ไม่มีวันหายไปพร้อม process ที่ตาย

## 2.2 Compensation และ Saga Pattern

### ปัญหา: ทำสำเร็จไปครึ่งทาง แล้วขั้นตอนหลังล้มเหลว

ระบบแบบ microservices ที่ business process หนึ่งกระบวนการต้องแตะหลาย service (เชื่อมกับ [Volume 11 — Saga](../11-microservices/04-saga-and-resilience.md)) ไม่มี distributed transaction แบบ ACID ให้ใช้ข้าม service ถ้าขั้นตอนที่ 1 และ 2 สำเร็จไปแล้ว แต่ขั้นตอนที่ 3 ล้มเหลว (แม้จะ retry จนหมดสิทธิ์แล้วก็ตาม) ระบบต้อง**ย้อนกลับผลของขั้นตอนที่ทำสำเร็จไปแล้ว** ด้วย compensating action ของตัวเอง — Temporal เหมาะเป็นตัว orchestrate Saga แบบนี้อย่างมาก เพราะ workflow code ที่อ่านง่ายเหมือนโค้ด sequential ธรรมดาสามารถแทรก compensation logic ไว้ในจุดที่ควรได้ตรงไปตรงมา (ต่างจาก choreography-based Saga ที่ logic การชดเชยกระจายอยู่หลาย service มองเห็นภาพรวมได้ยากกว่า)

### ตัวอย่างเต็ม: Order + Payment + Inventory

```go
func OrderWorkflow(ctx workflow.Context, orderID string) error {
    opts := workflow.ActivityOptions{
        StartToCloseTimeout: 10 * time.Second,
        RetryPolicy:         defaultRetryPolicy,
    }
    ctx = workflow.WithActivityOptions(ctx, opts)

    // ขั้นตอนที่ 1: เก็บเงิน
    err := workflow.ExecuteActivity(ctx, ChargePayment, orderID).Get(ctx, nil)
    if err != nil {
        // ยังไม่มีขั้นตอนไหนสำเร็จมาก่อน ไม่ต้อง compensate อะไร
        return fmt.Errorf("charge payment failed: %w", err)
    }

    // ขั้นตอนที่ 2: หักสต็อก
    err = workflow.ExecuteActivity(ctx, ReserveInventory, orderID).Get(ctx, nil)
    if err != nil {
        // ขั้นตอนที่ 1 (เก็บเงิน) สำเร็จไปแล้ว ต้อง compensate ด้วยการคืนเงิน
        compErr := workflow.ExecuteActivity(ctx, RefundPayment, orderID).Get(ctx, nil)
        if compErr != nil {
            // compensation เองก็ล้มเหลว — ต้อง alert ให้คนเข้ามาดูแล (ไม่ควร silent fail)
            workflow.ExecuteActivity(ctx, AlertOpsTeam, orderID, compErr).Get(ctx, nil)
        }
        return fmt.Errorf("reserve inventory failed, payment refunded: %w", err)
    }

    // ขั้นตอนที่ 3: จัดส่ง
    err = workflow.ExecuteActivity(ctx, ShipOrder, orderID).Get(ctx, nil)
    if err != nil {
        // ขั้นตอนที่ 1 และ 2 สำเร็จไปแล้ว ต้อง compensate ทั้งคู่ ตามลำดับย้อนกลับ
        workflow.ExecuteActivity(ctx, ReleaseInventory, orderID).Get(ctx, nil)
        workflow.ExecuteActivity(ctx, RefundPayment, orderID).Get(ctx, nil)
        return fmt.Errorf("ship order failed, inventory released and payment refunded: %w", err)
    }

    return nil
}
```

### สังเกตหลักการสำคัญ 3 ข้อจากตัวอย่างนี้

1. **Compensation ต้องย้อนลำดับกับขั้นตอนที่ทำสำเร็จ** — ถ้าล้มเหลวตอนขั้นตอนที่ 3 ต้อง compensate ขั้นตอนที่ 2 ก่อน แล้วค่อยขั้นตอนที่ 1 (เหมือนแกะสแต็ก ไม่ใช่ทำตามลำดับเดิม)
2. **แต่ละ compensating action ก็คือ activity ธรรมดา** ที่มี retry policy ของตัวเองได้เช่นกัน (เช่น `RefundPayment` อาจ retry เองถ้า payment gateway timeout ชั่วคราว)
3. **compensation ที่ล้มเหลวเองต้องมีทางออกสุดท้ายเสมอ** (ในตัวอย่างคือ `AlertOpsTeam`) — ไม่มีระบบอัตโนมัติใดรับประกัน compensation จะสำเร็จ 100% เสมอ (เช่น payment gateway ล่มยาวจนเกิน retry budget) การออกแบบ Saga ที่ดีต้องมีจุดที่ยอมรับว่า "ต้องให้คนเข้ามาแก้ไข" แทนที่จะพยายาม retry ไม่มีที่สิ้นสุดหรือ silent fail

### ข้อดีของการใช้ Temporal Orchestrate Saga เทียบกับเขียนเอง

ถ้าเขียน orchestration logic นี้ด้วยมือ (เช่น เก็บ state "ทำถึงขั้นตอนไหนแล้ว" ไว้ใน database ตาราง state machine เอง) ต้องจัดการเองทั้งหมด: ถ้า process ที่ orchestrate ล่มระหว่างกำลัง compensate ใครจะมารันต่อ, จะรู้ได้อย่างไรว่า compensation ไหนทำไปแล้วบ้าง — ทั้งหมดนี้คือสิ่งที่ **durable execution** จากบทที่ 1 แก้ให้แบบอัตโนมัติอยู่แล้ว โค้ด Saga ที่เขียนเป็น Temporal workflow จึงอ่านง่ายเหมือน pseudo-code ตรงไปตรงมา แต่ได้ความทนทานต่อความล้มเหลวระดับ production โดยไม่ต้องเขียน state machine เองเลย

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| Retry Policy | กำหนด initial interval, backoff coefficient, max attempts, non-retryable error ต่อ activity |
| Retry state ที่ persist | จำนวนครั้ง/เวลาที่ retry ไปแล้วอยู่ใน event history ไม่หายแม้ worker ตาย ต่างจาก retry loop มือเปล่า |
| Saga ผ่าน Temporal | workflow code สั่ง compensating activity เรียงย้อนลำดับกับขั้นตอนที่สำเร็จไปแล้ว |
| Compensation ที่ล้มเหลว | ต้องมีทางออกสุดท้ายเสมอ (alert คน) ไม่ใช่ retry ไม่จำกัดหรือ silent fail |

## คำถามทบทวน

1. อธิบายว่าทำไม retry ของ Temporal ถึงทนต่อ worker crash ได้ ในขณะที่ retry loop ที่เขียนเองในโค้ดธรรมดาไม่ทน
2. `NonRetryableErrorTypes` มีประโยชน์อย่างไร ยกตัวอย่าง error หนึ่งที่ควรอยู่ในลิสต์นี้พร้อมเหตุผล
3. ในตัวอย่าง Order Workflow ถ้าล้มเหลวตอนขั้นตอน `ShipOrder` ทำไมต้อง compensate `ReserveInventory` ก่อน `ChargePayment` ไม่ใช่ตามลำดับเดิม
4. ถ้า compensating action เองล้มเหลวด้วย (เช่น `RefundPayment` เรียกไม่สำเร็จ) ควรออกแบบระบบอย่างไรต่อ ทำไมถึงไม่ควร retry ไปเรื่อยๆ อย่างไม่มีที่สิ้นสุด

---

ก่อนหน้า: [บทที่ 1 — Workflow และ Activity](01-workflow-and-activity.md) | กลับสู่ [สารบัญ Volume 13](README.md)
