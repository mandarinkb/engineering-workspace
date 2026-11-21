"use client";

import Link from "next/link";
import { useState } from "react";

import { useWorkflowRuns, WorkflowStatusBadge } from "@/entities/workflow-run";

import { filterRuns, RUN_STATUS_FILTER_LABELS, type RunStatusFilter } from "../lib/filter-runs";

const PAGE_SIZE = 20;

// WorkflowRunTable แสดง workflow_id (ไม่ใช่ workflow_name ซึ่งเป็นชื่อ workflow type "Execute"
// เหมือนกันทุกแถว) พร้อม filter สถานะ/ค้นหา และแบ่งหน้าฝั่ง client เหมือน job-table — schedule
// ที่รันทุกนาทีสร้าง run วันละ 1,440 ตัว ตารางที่ไม่แบ่งหน้าจะยาวไม่จบ
export function WorkflowRunTable() {
  const { data: runs, isLoading, isError, error } = useWorkflowRuns();
  const [status, setStatus] = useState<RunStatusFilter>("");
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);

  if (isLoading) {
    return <p className="text-gray-500">กำลังโหลด workflow run...</p>;
  }
  if (isError) {
    return <p className="text-red-600">โหลด workflow run ไม่สำเร็จ: {error.message}</p>;
  }
  if (!runs || runs.length === 0) {
    return <p className="text-gray-500">ยังไม่มี workflow run เลย</p>;
  }

  const filtered = filterRuns(runs, { status, search });
  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE));
  // ข้อมูล refetch ทุก 10 วินาที จำนวนหน้าอาจลดลงได้ — clamp ไว้ไม่ให้ค้างอยู่หน้าที่ไม่มีแล้ว
  const currentPage = Math.min(page, totalPages);
  const rows = filtered.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-3">
        <input
          type="search"
          value={search}
          onChange={(event) => {
            setSearch(event.target.value);
            setPage(1);
          }}
          placeholder="ค้นหา workflow ID / ชื่อ schedule..."
          aria-label="ค้นหา workflow ID"
          className="w-72 rounded border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-transparent"
        />
        <select
          value={status}
          onChange={(event) => {
            setStatus(event.target.value as RunStatusFilter);
            setPage(1);
          }}
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
        <span className="text-sm text-gray-500 dark:text-gray-400">
          {filtered.length} จาก {runs.length} run
        </span>
      </div>

      {rows.length === 0 ? (
        <p className="text-gray-500">ไม่มี workflow run ที่ตรงกับตัวกรอง</p>
      ) : (
        <table className="w-full border-collapse text-left text-sm">
          <thead>
            <tr className="border-b border-gray-200 text-gray-500 dark:border-gray-700">
              <th className="py-2 pr-4">Workflow ID</th>
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

      {totalPages > 1 && (
        <div className="flex items-center gap-3 text-sm">
          <button
            type="button"
            disabled={currentPage <= 1}
            onClick={() => setPage(currentPage - 1)}
            className="rounded border border-gray-300 px-3 py-1 hover:bg-gray-50 disabled:opacity-40 dark:border-gray-600 dark:hover:bg-gray-800"
          >
            ก่อนหน้า
          </button>
          <span className="text-gray-500 dark:text-gray-400">
            หน้า {currentPage} / {totalPages}
          </span>
          <button
            type="button"
            disabled={currentPage >= totalPages}
            onClick={() => setPage(currentPage + 1)}
            className="rounded border border-gray-300 px-3 py-1 hover:bg-gray-50 disabled:opacity-40 dark:border-gray-600 dark:hover:bg-gray-800"
          >
            ถัดไป
          </button>
        </div>
      )}
    </div>
  );
}
