import { z } from "zod";

import type { ConfigField } from "../types";

// buildConfigSchema สร้าง zod schema แบบ "dynamic" จาก config_schema ที่ backend ประกาศไว้
// (bot.ConfigSchema() ฝั่ง Go — ดู internal/bot/bot.go) นี่คือหัวใจของ dynamic form ตาม
// roadmap ข้อ 1.3: field ไม่ได้ hard-code ไว้ตายตัวในโค้ด frontend เลยแม้แต่ตัวเดียว แต่
// สร้าง schema (และ form ที่ bot-form.tsx render ตาม schema นี้) ขึ้นมาใหม่ทุกครั้งตาม
// bot ที่ user เลือก — เพิ่ม bot ชนิดใหม่ที่มี config field ต่างไปได้ที่ backend อย่างเดียว
// โดยไม่ต้องแก้โค้ด frontend เลยสักบรรทัด ตรงกับหลักการ "pluggable" ของทั้งระบบ BOP
export function buildConfigSchema(fields: ConfigField[]) {
  const shape: Record<string, z.ZodTypeAny> = {};

  for (const field of fields) {
    // เริ่มจาก string schema พื้นฐานก่อนเสมอ แล้วค่อยเติมกฎเพิ่มตาม field.type — ตอนนี้
    // รองรับแค่ "url" (validate รูปแบบ URL ด้วย) กับ "text" (string ธรรมดา) เพิ่ม
    // ConfigFieldType ใหม่ในอนาคตแค่เพิ่ม case ตรงนี้ที่เดียว
    let fieldSchema = z.string();
    if (field.type === "url") {
      fieldSchema = fieldSchema.url("ต้องเป็น URL ที่ถูกต้อง");
    }

    if (field.required) {
      shape[field.key] = fieldSchema.min(1, `กรอก ${field.label}`);
    } else {
      // .optional() เปลี่ยน type ให้ยอมรับค่า undefined ได้ด้วย (user ไม่กรอก field
      // นี้ก็ผ่าน validate) — ต้องเรียกเป็นขั้นตอนสุดท้ายเสมอ เพราะเปลี่ยน type จาก
      // ZodString เป็น ZodOptional<ZodString> ซึ่งไม่มี .min()/.url() ให้เรียกต่อแล้ว
      shape[field.key] = fieldSchema.optional();
    }
  }

  return z.object(shape);
}
