import type { RunStatusFilter } from "@/entities/workflow-run";

// ป้ายของตัวเลือกใน dropdown สถานะ — ค่าจริงส่งไปกรองฝั่ง server ("failed" รวม failed/
// terminated/timed out ไว้ด้วยกัน ตรงกับที่ badge ใช้สีแดง เพราะคนดูสนใจว่า "พังไหม" มากกว่า
// พังแบบไหน)
export const RUN_STATUS_FILTER_LABELS: Record<Exclude<RunStatusFilter, "">, string> = {
  running: "running",
  completed: "succeeded",
  failed: "failed / terminated / timed out",
  canceled: "cancelled",
};
