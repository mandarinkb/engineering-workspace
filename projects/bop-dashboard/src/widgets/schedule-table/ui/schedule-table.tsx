"use client";

import Link from "next/link";

import { useScheduleActions } from "../api/use-schedule-actions";
import { useSchedules } from "../api/use-schedules";
import type { ScheduleSummary } from "../types";
import { ScheduleStatusBadge } from "./schedule-status-badge";

// specLabel แปลง spec ของ schedule (interval หรือ cron — mutually exclusive เสมอ ดู
// ScheduleSpecInput) เป็นข้อความอ่านง่ายบรรทัดเดียวสำหรับตาราง
function specLabel(s: ScheduleSummary): string {
  if (s.cron_expression) return s.cron_expression;
  if (s.interval_seconds) return `ทุก ${s.interval_seconds} วินาที`;
  return "-";
}

// ScheduleTable คือหน้าหลักของการจัดการ schedule (เทียบเท่า Job list ของ Control-M) —
// แต่ละแถวมีปุ่ม pause/unpause/trigger ในตัวเลย ไม่ต้องกดเข้าไปหน้า detail ก่อน เพราะ
// เป็น action ที่ปลอดภัย/reverse ได้ง่าย (ต่างจาก delete ที่ต้องเข้าไปหน้า detail
// เท่านั้น — ดู schedule-detail.tsx)
export function ScheduleTable() {
  const { data: schedules, isLoading, isError, error } = useSchedules();

  if (isLoading) {
    return <p className="text-gray-500">กำลังโหลดรายการ schedule...</p>;
  }
  if (isError) {
    return <p className="text-red-600">โหลดรายการ schedule ไม่สำเร็จ: {error.message}</p>;
  }
  if (!schedules || schedules.length === 0) {
    return <p className="text-gray-500">ยังไม่มี schedule เลย — กด &quot;สร้าง schedule ใหม่&quot; เพื่อเริ่ม</p>;
  }

  return (
    <table className="w-full border-collapse text-left text-sm">
      <thead>
        <tr className="border-b border-gray-200 text-gray-500">
          <th className="py-2 pr-4">ID</th>
          <th className="py-2 pr-4">Workflow</th>
          <th className="py-2 pr-4">สถานะ</th>
          <th className="py-2 pr-4">ตารางเวลา</th>
          <th className="py-2 pr-4">รอบถัดไป</th>
          <th className="py-2 pr-4">การจัดการ</th>
        </tr>
      </thead>
      <tbody>
        {schedules.map((schedule) => (
          <ScheduleRow key={schedule.id} schedule={schedule} />
        ))}
      </tbody>
    </table>
  );
}

// ScheduleRow แยกออกมาเป็น component ต่างหาก (ไม่ใส่ logic ของปุ่มไว้ใน .map() ตรงๆ)
// เพราะ useScheduleActions เป็น hook — เรียก hook ข้างใน callback ของ .map() ตรงๆ ผิด
// กฎ "Rules of Hooks" ของ React (hook ต้องเรียกจาก component/hook function เท่านั้น
// ไม่ใช่จาก loop/condition/callback ธรรมดา) แยกเป็น component ทำให้แต่ละแถวมี hook
// instance ของตัวเอง ถูกต้องตามกฎ
function ScheduleRow({ schedule }: { schedule: ScheduleSummary }) {
  const { pause, unpause, trigger } = useScheduleActions(schedule.id);

  return (
    <tr className="border-b border-gray-100 hover:bg-gray-50">
      <td className="py-2 pr-4">
        <Link href={`/schedules/${schedule.id}`} className="font-mono text-blue-600 hover:underline">
          {schedule.id}
        </Link>
      </td>
      <td className="py-2 pr-4">{schedule.workflow_name}</td>
      <td className="py-2 pr-4">
        <ScheduleStatusBadge paused={schedule.paused} />
      </td>
      <td className="py-2 pr-4 font-mono text-gray-600">{specLabel(schedule)}</td>
      <td className="py-2 pr-4 text-gray-500">
        {schedule.next_action_times?.[0] ? new Date(schedule.next_action_times[0]).toLocaleString() : "-"}
      </td>
      <td className="py-2 pr-4">
        <div className="flex gap-2">
          <button
            type="button"
            onClick={() => trigger.mutate(undefined)}
            disabled={trigger.isPending}
            className="rounded border border-gray-300 px-2 py-1 text-xs hover:bg-gray-100 disabled:opacity-40"
            title="สั่งรันทันที นอกตารางเวลาปกติ"
          >
            Run Now
          </button>
          {schedule.paused ? (
            <button
              type="button"
              onClick={() => unpause.mutate(undefined)}
              disabled={unpause.isPending}
              className="rounded border border-gray-300 px-2 py-1 text-xs hover:bg-gray-100 disabled:opacity-40"
            >
              Unpause
            </button>
          ) : (
            <button
              type="button"
              onClick={() => pause.mutate(undefined)}
              disabled={pause.isPending}
              className="rounded border border-gray-300 px-2 py-1 text-xs hover:bg-gray-100 disabled:opacity-40"
            >
              Pause
            </button>
          )}
        </div>
      </td>
    </tr>
  );
}
