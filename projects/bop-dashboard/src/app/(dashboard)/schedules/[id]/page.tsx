import { ScheduleDetailView } from "@/features/schedules/components/schedule-detail";

interface ScheduleDetailPageProps {
  params: Promise<{ id: string }>; // Next.js 15: params เป็น Promise เสมอ ต้อง await
}

export default async function ScheduleDetailPage({ params }: ScheduleDetailPageProps) {
  const { id } = await params;
  return (
    <div className="flex flex-col gap-6">
      <h1 className="break-all font-mono text-2xl font-semibold">{id}</h1>
      <ScheduleDetailView id={id} />
    </div>
  );
}
