import { describe, expect, it } from "vitest";

import type { ConfigField } from "../types";
import { buildConfigSchema } from "./bot-config.schema";

// ทดสอบตาม roadmap ข้อ 1.9 ตรงๆ: "test form validation logic" — buildConfigSchema คือ
// จุดที่ dynamic form ทั้งหมดพึ่งพา (ดู bot-form.tsx) ถ้า schema ที่สร้างออกมาผิด ฟอร์ม
// ทั้งหน้าจะ validate ผิดตามไปด้วยโดยไม่รู้ตัว
describe("buildConfigSchema", () => {
  const urlField: ConfigField = { key: "url", label: "URL", type: "url", required: true };
  const optionalTextField: ConfigField = { key: "note", label: "Note", type: "text", required: false };

  it("ปฏิเสธ field required ที่ไม่ได้กรอก", () => {
    const schema = buildConfigSchema([urlField]);
    const result = schema.safeParse({ url: "" });
    expect(result.success).toBe(false);
  });

  it("ปฏิเสธค่าที่ไม่ใช่รูปแบบ URL สำหรับ field ชนิด url", () => {
    const schema = buildConfigSchema([urlField]);
    const result = schema.safeParse({ url: "not-a-url" });
    expect(result.success).toBe(false);
  });

  it("ยอมรับ URL ที่ถูกต้อง", () => {
    const schema = buildConfigSchema([urlField]);
    const result = schema.safeParse({ url: "https://example.com" });
    expect(result.success).toBe(true);
  });

  it("ยอมรับ field ที่ไม่ required แม้ไม่ได้กรอกเลย (undefined)", () => {
    const schema = buildConfigSchema([optionalTextField]);
    const result = schema.safeParse({});
    expect(result.success).toBe(true);
  });

  it("สร้าง field หลายตัวพร้อมกันได้ ผสม required/optional", () => {
    const schema = buildConfigSchema([urlField, optionalTextField]);
    const missingRequired = schema.safeParse({ note: "hello" });
    expect(missingRequired.success).toBe(false);

    const valid = schema.safeParse({ url: "https://example.com", note: "hello" });
    expect(valid.success).toBe(true);
  });
});
