import { useQuery } from "@tanstack/react-query";

import { apiClient } from "@/lib/api-client";
import type { ScheduleDetail } from "../types";

// useSchedule เรียก GET /schedules/{id} — รวม overlap policy, ประวัติการรันล่าสุด,
// next run time, และ steps (มาจาก internal/schedule.Repository ฝั่ง backend อาจว่าง
// เปล่าได้ ดูหมายเหตุที่ features/schedules/types/index.ts)
export function useSchedule(id: string) {
  return useQuery({
    queryKey: ["schedules", id],
    queryFn: () => apiClient<ScheduleDetail>(`/schedules/${encodeURIComponent(id)}`),
  });
}
