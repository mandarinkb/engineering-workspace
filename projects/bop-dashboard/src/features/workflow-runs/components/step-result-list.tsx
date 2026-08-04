import type { StepResult } from "../types";

interface StepResultListProps {
  steps: StepResult[];
}

// StepResultList แสดงผลลัพธ์ทีละ step ของ workflow run เดียว — steps มาจาก Temporal
// Query handler "steps" (ดู workflow.Execute ใน internal/workflow/temporal.go) ทำงานได้
// ทั้งตอน workflow กำลังรันอยู่ (เห็นแค่ step ที่ทำไปแล้ว) และจบไปแล้ว (เห็นครบทุก step)
export function StepResultList({ steps }: StepResultListProps) {
  if (steps.length === 0) {
    return <p className="text-gray-500">ยังไม่มี step ไหนรันเสร็จเลย (workflow อาจเพิ่งเริ่ม)</p>;
  }

  return (
    <ol className="flex flex-col gap-2">
      {steps.map((step) => (
        <li key={step.step_index} className="rounded border border-gray-200 p-3">
          <p className="font-mono text-sm">
            step {step.step_index}: {step.bot_name}
          </p>
          {/* step ที่ล้มเหลวมี error แต่ไม่มี summary (คนละกันเสมอ ดู StepResult ฝั่ง Go
              ที่ activity.go/temporal.go) แสดง error เป็นสีแดงแยกจาก summary ปกติ */}
          {step.error ? (
            <p className="mt-1 text-sm text-red-600">{step.error}</p>
          ) : (
            <p className="mt-1 text-sm text-gray-600">{step.summary}</p>
          )}
        </li>
      ))}
    </ol>
  );
}
