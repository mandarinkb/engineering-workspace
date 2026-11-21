import type { WorkflowRunListParams, WorkflowRunPage, WorkflowRunSummary } from "../model/types";

// runListPath สร้าง path ของ GET /workflow-runs — page_token เป็น opaque string จาก backend
// (base64 ของ NextPageToken ของ Temporal) ส่งกลับไปตามเดิมเท่านั้น ห้าม parse
export function runListPath(params: WorkflowRunListParams, pageToken?: string): string {
  const query = new URLSearchParams();
  if (params.status) query.set("status", params.status);
  if (params.scheduleId) query.set("schedule_id", params.scheduleId);
  if (params.pageSize) query.set("page_size", String(params.pageSize));
  if (pageToken) query.set("page_token", pageToken);
  const qs = query.toString();
  return qs ? `/workflow-runs?${qs}` : "/workflow-runs";
}

// nextRunPageToken ใช้เป็น getNextPageParam ของ useInfiniteQuery — undefined = หน้าสุดท้าย
export function nextRunPageToken(page: WorkflowRunPage): string | undefined {
  return page.next_page_token || undefined;
}

// flattenRunPages ต่อทุกหน้าที่โหลดมาเป็นรายการเดียว และตัด run ที่ซ้ำทิ้ง — ปกติ token ของ
// Temporal ไม่ให้ run ซ้ำข้ามหน้า แต่กันไว้เผื่อ run ใหม่เข้ามาระหว่างกดโหลดเพิ่มแล้วลำดับเลื่อน
// (React key ซ้ำในตารางจะทำให้ render เพี้ยน)
export function flattenRunPages(pages: WorkflowRunPage[]): WorkflowRunSummary[] {
  const seen = new Set<string>();
  const runs: WorkflowRunSummary[] = [];
  for (const page of pages) {
    for (const run of page.runs) {
      const key = `${run.workflow_id}/${run.run_id}`;
      if (seen.has(key)) continue;
      seen.add(key);
      runs.push(run);
    }
  }
  return runs;
}
