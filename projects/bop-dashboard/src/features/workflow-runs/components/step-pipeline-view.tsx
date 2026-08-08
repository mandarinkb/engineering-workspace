import { buildPipelineNodes } from "../lib/pipeline";
import type { StepResult } from "../types";
import { PipelineNode } from "./pipeline-node";

interface StepPipelineViewProps {
  steps: StepResult[];
  runStatus: string;
}

// StepPipelineView คือ "Feature 1" ของ PHASE3-DASHBOARD-TODO.md (Workflow Relationship
// View เฟส A) — แสดง step เรียงต่อกันเป็นแนวนอน (แนวตั้งบนจอเล็ก) เชื่อมด้วยเส้นธรรมดา
// ไม่ใช้ library กราฟ (react-flow ฯลฯ) เพราะข้อมูลปัจจุบันเป็นสายตรงเสมอ (ดู
// เหตุผลเต็มๆ ที่ lib/pipeline.ts) — ถ้า backend รองรับ condition-based relationship
// จริง (many-to-many ข้าม workflow) ในอนาคต ค่อยพิจารณาสลับไปใช้ library กราฟจริงจัง
export function StepPipelineView({ steps, runStatus }: StepPipelineViewProps) {
  const nodes = buildPipelineNodes(steps, runStatus);

  if (nodes.length === 0) {
    return <p className="text-gray-500">ยังไม่มี step ไหนรันเสร็จเลย (workflow อาจเพิ่งเริ่ม)</p>;
  }

  return (
    <div className="flex flex-col items-stretch gap-0 overflow-x-auto sm:flex-row sm:items-center">
      {nodes.map((node, i) => (
        <div key={node.stepIndex} className="flex flex-col items-center sm:flex-row">
          <PipelineNode node={node} />
          {i < nodes.length - 1 && (
            // เส้นเชื่อมระหว่าง node — แนวตั้งบนจอเล็ก (flex-col ของ parent), แนวนอนบน
            // จอใหญ่ (sm:flex-row) สลับทิศทางตาม breakpoint เดียวกับ container ด้านบน
            <div aria-hidden className="h-6 w-px bg-gray-300 sm:h-px sm:w-6 dark:bg-gray-600" />
          )}
        </div>
      ))}
    </div>
  );
}
