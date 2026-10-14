import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";

import { Providers } from "./providers";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "BOP Dashboard",
  description: "Bot Orchestration Platform control dashboard",
};

// theme-init-script คือสคริปต์เล็กๆ ที่ต้องรันก่อน React hydrate เสมอ (ใส่ตรงๆ ใน
// <head> ผ่าน dangerouslySetInnerHTML ไม่ใช่ import ไฟล์ .js เพราะต้องเป็น
// render-blocking แบบ inline ถึงจะทันเวลาก่อน browser paint ครั้งแรก) อ่าน theme ที่
// เคยเลือกไว้จาก localStorage (fallback ไปที่ prefers-color-scheme ถ้ายังไม่เคยเลือก)
// แล้วใส่ class "dark" ให้ <html> ทันที — ป้องกัน "flash of wrong theme" (เห็นหน้าขาว
// วาบก่อนสลับเป็นมืดตอน React เพิ่ง mount ThemeProvider เสร็จ) ตรรกะต้องตรงกับ
// getInitialTheme() ใน theme-provider.tsx เป๊ะ (คนละที่กันแต่ทำเรื่องเดียวกัน เพราะ
// อันนี้ต้องรันเป็น plain JS ก่อน React จะทำงานได้เลยด้วยซ้ำ)
const themeInitScript = `
(function () {
  try {
    var stored = window.localStorage.getItem("bop-theme");
    var isDark = stored === "dark" || (stored !== "light" && window.matchMedia("(prefers-color-scheme: dark)").matches);
    document.documentElement.classList.toggle("dark", isDark);
  } catch (e) {}
})();
`;

// RootLayout ยังเป็น Server Component ตามปกติ (ไม่มี "use client") — QueryClientProvider
// ที่ต้องทำงานฝั่ง client ถูกแยกไปอยู่ใน <Providers> ต่างหาก (ดู providers.tsx) ครอบ
// {children} ไว้อีกชั้นเดียว
//
// suppressHydrationWarning บน <html> จำเป็นเพราะ theme-init-script (ด้านบน) แก้ class
// ของ <html> ก่อน React hydrate — ทำให้ HTML ที่ server render (ไม่มี class "dark")
// กับ DOM จริงตอน browser โหลดเสร็จ (อาจมี class "dark" แล้ว) ไม่ตรงกัน ซึ่งเป็นกรณีที่
// รู้/ตั้งใจอยู่แล้ว ไม่ใช่ bug จึงบอก React ให้ข้าม warning นี้ไปเฉพาะ attribute
// class ของ element นี้เท่านั้น (ไม่กระทบการเช็ค hydration mismatch อื่นๆ ในแอป)
export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeInitScript }} />
      </head>
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased`}
      >
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
