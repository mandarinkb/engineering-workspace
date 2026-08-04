import { useQuery } from "@tanstack/react-query";

import { apiClient } from "@/lib/api-client";
import type { Job } from "../types";

// useJobs เรียก GET /jobs — ใช้กับหน้า job history (roadmap ข้อ 1.4) filter ตาม
// bot/status/ช่วงเวลาทำฝั่ง client ล้วนๆ ใน job-table.tsx (ไม่ได้ทำ server-side filter
// เพราะ backend ยังไม่มี query parameter รองรับ และจำนวน job ทั้งระบบตอนนี้ยังไม่เยอะ
// พอที่ client-side filter จะกลายเป็นปัญหา performance)
export function useJobs() {
  return useQuery({
    queryKey: ["jobs"],
    queryFn: () => apiClient<Job[]>("/jobs"),
  });
}
