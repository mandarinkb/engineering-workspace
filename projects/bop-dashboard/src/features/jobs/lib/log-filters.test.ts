import { describe, expect, it } from "vitest";

import type { LogLine } from "../types";
import { filterLogLines } from "./log-filters";

const LINES: LogLine[] = [
  { job_id: "1", time: "2026-01-01T00:00:00Z", level: "info", message: "starting job" },
  { job_id: "1", time: "2026-01-01T00:00:01Z", level: "warn", message: "retrying request" },
  { job_id: "1", time: "2026-01-01T00:00:02Z", level: "error", message: "request failed" },
];

describe("filterLogLines", () => {
  it("คืนทุกบรรทัดถ้า level เป็น all และไม่มี query", () => {
    expect(filterLogLines(LINES, "all", "")).toHaveLength(3);
  });

  it("กรองตาม level ได้", () => {
    const result = filterLogLines(LINES, "error", "");
    expect(result).toHaveLength(1);
    expect(result[0].message).toBe("request failed");
  });

  it("กรองตาม query แบบ case-insensitive substring", () => {
    const result = filterLogLines(LINES, "all", "REQUEST");
    expect(result.map((l) => l.message)).toEqual(["retrying request", "request failed"]);
  });

  it("ผสม level และ query พร้อมกันได้", () => {
    const result = filterLogLines(LINES, "warn", "retry");
    expect(result).toHaveLength(1);
    expect(result[0].level).toBe("warn");
  });

  it("query ที่เป็นช่องว่างล้วนๆ ถือว่าไม่กรอง", () => {
    expect(filterLogLines(LINES, "all", "   ")).toHaveLength(3);
  });
});
