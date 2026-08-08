"use client";

import { StatCard } from "@/components/ui/stat-card";

import { useJobs } from "../api/use-jobs";
import { computeJobStats } from "../lib/job-stats";

// JobStats แสดง stat card สรุปด้านบนหน้า job history — เรียก useJobs() เอง (เหมือน
// JobTable) แทนที่จะรับ jobs เป็น prop จาก page.tsx เพราะ page.tsx เป็น Server
// Component (ไม่มี "use client") เรียก hook ฝั่ง client ไม่ได้เลย — TanStack Query
// dedupe request ให้เองอัตโนมัติอยู่แล้วถ้า queryKey ตรงกัน (["jobs"] เดียวกับที่
// JobTable ใช้) จึงไม่ได้ยิง request ซ้ำสองครั้งจริงๆ แค่ใช้ cache เดียวกันคนละจุดเรียก
export function JobStats() {
  const { data: jobs } = useJobs();
  const stats = computeJobStats(jobs ?? []);

  return (
    <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
      <StatCard label="Job ทั้งหมด" value={stats.total} />
      <StatCard
        label="Success Rate"
        value={stats.successRatePercent === null ? "-" : `${stats.successRatePercent}%`}
        tone={stats.successRatePercent === null ? "gray" : stats.successRatePercent >= 80 ? "green" : "red"}
      />
      <StatCard label="วันนี้" value={stats.todayCount} />
      <StatCard label="ล้มเหลวทั้งหมด" value={stats.failed} tone={stats.failed > 0 ? "red" : "gray"} />
    </div>
  );
}
