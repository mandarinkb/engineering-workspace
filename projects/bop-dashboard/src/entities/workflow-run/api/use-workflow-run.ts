import { useQuery } from "@tanstack/react-query";

import { apiClient } from "@/shared/api";
import type { WorkflowRunDetail } from "../model/types";
import { isRunInProgress } from "../lib/run-status";

// ดึงรายละเอียด run ใหม่ทุก 2 วินาทีจนกว่าจะจบ — เท่ากับหน้า Job (entities/job/api/use-job.ts)
// ไม่งั้นหน้าที่เปิดค้างตอน run ยังรันอยู่จะค้างที่ "running" แม้ run จบไปแล้ว
const RUN_DETAIL_REFETCH_MS = 2000;

// useWorkflowRun เรียก GET /workflow-runs/{id} — id ตรงนี้คือ Temporal Workflow ID
// (ไม่ใช่ random Run ID แบบเวอร์ชันก่อนย้ายไป Temporal) ฝั่ง backend ดึงผลลัพธ์ทีละ step
// ผ่าน Temporal Query handler "steps" ที่ประกาศไว้ใน workflow.Execute (ดู
// internal/workflow/temporal.go) ทำงานได้ทั้งตอน workflow ยังรันอยู่และจบไปแล้ว
export function useWorkflowRun(id: string) {
  return useQuery({
    queryKey: ["workflow-runs", id],
    queryFn: () => apiClient<WorkflowRunDetail>(`/workflow-runs/${encodeURIComponent(id)}`),
    refetchInterval: (query) => (isRunInProgress(query.state.data?.status) ? RUN_DETAIL_REFETCH_MS : false),
  });
}
