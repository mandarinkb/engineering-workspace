// mirror จาก internal/job/job.go (Job struct) และ internal/logger/logger.go (Line struct)
// — casing เป็น snake_case ตรงกับ json tag ฝั่ง Go เป๊ะ (ทั้ง API ใช้ snake_case สม่ำเสมอ
// กันแล้วทั้งหมด ไม่มีจุดไหนผสม PascalCase อีกต่อไป) เช็ค commit ล่าสุดของ
// projects/bot-orchestration-platform ก่อนเชื่อ 100% เสมอถ้าไฟล์นี้ดูตกยุค
export type JobStatus = "pending" | "running" | "succeeded" | "failed" | "cancelled";

export interface Job {
  id: string;
  bot_name: string;
  config: Record<string, string>;
  status: JobStatus;
  summary?: string;
  error?: string;
  created_at: string; // ISO timestamp string
  started_at?: string;
  ended_at?: string;
}

export type LogLevel = "info" | "warn" | "error";

export interface LogLine {
  job_id: string;
  time: string;
  level: LogLevel;
  message: string;
}
