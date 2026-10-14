import { StatusBadge, type BadgeTone } from "@/components/ui/status-badge";

import type { PipelineNodeData } from "../lib/pipeline";

const TONE_BY_STATUS: Record<PipelineNodeData["status"], BadgeTone> = {
  succeeded: "green",
  failed: "red",
  running: "yellow",
};

interface PipelineNodeProps {
  node: PipelineNodeData;
}

// PipelineNode คือ node เดียวในผัง (ดู step-pipeline-view.tsx) — ใช้ BadgeTone/StatusBadge
// เดียวกับ WorkflowStatusBadge/JobStatusBadge เพื่อสีตรงกันทั้งแอป (เขียว=สำเร็จ,
// แดง=ล้มเหลว, เหลือง=กำลังทำงาน)
export function PipelineNode({ node }: PipelineNodeProps) {
  return (
    <div className="flex min-w-40 flex-col gap-1 rounded border border-gray-200 p-3 dark:border-gray-700">
      <div className="flex items-center justify-between gap-2">
        <span className="font-mono text-xs text-gray-500 dark:text-gray-400">step {node.stepIndex}</span>
        <StatusBadge label={node.status} tone={TONE_BY_STATUS[node.status]} />
      </div>
      <p className="truncate font-mono text-sm" title={node.botName}>
        {node.botName}
      </p>
      {node.error ? (
        <p className="text-xs text-red-600 dark:text-red-400">{node.error}</p>
      ) : node.summary ? (
        <p className="text-xs text-gray-600 dark:text-gray-300">{node.summary}</p>
      ) : null}
    </div>
  );
}
