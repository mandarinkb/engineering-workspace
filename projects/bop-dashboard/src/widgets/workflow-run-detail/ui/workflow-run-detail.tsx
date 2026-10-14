"use client";

import { useState } from "react";

import { useWorkflowRun } from "../api/use-workflow-run";
import { StepPipelineView } from "./step-pipeline-view";
import { StepResultList } from "./step-result-list";
import { WorkflowStatusBadge } from "./workflow-status-badge";

interface WorkflowRunDetailProps {
  workflowId: string;
}

// WorkflowRunDetail ต้องเป็น Client Component ("use client") เพราะเรียก useWorkflowRun
// (TanStack Query hook) — แยกออกมาจาก page.tsx (Server Component) ตามกติกาเดียวกับ
// job-log-viewer.tsx: page.tsx บางที่สุด ส่วน logic การดึงข้อมูล/render จริงอยู่ใน
// features/ เสมอ
//
// view toggle: default เป็น "pipeline" (ผังแนวนอน ดูภาพรวมง่ายกว่า) แต่เก็บ
// StepResultList (list ธรรมดา) ไว้เป็นทางเลือกเดิมที่สลับกลับไปดูได้เสมอ — บางกรณี list
// อาจอ่านง่ายกว่า (เช่น summary/error ยาวๆ ที่ pipeline node ตัดคำทิ้งไปเพราะพื้นที่จำกัด)
export function WorkflowRunDetail({ workflowId }: WorkflowRunDetailProps) {
  const { data: run, isLoading, isError, error } = useWorkflowRun(workflowId);
  const [view, setView] = useState<"pipeline" | "list">("pipeline");

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
      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <WorkflowStatusBadge status={run.status} />
          <span className="text-sm text-gray-500 dark:text-gray-400">
            เริ่มเมื่อ {new Date(run.started_at).toLocaleString()}
          </span>
        </div>
        <div className="flex gap-1 text-xs">
          <button
            type="button"
            onClick={() => setView("pipeline")}
            className={
              view === "pipeline"
                ? "rounded border border-blue-600 bg-blue-50 px-2 py-1 text-blue-700 dark:border-blue-400 dark:bg-blue-950 dark:text-blue-300"
                : "rounded border border-gray-300 px-2 py-1 text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-800"
            }
          >
            Pipeline
          </button>
          <button
            type="button"
            onClick={() => setView("list")}
            className={
              view === "list"
                ? "rounded border border-blue-600 bg-blue-50 px-2 py-1 text-blue-700 dark:border-blue-400 dark:bg-blue-950 dark:text-blue-300"
                : "rounded border border-gray-300 px-2 py-1 text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-800"
            }
          >
            List
          </button>
        </div>
      </div>
      {view === "pipeline" ? (
        <StepPipelineView steps={run.steps} runStatus={run.status} />
      ) : (
        <StepResultList steps={run.steps} />
      )}
    </div>
  );
}
