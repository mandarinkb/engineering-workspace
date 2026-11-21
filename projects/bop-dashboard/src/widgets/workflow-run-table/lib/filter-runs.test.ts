import { describe, expect, it } from "vitest";

import type { WorkflowRunSummary } from "@/entities/workflow-run";

import { searchRuns } from "./filter-runs";

function run(workflow_id: string): WorkflowRunSummary {
  return { workflow_id, run_id: "r", workflow_name: "Execute", status: "Completed", started_at: "2026-11-21T03:46:00Z" };
}

const RUNS = [
  run("adhoc-xyz"),
  run("check-example-com-2026-11-21T03:47:00Z"),
  run("check-example-com-2026-11-21T03:46:00Z"),
];

describe("searchRuns", () => {
  it("ค้นหาว่าง = คืนทั้งหมดตามลำดับเดิม", () => {
    expect(searchRuns(RUNS, "  ")).toEqual(RUNS);
  });

  it("ค้นหาชื่อ schedule ใน workflow_id แบบไม่สนตัวพิมพ์ คงลำดับเดิม", () => {
    expect(searchRuns(RUNS, "  CHECK-example ").map((r) => r.workflow_id)).toEqual([
      "check-example-com-2026-11-21T03:47:00Z",
      "check-example-com-2026-11-21T03:46:00Z",
    ]);
  });
});
