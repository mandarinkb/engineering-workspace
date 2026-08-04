import { ScheduleForm } from "@/features/schedules/components/schedule-form";

export default function NewSchedulePage() {
  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-2xl font-semibold">สร้าง Schedule ใหม่</h1>
      <ScheduleForm />
    </div>
  );
}
