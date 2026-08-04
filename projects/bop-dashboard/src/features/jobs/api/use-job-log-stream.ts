import { useEffect, useState } from "react";

import { connectSSE, type SSEStatus } from "@/lib/sse-client";
import type { LogLine } from "../types";

// useJobLogStream คือ hook หลักของ Live Job Monitor (roadmap ข้อ 1.5 — จุดที่ยากที่สุด
// และมีค่าที่สุดของทั้ง phase) — เปิดเชื่อมต่อ SSE ไปที่ GET /jobs/{id}/logs/stream แล้ว
// สะสม log ทุกบรรทัดที่ได้รับเข้า state
//
// หมายเหตุสำคัญ (อ่านคู่กับ internal/api/http/handler.go: handleStreamJobLogs): endpoint
// SSE นี้ "ส่ง backfill (log เก่าทั้งหมด) มาเป็นชุด event แรกๆ ให้เองอัตโนมัติ" ก่อนจะ
// ส่งต่อด้วย event ใหม่ที่เกิดขึ้นแบบ real-time ต่อท้ายในการเชื่อมต่อเดียวกัน — hook นี้
// จึง "ไม่ต้อง" ยิง GET /jobs/{id}/logs (REST) แยกต่างหากมา backfill เองอีกชั้น (ถ้าทำ
// แบบนั้นจะเห็น log เก่าซ้ำกัน 2 ชุด: ครั้งหนึ่งจาก REST อีกครั้งจาก SSE) แค่เปิด SSE
// อย่างเดียวก็ได้ทั้ง backfill และ live update มาในลำดับที่ถูกต้องอยู่แล้ว
export function useJobLogStream(jobId: string) {
  const [lines, setLines] = useState<LogLine[]>([]);
  const [status, setStatus] = useState<SSEStatus>("connecting");

  useEffect(() => {
    // reset ทุกครั้งที่ jobId เปลี่ยน (เช่น user กด back แล้วเปิด job อื่น) ไม่งั้น log
    // ของ job เก่าจะค้างปนกับ log ของ job ใหม่
    setLines([]);
    setStatus("connecting");

    const disconnect = connectSSE<LogLine>(`/jobs/${jobId}/logs/stream`, {
      onMessage: (line) => setLines((prev) => [...prev, line]),
      onStatusChange: setStatus,
    });

    // cleanup function ของ useEffect — เรียกตอน jobId เปลี่ยน (ก่อนรัน effect รอบใหม่)
    // และตอน component unmount (user ออกจากหน้านี้ไปเลย) ปิด SSE connection เก่าทิ้งเสมอ
    // ป้องกัน connection ค้างอยู่เบื้องหลังทั้งที่ไม่มีใครดูอยู่แล้ว
    return disconnect;
  }, [jobId]);

  return { lines, status };
}
