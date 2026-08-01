# บทที่ 1: Language Fundamentals

เนื้อหาบทนี้ไม่ใช่ tutorial สอน syntax เบื้องต้น (สมมติว่าอ่าน `for`, `if`, `func` ออกอยู่แล้ว) แต่จะเจาะจุดที่ Go **ต่างจากภาษาอื่นอย่างมีนัยสำคัญ** และเป็นจุดที่ backend engineer ระดับ senior ต้องเข้าใจลึกจริง ไม่ใช่แค่ใช้งานได้ เพราะความเข้าใจผิดในเรื่องเหล่านี้ (โดยเฉพาะ pointer/value receiver และ slice internals) คือต้นเหตุของ bug ที่ debug ยากที่สุดในโค้ด Go จำนวนมาก

## 1.1 Type System และ Struct

Go เป็นภาษา **statically typed** และ **strongly typed** — type ถูกตรวจสอบตอน compile time และไม่มีการแปลง type แบบ implicit เหมือน JavaScript หรือ Python (แม้แต่ `int32` กับ `int64` ก็บวกกันตรงๆ ไม่ได้ ต้อง cast ชัดเจน) ปรัชญานี้แลกความสะดวกบางส่วนเพื่อความชัดเจนและลด bug ที่มาจากการแปลง type แบบเงียบๆ

**Struct** คือหน่วยพื้นฐานในการจัดกลุ่มข้อมูลใน Go (ไม่มี `class`):

```go
type User struct {
    ID        int64
    Name      string
    CreatedAt time.Time
}

// Struct literal
u := User{ID: 1, Name: "Somchai", CreatedAt: time.Now()}
```

จุดสำคัญที่ต่างจากภาษา OOP ทั่วไป: Go **ไม่มี inheritance** (class A extends class B) มีแต่ **composition ผ่าน embedding**:

```go
type Base struct {
    ID        int64
    CreatedAt time.Time
}

type User struct {
    Base        // embedded field — ไม่มีชื่อ field ตรงๆ
    Name string
}

u := User{Base: Base{ID: 1}, Name: "Somchai"}
fmt.Println(u.ID)   // เข้าถึง field ของ Base ได้ตรงๆ เหมือน "promote" ขึ้นมา
```

`embedding` ให้ความสะดวกคล้าย inheritance (เข้าถึง field/method ของ struct ที่ฝังได้ตรงๆ) แต่**ไม่ใช่** inheritance จริง — ไม่มี polymorphism แบบ virtual method, ไม่มี `is-a` relationship ที่ compiler บังคับ `User` ไม่ใช่ `Base` ในทาง type system (แปลง type ตรงๆ ไม่ได้) มันเป็นแค่ syntactic sugar ที่ทำให้เขียน `u.ID` แทน `u.Base.ID` ได้ นี่คือเหตุผลที่นักออกแบบ Go เลือกคำว่า "composition over inheritance" เป็นหลักการหลักของภาษา

## 1.2 Interface — Implicit Implementation

นี่คือฟีเจอร์ที่เป็นเอกลักษณ์ของ Go มากที่สุดตัวหนึ่ง และเป็นรากฐานของสถาปัตยกรรมทั้งหมดใน [บทที่ 4](04-architecture-patterns.md)

ในภาษาอย่าง Java หรือ C# การจะบอกว่า class หนึ่ง implement interface หนึ่ง ต้องประกาศชัดเจนด้วย `implements`/`:` แต่ Go ใช้ **structural typing** — type ใดก็ตามที่มี method ครบตามที่ interface กำหนด จะถือว่า implement interface นั้น**โดยอัตโนมัติ** โดยไม่ต้องเขียนอะไรบอกเลย:

```go
// ประกาศที่ไหนก็ได้ ไม่ต้องอยู่ package เดียวกับ implementation
type Writer interface {
    Write(p []byte) (n int, err error)
}

type FileLogger struct{ /* ... */ }

// ไม่มีการเขียน "FileLogger implements Writer" ที่ไหนเลย
func (f *FileLogger) Write(p []byte) (n int, err error) {
    // ...
    return len(p), nil
}

var w Writer = &FileLogger{} // compile ผ่าน เพราะ *FileLogger มี method Write ตรง signature
```

### ทำไมเรื่องนี้ถึงสำคัญมาก

1. **Decoupling ตามธรรมชาติ** — package ที่นิยาม interface ไม่จำเป็นต้องรู้จักหรือ import package ที่ implement เลย ทิศทางของ dependency จึงเป็นไปตามที่ผู้ออกแบบต้องการ ไม่ใช่ตามลำดับที่เขียนโค้ด — นี่คือกลไกที่ทำให้ Dependency Inversion Principle (หัวใจของ Clean Architecture และ Hexagonal Architecture ใน [บทที่ 4](04-architecture-patterns.md)) ทำได้ง่ายและเป็นธรรมชาติมากใน Go โดยไม่ต้องพึ่ง framework หรือ annotation ใดๆ
2. **Interface เล็กและถูกนิยามฝั่งผู้ใช้ (consumer)** — idiom ของ Go คือ "accept interfaces, return structs" และนิยม interface ที่มี 1-2 method เท่านั้น (เช่น `io.Reader`, `io.Writer`) เพราะ interface ยิ่งเล็ก ยิ่งมี type ที่ implement ได้ง่ายและหลากหลาย ต่างจากภาษา OOP ดั้งเดิมที่มักออกแบบ interface ใหญ่ๆ ไว้ล่วงหน้า
3. **Testability** — เพราะ implicit implementation ทำให้เขียน mock/stub ที่ implement interface เดียวกันได้ง่ายมาก โดยไม่ต้องแก้โค้ด production เลย (ดู [บทที่ 3](03-testing.md))

### Interface ว่าง และ Type Assertion

`interface{}` (หรือ `any` ตั้งแต่ Go 1.18) คือ interface ที่ไม่มี method เลย ทุก type จึง implement มันได้เสมอ ใช้เป็น "รับได้ทุกอย่าง" แต่แลกมาด้วยการเสีย type safety ตอน compile time — ต้องใช้ **type assertion** หรือ **type switch** เพื่อดึง concrete type กลับมาตอน runtime:

```go
func describe(v any) string {
    switch x := v.(type) {
    case int:
        return fmt.Sprintf("int: %d", x)
    case string:
        return fmt.Sprintf("string: %s", x)
    default:
        return "unknown"
    }
}
```

ใช้ `any`/`interface{}` ให้น้อยที่สุดเท่าที่จำเป็น เพราะทุกครั้งที่ใช้ คือการยอมสละ compile-time safety ที่เป็นจุดแข็งหลักของภาษาไปแลกกับความยืดหยุ่น — ส่วนมากปัญหาแบบนี้แก้ได้ดีกว่าด้วย **generics** (ดูหัวข้อ 1.7)

## 1.3 Pointer vs Value Receiver

Go ส่ง argument แบบ **pass by value เสมอ** (แม้แต่ pointer เองก็ถูก copy ค่า address ไปตอนส่ง) เรื่องนี้กระทบโดยตรงกับการเลือก receiver ของ method

```go
type Counter struct {
    count int
}

// Value receiver — รับ "สำเนา" ของ Counter มาทำงาน
func (c Counter) IncrementWrong() {
    c.count++ // แก้ไขสำเนา ไม่ใช่ตัวจริง
}

// Pointer receiver — รับ address ของ Counter ตัวจริงมาทำงาน
func (c *Counter) Increment() {
    c.count++ // แก้ไขตัวจริงผ่าน pointer
}
```

### ตัวอย่าง bug จริงจากการใช้ value receiver ผิดที่

```go
func main() {
    c := Counter{count: 0}
    for i := 0; i < 5; i++ {
        c.IncrementWrong()
    }
    fmt.Println(c.count) // พิมพ์ 0 ไม่ใช่ 5!
}
```

`IncrementWrong` รับ `Counter` เป็น value receiver โค้ดข้างในทำงานกับ **สำเนา** ของ `c` เท่านั้น การ `c.count++` ข้างในจึงไม่มีผลอะไรกับ `c` ตัวจริงใน `main` เลย นี่คือ bug คลาสสิกที่ compile ผ่านปกติ ไม่มี warning ใดๆ (เพราะ syntax ถูกต้องทั้งหมด) และมักถูกจับได้ตอน runtime behavior ผิดเท่านั้น — เป็นเหตุผลสำคัญที่ทำให้เรื่องนี้ต้อง "เข้าใจจริง" ไม่ใช่จำ pattern ไปใช้เฉยๆ

### กฎการเลือก receiver

| ใช้ pointer receiver เมื่อ | ใช้ value receiver เมื่อ |
|---|---|
| Method ต้องแก้ไข field ของ struct | Struct เป็น immutable โดยธรรมชาติ (เช่น value object อย่าง `Money`, `Point`) |
| Struct มีขนาดใหญ่ (copy แพง) | Struct เล็กมาก (เช่น มีแค่ 1-2 field แบบ primitive) |
| Struct มี field ที่เป็น mutex, slice ที่ต้องจัดการ state ภายใน | ไม่มี state ที่ต้อง mutate เลย |
| Type อื่นๆ ใน struct เดียวกันมี method เป็น pointer receiver แล้ว (ต้อง**สม่ำเสมอ**) | ต้องการ semantics แบบ "ส่งสำเนาไปใช้ได้อย่างปลอดภัย" |

> **กฎที่ต้องจำ**: ถ้า struct หนึ่งมี method ไหนก็ตามที่เป็น pointer receiver ควรทำให้**ทุก** method ของ struct นั้นเป็น pointer receiver ให้หมด เพื่อความสม่ำเสมอ (Go team เองแนะนำแบบนี้ใน [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments#receiver-type)) เพราะการผสมกันทำให้พฤติกรรมเรื่อง "แก้ตัวจริงหรือสำเนา" ไม่สอดคล้องกันในตัว type เดียวและงงเวลาอ่านโค้ด

อีกจุดสำคัญ: interface จะ satisfy ได้จาก pointer receiver ก็ต่อเมื่อใช้ pointer type เท่านั้น — ถ้า `Increment()` เป็น pointer receiver แล้วมี interface `Incrementer` ที่ต้องการ method นี้ ตัวแปรที่จะสอดคล้อง interface ได้ต้องเป็น `*Counter` ไม่ใช่ `Counter` เฉยๆ (Go compiler จะฟ้อง `does not implement` ถ้าใช้ value type)

## 1.4 Error Handling แบบ Go

### ทำไม Go ไม่มี exception

ภาษาส่วนใหญ่ใช้ `try/catch/throw` จัดการ error ทำให้ control flow "กระโดด" ข้าม stack ได้แบบไม่เห็นชัดจากการอ่านโค้ดตรงๆ — ผู้เขียนภาษา Go มองว่านี่คือปัญหา ไม่ใช่ feature: มันซ่อน path ที่ error เกิดขึ้นได้ไว้ในทุกบรรทัด ทำให้ผู้อ่านโค้ดต้องจำเองว่าบรรทัดไหน "อาจ throw" บ้าง

Go เลือกให้ **`error` เป็นแค่ value ธรรมดา** ที่ return กลับมาเหมือน return value อื่นๆ:

```go
type error interface {
    Error() string
}

func Divide(a, b float64) (float64, error) {
    if b == 0 {
        return 0, errors.New("division by zero")
    }
    return a / b, nil
}

result, err := Divide(10, 0)
if err != nil {
    // จัดการ error ตรงจุดที่เกิด ชัดเจน อยู่ในลำดับการอ่านโค้ดปกติ
    log.Printf("divide failed: %v", err)
    return
}
```

ข้อดี: ทุกจุดที่อาจเกิด error ถูกบังคับให้ "มองเห็นได้" ในลายเซ็นของฟังก์ชัน (`(T, error)`) และผู้เรียกต้องตัดสินใจ (แม้จะแค่ `_ = err` ก็ตาม เพียงแต่ linter ส่วนใหญ่จะเตือน) ข้อเสีย: โค้ดมี `if err != nil { return err }` เกลื่อนไปหมด ซึ่งเป็น trade-off ที่ Go ยอมรับเพื่อความชัดเจน

> Go ยังมี `panic`/`recover` แต่สงวนไว้สำหรับสถานการณ์ที่โปรแกรม**ไม่ควรทำงานต่อได้จริงๆ** (เช่น programmer error อย่าง index out of range, nil pointer dereference) ไม่ใช่กลไก error handling ปกติของ business logic — การ `panic` เพื่อ handle error ทั่วไปถือว่าผิด idiom ของภาษา

### errors.Is และ errors.As

ตั้งแต่ Go 1.13 มาตรฐาน error wrapping และ inspection ถูกทำให้เป็นทางการผ่านสอง function นี้:

```go
var ErrNotFound = errors.New("record not found")

func FindUser(id int64) (*User, error) {
    // ...
    return nil, fmt.Errorf("find user %d: %w", id, ErrNotFound) // %w = wrap
}

// เรียกใช้:
_, err := FindUser(42)
if errors.Is(err, ErrNotFound) {
    // true แม้ err จะถูก wrap ซ้อนกันหลายชั้นแล้วก็ตาม
    // errors.Is เดิน chain ของ Unwrap() ไล่เทียบทีละชั้น
}
```

- **`errors.Is(err, target)`** — เช็คว่า `err` (หรือ error ใดๆ ใน wrap chain ของมัน) "เป็น" `target` sentinel error ตัวเดียวกันหรือไม่ (เทียบด้วย `==` หรือ custom `Is()` method) ใช้แทนการเทียบ `err == ErrNotFound` ตรงๆ ที่จะพังทันทีถ้า error ถูก wrap
- **`errors.As(err, &target)`** — ดึง error ใน chain ที่ตรงกับ **type** ที่ระบุออกมาใส่ `target` ใช้เมื่อต้องการเข้าถึง field ของ custom error type ไม่ใช่แค่เช็คว่าตรงกันไหม

```go
type ValidationError struct {
    Field string
    Msg   string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("%s: %s", e.Field, e.Msg)
}

var ve *ValidationError
if errors.As(err, &ve) {
    fmt.Println("field ที่ผิด:", ve.Field) // เข้าถึง field เฉพาะของ ValidationError ได้
}
```

### %w vs %v ตอน wrap error

```go
fmt.Errorf("query user: %w", err) // wrap — err ต้นฉบับยังถูกเข้าถึงได้ผ่าน errors.Unwrap/Is/As
fmt.Errorf("query user: %v", err) // ไม่ wrap — แปลง err เป็น string ล้วนๆ เท่านั้น ปลาย chain ที่นี่
```

หลักปฏิบัติ: wrap ด้วย `%w` เมื่อผู้เรียกอาจต้องตรวจสอบ error ต้นทางต่อ (เช่น เพื่อ retry เฉพาะ network error) เพิ่ม context ทุกชั้นที่ error เดินทางผ่าน (เช่น `"handler: "`, `"usecase: "`, `"repository: "`) เพื่อให้ log สุดท้ายอ่านเป็น stack trace แบบ human-readable ได้ในตัว โดยไม่ต้องพึ่ง stack trace ของ runtime

## 1.5 Slice Internals

Slice เป็นโครงสร้างข้อมูลที่ใช้บ่อยที่สุดใน Go และเป็นจุดที่ทำให้เกิด bug เงียบๆ บ่อยที่สุด เพราะพฤติกรรมของมันดู "เหมือน" array แบบ dynamic ทั่วไป แต่จริงๆ แล้วมี internal state ที่ต้องเข้าใจ

Slice ไม่ใช่ข้อมูลตรงๆ แต่เป็น **struct เล็กๆ 3 field** ที่ชี้ไปยัง underlying array:

```
slice header
┌──────────────┬────────┬────────┐
│ pointer       │  len   │  cap   │
└───────┬──────┴────────┴────────┘
        │
        ▼
┌───┬───┬───┬───┬───┬───┐
│ 0 │ 1 │ 2 │ 3 │ 4 │ 5 │  ← underlying array
└───┴───┴───┴───┴───┴───┘
```

- **pointer** — ชี้ไปตำแหน่งแรกของช่วงข้อมูลที่ slice นี้มองเห็นใน underlying array (ไม่จำเป็นต้องเป็นตำแหน่งแรกของ array จริง)
- **len** — จำนวน element ที่ "มองเห็น" ตอนนี้ (`len(s)`)
- **cap** — จำนวน element สูงสุดที่ขยายได้จากตำแหน่ง pointer โดยไม่ต้องจอง array ใหม่ (`cap(s)`)

### ทำไม slice สองตัวถึงแชร์ underlying array กันได้

```go
a := []int{1, 2, 3, 4, 5}
b := a[1:3] // b = [2, 3], แต่ b ชี้ไปยัง underlying array ตัวเดียวกับ a!

b[0] = 99
fmt.Println(a) // [1 99 3 4 5] — a เปลี่ยนไปด้วย ทั้งที่แก้ผ่าน b!
```

Slicing (`a[1:3]`) ไม่ copy ข้อมูล มันแค่สร้าง slice header ใหม่ที่ pointer/len/cap ต่างไป แต่ยังชี้ไปยัง array เดิม การแก้ผ่าน `b` จึงกระทบ `a` ด้วยเสมอ นี่ไม่ใช่ bug — เป็น semantics ที่ตั้งใจออกแบบมาเพื่อความเร็ว (ไม่ copy โดยไม่จำเป็น) แต่ทำให้ผู้เขียนโค้ดต้องรู้ตัวเองว่ากำลังแชร์ memory อยู่

### "Append surprises you" — ตัวอย่าง bug คลาสสิก

```go
func addSuffix(base []int) []int {
    return append(base, 999)
}

func main() {
    original := make([]int, 3, 5) // len=3, cap=5 — มีที่ว่างเหลืออีก 2 ช่อง
    original[0], original[1], original[2] = 1, 2, 3

    modified := addSuffix(original)

    fmt.Println(original) // [1 2 3]
    fmt.Println(modified) // [1 2 3 999]

    modified[0] = 111
    fmt.Println(original) // [111 2 3] — original เปลี่ยนไปด้วย!! เพราะ cap ยังพอ append เลยเขียนทับ array เดิม
}
```

สิ่งที่เกิดขึ้น: `original` มี `cap=5` แต่ `len=3` เมื่อ `append(base, 999)` เรียก มันยังมีที่ว่างในช่อง index 3 (เพราะ `cap(base) > len(base)`) `append` จึงเขียนค่า `999` ลงไปที่ index 3 ของ **underlying array เดิม** แล้ว return slice header ใหม่ที่ `len=4` โดยไม่ได้จอง array ใหม่เลย ผลคือ `modified` กับ `original` ยังคง**แชร์ underlying array เดียวกัน** แม้ว่า `modified[0] = 111` จะดูเหมือนแก้แค่ `modified` แต่กระทบ `original[0]` ด้วยเพราะมันคือ memory ตำแหน่งเดียวกัน

แต่ถ้า `cap` ไม่พอ (`len == cap`) `append` จะจอง array **ใหม่** (ปกติขยายเป็นประมาณ 2 เท่า สำหรับ slice ขนาดเล็ก และอัตราลดลงเมื่อ slice ใหญ่ขึ้น) copy ข้อมูลเดิมไปที่ใหม่ แล้ว `modified` กับ `original` จะ**แยกจากกันโดยสมบูรณ์** — นี่คือความน่ากลัวของ bug นี้: พฤติกรรมขึ้นกับ `cap` ที่มองไม่เห็นจากการอ่านโค้ดผิวเผิน โค้ดเดียวกันอาจทำงานถูกหรือผิดขึ้นกับว่า caller ส่ง slice ที่มี spare capacity มาหรือไม่

### วิธีป้องกัน

```go
// วิธีที่ 1: บังคับให้ append ต้อง allocate ใหม่เสมอ ด้วยการจำกัด cap ให้เท่ากับ len
func addSuffixSafe(base []int) []int {
    result := make([]int, len(base), len(base)) // cap == len พอดี ไม่มีที่เหลือ
    copy(result, base)
    return append(result, 999)
}

// วิธีที่ 2: ใช้ full slice expression จำกัด cap ตอน slice
sub := original[1:3:3] // [low:high:max] — บังคับ cap ของ sub ให้เท่ากับ high (3) พอดี ไม่แชร์ช่องว่างเกินนั้น
```

## 1.6 Array vs Slice

| | Array | Slice |
|---|---|---|
| ขนาด | คงที่ กำหนดตอน compile time เป็นส่วนหนึ่งของ type (`[5]int` กับ `[10]int` คนละ type กัน) | ยืดหยุ่น ปรับขนาดได้ (`append`) |
| Pass เข้าฟังก์ชัน | copy ทั้ง array (แพงถ้าอาร์เรย์ใหญ่) | copy แค่ slice header (pointer+len+cap) — เบา |
| การใช้งานจริง | ใช้น้อยมากในโค้ด Go ทั่วไป เหมาะกับข้อมูลขนาดคงที่ตายตัวจริงๆ (เช่น checksum `[32]byte`) | ใช้เป็น default สำหรับ collection แทบทุกกรณี |

ในทางปฏิบัติ โค้ด Go ระดับ production แทบไม่ใช้ array ตรงๆ เลยนอกจากกรณีพิเศษ (fixed-size buffer, hash) เพราะ slice ยืดหยุ่นกว่าและ cost การ copy header ต่ำกว่า copy ทั้ง array มาก

## 1.7 Generics

ก่อน Go 1.18 (2022) ภาษา Go ไม่มี generics เลย — ทางเลือกคือเขียนโค้ดซ้ำสำหรับแต่ละ type หรือใช้ `interface{}` แล้วเสีย type safety Go 1.18 เพิ่ม **type parameters** เข้ามาแก้ปัญหานี้โดยตรง

```go
// T คือ type parameter, constraint (คุณสมบัติที่ T ต้องมี) คือ comparable
func Contains[T comparable](slice []T, target T) bool {
    for _, v := range slice {
        if v == target {
            return true
        }
    }
    return false
}

Contains([]int{1, 2, 3}, 2)          // T = int
Contains([]string{"a", "b"}, "c")    // T = string
```

### Constraint แบบกำหนดเอง

```go
type Number interface {
    ~int | ~int64 | ~float64 // ~T หมายถึงรวม type ใดๆ ที่มี underlying type เป็น T ด้วย
}

func Sum[T Number](values []T) T {
    var total T
    for _, v := range values {
        total += v
    }
    return total
}
```

Package มาตรฐาน `slices` และ `maps` (ตั้งแต่ Go 1.21) เขียนด้วย generics ทั้งชุด (`slices.Contains`, `slices.Sort`, `maps.Keys`) แทนที่ต้องเขียน sort/contains แยกตาม type เองแบบสมัยก่อน

### เมื่อไหร่ควรใช้ generics เมื่อไหร่ไม่ควร

**ควรใช้เมื่อ:**
- เขียน data structure หรือ algorithm ที่ทำงานเหมือนกันทุก type จริงๆ (linked list, stack, `Map`/`Filter`/`Reduce` แบบ generic, repository interface ที่ generic ตาม entity type)
- แทนที่โค้ดที่ก่อนหน้านี้ต้อง copy-paste สำหรับแต่ละ type หรือต้องใช้ `interface{}` + type assertion ที่เสี่ยง panic ตอน runtime

**ไม่ควรใช้เมื่อ:**
- Logic เฉพาะเจาะจงกับ type ใด type หนึ่งอยู่แล้ว (generics ไม่ได้ช่วยอะไร มีแต่เพิ่มความซับซ้อนในการอ่าน)
- ใช้แค่เพื่อ "ดูทันสมัย" ทั้งที่ interface ปกติแก้ปัญหาได้ง่ายกว่าและอ่านง่ายกว่า — Go idiom ยังคงให้ความสำคัญกับ interface เป็นกลไก polymorphism หลักของภาษา generics เป็นเครื่องมือเสริมสำหรับกรณีที่ interface แก้ไม่ได้ (เช่น ต้องการ type ที่ return ตรงกับ type ที่รับเข้ามา ซึ่ง interface ทำไม่ได้ตรงๆ)

Rob Pike (หนึ่งในผู้ออกแบบภาษา) เคยกล่าวไว้ทำนองว่า "a little copying is better than a little dependency" — ปรัชญานี้ยังใช้ได้กับ generics เช่นกัน: อย่า generic ทุกอย่างเผื่ออนาคตที่ยังไม่มาถึง ให้ generic เฉพาะจุดที่พิสูจน์แล้วว่าโค้ดซ้ำจริงและซ้ำในรูปแบบเดียวกันหลาย type

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| Struct & Embedding | ไม่มี inheritance มีแต่ composition ผ่าน embedding — field/method ถูก "promote" ขึ้นมาเท่านั้น |
| Interface | Implicit implementation (structural typing) — ทำให้ decoupling ระหว่าง package เป็นธรรมชาติโดยไม่ต้องพึ่ง framework |
| Pointer vs Value receiver | Value receiver ทำงานกับสำเนา แก้ field แล้วไม่กระทบตัวจริง — ต้องใช้ pointer receiver ถ้าต้อง mutate state |
| Error handling | `error` เป็น value ธรรมดา ไม่ใช่ exception, wrap ด้วย `%w`, ตรวจสอบด้วย `errors.Is`/`errors.As` |
| Slice internals | Slice = pointer+len+cap ชี้ไปยัง underlying array, slicing/append ที่ cap ยังพอจะ "แชร์" array เดิมโดยไม่รู้ตัว |
| Array vs Slice | Array ขนาดคงที่ตาย ใช้น้อยมากในทางปฏิบัติ, slice ยืดหยุ่นและเป็น default |
| Generics | ใช้เมื่อ logic ซ้ำกันจริงข้าม type ไม่ใช้พร่ำเพรื่อแทน interface |

## คำถามทบทวน

1. ทำไม embedding ใน Go ถึงไม่ใช่ inheritance แม้จะเข้าถึง field ของ struct ที่ฝังได้ตรงๆ ก็ตาม?
2. อธิบายว่า "implicit implementation" ของ interface ใน Go ช่วยเรื่อง dependency direction ระหว่าง package ได้อย่างไร ทำไมถึงสำคัญกับ Clean Architecture
3. เขียนตัวอย่างโค้ดสั้นๆ ที่แสดง bug จากการใช้ value receiver กับ method ที่ควรจะ mutate state แล้วอธิบายว่าทำไมมันถึง compile ผ่านแต่ทำงานผิด
4. `original := make([]int, 3, 5)` แล้ว `modified := append(original, 999)` จากนั้นแก้ `modified[0]` จะกระทบ `original` หรือไม่ เพราะเหตุใด แล้วถ้า `make([]int, 3, 3)` จะต่างกันอย่างไร?
5. ยกตัวอย่างสถานการณ์ที่ควรใช้ generics และสถานการณ์ที่ควรใช้ interface ปกติแทน พร้อมเหตุผล

---

ถัดไป: [บทที่ 2 — Concurrency และ Context](02-concurrency-and-context.md)
