# บทที่ 3: React Fundamentals

ถ้าจำได้เพียงประโยคเดียวจากบทนี้ ให้จำว่า **UI = f(state)** — หน้าจอคือผลลัพธ์ของฟังก์ชันที่รับ state เข้าไป เมื่อ state เปลี่ยน React เรียกฟังก์ชันใหม่แล้วอัปเดต DOM เฉพาะส่วนที่ต่างให้เอง เราไม่ต้องสั่ง "เพิ่ม `<li>` ตรงนี้ ลบตรงนั้น" อีกต่อไป

## 3.1 ทำไมต้อง React

ใน W2·D2 ที่เขียน vanilla TS คุณต้องจำเองว่าข้อมูลเปลี่ยนแล้วต้องไปแก้ DOM ตรงไหน (**imperative**) React ให้เราประกาศว่า "ถ้าข้อมูลเป็นแบบนี้ หน้าจอควรเป็นแบบนี้" (**declarative**)

```tsx
// imperative (vanilla): บอกทีละขั้น
list.innerHTML = "";
for (const t of tasks) { const li = document.createElement("li"); li.textContent = t.title; list.append(li); }
counter.textContent = String(tasks.length);

// declarative (React): บอกผลลัพธ์
function TaskList({ tasks }: { tasks: Task[] }) {
  return (
    <>
      <p>{tasks.length} tasks</p>
      <ul>{tasks.map((t) => <li key={t.id}>{t.title}</li>)}</ul>
    </>
  );
}
```

เทียบกับ backend: เหมือนเขียน `html/template` ใน Go แล้ว render ใหม่ทุกครั้งที่ข้อมูลเปลี่ยน — แต่ React ทำในเบราว์เซอร์ และเปรียบเทียบผลลัพธ์ใหม่กับของเดิม (reconciliation) แล้วแก้ DOM เฉพาะจุดที่ต่าง จึงเร็ว

## 3.2 Component, JSX, Props

**Component** = function ที่คืน JSX ชื่อต้องขึ้นต้นด้วยตัวพิมพ์ใหญ่

```tsx
type Task = { id: number; title: string; status: "todo" | "doing" | "done" };

// props คือ argument ตัวเดียวที่เป็น object — destructure ตรง parameter
type TaskItemProps = {
  task: Task;
  onDelete: (id: number) => void;    // ส่ง function ลงไปเป็น prop ได้
};

export function TaskItem({ task, onDelete }: TaskItemProps) {
  return (
    <li className="flex items-center justify-between">
      <span>{task.title}</span>                       {/* {} = ใส่ expression JS */}
      <button onClick={() => onDelete(task.id)}>Delete</button>
    </li>
  );
}

// ใช้งาน
<TaskItem task={t} onDelete={handleDelete} />
```

กฎของ JSX ที่ต่างจาก HTML:

- `class` → `className`, `for` → `htmlFor`
- คืน element ตัวเดียว ถ้ามีหลายตัวให้ห่อด้วย `<>...</>` (Fragment)
- tag ต้องปิดเสมอ: `<input />`, `<img />`
- ใน `{}` ใส่ได้เฉพาะ **expression** (ค่าที่คำนวณได้) ใส่ `if`/`for` ไม่ได้ — ใช้ ternary และ `.map()` แทน
- style เป็น object: `style={{ marginTop: 8 }}` (แต่เราใช้ Tailwind)

**Props เป็น read-only** — component ห้ามแก้ props ของตัวเอง (เหมือนรับ argument แบบ value แล้วสัญญาว่าจะไม่ mutate) ข้อมูลไหลจาก parent ลง child ทางเดียว (**one-way data flow**) ถ้า child อยากเปลี่ยนข้อมูล ต้องเรียก callback ที่ parent ส่งมา

### `children`

```tsx
function Card({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="rounded-xl border p-4">
      <h3 className="font-semibold">{title}</h3>
      {children}
    </div>
  );
}

<Card title="Stats"><p>12 tasks</p></Card>
```

## 3.3 State และการ Re-render

**State** = ข้อมูลที่ component "จำ" ไว้ข้าม render และเมื่อเปลี่ยนแล้วหน้าจอต้องเปลี่ยนตาม

```tsx
import { useState } from "react";

export function TaskApp() {
  const [tasks, setTasks] = useState<Task[]>([]);   // [ค่าปัจจุบัน, ฟังก์ชันเปลี่ยนค่า]
  const [title, setTitle] = useState("");

  function addTask() {
    if (!title.trim()) return;
    setTasks([...tasks, { id: Date.now(), title, status: "todo" }]);   // array ใหม่
    setTitle("");
  }

  function toggle(id: number) {
    setTasks(tasks.map((t) =>
      t.id === id ? { ...t, status: t.status === "done" ? "todo" : "done" } : t   // object ใหม่
    ));
  }

  function remove(id: number) {
    setTasks(tasks.filter((t) => t.id !== id));
  }

  return (
    <div>
      {/* controlled input: ค่าใน input มาจาก state เสมอ */}
      <input value={title} onChange={(e) => setTitle(e.target.value)} />
      <button onClick={addTask}>Add</button>
      <ul>
        {tasks.map((t) => (
          <li key={t.id} onClick={() => toggle(t.id)}>{t.title} ({t.status})</li>
        ))}
      </ul>
    </div>
  );
}
```

### สิ่งที่เกิดขึ้นจริงเมื่อเรียก `setTasks`

1. React จดว่า state เปลี่ยน แล้ว **เรียก function `TaskApp` ใหม่ทั้งฟังก์ชัน**
2. ครั้งนี้ `useState` คืนค่าใหม่ ตัวแปรธรรมดาในฟังก์ชันถูกสร้างใหม่หมด
3. React เทียบ JSX ใหม่กับของเดิม แล้วแก้ DOM เฉพาะส่วนที่ต่าง

### Immutability — ทำไมห้าม `push`

React ตัดสินว่า state เปลี่ยนไหมด้วยการเทียบ **reference** (`Object.is(old, new)`) ไม่ได้ไล่เทียบข้างใน

```tsx
tasks.push(newTask);  setTasks(tasks);        // ❌ reference เดิม → React คิดว่าไม่เปลี่ยน → ไม่ re-render
setTasks([...tasks, newTask]);                // ✅ array ใหม่
```

| ต้องการ | เขียนแบบ immutable |
|---|---|
| เพิ่ม | `[...arr, item]` |
| ลบ | `arr.filter(x => x.id !== id)` |
| แก้ตัวเดียว | `arr.map(x => x.id === id ? { ...x, done: true } : x)` |
| แก้ field ของ object | `{ ...obj, title: "new" }` |

### State update เป็นแบบ snapshot

```tsx
const [count, setCount] = useState(0);
function plus3() {
  setCount(count + 1);
  setCount(count + 1);
  setCount(count + 1);   // ผลลัพธ์: 1 ไม่ใช่ 3 — count ใน render นี้คือ 0 ตลอดทั้งฟังก์ชัน
}
function plus3Correct() {
  setCount((c) => c + 1);   // updater function: รับค่าล่าสุดเสมอ
  setCount((c) => c + 1);
  setCount((c) => c + 1);   // 3
}
```

ใช้ updater function `(prev) => next` เมื่อค่าใหม่คำนวณจากค่าเก่า

### อะไรควรเป็น state อะไรไม่ควร

- ค่าที่**คำนวณจากของที่มีอยู่ได้** → **ไม่ใช่ state** คำนวณตอน render เลย

```tsx
const [tasks, setTasks] = useState<Task[]>([]);
const [filter, setFilter] = useState<"all" | Task["status"]>("all");

// ✅ derived — คำนวณทุก render (เร็วพอสำหรับข้อมูลหลักพันรายการ)
const visible = filter === "all" ? tasks : tasks.filter((t) => t.status === filter);
const doneCount = tasks.filter((t) => t.status === "done").length;

// ❌ อย่าเก็บ visible ไว้ใน useState แยก — ต้องคอยซิงก์ทุกครั้งที่ tasks เปลี่ยน ต้นเหตุ bug อันดับต้นๆ
```

## 3.4 Event, List, Conditional Rendering

### Event

```tsx
<button onClick={handleClick}>OK</button>           // ส่ง function (ไม่ใส่ ())
<button onClick={() => remove(t.id)}>x</button>     // ต้องส่ง argument → ห่อด้วย arrow
<button onClick={remove(t.id)}>x</button>           // ❌ เรียกทันทีตอน render!

<form onSubmit={(e) => { e.preventDefault(); addTask(); }}>
```

### List และ `key`

```tsx
<ul>
  {tasks.map((t) => <TaskItem key={t.id} task={t} onDelete={remove} />)}
</ul>
```

`key` ช่วยให้ React รู้ว่ารายการไหนคือตัวเดิมเมื่อ list เปลี่ยน (เพิ่ม/ลบ/สลับลำดับ) — ต้อง **unique และคงที่** ใช้ id จาก database **อย่าใช้ index** เมื่อ list ลบหรือสลับลำดับได้ ไม่งั้น state ภายใน (เช่นข้อความที่พิมพ์ค้างใน input) จะติดไปกับแถวผิดตัว

### Conditional rendering

```tsx
{loading && <Spinner />}                                  // แสดงเมื่อ true
{error ? <ErrorBox message={error} /> : <TaskList />}     // อย่างใดอย่างหนึ่ง
{tasks.length === 0 && <p>ยังไม่มีงาน</p>}

// ⚠️ กับดัก: {tasks.length && <List />} — ถ้า length = 0 จะ render เลข "0" ออกมาบนจอ
// ใช้ {tasks.length > 0 && ...} แทน

// เงื่อนไขซับซ้อน → early return
function TaskPanel({ state }: { state: LoadState }) {
  if (state.kind === "loading") return <Spinner />;
  if (state.kind === "error") return <ErrorBox message={state.message} />;
  return <TaskList tasks={state.data} />;
}
```

## 3.5 Lifting State และ Composition

เมื่อ component 2 ตัวต้องใช้ state เดียวกัน ให้ **ยก state ขึ้นไปไว้ที่ parent ที่ใกล้ที่สุดที่ครอบทั้งสองตัว** แล้วส่งลงมาเป็น props

```tsx
function App() {
  const [tasks, setTasks] = useState<Task[]>([]);       // state อยู่ที่นี่

  return (
    <>
      <TaskForm onAdd={(title) => setTasks((prev) => [...prev, newTask(title)])} />
      <TaskStats tasks={tasks} />
      <TaskList tasks={tasks} onDelete={(id) => setTasks((p) => p.filter((t) => t.id !== id))} />
    </>
  );
}

// TaskForm ไม่รู้ว่า tasks เก็บที่ไหน — รู้แค่ว่ามี onAdd ให้เรียก
function TaskForm({ onAdd }: { onAdd: (title: string) => void }) {
  const [title, setTitle] = useState("");     // state ของ input อยู่ที่ form เพราะไม่มีใครอื่นใช้
  return (
    <form onSubmit={(e) => { e.preventDefault(); onAdd(title); setTitle(""); }}>
      <input value={title} onChange={(e) => setTitle(e.target.value)} />
    </form>
  );
}
```

เทียบกับ backend: คล้าย dependency injection — child ไม่สร้าง dependency เอง แต่รับ callback จากข้างนอก ทำให้ test และใช้ซ้ำง่าย

**วิธีคิดแบบ "Thinking in React"** (อ่านที่ [react.dev](https://react.dev/learn/thinking-in-react)):
1. แตก UI เป็นกล่อง component
2. ทำเวอร์ชัน static (props อย่างเดียว) ก่อน
3. หา state ที่น้อยที่สุดที่จำเป็น (ที่เหลือ derive เอา)
4. ตัดสินว่า state อยู่ที่ไหน
5. ส่ง callback ลงไปให้ child เปลี่ยน state

## 3.6 `useEffect` — ซิงก์กับโลกภายนอก

`useEffect` ใช้เมื่อ component ต้อง**ซิงก์กับสิ่งที่อยู่นอก React**: network, timer, WebSocket/SSE, `document.title`, library ที่ไม่ใช่ React

```tsx
import { useEffect, useState } from "react";

function TaskPage() {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const controller = new AbortController();

    fetch("http://localhost:8080/tasks", { signal: controller.signal })
      .then((res) => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return res.json();
      })
      .then((data: Task[]) => setTasks(data))
      .catch((err) => { if (err.name !== "AbortError") setError(String(err)); })
      .finally(() => setLoading(false));

    // cleanup: React เรียกตอน component หายไป หรือก่อนรัน effect รอบใหม่
    return () => controller.abort();
  }, []);    // dependency array

  if (loading) return <p>Loading…</p>;
  if (error) return <p>Error: {error}</p>;
  return <ul>{tasks.map((t) => <li key={t.id}>{t.title}</li>)}</ul>;
}
```

### Dependency array

| เขียนแบบ | effect รันเมื่อ |
|---|---|
| `useEffect(fn)` | ทุก render (แทบไม่เคยต้องการ) |
| `useEffect(fn, [])` | ครั้งแรกที่ component ปรากฏ (mount) เท่านั้น |
| `useEffect(fn, [id])` | ครั้งแรก + ทุกครั้งที่ `id` เปลี่ยน |

**กฎ:** ทุกค่าจาก props/state ที่ใช้ใน effect ต้องอยู่ใน dependency array — ESLint plugin `react-hooks/exhaustive-deps` จะเตือน **อย่าปิดคำเตือนนี้**

### Lifecycle แบบย่อ

```text
mount  → render → แสดงบนจอ → effect รัน
update → render ใหม่ → แสดงบนจอ → (deps เปลี่ยน?) cleanup เก่า → effect ใหม่
unmount → cleanup
```

> โค้ดข้างบนเขียนเองเพื่อให้เข้าใจกลไก **ในงานจริงเราไม่ fetch ด้วย `useEffect` เอง** แต่ใช้ TanStack Query (บทที่ 4) หรือ fetch ใน Server Component (บทที่ 5)

## 3.7 กับดักของ `useEffect`, `useRef`, Stale Closure

### 1. Infinite loop

```tsx
useEffect(() => {
  fetch("/tasks").then((r) => r.json()).then(setTasks);
});   // ❌ ไม่มี deps → set state → re-render → effect รันอีก → วนไม่จบ
```

### 2. Race condition

ช่องค้นหาที่ยิง API ทุกครั้งที่ `query` เปลี่ยน: พิมพ์ "re" แล้ว "react" — ถ้า response ของ "re" ช้ากว่า มันจะมาทับผลของ "react" แก้ด้วย cleanup ที่ abort request เก่า (เหมือนตัวอย่าง 3.6) หรือใช้ flag `ignore`

### 3. StrictMode รัน effect 2 ครั้งใน dev

ใน development React จงใจ mount → unmount → mount ใหม่ทันที เพื่อจับ effect ที่ไม่มี cleanup ถ้าแอปพังเพราะรัน 2 ครั้ง แปลว่า effect นั้นมี bug — **ไม่ใช่** ให้ไปปิด StrictMode ใน production จะรันครั้งเดียว

### 4. ใช้ effect ในที่ที่ไม่ต้องใช้

```tsx
// ❌ ใช้ effect คำนวณค่าจาก state อื่น
const [fullName, setFullName] = useState("");
useEffect(() => setFullName(`${first} ${last}`), [first, last]);

// ✅ คำนวณตอน render
const fullName = `${first} ${last}`;

// ❌ ใช้ effect ตอบสนองการกดปุ่ม
// ✅ ทำใน event handler ตรงๆ
```

อ่านเพิ่ม: [You Might Not Need an Effect](https://react.dev/learn/you-might-not-need-an-effect) — บทความที่สำคัญที่สุดเรื่องนี้

### `useRef` — ค่าที่จำข้าม render แต่**ไม่**ทำให้ re-render

```tsx
const inputRef = useRef<HTMLInputElement>(null);
<input ref={inputRef} />
inputRef.current?.focus();          // เข้าถึง DOM element ตรงๆ

const timerRef = useRef<number | null>(null);   // เก็บ id ของ timer, ค่าที่ไม่ต้องแสดงบนจอ
```

### Stale closure

```tsx
const [count, setCount] = useState(0);
useEffect(() => {
  const id = setInterval(() => setCount(count + 1), 1000);   // ❌ count เป็น 0 ตลอด (จับค่าจาก render แรก)
  return () => clearInterval(id);
}, []);

setInterval(() => setCount((c) => c + 1), 1000);              // ✅ updater function ได้ค่าล่าสุดเสมอ
```

## 3.8 Custom Hook

Custom hook = function ที่ชื่อขึ้นต้นด้วย `use` และเรียก hook อื่นข้างใน ใช้แยก logic ที่มี state ออกจาก UI (คล้ายแยก service ออกจาก handler ใน Go)

```tsx
function useTasks() {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [version, setVersion] = useState(0);         // เปลี่ยนเพื่อสั่ง refetch

  useEffect(() => {
    const c = new AbortController();
    setLoading(true);
    fetch("http://localhost:8080/tasks", { signal: c.signal })
      .then((r) => (r.ok ? r.json() : Promise.reject(new Error(`HTTP ${r.status}`))))
      .then(setTasks)
      .catch((e) => e.name !== "AbortError" && setError(String(e)))
      .finally(() => setLoading(false));
    return () => c.abort();
  }, [version]);

  return { tasks, loading, error, refetch: () => setVersion((v) => v + 1) };
}

function useDebounce<T>(value: T, delay = 300): T {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const id = setTimeout(() => setDebounced(value), delay);
    return () => clearTimeout(id);       // พิมพ์ต่อก่อนครบเวลา → ยกเลิก timer เก่า
  }, [value, delay]);
  return debounced;
}

// ใช้งาน — component เหลือแค่ UI
function TaskPage() {
  const { tasks, loading, error, refetch } = useTasks();
  ...
}
```

### Rules of Hooks

1. เรียก hook ที่ **ระดับบนสุด** ของ component/custom hook เท่านั้น — ห้ามใน `if`, `for`, หลัง `return` ก่อนหน้า
2. เรียกได้เฉพาะใน React function component หรือ custom hook

เหตุผล: React จำ state ของแต่ละ hook ตาม**ลำดับการเรียก** ถ้าลำดับเปลี่ยนตามเงื่อนไข state จะสลับกัน

> **ดูของจริง:** `projects/bop-dashboard/src/entities/*/api/use-*.ts` — ทุก hook ในนั้นคือ custom hook ที่ห่อ TanStack Query ไว้

## 3.9 Context

เมื่อข้อมูลต้องส่งผ่านหลายชั้นจนน่ารำคาญ (**prop drilling**) เช่น theme, user ที่ login, locale ใช้ Context

```tsx
import { createContext, useContext, useState } from "react";

type Theme = "light" | "dark";
const ThemeContext = createContext<{ theme: Theme; toggle: () => void } | null>(null);

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [theme, setTheme] = useState<Theme>("light");
  const toggle = () => setTheme((t) => (t === "light" ? "dark" : "light"));
  return <ThemeContext.Provider value={{ theme, toggle }}>{children}</ThemeContext.Provider>;
}

// custom hook ห่อ useContext + error ที่ชัดเจนเมื่อลืมใส่ Provider
export function useTheme() {
  const ctx = useContext(ThemeContext);
  if (!ctx) throw new Error("useTheme must be used inside <ThemeProvider>");
  return ctx;
}

// ที่ root
<ThemeProvider><App /></ThemeProvider>

// ที่ไหนก็ได้ข้างใน
const { theme, toggle } = useTheme();
```

**อย่าใช้ Context เป็นที่เก็บทุกอย่าง:** ทุกครั้งที่ value เปลี่ยน ทุก component ที่ใช้ context นั้นจะ re-render ใช้กับข้อมูลที่เปลี่ยนไม่บ่อยและใช้กว้าง ส่วน**ข้อมูลจาก API ไม่ใช่งานของ Context** — ใช้ TanStack Query (บทที่ 4) ถ้า client state ซับซ้อนมากจริงๆ ค่อยดู Zustand

> **ดูของจริง:** `projects/bop-dashboard/src/app/providers/providers.tsx` (ลำดับการซ้อน provider และเหตุผล) และ `src/shared/lib/theme`

## สรุปบท

- UI = f(state): เปลี่ยน state → React เรียก component ใหม่ → อัปเดต DOM ส่วนที่ต่าง
- แก้ state แบบ immutable เสมอ, ค่าที่ derive ได้ไม่ต้องเป็น state
- `key` ต้องเป็น id ที่คงที่
- state อยู่ที่ parent ร่วมที่ใกล้ที่สุด, child ส่ง event ขึ้นผ่าน callback
- `useEffect` ใช้ซิงก์กับโลกภายนอกเท่านั้น และต้องมี cleanup
- Logic ที่มี state แยกเป็น custom hook, Context ใช้กับข้อมูลที่ใช้กว้างและเปลี่ยนไม่บ่อย
