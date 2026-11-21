"use client";

import type { Bot } from "@/entities/bot";
import type { DispatchMode, ScheduleStep } from "@/entities/schedule";

import { configSchemaFor, dispatchModeOf, EMPTY_STEP } from "../lib/steps";

const DISPATCH_MODE_LABELS: Record<DispatchMode, string> = {
  local: "Local Bot",
  remote_task_queue: "Remote Task Queue",
  k8s_job: "Kubernetes Job",
};

interface StepsEditorProps {
  steps: ScheduleStep[];
  onChange: (steps: ScheduleStep[]) => void;
  bots: Bot[] | undefined;
}

// StepsEditor คือส่วนแก้ steps (เลือก bot + config dynamic ตาม bot + dispatch mode) แยกออกมา
// จาก ScheduleForm เพื่อให้ RunWorkflowForm ใช้ซ้ำได้ — เป็น controlled component: state ของ
// steps อยู่ที่ฟอร์มที่เรียกใช้ ไฟล์นี้แค่ render และส่ง steps ชุดใหม่กลับผ่าน onChange
export function StepsEditor({ steps, onChange, bots }: StepsEditorProps) {
  function update(index: number, fn: (step: ScheduleStep) => ScheduleStep) {
    onChange(steps.map((s, i) => (i === index ? fn(s) : s)));
  }

  function addStep() {
    onChange([...steps, EMPTY_STEP]);
  }
  function removeStep(index: number) {
    onChange(steps.filter((_, i) => i !== index));
  }
  function setStepBot(index: number, botName: string) {
    // เปลี่ยน bot ของ step นี้ต้องล้าง config เดิมทิ้งเสมอ เพราะ config field ของ bot
    // ใหม่อาจไม่ตรงกับของ bot เดิมเลย (คนละ key กัน) เก็บ config เก่าไว้จะสับสน
    // ส่วน max_attempts ไม่ผูกกับ bot ตัวไหน จึงเก็บค่าเดิมไว้
    update(index, (s) => ({ bot_name: botName, config: {}, max_attempts: s.max_attempts }));
  }
  // setStepMaxAttempts รับค่าจากช่องตัวเลข — ช่องว่างแปลงเป็น undefined เพื่อไม่ส่ง field นี้
  // ไปเลย (backend ใช้ค่าเริ่มต้น 3) ส่วนค่าที่ไม่ถูกต้องเก็บไว้ตามที่พิมพ์ ให้ validateSteps บอก error
  function setStepMaxAttempts(index: number, value: string) {
    update(index, (s) => ({ ...s, max_attempts: value === "" ? undefined : Number(value) }));
  }
  function setStepConfigValue(index: number, key: string, value: string) {
    update(index, (s) => ({ ...s, config: { ...s.config, [key]: value } }));
  }

  // setStepDispatchMode สลับโหมดของ step — ต้องเคลียร์ field ของโหมดเดิมทิ้งเสมอ (ไม่ใช่
  // แค่เพิ่ม field ของโหมดใหม่เข้าไปเฉยๆ) เพราะ task_queue/k8s_job เป็น mutually
  // exclusive กันฝั่ง backend (ดู workflow.go: "ห้ามตั้งทั้ง TaskQueue และ K8sJob พร้อมกัน")
  // ตั้ง k8s_job เริ่มต้นเป็น {image: ""} ทันทีที่สลับมาโหมดนี้ (ไม่ใช่ undefined) เพื่อให้
  // ฟอร์มด้านล่าง render input ของ image ได้เลยโดยไม่ต้องเช็ค step.k8s_job ทุกจุด
  function setStepDispatchMode(index: number, mode: DispatchMode) {
    update(index, (s) => {
      // สร้าง base object จาก field ที่รู้จักตรงๆ (ไม่ destructure แล้วทิ้ง
      // task_queue/k8s_job เดิม) เพื่อเคลียร์ทั้งสอง field นี้ให้แน่ใจว่าไม่มีค่าเก่า
      // ของโหมดก่อนหน้าค้างอยู่ ไม่ว่าจะสลับมาโหมดไหนก็ตาม
      const base: ScheduleStep = { bot_name: s.bot_name, config: s.config, max_attempts: s.max_attempts };
      if (mode === "remote_task_queue") return { ...base, task_queue: "" };
      if (mode === "k8s_job") return { ...base, k8s_job: { image: "" } };
      return base;
    });
  }
  function setStepTaskQueue(index: number, value: string) {
    update(index, (s) => ({ ...s, task_queue: value }));
  }
  function setStepK8sJob(index: number, patch: Partial<NonNullable<ScheduleStep["k8s_job"]>>) {
    update(index, (s) => ({ ...s, k8s_job: { ...s.k8s_job, ...patch, image: patch.image ?? s.k8s_job?.image ?? "" } }));
  }

  return (
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
            aria-label={`bot ของ step ${index + 1}`}
            className="rounded border border-gray-300 px-3 py-2"
          >
            <option value="">-- เลือก bot --</option>
            {bots?.map((b) => (
              <option key={b.name} value={b.name}>
                {b.name}
              </option>
            ))}
          </select>

          {configSchemaFor(bots, step.bot_name).map((field) => (
            <div key={field.key}>
              <label className="block text-sm">
                {field.label}
                {field.required && <span className="text-red-600"> *</span>}
              </label>
              <input
                type={field.type === "url" ? "url" : "text"}
                value={step.config[field.key] ?? ""}
                onChange={(event) => setStepConfigValue(index, field.key, event.target.value)}
                aria-label={field.label}
                className="mt-1 w-full rounded border border-gray-300 px-3 py-2"
              />
            </div>
          ))}

          {/* จำนวน Attempt สูงสุดใช้ได้กับทุก dispatch mode จึงอยู่นอกกล่อง "รันที่ไหน" */}
          <div>
            <label className="block text-sm" htmlFor={`max-attempts-${index}`}>
              จำนวนครั้งที่ลองสูงสุด
            </label>
            <input
              id={`max-attempts-${index}`}
              type="number"
              min={1}
              step={1}
              placeholder="ค่าเริ่มต้น 3"
              value={step.max_attempts ?? ""}
              onChange={(event) => setStepMaxAttempts(index, event.target.value)}
              className="mt-1 w-40 rounded border border-gray-300 px-3 py-2"
            />
          </div>

          {/* Dispatch mode: local bot (default) / remote Temporal task queue /
              Kubernetes Job hybrid — ดู EXTERNAL-JOB-BOT-TODO.md,
              REMOTE-JOB-WORKER-TODO.md, K8S-JOB-HYBRID-EXECUTION-TODO.md ฝั่ง backend
              สำหรับความหมายเต็มๆ ของแต่ละทางเลือก bot_name ด้านบนยังจำเป็นต้องเลือก
              เสมอไม่ว่าโหมดไหน (ใช้เป็นแค่ label ใน StepResult ถ้าไม่ใช่ local — ไม่ได้
              ใช้ lookup อะไรฝั่ง backend เลยตอนนั้น) */}
          <div className="rounded border border-gray-100 bg-gray-50 p-3 dark:bg-gray-900">
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
  );
}
