"use client";

import Link from "next/link";

import { StatCard } from "@/shared/ui";
import { useJobs } from "@/entities/job";
import { useSchedules } from "@/entities/schedule";
import { useWorkflowRunCount, useWorkflowRunPage, WorkflowStatusBadge } from "@/entities/workflow-run";

import { computeOverviewStats } from "../lib/overview-stats";

// RECENT_FAILED_LIMIT คือจำนวน Workflow Run ที่ล้มเหลวล่าสุดที่แสดงใน panel (ขอเป็น page_size
// ตรง ๆ ไม่ต้องตัดเองฝั่ง client)
const RECENT_FAILED_LIMIT = 5;

// Overview คือหน้าแรกของ dashboard — สรุปสถานะจาก /jobs, /schedules และ /workflow-runs
// ไว้ในหน้าเดียว ใช้ query key เดียวกับหน้าอื่น (useJobs/useSchedules) จึงแชร์ cache กัน
// กดเข้าไปหน้า Jobs ต่อจะไม่ต้องโหลดใหม่ — workflow run นับและกรองฝั่ง server ทั้งคู่ ตัวเลข
// จึงถูกต้องไม่ว่าจะมี run กี่หน้า: "กำลังรัน" มาจาก GET /workflow-runs/count?status=running
// และรายการล้มเหลวคือหน้าแรกของ status=failed ขนาด 5 (backend เรียงใหม่สุดก่อนให้แล้ว)
export function Overview() {
  const jobs = useJobs();
  const schedules = useSchedules();
  const runningRuns = useWorkflowRunCount("running");
  const failedRuns = useWorkflowRunPage({ status: "failed", pageSize: RECENT_FAILED_LIMIT });

  const failedSources = [
    jobs.isError && "jobs",
    schedules.isError && "schedules",
    (runningRuns.isError || failedRuns.isError) && "workflow runs",
  ].filter(Boolean);
  const loading = jobs.isLoading || schedules.isLoading || runningRuns.isLoading || failedRuns.isLoading;

  const stats = computeOverviewStats(jobs.data ?? [], schedules.data ?? [], new Date());
  const runningRunCount = runningRuns.data?.count ?? 0;
  const recentFailedRuns = failedRuns.data?.runs ?? [];
  const rate = stats.successRate7dPercent;

  return (
    <div className="flex flex-col gap-6">
      {failedSources.length > 0 && (
        <p className="rounded-lg border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-300">
          โหลดข้อมูล {failedSources.join(", ")} ไม่สำเร็จ ตัวเลขด้านล่างอาจไม่ครบ
        </p>
      )}

      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <StatCard label="Job กำลังรัน" value={loading ? "…" : stats.runningJobs} tone={stats.runningJobs > 0 ? "yellow" : "gray"} />
        <StatCard label="Job ล้มเหลววันนี้" value={loading ? "…" : stats.failedJobsToday} tone={stats.failedJobsToday > 0 ? "red" : "gray"} />
        <StatCard
          label="Success rate 7 วัน"
          value={loading ? "…" : rate === null ? "-" : `${rate}%`}
          tone={rate === null ? "gray" : rate >= 80 ? "green" : "red"}
        />
        <StatCard label="Workflow กำลังรัน" value={loading ? "…" : runningRunCount} tone={runningRunCount > 0 ? "yellow" : "gray"} />
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <Panel title="Schedule ที่จะรันถัดไป" subtitle={`active ${stats.activeSchedules} ตัว`} moreHref="/schedules">
          {stats.upcomingSchedules.length === 0 ? (
            <Empty loading={loading}>ไม่มี schedule ที่รอรัน</Empty>
          ) : (
            stats.upcomingSchedules.map(({ schedule, nextRunAt }) => (
              <Row key={schedule.id} href={`/schedules/${encodeURIComponent(schedule.id)}`}>
                <span className="truncate font-medium">{schedule.id}</span>
                <span className="shrink-0 text-gray-500 dark:text-gray-400">{new Date(nextRunAt).toLocaleString()}</span>
              </Row>
            ))
          )}
        </Panel>

        <Panel title="Workflow run ที่ล้มเหลวล่าสุด" moreHref="/workflow-runs">
          {recentFailedRuns.length === 0 ? (
            <Empty loading={loading}>ไม่มี workflow run ที่ล้มเหลว</Empty>
          ) : (
            recentFailedRuns.map((run) => (
              <Row key={`${run.workflow_id}/${run.run_id}`} href={`/workflow-runs/${encodeURIComponent(run.workflow_id)}`}>
                <span className="flex min-w-0 items-center gap-2">
                  <WorkflowStatusBadge status={run.status} />
                  <span className="truncate font-medium">{run.workflow_id}</span>
                </span>
                <span className="shrink-0 text-gray-500 dark:text-gray-400">{new Date(run.started_at).toLocaleString()}</span>
              </Row>
            ))
          )}
        </Panel>
      </div>
    </div>
  );
}

function Panel({ title, subtitle, moreHref, children }: { title: string; subtitle?: string; moreHref: string; children: React.ReactNode }) {
  return (
    <section className="rounded-lg border border-gray-200 dark:border-gray-700">
      <header className="flex items-center justify-between border-b border-gray-200 px-4 py-3 dark:border-gray-700">
        <div>
          <h2 className="font-medium">{title}</h2>
          {subtitle && <p className="text-xs text-gray-500 dark:text-gray-400">{subtitle}</p>}
        </div>
        <Link href={moreHref} className="text-sm text-blue-600 hover:underline dark:text-blue-400">
          ดูทั้งหมด
        </Link>
      </header>
      <ul className="divide-y divide-gray-100 dark:divide-gray-800">{children}</ul>
    </section>
  );
}

function Row({ href, children }: { href: string; children: React.ReactNode }) {
  return (
    <li>
      <Link href={href} className="flex items-center justify-between gap-3 px-4 py-2.5 text-sm hover:bg-gray-50 dark:hover:bg-gray-800/50">
        {children}
      </Link>
    </li>
  );
}

function Empty({ loading, children }: { loading: boolean; children: React.ReactNode }) {
  return <li className="px-4 py-6 text-center text-sm text-gray-500 dark:text-gray-400">{loading ? "กำลังโหลด..." : children}</li>;
}
