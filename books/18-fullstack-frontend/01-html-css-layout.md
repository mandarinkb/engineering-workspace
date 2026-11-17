# บทที่ 1: HTML & CSS Layout

backend engineer ส่วนใหญ่ "อ่าน HTML ออก" แต่จัด layout ไม่ได้ ซึ่งเป็นเรื่องปกติ CSS ไม่ได้ยาก แต่**คิดคนละแบบ**กับโค้ดทั่วไป: เราไม่ได้สั่งให้คอมพิวเตอร์ทำทีละขั้น แต่ประกาศ "กฎ" ให้ browser คำนวณตำแหน่งเอง บทนี้สอนเฉพาะส่วนที่ใช้จริง 90% ของงาน: semantic HTML, box model, Flexbox, Grid, responsive และ Tailwind

## 1.1 Semantic HTML

HTML คือ **โครงสร้างและความหมาย** ไม่ใช่หน้าตา หน้าตาเป็นหน้าที่ของ CSS การเลือก tag ให้ถูกความหมายมีผลจริง 3 อย่าง: screen reader อ่านได้ถูก, SEO, และโค้ดอ่านง่ายเหมือนตั้งชื่อตัวแปรดี

```html
<!doctype html>
<html lang="th">
<head>
  <meta charset="utf-8" />
  <!-- บรรทัดนี้สำคัญมาก: ไม่ใส่ มือถือจะ render หน้าแบบ desktop แล้วย่อลง -->
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Thongchai — Backend Engineer</title>
  <link rel="stylesheet" href="style.css" />
</head>
<body>
  <header>
    <nav>
      <a href="/">Home</a>
      <a href="#projects">Projects</a>
    </nav>
  </header>

  <main>
    <section id="experience">
      <h2>Experience</h2>
      <article>
        <h3>Backend Engineer</h3>
        <p>Go, PostgreSQL, Kafka</p>
      </article>
    </section>
  </main>

  <footer>© 2026</footer>
</body>
</html>
```

| Tag | ใช้เมื่อ |
|---|---|
| `header` / `footer` | ส่วนหัว/ท้ายของหน้า หรือของ `article` |
| `nav` | กลุ่มลิงก์นำทางหลัก |
| `main` | เนื้อหาหลัก มีได้ **1 อันต่อหน้า** |
| `section` | กลุ่มเนื้อหาที่มีหัวข้อของตัวเอง |
| `article` | เนื้อหาที่ยกไปใช้ที่อื่นได้ทั้งก้อน (โพสต์, card สินค้า) |
| `div` / `span` | ไม่มีความหมาย ใช้เพื่อจัด layout อย่างเดียว (`div` = block, `span` = inline) |
| `button` vs `a` | `button` = **ทำ action** (submit, ลบ), `a` = **ไปที่อื่น** (เปลี่ยน URL) — อย่าเอา `div` มาทำปุ่ม |

**Block vs inline:** element แบบ block (`div`, `p`, `section`, `h1`) กินเต็มความกว้างและขึ้นบรรทัดใหม่ แบบ inline (`span`, `a`, `strong`) ไหลต่อกันในบรรทัดเหมือนข้อความ

## 1.2 Form และ Accessibility พื้นฐาน

ฟอร์มคือจุดที่ frontend คุยกับ backend ของคุณ

```html
<form action="/contact" method="post">
  <!-- for ต้องตรงกับ id: คลิกข้อความแล้ว focus ช่อง และ screen reader รู้ว่าช่องนี้คืออะไร -->
  <label for="email">Email</label>
  <input id="email" name="email" type="email" required />

  <label for="msg">Message</label>
  <textarea id="msg" name="message" minlength="10"></textarea>

  <label for="topic">Topic</label>
  <select id="topic" name="topic">
    <option value="job">Job offer</option>
    <option value="other">Other</option>
  </select>

  <button type="submit">Send</button>
</form>
```

- `name` คือ key ที่ถูกส่งไป server (เหมือน field ใน form-urlencoded ที่ Go อ่านด้วย `r.FormValue("email")`)
- `type="email"`, `required`, `minlength` คือ validation ของ browser — **ช่วย UX แต่ไม่ใช่ความปลอดภัย** ใครก็ยิง request ตรงได้ backend ต้อง validate เสมอ
- ใน React เราจะ**ไม่**ปล่อยให้ form submit แบบดั้งเดิม (โหลดหน้าใหม่) แต่จะ `preventDefault()` แล้วยิง `fetch` เอง (บทที่ 3–4)

Accessibility ขั้นต่ำที่ควรทำเป็นนิสัย: ทุก `input` มี `label`, ทุก `img` มี `alt`, ใช้ `button` สำหรับปุ่ม, กด Tab ไล่ได้ทั้งหน้า

## 1.3 Box Model

ทุก element คือกล่องสี่เหลี่ยม 4 ชั้น จากในออกนอก:

```text
┌──────────── margin (ระยะห่างจากกล่องอื่น โปร่งใส) ────────────┐
│ ┌────────── border (เส้นขอบ) ──────────────────────────────┐ │
│ │ ┌──────── padding (ระยะในกล่อง มีสีพื้นหลัง) ─────────┐ │ │
│ │ │              content (width × height)               │ │ │
│ │ └─────────────────────────────────────────────────────┘ │ │
│ └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

ค่า default ของ browser คือ `width` นับแค่ content — ใส่ `width: 300px; padding: 20px` ได้กล่องกว้างจริง 340px ซึ่งทำให้คำนวณยาก ทุกโปรเจกต์จึงเริ่มด้วย:

```css
*, *::before, *::after {
  box-sizing: border-box; /* width รวม padding + border แล้ว: 300px คือ 300px จริง */
}
body {
  margin: 0;
  font-family: system-ui, sans-serif;
  line-height: 1.5;
}
```

**หน่วยที่ใช้บ่อย:** `px` (ตายตัว), `rem` (เท่ากับขนาด font ของ root ปกติ 16px — ใช้กับ font/spacing เพื่อให้ขยายตามการตั้งค่าของ user), `%` (เทียบกับ parent), `vh`/`vw` (เทียบกับขนาดหน้าจอ)

## 1.4 Flexbox — จัดของเรียงใน **1 มิติ**

Flexbox ใช้จัด element ลูก**เรียงในแนวเดียว** (แถว หรือ คอลัมน์) ใส่ `display: flex` ที่ **parent** แล้วลูกทุกตัวจะเรียงตามกฎ

```css
.navbar {
  display: flex;
  justify-content: space-between; /* แกนหลัก (แนวนอน): โลโก้ชิดซ้าย เมนูชิดขวา */
  align-items: center;            /* แกนรอง (แนวตั้ง): จัดกึ่งกลางความสูง */
  gap: 1rem;                      /* ระยะห่างระหว่างลูก ใช้แทน margin */
  padding: 1rem 2rem;
}

.skills {
  display: flex;
  flex-wrap: wrap;   /* เต็มแถวแล้วขึ้นบรรทัดใหม่ */
  gap: 0.5rem;
}

.sidebar-layout {
  display: flex;
}
.sidebar-layout aside { width: 240px; flex-shrink: 0; } /* กว้างคงที่ ห้ามหด */
.sidebar-layout main  { flex: 1; }                      /* กินพื้นที่ที่เหลือทั้งหมด */
```

แนวคิดหลักที่ต้องจำ:

- **แกนหลัก (main axis)** = ทิศทางที่เรียง (`flex-direction: row` คือแนวนอน, `column` คือแนวตั้ง) ควบคุมด้วย `justify-content`
- **แกนรอง (cross axis)** = ตั้งฉากกับแกนหลัก ควบคุมด้วย `align-items`
- เปลี่ยนเป็น `flex-direction: column` แล้ว `justify-content` จะกลายเป็นแนวตั้ง — นี่คือจุดที่คนสับสนบ่อยที่สุด

สูตรจัดกึ่งกลางทุกอย่าง: `display: flex; justify-content: center; align-items: center;`

## 1.5 Grid — จัด layout **2 มิติ**

Grid ใช้เมื่อต้องควบคุมทั้งแถวและคอลัมน์พร้อมกัน เช่น card gallery หรือโครงหน้าทั้งหน้า

```css
/* card grid ที่ responsive เอง โดยไม่ต้องเขียน media query */
.projects {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 1.5rem;
}
/* อ่านว่า: สร้างคอลัมน์ให้มากที่สุดเท่าที่ใส่ได้ แต่ละคอลัมน์กว้างอย่างน้อย 260px
   และยืดเท่าๆ กัน (1fr = 1 ส่วนของพื้นที่ที่เหลือ) */

/* โครงหน้า dashboard */
.app {
  display: grid;
  grid-template-columns: 240px 1fr;
  grid-template-rows: 64px 1fr;
  grid-template-areas:
    "sidebar header"
    "sidebar main";
  min-height: 100vh;
}
.app > aside  { grid-area: sidebar; }
.app > header { grid-area: header; }
.app > main   { grid-area: main; }
```

**Flex หรือ Grid?** ใช้กฎง่ายๆ: เนื้อหากำหนดขนาด (เมนู, ปุ่มเรียงกัน, chip) → **Flex**; layout กำหนดขนาด (gallery, โครงหน้า, ฟอร์ม 2 คอลัมน์) → **Grid** ในงานจริงใช้ซ้อนกัน: Grid เป็นโครงหน้า ข้างในแต่ละช่องใช้ Flex

## 1.6 Responsive Design

เขียนแบบ **mobile-first**: CSS ปกติสำหรับมือถือ แล้วใช้ `min-width` media query เพิ่มเมื่อจอกว้างขึ้น

```css
.projects {
  display: grid;
  grid-template-columns: 1fr;          /* มือถือ: 1 คอลัมน์ */
  gap: 1rem;
}

@media (min-width: 640px) {            /* tablet ขึ้นไป */
  .projects { grid-template-columns: repeat(2, 1fr); }
}

@media (min-width: 1024px) {           /* desktop ขึ้นไป */
  .projects { grid-template-columns: repeat(3, 1fr); }
}

img { max-width: 100%; height: auto; } /* รูปไม่ล้นจอ */
```

ทำไม mobile-first: layout มือถือเรียบง่ายกว่า (ส่วนใหญ่เรียงลงเป็นแนวตั้ง) แล้ว "เพิ่ม" ความซับซ้อนตามจอที่ใหญ่ขึ้น ง่ายกว่าเริ่มจาก desktop แล้วคอยยกเลิก และ Tailwind ก็ทำงานแบบนี้

**Debug layout:** DevTools → Elements → คลิกป้าย `flex`/`grid` ข้าง element จะเห็นเส้นแบ่งของ layout ซ้อนบนหน้าจอ ใช้บ่อยมาก

## 1.7 Tailwind CSS

Tailwind คือ CSS ที่แตกเป็น class เล็กๆ แต่ละ class ทำอย่างเดียว แล้วเราเอามาประกอบกันใน HTML โดยตรง

```tsx
// แทนที่จะเขียน .card { padding: 1.5rem; border-radius: .75rem; ... } ในไฟล์ CSS
<div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900">
  <h3 className="text-lg font-semibold">Task title</h3>
  <p className="mt-1 text-sm text-gray-500">created 2 hours ago</p>
</div>

// responsive: prefix คือ breakpoint แบบ min-width (mobile-first เหมือนหัวข้อ 1.6)
<div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">...</div>

// state: hover, focus, disabled
<button className="rounded-lg bg-blue-600 px-4 py-2 text-white hover:bg-blue-700 disabled:opacity-50">
  Save
</button>
```

| เขียน CSS | Tailwind |
|---|---|
| `display: flex; justify-content: space-between; align-items: center` | `flex justify-between items-center` |
| `padding: 1rem` | `p-4` (1 หน่วย = 0.25rem) |
| `margin-top: 0.5rem` | `mt-2` |
| `gap: 1.5rem` | `gap-6` |
| `@media (min-width: 768px) { ... }` | `md:...` |

**ทำไมงานจริงนิยม:** ไม่ต้องตั้งชื่อ class, ไม่มี CSS ที่ไม่ได้ใช้ค้างอยู่, ค่าระยะ/สีเป็นระบบเดียวกันทั้งโปรเจกต์, แก้ component แล้วไม่ไปกระทบที่อื่น **ข้อเสีย:** class ยาว — แก้โดยแยกเป็น component (React) ไม่ใช่ย้ายกลับไปเขียน CSS

ควรเข้าใจหัวข้อ 1.3–1.6 ก่อนใช้ Tailwind เพราะ Tailwind เป็นแค่ชื่อย่อของ CSS เดิม ถ้าไม่รู้ว่า `items-center` ทำอะไรใน Flexbox ก็จะใช้ Tailwind ได้แค่ลอกตาม

> **ดูของจริง:** `projects/bop-dashboard/src/shared/ui/` — component กลางที่ห่อ class Tailwind ไว้ (เช่น `status-badge.tsx` ที่เลือกสีจาก prop `tone`)

## สรุปบท

- HTML = ความหมาย, CSS = หน้าตา เลือก tag ให้ตรงความหมาย
- เริ่มทุกโปรเจกต์ด้วย `box-sizing: border-box`
- 1 มิติ → Flexbox, 2 มิติ → Grid
- เขียนแบบ mobile-first ด้วย `min-width`
- Tailwind = CSS แบบย่อ เข้าใจ CSS ก่อนแล้วจะใช้ได้เร็วมาก
