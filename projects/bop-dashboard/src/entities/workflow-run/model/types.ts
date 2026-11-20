// mirror จาก internal/api/http/handler.go (workflowRunSummary/workflowRunDetail) และ
// internal/workflow/workflow.go (StepResult) — ผลลัพธ์จริงมาจาก Temporal Visibility API
// ไม่ใช่ database ของ BOP เอง (ดู TEMPORAL-MIGRATION-TODO.md) เก็บมาแค่เท่าที่ dashboard
// ต้องใช้แสดงผล ไม่ใช่ field ทั้งหมดที่ Temporal มีให้
export interface WorkflowRunSummary {
  workflow_id: string;
  run_id: string;
  workflow_name: string;
  // ค่าดิบจาก Temporal SDK เช่น "Running", "Completed" (เวอร์ชันเก่าเป็น
  // "WORKFLOW_EXECUTION_STATUS_RUNNING") — แปลงด้วย normalizeRunStatus ก่อนเทียบเสมอ
  status: string;
  started_at: string;
  ended_at?: string;
}

export interface StepResult {
  step_index: number;
  bot_name: string;
  summary?: string;
  error?: string;
}

export interface WorkflowRunDetail extends WorkflowRunSummary {
  // null ตอน run ยังไม่มี step ไหนรันเสร็จ (Go ส่ง nil slice ออกมาเป็น JSON null)
  steps: StepResult[] | null;
}
