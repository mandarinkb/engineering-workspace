import { useMutation, useQueryClient } from "@tanstack/react-query";

import { apiClient } from "@/lib/api-client";
import type { OverlapPolicy, ScheduleDetail, ScheduleSpecInput, ScheduleStep } from "../types";

export interface UpdateScheduleInput {
  workflow_name: string;
  steps: ScheduleStep[];
  spec: ScheduleSpecInput;
  overlap?: OverlapPolicy;
}

// useUpdateSchedule เรียก PUT /schedules/{id} — full replace เสมอ (ไม่ใช่ partial patch
// ดู comment ของ handleUpdateSchedule ฝั่ง backend) รับ id ตอนเรียก hook เพราะ URL ผูก
// กับ schedule ที่กำลังแก้ไขอยู่ตั้งแต่ต้น
export function useUpdateSchedule(id: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: UpdateScheduleInput) =>
      apiClient<ScheduleDetail>(`/schedules/${encodeURIComponent(id)}`, { method: "PUT", body: input }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["schedules"] });
      queryClient.invalidateQueries({ queryKey: ["schedules", id] });
    },
  });
}
