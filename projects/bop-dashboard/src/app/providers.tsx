"use client";

import { QueryClientProvider } from "@tanstack/react-query";
import { useState } from "react";

import { createQueryClient } from "@/lib/query-client";

// Providers ต้องเป็น Client Component ("use client" บรรทัดบนสุด) เพราะ
// QueryClientProvider ใช้ React Context ซึ่งทำงานได้เฉพาะฝั่ง client เท่านั้น — ถ้าเอา
// QueryClientProvider ไปใส่ตรงๆ ใน layout.tsx (ที่เป็น Server Component โดย default)
// จะ error ทันที ต้องแยกเป็น component ลูกแบบนี้แล้ว mark "use client" เฉพาะไฟล์นี้
// (root layout เองยังเป็น Server Component ได้ตามปกติ ไม่ต้อง "use client" ทั้งไฟล์)
//
// useState(() => createQueryClient()) แทนที่จะเป็นตัวแปร module-level ธรรมดา เพราะ
// Next.js render ฝั่ง server ด้วย (แม้ query hook ส่วนใหญ่จะทำงานฝั่ง client จริงๆ) —
// ถ้าใช้ตัวแปร module-level เดียว ทุก request ที่ server กำลัง render พร้อมกันจะแชร์
// QueryClient (และ cache ข้างใน) ตัวเดียวกันโดยไม่ตั้งใจ ซึ่งเสี่ยงข้อมูลของ user คนหนึ่ง
// รั่วไปโผล่ในของอีกคน — useState ทำให้แต่ละ component tree (แต่ละ request/แต่ละ client)
// ได้ QueryClient ของตัวเองเสมอ ฟังก์ชันใน useState เรียกแค่ครั้งเดียวตอน mount ครั้งแรก
// (lazy initialization) ไม่ใช่ทุกครั้งที่ re-render
export function Providers({ children }: { children: React.ReactNode }) {
  const [queryClient] = useState(() => createQueryClient());
  return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
}
