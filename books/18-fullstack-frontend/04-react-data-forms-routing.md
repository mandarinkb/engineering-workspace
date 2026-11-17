# บทที่ 4: Data Fetching, Form, Routing

บทที่ 3 เราเขียน `useEffect` + `fetch` + `useState` ×3 เองเพื่อให้เข้าใจกลไก บทนี้คือวิธีที่ใช้ในงานจริง: ยก "ข้อมูลจาก server" ออกจาก React state ไปให้ library จัดการ, ใช้ schema เดียว validate form ทั้ง type และ runtime, และทำหลายหน้าโดยไม่โหลดหน้าใหม่

## 4.1 Server State ≠ Client State

| | Client state | Server state |
|---|---|---|
| ตัวอย่าง | tab ที่เลือก, modal เปิด/ปิด, ข้อความใน input, theme | รายการ tasks, ข้อมูล user, job status |
| เจ้าของ | เบราว์เซอร์นี้ | database ฝั่ง backend |
| ปัญหา | น้อย | **เก่าได้** (คนอื่นแก้), ต้อง cache, loading/error, retry, dedupe request ซ้ำ, refetch เมื่อกลับมาที่แท็บ, pagination |
| เครื่องมือ | `useState`, Context | **TanStack Query** (หรือ SWR, RTK Query) |

ถ้าเขียน server state เองด้วย `useEffect` ทุกหน้าต้องเขียน loading/error/abort/refetch ซ้ำ และ 2 component ที่ใช้ข้อมูลเดียวกันจะยิง request 2 ครั้ง TanStack Query คือ **cache ฝั่ง client** ที่จัดการเรื่องเหล่านี้ให้ — คิดว่าเป็น Redis ตัวเล็กในเบราว์เซอร์ที่รู้วิธีไปดึงข้อมูลใหม่เอง

## 4.2 TanStack Query: `useQuery`

### Setup ครั้งเดียวที่ root

```tsx
// main.tsx (Vite) — Next.js ดูบทที่ 5 หัวข้อ 5.3
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { staleTime: 30_000 },   // ข้อมูลอายุไม่ถึง 30 วิ ถือว่ายังสด ไม่ต้อง refetch
  },
});

createRoot(document.getElementById("root")!).render(
  <QueryClientProvider client={queryClient}>
    <App />
    <ReactQueryDevtools />
  </QueryClientProvider>,
);
```

### API client กลาง + query hook

```ts
// shared/api.ts — fetch wrapper ตัวเดียวทั้งแอป (แบบย่อของ bop-dashboard/src/shared/api/api-client.ts)
const API_URL = import.meta.env.VITE_API_URL ?? "http://localhost:8080";

export class ApiError extends Error {
  constructor(message: string, public status: number) { super(message); }
}

export async function api<T>(path: string, init?: { method?: string; body?: unknown }): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, {
    method: init?.method ?? "GET",
    headers: init?.body ? { "Content-Type": "application/json" } : undefined,
    body: init?.body ? JSON.stringify(init.body) : undefined,
    credentials: "include",
  });
  if (!res.ok) throw new ApiError(await res.text() || res.statusText, res.status);
  return res.status === 204 ? (undefined as T) : res.json();
}
```

```ts
// entities/task/api.ts
import { useQuery } from "@tanstack/react-query";

export const taskKeys = {
  all: ["tasks"] as const,
  detail: (id: number) => ["tasks", id] as const,
};

export function useTasks() {
  return useQuery({
    queryKey: taskKeys.all,
    queryFn: () => api<Task[]>("/tasks"),
  });
}

export function useTask(id: number) {
  return useQuery({
    queryKey: taskKeys.detail(id),
    queryFn: () => api<Task>(`/tasks/${id}`),
  });
}
```

```tsx
function TaskPage() {
  const { data: tasks, isPending, isError, error, refetch } = useTasks();

  if (isPending) return <Skeleton />;
  if (isError) return <ErrorBox message={error.message} onRetry={refetch} />;
  return <TaskList tasks={tasks} />;    // ตรงนี้ TS รู้ว่า tasks ไม่ใช่ undefined แล้ว
}
```

เทียบกับโค้ดบทที่ 3: ไม่มี `useState` ไม่มี `useEffect` ไม่มี AbortController

### `queryKey` คือหัวใจ

- เป็น **cache key** — component ไหนเรียก `["tasks"]` ก็ได้ข้อมูลชุดเดียวกัน ยิง request ครั้งเดียว
- ถ้า key มีตัวแปร (`["tasks", id]`, `["tasks", { status, page }]`) เมื่อตัวแปรเปลี่ยน query จะดึงใหม่อัตโนมัติ (แทน dependency array ของ `useEffect`)
- เป็นลำดับชั้น: invalidate `["tasks"]` จะ invalidate `["tasks", 1]`, `["tasks", 2]` ด้วย

### ชีวิตของข้อมูลใน cache

```text
fetch สำเร็จ → fresh (ภายใน staleTime: ใช้ cache ไม่ยิงใหม่)
            → stale (ยังแสดง cache ทันที แต่จะ refetch เบื้องหลังเมื่อ: mount ใหม่ / กลับมาที่แท็บ / network กลับมา)
ไม่มีใครใช้ → หลัง gcTime (default 5 นาที) ลบออกจาก cache
```

ค่า default ที่ควรรู้: `staleTime: 0` (stale ทันที), retry 3 ครั้งเมื่อ error, refetch เมื่อ focus กลับมาที่หน้าต่าง

> **ดูของจริง:** `projects/bop-dashboard/src/shared/api/query-client.ts` — ตั้ง `staleTime` 30 วิ และ**ไม่ retry เมื่อเป็น 4xx** (retry 401/404 กี่ครั้งก็ได้ผลเดิม) หลักการเดียวกับ non-retryable error ฝั่ง backend

## 4.3 `useMutation` และ Invalidation

query = อ่าน, mutation = เขียน (POST/PATCH/DELETE)

```ts
import { useMutation, useQueryClient } from "@tanstack/react-query";

export function useCreateTask() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { title: string; status: TaskStatus }) =>
      api<Task>("/tasks", { method: "POST", body: input }),
    onSuccess: () => {
      // บอกว่า cache ของ tasks เก่าแล้ว → query ที่ใช้อยู่จะ refetch เอง
      qc.invalidateQueries({ queryKey: taskKeys.all });
    },
  });
}

export function useDeleteTask() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => api<void>(`/tasks/${id}`, { method: "DELETE" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: taskKeys.all }),
  });
}
```

```tsx
function DeleteButton({ id }: { id: number }) {
  const del = useDeleteTask();
  return (
    <button
      onClick={() => del.mutate(id, { onSuccess: () => toast("ลบแล้ว") })}
      disabled={del.isPending}
    >
      {del.isPending ? "กำลังลบ…" : "ลบ"}
    </button>
  );
}
```

**ทำไม invalidate แทนแก้ cache เอง?** ง่ายและถูกต้องเสมอ — server เป็น source of truth (อาจเติม `id`, `created_at`, หรือมี trigger แก้ข้อมูลอื่น) แลกกับ request เพิ่ม 1 ครั้ง

**Optimistic update** (ขั้นสูง รู้ไว้พอ): แก้ cache ทันทีก่อน server ตอบ (`onMutate` + `setQueryData`) ถ้า error ให้ rollback — ใช้กับ action ที่ต้องรู้สึกเร็วมาก เช่น กด like, toggle checkbox

> **ดูของจริง:** `projects/bop-dashboard/src/features/create-job/api/use-create-job.ts` และ `src/features/cancel-job/api/use-cancel-job.ts`

## 4.4 Form: react-hook-form + zod

### ทำไมไม่ใช้ `useState` ทุกช่อง

ฟอร์ม 10 ช่องด้วย `useState` = 10 state + 10 onChange + validate เอง + เก็บ error เอง + re-render ทุกตัวอักษร react-hook-form จัดการให้ และ zod ให้เราประกาศกฎ**ครั้งเดียว**ได้ทั้ง validation และ TypeScript type

### zod — schema validation ตอน runtime

```ts
import { z } from "zod";

export const taskFormSchema = z.object({
  title: z.string().trim().min(3, "อย่างน้อย 3 ตัวอักษร").max(100, "ไม่เกิน 100 ตัวอักษร"),
  status: z.enum(["todo", "doing", "done"]),
  dueDate: z.string().optional(),
});

export type TaskFormValues = z.infer<typeof taskFormSchema>;
// = { title: string; status: "todo" | "doing" | "done"; dueDate?: string }

// ใช้ validate ข้อมูลจาก API ได้ด้วย (แก้ปัญหา "as T ไม่ได้ตรวจจริง" จากบทที่ 2)
const taskSchema = taskFormSchema.extend({ id: z.number(), created_at: z.string() });
const tasks = z.array(taskSchema).parse(await res.json());   // shape ผิด → throw ชัดเจน
```

เทียบกับ Go: เหมือน struct + `validate:"required,min=3"` tag (go-playground/validator) แต่ type ถูก**สร้างจาก** schema

### react-hook-form

```tsx
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";

export function TaskForm({ defaultValues, onDone }: {
  defaultValues?: Partial<TaskFormValues>;
  onDone?: () => void;
}) {
  const create = useCreateTask();
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<TaskFormValues>({
    resolver: zodResolver(taskFormSchema),
    defaultValues: { status: "todo", ...defaultValues },
  });

  // handleSubmit validate ก่อน — callback ถูกเรียกเมื่อข้อมูลผ่าน schema แล้วเท่านั้น
  const onSubmit = handleSubmit(async (values) => {
    await create.mutateAsync(values);
    reset();
    onDone?.();
  });

  return (
    <form onSubmit={onSubmit} className="space-y-4">
      <div>
        <label htmlFor="title">Title</label>
        <input id="title" {...register("title")} className="w-full rounded border px-3 py-2" />
        {errors.title && <p className="text-sm text-red-600">{errors.title.message}</p>}
      </div>

      <div>
        <label htmlFor="status">Status</label>
        <select id="status" {...register("status")}>
          <option value="todo">Todo</option>
          <option value="doing">Doing</option>
          <option value="done">Done</option>
        </select>
      </div>

      {create.isError && <p className="text-red-600">{create.error.message}</p>}

      <button type="submit" disabled={isSubmitting}>Save</button>
    </form>
  );
}
```

- `register("title")` คืน `{ name, onChange, onBlur, ref }` แล้ว spread ลง input — input เป็น **uncontrolled** (DOM เก็บค่าเอง) จึงไม่ re-render ทุกตัวอักษร
- error จาก server (เช่น 409 title ซ้ำ) แสดงจาก `mutation.error` หรือใช้ `setError("title", ...)` ให้ไปโผล่ที่ช่อง

**Validate ฝั่ง frontend เพื่อ UX, validate ฝั่ง backend เพื่อความถูกต้องและความปลอดภัย — ต้องมีทั้งคู่เสมอ**

> **ดูของจริง:** `projects/bop-dashboard/src/features/auth/ui/login-form.tsx` (ฟอร์มพื้นฐาน), `src/entities/bot/model/bot-config.schema.ts` (สร้าง zod schema แบบ dynamic จาก config ที่ backend ส่งมา — ขั้นสูง), `src/features/schedule-form/ui/schedule-form.tsx`

## 4.5 Routing ฝั่ง Client (SPA)

SPA (Single Page Application) โหลด HTML ครั้งเดียว แล้วเปลี่ยน "หน้า" ด้วย JavaScript: router ฟัง URL (History API) แล้วเลือกว่าจะ render component ไหน — ไม่มี request ไป server เพื่อขอหน้าใหม่ (มีแค่ API call)

สัปดาห์ที่ 5 ใช้ React Router กับ Vite เพื่อให้เข้าใจแนวคิด สัปดาห์ที่ 6 Next.js จะใช้ระบบ routing ของตัวเองแทน

```tsx
// main.tsx — React Router v7
import { BrowserRouter, Routes, Route, Link, useParams, useNavigate } from "react-router";

<BrowserRouter>
  <Routes>
    <Route path="/" element={<Layout />}>              {/* layout ครอบหลายหน้า */}
      <Route index element={<TaskListPage />} />
      <Route path="tasks/new" element={<NewTaskPage />} />
      <Route path="tasks/:id" element={<TaskDetailPage />} />
      <Route path="*" element={<NotFound />} />
    </Route>
  </Routes>
</BrowserRouter>

// Layout ใช้ <Outlet /> เป็นจุดที่หน้าลูกจะถูก render
function Layout() {
  return (
    <>
      <nav><Link to="/">Tasks</Link> <Link to="/tasks/new">New</Link></nav>
      <main><Outlet /></main>
    </>
  );
}

function TaskDetailPage() {
  const { id } = useParams();                    // string เสมอ — แปลงเป็น number เอง
  const { data, isPending } = useTask(Number(id));
  const navigate = useNavigate();
  ...
  // หลังลบเสร็จ: navigate("/")
}
```

- ใช้ `<Link>` แทน `<a href>` เพื่อไม่ให้โหลดหน้าใหม่ทั้งหน้า
- URL เป็น state อย่างหนึ่ง: filter/page ที่อยากให้ share link ได้ ให้เก็บใน query string (`useSearchParams`) แทน `useState`
- **ข้อเสียของ SPA ล้วน:** หน้าแรกว่างจนกว่า JS จะโหลดเสร็จ (ช้า, SEO แย่), ต้องตั้ง server ให้ส่ง `index.html` ทุก path — นี่คือสิ่งที่ Next.js มาแก้ (บทที่ 5)

## 4.6 สรุปโครงที่ได้หลังจบสัปดาห์ 5

```text
react-tasks/src/
├── main.tsx              QueryClientProvider + BrowserRouter
├── shared/
│   ├── api.ts            api<T>() + ApiError
│   └── ui/               Button, Card, Badge, Spinner, ErrorBox
├── entities/task/
│   ├── types.ts          Task, TaskStatus
│   ├── schema.ts         zod schema
│   └── api.ts            taskKeys, useTasks, useTask, useCreateTask, useDeleteTask
├── features/task-form/   TaskForm
└── pages/                TaskListPage, TaskDetailPage, NewTaskPage, NotFound
```

## สรุปบท

- ข้อมูลจาก server ไม่ใช่ `useState` — ใช้ TanStack Query
- `queryKey` = cache key, เปลี่ยน key = fetch ใหม่, หลัง mutation ให้ invalidate
- zod ประกาศกฎครั้งเดียว ได้ทั้ง validation และ type; react-hook-form จัดการ state ของฟอร์ม
- Validate ทั้งสองฝั่งเสมอ
- SPA routing เปลี่ยนหน้าโดยไม่โหลด HTML ใหม่ — แต่มีข้อเสียเรื่องหน้าแรกและ SEO
