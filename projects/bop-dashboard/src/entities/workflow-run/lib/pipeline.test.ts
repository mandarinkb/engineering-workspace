import { describe, expect, it } from "vitest";

import type { StepResult } from "../types";
import { buildPipelineNodes } from "./pipeline";

const STEPS: StepResult[] = [
  { step_index: 0, bot_name: "http-status-checker", summary: "200 OK" },
  { step_index: 1, bot_name: "external-job", error: "exit code 1" },
];

describe("buildPipelineNodes", () => {
  it("derive สถานะ succeeded จาก step ที่ไม่มี error", () => {
    const nodes = buildPipelineNodes([STEPS[0]], "WORKFLOW_EXECUTION_STATUS_COMPLETED");
    expect(nodes[0].status).toBe("succeeded");
  });

  it("derive สถานะ failed จาก step ที่มี error", () => {
    const nodes = buildPipelineNodes([STEPS[1]], "WORKFLOW_EXECUTION_STATUS_FAILED");
    expect(nodes[0].status).toBe("failed");
  });

  it("ไม่เพิ่ม node พิเศษถ้า workflow จบไปแล้ว", () => {
    const nodes = buildPipelineNodes(STEPS, "WORKFLOW_EXECUTION_STATUS_COMPLETED");
    expect(nodes).toHaveLength(2);
  });

  it("เพิ่ม node พิเศษท้ายสุดถ้า workflow ยังรันอยู่", () => {
    const nodes = buildPipelineNodes(STEPS, "WORKFLOW_EXECUTION_STATUS_RUNNING");
    expect(nodes).toHaveLength(3);
    expect(nodes[2].status).toBe("running");
    expect(nodes[2].stepIndex).toBe(2);
  });

  it("ไม่มี step ไหนเลยแต่ workflow กำลังรันอยู่ ต้องเห็นแค่ node running ตัวเดียว", () => {
    const nodes = buildPipelineNodes([], "WORKFLOW_EXECUTION_STATUS_RUNNING");
    expect(nodes).toHaveLength(1);
    expect(nodes[0].status).toBe("running");
  });
});
