# บทที่ 4: Architecture Patterns

บทนี้พาไปดูว่าจะจัดโครงสร้างโปรเจกต์ Go ขนาดใหญ่ (ระดับ enterprise, มีทีมหลายคนดูแลร่วมกันในระยะยาว) อย่างไรให้ **testable, maintainable, และไม่ผูกติดกับ framework/database ตัวใดตัวหนึ่งจนเปลี่ยนไม่ได้** เนื้อหาจะครอบคลุมทั้งแนวคิดสถาปัตยกรรมระดับสูง (Clean Architecture, DDD, Hexagonal) และ pattern ระดับโค้ดที่ใช้ implement แนวคิดเหล่านั้นจริง (Repository, Dependency Injection, Middleware) เพราะทั้งสองส่วนนี้แยกกันพูดไม่ได้ — pattern ระดับโค้ดคือสิ่งที่ทำให้แนวคิดสถาปัตยกรรมเป็นจริงได้ในภาษา Go

## 4.1 Clean Architecture

### The Dependency Rule

หัวใจของ Clean Architecture (แนวคิดโดย Robert C. Martin) คือกฎข้อเดียว: **source code dependency ต้องชี้เข้าหาศูนย์กลางเสมอ (inward only)** ชั้นนอกรู้จักชั้นใน แต่ชั้นในต้อง**ไม่รู้จัก**ชั้นนอกเลย ไม่ว่ากรณีใดก็ตาม

```
┌─────────────────────────────────────────────────────┐
│  Frameworks & Drivers (DB, Web, UI, External API)     │
│  ┌─────────────────────────────────────────────────┐ │
│  │  Interface Adapters (Controller, Presenter, Gateway)│
│  │  ┌───────────────────────────────────────────┐   │ │
│  │  │  Use Cases (Application Business Rules)     │   │ │
│  │  │  ┌─────────────────────────────────────┐   │   │ │
│  │  │  │  Entities (Enterprise Business Rules) │   │   │ │
│  │  │  └─────────────────────────────────────┘   │   │ │
│  │  └───────────────────────────────────────────┘   │ │
│  └─────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────┘

              dependency ชี้เข้าใน (inward) เท่านั้น ─────►
```

- **Entities** — business rule ที่เป็นแก่นแท้ที่สุดขององค์กร ไม่เปลี่ยนตามว่าใช้ web/CLI/database ไหน (เช่น กฎว่า "ยอดเงินในบัญชีต้องไม่ติดลบ")
- **Use Cases** — business rule เฉพาะของแอปพลิเคชันนี้ (application-specific) orchestrate การทำงานระหว่าง entity หลายตัวเพื่อทำ business flow หนึ่งให้สำเร็จ (เช่น "ขั้นตอนการ checkout")
- **Interface Adapters** — แปลงข้อมูลไปมาระหว่างรูปแบบที่ use case สะดวกใช้ กับรูปแบบที่ framework/database ต้องการ (controller แปลง HTTP request เป็น input ของ use case, presenter แปลงผลลัพธ์กลับเป็น JSON response, repository implementation แปลง entity เป็น SQL row)
- **Frameworks & Drivers** — รายละเอียดที่เปลี่ยนได้ง่ายที่สุด (web framework, ORM, message broker) เป็นชั้นนอกสุดโดยเจตนา

### ทำไม dependency ต้องชี้เข้าในเท่านั้น

เพราะสิ่งที่ **เปลี่ยนบ่อยที่สุด** (database ที่ใช้, web framework, third-party API) ควรอยู่ชั้นนอกสุด และสิ่งที่ **มีค่าที่สุดและควรเปลี่ยนน้อยที่สุด** (business rule ขององค์กร) ควรอยู่ตรงกลางที่ไม่ถูกกระทบเมื่อรายละเอียดชั้นนอกเปลี่ยน — ถ้าวันหนึ่งทีมตัดสินใจย้ายจาก PostgreSQL ไป MySQL หรือจาก REST ไป gRPC การเปลี่ยนแปลงควรจำกัดอยู่แค่ชั้นนอก (interface adapters, frameworks) โดย use case และ entity ไม่ต้องแก้แม้แต่บรรทัดเดียว

ใน Go กฎนี้ทำได้เป็นธรรมชาติผ่าน **interface ที่นิยามในชั้นใน implement โดยชั้นนอก** (ตามที่อธิบายใน [บทที่ 1.2](01-language-fundamentals.md#12-interface--implicit-implementation)) — use case นิยาม interface ของ repository ที่ตัวเองต้องการ ส่วน infrastructure layer เป็นคน implement interface นั้น ทำให้ source code dependency ชี้เข้าในจริงๆ (use case ไม่ import package ของ infrastructure เลย) แม้ว่า control flow ตอน runtime จะวิ่งออกไปเรียก infrastructure ก็ตาม — นี่คือหัวใจของ **Dependency Inversion Principle**

## 4.2 DDD (Domain-Driven Design)

DDD คือแนวคิดที่เน้นออกแบบโค้ดให้สะท้อน "โมเดลของ business domain" ให้ตรงที่สุด ไม่ใช่ออกแบบตามโครงสร้างตาราง database หรือความสะดวกทางเทคนิค ต่อไปนี้คือ concept หลักที่ต้องเข้าใจ อธิบายผ่านตัวอย่าง e-commerce (`Order`):

### Entity

**Entity** คือ object ที่มี **identity** ที่ติดตัวไปตลอด แม้ attribute ข้างในจะเปลี่ยนไปก็ยังเป็น "ตัวเดียวกัน" — เทียบด้วย ID ไม่ใช่เทียบด้วยค่า field:

```go
type Order struct {
    ID       OrderID // identity — Order สองใบที่ ID ต่างกันคือคนละใบเสมอ แม้ field อื่นเหมือนกันทุกอย่าง
    Status   OrderStatus
    Items    []OrderItem
    Total    Money
}
```

### Value Object

**Value Object** คือ object ที่ **ไม่มี identity** — เทียบความเท่ากันด้วยค่าของมันเท่านั้น (สอง instance ที่ค่าเหมือนกันถือว่าเท่ากันเสมอ) และควรเป็น **immutable**:

```go
type Money struct {
    Amount   int64  // เก็บเป็นหน่วยเล็กสุด (สตางค์) หลีกเลี่ยง floating point กับเงิน
    Currency string
}

func (m Money) Add(other Money) (Money, error) {
    if m.Currency != other.Currency {
        return Money{}, errors.New("currency mismatch")
    }
    return Money{Amount: m.Amount + other.Amount, Currency: m.Currency}, nil // return ค่าใหม่ ไม่แก้ตัวเดิม
}
```

`Money{Amount: 10000, Currency: "THB"}` สอง instance ถือว่า "เท่ากัน" เสมอไม่ว่าจะสร้างจากที่ไหน ต่างจาก `Order` ที่ต้องเทียบด้วย `ID` เท่านั้น — สังเกตว่า `Add` ใช้ value receiver และ return ค่าใหม่แทนการแก้ค่าเดิม สอดคล้องกับหลัก immutability ของ value object (เชื่อมกับ [บทที่ 1.3](01-language-fundamentals.md#13-pointer-vs-value-receiver) เรื่อง value receiver เหมาะกับ type ที่ immutable โดยธรรมชาติ)

### Aggregate

**Aggregate** คือกลุ่มของ entity/value object ที่ถูกจัดการเป็น "หน่วยเดียว" เพื่อรักษาความถูกต้องของ business rule (invariant) ทุกครั้งที่แก้ไข — มี **Aggregate Root** เป็นทางเข้าเดียวที่โค้ดภายนอกได้รับอนุญาตให้แก้ไขข้อมูลภายใน aggregate นั้น:

```go
// Order คือ Aggregate Root — โค้ดภายนอกห้ามแก้ Items ตรงๆ ต้องผ่าน method ของ Order เท่านั้น
type Order struct {
    ID     OrderID
    Status OrderStatus
    items  []OrderItem // unexported — บังคับให้แก้ผ่าน method เท่านั้น
}

func (o *Order) AddItem(item OrderItem) error {
    if o.Status != StatusDraft {
        return errors.New("cannot add item to a confirmed order") // invariant: แก้รายการได้เฉพาะตอน draft
    }
    o.items = append(o.items, item)
    return nil
}

func (o *Order) Confirm() error {
    if len(o.items) == 0 {
        return errors.New("cannot confirm an empty order") // invariant: order ว่างเปล่ายืนยันไม่ได้
    }
    o.Status = StatusConfirmed
    return nil
}
```

`OrderItem` เป็นส่วนหนึ่งของ aggregate `Order` — ไม่มีความหมายอิสระนอก `Order` และไม่ควรถูกแก้ไขตรงๆ จากภายนอกโดยข้าม `Order` เพราะจะทำให้ invariant (เช่น "ห้ามแก้รายการหลัง confirm แล้ว") ถูกข้ามไปได้ การบังคับให้ field เป็น unexported (`items` ตัวเล็ก) แล้วเปิดแค่ method ที่ผ่านการตรวจสอบ (`AddItem`, `Confirm`) คือวิธีที่ Go ใช้บังคับกฎ "แก้ผ่าน aggregate root เท่านั้น" ในระดับภาษา

### Domain Service

เมื่อ business logic หนึ่งไม่ได้เป็นของ entity หรือ value object ตัวใดตัวหนึ่งโดยธรรมชาติ (เช่น ต้องใช้ข้อมูลจากหลาย aggregate ร่วมกัน) ให้ดึงออกมาเป็น **Domain Service** แทนที่จะฝืนยัดใส่ entity ตัวใดตัวหนึ่ง:

```go
// PricingService ต้องรู้ทั้ง Order และ CustomerTier (คนละ aggregate กัน) จึงไม่ควรเป็น method ของ Order เฉยๆ
type PricingService struct{}

func (s *PricingService) CalculateDiscount(order *Order, tier CustomerTier) Money {
    // logic คำนวณส่วนลดที่ขึ้นกับทั้ง order และ tier ของลูกค้า
    // ...
}
```

### Bounded Context

**Bounded Context** คือขอบเขตที่ model หนึ่งๆ มีความหมายและถูกต้องสมบูรณ์ในตัวเอง — คำศัพท์เดียวกันอาจมีความหมายต่างกันในแต่ละ context ได้ เช่นคำว่า "Product" ใน **Catalog context** (มี description, รูปภาพ, หมวดหมู่) กับ "Product" ใน **Inventory context** (มี SKU, จำนวนคงคลัง, warehouse location) คือคนละโมเดลกันโดยสิ้นเชิง แม้จะอ้างถึง "สินค้าชิ้นเดียวกัน" ในโลกจริงก็ตาม — พยายามยัดทุกอย่างเป็น struct `Product` ตัวเดียวใช้ร่วมกันทุก context มักจบลงด้วย struct ขนาดใหญ่ที่มี field จำนวนมากซึ่งแต่ละ context ใช้ไม่ครบ และแก้ยากเพราะทุกทีมแตะ struct เดียวกัน — แนวทางที่ถูกต้องคือแยกแต่ละ context เป็น service/module อิสระ (มักตรงกับขอบเขตของ [microservice ใน Volume 11](../11-microservices/README.md)) แล้วสื่อสารข้าม context ผ่าน explicit contract (API, event) เท่านั้น

## 4.3 Hexagonal Architecture (Ports & Adapters)

Hexagonal Architecture (คิดค้นโดย Alistair Cockburn) มีเป้าหมายเดียวกับ Clean Architecture คือแยก core business logic ออกจาก infrastructure แต่อธิบายด้วยคำศัพท์ **Port** และ **Adapter**:

- **Port** — interface ที่ **core (domain/application layer) เป็นคนนิยาม** ว่าต้องการอะไรจากโลกภายนอก (เช่น "ฉันต้องการที่เก็บและค้นหา order") ไม่สนใจว่าจะ implement ด้วยอะไร
- **Adapter** — implementation จริงของ port ที่อยู่**นอก** core เชื่อมต่อกับเทคโนโลยีจริง (PostgreSQL, REST API, message queue) — มีได้หลาย adapter ต่อหนึ่ง port (เช่น `PostgresOrderRepo` กับ `InMemoryOrderRepo` สำหรับ test ต่างก็ implement `OrderRepository` port เดียวกัน)

```
                    ┌───────────────────────────────┐
   Driving Adapter   │                                 │   Driven Adapter
   (เรียกเข้า core)   │            Core / Domain        │   (core เรียกออกไป)
                    │                                 │
┌──────────────┐    │   ┌─────────────────────────┐   │    ┌──────────────┐
│ HTTP Handler │───►│   │   Use Case / Domain       │   │───►│ Postgres Repo │
└──────────────┘    │   │   (ไม่รู้จัก HTTP/Postgres  │   │    └──────────────┘
┌──────────────┐    │   │   เลยแม้แต่น้อย)           │   │    ┌──────────────┐
│ gRPC Handler │───►│   │                           │   │───►│ Stripe Client │
└──────────────┘    │   └─────────────────────────┘   │    └──────────────┘
┌──────────────┐    │                                 │    ┌──────────────┐
│ CLI Command  │───►│         Port (interface)         │───►│ SQS Publisher │
└──────────────┘    │                                 │    └──────────────┘
                    └───────────────────────────────┘
```

จุดที่ Hexagonal เน้นชัดกว่า Clean Architecture คือการแยก adapter เป็นสองฝั่งอย่างชัดเจน:

- **Driving/Primary Adapter** — สิ่งที่ "เรียกเข้ามา" หา core (HTTP handler, gRPC handler, CLI, scheduled job) — core เป็นฝ่าย**ถูกเรียก**
- **Driven/Secondary Adapter** — สิ่งที่ core "เรียกออกไป" (database, external API, message queue) — core เป็นฝ่าย**เรียก**ผ่าน port ที่ตัวเองนิยาม

แนวคิดทั้งสอง (Clean Architecture และ Hexagonal) เป็นการมองปัญหาเดียวกันจากมุมต่างกัน แต่ผลลัพธ์ในโค้ด Go จริงแทบไม่ต่างกันเลย — ทั้งคู่จบลงที่หลักการเดียวกัน: **นิยาม interface ฝั่ง core แล้วให้ infrastructure เป็นคน implement**

## 4.4 Repository Pattern

**Repository** คือการนำ Port/Ports & Adapters มาปรับใช้เฉพาะกับ data access — แยก "การเข้าถึงข้อมูล" ออกจาก "business logic" โดยสิ้นเชิง

```go
// === domain layer: นิยาม interface (port) ===
package domain

type OrderRepository interface {
    FindByID(ctx context.Context, id OrderID) (*Order, error)
    Save(ctx context.Context, order *Order) error
}

// === infrastructure layer: implement interface (adapter) ===
package postgres

type OrderRepo struct {
    db *sql.DB
}

func NewOrderRepo(db *sql.DB) *OrderRepo {
    return &OrderRepo{db: db}
}

func (r *OrderRepo) FindByID(ctx context.Context, id domain.OrderID) (*domain.Order, error) {
    row := r.db.QueryRowContext(ctx, `SELECT id, status, total FROM orders WHERE id = $1`, id)
    var o domain.Order
    if err := row.Scan(&o.ID, &o.Status, &o.Total); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, domain.ErrOrderNotFound
        }
        return nil, fmt.Errorf("scan order: %w", err)
    }
    return &o, nil
}

func (r *OrderRepo) Save(ctx context.Context, order *domain.Order) error {
    _, err := r.db.ExecContext(ctx,
        `INSERT INTO orders (id, status, total) VALUES ($1, $2, $3)
         ON CONFLICT (id) DO UPDATE SET status = $2, total = $3`,
        order.ID, order.Status, order.Total)
    return err
}
```

`domain` package **ไม่ import** `postgres` package เลย (dependency ชี้เข้าในตาม Clean Architecture) ในขณะที่ `postgres` package import `domain` เพื่อรู้จัก type `Order` — ทิศทาง import ตรงข้ามกับทิศทางที่ control flow วิ่งตอน runtime พอดี ซึ่งเป็นแก่นของ Dependency Inversion

ประโยชน์เชิงปฏิบัติ: เปลี่ยนจาก PostgreSQL ไป MongoDB ทำได้โดยเขียน adapter ใหม่ (`mongo.OrderRepo` ที่ implement `domain.OrderRepository` เหมือนกัน) โดย use case และ domain logic ไม่ต้องแก้เลยแม้แต่บรรทัดเดียว และเขียน `InMemoryOrderRepo` ปลอมสำหรับ test ได้ง่ายมาก (ดู [บทที่ 3.3](03-testing.md#33-mocking-ผ่าน-interface))

## 4.5 Dependency Injection

**Dependency Injection (DI)** คือการที่ object หนึ่งไม่สร้าง dependency ของตัวเอง แต่ **รับมันเข้ามาจากภายนอก** — วิธีที่นิยมที่สุดใน Go คือ **constructor injection** (ผ่าน function `NewXxx`) ตรงไปตรงมา ไม่ต้องพึ่ง annotation หรือ reflection แบบภาษา Java/C#:

```go
type CheckoutUseCase struct {
    orderRepo domain.OrderRepository
    gateway   domain.PaymentGateway
    logger    *slog.Logger
}

func NewCheckoutUseCase(
    orderRepo domain.OrderRepository,
    gateway domain.PaymentGateway,
    logger *slog.Logger,
) *CheckoutUseCase {
    return &CheckoutUseCase{orderRepo: orderRepo, gateway: gateway, logger: logger}
}

// ประกอบร่างทั้งหมดที่จุดเดียวใน main() — เรียกว่า "composition root"
func main() {
    db, _ := sql.Open("postgres", dsn)
    logger := slog.Default()

    orderRepo := postgres.NewOrderRepo(db)
    gateway := stripe.NewClient(apiKey)

    checkoutUC := usecase.NewCheckoutUseCase(orderRepo, gateway, logger)
    handler := http.NewOrderHandler(checkoutUC)

    // ...
}
```

`CheckoutUseCase` รู้จักแค่ interface (`domain.OrderRepository`, `domain.PaymentGateway`) ไม่รู้เลยว่า concrete implementation ที่ถูกส่งเข้ามาคืออะไร ทุก dependency ถูก "ประกอบ" เข้าด้วยกันที่จุดเดียวคือ `main()` (เรียกว่า **composition root**) — จุดนี้เป็นจุดเดียวในทั้งโปรเจกต์ที่รู้จัก concrete type ทุกตัว ส่วนที่เหลือทั้งหมดคุยกันผ่าน interface ล้วนๆ

### Manual DI vs Library

โปรเจกต์ Go ส่วนใหญ่ (โดยเฉพาะขนาดเล็กถึงกลาง) ใช้ **manual DI** แบบข้างบนตรงๆ เพราะ Go ไม่มี dependency graph ที่ซับซ้อนแบบ Spring/NestJS ให้ต้องพึ่ง container ผ่าน reflection — เขียนเองใน `main()` อ่านง่ายและ compile-time safe (ผิด type ตรงไหน compiler ฟ้องทันที)

เมื่อ dependency graph ใหญ่ขึ้นมาก (หลายสิบ component ที่ inject ซ้อนกันหลายชั้น) มีเครื่องมือช่วยลด boilerplate:

- **[Wire](https://github.com/google/wire)** (Google) — generate โค้ด wiring จริงตอน compile time จาก provider function ที่ประกาศไว้ (ไม่ใช้ reflection ตอน runtime เลย ยังคง compile-time safety ของ Go ไว้ครบ)
- **[fx](https://github.com/uber-go/fx)** (Uber) — DI container แบบ runtime ที่ทำงานร่วมกับ lifecycle ของแอป (start/stop hook) เหมาะกับแอปที่มี component จำนวนมากและต้องการจัดการ lifecycle อย่างเป็นระบบ

ทั้งสองตัวเป็นเครื่องมือเสริมที่ **ไม่เปลี่ยนหลักการ** — ยังคงเป็น constructor injection ผ่าน interface เหมือนเดิมทุกประการ เพียงแค่ช่วยลดโค้ด wiring ที่ซ้ำซากเมื่อ dependency graph ใหญ่ขึ้นมากเท่านั้น

## 4.6 HTTP Middleware

**Middleware** ใน Go คือฟังก์ชันที่ **ห่อ** (wrap) `http.Handler` ตัวหนึ่งด้วย `http.Handler` อีกตัว เพื่อแทรก logic ก่อน/หลังการทำงานของ handler จริง — pattern นี้ทำงานได้เพราะ `http.Handler` เป็น interface เดียว (`ServeHTTP(w, r)`) ทำให้ handler และ middleware มี "รูปร่าง" เดียวกัน ห่อกันเป็นชั้นๆ ได้ไม่จำกัด (chain-of-responsibility pattern):

```go
type Middleware func(http.Handler) http.Handler

// Logging middleware — บันทึกทุก request ที่เข้ามา พร้อมเวลาที่ใช้
func Logging(logger *slog.Logger) Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            next.ServeHTTP(w, r) // เรียก handler/middleware ชั้นถัดไป
            logger.Info("request",
                "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
        })
    }
}

// Auth middleware — ตรวจสอบ token ก่อนปล่อยให้ handler จริงทำงาน
func Auth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        userID, err := validateToken(token)
        if err != nil {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return // ไม่เรียก next.ServeHTTP — ตัด chain ทันที ไม่ให้ handler จริงทำงาน
        }
        ctx := context.WithValue(r.Context(), userIDKey{}, userID)
        next.ServeHTTP(w, r.WithContext(ctx)) // แนบ userID ไปกับ ctx ให้ handler ชั้นในใช้ต่อ
    })
}

// Recovery middleware — กัน panic ใน handler ไม่ให้ทำให้ทั้ง server ล่ม
func Recovery(logger *slog.Logger) Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            defer func() {
                if err := recover(); err != nil {
                    logger.Error("panic recovered", "error", err, "path", r.URL.Path)
                    http.Error(w, "internal server error", http.StatusInternalServerError)
                }
            }()
            next.ServeHTTP(w, r)
        })
    }
}

// Chain — ห่อ handler ด้วยหลาย middleware ตามลำดับ
func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
    for i := len(middlewares) - 1; i >= 0; i-- {
        h = middlewares[i](h)
    }
    return h
}

// ใช้งาน:
finalHandler := Chain(orderHandler,
    Recovery(logger), // นอกสุด — ต้องดัก panic จากทุกชั้นข้างในรวมถึง Auth/Logging เองด้วย
    Logging(logger),
    Auth,
)
```

ลำดับใน `Chain` สำคัญมาก: `Recovery` ต้องอยู่นอกสุดเสมอเพื่อดัก panic จากทุก middleware ชั้นในรวมถึง `Auth` และ handler จริงด้วย ส่วน `Auth` ควรอยู่ก่อน handler จริงเสมอเพื่อตัด request ที่ไม่ผ่านการยืนยันตัวตนก่อนที่ business logic จะทำงาน — การจัดลำดับ middleware ผิดเป็นหนึ่งใน bug ที่พบบ่อยที่สุดในโค้ด Go ที่เขียน HTTP server เอง (ไม่ใช้ framework) โดยเฉพาะการลืมว่า chain ทำงานแบบ "นอกสุดเรียกก่อน" (outer-to-inner ตอนเข้า, inner-to-outer ตอนกลับ — คล้าย stack)

Framework อย่าง `chi`, `gin`, `echo` ให้ syntax ที่สะดวกกว่านี้สำหรับการลงทะเบียน middleware (`router.Use(...)`) แต่กลไกภายในคือ pattern เดียวกันนี้ทั้งหมด

## 4.7 สรุป: ทำไม idiom ของ Go ถึงเข้ากับ pattern เหล่านี้เป็นธรรมชาติ

ทุก pattern ในบทนี้ (Clean Architecture, Hexagonal, Repository, DI, Middleware) พึ่งพากลไกภาษาเพียงสองอย่างของ Go เป็นแกนหลัก:

1. **Implicit interface implementation** — ทำให้ "port" ฝั่ง domain กับ "adapter" ฝั่ง infrastructure แยกจากกันได้จริงในระดับ package โดยไม่ต้อง import ข้ามกันแบบ circular และไม่ต้องมี base class หรือ annotation ใดๆ มาช่วยประกาศความสัมพันธ์ — แค่ method signature ตรงกันก็พอ
2. **ไม่มี inheritance บังคับให้ใช้ composition** — เพราะไม่มีทางลัดแบบ "extend คลาสแม่แล้วได้ implementation มาฟรี" นักออกแบบ Go จึงถูกบังคับให้คิดเรื่อง dependency ระหว่าง component อย่างชัดเจนตั้งแต่แรก (ผ่าน struct field ที่เป็น interface, ผ่าน constructor) ซึ่งบังเอิญ (หรือไม่บังเอิญ) ตรงกับสิ่งที่ Clean Architecture และ Hexagonal Architecture ต้องการเป๊ะ

นี่คือเหตุผลที่ชุมชน Go ไม่ค่อยต้องถกเถียงกันว่า "จะ implement Dependency Inversion อย่างไรให้ถูกภาษา" เหมือนภาษาอื่นบางภาษา — มันเป็นทางที่เป็นธรรมชาติที่สุดของภาษาอยู่แล้ว ตราบใดที่ยึดหลัก "นิยาม interface เล็กๆ ฝั่งผู้ใช้งาน แล้วให้ผู้ implement เป็นฝ่ายเข้ามาสอดคล้องเอง" อย่างสม่ำเสมอทั่วทั้งโปรเจกต์

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| Clean Architecture | Dependency rule: source code dependency ชี้เข้าในเสมอ, entity → use case → interface adapter → framework |
| DDD | Entity (มี identity), Value Object (เทียบด้วยค่า, immutable), Aggregate (หน่วยรักษา invariant ผ่าน root เดียว), Domain Service, Bounded Context |
| Hexagonal | Port = interface ที่ core นิยาม, Adapter = implementation นอก core, แยก driving vs driven adapter ชัดเจน |
| Repository | Interface อยู่ domain layer, implementation อยู่ infrastructure layer — สลับ database ได้โดยไม่แตะ business logic |
| Dependency Injection | Constructor injection ผ่าน interface, ประกอบทั้งหมดที่ composition root (`main()`), wire/fx ช่วยลด boilerplate เมื่อ graph ใหญ่ |
| Middleware | Chain-of-handlers ผ่าน `func(http.Handler) http.Handler`, ลำดับสำคัญมาก (Recovery นอกสุดเสมอ) |
| ทำไม Go เข้ากันดี | Implicit interface + ไม่มี inheritance บังคับให้ design ผ่าน composition/interface ตั้งแต่แรก |

## คำถามทบทวน

1. อธิบาย "The Dependency Rule" ของ Clean Architecture และยกตัวอย่างว่าใน Go มันถูก enforce ด้วยกลไกภาษาอะไร
2. ยกตัวอย่าง Entity, Value Object, และ Aggregate ในโดเมนอื่นที่ไม่ใช่ e-commerce (เช่น ระบบจองโรงแรม หรือระบบธนาคาร) พร้อมอธิบายว่าทำไมแต่ละตัวถึงจัดเป็นประเภทนั้น
3. Port กับ Adapter ใน Hexagonal Architecture ต่างกับ Interface กับ Implementation ทั่วไปอย่างไร หรือจริงๆ แล้วเป็นเรื่องเดียวกัน?
4. ทำไม `domain.OrderRepository` (interface) ถึงต้องอยู่ใน package `domain` ไม่ใช่ package `postgres`? ถ้าสลับกันจะเกิดปัญหาอะไร?
5. ในตัวอย่าง middleware chain ทำไม `Recovery` middleware ต้องอยู่นอกสุดเสมอ ถ้าสลับให้ `Auth` อยู่นอกสุดแทนจะเกิดอะไรขึ้นถ้า `Auth` เอง panic?

---

ก่อนหน้า: [บทที่ 3 — Testing](03-testing.md) | กลับสู่ [สารบัญ Volume 4](README.md)
