"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useState } from "react";

import { TEMPORAL_UI_URL } from "@/shared/config";
import { useTheme } from "@/shared/lib/theme";
import { useLogout } from "@/features/auth";
import { useAuth } from "@/entities/user";

import { sidebarCookie } from "../model/sidebar-cookie";

// ไอคอนเป็น inline SVG (stroke ตาม currentColor) แทนการเพิ่ม icon library ทั้งก้อนเข้ามา
// เพื่อใช้แค่ไม่กี่ตัว — สีจะเปลี่ยนตาม text color ของลิงก์ (active/hover) เอง
function Icon({ d }: { d: string }) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.8}
      strokeLinecap="round"
      strokeLinejoin="round"
      className="h-5 w-5 shrink-0"
      aria-hidden="true"
    >
      <path d={d} />
    </svg>
  );
}

const NAV_LINKS = [
  // grid 4 ช่อง: หน้าภาพรวม
  { href: "/overview", label: "Overview", icon: "M4 4h6v6H4zM14 4h6v6h-6zM4 14h6v6H4zM14 14h6v6h-6z" },
  // หุ่นยนต์: หัว + เสาอากาศ + ตา
  { href: "/bots", label: "Bots", icon: "M12 4v3M7 7h10a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V9a2 2 0 0 1 2-2zM9.5 12h.01M14.5 12h.01M9 16h6" },
  // รายการ: บรรทัดพร้อม bullet
  { href: "/jobs", label: "Jobs", icon: "M9 6h11M9 12h11M9 18h11M4.5 6h.01M4.5 12h.01M4.5 18h.01" },
  // นาฬิกา
  { href: "/schedules", label: "Schedules", icon: "M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18zM12 7v5l3 2" },
  // workflow: โหนดเชื่อมกัน
  { href: "/workflow-runs", label: "Workflow Runs", icon: "M6 3a3 3 0 1 0 0 6 3 3 0 0 0 0-6zM18 15a3 3 0 1 0 0 6 3 3 0 0 0 0-6zM9 6h6a3 3 0 0 1 3 3v6M6 9v9a3 3 0 0 0 3 3h6" },
];

const ITEM = "flex items-center gap-3 rounded-md px-3 py-2 overflow-hidden whitespace-nowrap";
const ITEM_IDLE = "hover:bg-gray-800/60 hover:text-white";

// NavBar เป็น widget (ชั้น widgets ของ FSD) เพราะใช้ร่วมทุกหน้าใน (dashboard) route group
// (ดู app/(dashboard)/layout.tsx) และประกอบจากหลายชั้นล่าง: user ที่ login อยู่จาก
// entities/user, การ logout จาก features/auth และ theme จาก shared/lib/theme
//
// เรนเดอร์เป็น sidebar แนวตั้งด้านซ้าย (แบบ Temporal UI) พับ/กางได้: กาง = ไอคอน + ชื่อ,
// พับ = ไอคอนอย่างเดียว (ชี้เมาส์แล้วเห็นชื่อจาก title) — initialCollapsed มาจาก cookie ที่
// layout อ่านให้ฝั่ง server
export function NavBar({ initialCollapsed = false }: { initialCollapsed?: boolean }) {
  // usePathname() อาจคืน null ได้ตอน build เพราะมีโฟลเดอร์ pages/ ที่ root (ดู pages/README.md)
  const pathname = usePathname();
  const router = useRouter();
  const { user } = useAuth();
  const logout = useLogout();
  const { theme, toggleTheme } = useTheme();
  const [collapsed, setCollapsed] = useState(initialCollapsed);

  function toggleCollapsed() {
    const next = !collapsed;
    setCollapsed(next);
    document.cookie = sidebarCookie(next);
  }

  // label ซ่อนตอนพับ (sr-only ยังให้ screen reader อ่านชื่อได้)
  const label = (text: string) => <span className={collapsed ? "sr-only" : ""}>{text}</span>;

  return (
    <aside
      className={`sticky top-0 flex h-screen shrink-0 flex-col bg-gray-900 text-sm text-gray-300 transition-[width] duration-200 dark:border-r dark:border-gray-800 dark:bg-gray-950 ${
        collapsed ? "w-16" : "w-56"
      }`}
    >
      <div className={`flex items-center py-4 ${collapsed ? "justify-center px-2" : "justify-between pl-5 pr-3"}`}>
        {!collapsed && <span className="text-lg font-semibold tracking-wide text-white">BOP</span>}
        <button
          type="button"
          onClick={toggleCollapsed}
          aria-label={collapsed ? "กางเมนู" : "พับเมนู"}
          aria-expanded={!collapsed}
          title={collapsed ? "กางเมนู" : "พับเมนู"}
          className="rounded-md p-2 hover:bg-gray-800/60 hover:text-white"
        >
          <Icon d={collapsed ? "M4 6h16M4 12h16M4 18h16" : "M15 18l-6-6 6-6"} />
        </button>
      </div>

      <nav className="flex-1 space-y-1 px-3">
        {NAV_LINKS.map((link) => {
          const active = pathname?.startsWith(link.href);
          return (
            <Link
              key={link.href}
              href={link.href}
              aria-current={active ? "page" : undefined}
              title={collapsed ? link.label : undefined}
              className={`${ITEM} ${active ? "bg-gray-800 font-medium text-white" : ITEM_IDLE}`}
            >
              <Icon d={link.icon} />
              {label(link.label)}
            </Link>
          );
        })}

        {/* ลิงก์ออกนอกแอป (เปิดแท็บใหม่) — แยกด้วยเส้นคั่นให้รู้ว่าไม่ใช่หน้าของ dashboard เอง */}
        <div className="my-2 border-t border-gray-800" />
        <a
          href={TEMPORAL_UI_URL}
          target="_blank"
          rel="noopener noreferrer"
          title={collapsed ? "Temporal UI (เปิดแท็บใหม่)" : undefined}
          className={`${ITEM} ${ITEM_IDLE}`}
        >
          <Icon d="M14 4h6v6M20 4l-9 9M18 14v4a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4" />
          {label("Temporal UI")}
        </a>
      </nav>

      <div className="space-y-2 border-t border-gray-800 px-3 py-4">
        {user && (
          <div className={`flex items-center gap-3 py-1 ${collapsed ? "justify-center" : "px-3"}`} title={collapsed ? user.username : undefined}>
            <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-gray-700 text-xs font-semibold uppercase text-white">
              {user.username.slice(0, 1)}
            </span>
            {!collapsed && <span className="truncate">{user.username}</span>}
          </div>
        )}
        <button
          type="button"
          onClick={toggleTheme}
          aria-label="สลับธีมสว่าง/มืด"
          title={collapsed ? "สลับธีมสว่าง/มืด" : undefined}
          className={`${ITEM} ${ITEM_IDLE} w-full text-left`}
        >
          <span className="w-5 shrink-0 text-center">{theme === "dark" ? "☀️" : "🌙"}</span>
          {label(theme === "dark" ? "ธีมสว่าง" : "ธีมมืด")}
        </button>
        <button
          type="button"
          onClick={() => logout.mutate(undefined, { onSuccess: () => router.push("/login") })}
          title={collapsed ? "ออกจากระบบ" : undefined}
          className={`${ITEM} ${ITEM_IDLE} w-full text-left`}
        >
          <Icon d="M15 4h3a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2h-3M10 17l5-5-5-5M15 12H3" />
          {label("ออกจากระบบ")}
        </button>
      </div>
    </aside>
  );
}
