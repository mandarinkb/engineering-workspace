import { describe, expect, it } from "vitest";

import type { WorkflowRunSummary } from "@/entities/workflow-run";

import { filterRuns } from "./filter-runs";

function run(workflow_id: string, status: string, started_at: string): WorkflowRunSummary {
  return { workflow_id, run_id: "r", workflow_name: "Execute", status, started_at };
}

const RUNS = [
  run("check-example-com-2026-11-21T03:46:00Z", "Completed", "2026-11-21T03:46:00Z"),
  run("check-example-com-2026-11-21T03:47:00Z", "Failed", "2026-11-21T03:47:00Z"),
  run("demo-fail-abc", "TimedOut", "2026-11-21T03:40:00Z"),
  run("adhoc-xyz", "Running", "2026-11-21T03:48:00Z"),
];

describe("filterRuns", () => {
  it("ไม่มี filter = คืนทั้งหมด เรียงใหม่สุดก่อน", () => {
    expect(filterRuns(RUNS, { status: "", search: "" }).map((r) => r.workflow_id)).toEqual([
      "adhoc-xyz",
      "check-example-com-2026-11-21T03:47:00Z",
      "check-example-com-2026-11-21T03:46:00Z",
      "demo-fail-abc",
    ]);
  });

  it("status failed รวม timed out ด้วย", () => {
    expect(filterRuns(RUNS, { status: "failed", search: "" }).map((r) => r.workflow_id)).toEqual([
      "check-example-com-2026-11-21T03:47:00Z",
      "demo-fail-abc",
    ]);
  });

  it("ค้นหาชื่อ schedule ใน workflow_id แบบไม่สนตัวพิมพ์ ใช้ร่วมกับ status ได้", () => {
    expect(filterRuns(RUNS, { status: "completed", search: "  CHECK-example " }).map((r) => r.workflow_id)).toEqual([
      "check-example-com-2026-11-21T03:46:00Z",
    ]);
  });
});
