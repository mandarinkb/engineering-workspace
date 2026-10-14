import { useQuery } from "@tanstack/react-query";

import { apiClient } from "@/lib/api-client";
import type { WorkflowRunDetail } from "../types";

// useWorkflowRun เรียก GET /workflow-runs/{id} — id ตรงนี้คือ Temporal Workflow ID
// (ไม่ใช่ random Run ID แบบเวอร์ชันก่อนย้ายไป Temporal) ฝั่ง backend ดึงผลลัพธ์ทีละ step
// ผ่าน Temporal Query handler "steps" ที่ประกาศไว้ใน workflow.Execute (ดู
// internal/workflow/temporal.go) ทำงานได้ทั้งตอน workflow ยังรันอยู่และจบไปแล้ว
export function useWorkflowRun(id: string) {
  return useQuery({
    queryKey: ["workflow-runs", id],
    queryFn: () => apiClient<WorkflowRunDetail>(`/workflow-runs/${encodeURIComponent(id)}`),
  });
}
