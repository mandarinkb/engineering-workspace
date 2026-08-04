import { useMutation, useQueryClient } from "@tanstack/react-query";

import { apiClient } from "@/lib/api-client";
import type { OverlapPolicy, ScheduleDetail, ScheduleSpecInput, ScheduleStep } from "../types";

export interface CreateScheduleInput {
  id: string;
  workflow_name: string;
  steps: ScheduleStep[];
  spec: ScheduleSpecInput;
  overlap?: OverlapPolicy;
}

// useCreateSchedule เรียก POST /schedules — invalidate query "schedules" (list) ให้
// refetch ทันทีหลังสร้างสำเร็จ เหมือน pattern เดียวกับ useCreateJob ใน features/bots
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
