# Volume 1 — Computer Foundation

พื้นฐานที่วิศวกร backend ทุกคนต้องเข้าใจ ก่อนจะไปเขียนโค้ดหรือดีไซน์ระบบใดๆ เพราะทุกปัญหาประสิทธิภาพ (performance), การ debug เรื่อง concurrency, หรือการเข้าใจว่าทำไม container ถึงใช้ CPU/Memory แบบที่เห็น ล้วนย้อนกลับไปที่ความรู้พื้นฐานเรื่อง Computer Architecture และ Operating System ทั้งสิ้น

## เป้าหมายการเรียนรู้

- อธิบายได้ว่า CPU ประมวลผลคำสั่งอย่างไร และทำไม cache/memory hierarchy ถึงสำคัญกับ performance
- เข้าใจความแตกต่างระหว่าง process กับ thread และผลกระทบต่อการออกแบบโปรแกรม concurrent
- อธิบาย virtual memory และ scheduling ได้ในระดับที่เชื่อมโยงกับพฤติกรรมจริงของแอปพลิเคชัน (เช่น OOM kill, context switch overhead)

## สารบัญ

### [บทที่ 1 — Computer Architecture](01-computer-architecture.md)

- [ ] **CPU** — instruction cycle (fetch-decode-execute), pipeline, clock speed vs IPC
- [ ] **Core / Thread (hardware)** — multi-core vs hyper-threading (SMT), เมื่อไหร่ core เพิ่มแล้วเร็วขึ้นจริง เมื่อไหร่ไม่ช่วย
- [ ] **Cache** — L1/L2/L3, cache line, cache miss, false sharing, ทำไมโค้ดที่เข้าถึงข้อมูลแบบ sequential ถึงเร็วกว่า random
- [ ] **Memory** — RAM, memory bus, bandwidth vs latency
- [ ] **NUMA** — Non-Uniform Memory Access คืออะไร ทำไม server หลาย socket ถึงต้องสนใจเรื่อง NUMA affinity

### [บทที่ 2 — Operating System](02-operating-system.md)

- [ ] **Process** — process control block, address space, process states (running/waiting/ready)
- [ ] **Thread** — thread ภายใน process เดียวกันแชร์อะไรบ้าง ต่างจาก process อย่างไร, user-level vs kernel-level thread
- [ ] **Virtual Memory** — page table, paging, page fault, swap, ความสัมพันธ์กับ container memory limit
- [ ] **Scheduling** — scheduling algorithm (round-robin, CFS ของ Linux), context switch คืออะไรและมี cost เท่าไหร่, priority & nice value

## เชื่อมโยงกับ Volume อื่น

ความรู้ในเล่มนี้เป็นพื้นฐานของ [Volume 3 — Linux & Git](../03-linux/README.md) (process/scheduling ที่เห็นผ่านคำสั่ง `ps`, `top`) และ [Volume 4 — Golang](../04-golang/README.md) (goroutine scheduler ที่จำลองแนวคิด thread scheduling ในระดับ user-space)

## แหล่งอ้างอิงแนะนำ

ดูลิงก์และหนังสือแนะนำเพิ่มเติมได้ที่ [resources/](../../resources/README.md)
