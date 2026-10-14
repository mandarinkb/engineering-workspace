import { useMutation, useQueryClient } from "@tanstack/react-query";

import { apiClient } from "@/shared/api";
import type { OverlapPolicy, ScheduleDetail, ScheduleSpecInput, ScheduleStep } from "@/entities/schedule";

export interface CreateScheduleInput {
  id: string;
  workflow_name: string;
  steps: ScheduleStep[];
  spec: ScheduleSpecInput;
  overlap?: OverlapPolicy;
}

// useCreateSchedule เรียก POST /schedules — invalidate query "schedules" (list) ให้
// refetch ทันทีหลังสร้างสำเร็จ เหมือน pattern เดียวกับ useCreateJob ใน features/create-job
export function useCreateSchedule() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: CreateScheduleInput) =>
      apiClient<ScheduleDetail>("/schedules", { method: "POST", body: input }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["schedules"] });
    },
  });
}
