# บทที่ 2: Concurrency และ Context

Concurrency model ที่เรียบง่ายแต่ทรงพลังคือเหตุผลหลักที่ Go ถูกเลือกใช้เขียน infrastructure ระดับโลก (Kubernetes, Docker, Prometheus, etcd) บทนี้อธิบายตั้งแต่กลไกพื้นฐานที่สุด (goroutine คืออะไรจริงๆ) ไปจนถึง pattern และเครื่องมือควบคุม lifecycle ที่ใช้ในงาน production จริง

## 2.1 Goroutine vs OS Thread

### Goroutine คืออะไร

**Goroutine** คือ "user-level thread" ที่ Go runtime จัดการเอง ไม่ใช่ OS thread โดยตรง สร้างง่ายมากด้วยคีย์เวิร์ด `go`:

```go
go doSomething() // สร้าง goroutine ใหม่ ทำงานแบบ concurrent กับ goroutine ปัจจุบัน ไม่ block
```

จุดที่ต้องเข้าใจให้ลึก (เชื่อมกับ [Volume 1 — Operating System, หัวข้อ 2.2 Thread](../01-computer-foundation/02-operating-system.md#22-thread)): OS thread ถูกสร้างและ schedule โดย kernel โดยตรง (kernel-level thread, 1:1 model) มี stack เริ่มต้นขนาดใหญ่ (โดยทั่วไป 1-8 MB บน Linux) และการสร้าง/สลับ thread มี cost สูงเพราะต้องผ่าน syscall และ context switch ระดับ kernel

Goroutine ตรงกันข้าม:

- **Stack เริ่มต้นเล็กมาก** — ประมาณ 2 KB เท่านั้น และโตขึ้น/หดลงแบบ dynamic ตามที่ใช้จริง (segmented/copying stack)
- **สร้างและสลับได้เร็วกว่า OS thread มาก** — เพราะเป็นการจัดการใน user space ล้วนๆ ไม่ต้องผ่าน syscall ทุกครั้งที่สลับ
- **จำนวนที่สร้างได้** — โปรแกรม Go ทั่วไปสร้าง goroutine ได้เป็นแสนถึงล้านตัวโดยไม่ล่มระบบ ในขณะที่การสร้าง OS thread จำนวนเท่ากันจะทำให้ memory หมดและระบบค้างจาก context switch overhead ก่อนถึงตัวเลขนั้นมาก

### M:N Scheduling Model

Go runtime ใช้โมเดลที่เรียกว่า **M:N scheduling** — จับคู่ goroutine จำนวนมาก (M) ลงบน OS thread จำนวนจำกัด (N) โดย runtime scheduler เป็นคนตัดสินใจว่า ณ เวลาหนึ่ง goroutine ตัวไหนควรถูกรันบน OS thread ตัวไหน โมเดลนี้มีชื่อเรียกทางการว่า **GMP model**:

```
G = Goroutine (งานที่ต้องทำ)
M = Machine (OS thread จริง)
P = Processor (context ที่ถือ local run queue ของ goroutine, จำนวนเริ่มต้น = GOMAXPROCS)

┌─────────────────────────────────────────────┐
│              Go Runtime Scheduler             │
│                                               │
│   P0                P1                P2      │
│  ┌──────┐          ┌──────┐          ┌──────┐ │
│  │G G G │          │G G G │          │G G   │ │
│  │G G   │          │G     │          │G G G │ │
│  └──┬───┘          └──┬───┘          └──┬───┘ │
│     │                 │                 │     │
│     ▼                 ▼                 ▼     │
│    M0                M1                M2     │
│  (OS thread)      (OS thread)      (OS thread) │
└─────────────────────────────────────────────┘
```

- แต่ละ **P** ถือ local run queue ของ goroutine ที่พร้อมรัน และมีจำนวนเท่ากับ `GOMAXPROCS` (default = จำนวน logical CPU ของเครื่อง)
- แต่ละ **M** คือ OS thread จริง ต้องจับคู่กับ **P** ตัวหนึ่งก่อนถึงจะรัน goroutine ได้ (ดังนั้นจำนวน M ที่รัน Go code พร้อมกันจริงๆ ถูกจำกัดด้วยจำนวน P)
- เมื่อ goroutine หนึ่ง block บน syscall (เช่น อ่านไฟล์แบบ blocking) runtime จะแยก M นั้นออกจาก P แล้วสร้าง/ยืม M ใหม่มาทำงานกับ P ต่อ เพื่อไม่ให้ goroutine อื่นในคิวของ P นั้นต้องรอ — นี่คือเหตุผลที่โปรแกรม Go ยังคง responsive แม้มี goroutine จำนวนหนึ่งติด blocking I/O อยู่
- Goroutine ยังถูก **preempt** ได้ (ตั้งแต่ Go 1.14 เป็นต้นมา รองรับ asynchronous preemption แม้ใน loop ที่ไม่มี function call เลยก็ตาม) ทำให้ goroutine หนึ่งไม่สามารถ "แย่ง" CPU core ไปตลอดกาลจนตัวอื่นอดตายได้

> **เชื่อมโยง**: นี่คือการนำแนวคิด "user-level thread" และ "M:N scheduling" ที่อธิบายไว้ใน [Volume 1 — Operating System](../01-computer-foundation/02-operating-system.md) มาปรับใช้จริงในระดับภาษา Go runtime เองก็ต้องแก้ปัญหาเดียวกับที่ OS scheduler แก้ (fairness, load balancing ระหว่าง P, work stealing เมื่อ P หนึ่งว่างงานแต่ P อื่นมีงานค้าง) เพียงแต่ทำในระดับ user space ไม่ใช่ kernel space

## 2.2 Channel

**Channel** คือกลไกสื่อสารระหว่าง goroutine ที่ Go ออกแบบมาให้เป็นวิธีหลักในการแชร์ข้อมูล ปรัชญาของ Go สรุปเป็นประโยคที่มีชื่อเสียงว่า:

> "Do not communicate by sharing memory; instead, share memory by communicating."

แทนที่จะให้หลาย goroutine เข้าถึงตัวแปรเดียวกันแล้วต้องคอย lock เอง (แบบ thread ทั่วไป) Go สนับสนุนให้ "ส่งข้อมูลผ่าน channel" แทน — ownership ของข้อมูลถูกโอนไปพร้อมกับการส่ง ไม่ใช่แชร์กันตรงๆ

### Unbuffered vs Buffered Channel

```go
unbuffered := make(chan int)      // capacity 0
buffered := make(chan int, 3)     // capacity 3
```

- **Unbuffered channel** — การส่ง (`ch <- v`) จะ**บล็อก**จนกว่าจะมี goroutine อื่นมารับ (`<-ch`) พร้อมกันพอดี (synchronous handoff) เหมาะกับ pattern ที่ต้องการ "รู้แน่ชัด" ว่าอีกฝั่งได้รับข้อมูลแล้วจริงๆ ก่อนทำงานต่อ
- **Buffered channel** — ส่งได้โดยไม่บล็อกจนกว่า buffer จะเต็ม (asynchronous ในช่วงที่ buffer ยังมีที่ว่าง) เหมาะกับงานที่ต้องการ decouple ความเร็วของ producer กับ consumer ในระดับหนึ่ง

```go
func worker(id int, jobs <-chan int, results chan<- int) {
    for j := range jobs { // อ่านจนกว่า channel จะถูกปิดและว่างเปล่า
        results <- j * 2
    }
}
```

`<-chan int` (receive-only) และ `chan<- int` (send-only) คือ directional channel type — ใช้จำกัดสิทธิ์การใช้งาน channel ตอนส่งเป็น parameter เพื่อให้ compiler ช่วยป้องกัน misuse (เช่น worker ไม่ควร close channel ที่ตัวเองแค่รับข้อมูล)

### select

`select` คือกลไกรอหลาย channel operation พร้อมกัน คล้าย `switch` แต่ทำงานกับการสื่อสาร:

```go
select {
case msg := <-ch1:
    fmt.Println("จาก ch1:", msg)
case msg := <-ch2:
    fmt.Println("จาก ch2:", msg)
case <-time.After(2 * time.Second):
    fmt.Println("timeout รอเกิน 2 วินาที")
default:
    fmt.Println("ไม่มี channel ไหนพร้อมเลยตอนนี้ — ไม่บล็อก")
}
```

ถ้ามีหลาย case พร้อมกันในเวลาเดียวกัน `select` จะสุ่มเลือกหนึ่ง case แบบ uniform random (ป้องกันไม่ให้ case แรกได้เปรียบเสมอ) การมี `default` ทำให้ `select` ไม่บล็อกเลย (non-blocking check) ส่วนการไม่มี `default` ทำให้บล็อกจนกว่าจะมี case ใดพร้อม — pattern `case <-time.After(d)` เป็นวิธีมาตรฐานในการใส่ timeout ให้กับ channel operation ใดๆ

## 2.3 sync Package

Channel เหมาะกับการ "ส่งข้อมูล/สัญญาณ" ระหว่าง goroutine แต่บางสถานการณ์ก็ยังต้องพึ่ง primitive แบบ shared-memory synchronization ทั่วไป — Go มี package `sync` ให้:

```go
import "sync"

type SafeCounter struct {
    mu    sync.Mutex
    count map[string]int
}

func (c *SafeCounter) Inc(key string) {
    c.mu.Lock()
    defer c.mu.Unlock() // defer ให้ Unlock ทำงานแน่นอนแม้ panic ก็ตาม
    c.count[key]++
}

func (c *SafeCounter) Value(key string) int {
    c.mu.Lock()
    defer c.mu.Unlock()
    return c.count[key]
}
```

- **`sync.Mutex`** — mutual exclusion lock พื้นฐาน อนุญาตให้ goroutine เดียวเข้าถึง critical section ในเวลาหนึ่ง
- **`sync.RWMutex`** — แยก lock อ่าน (`RLock`/`RUnlock`) กับ lock เขียน (`Lock`/`Unlock`) หลาย goroutine อ่านพร้อมกันได้ (ถ้าไม่มีใครกำลังเขียน) แต่การเขียนต้องเป็น exclusive เสมอ เหมาะกับ workload ที่อ่านบ่อยกว่าเขียนมาก (เช่น cache ใน memory, config ที่ reload เป็นครั้งคราว)
- **`sync.WaitGroup`** — รอให้ชุด goroutine ทำงานจนครบก่อนไปต่อ

```go
var wg sync.WaitGroup
for i := 0; i < 5; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        fmt.Println("worker", id, "done")
    }(id) // ส่ง id เป็น parameter แทนการ capture ตัวแปร loop ตรงๆ — ป้องกัน closure bug คลาสสิก
}
wg.Wait() // บล็อกจนกว่า Done() จะถูกเรียกครบตามจำนวน Add()
```

> **กับดักคลาสสิก**: ก่อน Go 1.22 ตัวแปร loop (`i` หรือ `id` ในตัวอย่างข้างบนถ้าไม่ pass เป็น parameter) ถูกแชร์ข้ามทุก iteration ถ้า closure capture ตัวแปรนั้นตรงๆ โดยไม่ pass เป็น argument ทุก goroutine อาจเห็นค่าตัวสุดท้ายของ loop เหมือนกันหมด (Go 1.22 เปลี่ยน semantics ให้ loop variable เป็นตัวใหม่ทุก iteration แล้ว แต่โค้ดเก่าหรือ codebase ที่ pin เวอร์ชันต่ำกว่านั้นยังเจอปัญหานี้ได้ ควร pass เป็น parameter เข้า goroutine เสมอเพื่อความชัดเจนไม่ว่า Go version ไหน)

- **`sync.Once`** — รับประกันว่า function หนึ่งจะถูกเรียกแค่ **ครั้งเดียว** แม้จะถูกเรียกจากหลาย goroutine พร้อมกัน (เหมาะกับ lazy initialization แบบ thread-safe เช่น singleton, database connection pool)

```go
var (
    once sync.Once
    db   *sql.DB
)

func GetDB() *sql.DB {
    once.Do(func() {
        db, _ = sql.Open("postgres", dsn)
    })
    return db
}
```

## 2.4 Race Condition และ `-race`

**Race condition** เกิดเมื่อสอง goroutine (หรือมากกว่า) เข้าถึง memory เดียวกันพร้อมกัน โดยอย่างน้อยหนึ่งฝั่งเป็นการเขียน และไม่มี synchronization ควบคุม ผลลัพธ์ที่ได้ไม่แน่นอน (undefined behavior) ขึ้นกับจังหวะการ schedule ซึ่งเปลี่ยนไปทุกครั้งที่รัน

```go
counter := 0
var wg sync.WaitGroup
for i := 0; i < 1000; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        counter++ // race! อ่าน-บวก-เขียน ไม่ใช่ atomic operation เดียว
    }()
}
wg.Wait()
fmt.Println(counter) // มักได้ค่าน้อยกว่า 1000 และไม่คงที่ทุกครั้งที่รัน
```

`counter++` จริงๆ แล้วคือ 3 operation แยกกัน (read, add, write) ถ้าสอง goroutine อ่านค่าเดิมพร้อมกันก่อนที่อีกฝั่งจะเขียนค่าใหม่เสร็จ การเพิ่มค่าหนึ่งครั้งจะ "หายไป"

Go มี **race detector** ในตัว เปิดใช้ด้วย flag `-race`:

```bash
go run -race main.go
go test -race ./...
```

Race detector ทำงานโดย instrument โค้ดให้ track การเข้าถึง memory ของทุก goroutine แบบ runtime แล้วเปรียบเทียบ happens-before relationship (อิง [Vector clock algorithm](https://en.wikipedia.org/wiki/Vector_clock)) เพื่อจับ race ที่**เกิดขึ้นจริง**ระหว่างการรันนั้น (ไม่ใช่ static analysis ที่เดาจากโค้ดเฉยๆ) ข้อจำกัดคือ ตรวจจับได้เฉพาะ race ที่เกิดขึ้นจริงใน execution path ที่ถูกทดสอบเท่านั้น — ถ้า test coverage ไม่ครอบคลุม code path ที่มี race ก็ตรวจไม่เจอ นี่คือเหตุผลที่ `-race` ควรเปิดใช้เสมอใน CI pipeline ไม่ใช่แค่รันตอน dev เป็นครั้งคราว แม้จะมี performance overhead (ช้าลงและใช้ memory มากขึ้นหลายเท่า) ก็ตาม

## 2.5 Concurrency Pattern

### Worker Pool

จำกัดจำนวน goroutine ที่ทำงานพร้อมกัน แทนที่จะสร้าง goroutine ไม่จำกัดตามจำนวนงาน (ซึ่งเสี่ยง resource exhaustion ถ้างานมีจำนวนมากมาก):

```go
func workerPool(jobs <-chan int, results chan<- int, numWorkers int) {
    var wg sync.WaitGroup
    for w := 0; w < numWorkers; w++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for j := range jobs {
                results <- process(j)
            }
        }()
    }
    wg.Wait()
    close(results)
}
```

### Fan-out / Fan-in

**Fan-out** — กระจายงานจาก channel เดียวไปให้หลาย goroutine ประมวลผลพร้อมกัน; **Fan-in** — รวมผลลัพธ์จากหลาย channel กลับมาเป็น channel เดียว:

```go
func fanOut(in <-chan int, n int) []<-chan int {
    outs := make([]<-chan int, n)
    for i := 0; i < n; i++ {
        out := make(chan int)
        go func() {
            defer close(out)
            for v := range in {
                out <- process(v)
            }
        }()
        outs[i] = out
    }
    return outs
}

func fanIn(channels ...<-chan int) <-chan int {
    merged := make(chan int)
    var wg sync.WaitGroup
    wg.Add(len(channels))
    for _, ch := range channels {
        go func(c <-chan int) {
            defer wg.Done()
            for v := range c {
                merged <- v
            }
        }(ch)
    }
    go func() {
        wg.Wait()
        close(merged) // close หลังทุก goroutine ส่งครบแล้วเท่านั้น
    }()
    return merged
}
```

### Pipeline

เชื่อมหลาย stage เข้าด้วยกัน แต่ละ stage รับ input จาก channel และส่ง output เป็น channel ให้ stage ถัดไป:

```go
func generate(nums ...int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for _, n := range nums {
            out <- n
        }
    }()
    return out
}

func square(in <-chan int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for n := range in {
            out <- n * n
        }
    }()
    return out
}

// ใช้งาน: generate -> square -> square (pipe เข้าด้วยกัน)
for v := range square(square(generate(1, 2, 3))) {
    fmt.Println(v) // 1, 16, 81
}
```

หลักการสำคัญของทุก pattern ข้างต้น: **stage ที่เป็นคน `send` เข้า channel ควรเป็นคน `close` channel นั้นเสมอ** ไม่ใช่ฝั่งที่รับ (ป้องกัน panic จากการ close channel ซ้ำ หรือ send เข้า channel ที่ปิดไปแล้ว) และควรระวังเรื่อง **goroutine leak** — ถ้า consumer หยุดอ่านก่อนที่ producer จะส่งครบ (เช่น เจอ error แล้ว return ก่อน) producer จะติดบล็อกอยู่ที่ `out <- v` ตลอดไปเพราะไม่มีใครมารับอีก ทำให้ goroutine นั้นไม่มีวันจบและ memory รั่วไปเรื่อยๆ — นี่คือหนึ่งในเหตุผลหลักที่ `context.Context` (หัวข้อถัดไป) ถูกออกแบบให้ใช้เป็นกลไก cancellation มาตรฐานคู่กับทุก pattern เหล่านี้

## 2.6 context.Context

### Context ใช้ทำอะไร

`context.Context` คือกลไกมาตรฐานของ Go สำหรับส่งต่อ 3 อย่างข้าม call chain ของ goroutine:

1. **Cancellation signal** — บอกให้ operation ที่กำลังทำอยู่ "หยุด" เพราะไม่ต้องการผลลัพธ์แล้ว (เช่น client ตัดการเชื่อมต่อไปแล้ว)
2. **Deadline/timeout** — จำกัดเวลาที่ operation หนึ่งควรทำงานได้นานสุด
3. **Request-scoped value** — ข้อมูลที่ผูกกับ request หนึ่งๆ (เช่น request ID, trace ID สำหรับ [Volume 14 — Observability](../14-observability/README.md))

ปัญหาที่ `context` แก้: ในระบบที่ HTTP handler เรียก use case เรียก repository เรียก database driver เป็น call chain ลึกหลายชั้น ถ้า client ยกเลิก request กลางทาง (ปิด browser, timeout ฝั่ง client) โค้ดชั้นล่างสุด (เช่น query ที่กำลังรันอยู่ใน database) ควรจะรู้และหยุดทำงานทันที ไม่ใช่ทำงานต่อจนจบแล้วโยนผลลัพธ์ทิ้งเฉยๆ ซึ่งเปลืองทรัพยากรฟรี — `context` คือ "สาย" ที่ร้อยทุกชั้นเข้าด้วยกันให้สัญญาณนี้เดินทางผ่านได้

### WithCancel, WithTimeout, WithDeadline, WithValue

```go
// WithCancel — cancel เอง manual เมื่อไหร่ก็ได้
ctx, cancel := context.WithCancel(context.Background())
defer cancel() // เรียกเสมอเพื่อ release resource แม้ operation จะสำเร็จแล้วก็ตาม

// WithTimeout — cancel อัตโนมัติเมื่อครบเวลาที่กำหนด
ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
defer cancel()

// WithDeadline — เหมือน WithTimeout แต่ระบุเป็นเวลา absolute แทน duration
deadline := time.Now().Add(5 * time.Second)
ctx, cancel := context.WithDeadline(context.Background(), deadline)
defer cancel()

// WithValue — แนบข้อมูล request-scoped (ใช้อย่างระมัดระวัง ดูคำเตือนด้านล่าง)
ctx := context.WithValue(context.Background(), requestIDKey{}, "req-123")
```

ทุก `With*` function คืน `ctx` ใหม่ที่เป็น **child** ของ context เดิม และ `cancel` function ที่ต้องเรียกเสมอ (แม้ operation จะเสร็จก่อนครบ timeout ก็ตาม) เพื่อ release resource ภายใน — นี่คือเหตุผลที่ `defer cancel()` อยู่คู่กับ `context.With*` แทบทุกครั้งในโค้ด Go จริง

**Context ทำงานเป็น tree**: ถ้า parent context ถูก cancel (หรือ timeout) child context ทุกตัวที่ derive มาจากมันจะถูก cancel ตามไปด้วยโดยอัตโนมัติ — นี่คือกลไกที่ทำให้ cancel signal เดียว "กระจาย" ไปยังทุก goroutine ลูกที่เกี่ยวข้องได้พร้อมกัน

```
context.Background()
   └── ctx1 (WithTimeout 5s)
          └── ctx2 (WithCancel)     ← cancel ctx1 จะทำให้ ctx2 cancel ไปด้วย
                 └── ctx3 (WithValue) ← cancel ตามไปด้วยเช่นกัน
```

> **คำเตือนเรื่อง WithValue**: ใช้เก็บเฉพาะข้อมูลที่เป็น request-scoped metadata จริงๆ (request ID, trace ID, authenticated user ที่แนบมากับทุก layer) **ห้าม**ใช้เป็นช่องทางส่ง optional parameter หรือ dependency (เช่น database connection, config) เข้าฟังก์ชันแทนการส่งเป็น argument ตรงๆ เพราะทำให้ signature ของฟังก์ชันโกหก (ดูจากภายนอกไม่รู้เลยว่าฟังก์ชันต้องการอะไรบ้าง) และเสีย type safety ตอน compile time ไปโดยสิ้นเชิง (ต้อง type assertion ทุกครั้งที่ดึงค่ากลับมา)

### Convention: ctx เป็น argument แรกเสมอ

Go convention (บันทึกไว้เป็นทางการใน [package context doc](https://pkg.go.dev/context)) กำหนดให้ `ctx context.Context` เป็น **parameter แรก** ของทุกฟังก์ชันที่อาจต้องการ cancellation/timeout/value ตลอด call chain:

```go
func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context() // ทุก HTTP request ของ net/http มี context ผูกมากับตัวมันเองแล้ว

    order, err := h.orderUseCase.GetOrder(ctx, orderID)
    if err != nil {
        http.Error(w, err.Error(), statusFromError(err))
        return
    }
    json.NewEncoder(w).Encode(order)
}

func (uc *OrderUseCase) GetOrder(ctx context.Context, id string) (*Order, error) {
    return uc.orderRepo.FindByID(ctx, id) // ส่ง ctx ต่อไปเรื่อยๆ ไม่มีวันหยุดกลางทาง
}

func (r *PostgresOrderRepo) FindByID(ctx context.Context, id string) (*Order, error) {
    row := r.db.QueryRowContext(ctx, "SELECT * FROM orders WHERE id = $1", id)
    // QueryRowContext ผูก query กับ ctx โดยตรง — ถ้า ctx ถูก cancel ระหว่าง query
    // driver จะพยายามยกเลิก query ที่กำลังรันอยู่ใน database ทันที ไม่ต้องรอจนจบ
    // ...
}
```

ทุกชั้น (`handler → use case → repository → db driver`) รับ `ctx` เป็น argument แรกและส่งต่อไปเรื่อยๆ ไม่มีชั้นไหน "สร้าง context ใหม่จาก Background()" กลางทางโดยไม่มีเหตุผล เพราะจะทำให้สาย cancellation ขาดตอน — ถ้า client ยกเลิก HTTP request (r.Context() ถูก cancel โดย `net/http` เอง) สัญญาณนั้นจะไหลผ่านทุกชั้นไปจนถึง database driver และ query ที่กำลังรันอยู่จริงจะถูกยกเลิกจริง ไม่ใช่แค่ handler เลิกรอเฉยๆ ในขณะที่ query เบื้องหลังยังกินทรัพยากร database ต่อไปโดยไม่มีใครรอผล — นี่คือคุณค่าที่แท้จริงของการทำ context propagation ให้ครบทุกชั้นอย่างมีวินัย

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| Goroutine | User-level thread ที่ Go runtime จัดการเอง, M:N scheduling (GMP model), เบากว่า OS thread มาก |
| Channel | สื่อสารระหว่าง goroutine แทนการแชร์ memory ตรงๆ, unbuffered = synchronous handoff, buffered = async จนกว่าจะเต็ม |
| sync package | Mutex/RWMutex สำหรับ critical section, WaitGroup สำหรับรอ goroutine กลุ่มหนึ่งจบ, Once สำหรับ lazy init ที่ thread-safe |
| Race condition | เกิดจากเข้าถึง memory เดียวกันพร้อมกันโดยไม่ sync, ตรวจจับด้วย `-race` เสมอใน CI |
| Concurrency pattern | Worker pool จำกัดจำนวน goroutine, fan-out/fan-in กระจาย-รวมงาน, pipeline เชื่อม stage ผ่าน channel |
| context.Context | ส่ง cancellation/deadline/value ข้าม call chain, เป็น tree ที่ parent cancel แล้ว child cancel ตาม, ต้องเป็น argument แรกเสมอ |

## คำถามทบทวน

1. อธิบาย GMP model ของ Go scheduler และเชื่อมโยงว่ามันสัมพันธ์กับแนวคิด M:N scheduling ใน [Volume 1](../01-computer-foundation/02-operating-system.md) อย่างไร
2. Unbuffered channel กับ buffered channel ต่างกันอย่างไรในแง่ blocking behavior ยกตัวอย่างสถานการณ์ที่ควรเลือกแบบไหน
3. ทำไม `counter++` ถึงไม่ปลอดภัยเมื่อมีหลาย goroutine เรียกพร้อมกัน ทั้งที่ดูเป็นคำสั่งเดียว?
4. ในกลไก fan-in อธิบายว่าทำไมต้องรอ `wg.Wait()` เสร็จก่อนถึงจะ `close(merged)` ได้ ถ้า close ก่อนจะเกิดอะไรขึ้น?
5. อธิบายว่าทำไม `context.WithValue` ไม่ควรใช้แทนการส่ง dependency ปกติเป็น argument พร้อมยกตัวอย่างที่ใช้ผิดวิธี

---

ก่อนหน้า: [บทที่ 1 — Language Fundamentals](01-language-fundamentals.md) | ถัดไป: [บทที่ 3 — Testing](03-testing.md)
