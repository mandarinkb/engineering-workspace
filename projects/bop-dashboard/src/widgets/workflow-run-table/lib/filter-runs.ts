import type { RunStatusFilter, WorkflowRunSummary } from "@/entities/workflow-run";

// ป้ายของตัวเลือกใน dropdown สถานะ — ค่าจริงส่งไปกรองฝั่ง server ("failed" รวม failed/
// terminated/timed out ไว้ด้วยกัน ตรงกับที่ badge ใช้สีแดง เพราะคนดูสนใจว่า "พังไหม" มากกว่า
// พังแบบไหน)
export const RUN_STATUS_FILTER_LABELS: Record<Exclude<RunStatusFilter, "">, string> = {
  running: "running",
  completed: "succeeded",
  failed: "failed / terminated / timed out",
  canceled: "cancelled",
};

// searchRuns ค้นหาใน workflow_id แบบไม่สนตัวพิมพ์ เฉพาะแถวที่โหลดมาแล้ว — workflow ID ของ run
// ที่ schedule สั่งขึ้นต้นด้วยชื่อ schedule (เช่น check-example-com-2026-…) จึงพิมพ์ชื่อ
// schedule เพื่อกรองได้ ลำดับคงตามที่ backend ส่งมา (ใหม่สุดก่อน)
export function searchRuns(runs: WorkflowRunSummary[], search: string): WorkflowRunSummary[] {
  const needle = search.trim().toLowerCase();
  if (needle === "") return runs;
  return runs.filter((run) => run.workflow_id.toLowerCase().includes(needle));
}
