"use client";

import { useWorkflowRun } from "../api/use-workflow-run";
import { StepResultList } from "./step-result-list";
import { WorkflowStatusBadge } from "./workflow-status-badge";

interface WorkflowRunDetailProps {
  workflowId: string;
}

// WorkflowRunDetail ต้องเป็น Client Component ("use client") เพราะเรียก useWorkflowRun
// (TanStack Query hook) — แยกออกมาจาก page.tsx (Server Component) ตามกติกาเดียวกับ
// job-log-viewer.tsx: page.tsx บางที่สุด ส่วน logic การดึงข้อมูล/render จริงอยู่ใน
// features/ เสมอ
export function WorkflowRunDetail({ workflowId }: WorkflowRunDetailProps) {
  const { data: run, isLoading, isError, error } = useWorkflowRun(workflowId);

  if (isLoading) {
    return <p className="text-gray-500">กำลังโหลด workflow run...</p>;
  }
  if (isError) {
    return <p className="text-red-600">โหลด workflow run ไม่สำเร็จ: {error.message}</p>;
  }
  if (!run) {
    return null;
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center gap-3">
        <WorkflowStatusBadge status={run.status} />
        <span className="text-sm text-gray-500">เริ่มเมื่อ {new Date(run.started_at).toLocaleString()}</span>
      </div>
      <StepResultList steps={run.steps} />
    </div>
  );
}
