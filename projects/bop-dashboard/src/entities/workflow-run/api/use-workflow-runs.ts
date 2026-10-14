import { useQuery } from "@tanstack/react-query";

import { apiClient } from "@/lib/api-client";
import type { WorkflowRunSummary } from "../types";

// useWorkflowRuns เรียก GET /workflow-runs — ต่างจาก useJobs ตรงที่ backend endpoint นี้
// ไม่ได้อ่านจาก database ของ BOP เองเลย แต่ query ตรงไปที่ Temporal Visibility API (ดู
// internal/api/http/handler.go: handleListWorkflowRuns) ทุกครั้งที่เรียก — Temporal เป็น
// คนเก็บ event history ของ workflow run ทั้งหมดให้แทน BOP อยู่แล้ว
export function useWorkflowRuns() {
  return useQuery({
    queryKey: ["workflow-runs"],
    queryFn: () => apiClient<WorkflowRunSummary[]>("/workflow-runs"),
  });
}
