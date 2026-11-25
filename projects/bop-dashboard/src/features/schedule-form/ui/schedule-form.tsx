"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

// รายชื่อ bot + config schema มาจาก entities/bot ซึ่งเป็นชั้นล่างกว่า features ตามกฎของ
// FSD จึง import ได้ตามปกติ (เดิมตอนเป็น features/bots ต้องถือเป็นข้อยกเว้น cross-feature)
import { useBots } from "@/entities/bot";

import { useCreateSchedule } from "../api/use-create-schedule";
import { useUpdateSchedule } from "../api/use-update-schedule";
import type { OverlapPolicy, ScheduleDetail, ScheduleStep } from "@/entities/schedule";

import { initialCatchupWindow, validateCatchupWindow } from "../lib/catchup-window";
import { EMPTY_STEP, validateSteps } from "../lib/steps";
import { StepsEditor } from "./steps-editor";

// shellSchema คุม field ที่เป็น string ธรรมดาทั้งหมด (id, workflow_name, overlap) ผ่าน
// react-hook-form + zod ตามปกติ — ส่วน "ตารางเวลา" (interval/cron toggle) กับ "steps"
// (dynamic ตาม bot ที่เลือกต่อ step) ตั้งใจ "ไม่" ผ่าน zod เพราะ shape ของมัน
// เปลี่ยนไปตาม bot ที่เลือกแบบ runtime-dynamic (เหมือนที่ bot-form.tsx เจอ แต่ที่นี่ซับซ้อน
// กว่าอีกชั้นเพราะเป็น "array ของ dynamic shape" ไม่ใช่ dynamic shape เดียว) validate
// ส่วนนี้ด้วยมือใน onSubmit แทน — ง่ายกว่าประกอบ zod schema แบบ dynamic ซ้อน array
// มากสำหรับระบบที่ตอนนี้มี bot แค่ตัวเดียว ไม่คุ้มความซับซ้อนที่เพิ่มขึ้น
const shellSchema = z.object({
  id: z.string().min(1, "กรอก ID"),
  workflow_name: z.string().min(1, "กรอกชื่อ workflow"),
  overlap: z.string(),
});
type ShellValues = z.infer<typeof shellSchema>;

const OVERLAP_OPTIONS: OverlapPolicy[] = [
  "skip",
  "buffer_one",
  "buffer_all",
  "cancel_other",
  "terminate_other",
  "allow_all",
];

interface ScheduleFormProps {
  // initial มีค่า = edit mode (PUT ไปที่ /schedules/{initial.id}) ไม่มีค่า = create mode
  // (POST /schedules) — ฟอร์มเดียวใช้ได้ทั้งสองโหมด ต่างกันแค่ default value กับปลายทาง
  // ที่ submit ไปหา
  initial?: ScheduleDetail;
}

export function ScheduleForm({ initial }: ScheduleFormProps) {
  const router = useRouter();
  const isEdit = initial !== undefined;

  const { data: bots } = useBots();

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<ShellValues>({
    resolver: zodResolver(shellSchema),
    defaultValues: {
      id: initial?.id ?? "",
      workflow_name: initial?.workflow_name ?? "",
      overlap: initial?.overlap ?? "skip",
    },
  });

  const [specType, setSpecType] = useState<"interval" | "cron">(initial?.cron_expression ? "cron" : "interval");
  const [intervalSeconds, setIntervalSeconds] = useState(String(initial?.interval_seconds ?? 60));
  const [cronExpression, setCronExpression] = useState(initial?.cron_expression ?? "");
  const [catchupWindow, setCatchupWindow] = useState(initialCatchupWindow(initial?.catchup_window_seconds));

  const [steps, setSteps] = useState<ScheduleStep[]>(
    initial?.steps && initial.steps.length > 0 ? initial.steps : [EMPTY_STEP],
  );
  const [formError, setFormError] = useState<string | null>(null);

  // ต้องเรียกทั้งสอง hook เสมอไม่ว่าจะเป็นโหมดไหน (Rules of Hooks ห้ามเรียก hook แบบมี
  // เงื่อนไข) แล้วค่อยเลือกว่าจะใช้ตัวไหนตอน submit จริง — updateSchedule ผูกกับ
  // initial?.id ?? "" ไปก่อน (ไม่ถูกใช้จริงถ้าเป็น create mode)
  const createSchedule = useCreateSchedule();
  const updateSchedule = useUpdateSchedule(initial?.id ?? "");
  const mutation = isEdit ? updateSchedule : createSchedule;

  const onSubmit = handleSubmit((shell) => {
    setFormError(null);

    const stepsError = validateSteps(steps, bots);
    if (stepsError) {
      setFormError(stepsError);
      return;
    }

    if (specType === "interval") {
      const seconds = Number(intervalSeconds);
      if (!Number.isFinite(seconds) || seconds <= 0) {
        setFormError("interval ต้องเป็นตัวเลขมากกว่า 0 (หน่วยวินาที)");
        return;
      }
    } else if (!cronExpression.trim()) {
      setFormError("กรอก cron expression");
      return;
    }

    const catchupWindowError = validateCatchupWindow(catchupWindow);
    if (catchupWindowError) {
      setFormError(catchupWindowError);
      return;
    }
    const catchup_window_seconds = Number(catchupWindow);

    const spec =
      specType === "interval" ? { interval_seconds: Number(intervalSeconds) } : { cron_expression: cronExpression.trim() };
    const overlap = shell.overlap as OverlapPolicy;

    if (isEdit && initial) {
      updateSchedule.mutate(
        { workflow_name: shell.workflow_name, steps, spec, overlap, catchup_window_seconds },
        { onSuccess: () => router.push(`/schedules/${initial.id}`) },
      );
    } else {
      createSchedule.mutate(
        { id: shell.id, workflow_name: shell.workflow_name, steps, spec, overlap, catchup_window_seconds },
        { onSuccess: (created) => router.push(`/schedules/${created.id}`) },
      );
    }
  });

  return (
    <form onSubmit={onSubmit} className="flex max-w-xl flex-col gap-6">
      <div>
        <label htmlFor="id" className="block text-sm font-medium">
          ID
        </label>
        <input
          id="id"
          {...register("id")}
          disabled={isEdit}
          className="field mt-1 w-full px-3 py-2 disabled:bg-gray-100"
        />
        {errors.id && <p className="mt-1 text-sm text-red-600">{errors.id.message}</p>}
        {isEdit && (
          <p className="mt-1 text-xs text-gray-500">เปลี่ยน ID ไม่ได้ — ถ้าต้องการเปลี่ยนชื่อ ให้ลบตัวเก่าแล้วสร้างใหม่</p>
        )}
      </div>

      <div>
        <label htmlFor="workflow_name" className="block text-sm font-medium">
          Workflow Name
        </label>
        <input
          id="workflow_name"
          {...register("workflow_name")}
          className="field mt-1 w-full px-3 py-2"
        />
        {errors.workflow_name && <p className="mt-1 text-sm text-red-600">{errors.workflow_name.message}</p>}
      </div>

      <fieldset className="flex flex-col gap-3 rounded-lg border border-gray-200 p-4">
        <legend className="px-1 text-sm font-medium">ตารางเวลา</legend>
        <div className="flex gap-4 text-sm">
          <label className="flex items-center gap-2">
            <input type="radio" checked={specType === "interval"} onChange={() => setSpecType("interval")} />
            ทุก N วินาที
          </label>
          <label className="flex items-center gap-2">
            <input type="radio" checked={specType === "cron"} onChange={() => setSpecType("cron")} />
            Cron expression
          </label>
        </div>
        {specType === "interval" ? (
          <input
            type="number"
            min={1}
            value={intervalSeconds}
            onChange={(event) => setIntervalSeconds(event.target.value)}
            className="field w-40 px-3 py-2"
          />
        ) : (
          <input
            type="text"
            placeholder="0 9 * * MON-FRI"
            value={cronExpression}
            onChange={(event) => setCronExpression(event.target.value)}
            className="field w-full px-3 py-2 font-mono"
          />
        )}
      </fieldset>

      <div>
        <label htmlFor="overlap" className="block text-sm font-medium">
          Overlap Policy
        </label>
        <select
          id="overlap"
          {...register("overlap")}
          className="field mt-1 w-full px-3 py-2"
        >
          {OVERLAP_OPTIONS.map((o) => (
            <option key={o} value={o}>
              {o}
            </option>
          ))}
        </select>
      </div>

      <div>
        <label htmlFor="catchup_window_seconds" className="block text-sm font-medium">
          Catch-up Window (วินาที)
        </label>
        <input
          id="catchup_window_seconds"
          type="number"
          min={10}
          max={31536000}
          value={catchupWindow}
          onChange={(event) => setCatchupWindow(event.target.value)}
          className="field mt-1 w-40 px-3 py-2"
        />
        <p className="mt-1 text-xs text-gray-500">
          Missed Run ที่เก่ากว่านี้ถูกข้ามเมื่อระบบกลับมา (ค่าเริ่มต้น 3600 = 1 ชั่วโมง ต่ำสุด 10 สูงสุด 31536000 = 1 ปี)
        </p>
      </div>

      <StepsEditor steps={steps} onChange={setSteps} bots={bots} />

      {formError && <p className="text-sm text-red-600">{formError}</p>}
      {mutation.isError && <p className="text-sm text-red-600">{mutation.error.message}</p>}

      <button
        type="submit"
        disabled={mutation.isPending}
        className="btn-primary self-start px-4 py-2 disabled:opacity-50"
      >
        {mutation.isPending ? "กำลังบันทึก..." : isEdit ? "บันทึกการแก้ไข" : "สร้าง schedule"}
      </button>
    </form>
  );
}
