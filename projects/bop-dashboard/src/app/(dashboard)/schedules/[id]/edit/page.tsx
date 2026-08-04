import { ScheduleEditView } from "@/features/schedules/components/schedule-edit";

interface EditSchedulePageProps {
  params: Promise<{ id: string }>; // Next.js 15: params เป็น Promise เสมอ ต้อง await
}

export default async function EditSchedulePage({ params }: EditSchedulePageProps) {
  const { id } = await params;
  return (
    <div className="flex flex-col gap-6">
      <h1 className="break-all text-2xl font-semibold">แก้ไข {id}</h1>
      <ScheduleEditView id={id} />
    </div>
  );
}
