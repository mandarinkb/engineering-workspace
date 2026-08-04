// mirror จาก internal/api/http/schedule_handler.go — ห่อ client.ScheduleClient ของ
// Temporal เป็น HTTP endpoint เต็มชุด (create/list/get/update/delete/pause/unpause/
// trigger/backfill) เทียบเท่าการจัดการ job ของ Control-M

// OverlapPolicy คือชื่อ policy ที่ backend ยอมรับ (mirror จาก overlapPolicyByName ใน
// schedule_handler.go) — ตัดสินใจ "ถ้ารอบเก่ายังไม่จบ แล้วถึงเวลารอบใหม่พอดี จะทำยังไง"
export type OverlapPolicy = "skip" | "buffer_one" | "buffer_all" | "cancel_other" | "terminate_other" | "allow_all";

// ScheduleStep คือ 1 ขั้นตอนของ workflow ที่ schedule นี้สั่งรัน — mirror จาก
// internal/workflow/workflow.go (Step struct) เก็บแยกจาก StepResult ของ
// features/workflow-runs/types (นั่นคือ "ผลลัพธ์หลังรันไปแล้ว" ส่วนนี้คือ "นิยามว่าจะรันอะไร")
export interface ScheduleStep {
  bot_name: string;
  config: Record<string, string>;
}

// ScheduleSpec คือ JSON shape ของ "เมื่อไหร่ควรรัน" ที่ POST/PUT ส่งไป — mutually
// exclusive กันเสมอ (ระบุได้แค่อย่างใดอย่างหนึ่ง ไม่ใช่ทั้งคู่ ไม่ใช่ไม่มีเลย)
export interface ScheduleSpecInput {
  interval_seconds?: number;
  cron_expression?: string;
}

// ScheduleSummary คือรูปแบบที่ GET /schedules (list) คืนมา
export interface ScheduleSummary {
  id: string;
  workflow_name: string;
  paused: boolean;
  note?: string;
  interval_seconds?: number;
  cron_expression?: string;
  next_action_times?: string[];
}

export interface ScheduleAction {
  schedule_time: string;
  actual_time: string;
  workflow_id?: string;
}

// ScheduleDetail คือรูปแบบที่ GET /schedules/{id} คืนมา — steps มาจาก
// internal/schedule.Repository ของ backend เอง (ไม่ใช่จาก Temporal โดยตรง) อาจว่างเปล่า
// ได้ถ้า schedule นี้สร้างก่อนมี API ชุดนี้ (เช่น check-example-com-schedule ที่
// cmd/bop-worker สร้างตรงผ่าน SDK) หรือ backend เพิ่ง restart ไป — ดูหมายเหตุเต็มที่
// internal/schedule/schedule.go ฝั่ง backend
export interface ScheduleDetail extends ScheduleSummary {
  overlap: OverlapPolicy;
  steps?: ScheduleStep[];
  recent_actions?: ScheduleAction[];
  created_at?: string;
}
