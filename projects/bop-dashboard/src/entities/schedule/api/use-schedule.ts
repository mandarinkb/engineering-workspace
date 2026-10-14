import { useQuery } from "@tanstack/react-query";

import { apiClient } from "@/shared/api";
import type { ScheduleDetail } from "../model/types";

// useSchedule เรียก GET /schedules/{id} — รวม overlap policy, ประวัติการรันล่าสุด,
// next run time, และ steps (มาจาก internal/schedule.Repository ฝั่ง backend อาจว่าง
// เปล่าได้ ดูหมายเหตุที่ entities/schedule/model/types.ts)
export function useSchedule(id: string) {
  return useQuery({
    queryKey: ["schedules", id],
    queryFn: () => apiClient<ScheduleDetail>(`/schedules/${encodeURIComponent(id)}`),
  });
}
