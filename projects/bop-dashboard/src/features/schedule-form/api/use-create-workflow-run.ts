import { useMutation, useQueryClient } from "@tanstack/react-query";

import { apiClient } from "@/shared/api";
import type { ScheduleStep } from "@/entities/schedule";

export interface CreateWorkflowRunInput {
  workflow_name: string;
  steps: ScheduleStep[];
}

export interface CreateWorkflowRunResult {
  workflow_id: string;
  run_id: string;
}

// useCreateWorkflowRun ยิง POST /workflow-runs — รัน steps ทันทีครั้งเดียวโดยไม่ต้องสร้าง
// schedule (backend ตอบ 202 พร้อม workflow_id/run_id ของ run ที่เพิ่งเริ่ม)
export function useCreateWorkflowRun() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: CreateWorkflowRunInput) =>
      apiClient<CreateWorkflowRunResult>("/workflow-runs", { method: "POST", body: input }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["workflow-runs"] });
    },
  });
}
