# Fullstack + Infra Roadmap (Golang → React/Next.js → Infra)

roadmap ส่วนตัว ต่อยอดจากพื้นฐาน Golang ที่มีอยู่แล้ว เรียงลำดับ: **Golang (solidify) → React/Next.js (ไปสู่ fullstack) → Infra (ขยายไปสู่การออกแบบระบบสเกลใหญ่)**

หลักการของไฟล์นี้: เอาเฉพาะหัวข้อที่ **จำเป็นต่อการใช้งานจริง** ไม่ใช่ทฤษฎีครบทุกแง่มุม — ถ้าอยากได้เนื้อหาลึกแบบ textbook ของฝั่ง infra ดูที่ [books/](books/README.md) และ [ROADMAP.md](ROADMAP.md) ประกอบ ไฟล์นี้ทำหน้าที่แค่บอกว่า "เรียงลำดับยังไง" และ "หัวข้อไหนคุ้มเวลาที่สุด"

**กฎการเรียนของ roadmap นี้**: เรียนแล้วต้องมีของ ไม่ใช่แค่รู้ — แนะนำให้สร้างแอปเดียวต่อเนื่องไปเรื่อยๆ ทีละ layer (backend Go ที่มีอยู่ → เติม frontend → เติม infra) แทนที่จะเรียนแยกเป็นชิ้นๆ ไม่เชื่อมกัน

---

## Phase 0 — Golang (Solidify ของเดิม)

คุณเขียน Go เป็นหลักอยู่แล้ว เป้าหมายของ phase นี้ไม่ใช่เรียนใหม่ แต่เช็คว่าหัวข้อเหล่านี้ **แน่นจริง** ก่อนไปต่อ (อ่านลึกได้ที่ [books/04-golang/](books/04-golang/README.md)):

- [ ] Interface + implicit implementation ใช้ออกแบบ dependency injection ได้คล่อง
- [ ] Goroutine/channel/`sync` — เขียนโค้ด concurrent ที่ไม่มี race condition และรู้วิธีพิสูจน์ด้วย `-race`
- [ ] `context.Context` ส่งผ่าน call chain ถูกต้อง (cancellation/timeout)
- [ ] เขียน table-driven test ได้เป็นธรรมชาติ
- [ ] จัดโครงสร้างโปรเจกต์ตาม Clean Architecture/Repository pattern ได้โดยไม่ต้องเปิดตัวอย่าง

**เป้าหมายเมื่อจบ phase**: มี Go backend (REST API) อย่างน้อย 1 ตัวที่ผ่าน test, มี structure ที่ scale ต่อได้ — นี่จะเป็นฐานให้ frontend เรียกใช้ใน Phase 1

---

## Phase 1 — React + Next.js (ไปสู่ Fullstack)

เป้าหมาย: เขียน frontend ที่คุยกับ Go backend ของตัวเองได้จริง ไม่ใช่เก่งเท่า frontend specialist

### 1.1 พื้นฐานที่ต้องมีก่อนแตะ React

- [ ] JavaScript ES6+ ที่ใช้บ่อยใน React: arrow function, destructuring, spread/rest, template literal, `async/await`, module import/export
- [ ] **TypeScript** — แนะนำเรียนคู่ไปเลยตั้งแต่ต้น (ไม่ใช่เรียน JS ล้วนก่อนแล้วค่อยมาต่อ TS) เพราะพื้นฐาน static typing จาก Go จะทำให้เข้าใจ TS เร็วกว่าคนที่มาจาก JS ล้วน และโค้ด production จริงแทบไม่มีใครเขียน React ด้วย JS ล้วนแล้ว

### 1.2 React Core

- [ ] Component, JSX, props, การส่งข้อมูลจาก parent → child
- [ ] `useState` — state ภายใน component
- [ ] `useEffect` — side effect, dependency array, cleanup function (จุดที่มือใหม่พลาดบ่อยที่สุด)
- [ ] Conditional rendering, list rendering + `key` prop (ทำไม key สำคัญ — เชื่อมกับการทำงานของ React reconciliation)
- [ ] Form handling — controlled input, `onChange`/`onSubmit`
- [ ] `useRef`, `useContext`, การเขียน custom hook ของตัวเอง
- [ ] เข้าใจ component lifecycle/re-render คร่าวๆ (อะไรทำให้ component re-render)

### 1.3 State Management (เลือกเท่าที่จำเป็น)

- [ ] Local state (`useState`/`useReducer`) พอสำหรับ component ส่วนใหญ่ — อย่ารีบกระโดดไป global state library
- [ ] Context API สำหรับ state ที่ต้องแชร์ข้าม component ไม่กี่ level (เช่น theme, user session)
- [ ] **Zustand** (ทางเลือกที่เบากว่า Redux มาก เหมาะกับคนที่มาจาก backend เพราะ mental model แบบ store ตรงไปตรงมา) — เรียนถ้าโปรเจกต์เริ่มมี global state ซับซ้อนขึ้นจริงๆ ค่อยเรียน ไม่ต้องเรียนล่วงหน้า

### 1.4 เรียกข้อมูลจาก Backend (จุดที่ backend dev จะรู้สึกคุ้นเคยที่สุด)

- [ ] `fetch` พื้นฐาน, จัดการ loading/error state เอง
- [ ] **TanStack Query (React Query)** — จัดการ caching/loading/error/retry ให้อัตโนมัติ แนะนำอย่างยิ่งเพราะ mental model ตรงกับที่คุณคุ้นเคยจากฝั่ง server (คล้าย cache-aside pattern ที่อธิบายไว้ใน [books/06-redis/01-caching-and-ttl.md](books/06-redis/01-caching-and-ttl.md))
- [ ] เข้าใจ CORS จากฝั่ง frontend จริง (ทำไม browser บล็อก request ข้าม origin) — ทฤษฎีมีอยู่แล้วใน [books/02-network/](books/02-network/README.md) รอบนี้คือเจอปัญหาจริงจาก error message ใน browser console

### 1.5 Next.js เฉพาะทาง

- [ ] **App Router** (มาตรฐานปัจจุบัน ไม่ต้องเรียน Pages Router ก่อน) — file-based routing, `layout.tsx`, `page.tsx`
- [ ] **Server Component vs Client Component** — concept ที่สำคัญที่สุดของ Next.js สมัยใหม่ และต่างจาก React ทั่วไปชัดเจน ต้องเข้าใจว่าอะไรรันฝั่ง server อะไรรันฝั่ง browser และทำไมถึงแยกแบบนี้
- [ ] Rendering strategy: SSR (Server-Side Render), SSG (Static), ISR (Incremental Static Regeneration) — เลือกใช้ให้ถูกกับหน้าแต่ละแบบ
- [ ] API Routes / Route Handlers — เขียน backend endpoint เล็กๆ ใน Next.js เองได้ (มีประโยชน์ตอนทำ BFF — Backend For Frontend)
- [ ] Environment variable (`NEXT_PUBLIC_*` vs server-only) — เข้าใจว่าอะไรหลุดไปที่ browser ได้ อะไรต้องเก็บฝั่ง server เท่านั้น

### 1.6 Styling และ Form

- [ ] **Tailwind CSS** — utility-first ทำให้ backend dev ทำ UI ได้เร็วโดยไม่ต้องมี design sense ลึก
- [ ] **react-hook-form + zod** — จัดการ form validation แบบ type-safe (zod เขียน schema คล้าย struct validation ที่คุ้นเคยจาก Go)

### 1.7 Authentication ฝั่ง Frontend

- [ ] จัดการ JWT/session token ฝั่ง client (เก็บที่ไหนปลอดภัย — httpOnly cookie ดีกว่า localStorage สำหรับ token สำคัญ เชื่อมกับ [books/09-authentication/01-session-and-jwt.md](books/09-authentication/01-session-and-jwt.md))
- [ ] Protected route — redirect ถ้ายังไม่ login
- [ ] **NextAuth.js (Auth.js)** ถ้าต้องการ OAuth2 login (Google/GitHub) แบบไม่ต้องเขียน flow เองทั้งหมด — เชื่อมกับทฤษฎีที่มีอยู่แล้วใน [books/09-authentication/02-oauth2-and-oidc.md](books/09-authentication/02-oauth2-and-oidc.md)

### 1.8 Testing + Deploy

- [ ] Vitest/Jest + React Testing Library — test component พื้นฐาน (render, user interaction)
- [ ] Deploy ขึ้น Vercel ก่อน (ง่ายสุด เอาไว้ demo เร็ว) แล้วค่อย containerize ด้วย Docker (ต่อกับ [books/07-docker/](books/07-docker/README.md)) เพื่อ deploy บน infra ของตัวเองใน Phase 2

**เป้าหมายเมื่อจบ phase**: มี fullstack app 1 ตัว (Next.js frontend + Go backend เดิมจาก Phase 0) ที่ login ได้, เรียก API จริงได้, deploy ขึ้น production จริง (Vercel หรือ container ของตัวเอง)

---

## Phase 2 — Infra (ออกแบบระบบรองรับสเกลใหญ่)

เนื้อหาลึกมีอยู่แล้วใน [books/07](books/07-docker/README.md) ถึง [books/17](books/17-system-design/README.md) — รายการนี้คือ**ลำดับความสำคัญที่เรียงใหม่** ตามเป้าหมาย "หางานได้ + ออกแบบระบบสเกลล้านได้" ไม่ใช่เรียงตามเลขบทเดิม

### 2.1 ต้องมีก่อน (Foundation ของ Infra)

- [ ] [Docker](books/07-docker/README.md) — containerize ทั้ง frontend และ backend จาก Phase 1
- [ ] [Kubernetes](books/08-kubernetes/README.md) — deploy app จริงขึ้น cluster (Pod/Deployment/Service/Ingress ก่อน ที่เหลือค่อยตามทีหลัง)
- [ ] [CI/CD](books/16-cicd/README.md) — pipeline อัตโนมัติ build→test→deploy ของ app ตัวเดียวกัน

**เช็คพอยต์**: app จาก Phase 1 ต้อง deploy ผ่าน pipeline อัตโนมัติขึ้น K8s ได้จริง ไม่ deploy มือแล้ว

### 2.2 ทำให้ "ดูเป็นระบบ production จริง" (ตัวสร้างความต่างตอนสัมภาษณ์)

- [ ] [Observability](books/14-observability/README.md) — ใส่ metrics (Prometheus), log (Fluent Bit/OpenSearch), trace (OpenTelemetry/Jaeger) เข้าไปใน app เดิม — เชื่อม request จาก frontend → backend → DB ให้ trace เห็นทั้งเส้นทาง
- [ ] [System Design](books/17-system-design/README.md) — โฟกัสบทที่ 1 (CAP), บทที่ 2 (Sharding/Replication/Cache), บทที่ 3 (HA/DR) เพื่อ**อธิบาย trade-off เป็นคำพูดได้** ในสัมภาษณ์แบบ "ออกแบบระบบรองรับ 1 ล้าน request/วัน"

### 2.3 ขยายเป็นสถาปัตยกรรมสเกลใหญ่ (ตอบโจทย์ "รองรับหลักล้าน request")

- [ ] [Authentication](books/09-authentication/README.md) — ทำ auth ให้ถูกต้องระดับ production (JWT/OAuth2/RBAC) ไม่ใช่แค่ login demo
- [ ] [APISIX](books/10-apisix/README.md) — วาง API Gateway หน้า backend, rate limit, รวม cross-cutting concern ไว้จุดเดียว
- [ ] [PostgreSQL](books/05-postgresql/README.md) เจาะบทที่ 3 (Transaction/MVCC) และบทที่ 5 (Partition/Replication) — จุดที่ระบบสเกลใหญ่มักพังเพราะ database ก่อนเสมอ
- [ ] [Redis](books/06-redis/README.md) — cache layer จริงจัง ไม่ใช่แค่ session store
- [ ] [Microservices](books/11-microservices/README.md) — เมื่อไหร่ควรแตก service (และเมื่อไหร่ไม่ควร) Retry/Circuit Breaker/Idempotency
- [ ] [Kafka](books/12-kafka/README.md) — event-driven สำหรับงานที่ decouple กันได้ (เช่น notification, analytics)

### 2.4 ตัวสร้างความแตกต่าง (Advanced Differentiator — ทำเมื่อมีเวลา)

- [ ] [Temporal](books/13-temporal/README.md) — orchestration ของ business process ซับซ้อน (เช่น payment flow) น้อยคนในตลาดที่มีของจริงเรื่องนี้ ถ้ามีคือจุดขายชัดเจน
- [ ] [Security](books/15-security/README.md) — OWASP Top 10 อย่างน้อยบทที่ 1 ต้องตอบได้ทุกข้อ

**เป้าหมายเมื่อจบ phase**: [projects/ecommerce/](projects/ecommerce/README.md) หรือระบบเทียบเท่า ที่มี frontend (Next.js) + backend (Go, อาจแตกเป็น 2-3 service) + gateway + queue + observability ครบ deploy บน K8s ผ่าน CI/CD — พร้อมอธิบาย trade-off ทุกจุดในสัมภาษณ์ได้

---

## สรุปลำดับภาพรวม

```
Phase 0: Golang (solidify)          ──► มี Go API พร้อม test
Phase 1: React + Next.js            ──► app เดิมมี frontend, fullstack demo ได้
Phase 2.1: Docker + K8s + CI/CD     ──► app เดิม deploy อัตโนมัติจริง
Phase 2.2: Observability + Sys Design ──► อธิบาย/โชว์ระบบระดับ production ได้
Phase 2.3: Auth + Gateway + DB scale + Microservices + Kafka ──► ระบบรองรับสเกลใหญ่จริง
Phase 2.4: Temporal + Security      ──► จุดขายที่ต่างจากคนอื่น
```

อย่าข้าม phase หรือเรียนขนานกันหลาย phase พร้อมกัน — ทำทีละ phase ให้ app ตัวเดียวโตขึ้นเรื่อยๆ ตอนจบจะได้ 1 ระบบที่เล่าเรื่องได้ครบ ดีกว่าความรู้กระจัดกระจายหลายเรื่องที่ไม่เชื่อมกัน
