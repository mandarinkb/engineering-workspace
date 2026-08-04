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

// RootLayout ยังเป็น Server Component ตามปกติ (ไม่มี "use client") — QueryClientProvider
// ที่ต้องทำงานฝั่ง client ถูกแยกไปอยู่ใน <Providers> ต่างหาก (ดู providers.tsx) ครอบ
// {children} ไว้อีกชั้นเดียว
export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased`}
      >
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
