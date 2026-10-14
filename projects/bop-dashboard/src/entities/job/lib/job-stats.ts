import type { Job } from "../types";

// JobStats คือผลลัพธ์ของ computeJobStats — แยกเป็น pure function ต่างหากจาก
// job-stats.tsx (component) ด้วยเหตุผลเดียวกับ log-filters.ts: ทดสอบ logic การคำนวณ
// ได้โดยไม่ต้อง render component จริง
export interface JobStats {
  total: number;
  succeeded: number;
  failed: number;
  // successRatePercent เป็น null ถ้ายังไม่มี job ที่ "จบแล้ว" เลยสักตัว (แบ่งเลขไม่ได้ —
  // 0/0 ไม่มีความหมาย ต่างจาก success rate 0% ที่แปลว่า "จบแล้วแต่ล้มเหลวหมดทุกตัว")
  successRatePercent: number | null;
  todayCount: number;
  // recentFailures ตัดไว้แค่ 5 ตัวล่าสุด (เรียงใหม่สุดก่อน) พอสำหรับ "เห็นภาพคร่าวๆ" บน
  // stat card ไม่ใช่ list เต็ม (ดูรายละเอียดทั้งหมดได้จากตารางด้านล่างอยู่แล้ว)
  recentFailures: Job[];
}

const TERMINAL_STATUSES: Job["status"][] = ["succeeded", "failed", "cancelled"];

export function computeJobStats(jobs: Job[]): JobStats {
  const succeeded = jobs.filter((j) => j.status === "succeeded").length;
  const failed = jobs.filter((j) => j.status === "failed").length;
  const terminalCount = jobs.filter((j) => TERMINAL_STATUSES.includes(j.status)).length;

  const todayLabel = new Date().toDateString();
  const todayCount = jobs.filter((j) => new Date(j.created_at).toDateString() === todayLabel).length;

  const recentFailures = jobs
    .filter((j) => j.status === "failed")
    .slice()
    .sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
    .slice(0, 5);

  return {
    total: jobs.length,
    succeeded,
    failed,
    successRatePercent: terminalCount === 0 ? null : Math.round((succeeded / terminalCount) * 100),
    todayCount,
    recentFailures,
  };
}
