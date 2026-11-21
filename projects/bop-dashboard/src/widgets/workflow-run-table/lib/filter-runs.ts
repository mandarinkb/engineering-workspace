import { isFailedRunStatus, normalizeRunStatus, type WorkflowRunSummary } from "@/entities/workflow-run";

// RunStatusFilter คือตัวเลือกใน dropdown — "failed" รวม failed/terminated/timed out ไว้ด้วยกัน
// (ตรงกับที่ badge ใช้สีแดง) เพราะคนดูสนใจว่า "พังไหม" มากกว่าพังแบบไหน
export type RunStatusFilter = "" | "running" | "completed" | "failed" | "canceled";

export const RUN_STATUS_FILTER_LABELS: Record<Exclude<RunStatusFilter, "">, string> = {
  running: "running",
  completed: "succeeded",
  failed: "failed / terminated / timed out",
  canceled: "cancelled",
};

export interface RunFilters {
  status: RunStatusFilter;
  // ค้นหาใน workflow_id แบบไม่สนตัวพิมพ์ — workflow ID ของ run ที่ schedule สั่งขึ้นต้นด้วย
  // ชื่อ schedule (เช่น check-example-com-2026-…) จึงพิมพ์ชื่อ schedule เพื่อกรองได้
  search: string;
}

function matchesStatus(run: WorkflowRunSummary, filter: RunStatusFilter): boolean {
  if (filter === "") return true;
  const status = normalizeRunStatus(run.status);
  return filter === "failed" ? isFailedRunStatus(status) : status === filter;
}

// filterRuns กรองแล้วเรียงใหม่สุดก่อน (backend เรียงมาแล้วก็จริง แต่ไม่ได้รับประกันไว้)
export function filterRuns(runs: WorkflowRunSummary[], filters: RunFilters): WorkflowRunSummary[] {
  const search = filters.search.trim().toLowerCase();
  return runs
    .filter((run) => matchesStatus(run, filters.status))
    .filter((run) => search === "" || run.workflow_id.toLowerCase().includes(search))
    .sort((a, b) => new Date(b.started_at).getTime() - new Date(a.started_at).getTime());
}
