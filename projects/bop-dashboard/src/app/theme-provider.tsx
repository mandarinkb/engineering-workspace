"use client";

import { createContext, useContext, useEffect, useState } from "react";

type Theme = "light" | "dark";

const STORAGE_KEY = "bop-theme";

interface ThemeContextValue {
  theme: Theme;
  toggleTheme: () => void;
}

const ThemeContext = createContext<ThemeContextValue | null>(null);

// applyThemeClass ใส่/เอา class "dark" ออกจาก <html> ตรงๆ — Tailwind v4 ใช้
// @custom-variant dark (&:where(.dark, .dark *)) ที่ประกาศไว้ใน globals.css ทำให้ทุก
// utility class แบบ dark:xxx ทำงานตาม class นี้เท่านั้น ไม่ใช่ prefers-color-scheme
// โดยตรงอีกต่อไป (แต่ยัง fallback ไปใช้ prefers-color-scheme ตอนยังไม่เคยเลือกเองเลย
// ดู getInitialTheme ด้านล่าง)
function applyThemeClass(theme: Theme) {
  document.documentElement.classList.toggle("dark", theme === "dark");
}

// getInitialTheme อ่านค่าที่ user เคยเลือกไว้จาก localStorage ก่อนเสมอ — ถ้าไม่เคยเลือก
// เลย (ครั้งแรกที่เปิดเว็บ) ค่อย fallback ไปดู prefers-color-scheme ของเบราว์เซอร์/OS
// แทน ลำดับความสำคัญนี้ตรงกับพฤติกรรมที่ user คาดหวัง: "ครั้งแรกใช้ theme ของเครื่อง
// ฉัน แต่ถ้าฉันเคยกด toggle เองแล้ว จำค่านั้นไว้เสมอไม่ว่า OS จะตั้งค่าเป็นอะไร"
function getInitialTheme(): Theme {
  const stored = window.localStorage.getItem(STORAGE_KEY);
  if (stored === "light" || stored === "dark") {
    return stored;
  }
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

// ThemeProvider mount ที่ src/app/providers.tsx ครอบทั้งแอป — ตั้งใจ hand-roll เอง
// (ไม่ใช้ next-themes) ตาม ethos เดียวกับ ToastProvider (src/components/ui/toast-provider.tsx)
//
// หมายเหตุเรื่อง hydration: ค่าเริ่มต้นของ useState เป็น "light" เสมอ (ไม่เรียก
// getInitialTheme() ตรงๆ ตอน initial state) เพราะ localStorage/matchMedia อ่านได้แค่
// ฝั่ง client เท่านั้น ถ้าเรียกตอน render ครั้งแรกจะได้ค่าไม่ตรงกันระหว่าง server-rendered
// HTML กับ client — ใช้ useEffect อ่านค่าจริงหลัง mount แทน (ไม่มี flash ของ theme ผิด
// เพราะมี inline <script> ใน src/app/layout.tsx ใส่ class "dark" ให้ <html> ไปก่อนแล้ว
// ตั้งแต่ก่อน React จะ hydrate ด้วยซ้ำ — ไฟล์นี้แค่ sync React state ให้ตรงกับ class ที่
// มีอยู่แล้วจริงบน DOM)
export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [theme, setTheme] = useState<Theme>("light");

  useEffect(() => {
    setTheme(getInitialTheme());
  }, []);

  useEffect(() => {
    applyThemeClass(theme);
    window.localStorage.setItem(STORAGE_KEY, theme);
  }, [theme]);

  function toggleTheme() {
    setTheme((prev) => (prev === "light" ? "dark" : "light"));
  }

  return <ThemeContext.Provider value={{ theme, toggleTheme }}>{children}</ThemeContext.Provider>;
}

export function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext);
  if (!ctx) {
    throw new Error("useTheme ต้องเรียกภายใต้ <ThemeProvider> เท่านั้น");
  }
  return ctx;
}
