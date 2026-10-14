"use client";

import Link from "next/link";

import { useWorkflowRuns } from "../api/use-workflow-runs";
import { WorkflowStatusBadge } from "./workflow-status-badge";

// WorkflowRunTable แสดงว่า scheduled workflow (ตัวอย่าง check-example-com ที่
// cmd/bop-worker สร้าง Schedule ไว้ — ดู TEMPORAL-MIGRATION-TODO.md) รันไปแล้วกี่รอบ
// สำเร็จ/ล้มเหลวรอบไหนบ้าง — ไม่มี filter/pagination เหมือน job-table.tsx เพราะจำนวน
// workflow run ยังน้อยกว่า job เดี่ยวๆ มาก (แค่ workflow เดียวที่รันทุก 1 นาที)
export function WorkflowRunTable() {
  const { data: runs, isLoading, isError, error } = useWorkflowRuns();

  if (isLoading) {
    return <p className="text-gray-500">กำลังโหลด workflow run...</p>;
  }
  if (isError) {
    return <p className="text-red-600">โหลด workflow run ไม่สำเร็จ: {error.message}</p>;
  }
  if (!runs || runs.length === 0) {
    return <p className="text-gray-500">ยังไม่มี workflow run เลย</p>;
  }

  return (
    <table className="w-full border-collapse text-left text-sm">
      <thead>
        <tr className="border-b border-gray-200 text-gray-500">
          <th className="py-2 pr-4">Workflow</th>
          <th className="py-2 pr-4">Status</th>
          <th className="py-2 pr-4">เริ่มเมื่อ</th>
        </tr>
      </thead>
      <tbody>
        {runs.map((run) => (
          <tr key={run.workflow_id} className="border-b border-gray-100 hover:bg-gray-50">
            <td className="py-2 pr-4">
              <Link href={`/workflow-runs/${run.workflow_id}`} className="font-mono text-blue-600 hover:underline">
                {run.workflow_name}
              </Link>
            </td>
            <td className="py-2 pr-4">
              <WorkflowStatusBadge status={run.status} />
            </td>
            <td className="py-2 pr-4 text-gray-500">{new Date(run.started_at).toLocaleString()}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
