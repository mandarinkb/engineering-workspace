"use client";

import { useEffect, useRef } from "react";

import { useJob } from "../api/use-job";
import { useJobLogStream } from "../api/use-job-log-stream";
import { JobStatusBadge } from "./job-status-badge";

const CONNECTION_LABEL: Record<string, string> = {
  connecting: "กำลังเชื่อมต่อ...",
  connected: "เชื่อมต่อแล้ว (real-time)",
  disconnected: "หลุดการเชื่อมต่อ กำลังลองเชื่อมใหม่...",
};

interface JobLogViewerProps {
  jobId: string;
}

// JobLogViewer คือ component ตัวจริงของ Live Job Monitor (roadmap ข้อ 1.5) — รวม 2
// แหล่งข้อมูล: useJob (สถานะปัจจุบันของตัว Job เอง, poll ทุก 2 วินาทีจนกว่าจะจบ) กับ
// useJobLogStream (log แบบ real-time ผ่าน SSE พร้อม backfill ในตัว — ดูคำอธิบายเต็มที่
// use-job-log-stream.ts) แสดงคู่กันเป็นหน้าเดียว
export function JobLogViewer({ jobId }: JobLogViewerProps) {
  const { data: job } = useJob(jobId);
  const { lines, status } = useJobLogStream(jobId);
  const bottomRef = useRef<HTMLDivElement>(null);

  // auto-scroll ไปที่บรรทัดล่าสุดทุกครั้งที่จำนวน log เปลี่ยน (มี line ใหม่เข้ามา) — ใช้
  // lines.length เป็น dependency แทนที่จะเป็น lines ทั้ง array เอง เพราะ array reference
  // เปลี่ยนทุกครั้งที่ setLines ถูกเรียกอยู่แล้ว (ทำให้ effect รันซ้ำเท่ากันไม่ว่าจะดู
  // length หรือดู array ตรงๆ) แต่ .length เป็น primitive number เทียบง่ายกว่าและสื่อ
  // เจตนาชัดกว่า: "scroll ทุกครั้งที่จำนวนบรรทัดเพิ่มขึ้น" ไม่ใช่ "scroll ทุกครั้งที่
  // อะไรก็ตามใน array เปลี่ยน"
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [lines.length]);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-3">
        {job && <JobStatusBadge status={job.status} />}
        <span className="text-sm text-gray-500">{CONNECTION_LABEL[status]}</span>
      </div>

      <div className="h-96 overflow-y-auto rounded border border-gray-200 bg-gray-950 p-4 font-mono text-sm text-gray-100">
        {lines.length === 0 ? (
          <p className="text-gray-500">ยังไม่มี log</p>
        ) : (
          // ใช้ index เป็น key ตรงนี้ปลอดภัย (ต่างจากกรณีทั่วไปที่ควรเลี่ยง) เพราะ lines
          // เป็น "append-only list" ล้วนๆ (มีแต่เพิ่มต่อท้าย ไม่มีการแทรก/ลบ/สลับลำดับ
          // กลางทางเลย) index ของแต่ละบรรทัดจึงคงที่ตลอดอายุของมันเสมอ ไม่เกิดปัญหา
          // React reuse component ผิดตัวแบบที่มักเกิดกับ list ที่ reorder ได้
          lines.map((line, i) => (
            <div
              key={i}
              className={
                line.level === "error" ? "text-red-400" : line.level === "warn" ? "text-yellow-400" : undefined
              }
            >
              <span className="text-gray-500">{new Date(line.time).toLocaleTimeString()}</span> {line.message}
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
