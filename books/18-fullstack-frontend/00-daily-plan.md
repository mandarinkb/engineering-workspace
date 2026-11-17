# แผนรายวัน 8 สัปดาห์

## รูปแบบของแต่ละสัปดาห์

| วัน | เวลา | รูปแบบ |
|---|---|---|
| จันทร์ | 21:00–22:15 | ทฤษฎี + โค้ด |
| อังคาร | 21:00–22:15 | ทฤษฎี + โค้ด |
| **พุธ** | — | **พัก** |
| พฤหัส | 21:00–22:15 | ทฤษฎี + โค้ด |
| ศุกร์ | 21:00–22:15 | ทฤษฎีเบาๆ 45 นาที + **ทบทวน backend 30 นาที** (กันความรู้ backend หาย — งานแรกหลัง layoff มีโอกาสเป็น backend สูง) |
| เสาร์ (หรืออาทิตย์) | 2–2.5 ชม. ช่วงกลางวัน | Mini project / milestone |
| **อีกวัน** | — | **พักเต็มวัน** |

แต่ละคืนแบ่งเป็น:

```text
21:00–21:25  ทฤษฎี   อ่านบทในเล่มนี้ (+ ลิงก์ทางการถ้ามีเวลา)
21:25–22:05  โค้ด    ให้ AI เขียนตาม prompt → อ่าน → อธิบายกลับ → "ลองแก้เอง"
22:05–22:15  สรุป    ตอบคำถามท้าย session ลงไฟล์ NOTES.md (3–5 บรรทัดพอ)
22:15        ปิดเครื่อง
```

โค้ดฝึกทุกอย่างอยู่ใต้ `source-code/frontend/` (ดูโครงสร้างที่ [README ของโฟลเดอร์นั้น](../../source-code/frontend/README.md))

> **ตามไม่ทัน?** เลื่อนแผนทั้งหมดออกไป อย่าอัดสองวันในคืนเดียว สัปดาห์ 1–2 ตัดได้ถ้ารู้อยู่แล้ว สัปดาห์ 3–6 **ห้ามตัด** เป็นแกนของทั้งหมด

---

## สัปดาห์ 1 — HTML/CSS Layout และ JavaScript เบื้องต้น

**จบสัปดาห์ต้องทำได้:** จัดหน้าเว็บ responsive ด้วย Flexbox/Grid ได้เอง และอ่าน JS สมัยใหม่ออก

### W1·D1 จันทร์ — Setup + Semantic HTML

- [ ] **Setup (15 นาที):** เช็ค `node -v` (เครื่องนี้มี v26 แล้ว ใช้ได้เลย), VS Code extension: ESLint, Prettier, Tailwind CSS IntelliSense, สร้างไฟล์ `source-code/frontend/NOTES.md`
- [ ] **ทฤษฎี:** [บทที่ 1](01-html-css-layout.md) หัวข้อ 1.1–1.2 (semantic HTML, form)
- [ ] **โค้ด:** สร้าง `source-code/frontend/w01-resume/index.html` — หน้า resume ของตัวเอง **HTML ล้วน ยังไม่ต้องมี CSS**: `header`, `nav`, `main`, `section` (ประสบการณ์, skill), `footer`, ฟอร์มติดต่อ (name, email, message)
  - AI prompt: `เขียนโครง HTML semantic สำหรับหน้า resume ของ backend developer มี section ประสบการณ์ skill และฟอร์มติดต่อ ยังไม่ต้องมี CSS อธิบายว่าทำไมเลือกแต่ละ tag`
- [ ] **ลองแก้เอง:** ใส่ข้อมูลจริงของตัวเอง, เพิ่ม `required` และ `type="email"` ให้ฟอร์ม แล้วลอง submit แบบว่าง
- [ ] **สรุป:** `div` กับ `section` ต่างกันยังไง? `label for` ช่วยอะไร?

### W1·D2 อังคาร — Box Model + Flexbox

- [ ] **ทฤษฎี:** บทที่ 1 หัวข้อ 1.3–1.4
- [ ] **วอร์มอัพ (10 นาที):** [Flexbox Froggy](https://flexboxfroggy.com/) ให้ถึงด่าน 12
- [ ] **โค้ด:** เพิ่ม `style.css` ให้ w01-resume — navbar แนวนอน (โลโก้ซ้าย เมนูขวา), แถว skill เป็น "chip" เรียงต่อกันและขึ้นบรรทัดใหม่เองเมื่อเต็ม (`flex-wrap`)
- [ ] **ลองแก้เอง:** เปิด DevTools (Cmd+Opt+I) → แท็บ Elements → คลิก element ดู box model, ลองแก้ `padding`/`margin` สดในเบราว์เซอร์
- [ ] **สรุป:** `justify-content` กับ `align-items` ต่างกันยังไง? `box-sizing: border-box` แก้ปัญหาอะไร?

### W1·D3 พฤหัส — Grid + Responsive

- [ ] **ทฤษฎี:** บทที่ 1 หัวข้อ 1.5–1.6
- [ ] **วอร์มอัพ (10 นาที):** [Grid Garden](https://cssgridgarden.com/) ให้ถึงด่าน 10
- [ ] **โค้ด:** section "Projects" เป็น card grid — มือถือ 1 คอลัมน์, tablet 2, desktop 3 (mobile-first + media query หรือ `repeat(auto-fill, minmax(...))`)
- [ ] **ลองแก้เอง:** DevTools → Toggle device toolbar (Cmd+Shift+M) ทดสอบที่ความกว้าง 375px ให้ไม่มี scroll แนวนอน
- [ ] **สรุป:** เมื่อไหร่ใช้ Flexbox เมื่อไหร่ใช้ Grid?

### W1·D4 ศุกร์ — JavaScript ต่างจาก Go ตรงไหน

- [ ] **ทฤษฎี (45 นาที):** [บทที่ 2](02-javascript-typescript.md) หัวข้อ 2.1–2.2
- [ ] **โค้ด:** สร้าง `w01-js/basics.js` รันด้วย `node basics.js` — ลองทุกตัวอย่างในหัวข้อ 2.1 ด้วยตัวเอง (โดยเฉพาะ `==` vs `===`, object เป็น reference, closure)
- [ ] **Backend 30 นาที:** ทบทวน Go — เลือก 1 เรื่องจาก [Volume 4](../04-golang/README.md) (เช่น interface, context)
- [ ] **สรุป:** ทำไม `const arr = []; arr.push(1)` ไม่ error?

### W1·D5 เสาร์ — Array methods + ปิดงาน resume (2 ชม.)

- [ ] **ทฤษฎี:** บทที่ 2 หัวข้อ 2.3
- [ ] **โค้ด 1 (45 นาที):** `w01-js/jobs.js` — สร้าง array ของ job ปลอม 10 ตัว (`{id, bot, status, created_at}`) แล้วเขียนฟังก์ชันด้วย `filter`/`map`/`reduce`/`sort`: นับตามสถานะ, หา success rate, หา 5 job ที่ fail ล่าสุด
- [ ] **อ่านโค้ดจริง (15 นาที):** เทียบกับ `projects/bop-dashboard/src/entities/job/lib/job-stats.ts` — โค้ดจริงทำแบบเดียวกันไหม? (ยังไม่ต้องเข้าใจ syntax TypeScript)
- [ ] **โค้ด 2 (45 นาที):** ทำ w01-resume ให้เสร็จและดูดีบนมือถือ
- [ ] **Checkpoint สัปดาห์ 1:** ทำ layout แบบนี้ได้โดยไม่ต้องถาม AI ไหม? (navbar + grid 3 คอลัมน์ที่ยุบเหลือ 1 บนมือถือ) — ถ้าไม่ได้ ใช้วันอาทิตย์ 30 นาทีทวนเท่านั้น

---

## สัปดาห์ 2 — Async, DOM, TypeScript + Go API ของเราเอง

**จบสัปดาห์ต้องทำได้:** เขียน TypeScript ดึงข้อมูลจาก API แล้ว render ลงหน้าเว็บได้โดยไม่ใช้ framework

### W2·D1 จันทร์ — Event loop, Promise, async/await, fetch

- [ ] **ทฤษฎี:** บทที่ 2 หัวข้อ 2.4
- [ ] **โค้ด:** `w02-async/fetch.mjs` — ดึง `https://jsonplaceholder.typicode.com/todos?_limit=5` ด้วย `fetch` + `async/await`, จัดการ `res.ok` เป็น false, จัดการ network error ด้วย `try/catch`, ลองยิง 3 request พร้อมกันด้วย `Promise.all`
- [ ] **ทำให้พัง:** ลบ `await` หน้า `res.json()` แล้ว `console.log` ดูว่าได้อะไร
- [ ] **สรุป:** `Promise.all` คล้ายอะไรใน Go? (คำใบ้: `errgroup`) `fetch` throw เมื่อไหร่ และ**ไม่** throw เมื่อไหร่?

### W2·D2 อังคาร — DOM + Event + Vite

- [ ] **ทฤษฎี:** บทที่ 2 หัวข้อ 2.5
- [ ] **โค้ด:** `cd source-code/frontend && npm create vite@latest w02-vanilla -- --template vanilla-ts` → หน้าแสดงรายการ todo จาก jsonplaceholder, มี input + ปุ่ม "เพิ่ม" (เพิ่มเข้า array ในหน่วยความจำแล้ว render ใหม่)
  - AI prompt: `เขียน main.ts ด้วย vanilla TypeScript (ไม่ใช้ framework) ดึง todo จาก jsonplaceholder แล้ว render เป็น <ul> มี form เพิ่ม todo ใหม่ที่ render list ใหม่ทั้งหมดทุกครั้ง ใช้ event delegation สำหรับปุ่มลบ`
- [ ] **ลองแก้เอง:** เพิ่มปุ่ม toggle done (ขีดฆ่าข้อความ)
- [ ] **สรุป:** การต้อง "render ใหม่เองทุกครั้งที่ข้อมูลเปลี่ยน" เจ็บตรงไหน? (นี่คือปัญหาที่ React มาแก้ — จำความรู้สึกนี้ไว้)

### W2·D3 พฤหัส — TypeScript พื้นฐาน

- [ ] **ทฤษฎี:** บทที่ 2 หัวข้อ 2.6 (type, interface, union, literal type, narrowing)
- [ ] **โค้ด:** ใน w02-vanilla ประกาศ `interface Todo`, type ของ response, เปิด `strict` ใน `tsconfig.json` แล้วแก้ทุก error จน `npx tsc --noEmit` ผ่าน
- [ ] **ลองแก้เอง:** เพิ่ม `type Status = "todo" | "doing" | "done"` แทน boolean `completed` แล้วดูว่า compiler บังคับให้แก้ตรงไหนบ้าง
- [ ] **สรุป:** TypeScript type หายไปตอน runtime — แปลว่าข้อมูลจาก API ที่ผิด shape จะเกิดอะไรขึ้น? (คำตอบนำไปสู่ zod ในสัปดาห์ 5)

### W2·D4 ศุกร์ — Generic, `unknown` + อ่าน api-client จริง

- [ ] **ทฤษฎี (45 นาที):** บทที่ 2 หัวข้อ 2.7 (generic, utility type, `unknown` vs `any`)
- [ ] **อ่านโค้ดจริง:** `projects/bop-dashboard/src/shared/api/api-client.ts` และ `api-error.ts` — อ่านคอมเมนต์ทั้งหมด เข้าใจว่าทำไม `apiClient<T>` เป็น generic, ทำไม `credentials: "include"`, ทำไมเช็ค 204
- [ ] **Backend 30 นาที:** ทบทวน SQL/PostgreSQL ([Volume 5](../05-postgresql/README.md)) — index, transaction isolation
- [ ] **สรุป:** เขียน `apiClient` แบบย่อของตัวเองลง NOTES จากความจำ (ไม่ต้องรันได้)

### W2·D5 เสาร์ — Milestone 1: Go tasks-api + vanilla TS client (2.5 ชม.)

- [ ] **โค้ด Go (1 ชม.):** ให้ AI เขียน `source-code/frontend/tasks-api/` (Go, `net/http` ล้วน, in-memory store, mutex) ส่วนนี้คุณถนัด — ตรวจโค้ดแบบ reviewer
  - endpoint: `GET /tasks`, `GET /tasks/{id}`, `POST /tasks`, `PATCH /tasks/{id}`, `DELETE /tasks/{id}`
  - Task: `{id, title, status: "todo"|"doing"|"done", created_at}`
  - **มี CORS middleware** อนุญาต `http://localhost:5173` (port ของ Vite) — ยังไม่ต้องเข้าใจลึก บทที่ 6 จะอธิบาย
- [ ] **โค้ด frontend (1 ชม.):** เปลี่ยน w02-vanilla ให้ยิง tasks-api แทน jsonplaceholder: list/add/toggle/delete
- [ ] **ทำให้พัง:** ปิด CORS middleware ใน Go แล้วดู error ใน Console ของเบราว์เซอร์ — สังเกตว่า request **ถึง** server แต่เบราว์เซอร์ไม่ยอมให้ JS อ่าน response
- [ ] **Checkpoint สัปดาห์ 2:** มี backend ของตัวเอง 1 ตัว ที่จะใช้ต่อถึงสัปดาห์ 8 ✅

---

## สัปดาห์ 3 — React Fundamentals

**จบสัปดาห์ต้องทำได้:** สร้าง UI แบบ component ที่มี state, list, form ได้

### W3·D1 จันทร์ — ทำไมต้อง React, Component, JSX, Props

- [ ] **ทฤษฎี:** [บทที่ 3](03-react-fundamentals.md) หัวข้อ 3.1–3.2
- [ ] **โค้ด:** `npm create vite@latest react-tasks -- --template react-ts` → แตก UI เป็น component: `<App>`, `<Header title=...>`, `<TaskItem task=...>` (ใช้ข้อมูล hard-code array ก่อน)
- [ ] **ลองแก้เอง:** เพิ่ม prop `onDelete` ให้ `TaskItem` (ยังไม่ต้องทำงานจริง แค่ `console.log`)
- [ ] **สรุป:** props เทียบกับ argument ของฟังก์ชัน Go — ทำไม component แก้ props ตัวเองไม่ได้?

### W3·D2 อังคาร — State และการ re-render

- [ ] **ทฤษฎี:** บทที่ 3 หัวข้อ 3.3
- [ ] **โค้ด:** ย้าย tasks เข้า `useState`, ทำ input + ปุ่มเพิ่ม (controlled input), ปุ่มลบ, ปุ่ม toggle status
- [ ] **ทำให้พัง:** เปลี่ยนจาก `setTasks([...tasks, newTask])` เป็น `tasks.push(newTask); setTasks(tasks)` — ทำไมหน้าจอไม่อัปเดต?
- [ ] **สรุป:** "state เปลี่ยน → component function ถูกเรียกใหม่" หมายความว่าตัวแปรธรรมดาในฟังก์ชันเป็นยังไงหลัง re-render?

### W3·D3 พฤหัส — Event, List + key, Conditional rendering

- [ ] **ทฤษฎี:** บทที่ 3 หัวข้อ 3.4
- [ ] **โค้ด:** แสดง "ยังไม่มีงาน" เมื่อ list ว่าง, badge สีตาม status (ใช้ object map แบบ `job-status-badge.tsx` ใน bop-dashboard), นับจำนวนงานที่ done
- [ ] **ทำให้พัง:** ใช้ `key={index}` แล้วลบรายการแรกขณะที่ input ในรายการมีข้อความค้างอยู่ — ดูว่าข้อความย้ายไปอยู่ผิดแถว
- [ ] **สรุป:** ทำไม `key` ต้องเป็น id ที่คงที่?

### W3·D4 ศุกร์ — Lifting state, composition, children

- [ ] **ทฤษฎี (45 นาที):** บทที่ 3 หัวข้อ 3.5 + อ่าน [Thinking in React](https://react.dev/learn/thinking-in-react)
- [ ] **โค้ด (สั้น):** แยก `<TaskForm onAdd>` ออกจาก `<App>` — state อยู่ที่ App, ลูกส่ง event ขึ้นมาผ่าน callback
- [ ] **Backend 30 นาที:** ทบทวน auth/JWT ([Volume 9](../09-authentication/README.md)) — จะได้ใช้ในสัปดาห์ 7
- [ ] **สรุป:** state ควรอยู่ที่ component ไหน? (กฎ: ตัวที่ใกล้ที่สุดที่เป็น parent ร่วมของทุกตัวที่ใช้)

### W3·D5 เสาร์ — Derived state + filter (2 ชม.)

- [ ] **โค้ด:** เพิ่มแท็บ filter All / Todo / Doing / Done และช่องค้นหา — **ห้ามเก็บ filtered list ใน state** ให้คำนวณจาก `tasks` + `filter` ทุก render
- [ ] **อ่านโค้ดจริง:** `projects/bop-dashboard/src/shared/ui/stat-card.tsx`, `status-badge.tsx`, `src/entities/bot/ui/bot-card.tsx` — สังเกตว่า component เล็กๆ รับ props แล้วแสดงผลอย่างเดียว
- [ ] **ลองแก้เอง:** แยก `<TaskStats tasks>` แสดงจำนวนแต่ละสถานะ
- [ ] **Checkpoint สัปดาห์ 3:** เขียน counter + todo list ด้วย React จากศูนย์ใน 20 นาทีโดยไม่ดูโค้ดเก่า ได้ไหม?

---

## สัปดาห์ 4 — Effect, Custom Hook, Context, Tailwind

**จบสัปดาห์ต้องทำได้:** ต่อ React เข้ากับ API จริง และเข้าใจปัญหาของ `useEffect`

### W4·D1 จันทร์ — useEffect: ซิงก์กับโลกภายนอก

- [ ] **ทฤษฎี:** บทที่ 3 หัวข้อ 3.6
- [ ] **โค้ด:** react-tasks ดึง tasks จาก tasks-api ด้วย `useEffect` + `fetch` เอง (แบบ manual) — มี state `loading`, `error`, `data` (เพิ่ม `http://localhost:5173` ใน CORS ของ Go ถ้ายังไม่มี)
- [ ] **ลองแก้เอง:** ปุ่มเพิ่ม/ลบ ต้องยิง API แล้ว refetch
- [ ] **สรุป:** dependency array `[]` กับไม่ใส่เลยต่างกันยังไง?

### W4·D2 อังคาร — กับดักของ useEffect + useRef

- [ ] **ทฤษฎี:** บทที่ 3 หัวข้อ 3.7
- [ ] **ทำให้พัง 3 แบบ (ทำทีละแบบ แล้วแก้กลับ):**
  1. ลบ dependency array → infinite loop (ดู Network tab)
  2. ทำช่องค้นหาที่ยิง API ทุกตัวอักษร แล้วหน่วงฝั่ง Go แบบสุ่ม → ผลลัพธ์เก่าทับผลใหม่ (race condition) → แก้ด้วย `AbortController` ใน cleanup
  3. สังเกตว่า dev mode ยิง request 2 ครั้ง (StrictMode) — ทำไม?
- [ ] **สรุป:** เขียนลง NOTES: "useEffect ใช้เมื่อ ... ไม่ควรใช้เมื่อ ..."

### W4·D3 พฤหัส — Custom hook

- [ ] **ทฤษฎี:** บทที่ 3 หัวข้อ 3.8
- [ ] **โค้ด:** ย้าย logic fetch ไปเป็น `useTasks()` คืน `{ data, loading, error, refetch }`, เขียน `useDebounce(value, delay)` สำหรับช่องค้นหา
- [ ] **สรุป:** custom hook ต่างจาก function ธรรมดายังไง? ทำไมต้องขึ้นต้นด้วย `use`?

### W4·D4 ศุกร์ — Context

- [ ] **ทฤษฎี (45 นาที):** บทที่ 3 หัวข้อ 3.9
- [ ] **โค้ด (สั้น):** `ThemeContext` สลับ light/dark
- [ ] **อ่านโค้ดจริง:** `projects/bop-dashboard/src/app/providers/providers.tsx` — อ่านคอมเมนต์เรื่องลำดับ provider
- [ ] **Backend 30 นาที:** ทบทวน Docker/Compose ([Volume 7](../07-docker/README.md)) — จะได้ใช้ในสัปดาห์ 7
- [ ] **สรุป:** ทำไมไม่ควรเอาทุกอย่างใส่ Context?

### W4·D5 เสาร์ — Tailwind CSS (2 ชม.)

- [ ] **ทฤษฎี:** บทที่ 1 หัวข้อ 1.7
- [ ] **โค้ด:** ติดตั้ง Tailwind v4 ใน react-tasks ([คู่มือ Vite](https://tailwindcss.com/docs/installation/using-vite)) แล้ว restyle ทั้งแอป: card, ปุ่ม, form, badge, responsive, dark mode (`dark:`)
- [ ] **อ่านโค้ดจริง:** `projects/bop-dashboard/src/shared/ui/` ทั้งโฟลเดอร์ — ดูว่า class Tailwind ถูกจัดระเบียบยังไง
- [ ] **Checkpoint สัปดาห์ 4:** react-tasks คุยกับ Go API ได้ครบ CRUD และหน้าตาใช้ได้ ✅

---

## สัปดาห์ 5 — TanStack Query, Form, Routing

**จบสัปดาห์ต้องทำได้:** เขียนแอป React แบบที่ใช้ในงานจริง (ไม่มี `useEffect` ยิง API เองแล้ว)

### W5·D1 จันทร์ — Server state กับ useQuery

- [ ] **ทฤษฎี:** [บทที่ 4](04-react-data-forms-routing.md) หัวข้อ 4.1–4.2
- [ ] **โค้ด:** ติดตั้ง `@tanstack/react-query` + devtools, เปลี่ยน `useTasks()` ไปใช้ `useQuery` — ลบ `useState` loading/error ของเดิมทิ้ง
- [ ] **ลองแก้เอง:** สลับแท็บเบราว์เซอร์ไปมา ดู devtools ว่า refetch เมื่อไหร่ (`refetchOnWindowFocus`) ลองตั้ง `staleTime`
- [ ] **สรุป:** `queryKey` ทำหน้าที่อะไร? เทียบกับ cache key ใน Redis

### W5·D2 อังคาร — useMutation + invalidation

- [ ] **ทฤษฎี:** บทที่ 4 หัวข้อ 4.3
- [ ] **โค้ด:** `useCreateTask`, `useUpdateTask`, `useDeleteTask` ด้วย `useMutation` + `invalidateQueries(["tasks"])`, ปุ่ม disabled ระหว่าง `isPending`
- [ ] **อ่านโค้ดจริง:** `projects/bop-dashboard/src/features/create-job/api/use-create-job.ts`, `src/features/cancel-job/api/use-cancel-job.ts`
- [ ] **สรุป:** ทำไมหลัง mutate ต้อง invalidate แทนที่จะแก้ state เอง?

### W5·D3 พฤหัส — Form: react-hook-form + zod

- [ ] **ทฤษฎี:** บทที่ 4 หัวข้อ 4.4
- [ ] **โค้ด:** `TaskForm` ใหม่ด้วย react-hook-form + zod: title ต้องยาว 3–100 ตัว, status เป็น enum, แสดง error ใต้ช่อง, `z.infer` สร้าง type
- [ ] **อ่านโค้ดจริง:** `projects/bop-dashboard/src/entities/bot/model/bot-config.schema.ts` + `src/features/schedule-form/ui/schedule-form.tsx`
- [ ] **สรุป:** validate ฝั่ง frontend แล้ว ทำไม Go ยังต้อง validate ซ้ำ?

### W5·D4 ศุกร์ — Client-side routing

- [ ] **ทฤษฎี (45 นาที):** บทที่ 4 หัวข้อ 4.5
- [ ] **โค้ด (สั้น):** ติดตั้ง `react-router` → หน้า `/` (list), `/tasks/:id` (detail ใช้ `useQuery(["tasks", id])`), `/tasks/new`
- [ ] **Backend 30 นาที:** System design 1 โจทย์ ([Volume 17](../17-system-design/README.md)) — อธิบายออกเสียง 10 นาที
- [ ] **สรุป:** SPA routing ทำงานยังไงทั้งที่ไม่มีการโหลดหน้าใหม่จาก server?

### W5·D5 เสาร์ — Milestone 2: react-tasks สมบูรณ์ (2.5 ชม.)

- [ ] **โค้ด:** เก็บงานให้เสร็จ — loading skeleton, error state + ปุ่ม retry, empty state, toast เมื่อสร้าง/ลบสำเร็จ, หน้า 404
- [ ] **อ่านโค้ดจริง (30 นาที):** `projects/bop-dashboard/src/entities/job/api/` ทั้งโฟลเดอร์ (โดยเฉพาะ `use-job-log-stream.ts` — SSE คือ server push ที่คุณรู้จากฝั่ง backend)
- [ ] **Checkpoint สัปดาห์ 5:** อธิบายให้คนอื่นฟังได้ว่า ทำไมงานจริงใช้ TanStack Query แทน `useEffect` ✅ — **ถึงตรงนี้คุณเขียน React ระดับ junior–mid ได้แล้ว**

---

## สัปดาห์ 6 — Next.js App Router

**จบสัปดาห์ต้องทำได้:** สร้างแอป Next.js และแยกได้ว่าโค้ดไหนรันที่ server โค้ดไหนรันที่เบราว์เซอร์

### W6·D1 จันทร์ — ทำไมต้อง Framework, Routing, Layout

- [ ] **ทฤษฎี:** [บทที่ 5](05-nextjs-app-router.md) หัวข้อ 5.1–5.2
- [ ] **โค้ด:** `npx create-next-app@latest next-tasks` (TypeScript, Tailwind, App Router, ESLint: Yes) → สร้าง `app/layout.tsx` ที่มี navbar, หน้า `/`, `/tasks`, `/about`, ใช้ `<Link>`
- [ ] **อ่านโค้ดจริง:** โครงสร้าง `projects/bop-dashboard/app/` — `(auth)` กับ `(dashboard)` คือ route group
- [ ] **สรุป:** layout ต่างจาก page ยังไง? ทำไมเปลี่ยนหน้าแล้ว navbar ไม่ render ใหม่?

### W6·D2 อังคาร — Server Component vs Client Component

- [ ] **ทฤษฎี:** บทที่ 5 หัวข้อ 5.3 (**หัวข้อสำคัญที่สุดของบท**)
- [ ] **โค้ด:** หน้า `/tasks` เป็น Server Component ที่ `await fetch("http://localhost:8080/tasks")` ตรงๆ (ไม่ต้องมี CORS เพราะยิงจาก server!) + ปุ่ม toggle เป็น Client Component แยกไฟล์ (`"use client"`)
- [ ] **ทำให้พัง:** ใส่ `useState` ใน Server Component แล้วอ่าน error message, ใส่ `console.log` ใน server component แล้วหาว่า log ไปโผล่ที่ไหน (terminal ไม่ใช่ browser)
- [ ] **สรุป:** วาดรูปลง NOTES: request จากเบราว์เซอร์ → Next server → Go API → HTML กลับมา

### W6·D3 พฤหัส — Dynamic route, loading/error, Rendering mode

- [ ] **ทฤษฎี:** บทที่ 5 หัวข้อ 5.4–5.5
- [ ] **โค้ด:** `app/tasks/[id]/page.tsx`, `loading.tsx` (หน่วง Go 2 วินาทีเพื่อเห็นผล), `error.tsx`, `not-found.tsx` (เรียก `notFound()` เมื่อ Go ตอบ 404)
- [ ] **ลองแก้เอง:** `npm run build` แล้วอ่านตาราง output ว่าหน้าไหนเป็น static (○) หน้าไหนเป็น dynamic (ƒ)
- [ ] **สรุป:** CSR / SSR / SSG ต่างกันยังไง แต่ละแบบเหมาะกับหน้าแบบไหน?

### W6·D4 ศุกร์ — Route Handler, Server Action, Middleware

- [ ] **ทฤษฎี (45 นาที):** บทที่ 5 หัวข้อ 5.6–5.7
- [ ] **โค้ด (สั้น):** ลองทำ form สร้าง task ด้วย Server Action 1 ครั้ง (เพื่อให้รู้จัก) — โปรเจกต์หลักจะยังเรียก Go API เป็นแหล่งข้อมูลจริง
- [ ] **Backend 30 นาที:** ทบทวน Redis/Kafka 1 เรื่องที่อ่อนที่สุด
- [ ] **สรุป:** มี Go backend อยู่แล้ว ควรใช้ Route Handler/Server Action ทำอะไร และไม่ควรทำอะไร?

### W6·D5 เสาร์ — Milestone 3: ย้าย react-tasks เข้า Next.js (2.5 ชม.)

- [ ] **โค้ด:** ย้าย component, hook, form จาก react-tasks เข้า next-tasks — สร้าง `Providers` (QueryClientProvider) แบบเดียวกับ `providers.tsx` ของ bop-dashboard, หน้า list ใช้ TanStack Query ฝั่ง client, หน้า detail ใช้ server fetch
- [ ] **ตัดสินใจเอง:** หน้าไหนควรเป็น server/client? จดเหตุผลลง NOTES
- [ ] **Checkpoint สัปดาห์ 6:** อธิบายได้ว่า `"use client"` ไม่ได้แปลว่า "render แค่ฝั่ง client" ✅

---

## สัปดาห์ 7 — Fullstack: Auth, CORS, Database, Docker

**จบสัปดาห์ต้องทำได้:** ระบบ login ทำงานจริงข้าม Go ↔ Next.js และรันทั้งระบบด้วยคำสั่งเดียว

### W7·D1 จันทร์ — สถาปัตยกรรม + CORS ลึก

- [ ] **ทฤษฎี:** [บทที่ 6](06-fullstack-auth-deploy.md) หัวข้อ 6.1–6.2
- [ ] **โค้ด:** แก้ CORS ใน tasks-api ให้ถูกต้องสำหรับ cookie: `Access-Control-Allow-Origin` ระบุ origin ตรงๆ (ไม่ใช่ `*`), `Allow-Credentials: true`, รับ preflight `OPTIONS`
- [ ] **ทำให้พัง:** ตั้ง `Allow-Origin: *` พร้อม `credentials: "include"` แล้วอ่าน error ใน Console
- [ ] **สรุป:** preflight เกิดเมื่อไหร่? ทำไม curl ไม่เคยติด CORS?

### W7·D2 อังคาร — Auth ด้วย JWT ใน httpOnly cookie (ฝั่ง Go)

- [ ] **ทฤษฎี:** บทที่ 6 หัวข้อ 6.3
- [ ] **โค้ด Go:** `POST /login` (ตรวจ user/pass → set cookie `HttpOnly; SameSite=Lax`), `POST /logout`, `GET /me`, middleware ป้องกัน `/tasks/*`
- [ ] **อ่านโค้ดจริง:** ฝั่ง Go ของ BOP — `projects/bot-orchestration-platform/internal/pkg/jwt/jwt.go` และ `internal/adapter/http/handler.go` (ส่วน CORS ช่วงบรรทัด ~60–70 และส่วน cookie `bop_session` ช่วงบรรทัด ~305–370)
- [ ] **สรุป:** ทำไมไม่เก็บ JWT ใน `localStorage`?

### W7·D3 พฤหัส — Auth ฝั่ง Next.js

- [ ] **ทฤษฎี:** บทที่ 6 หัวข้อ 6.4
- [ ] **โค้ด:** หน้า `/login` (react-hook-form + zod + `useMutation`), hook `useMe()`, ปุ่ม logout, `middleware.ts` (หรือ `proxy.ts` ถ้าเป็น Next 16+) redirect เมื่อไม่มี cookie, จัดการ 401 → กลับหน้า login
- [ ] **อ่านโค้ดจริง:** `projects/bop-dashboard/middleware.ts`, `src/features/auth/`, `src/entities/user/model/use-auth.ts`
- [ ] **สรุป:** ทำไม middleware ของ Next **ไม่ใช่** security boundary?

### W7·D4 ศุกร์ — Environment variable + PostgreSQL

- [ ] **ทฤษฎี (45 นาที):** บทที่ 6 หัวข้อ 6.5
- [ ] **โค้ด:** ย้าย URL ของ API เป็น `NEXT_PUBLIC_API_URL` ใน `.env.local`, เพิ่ม `.env.example`; ให้ AI เปลี่ยน store ของ tasks-api เป็น PostgreSQL (`pgx`) — ตรวจโค้ดแบบ reviewer
- [ ] **Backend 30 นาที:** อยู่ในงานคืนนี้แล้ว (Postgres + Go) ✅
- [ ] **สรุป:** ตัวแปรที่ขึ้นต้น `NEXT_PUBLIC_` ถูกฝังไปไหน? ทำไมห้ามใส่ secret?

### W7·D5 เสาร์ — Milestone 4: Docker Compose ทั้งระบบ (2.5 ชม.)

- [ ] **ทฤษฎี:** บทที่ 6 หัวข้อ 6.6
- [ ] **โค้ด:** `Dockerfile` ของ Next (multi-stage + `output: "standalone"`), `Dockerfile` ของ Go, `docker-compose.yml` รวม postgres + api + web
- [ ] **รัน:** Mac ไม่มี Docker → รันบน Mini PC (`ssh minipc`, ใช้ `/usr/libexec/docker/cli-plugins/docker-compose` และเช็คพอร์ตด้วย `ss -ltn` ก่อน) หรือข้ามขั้นนี้แล้ว deploy frontend ขึ้น Vercel แทน
- [ ] **อ่านโค้ดจริง:** `projects/bop-dashboard/Dockerfile`
- [ ] **Checkpoint สัปดาห์ 7:** login → เห็น tasks ของตัวเอง → logout ทำงานครบ ✅ — **นี่คือ portfolio project ของคุณ**

---

## สัปดาห์ 8 — Testing, โครงสร้าง, Portfolio, สัมภาษณ์

**จบสัปดาห์ต้องทำได้:** มีโปรเจกต์พร้อมโชว์ + ตอบคำถามสัมภาษณ์ frontend ได้มั่นใจ

### W8·D1 จันทร์ — Testing

- [ ] **ทฤษฎี:** [บทที่ 7](07-testing-and-interview.md) หัวข้อ 7.1
- [ ] **โค้ด:** ติดตั้ง Vitest + Testing Library ใน next-tasks, เขียน test: pure function 1 ตัว (เช่น `computeTaskStats`), component 1 ตัว (`TaskStatusBadge`), form 1 ตัว (กด submit ว่างแล้วเห็น error)
- [ ] **อ่านโค้ดจริง:** `projects/bop-dashboard/vitest.config.ts`, `src/entities/job/lib/job-stats.test.ts`, `src/entities/job/ui/job-status-badge.test.tsx`
- [ ] **สรุป:** ทำไม Testing Library แนะนำให้หา element ด้วย role/text แทน class name?

### W8·D2 อังคาร — โครงสร้างโปรเจกต์ (Feature-Sliced Design)

- [ ] **ทฤษฎี:** บทที่ 7 หัวข้อ 7.2 + อ่าน `projects/bop-dashboard/ARCHITECTURE.md`
- [ ] **โค้ด:** จัด next-tasks เป็น `src/shared`, `src/entities/task`, `src/features/{create-task,auth}`, `src/widgets` (ไม่ต้องสมบูรณ์ ทำพอให้เข้าใจกฎ import)
- [ ] **สรุป:** FSD เทียบกับ clean architecture ใน Go ตรงไหน?

### W8·D3 พฤหัส — Performance + Accessibility

- [ ] **ทฤษฎี:** บทที่ 7 หัวข้อ 7.3
- [ ] **โค้ด:** รัน Lighthouse (Chrome DevTools) แก้คะแนน Accessibility ให้ ≥ 90, เปิด React DevTools Profiler ดูว่าพิมพ์ใน input แล้ว component ไหน re-render
- [ ] **สรุป:** `useMemo`/`memo` ควรใช้เมื่อไหร่? (คำตอบ: เมื่อวัดแล้วว่าช้า)

### W8·D4 ศุกร์ — ซ้อมสัมภาษณ์

- [ ] **ทฤษฎี + ซ้อม (75 นาที):** บทที่ 7 หัวข้อ 7.4 — ตอบออกเสียงทีละข้อ จับเวลาข้อละ 2 นาที ข้อไหนตอบไม่ได้ ให้ AI สัมภาษณ์ต่อ
  - AI prompt: `สัมภาษณ์ผมตำแหน่ง Fullstack (Go + React/Next.js) ระดับ mid ถามทีละข้อ รอผมตอบก่อน แล้วให้คะแนนพร้อมบอกจุดที่ขาด เริ่มจากคำถาม React`
- [ ] **สรุป:** จด 3 ข้อที่ตอบแย่ที่สุด ไว้ทวนอาทิตย์หน้า

### W8·D5 เสาร์ — Portfolio + Resume (2.5 ชม.)

- [ ] **README ของโปรเจกต์:** screenshot 3 รูป, diagram สถาปัตยกรรม (Next ↔ Go ↔ Postgres), วิธีรัน, การตัดสินใจทางเทคนิค 3–5 ข้อ (เช่น ทำไม JWT ใน httpOnly cookie)
- [ ] **Push ขึ้น GitHub** (repo แยก) + deploy demo ถ้าทำได้
- [ ] **Resume/LinkedIn:** ตำแหน่ง "Backend-focused Fullstack Engineer" + bullet ของโปรเจกต์นี้ + bop-dashboard (ถ้าเปิดเผยได้)
- [ ] **อัดวิดีโอ 2 นาที** อธิบายโปรเจกต์ (ซ้อมเล่าในสัมภาษณ์ได้ด้วย)
- [ ] 🎉 **จบ 8 สัปดาห์** — พักเต็มสุดสัปดาห์ที่เหลือ

---

## หลัง 8 สัปดาห์ (ถ้ายังมีเวลา)

ทำต่อทีละอย่าง สัปดาห์ละ 3 คืนพอ:

1. เพิ่ม feature ให้ portfolio: pagination ฝั่ง server, upload ไฟล์, realtime ด้วย SSE/WebSocket
2. E2E test ด้วย Playwright
3. ถ้าเจองานที่ใช้ Angular — เรียน Angular (component, signal, service, RxJS พื้นฐาน) แนวคิดจากเล่มนี้ใช้ต่อได้ 70%

## Template ไฟล์ NOTES.md

```markdown
## W1·D1 (2026-11-17)
- เรียน: semantic HTML, form
- เข้าใจแล้ว: label for ผูกกับ id ช่วยคลิกที่ข้อความแล้ว focus input + screen reader อ่านได้
- ยังงง: ...
- ค้างไว้: ...
```
