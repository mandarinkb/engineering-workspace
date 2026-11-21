import { describe, expect, it } from "vitest";

import type { Bot } from "@/entities/bot";

import { dispatchModeOf, validateSteps } from "./steps";

const BOTS: Bot[] = [
  { name: "http-status-checker", config_schema: [{ key: "url", label: "URL", type: "url", required: true }] },
];

describe("validateSteps", () => {
  it("ต้องมีอย่างน้อย 1 step", () => {
    expect(validateSteps([], BOTS)).toBe("ต้องมีอย่างน้อย 1 step");
  });

  it("ทุก step ต้องเลือก bot", () => {
    expect(validateSteps([{ bot_name: "", config: {} }], BOTS)).toBe("ทุก step ต้องเลือก bot");
  });

  it("ต้องกรอก config field ที่ required", () => {
    expect(validateSteps([{ bot_name: "http-status-checker", config: {} }], BOTS)).toBe(
      'step ที่ใช้ bot "http-status-checker" ต้องกรอก "URL"',
    );
  });

  it("Kubernetes Job ต้องมี image", () => {
    const step = { bot_name: "http-status-checker", config: { url: "https://example.com" }, k8s_job: { image: " " } };
    expect(validateSteps([step], BOTS)).toContain('ต้องกรอก "image"');
  });

  it("จำนวนครั้งที่ลองสูงสุดต้องเป็นจำนวนเต็มตั้งแต่ 1", () => {
    for (const max_attempts of [0, -1, 1.5]) {
      const step = { bot_name: "http-status-checker", config: { url: "https://example.com" }, max_attempts };
      expect(validateSteps([step], BOTS)).toBe(
        'step ที่ใช้ bot "http-status-checker" ต้องตั้ง "จำนวนครั้งที่ลองสูงสุด" เป็นจำนวนเต็มตั้งแต่ 1 ขึ้นไป',
      );
    }
  });

  it("จำนวนครั้งที่ลองสูงสุดที่ถูกต้องหรือเว้นว่างผ่าน", () => {
    const base = { bot_name: "http-status-checker", config: { url: "https://example.com" } };
    expect(validateSteps([{ ...base, max_attempts: 5 }], BOTS)).toBeNull();
    expect(validateSteps([{ ...base, max_attempts: undefined }], BOTS)).toBeNull();
  });

  it("ผ่านเมื่อครบ", () => {
    expect(validateSteps([{ bot_name: "http-status-checker", config: { url: "https://example.com" } }], BOTS)).toBeNull();
  });
});

describe("dispatchModeOf", () => {
  it("แยกโหมดจาก field ที่มีใน step", () => {
    expect(dispatchModeOf({ bot_name: "b", config: {} })).toBe("local");
    expect(dispatchModeOf({ bot_name: "b", config: {}, task_queue: "q" })).toBe("remote_task_queue");
    expect(dispatchModeOf({ bot_name: "b", config: {}, k8s_job: { image: "i" } })).toBe("k8s_job");
  });
});
