"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useState } from "react";

import { useJobs } from "../api/use-jobs";
import type { JobStatus } from "../types";
import { JobStatusBadge } from "./job-status-badge";

const PAGE_SIZE = 20;

// JobTable คือหน้าหลักของ roadmap ข้อ 1.4 — ดึง job ทั้งหมดผ่าน useJobs (TanStack Query
// cache/retry อัตโนมัติให้แล้ว ดู src/lib/query-client.ts) แล้วกรอง+แบ่งหน้าฝั่ง client
// เอง (ไม่พึ่ง backend ทำ filter/pagination เพราะ endpoint GET /jobs ยังไม่รองรับ query
// parameter พวกนี้ และจำนวน job ทั้งระบบตอนนี้ยังน้อยพอที่ client-side filter จะไม่ช้า)
//
// ตัวกรอง (bot/status) มาจาก URL query param โดยตรง (useSearchParams) ไม่ใช่ state ของ
// component นี้เอง — sync กับ job-filter.tsx (คนละไฟล์ที่เขียนค่าลง URL เดียวกัน) ผ่าน URL
// เป็นตัวกลาง ส่วน page (หน้าที่กำลังดูอยู่) ยังเป็น useState ธรรมดา เพราะไม่จำเป็นต้อง
// sync ข้าม URL (ไม่มีใครอยากแชร์ลิงก์ไปที่ "หน้า 3" ของอีกคนพอดี)
export function JobTable() {
  const { data: jobs, isLoading, isError, error } = useJobs();
  const searchParams = useSearchParams();
  const [page, setPage] = useState(1);

  if (isLoading) {
    return <p className="text-gray-500">กำลังโหลด job history...</p>;
  }
  if (isError) {
    return <p className="text-red-600">โหลด job history ไม่สำเร็จ: {error.message}</p>;
  }

  const botFilter = searchParams.get("bot")?.toLowerCase() ?? "";
  const statusFilter = (searchParams.get("status") as JobStatus | "" | null) ?? "";

  const filtered = (jobs ?? []).filter((job) => {
    if (botFilter && !job.bot_name.toLowerCase().includes(botFilter)) return false;
    if (statusFilter && job.status !== statusFilter) return false;
    return true;
  });

  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE));
  const currentPage = Math.min(page, totalPages);
  const rows = filtered.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE);

  if (filtered.length === 0) {
    return <p className="text-gray-500">ยังไม่มี job ที่ตรงกับตัวกรอง</p>;
  }

  return (
    <div className="flex flex-col gap-3">
      <table className="w-full border-collapse text-left text-sm">
        <thead>
          <tr className="border-b border-gray-200 text-gray-500">
            <th className="py-2 pr-4">Bot</th>
            <th className="py-2 pr-4">Status</th>
            <th className="py-2 pr-4">Summary</th>
            <th className="py-2 pr-4">เริ่มเมื่อ</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((job) => (
            <tr key={job.id} className="border-b border-gray-100 hover:bg-gray-50">
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
