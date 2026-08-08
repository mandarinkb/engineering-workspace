"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

// cross-feature import ที่ตั้งใจ (ผิดกติกาทั่วไปที่ PHASE1-DASHBOARD-TODO.md วางไว้ว่า
// features/X ห้าม import จาก features/Y ตรงๆ) — เหตุผลเดียวกับที่ NavBar (ดู
// src/components/layout/nav-bar.tsx) import จาก features/auth: "bot catalog" (รายชื่อ
// bot + config schema ของแต่ละตัว) เป็นข้อมูลอ้างอิงที่หลาย feature ต้องใช้จริง
// (features/bots ใช้ทำหน้า bot list, features/schedules ใช้เลือก bot ต่อ step) ไม่ใช่
// business logic เฉพาะของ bots feature — ถือเป็นข้อยกเว้นที่ยอมรับได้เพราะเป็น
// cross-cutting reference data ไม่ใช่การพันกันของ business logic สอง feature จริงๆ
import { useBots } from "@/features/bots/api/use-bots";
import type { ConfigField } from "@/features/bots/types";

import { useCreateSchedule } from "../api/use-create-schedule";
import { useUpdateSchedule } from "../api/use-update-schedule";
import type { DispatchMode, OverlapPolicy, ScheduleDetail, ScheduleStep } from "../types";

// dispatchModeOf derive DispatchMode จาก field ที่มีอยู่จริงใน step (ดู workflow.Step
// ฝั่ง Go: TaskQueue/K8sJob เป็น optional field ที่ mutually exclusive กันโดยธรรมชาติ —
// ไม่ได้เก็บเป็น field แยกตรงๆ ใน ScheduleStep เพราะ backend ไม่รู้จัก concept
// "dispatch mode" เลย รู้แค่ว่า step มี task_queue/k8s_job หรือไม่เท่านั้น)
function dispatchModeOf(step: ScheduleStep): DispatchMode {
  if (step.k8s_job) return "k8s_job";
  if (step.task_queue) return "remote_task_queue";
  return "local";
}

const DISPATCH_MODE_LABELS: Record<DispatchMode, string> = {
  local: "Local Bot",
  remote_task_queue: "Remote Task Queue",
  k8s_job: "Kubernetes Job",
};

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

  const [steps, setSteps] = useState<ScheduleStep[]>(
    initial?.steps && initial.steps.length > 0 ? initial.steps : [{ bot_name: "", config: {} }],
  );
  const [formError, setFormError] = useState<string | null>(null);

  // ต้องเรียกทั้งสอง hook เสมอไม่ว่าจะเป็นโหมดไหน (Rules of Hooks ห้ามเรียก hook แบบมี
  // เงื่อนไข) แล้วค่อยเลือกว่าจะใช้ตัวไหนตอน submit จริง — updateSchedule ผูกกับ
  // initial?.id ?? "" ไปก่อน (ไม่ถูกใช้จริงถ้าเป็น create mode)
  const createSchedule = useCreateSchedule();
  const updateSchedule = useUpdateSchedule(initial?.id ?? "");
  const mutation = isEdit ? updateSchedule : createSchedule;

  function addStep() {
    setSteps((prev) => [...prev, { bot_name: "", config: {} }]);
  }
  function removeStep(index: number) {
    setSteps((prev) => prev.filter((_, i) => i !== index));
  }
  function setStepBot(index: number, botName: string) {
    // เปลี่ยน bot ของ step นี้ต้องล้าง config เดิมทิ้งเสมอ เพราะ config field ของ bot
    // ใหม่อาจไม่ตรงกับของ bot เดิมเลย (คนละ key กัน) เก็บ config เก่าไว้จะสับสน
    setSteps((prev) => prev.map((s, i) => (i === index ? { bot_name: botName, config: {} } : s)));
  }
  function setStepConfigValue(index: number, key: string, value: string) {
    setSteps((prev) => prev.map((s, i) => (i === index ? { ...s, config: { ...s.config, [key]: value } } : s)));
  }

  // setStepDispatchMode สลับโหมดของ step — ต้องเคลียร์ field ของโหมดเดิมทิ้งเสมอ (ไม่ใช่
  // แค่เพิ่ม field ของโหมดใหม่เข้าไปเฉยๆ) เพราะ task_queue/k8s_job เป็น mutually
  // exclusive กันฝั่ง backend (ดู workflow.go: "ห้ามตั้งทั้ง TaskQueue และ K8sJob พร้อมกัน")
  // ตั้ง k8s_job เริ่มต้นเป็น {image: ""} ทันทีที่สลับมาโหมดนี้ (ไม่ใช่ undefined) เพื่อให้
  // ฟอร์มด้านล่าง render input ของ image ได้เลยโดยไม่ต้องเช็ค step.k8s_job ทุกจุด
  function setStepDispatchMode(index: number, mode: DispatchMode) {
    setSteps((prev) =>
      prev.map((s, i) => {
        if (i !== index) return s;
        // สร้าง base object จาก field ที่รู้จักตรงๆ (ไม่ destructure แล้วทิ้ง
        // task_queue/k8s_job เดิม) เพื่อเคลียร์ทั้งสอง field นี้ให้แน่ใจว่าไม่มีค่าเก่า
        // ของโหมดก่อนหน้าค้างอยู่ ไม่ว่าจะสลับมาโหมดไหนก็ตาม
        const base: ScheduleStep = { bot_name: s.bot_name, config: s.config };
        if (mode === "remote_task_queue") return { ...base, task_queue: "" };
        if (mode === "k8s_job") return { ...base, k8s_job: { image: "" } };
        return base;
      }),
    );
  }
  function setStepTaskQueue(index: number, value: string) {
    setSteps((prev) => prev.map((s, i) => (i === index ? { ...s, task_queue: value } : s)));
  }
  function setStepK8sJob(index: number, patch: Partial<NonNullable<ScheduleStep["k8s_job"]>>) {
    setSteps((prev) =>
      prev.map((s, i) => (i === index ? { ...s, k8s_job: { ...s.k8s_job, ...patch, image: patch.image ?? s.k8s_job?.image ?? "" } } : s)),
    );
  }

  function configSchemaFor(botName: string): ConfigField[] {
    return bots?.find((b) => b.name === botName)?.config_schema ?? [];
  }

  const onSubmit = handleSubmit((shell) => {
    setFormError(null);

    if (steps.length === 0) {
      setFormError("ต้องมีอย่างน้อย 1 step");
      return;
    }
    for (const step of steps) {
      if (!step.bot_name) {
        setFormError("ทุก step ต้องเลือก bot");
        return;
      }
      for (const field of configSchemaFor(step.bot_name)) {
        if (field.required && !step.config[field.key]) {
          setFormError(`step ที่ใช้ bot "${step.bot_name}" ต้องกรอก "${field.label}"`);
          return;
        }
      }
      // K8s Job hybrid execution ต้องมี image เสมอ (ตรงกับ validation ฝั่ง backend ใน
      // internal/workflow/k8sjob.go: RunKubernetesJob คืน bot.ConfigError ถ้าไม่มี) —
      // เช็คตรงนี้ก่อน submit เพื่อให้ user เห็น error เร็วกว่ารอ backend ตอบกลับมา
      if (step.k8s_job && !step.k8s_job.image.trim()) {
        setFormError(`step ที่ใช้ bot "${step.bot_name}" (Kubernetes Job) ต้องกรอก "image"`);
        return;
      }
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

    const spec =
      specType === "interval" ? { interval_seconds: Number(intervalSeconds) } : { cron_expression: cronExpression.trim() };
    const overlap = shell.overlap as OverlapPolicy;

    if (isEdit && initial) {
      updateSchedule.mutate(
        { workflow_name: shell.workflow_name, steps, spec, overlap },
        { onSuccess: () => router.push(`/schedules/${initial.id}`) },
      );
    } else {
      createSchedule.mutate(
        { id: shell.id, workflow_name: shell.workflow_name, steps, spec, overlap },
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
          className="mt-1 w-full rounded border border-gray-300 px-3 py-2 disabled:bg-gray-100"
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
          className="mt-1 w-full rounded border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none"
        />
        {errors.workflow_name && <p className="mt-1 text-sm text-red-600">{errors.workflow_name.message}</p>}
      </div>

      <fieldset className="flex flex-col gap-3 rounded border border-gray-200 p-4">
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
            className="w-40 rounded border border-gray-300 px-3 py-2"
          />
        ) : (
          <input
            type="text"
            placeholder="0 9 * * MON-FRI"
            value={cronExpression}
            onChange={(event) => setCronExpression(event.target.value)}
            className="w-full rounded border border-gray-300 px-3 py-2 font-mono"
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
          className="mt-1 w-full rounded border border-gray-300 px-3 py-2"
        >
          {OVERLAP_OPTIONS.map((o) => (
            <option key={o} value={o}>
              {o}
            </option>
          ))}
        </select>
      </div>

      <fieldset className="flex flex-col gap-4 rounded border border-gray-200 p-4">
        <legend className="px-1 text-sm font-medium">Steps</legend>
        {steps.map((step, index) => (
          // key={index} ปลอดภัยตรงนี้เพราะ step แค่ append/remove ท้าย ไม่มี reorder
          // กลางทาง (เหมือนเหตุผลเดียวกับ job-log-viewer.tsx) — ถ้าในอนาคตเพิ่ม drag
          // reorder ต้องเปลี่ยนไปใช้ unique id ต่อ step แทน
          <div key={index} className="flex flex-col gap-3 rounded border border-gray-100 p-3">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium">Step {index + 1}: Bot</span>
              {steps.length > 1 && (
                <button type="button" onClick={() => removeStep(index)} className="text-xs text-red-600 hover:underline">
                  ลบ step นี้
                </button>
              )}
            </div>
            <select
              value={step.bot_name}
              onChange={(event) => setStepBot(index, event.target.value)}
              className="rounded border border-gray-300 px-3 py-2"
            >
              <option value="">-- เลือก bot --</option>
              {bots?.map((b) => (
                <option key={b.name} value={b.name}>
                  {b.name}
                </option>
              ))}
            </select>

            {configSchemaFor(step.bot_name).map((field) => (
              <div key={field.key}>
                <label className="block text-sm">
                  {field.label}
                  {field.required && <span className="text-red-600"> *</span>}
                </label>
                <input
                  type={field.type === "url" ? "url" : "text"}
                  value={step.config[field.key] ?? ""}
                  onChange={(event) => setStepConfigValue(index, field.key, event.target.value)}
                  className="mt-1 w-full rounded border border-gray-300 px-3 py-2"
                />
              </div>
            ))}

            {/* Dispatch mode: local bot (default) / remote Temporal task queue /
                Kubernetes Job hybrid — ดู EXTERNAL-JOB-BOT-TODO.md,
                REMOTE-JOB-WORKER-TODO.md, K8S-JOB-HYBRID-EXECUTION-TODO.md ฝั่ง backend
                สำหรับความหมายเต็มๆ ของแต่ละทางเลือก bot_name ด้านบนยังจำเป็นต้องเลือก
                เสมอไม่ว่าโหมดไหน (ใช้เป็นแค่ label ใน StepResult ถ้าไม่ใช่ local — ไม่ได้
                ใช้ lookup อะไรฝั่ง backend เลยตอนนั้น) */}
            <div className="rounded border border-gray-100 bg-gray-50 p-3">
              <p className="mb-2 text-xs font-medium text-gray-500">รันที่ไหน</p>
              <div className="mb-3 flex flex-wrap gap-4 text-sm">
                {(Object.keys(DISPATCH_MODE_LABELS) as DispatchMode[]).map((mode) => (
                  <label key={mode} className="flex items-center gap-2">
                    <input
                      type="radio"
                      checked={dispatchModeOf(step) === mode}
                      onChange={() => setStepDispatchMode(index, mode)}
                    />
                    {DISPATCH_MODE_LABELS[mode]}
                  </label>
                ))}
              </div>

              {dispatchModeOf(step) === "remote_task_queue" && (
                <div>
                  <label className="block text-sm">Task Queue</label>
                  <input
                    type="text"
                    placeholder="external-job-queue"
                    value={step.task_queue ?? ""}
                    onChange={(event) => setStepTaskQueue(index, event.target.value)}
                    className="mt-1 w-full rounded border border-gray-300 px-3 py-2 font-mono"
                  />
                </div>
              )}

              {dispatchModeOf(step) === "k8s_job" && (
                <div className="flex flex-col gap-3">
                  <div>
                    <label className="block text-sm">
                      Image<span className="text-red-600"> *</span>
                    </label>
                    <input
                      type="text"
                      placeholder="registry.example.com/report-job:v1"
                      value={step.k8s_job?.image ?? ""}
                      onChange={(event) => setStepK8sJob(index, { image: event.target.value })}
                      className="mt-1 w-full rounded border border-gray-300 px-3 py-2 font-mono"
                    />
                  </div>
                  <div>
                    <label className="block text-sm">Args (คั่นด้วยจุลภาค)</label>
                    <input
                      type="text"
                      placeholder="--jobcode=SEND_REPORT,--dry-run"
                      value={(step.k8s_job?.args ?? []).join(",")}
                      onChange={(event) =>
                        setStepK8sJob(index, {
                          args: event.target.value
                            .split(",")
                            .map((s) => s.trim())
                            .filter(Boolean),
                        })
                      }
                      className="mt-1 w-full rounded border border-gray-300 px-3 py-2 font-mono"
                    />
                  </div>
                  <div>
                    <label className="block text-sm">Namespace (ไม่บังคับ)</label>
                    <input
                      type="text"
                      placeholder="bop-jobs"
                      value={step.k8s_job?.namespace ?? ""}
                      onChange={(event) => setStepK8sJob(index, { namespace: event.target.value || undefined })}
                      className="mt-1 w-full rounded border border-gray-300 px-3 py-2 font-mono"
                    />
                  </div>
                </div>
              )}
            </div>
          </div>
        ))}
        <button
          type="button"
          onClick={addStep}
          className="self-start rounded border border-gray-300 px-3 py-1 text-sm hover:bg-gray-50"
        >
          + เพิ่ม step
        </button>
      </fieldset>

      {formError && <p className="text-sm text-red-600">{formError}</p>}
      {mutation.isError && <p className="text-sm text-red-600">{mutation.error.message}</p>}

      <button
        type="submit"
        disabled={mutation.isPending}
        className="self-start rounded bg-blue-600 px-4 py-2 font-medium text-white hover:bg-blue-700 disabled:opacity-50"
      >
        {mutation.isPending ? "กำลังบันทึก..." : isEdit ? "บันทึกการแก้ไข" : "สร้าง schedule"}
      </button>
    </form>
  );
}
