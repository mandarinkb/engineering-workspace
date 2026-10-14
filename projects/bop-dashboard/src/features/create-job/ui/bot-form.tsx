"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { useCreateJob } from "../api/use-create-job";
import { buildConfigSchema } from "../schemas/bot-config.schema";
import type { Bot } from "../types";

interface BotFormProps {
  bot: Bot;
  onCreated?: (jobId: string) => void;
}

// BotForm คือ dynamic form ตัวจริงตาม roadmap ข้อ 1.3 — field ที่ render (ผ่าน
// bot.config_schema.map(...) ด้านล่าง) เปลี่ยนไปตาม bot ที่ได้รับมาเป็น prop ทั้งหมด
// ไม่มี field ไหน hard-code ไว้ในไฟล์นี้เลย
//
// สำคัญมาก: parent component (bot-list.tsx) ต้องใส่ key={bot.name} ให้ <BotForm> เสมอ —
// key ที่เปลี่ยนไปตาม bot บังคับให้ React "สร้าง component instance ใหม่ทั้งหมด" ทุกครั้ง
// ที่ user เลือก bot ตัวอื่น (unmount ตัวเก่า mount ตัวใหม่) แทนที่จะพยายาม reuse
// instance เดิม — ถ้าไม่ใส่ key ให้ถูกต้อง React จะ reuse form state เก่าทิ้งไว้ปนกับ
// schema ใหม่ (จาก useForm ที่ resolver เปลี่ยนไปแล้วแต่ field ค่าเก่ายังค้างอยู่) ทำให้
// validate ผิดเพี้ยนได้
export function BotForm({ bot, onCreated }: BotFormProps) {
  const createJob = useCreateJob(bot.name);
  const schema = buildConfigSchema(bot.config_schema);

  // ไม่ระบุ generic <T> ให้ useForm ตรงๆ ตั้งใจ — ปล่อยให้ TypeScript infer type ของ
  // form values จาก resolver: zodResolver(schema) เอง เพราะ schema สร้างขึ้นแบบ dynamic
  // ตอน runtime (ตาม bot.config_schema ที่ได้จาก API) จึงไม่มีทางรู้ type ที่แน่นอนตอน
  // compile time ล่วงหน้าได้ — การระบุ generic เองตรงๆ (เช่น Record<string, string>)
  // จะขัดกับ type ที่ zodResolver(schema) อนุมานได้จริงจนกลายเป็น type error
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm({ resolver: zodResolver(schema) });

  const onSubmit = handleSubmit((values) => {
    // filter ค่าที่ undefined/ว่างเปล่าออกก่อนส่ง เพราะ bot.Config ฝั่ง Go เป็น
    // map[string]string ธรรมดา (ไม่รองรับ undefined) — field ที่ optional แล้วไม่ได้
    // กรอกไม่ควรส่ง key นั้นไปให้ backend เลยดีกว่าส่งเป็น string ว่างเปล่า
    // values ถูก infer เป็น {} (empty object type) เพราะ schema สร้างแบบ dynamic ตาม
    // เหตุผลเดียวกับที่ useForm ด้านบนไม่ระบุ generic — ต้อง assert type ตรงนี้เพราะรู้
    // อยู่แล้วจาก runtime ว่าทุก field ใน schema เป็น string/undefined ล้วนๆ (input ธรรมดา
    // ทั้งหมด ไม่มี field ชนิดอื่นในระบบนี้)
    const config: Record<string, string> = {};
    for (const [key, value] of Object.entries(values as Record<string, string | undefined>)) {
      if (value) config[key] = value;
    }
    createJob.mutate(config, {
      onSuccess: (job) => onCreated?.(job.id),
    });
  });

  return (
    <form onSubmit={onSubmit} className="flex flex-col gap-4">
      {bot.config_schema.map((field) => (
        <div key={field.key}>
          <label htmlFor={field.key} className="block text-sm font-medium">
            {field.label}
            {field.required && <span className="text-red-600"> *</span>}
          </label>
          {/* type="url" ให้เบราว์เซอร์ validate รูปแบบ URL เบื้องต้นให้ฟรีๆ ก่อนแม้แต่
              zod จะทำงาน (แค่ hint ระดับ UX ไม่ใช่ validation ตัวจริง — zod ยังคง
              ตรวจสอบซ้ำอีกชั้นเสมอตอน submit เพราะห้ามเชื่อฝั่ง client อย่างเดียว) */}
          <input
            id={field.key}
            type={field.type === "url" ? "url" : "text"}
            {...register(field.key)}
            className="mt-1 w-full rounded border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none"
          />
          {errors[field.key] && (
            <p className="mt-1 text-sm text-red-600">{String(errors[field.key]?.message)}</p>
          )}
        </div>
      ))}

      {createJob.isError && <p className="text-sm text-red-600">{createJob.error.message}</p>}

      <button
        type="submit"
        disabled={createJob.isPending}
        className="rounded bg-blue-600 px-4 py-2 font-medium text-white hover:bg-blue-700 disabled:opacity-50"
      >
        {createJob.isPending ? "กำลังสั่งรัน..." : `รัน ${bot.name}`}
      </button>
    </form>
  );
}
