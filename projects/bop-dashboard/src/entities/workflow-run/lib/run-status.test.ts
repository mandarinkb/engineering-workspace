import { describe, expect, it } from "vitest";

import { isFailedRunStatus, isRunInProgress, normalizeRunStatus } from "./run-status";

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

describe("isRunInProgress", () => {
  it("running ต้องดึงข้อมูลซ้ำ ทั้งชื่อสั้นและชื่อ enum ยาว", () => {
    expect(isRunInProgress("Running")).toBe(true);
    expect(isRunInProgress("WORKFLOW_EXECUTION_STATUS_RUNNING")).toBe(true);
  });

  it("สถานะที่จบแล้วทุกแบบไม่ต้องดึงซ้ำ", () => {
    for (const status of ["Completed", "Failed", "Canceled", "Terminated", "TimedOut", "ContinuedAsNew"]) {
      expect(isRunInProgress(status), status).toBe(false);
    }
  });

  it("สถานะที่ไม่รู้จักหรือยังไม่มีข้อมูลไม่ดึงซ้ำ", () => {
    expect(isRunInProgress("Unspecified")).toBe(false);
    expect(isRunInProgress(undefined)).toBe(false);
  });
});
