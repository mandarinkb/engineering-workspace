# บทที่ 2: Operating System

## 2.1 Process

**Process** คือโปรแกรมที่กำลังทำงานอยู่ (a program in execution) แต่ละ process มี address space เป็นของตัวเอง แยกขาดจาก process อื่นโดยสมบูรณ์ (จัดการผ่าน virtual memory — ดูหัวข้อ 2.3)

### Process Control Block (PCB)

OS เก็บข้อมูลของแต่ละ process ไว้ในโครงสร้างที่เรียกว่า PCB ประกอบด้วย:

- Process ID (PID) และ Parent Process ID (PPID)
- Process state (ดูด้านล่าง)
- Program Counter, register values (สำหรับ context switch)
- Memory management info (page table pointer)
- Open file descriptor list
- Scheduling info (priority, nice value)

### Address Space Layout

Process หนึ่งตัวมี virtual address space ที่แบ่งเป็นส่วนต่างๆ โดยทั่วไป (บน Linux):

```
High address
┌─────────────────┐
│      Stack       │  ← โตลง (function call, local variable)
│        ↓         │
│                   │
│        ↑         │
│      Heap        │  ← โตขึ้น (dynamic allocation)
├─────────────────┤
│   BSS (uninit)   │  ← global variable ที่ยังไม่ initialize
│  Data (init)     │  ← global variable ที่ initialize แล้ว
│      Text        │  ← โค้ดโปรแกรม (read-only)
└─────────────────┘
Low address
```

### Process States

```
        create
          │
          ▼
        ┌────┐   dispatch    ┌─────────┐
        │ New├───────────────►│ Running │
        └────┘                └────┬────┘
                                    │ I/O wait / event wait
                     interrupt/    ▼
                     time slice  ┌──────────┐
        ┌─────────┐◄────────────┤          │
        │  Ready  │              │ Waiting  │
        └────┬────┘◄─────────────┤ (Blocked)│
             │      I/O complete └──────────┘
             │ dispatch
             ▼
        ┌──────────┐
        │Terminated│
        └──────────┘
```

- **New** — เพิ่งถูกสร้าง ยังไม่พร้อมรัน
- **Ready** — พร้อมรัน รอ scheduler เลือก
- **Running** — กำลังถูกประมวลผลอยู่บน CPU จริง
- **Waiting/Blocked** — รออะไรบางอย่าง (I/O, lock, signal) ยังใช้ CPU ไม่ได้แม้ CPU จะว่าง
- **Terminated** — จบการทำงานแล้ว

### fork/exec

Linux สร้าง process ใหม่ด้วยแนวคิด **fork** (copy process ปัจจุบันเป็น process ลูก — ใช้ copy-on-write เพื่อไม่ต้อง copy memory จริงทันที) ตามด้วย **exec** (แทนที่ image ของ process ลูกด้วยโปรแกรมใหม่) นี่คือกลไกเบื้องหลังของทุกครั้งที่ shell รันคำสั่งใหม่

### Inter-Process Communication (IPC)

เพราะแต่ละ process แยก address space กันอย่างเด็ดขาด การสื่อสารข้าม process ต้องผ่านกลไกที่ OS จัดให้ เช่น pipe, message queue, shared memory, socket (unix domain socket หรือ network socket) — แนวคิดเดียวกับที่ container ใช้สื่อสารกันผ่าน network namespace

## 2.2 Thread

**Thread** คือหน่วยของการประมวลผลที่เล็กกว่า process หลาย thread ภายใน process เดียวกัน**แชร์** address space, heap, และ file descriptor เดียวกัน แต่แต่ละ thread มี **stack และ register ของตัวเอง**

```
┌─────────────────────────────────┐
│           Process                │
│  ┌─────────┐  shared: Code,     │
│  │ Thread 1│  Heap, Data,        │
│  │ (stack) │  File Descriptors   │
│  ├─────────┤                     │
│  │ Thread 2│                     │
│  │ (stack) │                     │
│  └─────────┘                     │
└─────────────────────────────────┘
```

### ทำไม Thread ถึง "ถูก" กว่า Process

- สร้าง thread เร็วกว่าสร้าง process มาก (ไม่ต้อง copy address space)
- สลับ (context switch) ระหว่าง thread ในกระบวนการเดียวกันถูกกว่า สลับระหว่าง process เพราะไม่ต้องเปลี่ยน page table/flush TLB
- แต่ก็มีข้อเสีย: เพราะแชร์ memory กัน จึงต้องระวัง race condition และ synchronization ให้ดี ต่างจาก process ที่แยกกันโดยธรรมชาติ

### User-level vs Kernel-level Thread

- **Kernel-level thread** — kernel รู้จักและ schedule เอง (1:1 model) เช่น POSIX thread (pthread) มาตรฐานบน Linux
- **User-level thread** — library ใน user space จัดการเอง kernel มองเห็นแค่ process เดียว (M:1 หรือ M:N model) เร็วกว่าในการสร้าง/สลับ แต่ถ้า thread หนึ่งบล็อกที่ syscall ทั้ง process อาจค้างได้ถ้า implementation ไม่ดี

> **เชื่อมโยงกับ Go**: goroutine ใน [Volume 4 — Golang](../04-golang/README.md) คือ user-level thread ที่ Go runtime จัดการเอง (M:N scheduling model — mapping goroutine จำนวนมากลงบน OS thread จำนวนจำกัด) นี่คือเหตุผลที่สร้าง goroutine หลักแสนตัวได้โดยไม่ล่ม ในขณะที่สร้าง OS thread จำนวนเท่ากันจะทำให้ระบบพังเพราะแต่ละ OS thread ใช้ memory และ context switch cost สูงกว่ามาก

## 2.3 Virtual Memory

### ทำไมต้องมี Virtual Memory

1. **Isolation** — แต่ละ process เห็น address space เป็นของตัวเองทั้งหมด ไม่เห็น/แก้ memory ของ process อื่นได้โดยตรง
2. **Abstraction** — โปรแกรมไม่ต้องรู้ว่า physical memory เหลือเท่าไหร่ หรือ layout จริงเป็นอย่างไร
3. **การใช้ memory เกินกว่า RAM จริง** — ผ่านกลไก swap

### Paging

Virtual address space ถูกแบ่งเป็นบล็อกขนาดคงที่เรียกว่า **page** (ทั่วไป 4KB) และ physical memory แบ่งเป็น **frame** ขนาดเท่ากัน **Page table** คือตารางที่ map virtual page ไปยัง physical frame

```
Virtual Address → [Page Table] → Physical Address
     Page 0      →   Frame 5
     Page 1      →   Frame 2
     Page 2      →   (not present, on disk / not yet allocated)
```

### TLB (Translation Lookaside Buffer)

การแปล virtual→physical address ทุกครั้งผ่าน page table (ซึ่งอยู่ใน memory) จะช้ามาก จึงมี **TLB** เป็น cache พิเศษใน CPU ที่เก็บผลการแปลที่ใช้บ่อยๆ ไว้ **TLB miss** ทำให้ต้องเดิน page table จริง (page table walk) ซึ่งช้ากว่ามาก — เป็นอีกเหตุผลที่ locality ของการเข้าถึง memory (บทที่ 1) ส่งผลต่อ performance จริง

### Page Fault

เมื่อ CPU เข้าถึง virtual address ที่ page table บอกว่า "ไม่ present" จะเกิด **page fault** ซึ่ง OS จะจัดการ 2 กรณีหลัก:

- **Minor fault** — page มีอยู่ใน memory จริงแล้ว (เช่นถูก share กับ process อื่น หรือเพิ่งถูก allocate) แค่ต้องอัปเดต page table — เร็ว
- **Major fault** — ต้องไปอ่านข้อมูลจาก disk (swap หรือ memory-mapped file) เข้ามาใน RAM ก่อน — ช้ามากเทียบกับความเร็ว RAM

### Swap

เมื่อ RAM ไม่พอ OS จะย้ายหน้า memory ที่ไม่ค่อยถูกใช้ไปเก็บใน disk (swap space) ชั่วคราว แล้วดึงกลับมาเมื่อถูกเข้าถึงอีกครั้ง (ผ่าน major page fault) — ถ้าระบบ swap บ่อยเกินไป (thrashing) performance จะตกฮวบเพราะ disk ช้ากว่า RAM มาก

### เชื่อมโยงกับ Container

Container memory limit (เช่นใน Docker/Kubernetes — [Volume 7](../07-docker/README.md), [Volume 8](../08-kubernetes/README.md)) ทำงานผ่าน Linux cgroups ซึ่งควบคุมปริมาณ physical memory (RSS) ที่ container ใช้ได้ ถ้า process ใน container พยายามใช้ memory เกิน limit และไม่มี swap ให้ใช้ kernel จะสั่ง **OOM Kill** ทันที — เข้าใจ virtual memory และ page fault ช่วยให้ debug ปัญหา "container ถูก OOM kill ทั้งที่โค้ดดูไม่น่ากินแรม" ได้ตรงจุดขึ้นมาก

## 2.4 Scheduling

### ทำไมต้อง Schedule

CPU core มีจำนวนจำกัด แต่ process/thread ที่พร้อมรัน (ready) มักมีมากกว่า CPU scheduler คือส่วนของ OS ที่ตัดสินใจว่า ณ เวลาหนึ่ง จะให้ CPU core ไหนรัน process/thread ตัวไหน

### Scheduling Algorithm พื้นฐาน

- **FCFS (First-Come-First-Served)** — เรียงคิวตามลำดับมาถึง ง่ายแต่ process ที่ใช้เวลานานจะบล็อก process อื่น (convoy effect)
- **Round Robin** — แต่ละ process ได้ CPU time เท่ากันเป็นช่วงๆ (time quantum) แล้วสลับกันไป
- **Priority Scheduling** — process ที่ priority สูงกว่าได้รันก่อน (เสี่ยง starvation ถ้า priority ต่ำไม่เคยได้รันเลย)
- **CFS (Completely Fair Scheduler)** — scheduler มาตรฐานของ Linux (default สำหรับ process ทั่วไป) พยายามแบ่ง CPU time ให้ "แฟร์" ที่สุดโดยใช้แนวคิด **virtual runtime**: process ไหนได้ใช้ CPU สะสมน้อยกว่า จะถูกเลือกให้รันก่อน ทำให้ในระยะยาวทุก process ได้ CPU share ใกล้เคียงกันตาม weight/priority ของมัน

### Nice Value และ Scheduling Class

- **Nice value** (-20 ถึง 19) กำหนด priority ของ process ทั่วไปภายใต้ CFS ค่า nice ต่ำ (ติดลบ) = priority สูง = ได้ CPU share มากกว่า
- Linux มี scheduling class อื่นสำหรับงานพิเศษ เช่น `SCHED_FIFO`/`SCHED_RR` สำหรับงาน real-time ที่ต้องการ latency ต่ำและคาดเดาได้ ซึ่งจะได้ priority เหนือกว่า process ทั่วไปใน CFS เสมอ

### Context Switch

การสลับจาก process/thread หนึ่งไปอีกตัวหนึ่งเรียกว่า **context switch** ซึ่งมี cost จริง ไม่ใช่ฟรี:

1. บันทึก register/state ของ process เดิมลง PCB
2. โหลด register/state ของ process ใหม่จาก PCB
3. ถ้าเป็นการสลับข้าม process (ไม่ใช่ thread ในกลุ่มเดียวกัน) ต้องเปลี่ยน page table ด้วย ซึ่งทำให้ TLB ที่สะสมไว้ (บทที่ 2.3) invalid ไปด้วย ต้องเริ่มสะสมใหม่ (**TLB flush**) — นี่คือเหตุผลหลักที่ context switch ข้าม process แพงกว่าสลับ thread ภายใน process เดียวกัน

context switch ที่ถี่เกินไป (เช่น มี thread จำนวนมากแย่งกันรันบน core น้อย) ทำให้เวลาส่วนใหญ่ถูกใช้ไปกับการสลับแทนที่จะเป็นงานจริง — เป็นสาเหตุคลาสสิกของปัญหา "CPU usage สูงแต่ throughput ไม่ขึ้น"

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| Process | หน่วยแยก isolation ระดับ address space, มี state ชัดเจน (new/ready/running/waiting/terminated) |
| Thread | แชร์ address space กันภายใน process เดียว ถูกกว่า process แต่ต้อง sync เอง |
| Virtual Memory | ทุก process เห็น address space แยกกัน ผ่าน page table + TLB, page fault เกิดเมื่อ page ไม่ present |
| Swap / OOM | RAM เต็ม → swap (ช้า) หรือถ้าอยู่ใน container/cgroup ที่ไม่มี swap → OOM kill |
| Scheduling | CFS ใช้ virtual runtime แบ่ง CPU อย่างแฟร์ตาม nice/priority, context switch มี cost จริงโดยเฉพาะข้าม process |

## คำถามทบทวน

1. อธิบายความแตกต่างระหว่าง process กับ thread โดยเน้นว่า "แชร์" อะไรกันบ้างและ "ไม่แชร์" อะไรบ้าง
2. Minor page fault กับ Major page fault ต่างกันอย่างไร แบบไหนกระทบ performance มากกว่า เพราะเหตุใด
3. ทำไม container ถึงถูก OOM kill ได้ทั้งที่ host เครื่องยังมี RAM เหลือเยอะ?
4. เพราะเหตุใด context switch ระหว่าง 2 thread ในกระบวนการเดียวกันถึงถูกกว่า context switch ระหว่าง 2 process?
5. Goroutine ใน Go เกี่ยวข้องกับแนวคิด user-level thread และ M:N scheduling อย่างไร ทำไมถึงสร้างได้เป็นแสนตัวโดยไม่ล่มระบบ?

---

ก่อนหน้า: [บทที่ 1 — Computer Architecture](01-computer-architecture.md) | กลับสู่ [สารบัญ Volume 1](README.md)
