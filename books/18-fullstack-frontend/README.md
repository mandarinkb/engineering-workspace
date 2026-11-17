# Volume 18 — Fullstack (Frontend สำหรับ Backend Engineer)

เล่มนี้เขียนสำหรับคนที่เป็น backend อยู่แล้ว (Go, PostgreSQL, Docker, auth) และต้องการขยับเป็น fullstack แบบเร่งรัดใน **8 สัปดาห์** โดยใช้เวลาหลังเลิกงาน (21:00–22:30) โดยไม่กระทบการนอน

เนื้อหาจึงไม่ปูพื้นเรื่องที่ backend รู้อยู่แล้ว (HTTP, JSON, REST, JWT, SQL) แต่เจาะเรื่องที่ต่างออกไปเมื่อโค้ดของเรา **รันในเบราว์เซอร์ของ user**: layout, state, rendering, การดึงข้อมูลแบบ async, form และ boundary ระหว่าง server/client

## Stack ที่เลือก (และเหตุผล)

| ชั้น | เลือก | เหตุผล |
|---|---|---|
| ภาษา | **TypeScript** | ตลาดงานแทบทั้งหมดใช้ type system ใกล้ความรู้สึกของ Go |
| UI library | **React 19** | ตำแหน่งงานมากที่สุด |
| Framework | **Next.js (App Router)** | มาตรฐานของ React ฝั่ง production ปัจจุบัน |
| Styling | **Tailwind CSS v4** | ไม่ต้องเสียเวลาออกแบบ CSS เอง ได้ผลลัพธ์สวยเร็ว |
| Server state | **TanStack Query v5** | จัดการ loading/error/cache ของ API ให้หมด |
| Form | **react-hook-form + zod** | validate ด้วย schema เดียว ทั้ง type และ runtime |
| Test | **Vitest + Testing Library** | เร็ว ใช้แทน Jest ได้ |

ทั้งหมดนี้**ตรงกับ stack ของ [projects/bop-dashboard](../../projects/bop-dashboard/README.md)** ซึ่งมีอยู่ใน workspace แล้ว เราจึงใช้มันเป็น "โค้ดจริงไว้อ่าน" ตลอดเล่ม

> ถ้าเป้าหมายคือธนาคาร/ประกัน/องค์กรใหญ่ที่ใช้ **Angular** — แนวคิดบทที่ 1, 2, 4 (data fetching, form), 6, 7 ใช้ได้เหมือนเดิม เปลี่ยนแค่บทที่ 3 และ 5 เป็น Angular (component, signal, service, router) แต่อย่าเรียนสองตัวพร้อมกัน

## วิธีใช้เล่มนี้

1. เปิด **[แผนรายวัน](00-daily-plan.md)** ทุกคืน ทำทีละ session ติ๊ก checkbox
2. แต่ละ session มี 3 ช่วง: **ทฤษฎี (~25 นาที) → โค้ด (~40 นาที) → สรุป (~10 นาที)**
3. โค้ดฝึกทั้งหมดอยู่ที่ [source-code/frontend/](../../source-code/frontend/README.md)

### วิธีเรียนแบบ "ให้ AI เขียน แล้วเราศึกษาตาม" ให้ได้ผลจริง

การให้ AI เขียนโค้ดแล้วอ่านตามได้ผลดี **ถ้า** ทำครบ 5 ขั้นนี้ ไม่งั้นจะรู้สึกว่าเข้าใจ แต่เขียนเองไม่ได้:

1. **อ่านทฤษฎีก่อน** ค่อยขอโค้ด — จะได้รู้ว่ากำลังดูอะไร
2. **ขอโค้ดเล็ก** — ไม่เกิน ~80 บรรทัดต่อครั้ง ขอให้ใส่คอมเมนต์อธิบาย "ทำไม" ไม่ใช่ "ทำอะไร"
3. **อธิบายกลับ** — ปิดคำอธิบาย แล้วพิมพ์ด้วยคำตัวเองว่าแต่ละบล็อกทำอะไร ส่งให้ AI ตรวจ
4. **แก้ 1 อย่างเอง** — เพิ่ม field, เปลี่ยนเงื่อนไข, เพิ่มปุ่ม (โจทย์ "ลองแก้เอง" ในแผนรายวัน)
5. **ทำให้พังโดยตั้งใจ** — ลบ dependency ใน `useEffect`, ลืม `key`, ลืม `await` แล้วดูว่าเกิดอะไร จำได้แม่นที่สุด

prompt พื้นฐานที่ใช้ได้ทุกวัน:

```text
ผมเป็น backend Go กำลังเรียน <หัวข้อ> ช่วยเขียน <สิ่งที่ต้องการ> ด้วย TypeScript
- ไม่เกิน 80 บรรทัด
- คอมเมนต์อธิบายเหตุผล (ทำไม) เป็นภาษาไทย
- ถ้ามีแนวคิดที่ต่างจาก backend ให้เทียบกับ Go
- ท้ายคำตอบ ให้คำถาม 3 ข้อเพื่อทดสอบว่าผมเข้าใจจริง
```

## วิธีเรียน: เน้นอะไร และเรียนยังไงให้จำ

### เน้นอะไร (80/20)

**🔴 ต้องเข้าใจลึก อธิบายให้คนอื่นฟังได้** — ถูกถามในสัมภาษณ์บ่อยและเป็นต้นเหตุ bug ส่วนใหญ่

1. **State กับการ re-render** ([3.3](03-react-fundamentals.md)) — state เปลี่ยน → function ถูกเรียกใหม่, immutability, derived state เข้าใจข้อนี้แล้ว React ที่เหลือส่วนใหญ่จะเข้าใจตาม
2. **`useEffect` และกับดัก** ([3.6–3.7](03-react-fundamentals.md)) — dependency, cleanup, stale closure และรู้ว่าเมื่อไหร่*ไม่ควร*ใช้
3. **Server state vs client state** ([4.1](04-react-data-forms-routing.md)) และ TanStack Query
4. **Server Component vs Client Component** ([5.3](05-nextjs-app-router.md)) — เรื่องที่สับสนที่สุดของ Next.js
5. **CORS, cookie, auth** ([บทที่ 6](06-fullstack-auth-deploy.md)) — จุดที่คุณได้เปรียบ ควรตอบได้ลึกกว่าคนที่ทำ frontend อย่างเดียว

**🟡 ใช้เป็น ไม่ต้องจำ** — CSS/Tailwind, react-hook-form, zod, React Router, syntax ของ Next.js เปิดเอกสารหรือถาม AI ได้ตลอด แค่รู้ว่ามีเครื่องมืออะไรและแต่ละตัวแก้ปัญหาอะไร

**⚪ ข้ามไปก่อน** — Redux, CSS ขั้นสูง (animation, design system), Webpack config, class component, framework ตัวที่สอง เรียนเมื่องานต้องใช้

### เทคนิคการเรียน

1. **AI เป็นติวเตอร์ ไม่ใช่คนเขียนแทน** — ทุกโค้ดที่ AI เขียน ต้องตอบได้ว่า "ลบบรรทัดนี้แล้วจะเกิดอะไร" บรรทัดไหนตอบไม่ได้ ถามจนเข้าใจ (ดูขั้นตอนเต็มในหัวข้อก่อนหน้า) prompt ที่ช่วยได้มาก:

   ```text
   อย่าเพิ่งให้คำตอบ ถามผมทีละคำถามจนผมหาคำตอบเองได้
   ```

2. **ทำให้พังโดยตั้งใจ (ได้ผลที่สุด)** — ลบ `key`, ลบ dependency array, ลืม `await`, ตั้ง CORS เป็น `*` แล้วอ่าน error ด้วยตัวเอง เจอในงานจริงจะรู้ทันที โจทย์ "ทำให้พัง" ในแผนรายวันอย่าข้าม

3. **เทียบกับ Go ตลอด** — props ≈ argument, custom hook ≈ แยก service ออกจาก handler, queryKey ≈ cache key ใน Redis, FSD ≈ clean architecture เจอเรื่องใหม่ให้ถามตัวเองว่า "ใน Go คืออะไร"

4. **พิมพ์จากความจำวันละครั้ง** — ก่อนเลิก ปิดโค้ดแล้วพิมพ์ส่วนสำคัญของวันใหม่ 5–10 นาที (เช่น `useQuery` + `useMutation`, CORS middleware) การดึงความรู้ออกมาเอง (active recall) ทำให้จำได้นานกว่าอ่านซ้ำมาก

5. **NOTES.md สั้นๆ ด้วยคำของตัวเอง** — ใช้ช่อง "ยังงง" ให้เป็นประโยชน์ เช้าวันรุ่งขึ้นอ่าน NOTES ของเมื่อคืน 2 นาทีระหว่างเดินทาง = ทวนแบบเว้นระยะ (spaced repetition) ที่แทบไม่เสียเวลา

6. **อ่านโค้ดจริงแบบมีคำถามนำ** — อย่าอ่าน bop-dashboard จากบนลงล่าง ตั้งคำถามก่อน เช่น "กด login แล้วเกิดอะไรขึ้น" แล้วไล่ไฟล์: `login-form.tsx` → `use-login.ts` → `api-client.ts` → Go `handler.go` → cookie → `middleware.ts` การไล่ flow ตรงกับคำถามสัมภาษณ์ "อธิบายระบบที่เคยทำ"

7. **กฎ 20 นาที** — ติดเกิน 20 นาที ถาม AI หรือข้ามไปก่อน อย่าติดจนเลยเวลานอน เรื่องที่ติดมักคลายเองเมื่อเรียนบทถัดไป

8. **พูดออกเสียง** — ตั้งแต่สัปดาห์ 3 ทุกวันศุกร์ อธิบายหัวข้อ 🔴 สักข้อออกเสียง 2 นาทีเหมือนอยู่ในห้องสัมภาษณ์ อธิบายไม่ลื่น = ยังไม่เข้าใจจริง

### กับดักที่คนจาก backend มักพลาด

- **อยากเข้าใจ CSS ให้สมบูรณ์ก่อน** จนเสียไปหลายวัน — จัด layout ได้พอแล้ว ไปต่อ
- **เก็บทุกอย่างใน state** — ก่อนสร้าง state ใหม่ ถามว่า "คำนวณจากค่าที่มีอยู่ได้ไหม"
- **ใช้ `useEffect` เหมือน event handler** — ค่าเปลี่ยนเพราะ user กดอะไร ให้ทำใน handler ตรงๆ
- **มองข้าม loading / error / empty state** — ผู้สัมภาษณ์ดูจุดนี้ก่อนเลย

## กติกาเรื่องสุขภาพ (สำคัญกว่าความเร็ว)

- **หยุด 22:30 ไม่ว่าจะค้างตรงไหน** จดว่าค้างอะไรไว้ในไฟล์ `NOTES.md` แล้วค่อยต่อพรุ่งนี้ — สมองจะคิดต่อให้ตอนหลับ
- ทำ **4 คืน + 1 ช่วงวันหยุด** ต่อสัปดาห์ พุธพัก อาทิตย์พัก
- งดจอ 30 นาทีก่อนนอน
- คืนที่เหนื่อยมาก: ทบทวน 20 นาที (อ่านสรุปของ session ก่อน) แล้วนอน — นับว่าทำ
- ตามไม่ทันแผน **ห้ามทำเพิ่มเพื่อไล่** ให้เลื่อนแผนทั้งชุดไปแทน ทำช้าแต่ต่อเนื่องดีกว่า burn out ในสัปดาห์ที่ 3

## สารบัญ

### [แผนรายวัน 8 สัปดาห์](00-daily-plan.md)

### [บทที่ 1 — HTML & CSS Layout](01-html-css-layout.md) · สัปดาห์ 1

- [ ] Semantic HTML, form, accessibility พื้นฐาน
- [ ] Box model, Flexbox, Grid, responsive
- [ ] Tailwind CSS ในฐานะ "CSS แบบ utility"

### [บทที่ 2 — JavaScript & TypeScript สำหรับเบราว์เซอร์](02-javascript-typescript.md) · สัปดาห์ 1–2

- [ ] ความต่างของ JS จาก Go (value/reference, truthy, `this`, closure)
- [ ] Array methods, destructuring, spread, optional chaining
- [ ] Event loop, Promise, `async/await`, `fetch`
- [ ] DOM และ event
- [ ] TypeScript: type, interface, union, generic, narrowing

### [บทที่ 3 — React Fundamentals](03-react-fundamentals.md) · สัปดาห์ 3–4

- [ ] Component, JSX, props
- [ ] State, re-render, immutability
- [ ] Event, list + `key`, conditional rendering
- [ ] `useEffect` และ lifecycle, custom hook
- [ ] Context, การยก state ขึ้น (lifting state)

### [บทที่ 4 — Data Fetching, Form, Routing](04-react-data-forms-routing.md) · สัปดาห์ 5

- [ ] Server state vs client state
- [ ] TanStack Query: `useQuery`, `useMutation`, invalidation
- [ ] react-hook-form + zod
- [ ] Routing ฝั่ง client

### [บทที่ 5 — Next.js App Router](05-nextjs-app-router.md) · สัปดาห์ 6

- [ ] File-based routing, layout, dynamic route
- [ ] Server Component vs Client Component
- [ ] Rendering: CSR / SSR / SSG และ caching
- [ ] Route Handler, Server Action, middleware/proxy

### [บทที่ 6 — Fullstack: Go + Next.js, Auth, Deploy](06-fullstack-auth-deploy.md) · สัปดาห์ 7

- [ ] สถาปัตยกรรม: frontend แยกจาก backend
- [ ] CORS, cookie, JWT httpOnly
- [ ] Environment variable ฝั่ง client/server
- [ ] Docker Compose ทั้งระบบ, deploy

### [บทที่ 7 — Testing, โครงสร้างโปรเจกต์, เตรียมสัมภาษณ์](07-testing-and-interview.md) · สัปดาห์ 8

- [ ] Vitest + Testing Library
- [ ] Feature-Sliced Design (อ่านจาก bop-dashboard)
- [ ] Performance พื้นฐาน
- [ ] คำถามสัมภาษณ์ frontend/fullstack ที่เจอบ่อย พร้อมแนวตอบ

## Lab / Source Code ที่เกี่ยวข้อง

- [source-code/frontend/](../../source-code/frontend/README.md) — โค้ดฝึกตามแผนรายวัน
- [projects/bop-dashboard/](../../projects/bop-dashboard/README.md) — โค้ด Next.js จริงไว้อ่านประกอบ
- [projects/bot-orchestration-platform/](../../projects/bot-orchestration-platform/README.md) — Go backend ที่ dashboard เรียก

## แหล่งอ้างอิงหลัก

- [MDN Web Docs](https://developer.mozilla.org/) — HTML/CSS/JS อ้างอิงที่เชื่อถือได้ที่สุด
- [react.dev/learn](https://react.dev/learn) — เอกสาร React ทางการ (ดีมาก อ่านตามได้เลย)
- [nextjs.org/docs](https://nextjs.org/docs) และ [nextjs.org/learn](https://nextjs.org/learn)
- [TypeScript Handbook](https://www.typescriptlang.org/docs/handbook/intro.html)
- [TanStack Query docs](https://tanstack.com/query/latest)
- [Flexbox Froggy](https://flexboxfroggy.com/) / [Grid Garden](https://cssgridgarden.com/) — เกมฝึก layout
