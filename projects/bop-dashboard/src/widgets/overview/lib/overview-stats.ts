import type { Job } from "@/entities/job";
import type { ScheduleSummary } from "@/entities/schedule";

// overview-stats รวมตัวเลขจาก job และ schedule มาไว้หน้าเดียว (ตัวเลขของ workflow run นับและ
// กรองฝั่ง server แล้ว ดู ui/overview.tsx) — อยู่ใน widget
// (ไม่ใช่ entities) เพราะต้องรู้จักหลาย entity พร้อมกัน ซึ่ง entity หนึ่งห้าม import อีก entity
// เป็น pure function (รับ now เข้ามา) เพื่อให้ test ได้โดยไม่ต้อง mock เวลา

const SEVEN_DAYS_MS = 7 * 24 * 60 * 60 * 1000;
const LIST_LIMIT = 5;

export interface UpcomingSchedule {
  schedule: ScheduleSummary;
  nextRunAt: string;
}

export interface OverviewStats {
  runningJobs: number;
  failedJobsToday: number;
  // success rate ของ job ที่จบแล้ว (succeeded/failed/cancelled) ที่สร้างใน 7 วันล่าสุด
  // null = ยังไม่มี job ที่จบในช่วงนั้นเลย
  successRate7dPercent: number | null;
  activeSchedules: number;
  upcomingSchedules: UpcomingSchedule[];
}

export function computeOverviewStats(
  jobs: Job[],
  schedules: ScheduleSummary[],
  now: Date,
): OverviewStats {
  const today = now.toDateString();
  const weekAgo = now.getTime() - SEVEN_DAYS_MS;

  const lastWeekTerminal = jobs.filter(
    (j) => j.status !== "pending" && j.status !== "running" && new Date(j.created_at).getTime() >= weekAgo,
  );
  const lastWeekSucceeded = lastWeekTerminal.filter((j) => j.status === "succeeded").length;

  const active = schedules.filter((s) => !s.paused);
  const upcomingSchedules = active
    .flatMap((schedule) => {
      // เอาเวลาถัดไปที่ยังไม่ผ่าน (backend คืนมาเรียงแล้ว แต่กันไว้เผื่อข้อมูลค้าง)
      const next = schedule.next_action_times?.find((t) => new Date(t).getTime() >= now.getTime());
      return next ? [{ schedule, nextRunAt: next }] : [];
    })
    .sort((a, b) => new Date(a.nextRunAt).getTime() - new Date(b.nextRunAt).getTime())
    .slice(0, LIST_LIMIT);

  return {
    runningJobs: jobs.filter((j) => j.status === "running").length,
    failedJobsToday: jobs.filter((j) => j.status === "failed" && new Date(j.created_at).toDateString() === today).length,
    successRate7dPercent:
      lastWeekTerminal.length === 0 ? null : Math.round((lastWeekSucceeded / lastWeekTerminal.length) * 100),
    activeSchedules: active.length,
    upcomingSchedules,
  };
}
