"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";

import { useTheme } from "@/shared/lib/theme";
import { useLogout } from "@/features/auth";
import { useAuth } from "@/entities/user";

const NAV_LINKS = [
  { href: "/bots", label: "Bots" },
  { href: "/jobs", label: "Jobs" },
  { href: "/schedules", label: "Schedules" },
  { href: "/workflow-runs", label: "Workflow Runs" },
];

// NavBar เป็น widget (ชั้น widgets ของ FSD) เพราะใช้ร่วมทุกหน้าใน (dashboard) route group
// (ดู app/(dashboard)/layout.tsx) และประกอบจากหลายชั้นล่าง: user ที่ login อยู่จาก
// entities/user, การ logout จาก features/auth และ theme จาก shared/lib/theme — widget
// import ชั้นล่างเหล่านี้ได้ตามกฎปกติ ไม่ต้องเป็นข้อยกเว้นเหมือนตอนอยู่ที่ src/components
export function NavBar() {
  const pathname = usePathname();
  const router = useRouter();
  const { user } = useAuth();
  const logout = useLogout();
  // useTheme มาจาก src/shared/lib/theme/theme-provider.tsx (ไม่ใช่ features/) — ข้อยกเว้นเดียวกับ
  // auth ด้านบน: theme เป็น cross-cutting concern ที่ nav ต้องรู้ (ปุ่ม toggle ต้องอยู่
  // ตรงนี้) ไม่ใช่ concept เฉพาะของ feature ไหนเลย
  const { theme, toggleTheme } = useTheme();

  return (
    <nav className="flex items-center justify-between border-b border-gray-200 px-6 py-4 dark:border-gray-700">
      <div className="flex items-center gap-6">
        <span className="font-semibold">BOP</span>
        {NAV_LINKS.map((link) => (
          <Link
            key={link.href}
            href={link.href}
            className={
              pathname.startsWith(link.href)
                ? "font-medium text-blue-600 dark:text-blue-400"
                : "text-gray-600 hover:text-gray-900 dark:text-gray-300 dark:hover:text-white"
            }
          >
            {link.label}
          </Link>
        ))}
      </div>
      <div className="flex items-center gap-4 text-sm text-gray-600 dark:text-gray-300">
        {user && <span>{user.username}</span>}
        <button
          type="button"
          onClick={toggleTheme}
          aria-label="สลับธีมสว่าง/มืด"
          className="rounded border border-gray-300 px-2 py-1 hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-800"
        >
          {theme === "dark" ? "☀️" : "🌙"}
        </button>
        <button
          type="button"
          onClick={() => logout.mutate(undefined, { onSuccess: () => router.push("/login") })}
          className="rounded border border-gray-300 px-3 py-1 hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-800"
        >
          ออกจากระบบ
        </button>
      </div>
    </nav>
  );
}
