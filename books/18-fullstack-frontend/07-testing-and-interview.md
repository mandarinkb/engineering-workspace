# บทที่ 7: Testing, โครงสร้างโปรเจกต์, เตรียมสัมภาษณ์

บทสุดท้ายรวม 3 เรื่องที่แยก "เขียน React ได้" ออกจาก "พร้อมทำงานในทีม": test, โครงสร้างโค้ดที่โตได้, และการอธิบายสิ่งที่ทำได้ในห้องสัมภาษณ์

## 7.1 Testing ด้วย Vitest + Testing Library

### Test อะไร (เรียงตามความคุ้ม)

1. **Pure function** — logic คำนวณ แปลงข้อมูล filter (เร็วสุด เขียนง่ายสุด คุ้มสุด) → แยก logic ออกจาก component ให้ได้มากที่สุด
2. **Component** — render ถูกตาม props, interaction (กดปุ่ม → เห็นผล), form validation
3. **E2E** (Playwright) — flow สำคัญจริงๆ ไม่กี่เส้น เช่น login → สร้าง task → เห็นในรายการ

### Setup

```bash
npm i -D vitest @vitejs/plugin-react jsdom @testing-library/react @testing-library/user-event @testing-library/jest-dom
```

```ts
// vitest.config.ts
import react from "@vitejs/plugin-react";
import path from "node:path";
import { defineConfig } from "vitest/config";

export default defineConfig({
  plugins: [react()],
  test: { environment: "jsdom", setupFiles: ["./vitest.setup.ts"], globals: true },
  resolve: { alias: { "@": path.resolve(__dirname, "./src") } },   // ให้ตรงกับ tsconfig paths
});

// vitest.setup.ts
import "@testing-library/jest-dom/vitest";   // เพิ่ม matcher เช่น toBeInTheDocument, toHaveClass
```

### Pure function — เหมือน table-driven test ใน Go

```ts
// entities/task/lib/task-stats.test.ts
import { describe, expect, it } from "vitest";
import { computeTaskStats } from "./task-stats";

describe("computeTaskStats", () => {
  it.each([
    { name: "empty", tasks: [], want: { total: 0, done: 0, percent: null } },
    { name: "half done", tasks: [t("todo"), t("done")], want: { total: 2, done: 1, percent: 50 } },
  ])("$name", ({ tasks, want }) => {
    expect(computeTaskStats(tasks)).toEqual(want);
  });
});
```

### Component

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

it("แสดง error เมื่อ submit title ว่าง", async () => {
  const user = userEvent.setup();
  render(<TaskForm onSubmit={vi.fn()} />);

  await user.click(screen.getByRole("button", { name: /save/i }));

  expect(await screen.findByText("อย่างน้อย 3 ตัวอักษร")).toBeInTheDocument();
});

it("เรียก onSubmit พร้อมค่าที่กรอก", async () => {
  const user = userEvent.setup();
  const onSubmit = vi.fn();
  render(<TaskForm onSubmit={onSubmit} />);

  await user.type(screen.getByLabelText("Title"), "Learn testing");
  await user.click(screen.getByRole("button", { name: /save/i }));

  expect(onSubmit).toHaveBeenCalledWith(expect.objectContaining({ title: "Learn testing" }));
});
```

**หลักของ Testing Library:** ทดสอบแบบที่ user ใช้ หา element ด้วย `getByRole` / `getByLabelText` / `getByText` (ลำดับความชอบ) ไม่ใช่ class หรือ id — test จะไม่พังเมื่อ refactor หน้าตา และบังคับให้ markup มี accessibility ไปในตัว (หา `getByLabelText` ไม่เจอ = ลืม `<label>`)

| query | ใช้เมื่อ |
|---|---|
| `getBy…` | ต้องมีตอนนี้ (ไม่เจอ → throw) |
| `queryBy…` | เช็คว่า**ไม่มี** (`expect(queryByText("x")).not.toBeInTheDocument()`) |
| `findBy…` | จะมีในอนาคต (async — รอ fetch/state update) |

component ที่ใช้ TanStack Query ต้องห่อด้วย `QueryClientProvider` ตัวใหม่ทุก test (`retry: false`) และ mock API ด้วย `vi.fn()` หรือ **MSW** (Mock Service Worker — ดักที่ระดับ network สมจริงกว่า)

> **ดูของจริง:** `projects/bop-dashboard/vitest.config.ts`, `src/entities/job/lib/job-stats.test.ts` (pure function), `src/entities/job/ui/job-status-badge.test.tsx` (component), `src/entities/bot/model/bot-config.schema.test.ts` (zod schema)

## 7.2 โครงสร้างโปรเจกต์: Feature-Sliced Design (FSD)

เมื่อแอปโตเกินสิบกว่าไฟล์ การจัดตามประเภท (`components/`, `hooks/`, `utils/`) จะกลายเป็นโฟลเดอร์ที่มีร้อยไฟล์ไม่รู้ว่าอะไรเกี่ยวกับอะไร FSD จัดตาม **ชั้น × เรื่องทางธุรกิจ**

```text
src/
├── app/        providers, global style                    ← บนสุด
├── pages/      1 หน้า = ประกอบ widgets/features
├── widgets/    บล็อก UI ใหญ่ (navbar, task-table)
├── features/   action ของ user (create-task, auth, filter-tasks)
├── entities/   business entity: task, user (type, api hook, UI พื้นฐาน)
└── shared/     api client, ui kit, lib — ไม่รู้จัก business เลย  ← ล่างสุด
```

กฎ 3 ข้อ:

1. **import ลงล่างได้อย่างเดียว** (`features` ใช้ `entities` ได้ `entities` ห้ามใช้ `features`)
2. **slice ในชั้นเดียวกันห้าม import กันเอง** (`features/create-task` ห้าม import `features/auth`)
3. **ข้าม slice ต้องผ่าน `index.ts`** (public API) — `@/entities/task` ไม่ใช่ `@/entities/task/api/use-tasks`

เทียบกับ Clean Architecture ใน Go ที่คุณทำอยู่:

| Go (clean architecture) | FSD |
|---|---|
| `domain` (entity ไม่พึ่งใคร) | `entities` + `shared` |
| `usecase` | `features` |
| `adapter/http` (handler ประกอบ usecase) | `widgets` / `pages` |
| dependency rule: ชั้นในห้ามรู้จักชั้นนอก | import ลงล่างได้อย่างเดียว |
| package export เฉพาะตัวพิมพ์ใหญ่ | `index.ts` เป็น public API |

ไม่จำเป็นต้องใช้ FSD ทุกโปรเจกต์ (โปรเจกต์เล็กใช้แค่ `components/` + `lib/` ก็พอ) แต่ถ้าสัมภาษณ์แล้วอธิบายได้ว่า "จัดโครงสร้างตามหลักอะไร และหลักนั้นแก้ปัญหาอะไร" จะได้คะแนนมาก

> **ดูของจริง:** `projects/bop-dashboard/ARCHITECTURE.md` ทั้งไฟล์ และ `eslint.config.mjs` ที่บังคับกฎ import ด้วย `no-restricted-imports`

## 7.3 Performance และ Accessibility

### Performance — วัดก่อนแก้เสมอ

| ปัญหา | วิธีดู | วิธีแก้ |
|---|---|---|
| หน้าโหลดช้า | Lighthouse, Network tab | Server Component, ลด JS ที่ส่งไป browser (`"use client"` ให้ลึก), `next/image`, `next/font`, dynamic import (`next/dynamic`) |
| พิมพ์/คลิกแล้วหน่วง | React DevTools → Profiler → "Highlight updates" | ย้าย state ลงไปใกล้ที่ใช้ (state ที่ root ทำให้ทั้ง tree re-render), แยก component |
| list ยาวหลายพันแถว | Profiler | pagination ฝั่ง server หรือ virtualization (`@tanstack/react-virtual`) |
| request ซ้ำ/waterfall | Network tab | TanStack Query dedupe, `Promise.all`, fetch ที่ server |

`useMemo`, `useCallback`, `memo` — **ใช้เมื่อวัดแล้วว่าช้าเท่านั้น** การใส่ทุกที่ทำให้โค้ดอ่านยากและอาจช้ากว่าเดิม (React Compiler ในเวอร์ชันใหม่ทำ memo ให้อัตโนมัติด้วย)

**Core Web Vitals** ที่ควรรู้ชื่อ: **LCP** (เนื้อหาหลักแสดงเร็วแค่ไหน), **INP** (ตอบสนองต่อการกดเร็วแค่ไหน), **CLS** (layout กระโดดไหม — แก้ด้วยการกำหนดขนาดรูปและ skeleton)

### Accessibility checklist

- [ ] ทุก input มี `<label>`, ทุกรูปมี `alt`
- [ ] ปุ่มคือ `<button>` ลิงก์คือ `<a>`/`<Link>`
- [ ] กด Tab ไล่ได้ทั้งหน้า เห็น focus ring ชัด
- [ ] สีตัวอักษรตัดกับพื้นพอ (Lighthouse ตรวจให้)
- [ ] สถานะไม่สื่อด้วยสีอย่างเดียว (badge สีแดงต้องมีข้อความ "failed" ด้วย)

## 7.4 คำถามสัมภาษณ์ที่เจอบ่อย พร้อมแนวตอบ

ซ้อมตอบ**ออกเสียง** ข้อละไม่เกิน 2 นาที โครงที่ดี: **นิยาม → ทำไม/แก้ปัญหาอะไร → ตัวอย่างจากโปรเจกต์ของตัวเอง**

### JavaScript / TypeScript

**1. `==` กับ `===` ต่างกันยังไง**
`==` แปลง type ก่อนเทียบ (`0 == ""` เป็น true) `===` ไม่แปลง ใช้ `===` เสมอเพื่อไม่ให้เกิดการแปลงที่คาดไม่ถึง

**2. อธิบาย event loop**
JS รันบน thread เดียว งาน I/O ถูกส่งให้ browser/runtime ทำ เมื่อเสร็จ callback เข้าคิว event loop หยิบมารันเมื่อ call stack ว่าง (microtask เช่น Promise ก่อน macrotask เช่น `setTimeout`) ผลคือ I/O ไม่ block แต่งาน CPU หนักจะ block ทั้งหน้า

**3. Promise กับ async/await**
async/await คือ syntax ที่อยู่บน Promise ทำให้เขียนโค้ด async เหมือน sync และใช้ `try/catch` ได้ งานที่ไม่ขึ้นต่อกันควรรันพร้อมกันด้วย `Promise.all` ไม่ใช่ `await` ต่อกันทีละตัว

**4. `interface` กับ `type` ใน TypeScript**
ใช้แทนกันได้เกือบทั้งหมด `interface` ขยาย/merge ได้ `type` ทำ union และ mapped type ได้ ทีมส่วนใหญ่เลือกแบบใดแบบหนึ่งให้สม่ำเสมอ

**5. `any` กับ `unknown`**
`any` ปิด type check, `unknown` บังคับให้ตรวจ type ก่อนใช้ — ข้อมูลจากภายนอกควรเป็น `unknown` แล้ว validate (เช่นด้วย zod)

### React

**6. Virtual DOM / reconciliation คืออะไร**
เมื่อ state เปลี่ยน React เรียก component ใหม่ได้ tree ของ element ใหม่ เทียบกับของเดิมแล้วแก้ DOM จริงเฉพาะจุดที่ต่าง เพราะการแก้ DOM แพงกว่าการเทียบ object ใน JS

**7. ทำไม list ต้องมี `key` และทำไมไม่ควรใช้ index**
`key` บอก React ว่าแถวไหนคือตัวเดิมเมื่อ list เปลี่ยน ถ้าใช้ index แล้วลบหรือสลับลำดับ แถวจะถูกจับคู่ผิด state ภายในของแถว (เช่น input) จะไปอยู่ผิดแถว ควรใช้ id ที่คงที่

**8. ทำไมต้องอัปเดต state แบบ immutable**
React ตัดสินว่าต้อง re-render หรือไม่ด้วยการเทียบ reference ถ้าแก้ object เดิม reference ไม่เปลี่ยน React จึงไม่รู้ ต้องสร้าง object/array ใหม่ด้วย spread/map/filter

**9. `useEffect` ใช้เมื่อไหร่ และ dependency array ทำงานยังไง**
ใช้ซิงก์กับระบบภายนอก (subscription, timer, DOM API) deps บอกว่า effect ต้องรันใหม่เมื่อค่าไหนเปลี่ยน ต้องมี cleanup เพื่อกัน memory leak และ race condition — **และบอกด้วยว่าไม่ควรใช้ effect** คำนวณ derived state หรือ fetch ข้อมูลในงานจริง (ใช้ TanStack Query หรือ server component)

**10. State management: เลือกอะไรเมื่อไหร่**
แยก server state (TanStack Query) กับ client state ก่อน client state: local `useState` → ยกขึ้น parent → Context (ข้อมูลกว้างเปลี่ยนน้อย เช่น theme) → Zustand/Redux (client state ซับซ้อนที่แชร์กันมาก) — ส่วนใหญ่ไม่ต้องถึงขั้นสุดท้าย

**11. Controlled กับ uncontrolled component**
controlled = ค่าอยู่ใน React state (`value` + `onChange`) uncontrolled = DOM เก็บค่าเอง อ่านผ่าน ref react-hook-form ใช้แบบ uncontrolled จึงไม่ re-render ทุกตัวอักษร

**12. ทำไม component re-render และลดยังไง**
re-render เมื่อ state ของตัวเองเปลี่ยน, parent re-render, หรือ context ที่ใช้เปลี่ยน ลดโดยวาง state ให้ใกล้ที่ใช้, แยก component, และใช้ `memo` เมื่อวัดแล้วว่าจำเป็น

### Next.js

**13. CSR / SSR / SSG / ISR ต่างกันยังไง**
(ดูตารางบทที่ 5.5) ตอบพร้อมตัวอย่างว่าหน้าไหนในโปรเจกต์ใช้แบบไหนและทำไม

**14. Server Component กับ Client Component**
Server Component รันบน server เท่านั้น fetch/เข้าถึง secret ได้ ไม่เพิ่ม JS ใน browser แต่ใช้ state/event ไม่ได้ Client Component (`"use client"`) interactive ได้ ยังถูก pre-render เป็น HTML บน server แล้ว hydrate ใน browser — วาง client boundary ให้ลึกที่สุด

**15. Hydration คืออะไร hydration error เกิดจากอะไร**
React ใน browser ผูก event เข้ากับ HTML ที่ server render มา ถ้า HTML จาก server กับผล render แรกใน browser ไม่ตรงกัน (ใช้ `Date.now()`, `Math.random()`, `window` ตอน render) จะเกิด hydration error — แก้โดยย้ายไปไว้ใน `useEffect` หรือทำให้ค่าตรงกัน

### Fullstack (จุดที่คุณได้เปรียบ — ตอบให้ลึก)

**16. CORS คืออะไร แก้ยังไง**
นโยบายของ browser ที่ห้าม JS อ่าน response ข้าม origin เว้นแต่ server อนุญาตผ่าน header แก้ที่ server: ระบุ origin ที่อนุญาต (ห้าม `*` ถ้าใช้ cookie), `Allow-Credentials`, ตอบ preflight `OPTIONS` หรือใช้ reverse proxy ให้เป็น origin เดียว

**17. เก็บ token ไว้ที่ไหนดี**
web app: JWT ใน httpOnly cookie + SameSite + Secure เพราะ JS (และ XSS) อ่านไม่ได้ ป้องกัน CSRF ด้วย SameSite และ/หรือ CSRF token; localStorage ง่ายแต่โดน XSS ขโมย token ได้

**18. ป้องกันหน้าที่ต้อง login ยังไง**
อธิบาย 3 ชั้นในบทที่ 6.4 — และย้ำว่า security จริงอยู่ที่ backend

**19. ออกแบบ API ให้ frontend ใช้ง่ายยังไง**
response shape สม่ำเสมอ, error format เดียวกันพร้อม field-level error สำหรับ form, pagination แบบ cursor, ID ใหญ่ส่งเป็น string (number ของ JS เพี้ยนเกิน 2^53), เวลาเป็น ISO 8601 UTC, OpenAPI spec เพื่อ generate type ฝั่ง frontend

**20. เล่าโปรเจกต์ที่ภูมิใจ**
ใช้โครง **STAR** (Situation, Task, Action, Result) ไม่เกิน 2 นาที เน้นการตัดสินใจทางเทคนิค 2–3 ข้อพร้อมเหตุผล เช่น "เลือก httpOnly cookie แทน localStorage เพราะ…", "แยก server/client component ตาม…" และสิ่งที่จะทำต่างไปถ้าทำใหม่

### คำถามแบบลงมือทำ (live coding) ที่เจอบ่อย

ฝึกทำแต่ละข้อให้เสร็จใน 20–30 นาทีโดยไม่ใช้ AI:

- [ ] Todo list (เพิ่ม/ลบ/toggle/filter)
- [ ] Search box ที่ debounce แล้วเรียก API แสดง loading/error/empty
- [ ] ดึงข้อมูลแบบแบ่งหน้า (pagination) พร้อมปุ่ม next/prev
- [ ] Form สมัครสมาชิกพร้อม validation
- [ ] Modal ที่ปิดได้ด้วยปุ่ม Esc และคลิกนอกกล่อง
- [ ] Countdown timer (ทดสอบความเข้าใจ `useEffect` cleanup และ stale closure)

## 7.5 Checklist ก่อนส่ง Portfolio

- [ ] README: ภาพ screenshot, diagram สถาปัตยกรรม, วิธีรัน (คำสั่งเดียว), tech stack, การตัดสินใจทางเทคนิคพร้อมเหตุผล
- [ ] มี demo ที่เปิดได้จริง (หรือวิดีโอ 2 นาที)
- [ ] ไม่มี secret ใน repo (`.env.local` อยู่ใน `.gitignore`, มี `.env.example`)
- [ ] `npm run lint`, `npm run typecheck`, `npm run test`, `npm run build` ผ่านทั้งหมด
- [ ] มี test อย่างน้อย: pure function, component, form
- [ ] Loading / error / empty state ครบทุกหน้า
- [ ] ใช้บนมือถือได้
- [ ] Commit history อ่านรู้เรื่อง (ไม่ใช่ "fix", "fix2", "wip" ทั้งหมด)

## สรุปบท

- Test pure function ก่อน แยก logic ออกจาก component ให้มากที่สุด, test component แบบที่ user ใช้
- FSD = clean architecture ฝั่ง frontend: ชั้น, dependency ไหลทางเดียว, public API
- วัดก่อน optimize; accessibility เป็นนิสัย ไม่ใช่งานเพิ่ม
- สัมภาษณ์: นิยาม → ทำไม → ตัวอย่างจากโปรเจกต์ตัวเอง และใช้ความรู้ backend เป็นจุดขาย
