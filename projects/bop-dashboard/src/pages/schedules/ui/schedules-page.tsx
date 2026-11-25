import Link from "next/link";

import { ScheduleTable } from "@/widgets/schedule-table";

export function SchedulesPage() {
  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl">Schedules</h1>
        <Link
          href="/schedules/new"
          className="btn-primary px-4 py-2 text-sm"
        >
          + สร้าง schedule ใหม่
        </Link>
      </div>
      <ScheduleTable />
    </div>
  );
}
