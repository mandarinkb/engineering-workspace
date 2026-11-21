import type { Bot, ConfigField } from "@/entities/bot";
import type { DispatchMode, ScheduleStep } from "@/entities/schedule";

// dispatchModeOf derive DispatchMode จาก field ที่มีอยู่จริงใน step (ดู workflow.Step
// ฝั่ง Go: TaskQueue/K8sJob เป็น optional field ที่ mutually exclusive กันโดยธรรมชาติ —
// ไม่ได้เก็บเป็น field แยกตรงๆ ใน ScheduleStep เพราะ backend ไม่รู้จัก concept
// "dispatch mode" เลย รู้แค่ว่า step มี task_queue/k8s_job หรือไม่เท่านั้น)
export function dispatchModeOf(step: ScheduleStep): DispatchMode {
  if (step.k8s_job) return "k8s_job";
  if (step.task_queue) return "remote_task_queue";
  return "local";
}

export function configSchemaFor(bots: Bot[] | undefined, botName: string): ConfigField[] {
  return bots?.find((b) => b.name === botName)?.config_schema ?? [];
}

export const EMPTY_STEP: ScheduleStep = { bot_name: "", config: {} };

// validateSteps เช็ค steps ด้วยมือก่อน submit (ไม่ผ่าน zod เพราะ shape ของ config เปลี่ยน
// ตาม bot ที่เลือกแบบ runtime) — คืนข้อความ error ตัวแรกที่เจอ หรือ null ถ้าผ่าน ใช้ร่วมกัน
// ทั้งฟอร์ม schedule และฟอร์ม run workflow (backend รับ []workflow.Step shape เดียวกัน)
export function validateSteps(steps: ScheduleStep[], bots: Bot[] | undefined): string | null {
  if (steps.length === 0) {
    return "ต้องมีอย่างน้อย 1 step";
  }
  for (const step of steps) {
    if (!step.bot_name) {
      return "ทุก step ต้องเลือก bot";
    }
    for (const field of configSchemaFor(bots, step.bot_name)) {
      if (field.required && !step.config[field.key]) {
        return `step ที่ใช้ bot "${step.bot_name}" ต้องกรอก "${field.label}"`;
      }
    }
    // K8s Job hybrid execution ต้องมี image เสมอ (ตรงกับ validation ฝั่ง backend ใน
    // internal/workflow/k8sjob.go: RunKubernetesJob คืน bot.ConfigError ถ้าไม่มี) —
    // เช็คตรงนี้ก่อน submit เพื่อให้ user เห็น error เร็วกว่ารอ backend ตอบกลับมา
    if (step.k8s_job && !step.k8s_job.image.trim()) {
      return `step ที่ใช้ bot "${step.bot_name}" (Kubernetes Job) ต้องกรอก "image"`;
    }
  }
  return null;
}
