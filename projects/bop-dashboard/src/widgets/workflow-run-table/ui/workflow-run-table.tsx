"use client";

import Link from "next/link";
import { useState } from "react";

import {
  flattenRunPages,
  useWorkflowRunPages,
  WorkflowStatusBadge,
  type RunStatusFilter,
} from "@/entities/workflow-run";

import { RUN_STATUS_FILTER_LABELS, searchRuns } from "../lib/filter-runs";

// WorkflowRunTable แสดง workflow_id (ไม่ใช่ workflow_name ซึ่งเป็นชื่อ workflow type "Execute"
// เหมือนกันทุกแถว) โหลดทีละหน้าจาก backend ด้วยปุ่ม "โหลดเพิ่ม" — schedule ที่รันทุกนาทีสร้าง
// run วันละ 1,440 ตัว โหลดทั้งหมดมาแบ่งหน้าฝั่ง client แบบเดิมไม่ไหว filter สถานะส่งไปกรองฝั่ง
// server ส่วนช่องค้นหากรองเฉพาะแถวที่โหลดมาแล้ว
export function WorkflowRunTable() {
  const [status, setStatus] = useState<RunStatusFilter>("");
  const [search, setSearch] = useState("");
  const { data, isLoading, isError, error, hasNextPage, fetchNextPage, isFetchingNextPage } = useWorkflowRunPages({
    status,
  });

  const loaded = flattenRunPages(data?.pages ?? []);
  const rows = searchRuns(loaded, search);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-3">
        <input
          type="search"
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          placeholder="ค้นหา workflow ID / ชื่อ schedule..."
          aria-label="ค้นหา workflow ID"
          className="w-72 rounded border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-transparent"
        />
        <select
          value={status}
          onChange={(event) => setStatus(event.target.value as RunStatusFilter)}
          aria-label="กรองตามสถานะ"
          className="rounded border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-transparent"
        >
          <option value="">ทุกสถานะ</option>
          {(Object.keys(RUN_STATUS_FILTER_LABELS) as Exclude<RunStatusFilter, "">[]).map((key) => (
            <option key={key} value={key}>
              {RUN_STATUS_FILTER_LABELS[key]}
            </option>
          ))}
        </select>
        {data && (
          <span className="text-sm text-gray-500 dark:text-gray-400">
            {search.trim() ? `${rows.length} จาก ` : ""}
            {loaded.length} run ที่โหลดแล้ว{hasNextPage ? "" : " (ครบแล้ว)"}
          </span>
        )}
      </div>

      {isLoading ? (
        <p className="text-gray-500">กำลังโหลด workflow run...</p>
      ) : isError ? (
        <p className="text-red-600">โหลด workflow run ไม่สำเร็จ: {error.message}</p>
      ) : loaded.length === 0 && status === "" ? (
        <p className="text-gray-500">ยังไม่มี workflow run เลย</p>
      ) : rows.length === 0 ? (
        <p className="text-gray-500">ไม่มี workflow run ที่ตรงกับตัวกรอง</p>
      ) : (
        <table className="w-full border-collapse text-left text-sm">
          <thead>
            <tr className="border-b border-gray-200 text-gray-500 dark:border-gray-700">
              {/* กว้าง 60% ให้ ID เต็มของ run จาก schedule (ราว 40 ตัวอักษร) ไม่โดนตัดตรงส่วนเวลา ซึ่งเป็นส่วนที่แยกแต่ละ run */}
              <th className="w-3/5 py-2 pr-4">Workflow ID</th>
              <th className="py-2 pr-4">Status</th>
              <th className="py-2 pr-4">เริ่มเมื่อ</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((run) => (
              <tr
                key={`${run.workflow_id}/${run.run_id}`}
                className="border-b border-gray-100 hover:bg-gray-50 dark:border-gray-800 dark:hover:bg-gray-800/50"
              >
                <td className="max-w-0 py-2 pr-4">
                  <Link
                    href={`/workflow-runs/${encodeURIComponent(run.workflow_id)}`}
                    title={run.workflow_id}
                    className="block truncate font-mono text-blue-600 hover:underline dark:text-blue-400"
                  >
                    {run.workflow_id}
                  </Link>
                </td>
                <td className="py-2 pr-4">
                  <WorkflowStatusBadge status={run.status} />
                </td>
                <td className="whitespace-nowrap py-2 pr-4 text-gray-500 dark:text-gray-400">
                  {new Date(run.started_at).toLocaleString()}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {hasNextPage && (
        <div className="text-sm">
          <button
            type="button"
            disabled={isFetchingNextPage}
            onClick={() => fetchNextPage()}
            className="rounded border border-gray-300 px-3 py-1 hover:bg-gray-50 disabled:opacity-40 dark:border-gray-600 dark:hover:bg-gray-800"
          >
            {isFetchingNextPage ? "กำลังโหลด..." : "โหลดเพิ่ม"}
          </button>
        </div>
      )}
    </div>
  );
}
