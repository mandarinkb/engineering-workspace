import { describe, expect, it } from "vitest";

import type { Job } from "../types";
import { computeJobStats } from "./job-stats";

function makeJob(overrides: Partial<Job>): Job {
  return {
    id: "job-1",
    bot_name: "http-status-checker",
    config: {},
    status: "succeeded",
    created_at: new Date().toISOString(),
    ...overrides,
  };
}

describe("computeJobStats", () => {
  it("successRatePercent เป็น null ถ้ายังไม่มี job ที่จบเลย", () => {
    const stats = computeJobStats([makeJob({ status: "pending" }), makeJob({ status: "running" })]);
    expect(stats.successRatePercent).toBeNull();
  });

  it("คำนวณ success rate จาก job ที่จบแล้วเท่านั้น (ไม่รวม pending/running)", () => {
    const jobs = [
      makeJob({ status: "succeeded" }),
      makeJob({ status: "succeeded" }),
      makeJob({ status: "failed" }),
      makeJob({ status: "running" }), // ไม่นับ เพราะยังไม่จบ
    ];
    const stats = computeJobStats(jobs);
    expect(stats.successRatePercent).toBe(67); // 2/3 ปัดเป็น 67%
  });

  it("นับ job วันนี้ถูกต้อง แยกจาก job เก่า", () => {
    const jobs = [
      makeJob({ created_at: new Date().toISOString() }),
      makeJob({ created_at: "2020-01-01T00:00:00Z" }),
    ];
    expect(computeJobStats(jobs).todayCount).toBe(1);
  });

  it("recentFailures คืนเฉพาะ job ที่ failed เรียงล่าสุดก่อน ตัดที่ 5 ตัว", () => {
    const jobs = Array.from({ length: 7 }, (_, i) =>
      makeJob({ id: `job-${i}`, status: "failed", created_at: new Date(2026, 0, i + 1).toISOString() }),
    );
    const stats = computeJobStats(jobs);
    expect(stats.recentFailures).toHaveLength(5);
    expect(stats.recentFailures[0].id).toBe("job-6"); // วันที่ล่าสุดสุด (2026-01-07) ต้องมาก่อน
  });
});
