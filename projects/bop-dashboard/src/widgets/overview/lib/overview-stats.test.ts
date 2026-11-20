import { describe, expect, it } from "vitest";

import type { Job } from "@/entities/job";
import type { ScheduleSummary } from "@/entities/schedule";
import type { WorkflowRunSummary } from "@/entities/workflow-run";

import { computeOverviewStats } from "./overview-stats";

const NOW = new Date("2026-11-20T12:00:00");
const DAY = 24 * 60 * 60 * 1000;
const ago = (ms: number) => new Date(NOW.getTime() - ms).toISOString();
const later = (ms: number) => new Date(NOW.getTime() + ms).toISOString();

function job(overrides: Partial<Job>): Job {
  return { id: "j", bot_name: "bot", config: {}, status: "succeeded", created_at: NOW.toISOString(), ...overrides };
}

function schedule(overrides: Partial<ScheduleSummary>): ScheduleSummary {
  return { id: "s", workflow_name: "wf", paused: false, ...overrides };
}

function run(overrides: Partial<WorkflowRunSummary>): WorkflowRunSummary {
  return {
    workflow_id: "w",
    run_id: "r",
    workflow_name: "wf",
    status: "Completed",
    started_at: NOW.toISOString(),
    ...overrides,
  };
}

describe("computeOverviewStats", () => {
  it("นับ job ที่กำลังรัน และ job ที่ fail เฉพาะของวันนี้", () => {
    const stats = computeOverviewStats(
      [
        job({ status: "running" }),
        job({ status: "running" }),
        job({ status: "failed" }),
        job({ status: "failed", created_at: ago(2 * DAY) }), // fail แต่ไม่ใช่วันนี้
      ],
      [],
      [],
      NOW,
    );
    expect(stats.runningJobs).toBe(2);
    expect(stats.failedJobsToday).toBe(1);
  });

  it("success rate 7 วันนับเฉพาะ job ที่จบแล้วในช่วง 7 วัน", () => {
    const stats = computeOverviewStats(
      [
        job({ status: "succeeded", created_at: ago(1 * DAY) }),
        job({ status: "succeeded", created_at: ago(3 * DAY) }),
        job({ status: "failed", created_at: ago(6 * DAY) }),
        job({ status: "running" }), // ยังไม่จบ ไม่นับ
        job({ status: "failed", created_at: ago(10 * DAY) }), // เก่ากว่า 7 วัน ไม่นับ
      ],
      [],
      [],
      NOW,
    );
    expect(stats.successRate7dPercent).toBe(67);
  });

  it("success rate เป็น null ถ้าไม่มี job ที่จบใน 7 วัน", () => {
    const stats = computeOverviewStats([job({ status: "pending" })], [], [], NOW);
    expect(stats.successRate7dPercent).toBeNull();
  });

  it("schedule ถัดไปเรียงตามเวลา ข้าม schedule ที่ pause และเวลาที่ผ่านไปแล้ว", () => {
    const stats = computeOverviewStats(
      [],
      [
        schedule({ id: "late", next_action_times: [later(3 * DAY)] }),
        schedule({ id: "soon", next_action_times: [ago(1000), later(60_000)] }),
        schedule({ id: "paused", paused: true, next_action_times: [later(1000)] }),
        schedule({ id: "none" }),
      ],
      [],
      NOW,
    );
    expect(stats.activeSchedules).toBe(3);
    expect(stats.upcomingSchedules.map((u) => u.schedule.id)).toEqual(["soon", "late"]);
    expect(stats.upcomingSchedules[0].nextRunAt).toBe(later(60_000));
  });

  it("workflow run ที่ fail ล่าสุดรวม terminated/timed out (ชื่อ enum ยาวแบบเก่าก็นับ) เรียงใหม่สุดก่อน จำกัด 5 ตัว", () => {
    const failed = Array.from({ length: 6 }, (_, i) =>
      run({ workflow_id: `f${i}`, status: "Failed", started_at: ago(i * 60_000) }),
    );
    const stats = computeOverviewStats(
      [],
      [],
      [
        ...failed,
        run({ workflow_id: "t", status: "WORKFLOW_EXECUTION_STATUS_TIMED_OUT", started_at: ago(30_000) }),
        run({ workflow_id: "ok" }),
        run({ workflow_id: "live", status: "Running" }),
      ],
      NOW,
    );
    expect(stats.runningRuns).toBe(1);
    expect(stats.recentFailedRuns.map((r) => r.workflow_id)).toEqual(["f0", "t", "f1", "f2", "f3"]);
  });
});
