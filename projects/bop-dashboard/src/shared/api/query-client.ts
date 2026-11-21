import { QueryClient } from "@tanstack/react-query";

import { ApiError } from "./api-error";

// LIST_REFETCH_INTERVAL_MS คือรอบ refetch อัตโนมัติของหน้า list (jobs, schedules, workflow
// runs) ให้ dashboard ที่เปิดค้างไว้เห็นตัวเลขขยับเองโดยไม่ต้องกด refresh — TanStack Query
// หยุด refetch ตอนแท็บถูกซ่อนอยู่ให้เอง (refetchIntervalInBackground ค่า default เป็น false)
export const LIST_REFETCH_INTERVAL_MS = 10_000;

// createQueryClient สร้าง TanStack QueryClient ตัวใหม่พร้อม config กลางที่ใช้ร่วมกันทั้ง
// แอป — แยกเป็นฟังก์ชันแทนที่จะสร้าง instance เดียวระดับ module เพราะ Next.js App Router
// render ฝั่ง server ด้วย (แม้ query hook ส่วนใหญ่จะทำงานเฉพาะฝั่ง client) การสร้าง
// QueryClient ใหม่ทุกครั้งที่ Providers component (src/app/providers/providers.tsx) mount ป้องกันไม่ให้
// request ของ user คนหนึ่งไปแชร์ cache กับ user อีกคนโดยไม่ตั้งใจตอนรันบน server ที่มีหลาย
// request พร้อมกัน (ทุก request ควรได้ QueryClient ของตัวเอง)
export function createQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        // staleTime 30 วินาที: ข้อมูล (เช่น job list) ที่เพิ่งโหลดมาไม่ถึง 30 วินาที
        // ยังถือว่า "สดอยู่" ไม่ต้อง refetch ซ้ำทันทีถ้ามี component อื่น mount ขึ้นมาขอ
        // ข้อมูลเดียวกัน — ค่ากลางๆ ที่สมดุลระหว่างข้อมูลใหม่กับจำนวน request ที่ยิงออกไป
        staleTime: 30_000,
        // retry เฉพาะ error ที่ "อาจจะ" หายเองได้ถ้าลองใหม่ (เช่น network ล่มชั่วคราว,
        // 5xx จาก server) — ไม่ retry error 4xx เพราะ retry กี่ครั้งก็ได้ผลลัพธ์เดิม
        // เสมอ (เช่น 401 unauthorized, 404 not found) เสียเวลาและ resource ไปฟรีๆ
        // เหมือนหลักการเดียวกับ NonRetryableErrorTypes ฝั่ง Temporal ใน backend
        retry: (failureCount, error) => {
          if (error instanceof ApiError && error.status >= 400 && error.status < 500) {
            return false;
          }
          return failureCount < 2;
        },
      },
    },
  });
}
