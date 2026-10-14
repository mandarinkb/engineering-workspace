import type { StepResult } from "../types";

// ข้อจำกัดสำคัญที่ต้องรู้ก่อนอ่านไฟล์นี้: StepResult (mirror จาก internal/workflow/
// workflow.go ฝั่ง Go) ไม่มี field สถานะของตัวเอง มีแค่ step_index/bot_name/summary?/
// error? — "steps" array ที่ backend คืนมามีแค่ step ที่ "รันเสร็จแล้ว" เท่านั้น (ทั้ง
// สำเร็จและล้มเหลว) ไม่มีวันเห็น step ที่ยังไม่ถึงคิวเลย และไม่มี field บอก "ทั้งหมดกี่
// step" ด้วย — ทำให้ pipeline นี้ render ได้แค่ 2 อย่าง: (1) step ที่รันไปแล้วจริง
// (derive สถานะจาก error: มี error → failed, ไม่มี → succeeded) (2) node พิเศษท้ายสุด
// แสดงว่า "กำลังทำงานอยู่" ถ้า workflow ทั้งก้อนยังไม่จบ — **ไม่เดา** step ที่ยังไม่ถึงคิว
// เพราะไม่มีข้อมูลจริงรองรับเลย (ดู K8S-JOB-HYBRID-EXECUTION-TODO.md/PHASE3-DASHBOARD-TODO.md
// สำหรับ context เต็มๆ ว่าทำไมถึงเป็นข้อจำกัดที่ยอมรับไว้)
export type PipelineNodeStatus = "succeeded" | "failed" | "running";

export interface PipelineNodeData {
  stepIndex: number;
  botName: string;
  status: PipelineNodeStatus;
  summary?: string;
  error?: string;
}

// ค่า status string ดิบของ Temporal ที่แปลว่า "ยังรันอยู่" — ตรงกับ enum
// WORKFLOW_EXECUTION_STATUS_RUNNING ที่ workflow-status-badge.tsx ใช้ map เป็น badge สี
// เหลืองอยู่แล้ว (ใช้ค่าเดียวกัน ไม่ import จากไฟล์นั้นตรงๆ เพราะไม่ได้ export เป็น
// constant แยกไว้ — ประกาศซ้ำที่นี่สั้นกว่าการ refactor ไฟล์นั้นเพื่อ export เพิ่ม)
const RUNNING_STATUS_VALUE = "WORKFLOW_EXECUTION_STATUS_RUNNING";

export function buildPipelineNodes(steps: StepResult[], runStatus: string): PipelineNodeData[] {
  const nodes: PipelineNodeData[] = steps.map((step) => ({
    stepIndex: step.step_index,
    botName: step.bot_name,
    status: step.error ? "failed" : "succeeded",
    summary: step.summary,
    error: step.error,
  }));

  if (runStatus === RUNNING_STATUS_VALUE) {
    nodes.push({
      stepIndex: steps.length,
      botName: "กำลังทำงาน...",
      status: "running",
    });
  }

  return nodes;
}
