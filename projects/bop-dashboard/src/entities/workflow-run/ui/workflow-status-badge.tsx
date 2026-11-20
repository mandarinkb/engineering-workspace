import { StatusBadge, type BadgeTone } from "@/shared/ui";

import { normalizeRunStatus, type RunStatus } from "../lib/run-status";

// STATUS_META แปลงสถานะ (ผ่าน normalizeRunStatus ซึ่งรับทั้ง "Running" และ "WORKFLOW_EXECUTION_STATUS_RUNNING" —
// ดูที่มาที่ info.GetStatus().String() ใน internal/api/http/handler.go: toRunSummary)
// ให้เป็น label สั้นๆ อ่านง่ายกับสีที่สอดคล้องกับ JobStatusBadge (running=เหลือง,
// completed=เขียว, failed/timeout/terminated=แดง) — ให้ UX สอดคล้องกันทั้งหน้า job
// history และหน้า workflow run แม้จะเป็นคนละ status enum กันจากต้นทาง
const STATUS_META: Record<RunStatus, { label: string; tone: BadgeTone }> = {
  running: { label: "running", tone: "yellow" },
  completed: { label: "succeeded", tone: "green" },
  failed: { label: "failed", tone: "red" },
  canceled: { label: "cancelled", tone: "gray" },
  terminated: { label: "terminated", tone: "red" },
  timed_out: { label: "timed out", tone: "red" },
  continued_as_new: { label: "continued", tone: "gray" },
  unknown: { label: "unknown", tone: "gray" },
};

export function WorkflowStatusBadge({ status }: { status: string }) {
  // fallback: ถ้าเจอ enum string ที่ไม่ได้ map ไว้ (Temporal เพิ่ม status ใหม่ในอนาคต)
  // แสดง string ดิบไปเลยแทนที่จะ crash หรือซ่อนข้อมูลไป — ดีกว่าให้ user เห็นอะไรบางอย่าง
  // มากกว่าไม่เห็นอะไรเลย
  const normalized = normalizeRunStatus(status);
  const meta = normalized === "unknown" ? { label: status, tone: "gray" as const } : STATUS_META[normalized];
  return <StatusBadge label={meta.label} tone={meta.tone} />;
}
