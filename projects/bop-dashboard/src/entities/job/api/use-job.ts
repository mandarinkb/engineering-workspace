import { useQuery } from "@tanstack/react-query";

import { apiClient } from "@/lib/api-client";
import type { Job } from "../types";

// useJob เรียก GET /jobs/{id} — ใช้แสดงหัวข้อ/สถานะ/summary ของหน้า live monitor (ข้อ
// 1.5) คู่กับ log ที่มาจาก useJobLogStream แยกต่างหาก (คนละ endpoint คนละ concern:
// อันนี้คือ "สถานะปัจจุบันของ job" ส่วน log stream คือ "ข้อความ progress ระหว่างทาง")
//
// refetchInterval แบบ function เช็คสถานะปัจจุบันของ query เอง — poll ทุก 2 วินาที
// ตราบใดที่ job ยังไม่จบ (pending/running) แล้วหยุด poll เองอัตโนมัติทันทีที่จบ
// (succeeded/failed/cancelled) ไม่ต้องเขียน setInterval/clearInterval เอง TanStack
// Query จัดการ lifecycle นี้ให้ทั้งหมด
export function useJob(id: string) {
  return useQuery({
    queryKey: ["jobs", id],
    queryFn: () => apiClient<Job>(`/jobs/${id}`),
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      return status === "pending" || status === "running" ? 2000 : false;
    },
  });
}
