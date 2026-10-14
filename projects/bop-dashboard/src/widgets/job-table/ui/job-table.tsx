"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useState } from "react";

import { useCancelJob } from "../api/use-cancel-job";
import { useJobs } from "../api/use-jobs";
import type { Job, JobStatus } from "../types";
import { JobStatusBadge } from "./job-status-badge";

const PAGE_SIZE = 20;

// CANCELLABLE_STATUSES ตรงกับเงื่อนไขที่ backend ยอม cancel จริง (worker.Pool.Cancel คืน
// false ถ้า job ไม่ได้กำลังรันอยู่ — ดู handleCancelJob) เอาไว้ตัดสินใจว่าจะโชว์
// checkbox ให้เลือก job แถวไหนได้บ้าง (ไม่ให้เลือก job ที่จบไปแล้ว เพราะกด cancel ไปก็
// จะเจอ 409 Conflict เปล่าๆ)
const CANCELLABLE_STATUSES: JobStatus[] = ["pending", "running"];

function isWithinDateRange(job: Job, from: string, to: string): boolean {
  const createdAt = new Date(job.created_at).getTime();
  if (from && createdAt < new Date(from).getTime()) return false;
  // "to" เป็นแค่วันที่ (ไม่มีเวลา) — บวก 1 วันแล้วเทียบแบบ "น้อยกว่า" แทน "น้อยกว่าเท่ากับ"
  // ทำให้ครอบคลุมทั้งวันที่เลือกไว้ (เช่น to=2026-01-05 ต้องรวม job ที่สร้างเวลา 23:59
  // ของวันที่ 5 ด้วย ไม่ใช่ตัดที่เที่ยงคืนของวันเดียวกันซึ่งจะเท่ากับไม่รวมวันนั้นเลย)
  if (to) {
    const toExclusive = new Date(to).getTime() + 24 * 60 * 60 * 1000;
    if (createdAt >= toExclusive) return false;
  }
  return true;
}

// JobTable คือหน้าหลักของ roadmap ข้อ 1.4 — ดึง job ทั้งหมดผ่าน useJobs (TanStack Query
// cache/retry อัตโนมัติให้แล้ว ดู src/lib/query-client.ts) แล้วกรอง+แบ่งหน้าฝั่ง client
// เอง (ไม่พึ่ง backend ทำ filter/pagination เพราะ endpoint GET /jobs ยังไม่รองรับ query
// parameter พวกนี้ และจำนวน job ทั้งระบบตอนนี้ยังน้อยพอที่ client-side filter จะไม่ช้า)
//
// ตัวกรอง (bot/status/from/to) มาจาก URL query param โดยตรง (useSearchParams) ไม่ใช่
// state ของ component นี้เอง — sync กับ job-filter.tsx (คนละไฟล์ที่เขียนค่าลง URL
// เดียวกัน) ผ่าน URL เป็นตัวกลาง ส่วน page/selectedIds ยังเป็น useState ธรรมดา เพราะไม่
// จำเป็นต้อง sync ข้าม URL
export function JobTable() {
  const { data: jobs, isLoading, isError, error } = useJobs();
  const searchParams = useSearchParams();
  const [page, setPage] = useState(1);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const cancelJob = useCancelJob();

  if (isLoading) {
    return <p className="text-gray-500">กำลังโหลด job history...</p>;
  }
  if (isError) {
    return <p className="text-red-600">โหลด job history ไม่สำเร็จ: {error.message}</p>;
  }

  const botFilter = searchParams.get("bot")?.toLowerCase() ?? "";
  const statusFilter = (searchParams.get("status") as JobStatus | "" | null) ?? "";
  const fromFilter = searchParams.get("from") ?? "";
  const toFilter = searchParams.get("to") ?? "";

  const filtered = (jobs ?? []).filter((job) => {
    if (botFilter && !job.bot_name.toLowerCase().includes(botFilter)) return false;
    if (statusFilter && job.status !== statusFilter) return false;
    if (!isWithinDateRange(job, fromFilter, toFilter)) return false;
    return true;
  });

  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE));
  const currentPage = Math.min(page, totalPages);
  const rows = filtered.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE);
  const cancellableRowIds = rows.filter((j) => CANCELLABLE_STATUSES.includes(j.status)).map((j) => j.id);
  const allCancellableSelected = cancellableRowIds.length > 0 && cancellableRowIds.every((id) => selectedIds.has(id));

  function toggleRow(id: string) {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  function toggleSelectAll() {
    setSelectedIds((prev) => {
      if (allCancellableSelected) {
        const next = new Set(prev);
        for (const id of cancellableRowIds) next.delete(id);
        return next;
      }
      return new Set([...prev, ...cancellableRowIds]);
    });
  }

  // bulkCancel ยิง DELETE /jobs/{id} พร้อมกันทีละหลาย id ผ่าน Promise.allSettled — ไม่มี
  // bulk endpoint ฝั่ง backend ให้เรียกทีเดียว (ดู use-cancel-job.ts) ใช้ allSettled แทน
  // Promise.all เพราะบาง job อาจ cancel ไม่สำเร็จ (เช่นจบไปพอดีก่อนคำสั่งไปถึง) ไม่อยาก
  // ให้ job อื่นที่ cancel สำเร็จพลอย fail ไปด้วยเพราะ Promise.all หยุดที่ error แรกที่เจอ
  async function bulkCancel() {
    const ids = Array.from(selectedIds);
    await Promise.allSettled(ids.map((id) => cancelJob.mutateAsync(id)));
    setSelectedIds(new Set());
  }

  if (filtered.length === 0) {
    return <p className="text-gray-500">ยังไม่มี job ที่ตรงกับตัวกรอง</p>;
  }

  return (
    <div className="flex flex-col gap-3">
      {selectedIds.size > 0 && (
        <div className="flex items-center gap-3 rounded border border-blue-200 bg-blue-50 px-3 py-2 text-sm">
          <span>เลือกไว้ {selectedIds.size} งาน</span>
          <button
            type="button"
            onClick={bulkCancel}
            disabled={cancelJob.isPending}
            className="rounded bg-red-600 px-3 py-1 text-xs font-medium text-white hover:bg-red-700 disabled:opacity-50"
          >
            {cancelJob.isPending ? "กำลังยกเลิก..." : "ยกเลิกที่เลือก"}
          </button>
        </div>
      )}

      <table className="w-full border-collapse text-left text-sm">
        <thead>
          <tr className="border-b border-gray-200 text-gray-500">
            <th className="w-8 py-2">
              <input
                type="checkbox"
                checked={allCancellableSelected}
                disabled={cancellableRowIds.length === 0}
                onChange={toggleSelectAll}
                aria-label="เลือกทั้งหมดที่ยกเลิกได้"
              />
            </th>
            <th className="py-2 pr-4">Bot</th>
            <th className="py-2 pr-4">Status</th>
            <th className="py-2 pr-4">Summary</th>
            <th className="py-2 pr-4">เริ่มเมื่อ</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((job) => (
            <tr key={job.id} className="border-b border-gray-100 hover:bg-gray-50">
              <td className="py-2">
                <input
                  type="checkbox"
                  checked={selectedIds.has(job.id)}
                  disabled={!CANCELLABLE_STATUSES.includes(job.status)}
                  onChange={() => toggleRow(job.id)}
                  aria-label={`เลือก job ${job.id}`}
                />
              </td>
              <td className="py-2 pr-4">
                {/* ลิงก์ไปหน้า live monitor ของ job นี้ (ข้อ 1.5) — ใช้ได้ทั้ง job ที่
                    ยังรันอยู่ (เห็น log ไหลสด) และที่จบไปแล้ว (เห็น log ย้อนหลังทั้งหมด) */}
                <Link href={`/jobs/${job.id}`} className="text-blue-600 hover:underline">
                  {job.bot_name}
                </Link>
              </td>
              <td className="py-2 pr-4">
                <JobStatusBadge status={job.status} />
              </td>
              <td className="py-2 pr-4 text-gray-600">{job.summary || job.error || "-"}</td>
              <td className="py-2 pr-4 text-gray-500">{new Date(job.created_at).toLocaleString()}</td>
            </tr>
          ))}
        </tbody>
      </table>

      {totalPages > 1 && (
        <div className="flex items-center gap-3 text-sm">
          <button
            type="button"
            disabled={currentPage <= 1}
            onClick={() => setPage((p) => p - 1)}
            className="rounded border border-gray-300 px-3 py-1 disabled:opacity-40"
          >
            ก่อนหน้า
          </button>
          <span>
            หน้า {currentPage} / {totalPages}
          </span>
          <button
            type="button"
            disabled={currentPage >= totalPages}
            onClick={() => setPage((p) => p + 1)}
            className="rounded border border-gray-300 px-3 py-1 disabled:opacity-40"
          >
            ถัดไป
          </button>
        </div>
      )}
    </div>
  );
}
