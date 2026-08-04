import Link from "next/link";

import { ScheduleTable } from "@/features/schedules/components/schedule-table";

export default function SchedulesPage() {
  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Schedules</h1>
        <Link
          href="/schedules/new"
          className="rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
        >
          + สร้าง schedule ใหม่
        </Link>
      </div>
      <ScheduleTable />
    </div>
  );
}
