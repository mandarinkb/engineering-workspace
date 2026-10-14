import { StatusBadge, type BadgeTone } from "@/components/ui/status-badge";

// STATUS_META แปลง Temporal enum string ดิบ (เช่น "WORKFLOW_EXECUTION_STATUS_RUNNING" —
// ดูที่มาที่ info.GetStatus().String() ใน internal/api/http/handler.go: toRunSummary)
// ให้เป็น label สั้นๆ อ่านง่ายกับสีที่สอดคล้องกับ JobStatusBadge (running=เหลือง,
// completed=เขียว, failed/timeout/terminated=แดง) — ให้ UX สอดคล้องกันทั้งหน้า job
// history และหน้า workflow run แม้จะเป็นคนละ status enum กันจากต้นทาง
const STATUS_META: Record<string, { label: string; tone: BadgeTone }> = {
  WORKFLOW_EXECUTION_STATUS_RUNNING: { label: "running", tone: "yellow" },
  WORKFLOW_EXECUTION_STATUS_COMPLETED: { label: "succeeded", tone: "green" },
  WORKFLOW_EXECUTION_STATUS_FAILED: { label: "failed", tone: "red" },
  WORKFLOW_EXECUTION_STATUS_CANCELED: { label: "cancelled", tone: "gray" },
  WORKFLOW_EXECUTION_STATUS_TERMINATED: { label: "terminated", tone: "red" },
  WORKFLOW_EXECUTION_STATUS_TIMED_OUT: { label: "timed out", tone: "red" },
  WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW: { label: "continued", tone: "gray" },
};

export function WorkflowStatusBadge({ status }: { status: string }) {
  // fallback: ถ้าเจอ enum string ที่ไม่ได้ map ไว้ (Temporal เพิ่ม status ใหม่ในอนาคต)
  // แสดง string ดิบไปเลยแทนที่จะ crash หรือซ่อนข้อมูลไป — ดีกว่าให้ user เห็นอะไรบางอย่าง
  // มากกว่าไม่เห็นอะไรเลย
  const meta = STATUS_META[status] ?? { label: status, tone: "gray" as const };
  return <StatusBadge label={meta.label} tone={meta.tone} />;
}
