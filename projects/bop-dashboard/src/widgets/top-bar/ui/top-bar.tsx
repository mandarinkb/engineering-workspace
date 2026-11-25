"use client";

import { useRouter } from "next/navigation";

import type { Readiness } from "@/shared/api";
import { useTheme } from "@/shared/lib/theme";
import { Icon } from "@/shared/ui";
import { useLogout } from "@/features/auth";
import { useAuth } from "@/entities/user";
import { useReadiness } from "../api/use-readiness";

const READINESS_META: Record<Readiness, { label: string; dot: string }> = {
  ready: { label: "ระบบปกติ", dot: "bg-green-500" },
  not_ready: { label: "ระบบไม่พร้อม (Temporal/ฐานข้อมูล)", dot: "bg-red-500" },
  api_down: { label: "เรียก API ไม่ติด", dot: "bg-red-500" },
};

const ICON_BUTTON =
  "flex h-9 w-9 items-center justify-center rounded-md text-gray-900 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-white/10";

// TopBar คือแถบบนสุดแบบ Temporal UI: ซ้ายเป็นกล่องสถานะระบบ (ตำแหน่งเดียวกับกล่อง namespace
// ของ Temporal) ขวาเป็นปุ่มสลับธีม ชื่อ user และออกจากระบบ — ย้ายมาจากท้าย sidebar
// ข้อมูลสถานะยังมาจาก GET /readyz เหมือนเดิม (role="status" ให้ screen reader อ่านเมื่อเปลี่ยน)
export function TopBar() {
  const router = useRouter();
  const { user } = useAuth();
  const logout = useLogout();
  const { theme, toggleTheme } = useTheme();
  const { data: readiness } = useReadiness();
  // ยังไม่ได้ผลรอบแรก = สีเทา ไม่เดาว่าปกติหรือล่ม
  const status = readiness ? READINESS_META[readiness] : { label: "กำลังเช็คระบบ...", dot: "bg-gray-500" };

  return (
    <header className="flex h-14 items-center justify-between border-b border-gray-200 px-10 dark:border-gray-800">
      <div
        role="status"
        className="flex items-center gap-3 rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm dark:border-gray-600 dark:bg-transparent"
      >
        <span className={`h-2.5 w-2.5 rounded-full ${status.dot}`} />
        {status.label}
      </div>

      <div className="flex items-center gap-1">
        <button
          type="button"
          onClick={toggleTheme}
          aria-label="สลับธีมสว่าง/มืด"
          title={theme === "dark" ? "ธีมสว่าง" : "ธีมมืด"}
          className={ICON_BUTTON}
        >
          {theme === "dark" ? "☀️" : "🌙"}
        </button>
        {user && (
          <div className="mx-2 flex items-center gap-2 text-sm">
            <span className="flex h-7 w-7 items-center justify-center rounded-full bg-blue-600 text-xs font-semibold uppercase text-white">
              {user.username.slice(0, 1)}
            </span>
            <span>{user.username}</span>
          </div>
        )}
        <button
          type="button"
          onClick={() => logout.mutate(undefined, { onSuccess: () => router.push("/login") })}
          aria-label="ออกจากระบบ"
          title="ออกจากระบบ"
          className={ICON_BUTTON}
        >
          <Icon d="M15 4h3a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2h-3M10 17l5-5-5-5M15 12H3" />
        </button>
      </div>
    </header>
  );
}
