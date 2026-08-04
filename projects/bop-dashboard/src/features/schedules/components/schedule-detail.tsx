"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";

import { useDeleteSchedule } from "../api/use-delete-schedule";
import { useSchedule } from "../api/use-schedule";
import { useScheduleActions } from "../api/use-schedule-actions";
import { ScheduleStatusBadge } from "./schedule-status-badge";

interface ScheduleDetailViewProps {
  id: string;
}

// ScheduleDetailView คือหน้าเดียวที่รวมทุกปุ่มจัดการ schedule ไว้ (pause/unpause/
// trigger/backfill/edit/delete) ต่างจาก schedule-table.tsx ที่มีแค่ปุ่มปลอดภัย
// (pause/unpause/trigger) ให้กดจากหน้า list ตรงๆ — delete กับ backfill ต้องเข้ามาที่นี่
// เท่านั้นเพราะเป็น action ที่ผลกระทบมากกว่า (ลบถาวร / รันย้อนหลังหลายรอบพร้อมกัน)
export function ScheduleDetailView({ id }: ScheduleDetailViewProps) {
  const router = useRouter();
  const { data: schedule, isLoading, isError, error } = useSchedule(id);
  const { pause, unpause, trigger, backfill } = useScheduleActions(id);
  const deleteSchedule = useDeleteSchedule();

  const [showBackfill, setShowBackfill] = useState(false);
  const [backfillStart, setBackfillStart] = useState("");
  const [backfillEnd, setBackfillEnd] = useState("");

  if (isLoading) {
    return <p className="text-gray-500">กำลังโหลด...</p>;
  }
  if (isError) {
    return <p className="text-red-600">โหลด schedule ไม่สำเร็จ: {error.message}</p>;
  }
  if (!schedule) {
    return null;
  }

  function handleDelete() {
    // confirm() เป็น browser API พื้นฐานที่สุดสำหรับ "ยืนยันก่อนทำ action ที่ย้อนกลับ
    // ไม่ได้" — เพียงพอสำหรับตอนนี้ ไม่ต้องสร้าง modal component เองสำหรับ use case
    // เดียวที่ใช้ตรงนี้ที่เดียว (YAGNI)
    if (!confirm(`ลบ schedule "${id}" ถาวร? การกระทำนี้ย้อนกลับไม่ได้`)) {
      return;
    }
    deleteSchedule.mutate(id, { onSuccess: () => router.push("/schedules") });
  }

  function handleBackfill() {
    if (!backfillStart || !backfillEnd) return;
    backfill.mutate(
      { start_time: new Date(backfillStart).toISOString(), end_time: new Date(backfillEnd).toISOString() },
      { onSuccess: () => setShowBackfill(false) },
    );
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-center gap-3">
        <ScheduleStatusBadge paused={schedule.paused} />
        <span className="text-sm text-gray-500">Workflow: {schedule.workflow_name}</span>
        <span className="text-sm text-gray-500">Overlap: {schedule.overlap}</span>
      </div>

      <div className="flex flex-wrap gap-2">
        <button
          type="button"
          onClick={() => trigger.mutate(undefined)}
          disabled={trigger.isPending}
          className="rounded border border-gray-300 px-3 py-1.5 text-sm hover:bg-gray-50 disabled:opacity-40"
          title="สั่งรันทันที นอกตารางเวลาปกติ"
        >
          Run Now
        </button>
        {schedule.paused ? (
          <button
            type="button"
            onClick={() => unpause.mutate(undefined)}
            disabled={unpause.isPending}
            className="rounded border border-gray-300 px-3 py-1.5 text-sm hover:bg-gray-50 disabled:opacity-40"
          >
            Unpause
          </button>
        ) : (
          <button
            type="button"
            onClick={() => pause.mutate(undefined)}
            disabled={pause.isPending}
            className="rounded border border-gray-300 px-3 py-1.5 text-sm hover:bg-gray-50 disabled:opacity-40"
          >
            Pause
          </button>
        )}
        <Link
          href={`/schedules/${id}/edit`}
          className="rounded border border-gray-300 px-3 py-1.5 text-sm hover:bg-gray-50"
        >
          แก้ไข
        </Link>
        <button
          type="button"
          onClick={() => setShowBackfill((v) => !v)}
          className="rounded border border-gray-300 px-3 py-1.5 text-sm hover:bg-gray-50"
        >
          Backfill
        </button>
        <button
          type="button"
          onClick={handleDelete}
          disabled={deleteSchedule.isPending}
          className="rounded border border-red-300 px-3 py-1.5 text-sm text-red-600 hover:bg-red-50 disabled:opacity-40"
        >
          ลบ
        </button>
      </div>

      {showBackfill && (
        <div className="flex max-w-sm flex-col gap-3 rounded border border-gray-200 p-4">
          <p className="text-sm text-gray-600">
            รัน schedule นี้ย้อนหลังตามช่วงเวลาที่ระบุ ราวกับเวลานั้นเพิ่งผ่านไปตอนนี้ (เทียบเท่า &quot;Rerun&quot;)
          </p>
          <label className="text-sm">
            เริ่ม
            <input
              type="datetime-local"
              value={backfillStart}
              onChange={(event) => setBackfillStart(event.target.value)}
              className="mt-1 w-full rounded border border-gray-300 px-3 py-2"
            />
          </label>
          <label className="text-sm">
            สิ้นสุด
            <input
              type="datetime-local"
              value={backfillEnd}
              onChange={(event) => setBackfillEnd(event.target.value)}
              className="mt-1 w-full rounded border border-gray-300 px-3 py-2"
            />
          </label>
          <button
            type="button"
            onClick={handleBackfill}
            disabled={backfill.isPending}
            className="self-start rounded bg-blue-600 px-3 py-1.5 text-sm text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {backfill.isPending ? "กำลังส่ง..." : "ยืนยัน backfill"}
          </button>
          {backfill.isError && <p className="text-sm text-red-600">{backfill.error.message}</p>}
        </div>
      )}

      <div>
        <h2 className="mb-2 text-lg font-semibold">Steps</h2>
        {schedule.steps && schedule.steps.length > 0 ? (
          <ol className="flex flex-col gap-2">
            {schedule.steps.map((step, index) => (
              <li key={index} className="rounded border border-gray-200 p-3 font-mono text-sm">
                step {index + 1}: {step.bot_name} — {JSON.stringify(step.config)}
              </li>
            ))}
          </ol>
        ) : (
          <p className="text-sm text-gray-500">
            ไม่มีข้อมูล steps ให้ดู (schedule นี้อาจสร้างก่อนมี API ชุดนี้ หรือ backend เพิ่ง restart ไป — ดูหมายเหตุที่
            internal/schedule/schedule.go ฝั่ง backend)
          </p>
        )}
      </div>

      <div>
        <h2 className="mb-2 text-lg font-semibold">ประวัติการรันล่าสุด</h2>
        {schedule.recent_actions && schedule.recent_actions.length > 0 ? (
          <ol className="flex flex-col gap-2">
            {schedule.recent_actions.map((action, index) => (
              <li key={index} className="rounded border border-gray-200 p-3 text-sm">
                <span className="text-gray-500">{new Date(action.actual_time).toLocaleString()}</span>
                {action.workflow_id && <span className="ml-2 font-mono">{action.workflow_id}</span>}
              </li>
            ))}
          </ol>
        ) : (
          <p className="text-sm text-gray-500">ยังไม่มีประวัติการรันเลย</p>
        )}
      </div>
    </div>
  );
}
