import { describe, expect, it } from "vitest";

import { isFailedRunStatus, normalizeRunStatus } from "./run-status";

describe("normalizeRunStatus", () => {
  it("รับชื่อสั้นที่ backend ส่งมาจริง", () => {
    expect(normalizeRunStatus("Running")).toBe("running");
    expect(normalizeRunStatus("Completed")).toBe("completed");
    expect(normalizeRunStatus("TimedOut")).toBe("timed_out");
    expect(normalizeRunStatus("ContinuedAsNew")).toBe("continued_as_new");
  });

  it("รับชื่อ enum ยาวแบบเดิมได้ด้วย", () => {
    expect(normalizeRunStatus("WORKFLOW_EXECUTION_STATUS_FAILED")).toBe("failed");
    expect(normalizeRunStatus("WORKFLOW_EXECUTION_STATUS_TIMED_OUT")).toBe("timed_out");
  });

  it("ค่าที่ไม่รู้จักเป็น unknown", () => {
    expect(normalizeRunStatus("Unspecified")).toBe("unknown");
  });
});

describe("isFailedRunStatus", () => {
  it("นับ failed, terminated และ timed_out เป็นล้มเหลว", () => {
    expect(["failed", "terminated", "timed_out", "completed", "canceled"].map((s) => isFailedRunStatus(normalizeRunStatus(s)))).toEqual([
      true,
      true,
      true,
      false,
      false,
    ]);
  });
});
