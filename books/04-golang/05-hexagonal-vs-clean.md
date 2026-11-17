# บทเสริม: Hexagonal vs Clean Architecture

บทนี้สรุปความต่างของสองแบบให้อ่านจบได้ในไม่กี่นาที รายละเอียดของแต่ละแบบอยู่ใน [บทที่ 4](04-architecture-patterns.md) หัวข้อ 4.1 (Clean) และ 4.3 (Hexagonal)

## สรุปในประโยคเดียว

**ทั้งสองแบบมีแก่นเดียวกัน** คือแยก business logic ออกจาก infrastructure (DB, HTTP, Kafka) และให้ dependency ชี้เข้าหา business logic เสมอ ที่ต่างกันคือ**วิธีมอง**และ**ความละเอียดของการแบ่งชั้น**

> **Clean Architecture ≈ Hexagonal + แบ่งข้างใน core เพิ่มเป็น Entities กับ Use Cases**

## เปรียบเทียบกับร้านอาหาร

### 🍜 Hexagonal: ครัวที่มี "ช่อง" มาตรฐาน

ลองนึกถึงร้านก๋วยเตี๋ยวที่รับออเดอร์ได้หลายทาง

```text
   รับออเดอร์ (ฝั่งที่เรียกเรา)                           สั่งของ (ฝั่งที่เราเรียก)

   ลูกค้าหน้าร้าน ─┐                                     ┌─ Makro
   Grab          ─┼─► [ช่องรับออเดอร์] ► ครัว ► [ช่องสั่งวัตถุดิบ] ─┼─ ตลาดสด
   LINE MAN      ─┘                                     └─ ซัพพลายเออร์เจ้าใหม่
```

กฎของร้านนี้มีข้อเดียว: **ครัวไม่ต้องรู้ว่าออเดอร์มาจากไหน และไม่ต้องรู้ว่าวัตถุดิบมาจากเจ้าไหน**

- ทุกช่องทางต้องแปลงออเดอร์เป็น**ใบสั่งแบบเดียวกัน**ก่อนส่งเข้าครัว นี่คือ **Port**
- Grab, LINE MAN และพนักงานหน้าร้าน คือคนแปลงออเดอร์ให้เป็นใบสั่งนั้น นี่คือ **Adapter**
- วันไหนเพิ่ม ShopeeFood ก็แค่เพิ่ม adapter ตัวใหม่ ครัวไม่ต้องเปลี่ยนอะไรเลย

แต่ Hexagonal **ไม่ได้บอกว่าข้างในครัวต้องจัดยังไง** ครัวจะวุ่นแค่ไหนก็ได้ ขอแค่ "ช่อง" เป็นมาตรฐาน

### 🏢 Clean: ใช้ "ช่อง" เหมือนกัน แต่จัดระเบียบข้างในครัวด้วย

Clean Architecture เห็นด้วยกับเรื่องช่องทุกข้อ แล้ว**แบ่งข้างในครัวเพิ่มอีก 2 ชั้น**

| ชั้น | ในร้านอาหาร | ตัวอย่าง |
|---|---|---|
| **Entities** (ในสุด) | **สูตรอาหารและกฎของร้าน**: ไม่เปลี่ยนไม่ว่าจะขายช่องทางไหน | ต้มยำต้องเปรี้ยวเผ็ด, เมนูฮาลาลห้ามมีหมู, ชามใหญ่ต้องมีเส้น 150 กรัม |
| **Use Cases** | **ขั้นตอนการทำงานของร้าน** | รับออเดอร์ → เช็คว่าวัตถุดิบพอไหม → ปรุงตามสูตร → แจ้งว่าอาหารเสร็จ |
| **Interface Adapters** | **พนักงานที่แปลงงาน** | พนักงานรับออเดอร์จาก Grab, คนจดสต็อกลงสมุด |
| **Frameworks & Drivers** (นอกสุด) | **โลกภายนอก** | แอป Grab, Makro, เครื่อง POS |

**กฎสำคัญ:** คนในชั้นในไม่รู้จักชั้นนอก สูตรต้มยำไม่มีคำว่า "Grab" อยู่ในนั้น ขั้นตอนของร้านไม่ระบุชื่อ "Makro"

### 🔍 ลองดูว่าเปลี่ยนอะไรแล้วกระทบส่วนไหน

| เหตุการณ์ | Hexagonal แก้ที่ | Clean แก้ที่ |
|---|---|---|
| เพิ่ม ShopeeFood | adapter ใหม่ | adapter ใหม่ (**เหมือนกัน**) |
| เปลี่ยนจาก Makro เป็นตลาดสด | adapter ฝั่งวัตถุดิบ | adapter (**เหมือนกัน**) |
| เปลี่ยนขั้นตอนจาก "จ่ายก่อนทำ" เป็น "ทำก่อนจ่าย" | ใน "ครัว" (ปนอยู่กับสูตร) | **Use Case** อย่างเดียว สูตรไม่โดนแตะ |
| เปลี่ยนสูตรต้มยำให้เผ็ดน้อยลง | ใน "ครัว" (ปนอยู่กับขั้นตอน) | **Entity** อย่างเดียว ขั้นตอนไม่โดนแตะ |
| เปิดสาขาสองที่ใช้ขั้นตอนต่างกัน (มีหน้าร้าน กับ delivery อย่างเดียว) | ต้องระวังว่า copy สูตรไปสองที่ | Use Case สองตัว **ใช้ Entity ชุดเดียวกัน** |

สองแถวแรก**เหมือนกันทุกอย่าง** เพราะทั้งสองแบบแยกโลกภายนอกออกไปแล้ว ส่วนสามแถวล่างเป็นจุดที่ Clean เก่งกว่า เพราะแยก "สูตร" ออกจาก "ขั้นตอน"

## ภาพในหัว

### Hexagonal (Ports & Adapters): แค่ "ข้างใน" กับ "ข้างนอก"

```text
  Driving (ฝั่งที่เรียกเรา)                           Driven (ฝั่งที่เราเรียก)

  HTTP handler ─┐                                ┌─ PostgreSQL adapter
  gRPC handler ─┼──► [Port] ──► CORE ──► [Port] ─┼─ Kafka adapter
  CLI / test   ─┘                                └─ Mock (test)
```

- **Core** = business logic ทั้งหมด ข้างในจะจัดยังไงก็ได้ แนวคิดนี้ไม่ได้กำหนดไว้
- **Port** = interface ที่ core ประกาศว่า "ฉันต้องการสิ่งนี้" (driven) หรือ "ฉันให้บริการสิ่งนี้" (driving)
- **Adapter** = ตัว implement port ด้วยเทคโนโลยีจริง

### Clean Architecture: วงกลมซ้อน 4 ชั้น

```text
┌──────────────────────────────────────────────┐
│ Frameworks & Drivers   (DB, web, Kafka)      │
│ ┌──────────────────────────────────────────┐ │
│ │ Interface Adapters  (handler, repo impl) │ │
│ │ ┌──────────────────────────────────────┐ │ │
│ │ │ Use Cases       (application logic)  │ │ │
│ │ │ ┌──────────────────────────────────┐ │ │ │
│ │ │ │ Entities        (domain rules)   │ │ │ │
│ │ │ └──────────────────────────────────┘ │ │ │
│ │ └──────────────────────────────────────┘ │ │
│ └──────────────────────────────────────────┘ │
└──────────────────────────────────────────────┘
      Dependency Rule: ชี้เข้าข้างในเท่านั้น
```

## ตารางเทียบ

| | Hexagonal | Clean |
|---|---|---|
| ผู้เสนอ / ปี | Alistair Cockburn, 2005 | Robert C. Martin (Uncle Bob), 2012 |
| ภาพในหัว | ข้างใน กับ ข้างนอก (2 ฝั่ง) | วงกลมซ้อน 4 ชั้น |
| แบ่งข้างใน core ไหม | **ไม่** ปล่อยให้ตัดสินใจเอง | **แบ่ง** เป็น Entities และ Use Cases |
| คำศัพท์หลัก | Port, Adapter, Driving, Driven | Entity, Use Case, Interface Adapter, Dependency Rule |
| จุดเน้น | **ความสมมาตร**: ทุกอย่างข้างนอกเสียบผ่าน port เหมือนกัน สลับ adapter ได้ | **การแบ่งหน้าที่ข้างใน**: กฎธุรกิจหลักแยกจาก flow ของแอป |
| ความซับซ้อน | น้อยกว่า | มากกว่า (ชั้นและไฟล์เยอะกว่า) |

## ตัวอย่างโค้ด Go: ระบบสร้าง Task

### Hexagonal

```go
// internal/core/task.go — core ทั้งหมดอยู่ที่เดียว
type Task struct {
	ID, Title string
	Done      bool
}

// Driven port: core ประกาศว่าต้องการที่เก็บข้อมูล (ไม่สนว่าเป็น Postgres หรืออะไร)
type TaskRepository interface {
	Save(ctx context.Context, t Task) error
}

// Driving port + business logic อยู่ในก้อนเดียวกัน
type TaskService struct{ repo TaskRepository }

func (s *TaskService) Create(ctx context.Context, title string) (Task, error) {
	if len(title) < 3 { // กฎธุรกิจอยู่ใน service
		return Task{}, errors.New("title too short")
	}
	t := Task{ID: uuid.NewString(), Title: title}
	return t, s.repo.Save(ctx, t)
}
```

```text
internal/
├── core/              ← Task, TaskRepository (port), TaskService
└── adapter/
    ├── http/          ← driving adapter: เรียก TaskService
    └── postgres/      ← driven adapter: implement TaskRepository
```

### Clean

```go
// internal/domain/task.go — Entity: กฎที่จริงเสมอ ไม่ว่าแอปไหนใช้
type Task struct {
	ID, Title string
	Done      bool
}

func NewTask(id, title string) (Task, error) {
	if len(title) < 3 { // กฎของ domain อยู่กับ entity
		return Task{}, ErrTitleTooShort
	}
	return Task{ID: id, Title: title}, nil
}
```

```go
// internal/usecase/create_task.go — Use Case: flow ของแอปนี้
type TaskRepository interface { // interface ประกาศในชั้น usecase (ชั้นใน)
	Save(ctx context.Context, t domain.Task) error
}

type CreateTask struct {
	repo  TaskRepository
	idGen func() string
}

func (uc *CreateTask) Execute(ctx context.Context, title string) (domain.Task, error) {
	t, err := domain.NewTask(uc.idGen(), title)
	if err != nil {
		return domain.Task{}, err
	}
	return t, uc.repo.Save(ctx, t)
}
```

```text
internal/
├── domain/            ← Entities (ไม่ import ใครเลย)
├── usecase/           ← Use Cases + interface ของ repository
├── adapter/http/      ← Interface Adapter: แปลง HTTP เป็น input ของ use case
└── repository/        ← Interface Adapter: implement repository ด้วย Postgres
```

**ความต่างที่เห็นในโค้ด:** กฎ "title ยาวอย่างน้อย 3 ตัว" ใน Hexagonal อยู่ใน service ส่วนใน Clean ถูกแยกไปอยู่ที่ **entity** แล้ว use case แค่เรียกใช้

## ตัวอย่างที่เห็นความต่างชัดกว่า: ระบบโอนเงิน

ตัวอย่าง Task ข้างบนมีกฎแค่ข้อเดียว เลยยังเห็นความต่างไม่ชัด ระบบโอนเงินมีกฎธุรกิจเยอะกว่า จึงเห็นได้ชัดกว่า

- **กฎ (Entity):** ยอดเงินห้ามติดลบ, โอนได้ไม่เกิน 50,000 บาทต่อครั้ง, บัญชีที่ถูกระงับห้ามโอน
- **ขั้นตอน (Use Case):** โหลดบัญชีทั้งสอง → ตัดเงิน → เพิ่มเงิน → บันทึก → ส่ง event แจ้งเตือน

### แบบ Hexagonal: กฎกับขั้นตอนอยู่ด้วยกันใน core

```go
// internal/core/transfer_service.go
func (s *TransferService) Transfer(ctx context.Context, from, to string, amount int64) error {
	src, _ := s.accounts.Get(ctx, from) // port → adapter Postgres
	dst, _ := s.accounts.Get(ctx, to)

	// กฎกับขั้นตอนปนกันในฟังก์ชันเดียว
	if src.Frozen {
		return ErrFrozen
	}
	if amount > 50_000 {
		return ErrLimit
	}
	if src.Balance < amount {
		return ErrInsufficient
	}
	src.Balance -= amount
	dst.Balance += amount

	s.accounts.Save(ctx, src, dst)
	s.notifier.Send(ctx, from, "โอนสำเร็จ") // port → adapter Kafka/LINE
	return nil
}
```

ใช้ได้ดี อ่านง่าย แต่ถ้ามีช่องทาง "ถอนเงินที่ตู้ ATM" เพิ่มเข้ามา จะต้องเขียน `if src.Frozen`, `if src.Balance < amount` **ซ้ำ**อีกรอบ

### แบบ Clean: กฎย้ายไปอยู่กับ Entity

```go
// internal/domain/account.go — สูตร/กฎ (ไม่รู้จัก DB, HTTP, Kafka)
func (a *Account) Withdraw(amount int64) error {
	if a.Frozen {
		return ErrFrozen
	}
	if amount > 50_000 {
		return ErrLimit
	}
	if a.Balance < amount {
		return ErrInsufficient
	}
	a.Balance -= amount
	return nil
}

func (a *Account) Deposit(amount int64) { a.Balance += amount }
```

```go
// internal/usecase/transfer.go — ขั้นตอน
func (uc *Transfer) Execute(ctx context.Context, from, to string, amount int64) error {
	src, _ := uc.accounts.Get(ctx, from)
	dst, _ := uc.accounts.Get(ctx, to)

	if err := src.Withdraw(amount); err != nil { // เรียกกฎ ไม่เขียนกฎเอง
		return err
	}
	dst.Deposit(amount)

	uc.accounts.Save(ctx, src, dst)
	uc.notifier.Send(ctx, from, "โอนสำเร็จ")
	return nil
}

// internal/usecase/atm_withdraw.go — ขั้นตอนใหม่ ใช้กฎชุดเดิม
func (uc *ATMWithdraw) Execute(ctx context.Context, id string, amount int64) error {
	acc, _ := uc.accounts.Get(ctx, id)
	if err := acc.Withdraw(amount); err != nil { // ไม่ต้องเขียนกฎซ้ำ
		return err
	}
	uc.accounts.Save(ctx, acc)
	return uc.atm.Dispense(ctx, amount)
}
```

พอเปลี่ยนวงเงินจาก 50,000 เป็น 100,000 แก้ **ที่เดียว** ใน `Withdraw` แล้วทั้งการโอนและการถอนจาก ATM ได้กฎใหม่พร้อมกัน

> ตัวอย่างนี้ละ error ของ `Get`/`Save` ไว้ (`_`) และยังไม่มี transaction เพื่อให้เห็นโครงชัดๆ ในโค้ดจริงต้องจัดการ error ทุกจุด และตัด/เพิ่มเงินภายใน DB transaction เดียวกัน

## สรุปให้จำง่าย

| | ประโยคเดียว | เทียบร้านอาหาร |
|---|---|---|
| **Hexagonal** | "ข้างนอกต่อเข้ามาผ่านช่องมาตรฐาน" | ครัวมีช่องรับออเดอร์และช่องสั่งของ ข้างในจัดยังไงก็ได้ |
| **Clean** | "ช่องมาตรฐาน + ข้างในแยกกฎออกจากขั้นตอน" | เหมือนกัน และแยก**สูตรอาหาร**ออกจาก**ขั้นตอนของร้าน** |

- **ร้านก๋วยเตี๋ยวรถเข็น** (CRUD, service เล็ก) ใช้ Hexagonal ก็พอ แยกสูตรเพิ่มก็เปลืองแรงเปล่าๆ
- **ร้านอาหารเครือใหญ่หลายสาขา** (ธนาคาร ประกัน ระบบ order ที่กฎซับซ้อนและใช้ซ้ำหลายที่) ควรแยกแบบ Clean

## สิ่งที่เหมือนกัน (สำคัญกว่าความต่าง)

1. **Interface ประกาศที่ฝั่งผู้ใช้** (core/usecase) ไม่ใช่ฝั่งที่ implement ตรงกับสำนวนของ Go *"accept interfaces, return structs"* และ implicit interface ของ Go ทำให้เป็นธรรมชาติมาก
2. **Business logic ไม่ import** `database/sql`, `net/http` หรือ library ของ Kafka
3. **Test ง่าย** ใช้ mock repository แทน Postgres ได้ทันที
4. **ประกอบทุกอย่างใน `main.go`** (manual dependency injection, ดู [4.5](04-architecture-patterns.md))

## ในงานจริง

- ทีมส่วนใหญ่**ผสมกัน** ใช้โครงโฟลเดอร์แบบ Clean (`domain`, `usecase`, `repository`) แต่เรียกตัวที่ต่อกับภายนอกว่า `adapter` แบบ Hexagonal
- ตัวอย่างใน workspace นี้: [projects/bot-orchestration-platform](../../projects/bot-orchestration-platform/README.md) ใช้แบบผสม
  - `internal/domain/` (bot, job, schedule, workflow) = Entities
  - `internal/usecase/` (auth, job, schedule, workflowrun) = Use Cases
  - `internal/adapter/http/` = driving adapter
  - `internal/repository/memory/`, `internal/external/` (temporal, k8s, bots) = driven adapters
- **เลือกยังไง:**
  - service เล็กหรือ CRUD → Hexagonal (หรือแค่ handler → service → repo) ก็พอ
  - ระบบที่มีกฎธุรกิจซับซ้อน (การเงิน ประกัน order) → แยก domain แบบ Clean
  - ถ้าแบ่งชั้นมากเกินจำเป็น จะได้ไฟล์เยอะโดยไม่ได้ประโยชน์อะไร

## ถ้าโดนถามในสัมภาษณ์

> "ทั้งสองแบบมีเป้าหมายเดียวกันคือแยก business logic ออกจาก infrastructure โดยให้ dependency ชี้เข้าหา core Hexagonal มองระบบเป็นข้างในกับข้างนอกที่เชื่อมกันผ่าน port และ adapter โดยไม่ได้กำหนดว่าข้างในต้องจัดยังไง ส่วน Clean Architecture แบ่งข้างในเพิ่มเป็น entity กับ use case และมี dependency rule ชัดเจน ใน Go ทั้งสองแบบทำได้ง่ายเพราะ interface เป็น implicit เลยประกาศ interface ไว้ที่ฝั่ง core ได้ โปรเจกต์ที่ผมทำใช้แบบผสม ..."

แล้วต่อด้วยตัวอย่างจากโปรเจกต์จริง เช่น ถ้าเปลี่ยน repository จาก memory เป็น Postgres ต้องแก้แค่ adapter กับ `main.go` ส่วน usecase ไม่ต้องแตะเลย

## คำถามทบทวน

1. ถ้าต้องเปลี่ยนจาก PostgreSQL เป็น MongoDB ต้องแก้ไฟล์ในชั้นไหนบ้าง (ตอบทั้งสองแบบ)
2. ทำไม interface `TaskRepository` ถึงประกาศใน `usecase` แทนที่จะอยู่ใน `repository`
3. HTTP handler เป็น driving หรือ driven adapter และ Kafka consumer กับ Kafka producer อยู่ฝั่งไหน
4. กฎธุรกิจแบบไหนควรอยู่ใน entity และแบบไหนควรอยู่ใน use case
5. ในระบบโอนเงิน ถ้าเพิ่มกฎ "โอนต่างธนาคารมีค่าธรรมเนียม 10 บาท" ควรอยู่ใน `Account.Withdraw` หรือใน use case `Transfer` เพราะอะไร (คำใบ้: กฎนี้ใช้กับการถอน ATM ด้วยไหม)
