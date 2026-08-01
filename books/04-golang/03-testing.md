# บทที่ 3: Testing

Go มี testing framework ในตัว (`testing` package) เป็นส่วนหนึ่งของ standard library โดยไม่ต้องพึ่ง third-party framework ใดๆ เลยก็เขียน test ที่มีคุณภาพสูงได้ครบ — ปรัชญานี้สอดคล้องกับแนวทาง "batteries included" ของ Go ทั้งภาษา บทนี้เน้นที่ **table-driven test** ซึ่งเป็นรูปแบบการเขียน test ที่ idiomatic ที่สุดของ Go และเป็นสิ่งที่ผู้สัมภาษณ์ระดับ senior มักถามหา

## 3.1 testing Package พื้นฐาน

Test file ต้องอยู่ในไฟล์ที่ลงท้ายด้วย `_test.go` และแต่ละ test function ต้องมี signature `func TestXxx(t *testing.T)`:

```go
// math.go
package mathutil

func Add(a, b int) int {
    return a + b
}

// math_test.go
package mathutil

import "testing"

func TestAdd(t *testing.T) {
    got := Add(2, 3)
    want := 5
    if got != want {
        t.Errorf("Add(2, 3) = %d; want %d", got, want)
    }
}
```

รันด้วย `go test ./...` (รัน test ทุก package ใน module) หรือ `go test -v ./...` (verbose แสดงชื่อ test ที่รันทั้งหมดพร้อมผล PASS/FAIL)

### t.Error vs t.Fatal

- **`t.Errorf`/`t.Error`** — mark ว่า test ล้มเหลว แต่**ทำงานต่อ**จนจบ function (เหมาะกับ assertion ที่อยากเห็น failure ทุกจุดในครั้งเดียว ไม่ใช่หยุดที่จุดแรก)
- **`t.Fatalf`/`t.Fatal`** — mark ว่าล้มเหลวแล้ว**หยุดทันที** (เหมาะกับกรณีที่ขั้นตอนถัดไปจะ panic หรือไม่มีความหมายถ้าเงื่อนไขก่อนหน้าไม่ผ่าน เช่น setup ล้มเหลว)

```go
resp, err := http.Get(server.URL)
if err != nil {
    t.Fatalf("unexpected error: %v", err) // ถ้า err ไม่ nil อ่าน resp.Body ต่อจะ panic แน่ๆ ต้องหยุด
}
defer resp.Body.Close()
```

## 3.2 Table-Driven Test

นี่คือรูปแบบการเขียน test ที่ idiomatic ที่สุดของ Go — แทนที่จะเขียน `TestXxx` แยกกันหลายฟังก์ชันสำหรับแต่ละ case ให้นิยาม **ตาราง (slice ของ struct)** ที่เก็บ input/output ที่คาดหวังของทุก case แล้ววน loop เดียวรันทั้งหมด:

```go
func TestDivide(t *testing.T) {
    tests := []struct {
        name    string
        a, b    float64
        want    float64
        wantErr bool
    }{
        {name: "normal division", a: 10, b: 2, want: 5, wantErr: false},
        {name: "division by zero", a: 10, b: 0, want: 0, wantErr: true},
        {name: "negative numbers", a: -10, b: 2, want: -5, wantErr: false},
        {name: "zero dividend", a: 0, b: 5, want: 0, wantErr: false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Divide(tt.a, tt.b)

            if (err != nil) != tt.wantErr {
                t.Fatalf("Divide(%v, %v) error = %v, wantErr %v", tt.a, tt.b, err, tt.wantErr)
            }
            if err == nil && got != tt.want {
                t.Errorf("Divide(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
            }
        })
    }
}
```

### ทำไม pattern นี้ถึงเป็น idiom ที่ดี

1. **ลดโค้ดซ้ำ** — logic การรัน/assert เขียนครั้งเดียว case ใหม่แค่เพิ่มแถวในตาราง ไม่ต้อง copy-paste function ทดสอบทั้งก้อน
2. **`t.Run(tt.name, ...)` ทำให้แต่ละ case เป็น subtest แยกกัน** — เห็นผลแยกทีละ case ใน output (`--- PASS: TestDivide/normal_division`), รัน case เดียวได้ด้วย `go test -run TestDivide/division_by_zero`, และถ้า case หนึ่ง fail case อื่นยังรันต่อได้ครบ (ไม่ใช่ทั้ง TestDivide หยุดตั้งแต่ case แรกที่พัง)
3. **บังคับให้คิดถึง edge case อย่างเป็นระบบ** — พอเห็นเป็นตาราง มักเห็นได้ทันทีว่า case ไหนยังขาด (boundary value, error case, zero value)
4. **อ่านง่ายเป็นเอกสารในตัว** — ตารางที่ดีคือสเปกของฟังก์ชันในรูปแบบที่รันได้จริง (executable specification)

## 3.3 Mocking ผ่าน Interface

เพราะ Go ใช้ implicit interface implementation (ดู [บทที่ 1.2](01-language-fundamentals.md#12-interface--implicit-implementation)) การเขียนโค้ดที่ testable จึงทำได้เป็นธรรมชาติมาก โดยไม่ต้องพึ่ง mocking framework ที่ทำ bytecode manipulation แบบภาษาอื่น — แค่ออกแบบให้ business logic depend บน **interface** ไม่ใช่ **concrete implementation** ตั้งแต่แรก (แนวทางนี้คือสิ่งที่ทำให้ dependency injection ใน [บทที่ 4](04-architecture-patterns.md) ทำงานร่วมกับ testing ได้อย่างลงตัว):

```go
// domain layer: นิยาม interface (port) — ไม่รู้จัก implementation จริงเลย
type PaymentGateway interface {
    Charge(ctx context.Context, amount int64, cardToken string) (string, error)
}

// use case ขึ้นกับ interface เท่านั้น ไม่ขึ้นกับ Stripe/Omise ตัวจริง
type CheckoutUseCase struct {
    gateway PaymentGateway
}

func (uc *CheckoutUseCase) Checkout(ctx context.Context, amount int64, token string) error {
    txnID, err := uc.gateway.Charge(ctx, amount, token)
    if err != nil {
        return fmt.Errorf("checkout: %w", err)
    }
    log.Println("charged, txn:", txnID)
    return nil
}
```

Test เขียน mock ของ `PaymentGateway` เองตรงๆ โดยไม่ต้องแตะโค้ด production เลย เพราะ `PaymentGateway` แค่เป็น interface — struct อะไรก็ได้ที่มี method `Charge` ตรง signature ก็ใช้แทนได้ทันที:

```go
type mockGateway struct {
    chargeFunc func(ctx context.Context, amount int64, cardToken string) (string, error)
}

func (m *mockGateway) Charge(ctx context.Context, amount int64, cardToken string) (string, error) {
    return m.chargeFunc(ctx, amount, cardToken)
}

func TestCheckoutUseCase_Checkout(t *testing.T) {
    tests := []struct {
        name       string
        chargeFunc func(ctx context.Context, amount int64, cardToken string) (string, error)
        wantErr    bool
    }{
        {
            name: "successful charge",
            chargeFunc: func(ctx context.Context, amount int64, cardToken string) (string, error) {
                return "txn_123", nil
            },
            wantErr: false,
        },
        {
            name: "gateway rejects card",
            chargeFunc: func(ctx context.Context, amount int64, cardToken string) (string, error) {
                return "", errors.New("card declined")
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            uc := &CheckoutUseCase{gateway: &mockGateway{chargeFunc: tt.chargeFunc}}
            err := uc.Checkout(context.Background(), 1000, "tok_abc")
            if (err != nil) != tt.wantErr {
                t.Errorf("Checkout() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

Pattern `mockXxx` ที่ห่อ closure (`chargeFunc`) แบบนี้เรียกกันในวงการว่า **function field mock** — ยืดหยุ่นกว่า mock ที่ hardcode พฤติกรรมตายตัว เพราะแต่ละ test case กำหนดพฤติกรรมของ mock ได้อิสระผ่านตาราง โดยไม่ต้องสร้าง mock struct แยกสำหรับทุก scenario

ทีมที่ต้องการ mock ที่ generate อัตโนมัติ (ลด boilerplate การเขียน mock เองสำหรับ interface ใหญ่ๆ) มักใช้เครื่องมืออย่าง [`gomock`](https://github.com/uber-go/mock) หรือ [`mockery`](https://github.com/vektra/mockery) ซึ่งอ่าน interface แล้ว generate mock code ให้อัตโนมัติ — แต่หลักการเบื้องหลังยังเหมือนเดิมทุกประการ: **ทำได้เพราะ interface implementation เป็น implicit**

## 3.4 Benchmark Test

Function ที่ชื่อขึ้นต้นด้วย `Benchmark` และรับ `*testing.B` คือ benchmark test — ใช้วัดประสิทธิภาพ (เวลา, จำนวน memory allocation) ของโค้ดอย่างเป็นระบบ:

```go
func BenchmarkAdd(b *testing.B) {
    for i := 0; i < b.N; i++ {
        Add(2, 3)
    }
}

func BenchmarkStringConcat(b *testing.B) {
    for i := 0; i < b.N; i++ {
        s := ""
        for j := 0; j < 100; j++ {
            s += "x" // string concat แบบ naive — allocate ใหม่ทุกครั้ง
        }
    }
}

func BenchmarkStringBuilder(b *testing.B) {
    for i := 0; i < b.N; i++ {
        var sb strings.Builder
        for j := 0; j < 100; j++ {
            sb.WriteString("x")
        }
        _ = sb.String()
    }
}
```

รันด้วย:

```bash
go test -bench=. -benchmem ./...
```

### อ่านผล benchmark

```
BenchmarkAdd-8                 1000000000    0.25 ns/op    0 B/op    0 allocs/op
BenchmarkStringConcat-8            120000    9532 ns/op   4960 B/op   99 allocs/op
BenchmarkStringBuilder-8          1850000     645 ns/op    504 B/op     7 allocs/op
```

- **`-8`** ต่อท้ายชื่อ = ค่า `GOMAXPROCS` ตอนรัน
- **คอลัมน์แรก** — จำนวนรอบที่ `b.N` ถูกปรับให้รันจริง (Go ปรับ `b.N` ขึ้นอัตโนมัติจนกว่าผลลัพธ์จะเสถียรพอที่จะวัดได้แม่นยำ — ไม่ต้องกำหนดเอง)
- **`ns/op`** — เวลาเฉลี่ยต่อการรันหนึ่งครั้ง (nanosecond/operation) — ตัวเลขที่สำคัญที่สุดในการเทียบ performance
- **`-benchmem`** เพิ่มสองคอลัมน์: **`B/op`** (จำนวน byte ที่ allocate ต่อครั้ง) และ **`allocs/op`** (จำนวนครั้งที่ allocate memory ต่อครั้ง) — สำคัญมากเพราะ allocation ที่มากเกินจำเป็นสร้างภาระให้ garbage collector และมักเป็นสาเหตุของ latency ที่ไม่แน่นอน (GC pause) มากกว่า raw CPU time เสียอีก

จากตัวอย่างข้างบน เห็นชัดว่า `strings.Builder` เร็วกว่าและ allocate น้อยกว่า string concatenation แบบ `+=` มาก (เพราะ `+=` สร้าง string ใหม่ทุกครั้งที่บวก ในขณะที่ `Builder` เขียนต่อท้าย buffer เดียวและขยายแบบ amortized คล้ายกับ slice `append`) — นี่คือตัวอย่างจริงว่าทำไม benchmark ถึงสำคัญ: ความแตกต่างแบบนี้มองจากโค้ดเฉยๆ ไม่เห็นชัด ต้องวัดจริงถึงจะรู้

## 3.5 Unit Test vs Integration Test

| | Unit Test | Integration Test |
|---|---|---|
| ขอบเขต | ทดสอบ function/struct เดียว แยกขาดจากภายนอกทั้งหมด | ทดสอบการทำงานร่วมกันของหลาย component จริง (database, message queue, HTTP call) |
| Dependency ภายนอก | Mock/stub ทั้งหมด (ไม่แตะ network, disk, database จริง) | ใช้ของจริงหรือใกล้เคียงของจริงมากที่สุด (เช่น container ของ database จริงผ่าน `testcontainers`) |
| ความเร็ว | เร็วมาก (milliseconds) รันได้หลายพันเทสใน CI ไม่กี่วินาที | ช้ากว่ามาก (ต้อง spin up dependency จริง) |
| ความมั่นใจที่ได้ | มั่นใจว่า logic ถูกต้องแบบแยกส่วน | มั่นใจว่าระบบทำงานถูกต้องแบบ end-to-end จริง รวมถึง query SQL จริง, network call จริง |
| จำนวนที่ควรมี | มากที่สุด (ฐานของ testing pyramid) | น้อยกว่า เฉพาะจุดที่คุ้มค่า (critical path) |

ในโปรเจกต์ Go ที่ยึด Clean Architecture ([บทที่ 4](04-architecture-patterns.md)) unit test มักครอบคลุม **use case layer** และ **domain logic** เป็นหลัก (mock repository/gateway ทั้งหมดตามที่แสดงในหัวข้อ 3.3) ส่วน integration test มักครอบคลุม **repository layer จริง** (ยิง query จริงใส่ database ทดสอบผ่าน Docker container) และ **HTTP handler แบบ end-to-end** (ยิง request จริงผ่าน `httptest.Server` ทดสอบทั้ง chain)

Convention ที่ใช้บ่อยในการแยกสอง test แบบนี้ในโค้ด Go:

```go
//go:build integration

package repository_test

func TestPostgresOrderRepo_FindByID(t *testing.T) {
    // เชื่อมต่อ database จริง (มักผ่าน testcontainers-go หรือ docker-compose)
    // ...
}
```

build tag `//go:build integration` ทำให้ test นี้ **ไม่ถูกรวม** ใน `go test ./...` แบบปกติ ต้องสั่งรันแยกด้วย `go test -tags=integration ./...` เท่านั้น — แยก pipeline ของ CI ให้ unit test รันเร็วทุก commit ในขณะที่ integration test รันในรอบที่กว้างกว่า (เช่น ก่อน merge เข้า main หรือ nightly build) เพื่อ balance ระหว่างความเร็วของ feedback loop กับความครอบคลุมของการทดสอบ

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| testing package | มาพร้อม standard library, ไฟล์ `_test.go`, `t.Error` ทำงานต่อ vs `t.Fatal` หยุดทันที |
| Table-driven test | นิยาม case เป็นตาราง วน loop เดียวรันทุก case ผ่าน `t.Run` — idiom หลักของ Go testing |
| Mocking ผ่าน interface | เพราะ implicit implementation ทำ mock ได้โดยไม่ต้องแตะโค้ด production เลย |
| Benchmark test | `BenchmarkXxx(b *testing.B)`, อ่าน `ns/op` และ `B/op`/`allocs/op` จาก `-benchmem` |
| Unit vs Integration | Unit เร็ว mock ทุกอย่าง เป็นฐานของ testing pyramid; Integration ช้ากว่าแต่มั่นใจ end-to-end กว่า แยกด้วย build tag |

## คำถามทบทวน

1. ทำไม table-driven test ถึงถือเป็น idiom หลักของการเขียน test ใน Go เทียบกับการเขียน `TestXxx` แยกฟังก์ชันสำหรับแต่ละ case?
2. `t.Error` กับ `t.Fatal` ต่างกันอย่างไร ยกตัวอย่างสถานการณ์ที่ต้องใช้ `t.Fatal` เท่านั้น
3. อธิบายว่าทำไม implicit interface implementation ของ Go (บทที่ 1) ถึงทำให้เขียน mock ได้ง่ายโดยไม่ต้องพึ่ง framework พิเศษ
4. อ่านผล benchmark `1850000  645 ns/op  504 B/op  7 allocs/op` แล้วอธิบายว่าแต่ละตัวเลขหมายถึงอะไร
5. อธิบายความแตกต่างของ unit test กับ integration test ในบริบท Clean Architecture ว่าแต่ละแบบควรเทสที่ layer ไหน

---

ก่อนหน้า: [บทที่ 2 — Concurrency และ Context](02-concurrency-and-context.md) | ถัดไป: [บทที่ 4 — Architecture Patterns](04-architecture-patterns.md)
