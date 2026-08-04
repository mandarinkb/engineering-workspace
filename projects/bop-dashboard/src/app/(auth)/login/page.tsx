import { LoginForm } from "@/features/auth/components/login-form";

// (auth) คือ route group (วงเล็บใน path ไม่กลายเป็นส่วนหนึ่งของ URL จริง — หน้านี้ยังอยู่
// ที่ /login ตรงๆ) แยกออกจาก (dashboard) เพื่อให้มี layout คนละแบบกัน: หน้านี้ไม่มี
// NavBar เลย (ดู app/(dashboard)/layout.tsx ที่มี NavBar) เพราะยังไม่ login จะโชว์เมนู
// ของ dashboard ไปก็ไม่มีประโยชน์
export default function LoginPage() {
  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="flex flex-col gap-6">
        <h1 className="text-center text-2xl font-semibold">BOP Dashboard</h1>
        <LoginForm />
      </div>
    </div>
  );
}
