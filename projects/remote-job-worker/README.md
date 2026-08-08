# remote-job-worker

Reference implementation ของ "remote Temporal worker" ตาม contract ที่ [REMOTE-JOB-WORKER-TODO.md](../bot-orchestration-platform/REMOTE-JOB-WORKER-TODO.md) (ใน [projects/bot-orchestration-platform/](../bot-orchestration-platform/README.md), แพลตฟอร์มหลักคือ **Bot Orchestration Platform / BOP**) กำหนดไว้ — พิสูจน์ว่า process/repo/Go module ที่แยกจาก BOP โดยสมบูรณ์ สามารถรับงานที่ BOP สั่งผ่าน Temporal task queue ได้จริง โดยที่ **ทั้งสองฝั่งไม่ต้อง import โค้ดของกันและกันเลยแม้แต่บรรทัดเดียว** (คุยกันด้วย JSON ผ่าน Temporal default DataConverter ล้วนๆ)

โมดูลนี้เป็น **placeholder/reference** ไม่ใช่ implementation ของ jobcode จริงของเจ้าของ repo (ยังไม่มีให้ทดสอบตอนเขียนโค้ดนี้ — ดู [EXTERNAL-JOB-BOT-TODO.md](../bot-orchestration-platform/EXTERNAL-JOB-BOT-TODO.md) Open Question ข้อ 1 ที่ตัดสินใจไว้เหมือนกัน) — สลับ jobcode dispatch ใน `internal/jobs/jobs.go` เป็น logic จริงได้เมื่อรู้ jobcode จริงแล้ว โครง worker/main.go ไม่ต้องแก้เลย

## สถาปัตยกรรม

```
cmd/worker/main.go     ต่อ Temporal client, เปิด worker.Worker บน task queue "external-job-queue",
                        register jobs.RunStep เป็น activity ชื่อ "RunStep"
internal/jobs/jobs.go  Step (input type, JSON shape ตรงกับ workflow.Step ของฝั่ง BOP) +
                        RunStep (activity function, dispatch ตาม Config["jobcode"])
```

ไม่มี `internal/workflow`/workflow function ใดๆ ในโมดูลนี้เลย — worker ตัวนี้มีหน้าที่แค่ "รับ-ทำ Activity" เท่านั้น ตัว workflow orchestration (ลำดับ step, retry policy, durable execution, scheduling) ยังเป็นหน้าที่ของ BOP (`cmd/bop-worker`) ทั้งหมดเหมือนเดิม

## jobcode ตัวอย่างที่ implement ไว้ (placeholder)

| jobcode | พฤติกรรม | สาธิตอะไร |
|---|---|---|
| `PING` | สำเร็จทันที คืน `"pong"` | smoke test พื้นฐาน |
| `LONG_TASK` | จำลองงานนาน 10 วินาที ส่ง heartbeat ทุก 2 วินาที | `activity.RecordHeartbeat` + เช็ค `ctx.Done()` ระหว่างทำงาน |
| `FAIL_TRANSIENT` | คืน error ธรรมดา (retryable) | ปัญหาชั่วคราว (เช่น dependency ภายนอกล่มชั่วคราว) ที่อยาก retry ตาม default |
| ว่างเปล่า/ไม่รู้จัก | คืน `temporal.ApplicationError` ชนิด `"ConfigError"` (non-retryable) | jobcode ผิด — retry กี่ครั้งก็ผลลัพธ์เดิม |

## รันยังไง

```bash
go build ./...
go vet ./...
go test -race ./...

# ต้องมี Temporal server รันอยู่ก่อน (docker-compose.yml ของ projects/bot-orchestration-platform/)
cd ../bot-orchestration-platform && docker compose up -d && cd -
go run ./cmd/worker    # เริ่ม poll task queue "external-job-queue"
```

ทดสอบ trigger จากฝั่ง BOP (ต้องรัน `cmd/bop-worker` และ `cmd/bop` ของ BOP ด้วย — ดู README ของ BOP):

```bash
curl -X POST localhost:8080/workflow-runs -d '{
  "workflow_name": "test-remote-ping",
  "steps": [{"bot_name": "remote-ping", "config": {"jobcode": "PING"}, "task_queue": "external-job-queue"}]
}'
curl localhost:8080/workflow-runs/<workflow_id>   # ควรเห็น step สำเร็จ summary "pong"
```

## Reference

- [REMOTE-JOB-WORKER-TODO.md](../bot-orchestration-platform/REMOTE-JOB-WORKER-TODO.md) — contract เต็มที่โมดูลนี้ implement ตาม
- [EXTERNAL-JOB-BOT-TODO.md](../bot-orchestration-platform/EXTERNAL-JOB-BOT-TODO.md) — ทางเลือกแรก (local exec ในเครื่อง/container เดียวกับ BOP) เหมาะกับ binary ที่ไม่คุ้มแก้เป็น Temporal worker แบบนี้
