"use client";

import { useEffect, useRef, useState } from "react";

import { useJob } from "../api/use-job";
import { useJobLogStream } from "../api/use-job-log-stream";
import { filterLogLines } from "../lib/log-filters";
import type { LogLevel } from "../types";
import { JobStatusBadge } from "./job-status-badge";
import { useWatchJob } from "./watched-jobs-provider";

const CONNECTION_LABEL: Record<string, string> = {
  connecting: "กำลังเชื่อมต่อ...",
  connected: "เชื่อมต่อแล้ว (real-time)",
  disconnected: "หลุดการเชื่อมต่อ กำลังลองเชื่อมใหม่...",
};

const LEVEL_OPTIONS: Array<{ value: LogLevel | "all"; label: string }> = [
  { value: "all", label: "ทั้งหมด" },
  { value: "info", label: "Info" },
  { value: "warn", label: "Warn" },
  { value: "error", label: "Error" },
];

interface JobLogViewerProps {
  jobId: string;
}

// highlightMatch ห่อคำแรกที่ตรงกับ query ด้วย <mark> — ทำแค่ครั้งแรกที่เจอต่อบรรทัด
// (ไม่ไล่หาทุก match) เพราะ log บรรทัดหนึ่งมักสั้น พอมองเห็นตำแหน่งที่ตรงแรกก็พอ ไม่คุ้ม
// เพิ่มความซับซ้อนของ regex/replaceAll แบบ multiple match สำหรับ use case นี้
function highlightMatch(text: string, query: string) {
  const trimmed = query.trim();
  if (!trimmed) return text;
  const idx = text.toLowerCase().indexOf(trimmed.toLowerCase());
  if (idx === -1) return text;
  return (
    <>
      {text.slice(0, idx)}
      <mark className="bg-yellow-300 text-black">{text.slice(idx, idx + trimmed.length)}</mark>
      {text.slice(idx + trimmed.length)}
    </>
  );
}

// JobLogViewer คือ component ตัวจริงของ Live Job Monitor (roadmap ข้อ 1.5) — รวม 2
// แหล่งข้อมูล: useJob (สถานะปัจจุบันของตัว Job เอง, poll ทุก 2 วินาทีจนกว่าจะจบ) กับ
// useJobLogStream (log แบบ real-time ผ่าน SSE พร้อม backfill ในตัว — ดูคำอธิบายเต็มที่
// use-job-log-stream.ts) แสดงคู่กันเป็นหน้าเดียว
//
// เพิ่มจากเดิม: filter ตาม level, ค้นหา/ไฮไลต์ข้อความ, ปุ่มดาวน์โหลด, toggle
// auto-scroll — filter/search ทำ inline ในตัว render (ไม่ใช้ useMemo) ตาม convention
// เดิมของโปรเจกต์นี้ (ดู job-table.tsx ที่ filter/paginate inline แบบเดียวกัน ไม่มี
// useMemo ที่ไหนในโปรเจกต์เลยตอนนี้)
export function JobLogViewer({ jobId }: JobLogViewerProps) {
  const { data: job } = useJob(jobId);
  const { lines, status } = useJobLogStream(jobId);
  const bottomRef = useRef<HTMLDivElement>(null);

  const [levelFilter, setLevelFilter] = useState<LogLevel | "all">("all");
  const [searchQuery, setSearchQuery] = useState("");
  const [autoScroll, setAutoScroll] = useState(true);

  // ลงทะเบียนให้ WatchedJobsProvider (mount ที่ src/app/providers.tsx) เริ่ม poll
  // สถานะของ job นี้ต่อไปเรื่อยๆ แม้ user จะออกจากหน้านี้ไปแล้วก่อน job จะจบ — เรียกครั้ง
  // เดียวตอน mount พอ ไม่ต้อง cleanup (ดูเหตุผลเต็มที่ watched-jobs-provider.tsx)
  useWatchJob(jobId);

  const filteredLines = filterLogLines(lines, levelFilter, searchQuery);

  // auto-scroll ไปที่บรรทัดล่าสุดทุกครั้งที่จำนวน log เปลี่ยน — ทำงานเฉพาะตอน
  // autoScroll เป็น true เท่านั้น (ผู้ใช้ปิดได้ตอนกำลังอ่าน log เก่าอยู่ ไม่อยากให้จอ
  // เลื่อนหนีตาม)
  useEffect(() => {
    if (!autoScroll) return;
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [lines.length, autoScroll]);

  function handleDownload() {
    const text = lines
      .map((line) => `[${new Date(line.time).toLocaleTimeString()}] ${line.level.toUpperCase()} ${line.message}`)
      .join("\n");
    const blob = new Blob([text], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `job-${jobId}.log.txt`;
    a.click();
    URL.revokeObjectURL(url);
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-3">
        {job && <JobStatusBadge status={job.status} />}
        <span className="text-sm text-gray-500 dark:text-gray-400">{CONNECTION_LABEL[status]}</span>
      </div>

      <div className="flex flex-wrap items-center gap-3">
        <div className="flex gap-1">
          {LEVEL_OPTIONS.map((opt) => (
            <button
              key={opt.value}
              type="button"
              onClick={() => setLevelFilter(opt.value)}
              className={
                levelFilter === opt.value
                  ? "rounded border border-blue-600 bg-blue-50 px-2 py-1 text-xs text-blue-700 dark:border-blue-400 dark:bg-blue-950 dark:text-blue-300"
                  : "rounded border border-gray-300 px-2 py-1 text-xs text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-800"
              }
            >
              {opt.label}
            </button>
          ))}
        </div>

        <input
          type="text"
          placeholder="ค้นหาใน log..."
          value={searchQuery}
          onChange={(event) => setSearchQuery(event.target.value)}
          className="rounded border border-gray-300 px-2 py-1 text-sm focus:border-blue-500 focus:outline-none dark:border-gray-600 dark:bg-gray-900"
        />

        <label className="flex items-center gap-1 text-xs text-gray-600 dark:text-gray-300">
          <input type="checkbox" checked={autoScroll} onChange={(event) => setAutoScroll(event.target.checked)} />
          Auto-scroll
        </label>

        <button
          type="button"
          onClick={handleDownload}
          disabled={lines.length === 0}
          className="rounded border border-gray-300 px-2 py-1 text-xs hover:bg-gray-50 disabled:opacity-40 dark:border-gray-600 dark:hover:bg-gray-800"
        >
          ดาวน์โหลด .txt
        </button>
      </div>

      <div className="h-96 overflow-y-auto rounded border border-gray-200 bg-gray-950 p-4 font-mono text-sm text-gray-100">
        {filteredLines.length === 0 ? (
          <p className="text-gray-500">{lines.length === 0 ? "ยังไม่มี log" : "ไม่มี log ที่ตรงกับตัวกรอง"}</p>
        ) : (
          // ใช้ index เป็น key ตรงนี้ปลอดภัย (ต่างจากกรณีทั่วไปที่ควรเลี่ยง) เพราะ lines
          // เป็น "append-only list" ล้วนๆ (มีแต่เพิ่มต่อท้าย ไม่มีการแทรก/ลบ/สลับลำดับ
          // กลางทางเลย) index ของแต่ละบรรทัดจึงคงที่ตลอดอายุของมันเสมอ ไม่เกิดปัญหา
          // React reuse component ผิดตัวแบบที่มักเกิดกับ list ที่ reorder ได้
          filteredLines.map((line, i) => (
            <div
              key={i}
              className={
                line.level === "error" ? "text-red-400" : line.level === "warn" ? "text-yellow-400" : undefined
              }
            >
              <span className="text-gray-500">{new Date(line.time).toLocaleTimeString()}</span>{" "}
              {highlightMatch(line.message, searchQuery)}
            </div>
          ))
        )}
        {/* div ว่างเปล่าตัวนี้คือ "จุดหมาย" ที่ scrollIntoView เลื่อนไปหา — trick มาตรฐาน
            สำหรับ auto-scroll chat/log viewer: มี element ว่างอยู่ท้ายสุดเสมอ แล้วสั่ง
            scroll ไปหา element นั้นแทนที่จะคำนวณตำแหน่ง scroll เอง */}
        <div ref={bottomRef} />
      </div>
    </div>
  );
}
