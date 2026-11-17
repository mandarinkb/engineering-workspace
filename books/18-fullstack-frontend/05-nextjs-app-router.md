# บทที่ 5: Next.js App Router

React เป็นแค่ library สำหรับวาด UI Next.js เป็น **framework** ที่เพิ่มสิ่งที่แอปจริงต้องใช้: routing, การ render ฝั่ง server, bundling, การจัดการรูปและ font, และ endpoint ฝั่ง server สำหรับ backend engineer บทนี้คือบทที่ "คลิก" ที่สุด เพราะครึ่งหนึ่งของ Next.js **รันบน server** — ซึ่งเป็นโลกที่คุณคุ้นเคย

> **เรื่องเวอร์ชัน:** เนื้อหาอิง Next.js 15 (เวอร์ชันเดียวกับ `projects/bop-dashboard`) ถ้า `create-next-app@latest` ให้เวอร์ชันใหม่กว่า แนวคิดในบทนี้ยังเหมือนเดิม จุดที่เปลี่ยนชื่อจะระบุไว้ (เช่น `middleware.ts` → `proxy.ts` ใน Next 16) เช็ค [release notes](https://nextjs.org/blog) เมื่อเจอสิ่งที่ไม่ตรง

## 5.1 ทำไมต้อง Framework

ปัญหาของ SPA ล้วน (บทที่ 4.5):

1. **หน้าแรกช้า** — browser ได้ HTML ว่างๆ ต้องโหลด JS → รัน → ยิง API → ค่อยเห็นข้อมูล (waterfall)
2. **SEO/share link แย่** — bot และ preview ของ LINE/Facebook เห็นหน้าว่าง
3. ต้องตั้งค่า router, bundler, code splitting, env เองทั้งหมด

Next.js render HTML ที่มีข้อมูลแล้วจาก server (เหมือน Go ที่ใช้ `html/template`) ส่งไปพร้อม JS ที่ทำให้หน้า interactive ได้ (**hydration**) — ได้ข้อดีของทั้ง server-rendered และ SPA

## 5.2 File-based Routing และ Layout

โฟลเดอร์ใน `app/` = URL, ไฟล์ชื่อพิเศษกำหนดหน้าที่

```text
app/
├── layout.tsx            ← root layout (บังคับมี) ครอบทุกหน้า: <html><body>
├── page.tsx              ← /
├── globals.css
├── about/
│   └── page.tsx          ← /about
├── tasks/
│   ├── layout.tsx        ← ครอบทุกหน้าใต้ /tasks
│   ├── page.tsx          ← /tasks
│   ├── loading.tsx       ← แสดงระหว่างรอ page.tsx โหลดข้อมูล
│   ├── error.tsx         ← แสดงเมื่อ page.tsx throw
│   ├── new/page.tsx      ← /tasks/new
│   └── [id]/page.tsx     ← /tasks/123 (dynamic segment)
├── (auth)/               ← route group: วงเล็บ = ไม่เป็นส่วนของ URL ใช้จัดกลุ่ม/แยก layout
│   └── login/page.tsx    ← /login
└── api/health/route.ts   ← GET /api/health (Route Handler)
```

```tsx
// app/layout.tsx
import "./globals.css";
import Link from "next/link";

export const metadata = { title: "Tasks" };    // <title> และ meta tag

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="th">
      <body className="min-h-screen bg-gray-50">
        <nav className="flex gap-4 border-b bg-white p-4">
          <Link href="/">Home</Link>
          <Link href="/tasks">Tasks</Link>
        </nav>
        <main className="mx-auto max-w-4xl p-6">{children}</main>
      </body>
    </html>
  );
}
```

- **layout ไม่ re-render เมื่อเปลี่ยนหน้า** (state ใน layout ยังอยู่) เปลี่ยนแค่ `children`
- `page.tsx` ต้อง `export default` component
- ไฟล์อื่นในโฟลเดอร์ (เช่น `task-card.tsx`) ไม่กลายเป็น route — วางไว้ข้างกันได้ (colocation)

> **ดูของจริง:** `projects/bop-dashboard/app/` — `(auth)` กับ `(dashboard)` เป็น route group ที่มี layout ต่างกัน (หน้า login ไม่มี navbar) และ `page.tsx` ทุกไฟล์แค่ re-export จาก `src/pages/` (เหตุผลอยู่ใน `ARCHITECTURE.md`)

## 5.3 Server Component vs Client Component

**หัวข้อที่สำคัญที่สุดของ Next.js App Router** — ใช้เวลากับมันให้มาก

ใน App Router ทุก component เป็น **Server Component โดย default**

| | Server Component (default) | Client Component (`"use client"`) |
|---|---|---|
| รันที่ไหน | server เท่านั้น (ตอน build หรือตอนมี request) | server (render HTML ครั้งแรก) **และ** browser (hydrate + interactive) |
| `async/await` ใน component | ✅ `await fetch(...)` / query DB ตรงๆ ได้ | ❌ |
| `useState`, `useEffect`, event handler (`onClick`) | ❌ | ✅ |
| อ่าน secret / env ฝั่ง server / DB | ✅ (โค้ดไม่ถูกส่งไป browser) | ❌ (ทุกอย่างถูกส่งไป browser) |
| เพิ่มขนาด JS ที่ browser ต้องโหลด | ไม่ | ใช่ |

```tsx
// app/tasks/page.tsx — Server Component
import { ToggleButton } from "./toggle-button";

export default async function TasksPage() {
  // fetch จาก server → server: ไม่ผ่าน browser จึงไม่มี CORS และ URL ภายในไม่ถูกเปิดเผย
  const res = await fetch(`${process.env.API_INTERNAL_URL}/tasks`, { cache: "no-store" });
  if (!res.ok) throw new Error("load tasks failed");     // ไปที่ error.tsx
  const tasks: Task[] = await res.json();

  console.log("render on server");   // log ไปโผล่ใน terminal ที่รัน `npm run dev` ไม่ใช่ใน browser

  return (
    <ul className="space-y-2">
      {tasks.map((t) => (
        <li key={t.id} className="flex justify-between rounded border bg-white p-3">
          {t.title}
          <ToggleButton id={t.id} status={t.status} />   {/* เกาะ client component ไว้เฉพาะส่วนที่ต้อง interactive */}
        </li>
      ))}
    </ul>
  );
}
```

```tsx
// app/tasks/toggle-button.tsx — Client Component
"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

export function ToggleButton({ id, status }: { id: number; status: TaskStatus }) {
  const router = useRouter();
  const [pending, setPending] = useState(false);

  async function toggle() {
    setPending(true);
    await fetch(`${process.env.NEXT_PUBLIC_API_URL}/tasks/${id}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ status: status === "done" ? "todo" : "done" }),
      credentials: "include",
    });
    setPending(false);
    router.refresh();     // สั่งให้ server component ของหน้านี้ render ใหม่ด้วยข้อมูลล่าสุด
  }

  return <button onClick={toggle} disabled={pending}>{status}</button>;
}
```

### กฎที่ต้องจำ

1. `"use client"` ประกาศ **boundary**: ไฟล์นั้นและทุกอย่างที่มัน `import` กลายเป็น client
2. **"use client" ไม่ได้แปลว่า render แค่ฝั่ง client** — ยังถูก render เป็น HTML บน server ในการโหลดครั้งแรก แค่ถูกส่ง JS ไปให้ browser ด้วยเพื่อให้ interactive
3. Server Component import Client Component ได้ แต่**กลับกันไม่ได้** (Client import Server ไม่ได้) — ถ้าต้องการ ให้ส่ง Server Component เข้าไปเป็น `children`
4. props ที่ส่งจาก server ไป client ต้อง serialize เป็น JSON ได้ (ส่ง function, class instance, `Date` ไม่ได้ — ส่ง string แทน)
5. วาง `"use client"` ให้ลึกที่สุด (ที่ใบไม้ของ tree) เพื่อให้ JS ที่ส่งไป browser น้อยที่สุด

### Provider ต้องเป็น Client Component

Context และ TanStack Query ใช้ state → ต้องอยู่ใน client component แต่ `app/layout.tsx` เป็น server component จึงต้องแยกไฟล์:

```tsx
// src/app/providers.tsx
"use client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState } from "react";

export function Providers({ children }: { children: React.ReactNode }) {
  // useState ไม่ใช่ตัวแปร module-level: server render หลาย request พร้อมกัน
  // ถ้าแชร์ QueryClient ตัวเดียว cache ของ user หนึ่งอาจรั่วไปอีกคน
  const [client] = useState(() => new QueryClient({ defaultOptions: { queries: { staleTime: 30_000 } } }));
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

// app/layout.tsx (server)
<body><Providers>{children}</Providers></body>
```

> **ดูของจริง:** `projects/bop-dashboard/src/app/providers/providers.tsx` — คอมเมนต์อธิบายทั้งสองเหตุผลข้างบนไว้ละเอียด

### แล้วจะ fetch ที่ไหน? (แนวทางสำหรับโปรเจกต์ที่มี Go backend)

| สถานการณ์ | ทางเลือก |
|---|---|
| หน้าที่ต้องการ SEO / หน้าแรกเร็ว / ข้อมูลไม่ต้อง realtime | Server Component `await fetch` |
| Dashboard หลัง login ที่มี interaction เยอะ, polling, filter ฝั่ง client | Client Component + TanStack Query (แบบ bop-dashboard) |
| ผสม | server fetch ข้อมูลเริ่มต้น แล้วส่งเป็น `initialData` ให้ TanStack Query |

ทั้งสองแบบถูกต้อง bop-dashboard เลือกแบบที่ 2 เกือบทั้งหมด เพราะเป็น dashboard ภายในที่ต้อง login และไม่ต้องการ SEO

## 5.4 Dynamic Route, loading, error, not-found

```tsx
// app/tasks/[id]/page.tsx
import { notFound } from "next/navigation";

// Next 15+: params เป็น Promise ต้อง await
export default async function TaskDetail({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const res = await fetch(`${process.env.API_INTERNAL_URL}/tasks/${id}`, { cache: "no-store" });

  if (res.status === 404) notFound();             // → not-found.tsx
  if (!res.ok) throw new Error(`HTTP ${res.status}`);   // → error.tsx

  const task: Task = await res.json();
  return <h1 className="text-2xl font-bold">{task.title}</h1>;
}

export async function generateMetadata({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return { title: `Task #${id}` };
}
```

```tsx
// app/tasks/[id]/loading.tsx — แสดงทันทีระหว่าง page.tsx กำลัง await (ใช้ React Suspense ข้างใต้)
export default function Loading() {
  return <div className="h-8 w-64 animate-pulse rounded bg-gray-200" />;
}

// app/tasks/[id]/error.tsx — ต้องเป็น client component
"use client";
export default function Error({ error, reset }: { error: Error; reset: () => void }) {
  return (
    <div>
      <p>เกิดข้อผิดพลาด: {error.message}</p>
      <button onClick={reset}>ลองใหม่</button>
    </div>
  );
}

// app/tasks/[id]/not-found.tsx
export default function NotFound() { return <p>ไม่พบงานนี้</p>; }
```

Client Component อ่าน param ด้วย hook: `useParams()`, query string ด้วย `useSearchParams()`, เปลี่ยนหน้าด้วย `useRouter().push("/tasks")` (import จาก `next/navigation` ไม่ใช่ `next/router` ซึ่งเป็นของ Pages Router รุ่นเก่า)

## 5.5 Rendering Mode และ Caching

| Mode | HTML ถูกสร้างเมื่อ | เหมาะกับ |
|---|---|---|
| **SSG** (Static) | ตอน `next build` ครั้งเดียว | หน้า marketing, docs, blog |
| **ISR** | ตอน build แล้วสร้างใหม่ทุก N วินาที | สินค้า, บทความที่เปลี่ยนไม่บ่อย |
| **SSR** (Dynamic) | ทุก request | ข้อมูลเฉพาะ user, ต้องสดเสมอ |
| **CSR** | ใน browser หลังโหลด JS | ส่วน interactive ใน dashboard |

Next.js ตัดสินให้เองจากโค้ด: ถ้าหน้าใช้ข้อมูลต่อ request (`cookies()`, `headers()`, `searchParams`, `fetch` แบบ `no-store`) จะเป็น dynamic นอกนั้นพยายามทำ static

```ts
await fetch(url, { cache: "no-store" });             // ดึงใหม่ทุก request (SSR)
await fetch(url, { next: { revalidate: 60 } });      // cache 60 วินาที (ISR)
await fetch(url, { cache: "force-cache" });          // cache จนกว่าจะ build ใหม่ (SSG)
```

ดูผลด้วย `npm run build` — ตารางท้าย output จะบอกว่าแต่ละ route เป็น `○ (Static)` หรือ `ƒ (Dynamic)`

> **caching เป็นส่วนที่เปลี่ยนบ่อยที่สุดระหว่างเวอร์ชัน** (Next 14 cache fetch โดย default, Next 15 ไม่ cache, Next 16 เพิ่ม `"use cache"`) ถ้าไม่แน่ใจ **ระบุ `cache` ให้ชัดทุกครั้ง** และเช็คเอกสารของเวอร์ชันที่ใช้

## 5.6 Route Handler และ Server Action

### Route Handler — endpoint HTTP ใน Next.js

```ts
// app/api/health/route.ts
import { NextResponse } from "next/server";

export async function GET() {
  return NextResponse.json({ ok: true, time: new Date().toISOString() });
}

export async function POST(request: Request) {
  const body = await request.json();
  return NextResponse.json({ received: body }, { status: 201 });
}
```

เหมือน `http.HandleFunc` ใน Go ใช้ Web standard `Request`/`Response`

### Server Action — function ฝั่ง server ที่เรียกจาก form ได้ตรงๆ

```tsx
// app/tasks/new/actions.ts
"use server";
import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";

export async function createTask(formData: FormData) {
  const title = String(formData.get("title") ?? "");
  if (title.length < 3) return { error: "title too short" };   // validate ฝั่ง server เสมอ

  await fetch(`${process.env.API_INTERNAL_URL}/tasks`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ title, status: "todo" }),
  });
  revalidatePath("/tasks");     // ล้าง cache ของหน้า /tasks
  redirect("/tasks");
}

// app/tasks/new/page.tsx
import { createTask } from "./actions";
export default function NewTask() {
  return (
    <form action={createTask}>
      <input name="title" />
      <button>Create</button>
    </form>
  );
}
```

Next.js สร้าง endpoint POST ให้อัตโนมัติ — **ถือว่าเป็น public endpoint** ต้องเช็ค auth และ validate input ทุกครั้งเหมือน handler ปกติ

### มี Go backend อยู่แล้ว ควรใช้สองอย่างนี้ทำอะไร?

- ✅ **BFF (Backend-for-Frontend):** รวมหลาย API ของ Go เป็น response เดียวที่เหมาะกับหน้าจอ, ซ่อน URL ภายใน, ทำ proxy เพื่อเลี่ยง CORS
- ✅ งานเล็กที่ผูกกับ UI: webhook ของ Next, revalidate cache, OG image
- ❌ **business logic และการเข้าถึง database หลัก** — ให้อยู่ที่ Go ที่เดียว ไม่งั้น logic จะกระจายสองที่

## 5.7 Middleware (Next 16+: Proxy)

โค้ดที่รัน**ก่อน** request จะถึงหน้าใดๆ — ใช้ redirect, rewrite, ตั้ง header

```ts
// middleware.ts (Next 15) / proxy.ts (Next 16+ — เปลี่ยนชื่อไฟล์และ function เป็น proxy)
import { NextResponse, type NextRequest } from "next/server";

export function middleware(request: NextRequest) {
  const hasSession = request.cookies.has("session");
  if (!hasSession && !request.nextUrl.pathname.startsWith("/login")) {
    return NextResponse.redirect(new URL("/login", request.url));
  }
  return NextResponse.next();
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],   // ไม่รันกับไฟล์ static
};
```

⚠️ **middleware ไม่ใช่ security boundary** — ส่วนใหญ่เช็คได้แค่ว่า "มี cookie" ไม่ได้ verify JWT (ไม่มี secret) การตรวจจริงต้องอยู่ที่ Go backend ทุก endpoint middleware มีไว้เพื่อ UX (ไม่ให้หน้า dashboard วาบขึ้นมาก่อนโดน redirect)

> **ดูของจริง:** `projects/bop-dashboard/middleware.ts` — คอมเมนต์อธิบายข้อจำกัดนี้ไว้ชัดเจน

## สรุปบท

- โฟลเดอร์ใน `app/` = route, `layout.tsx` ครอบและไม่ re-render, `loading`/`error`/`not-found` จัดการสถานะให้
- ทุกอย่างเป็น Server Component จนกว่าจะเขียน `"use client"` — วาง client boundary ให้ลึกที่สุด
- Server Component fetch ได้ตรงๆ ไม่ติด CORS ไม่เปิดเผย secret
- ระบุ cache ของ fetch ให้ชัดเสมอ
- Route Handler / Server Action = public endpoint ต้องตรวจ auth และ validate เอง; business logic ให้อยู่ที่ Go
- middleware/proxy = UX ไม่ใช่ security
