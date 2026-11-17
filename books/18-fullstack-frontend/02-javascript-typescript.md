# บทที่ 2: JavaScript & TypeScript สำหรับเบราว์เซอร์

บทนี้ไม่สอน syntax พื้นฐาน (`if`, `for`, function อ่านออกอยู่แล้วจากภาษาอื่น) แต่เจาะจุดที่ JavaScript **ต่างจาก Go** จนทำให้ backend engineer เขียน bug โดยไม่รู้ตัว และ feature สมัยใหม่ที่โค้ด React ใช้ทุกบรรทัด

## 2.1 JavaScript ต่างจาก Go ตรงไหน

### ตัวแปร: `const` / `let` (ห้ามใช้ `var`)

```js
const name = "task";   // ใช้เป็นค่า default — reassign ไม่ได้
let count = 0;         // ใช้เมื่อต้อง reassign จริงๆ
count++;

const arr = [1, 2];
arr.push(3);           // ได้! const ห้ามแค่ "ผูกตัวแปรใหม่" ไม่ได้ห้ามแก้ข้างใน
// arr = [];           // error
```

`const` ของ JS **ไม่ใช่** immutable — มันแค่ห้าม reassign ตัวแปร ต่างจาก `const` ของ Go ที่เป็นค่าคงที่ตอน compile

### Type แบบ dynamic และการแปลงค่าอัตโนมัติ

```js
1 + "2"        // "12"  (number ถูกแปลงเป็น string)
"5" - 2        // 3     (string ถูกแปลงเป็น number)
0 == ""        // true  ← == แปลง type ก่อนเทียบ
0 === ""       // false ← === ไม่แปลง
null == undefined   // true
```

**กฎ: ใช้ `===` และ `!==` เสมอ** (ESLint จะเตือนให้)

### Truthy / Falsy

ใน `if` ค่าต่อไปนี้ถือเป็น false: `false`, `0`, `""`, `null`, `undefined`, `NaN` นอกนั้น true ทั้งหมด (รวม `[]` และ `{}` ว่าง!)

```js
if (tasks.length) { ... }        // ใช้บ่อย: มีของ = true
const title = input || "Untitled";    // || ใช้ค่าขวาเมื่อซ้าย falsy (รวม 0 และ "")
const limit = userLimit ?? 20;        // ?? ใช้ค่าขวาเมื่อซ้ายเป็น null/undefined เท่านั้น
// userLimit = 0 → || ได้ 20 (ผิด!), ?? ได้ 0 (ถูก)
```

### `null` และ `undefined`

Go มี zero value; JS มีค่า "ไม่มี" 2 แบบ: `undefined` (ไม่ได้กำหนดค่า, field ไม่มีใน object) และ `null` (ตั้งใจให้ว่าง) ในทางปฏิบัติจัดการเหมือนกันด้วย `??` และ `?.`

### Object และ Array เป็น reference เสมอ

```js
const a = { title: "x" };
const b = a;          // ไม่ได้ copy! b ชี้ไปที่ object เดียวกัน (เหมือน pointer ใน Go)
b.title = "y";
console.log(a.title); // "y"

const c = { ...a };   // shallow copy — object ใหม่ (field ระดับแรก)
c.title = "z";        // a ไม่เปลี่ยน
```

นี่คือรากของกฎ **immutability** ใน React (บทที่ 3): แก้ state ต้องสร้าง object/array ใหม่ ไม่ใช่แก้ของเดิม

### Number มีแบบเดียว

JS มี `number` แบบเดียว (float64) ไม่มี `int`/`int64` แยก ระวังเลขเกิน 2^53 (เช่น ID แบบ int64 จาก Go หรือ snowflake ID) จะเพี้ยน → ให้ Go ส่ง ID ใหญ่ๆ เป็น string (`json:"id,string"`)

## 2.2 Function, Scope, Closure

```js
// 3 แบบที่เจอ — ใน React ใช้ arrow function เป็นหลัก
function add(a, b) { return a + b; }
const add2 = (a, b) => a + b;              // arrow: body บรรทัดเดียว return อัตโนมัติ
const toObj = (id) => ({ id });            // คืน object ต้องครอบวงเล็บ

// default parameter
const greet = (name = "guest") => `Hello ${name}`;   // template string ใช้ backtick
```

**Closure** — function จำตัวแปรจาก scope ที่สร้างมันไว้ (เหมือน closure ใน Go):

```js
function makeCounter() {
  let n = 0;
  return () => ++n;     // function นี้ "จับ" n ไว้
}
const next = makeCounter();
next(); // 1
next(); // 2
```

ใน React closure อยู่ทุกที่ — event handler ทุกตัวคือ closure ที่จับค่า state ของ render รอบนั้นไว้ (บทที่ 3 หัวข้อ 3.7 "stale closure")

**`this`:** ใน JS สมัยใหม่ (React function component) แทบไม่ต้องใช้ `this` เลย ถ้าเจอในโค้ดเก่า (class component) ให้รู้ว่าค่ามันขึ้นกับ "ใครเรียก" ไม่ใช่ "ประกาศที่ไหน" และ arrow function ไม่มี `this` ของตัวเอง

## 2.3 Syntax สมัยใหม่ที่ React ใช้ทุกบรรทัด

```js
const task = { id: 1, title: "Learn", status: "todo", meta: { tags: ["fe"] } };

// Destructuring
const { title, status } = task;
const { title: t } = task;                 // เปลี่ยนชื่อ
const [first, second] = ["a", "b"];        // array — useState ใช้แบบนี้: const [x, setX] = useState()

// Spread — copy/รวม
const updated = { ...task, status: "done" };          // copy แล้วทับ field
const tasks2 = [...tasks, newTask];                   // เพิ่มท้าย
const withoutFirst = tasks.slice(1);

// Optional chaining — เข้าถึง field ที่อาจไม่มี โดยไม่ crash
task.meta?.tags?.[0];      // "fe" หรือ undefined (ไม่ throw)
user?.profile?.name ?? "Anonymous";

// Shorthand
const id = 1;
const obj = { id };        // = { id: id }
```

### Array methods (ใช้แทน for loop เกือบทั้งหมด)

ทุกตัว **คืน array ใหม่** ไม่แก้ของเดิม (ยกเว้น `sort`, `push`, `splice`) — นี่คือเหตุผลที่ React ชอบมัน

```js
const jobs = [
  { id: 1, bot: "a", status: "failed",    ms: 120 },
  { id: 2, bot: "b", status: "succeeded", ms: 80 },
  { id: 3, bot: "a", status: "succeeded", ms: 200 },
];

jobs.filter((j) => j.status === "failed");          // เลือก → [{id:1...}]
jobs.map((j) => j.id);                              // แปลง  → [1, 2, 3]
jobs.find((j) => j.id === 2);                       // ตัวแรกที่ตรง หรือ undefined
jobs.some((j) => j.status === "failed");            // มีสักตัว → true
jobs.every((j) => j.ms < 500);                      // ทุกตัว → true
jobs.reduce((sum, j) => sum + j.ms, 0);             // รวม → 400

// นับตามสถานะ (group by)
const byStatus = jobs.reduce((acc, j) => {
  acc[j.status] = (acc[j.status] ?? 0) + 1;
  return acc;
}, {});   // { failed: 1, succeeded: 2 }

// sort แก้ array เดิม! copy ก่อนเสมอ (หรือใช้ toSorted ใน JS ใหม่)
const sorted = [...jobs].sort((a, b) => b.ms - a.ms);
```

> **ดูของจริง:** `projects/bop-dashboard/src/entities/job/lib/job-stats.ts` ใช้ `filter` / `slice().sort()` / `slice(0, 5)` แบบนี้ทั้งไฟล์

### Module (import/export)

```js
// math.js
export function add(a, b) { return a + b; }    // named export
export default function sub(a, b) { ... }     // default export (ไฟล์ละ 1)

// app.js
import sub, { add } from "./math.js";
```

เทียบกับ Go: ไฟล์หนึ่ง = module หนึ่ง (ไม่ใช่ทั้ง package) และต้อง `export` ชัดเจน (Go ใช้ตัวพิมพ์ใหญ่)

## 2.4 Event Loop, Promise, async/await, fetch

### JS มี thread เดียว

Go มี goroutine หลายพันตัวพร้อมกัน JS ในเบราว์เซอร์รันโค้ดของเราบน **thread เดียว** งาน I/O (network, timer) ถูกส่งให้ browser ทำ แล้วเมื่อเสร็จ callback จะเข้า **คิว** รอ thread ว่าง

```text
 Call stack (โค้ดเรา ทำทีละอย่าง)
      │ fetch(...) ──► Browser ทำ network ให้ (ไม่ block)
      │ ทำงานอื่นต่อ
      ▼
 stack ว่าง ◄── Event loop หยิบ callback จากคิวมารัน ◄── network เสร็จ
```

ผลที่ตามมา: **ห้ามมีโค้ดที่ CPU หนักนาน** (เช่น loop ล้านรอบ) เพราะจะ block ทั้งหน้า — กดปุ่มไม่ได้ scroll ไม่ได้

### Promise และ async/await

`Promise` = ค่าที่จะมาในอนาคต (คล้าย channel ที่ส่งค่าเดียว) `async/await` ทำให้เขียนเหมือนโค้ด synchronous

```ts
async function loadTasks() {
  try {
    const res = await fetch("http://localhost:8080/tasks");

    // ⚠️ fetch ไม่ throw เมื่อ server ตอบ 404/500 — throw แค่ตอน network ล่ม/CORS block
    if (!res.ok) {
      throw new Error(`HTTP ${res.status}: ${await res.text()}`);
    }
    return await res.json();      // parse body ก็เป็น async
  } catch (err) {
    console.error("load failed", err);
    throw err;
  }
}

// POST JSON
await fetch("http://localhost:8080/tasks", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ title: "Learn React" }),
});

// ทำพร้อมกัน (เหมือน errgroup.Wait — ตัวใดพัง ทั้งหมด reject)
const [tasks, user] = await Promise.all([loadTasks(), loadMe()]);

// ยกเลิก request (เหมือน context.WithCancel)
const controller = new AbortController();
fetch(url, { signal: controller.signal });
controller.abort();
```

ข้อผิดพลาดที่พบบ่อย: ลืม `await` → ได้ `Promise { <pending> }` แทนข้อมูล, ลืมเช็ค `res.ok` → พยายาม parse หน้า error HTML เป็น JSON

## 2.5 DOM และ Event

DOM คือ tree ของ object ที่ browser สร้างจาก HTML — JS อ่าน/แก้ tree นี้แล้วหน้าจอเปลี่ยนตาม

```ts
const list = document.querySelector<HTMLUListElement>("#tasks")!;
const form = document.querySelector<HTMLFormElement>("#task-form")!;
const input = document.querySelector<HTMLInputElement>("#title")!;

let tasks: { id: number; title: string }[] = [];

function render() {
  // วิธีบ้านๆ: สร้าง HTML ใหม่ทั้งหมดทุกครั้ง (React ทำสิ่งนี้ให้แบบฉลาดกว่า)
  list.innerHTML = tasks
    .map((t) => `<li data-id="${t.id}">${t.title} <button class="del">x</button></li>`)
    .join("");
}

form.addEventListener("submit", (e) => {
  e.preventDefault();                       // กันไม่ให้ browser submit แบบโหลดหน้าใหม่
  tasks = [...tasks, { id: Date.now(), title: input.value }];
  input.value = "";
  render();
});

// Event delegation: ฟัง event ที่ parent ตัวเดียว แทนการผูกทุกปุ่ม
list.addEventListener("click", (e) => {
  const target = e.target as HTMLElement;
  if (target.matches(".del")) {
    const id = Number(target.closest("li")!.dataset.id);
    tasks = tasks.filter((t) => t.id !== id);
    render();
  }
});
```

⚠️ `innerHTML` กับข้อความจาก user = ช่องโหว่ **XSS** (ถ้า title คือ `<img src=x onerror=alert(1)>`) — React escape ให้อัตโนมัติ นี่คืออีกเหตุผลที่ใช้ framework

ลองทำแบบนี้สัก 1 วัน (W2·D2) จะเข้าใจว่าทำไม React ถึงเกิดขึ้น: ทุกครั้งที่ข้อมูลเปลี่ยนต้องจำว่าต้องอัปเดต DOM ตรงไหน — พอแอปใหญ่ขึ้นจะคุมไม่อยู่

## 2.6 TypeScript พื้นฐาน

TypeScript = JavaScript + type ที่ตรวจตอน compile แล้ว**ลบทิ้งทั้งหมด**ตอน runtime (ต่างจาก Go ที่ type อยู่ใน binary)

```ts
// primitive
let title: string = "x";
let done: boolean = false;
let count = 0;                 // infer เป็น number เอง — ไม่ต้องเขียน type ถ้าชัดอยู่แล้ว

// object: interface หรือ type ก็ได้ (ใช้แทนกันได้ 95%)
interface Task {
  id: number;
  title: string;
  status: TaskStatus;
  description?: string;        // optional = string | undefined
  readonly createdAt: string;  // แก้ไม่ได้
}

// Union + literal type — ทรงพลังมาก ใช้แทน enum
type TaskStatus = "todo" | "doing" | "done";

// array
const tasks: Task[] = [];

// function
function findTask(tasks: Task[], id: number): Task | undefined {
  return tasks.find((t) => t.id === id);
}
```

### Structural typing — เหมือน interface ของ Go

```ts
interface HasId { id: number }
function printId(x: HasId) { console.log(x.id); }

printId({ id: 1, title: "anything" } as Task);  // ได้ — มี id ก็พอ ไม่ต้องประกาศ implements
```

### Narrowing — TypeScript ฉลาดตามเงื่อนไข

```ts
function label(task: Task | undefined) {
  if (!task) return "not found";      // หลังบรรทัดนี้ TS รู้ว่า task เป็น Task แน่นอน
  return task.title;
}

// Discriminated union — pattern ที่ใช้บ่อยมากกับ state ของ UI
type LoadState =
  | { kind: "loading" }
  | { kind: "error"; message: string }
  | { kind: "success"; data: Task[] };

function render(s: LoadState) {
  switch (s.kind) {
    case "loading": return "Loading...";
    case "error":   return s.message;        // TS รู้ว่ามี message
    case "success": return `${s.data.length} tasks`;
  }
}
```

Discriminated union คือสิ่งที่ Go ไม่มี (Go ต้องใช้ interface + type switch) ใช้ให้เป็น จะกำจัด state ที่เป็นไปไม่ได้ (เช่น loading=true และ error≠null พร้อมกัน)

### `Record` สำหรับ map

```ts
const toneByStatus: Record<TaskStatus, "gray" | "yellow" | "green"> = {
  todo: "gray",
  doing: "yellow",
  done: "green",
};   // ลืมใส่ key ไหน หรือใส่ key เกิน → compile error
```

> **ดูของจริง:** `projects/bop-dashboard/src/entities/job/ui/job-status-badge.tsx` ใช้ `Record<JobStatus, BadgeTone>` แบบนี้ — เพิ่มสถานะใหม่ใน type แล้ว compiler บังคับให้กำหนดสีด้วย

## 2.7 Generic, Utility Type, `unknown`

```ts
// Generic — เหมือน type parameter ของ Go 1.18+
async function apiGet<T>(path: string): Promise<T> {
  const res = await fetch(`http://localhost:8080${path}`);
  if (!res.ok) throw new Error(String(res.status));
  return (await res.json()) as T;        // ⚠️ as T คือ "เชื่อฉัน" — ไม่มีการตรวจจริง
}
const tasks = await apiGet<Task[]>("/tasks");   // tasks: Task[]

// Utility types ที่ใช้บ่อย
type NewTask   = Omit<Task, "id" | "createdAt">;   // ตัด field ออก — body ของ POST
type TaskPatch = Partial<Pick<Task, "title" | "status">>;  // เลือกบาง field และทำเป็น optional — body ของ PATCH
```

### `any` vs `unknown`

- `any` = ปิด type check ทั้งหมด (ห้ามใช้ ถ้าใช้คือเสียประโยชน์ของ TS)
- `unknown` = "ไม่รู้ว่าเป็นอะไร **ต้องตรวจก่อนใช้**" — ใช้กับข้อมูลจากภายนอก (API, `JSON.parse`, `catch (err)`)

```ts
try { ... } catch (err: unknown) {
  const message = err instanceof Error ? err.message : String(err);
}
```

**ปัญหาสำคัญ:** `as Task[]` ไม่ได้ตรวจว่า API ส่ง shape ถูกจริง ถ้า backend เปลี่ยน field ชื่อ TypeScript ไม่รู้เลย แก้ได้ 2 ทาง: (1) validate ตอน runtime ด้วย **zod** (บทที่ 4), (2) generate type จาก OpenAPI spec ของ backend

> **ดูของจริง:** อ่าน `projects/bop-dashboard/src/shared/api/api-client.ts` ทั้งไฟล์ — generic `<T>`, `body?: unknown`, การแยก network error (`status: 0`) ออกจาก HTTP error, การจัดการ 204 คอมเมนต์อธิบายเหตุผลไว้ทุกจุด

## สรุปบท

- ใช้ `const` เป็นหลัก, `===` เสมอ, `??` แทน `||` เมื่อ 0/"" เป็นค่าที่ถูกต้อง
- object/array เป็น reference — copy ด้วย spread ก่อนแก้
- `fetch` ไม่ throw เมื่อ 4xx/5xx เช็ค `res.ok` เสมอ
- JS thread เดียว — I/O ไม่ block แต่ CPU หนัก block ทั้งหน้า
- TypeScript type หายตอน runtime — ข้อมูลจากภายนอกต้อง validate
