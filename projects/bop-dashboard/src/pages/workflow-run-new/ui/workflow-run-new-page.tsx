import { RunWorkflowForm } from "@/features/schedule-form";

export function NewWorkflowRunPage() {
  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-2xl font-semibold">Run Workflow</h1>
      <p className="text-sm text-gray-500 dark:text-gray-400">รัน steps ทันทีครั้งเดียว โดยไม่ต้องสร้าง schedule</p>
      <RunWorkflowForm />
    </div>
  );
}
