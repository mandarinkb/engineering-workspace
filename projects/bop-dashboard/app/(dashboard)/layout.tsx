import { NavBar } from "@/components/layout/nav-bar";

// layout.tsx ของ route group (dashboard) ครอบทุกหน้าใต้ bots/, jobs/, workflow-runs/
// ด้วย NavBar เดียวกัน (roadmap ข้อ 1.6) — Next.js render layout.tsx ล้อมรอบ page.tsx
// ของทุก route ย่อยโดยอัตโนมัติ ไม่ต้อง import NavBar เข้าไปซ้ำในแต่ละ page.tsx เอง
export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen">
      <NavBar />
      <main className="p-6">{children}</main>
    </div>
  );
}
