// TEMPORAL_UI_URL คือ base URL ของ Temporal Web UI สำหรับลิงก์ออกไปดู workflow ตัวจริง —
// NEXT_PUBLIC_ ถูก inline เข้า JS bundle ตอน `next build` เหมือน NEXT_PUBLIC_API_URL
// (ส่งผ่าน build arg ใน Dockerfile/compose เท่านั้น ตั้งตอน container รันไม่มีผล)
export const TEMPORAL_UI_URL = (process.env.NEXT_PUBLIC_TEMPORAL_UI_URL ?? "http://localhost:8233").replace(/\/+$/, "");

// namespace เดียวที่ BOP ใช้ (ดู internal/external/temporal/workflow.go ของ backend)
const TEMPORAL_NAMESPACE = "default";

// temporalWorkflowUrl คืนลิงก์ไปหน้า history ของ workflow run หนึ่งตัวใน Temporal UI
export function temporalWorkflowUrl(workflowId: string, runId: string): string {
  return `${TEMPORAL_UI_URL}/namespaces/${TEMPORAL_NAMESPACE}/workflows/${encodeURIComponent(workflowId)}/${encodeURIComponent(runId)}/history`;
}
