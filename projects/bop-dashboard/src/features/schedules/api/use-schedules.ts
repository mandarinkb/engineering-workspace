import { useQuery } from "@tanstack/react-query";

import { apiClient } from "@/lib/api-client";
import type { ScheduleSummary } from "../types";

// useSchedules เรียก GET /schedules — list เร็วเพราะ backend ไม่ Describe() ทีละตัว
// (ดู handleListSchedules ใน internal/api/http/schedule_handler.go) เหมาะกับหน้า list
// ที่อาจมี schedule จำนวนมาก
export function useSchedules() {
  return useQuery({
    queryKey: ["schedules"],
    queryFn: () => apiClient<ScheduleSummary[]>("/schedules"),
  });
}
