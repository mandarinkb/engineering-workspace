import { cookies } from "next/headers";

import { TopBar } from "@/widgets/top-bar";
import { NavBar, SIDEBAR_COOKIE, isSidebarCollapsed } from "@/widgets/nav-bar";

// layout.tsx ของ route group (dashboard) ครอบทุกหน้าใต้ bots/, jobs/, workflow-runs/
// ด้วย NavBar เดียวกัน (roadmap ข้อ 1.6) — Next.js render layout.tsx ล้อมรอบ page.tsx
// ของทุก route ย่อยโดยอัตโนมัติ ไม่ต้อง import NavBar เข้าไปซ้ำในแต่ละ page.tsx เอง
//
// NavBar เป็น sidebar ซ้ายมือ (sticky สูงเต็มจอ) ส่วน main กินพื้นที่ที่เหลือ — min-w-0
// กันตารางกว้างๆ ดัน flex item ให้ล้นจอแทนที่จะ scroll ภายใน
//
// layout เป็น Server Component อ่าน cookie สถานะพับ/กางของ sidebar ตอน render ฝั่ง server
// แล้วส่งให้ NavBar เป็นค่าเริ่มต้น HTML แรกจึงพับ/กางถูกเลย ไม่กระพริบ (cookies() ทำให้
// route เหล่านี้ render แบบ dynamic ทุก request ซึ่งไม่เสียอะไรเพราะทุกหน้าต้อง login อยู่แล้ว)
export default async function DashboardLayout({ children }: { children: React.ReactNode }) {
  const cookieStore = await cookies();
  const collapsed = isSidebarCollapsed(cookieStore.get(SIDEBAR_COOKIE)?.value);

  return (
    <div className="flex min-h-screen">
      <NavBar initialCollapsed={collapsed} />
      <div className="flex min-w-0 flex-1 flex-col">
        <TopBar />
        <main className="flex-1 px-10 py-8">{children}</main>
      </div>
    </div>
  );
}
