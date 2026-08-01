# บทที่ 1: Computer Architecture

## 1.1 CPU

### Instruction Cycle

CPU ทำงานเป็นวงจรซ้ำๆ ที่เรียกว่า **fetch-decode-execute cycle**:

1. **Fetch** — ดึงคำสั่ง (instruction) จาก memory ตามตำแหน่งที่ Program Counter (PC) ชี้อยู่
2. **Decode** — แปลรหัสคำสั่งว่าเป็นคำสั่งอะไร ต้องใช้ operand ไหนบ้าง
3. **Execute** — ประมวลผลจริงใน ALU (Arithmetic Logic Unit) หรือหน่วยประมวลผลอื่น
4. **Writeback** — เขียนผลลัพธ์กลับไปที่ register หรือ memory

วงจรนี้เกิดขึ้นนับพันล้านครั้งต่อวินาที และเป็นพื้นฐานของทุกอย่างที่ CPU ทำ

### Pipeline

ถ้า CPU ทำ 4 ขั้นตอนนี้แบบเรียงลำดับทีละคำสั่งจนจบแล้วค่อยเริ่มคำสั่งถัดไป จะช้ามาก เทคนิค **pipelining** ทำให้ CPU ประมวลผลหลายคำสั่งพร้อมกันแบบ overlap โดยแต่ละ stage ทำงานกับคำสั่งคนละตัว คล้ายสายการผลิตในโรงงาน:

```
Cycle:     1     2     3     4     5     6
Instr 1:   F     D     E     W
Instr 2:         F     D     E     W
Instr 3:               F     D     E     W
```

ปัญหาที่ทำให้ pipeline สะดุด (hazard) มี 3 แบบหลัก:

- **Structural hazard** — hardware ไม่พอให้สองคำสั่งใช้พร้อมกัน (เช่น แย่งใช้หน่วยความจำเดียวกัน)
- **Data hazard** — คำสั่งถัดไปต้องใช้ผลลัพธ์ของคำสั่งก่อนหน้าที่ยังไม่เสร็จ
- **Control hazard** — เจอคำสั่ง branch (if/jump) แล้วไม่รู้ว่าจะ fetch คำสั่งไหนต่อ ต้องเดา (branch prediction) ถ้าเดาผิดต้องล้าง pipeline ทิ้ง (pipeline flush) ซึ่งมี cost สูง

CPU สมัยใหม่ยังมี **superscalar execution** (ประมวลผลหลายคำสั่งใน stage เดียวกันพร้อมกัน) และ **out-of-order execution** (สลับลำดับการ execute คำสั่งเพื่อไม่ให้รอ data hazard โดยยังคง "ผลลัพธ์" ให้เหมือนกับรันตามลำดับเดิม)

### Clock Speed vs IPC

ความเร็ว CPU มักถูกวัดด้วย **clock speed** (GHz) แต่ตัวเลขนี้บอกความเร็วได้ไม่ครบ ตัวชี้วัดที่สำคัญกว่าคือ:

```
Performance ∝ Clock Speed × IPC (Instructions Per Cycle)
```

CPU ที่ clock speed ต่ำกว่าแต่ IPC สูงกว่า (ทำงานได้หลายคำสั่งต่อ 1 clock cycle จาก superscalar/pipeline ที่ดีกว่า) อาจเร็วกว่า CPU ที่ clock speed สูงกว่าได้จริง นี่คือเหตุผลที่การเทียบ CPU ด้วย "GHz" อย่างเดียวถึงเข้าใจผิดได้ง่าย

> **ทำไมต้องรู้เรื่องนี้ในฐานะ backend engineer?** เวลาเลือก instance type บน cloud หรือวิเคราะห์ผล benchmark ต้องเข้าใจว่า core มากไม่ได้แปลว่าเร็วเสมอ และ clock speed สูงไม่ได้แปลว่า throughput สูงเสมอ

## 1.2 Core และ Thread (ระดับ Hardware)

- **Multi-core** — CPU ตัวเดียวมีหลายแกนประมวลผล (core) แต่ละ core ทำงานอิสระจากกันได้จริง เป็น parallelism แบบ "hardware จริง"
- **SMT (Simultaneous Multithreading)** หรือที่ Intel เรียก **Hyper-Threading** — 1 core แสดงตัวเป็น 2 logical thread (เช่น CPU 8 core 16 thread) โดยแชร์ execution unit เดียวกัน แต่มี register set แยกกัน

จุดสำคัญ: SMT **ไม่ใช่** การเพิ่ม core จริง มันแค่ทำให้ core เดิมใช้ resource ว่าง (เช่น ตอนรอข้อมูลจาก memory) ได้มีประสิทธิภาพขึ้น เพราะฉะนั้น 16 logical thread ไม่ได้แปลว่าเร็วเป็น 2 เท่าของ 8 core จริง งานที่ compute-heavy และแย่ง execution unit กันหนักๆ อาจได้ประโยชน์จาก SMT น้อยมาก หรือบางกรณีก็แย่ลงด้วยซ้ำ (resource contention)

> **เชื่อมโยง**: นี่คือเหตุผลที่ K8s `nproc`/CPU request มักอ้างอิง logical CPU (vCPU) ไม่ใช่ physical core — เวลาตั้ง resource limit ต้องเข้าใจว่า vCPU ที่เห็นอาจเป็น SMT thread ไม่ใช่ core จริงทั้งหมด

## 1.3 Cache

### ทำไมต้องมี Cache

CPU เร็วกว่า RAM มาก (นับสิบถึงร้อยเท่า) ถ้าทุกครั้งที่อ่าน/เขียนข้อมูลต้องวิ่งไปที่ RAM โดยตรง CPU จะเสียเวลารอมหาศาล **Cache** คือหน่วยความจำขนาดเล็กแต่เร็วมาก วางอยู่ระหว่าง CPU กับ RAM เพื่อเก็บข้อมูลที่มีโอกาสถูกใช้ซ้ำ

### Memory Hierarchy

```
Register  →  L1 Cache  →  L2 Cache  →  L3 Cache  →  RAM  →  Disk/SSD
 (เร็วสุด)                                                    (ช้าสุด แต่จุมากสุด)
```

- **L1** — เล็กที่สุด (~32-64 KB ต่อ core) เร็วที่สุด แยกเป็น L1i (instruction) กับ L1d (data)
- **L2** — ใหญ่กว่า (~256KB-1MB) ต่อ core ยังค่อนข้างเร็ว
- **L3** — ใหญ่สุดในบรรดา cache (หลาย MB) มักแชร์กันทุก core ใน CPU ตัวเดียว

### Cache Line

CPU ไม่ได้โหลดข้อมูลทีละ byte แต่โหลดเป็นก้อนที่เรียกว่า **cache line** (ทั่วไปขนาด 64 bytes) เพราะข้อมูลที่อยู่ติดกันมักถูกใช้พร้อมกัน (**spatial locality**) และข้อมูลที่เพิ่งใช้มีโอกาสถูกใช้ซ้ำเร็วๆ นี้ (**temporal locality**) — สองหลักการนี้คือรากฐานของการออกแบบ cache ทั้งหมด

### Cache Miss

เมื่อ CPU หาข้อมูลใน cache ไม่เจอ เรียกว่า **cache miss** แบ่งเป็น 3 ประเภท:

- **Compulsory miss** — ครั้งแรกที่เข้าถึงข้อมูล ยังไม่เคยถูกโหลดเข้า cache เลย (หลีกเลี่ยงไม่ได้)
- **Capacity miss** — cache เล็กเกินไป เก็บข้อมูลที่ใช้งานอยู่ทั้งหมดไม่ได้ ข้อมูลเก่าเลยถูกไล่ที่ (evict) ออกไปก่อนจะถูกใช้ซ้ำ
- **Conflict miss** — ข้อมูลหลายตัวแมปมาลง cache slot เดียวกัน (ขึ้นกับ cache associativity) ทำให้แย่งที่กันแม้ cache โดยรวมยังมีที่ว่าง

### False Sharing

ปัญหานี้สำคัญมากสำหรับโค้ด concurrent เพราะ CPU จัดการ cache coherence เป็นหน่วย **cache line** ไม่ใช่ทีละ byte ถ้าตัวแปรสองตัวที่ไม่เกี่ยวข้องกันเลย (เช่น ใช้กันคนละ goroutine/thread) ถูกจัดเก็บอยู่ใน cache line เดียวกัน การเขียนตัวแปรหนึ่งจะทำให้ core ที่ใช้อีกตัวแปรหนึ่งต้อง invalidate cache line ของตัวเองทั้งบรรทัด แม้จะไม่ได้แตะตัวแปรเดียวกันเลยก็ตาม

ตัวอย่าง (แนวคิด ใน Go):

```go
type Counter struct {
    a int64 // ถูกเขียนโดย goroutine A บ่อยๆ
    b int64 // ถูกเขียนโดย goroutine B บ่อยๆ — อยู่ cache line เดียวกับ a!
}
```

ถ้า `a` และ `b` ถูกเขียนพร้อมกันโดยคนละ core บ่อยๆ ประสิทธิภาพจะแย่ลงมาก ทั้งที่ไม่มี logical race เลย วิธีแก้คือเพิ่ม **padding** ให้แต่ละตัวแปรอยู่คนละ cache line:

```go
type Counter struct {
    a int64
    _ [56]byte // padding ให้ a กินเต็ม cache line (64 bytes)
    b int64
}
```

### Cache Mapping และ Write Policy

- **Direct-mapped** — แต่ละตำแหน่งใน memory แมปได้กับ cache slot เดียวเท่านั้น (เร็วแต่ conflict miss ง่าย)
- **Fully associative** — ข้อมูลวางที่ slot ไหนก็ได้ใน cache (ยืดหยุ่นสุดแต่แพงสุดในการค้นหา)
- **Set-associative** — ทางเลือกกลาง (เช่น 8-way set associative) ที่ CPU จริงส่วนใหญ่ใช้
- **Write-through** — เขียนข้อมูลลง cache และ memory พร้อมกันทันที (ช้ากว่าแต่ปลอดภัยกว่า)
- **Write-back** — เขียนลง cache ก่อน แล้วค่อย flush ลง memory ทีหลัง (เร็วกว่าแต่ซับซ้อนกว่าเรื่อง consistency)

## 1.4 Memory

- **RAM** — หน่วยความจำหลักที่ volatile (ข้อมูลหายเมื่อไฟดับ) เร็วกว่า disk มากแต่ยังช้ากว่า cache หลายสิบเท่า
- **Memory Bus** — เส้นทางที่ข้อมูลวิ่งระหว่าง CPU กับ RAM มีทั้ง address bus (บอกตำแหน่ง) และ data bus (ส่งข้อมูลจริง)
- **Bandwidth vs Latency** — สอง metric ที่ต่างกันและสำคัญคนละแบบ:
  - **Bandwidth** คือปริมาณข้อมูลที่ส่งได้ต่อวินาที (GB/s)
  - **Latency** คือเวลาที่ใช้กว่าจะได้ข้อมูลก้อนแรกกลับมา (nanosecond)

ระบบที่มี bandwidth สูงแต่ latency สูงด้วย (เช่น อ่านข้อมูลจาก disk แบบ sequential จำนวนมาก) เหมาะกับงาน batch/streaming ส่วนระบบที่ latency ต่ำสำคัญกว่า (เช่น random access เล็กๆ จำนวนมาก) จะรู้สึกช้าแม้ bandwidth จะสูงก็ตาม — หลักการเดียวกันนี้ใช้อธิบายพฤติกรรมของ network และ disk ได้ด้วย ไม่ใช่แค่ RAM

## 1.5 NUMA (Non-Uniform Memory Access)

Server ระดับ enterprise มักมี CPU มากกว่า 1 ตัว (multi-socket) แต่ละ socket มี memory ของตัวเอง เรียกว่า **NUMA node** CPU สามารถเข้าถึง memory ของ node ตัวเองได้เร็ว (local access) แต่ถ้าต้องเข้าถึง memory ของอีก socket หนึ่ง (remote access) จะช้ากว่าอย่างมีนัยสำคัญ เพราะต้องวิ่งผ่าน interconnect ระหว่าง socket (เช่น Intel QPI/UPI, AMD Infinity Fabric)

```
┌─────────────┐         ┌─────────────┐
│   CPU 0     │◄───────►│   CPU 1     │
│  (Node 0)   │Interconnect│ (Node 1)  │
├─────────────┤         ├─────────────┤
│  RAM (local)│         │  RAM (local)│
└─────────────┘         └─────────────┘
```

ผลกระทบต่องาน backend:

- แอปพลิเคชันที่ thread ถูก schedule ไปอยู่บน CPU node หนึ่ง แต่ข้อมูลที่ต้องใช้ถูก allocate ไว้บน memory ของอีก node หนึ่ง จะช้าลงโดยไม่มีเหตุผลชัดเจนจากมุมมองโค้ด
- database หรือระบบที่ latency-sensitive มากๆ (เช่น in-memory database) อาจต้องตั้งค่า **NUMA affinity/pinning** (เช่นด้วย `numactl`) เพื่อบังคับให้ process รันบน node เดียวกับ memory ที่ใช้
- Kubernetes มี "Topology Manager" ที่พยายามจัด pod ให้ CPU และ memory/device อยู่ NUMA node เดียวกันสำหรับ workload ที่ sensitive เรื่องนี้

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| Instruction cycle & pipeline | CPU ทำงานแบบ overlap หลายคำสั่ง แต่ branch/data hazard ทำให้สะดุดได้ |
| Clock speed vs IPC | Performance = Clock × IPC ไม่ใช่ GHz อย่างเดียว |
| Core vs SMT | SMT ไม่ใช่ core จริง เป็นการใช้ resource ว่างให้คุ้มขึ้นเท่านั้น |
| Cache & locality | ใช้ spatial/temporal locality ลด cache miss; ระวัง false sharing ในโค้ด concurrent |
| Bandwidth vs Latency | คนละมิติกัน ต้องรู้ว่างานของเราไวต่อมิติไหนมากกว่า |
| NUMA | Memory access ไม่ uniform บน multi-socket server ต้องคำนึงถึง locality ของ compute กับ data |

## คำถามทบทวน

1. ทำไมการเพิ่ม clock speed อย่างเดียวไม่ได้การันตีว่า CPU จะเร็วขึ้น?
2. อธิบาย false sharing ด้วยคำพูดตัวเอง และบอกวิธีแก้ 1 วิธี
3. ทำไม sequential memory access ถึงเร็วกว่า random access ทั้งที่ข้อมูลอยู่บน RAM เหมือนกัน?
4. เพราะเหตุใด container ที่ตั้ง CPU limit เป็น "2 vCPU" อาจไม่ได้ performance เท่ากับ 2 physical core เสมอไป?
5. NUMA ส่งผลต่อการออกแบบ high-performance service อย่างไร ยกตัวอย่างสถานการณ์ที่ต้องสนใจเรื่องนี้จริงจัง

---

ถัดไป: [บทที่ 2 — Operating System](02-operating-system.md)
