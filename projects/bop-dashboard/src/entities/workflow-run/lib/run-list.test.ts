import { describe, expect, it } from "vitest";

import type { WorkflowRunSummary } from "../model/types";
import { flattenRunPages, nextRunPageToken, runCountPath, runListPath } from "./run-list";

function run(workflow_id: string, run_id = "r"): WorkflowRunSummary {
  return { workflow_id, run_id, workflow_name: "Execute", status: "Running", started_at: "2026-11-21T03:46:00Z" };
}

describe("runListPath", () => {
  it("ไม่มีเงื่อนไข = path เปล่า", () => {
    expect(runListPath({})).toBe("/workflow-runs");
  });

  it("ส่ง status, page_size และ page_token ไปเป็น query string (escape token ให้)", () => {
    expect(runListPath({ status: "failed", pageSize: 5 }, "ab+/=")).toBe(
      "/workflow-runs?status=failed&page_size=5&page_token=ab%2B%2F%3D",
    );
  });

  it("status ว่างคือทุกสถานะ ไม่ส่ง param", () => {
    expect(runListPath({ status: "" })).toBe("/workflow-runs");
  });
});

describe("runCountPath", () => {
  it("ไม่ระบุสถานะ = นับทุกสถานะ ไม่ส่ง param", () => {
    expect(runCountPath()).toBe("/workflow-runs/count");
    expect(runCountPath("")).toBe("/workflow-runs/count");
  });

  it("ส่ง status เป็น query string", () => {
    expect(runCountPath("running")).toBe("/workflow-runs/count?status=running");
  });
});

describe("nextRunPageToken", () => {
  it("token ว่างแปลว่าไม่มีหน้าถัดไป", () => {
    expect(nextRunPageToken({ runs: [], next_page_token: "" })).toBeUndefined();
    expect(nextRunPageToken({ runs: [], next_page_token: "abc" })).toBe("abc");
  });
});

describe("flattenRunPages", () => {
  it("ต่อทุกหน้าตามลำดับ และตัด run ที่ซ้ำข้ามหน้าทิ้ง (เก็บตัวแรกที่เจอ)", () => {
    const pages = [
      { runs: [run("c"), run("b")], next_page_token: "t1" },
      // run ใหม่เข้ามาระหว่างโหลด ทำให้ "b" เลื่อนมาอยู่หน้าถัดไปด้วย
      { runs: [run("b"), run("a")], next_page_token: "" },
    ];
    expect(flattenRunPages(pages).map((r) => r.workflow_id)).toEqual(["c", "b", "a"]);
  });

  it("workflow ID เดียวกันแต่คนละ run ID ไม่ถือว่าซ้ำ", () => {
    const pages = [{ runs: [run("x", "r1"), run("x", "r2")], next_page_token: "" }];
    expect(flattenRunPages(pages)).toHaveLength(2);
  });
});
