"use client";

import { createContext, useCallback, useContext, useEffect, useRef, useState } from "react";

import { useToast } from "@/components/ui/toast-provider";
import { apiClient } from "@/lib/api-client";
import type { Job, JobStatus } from "../types";

const POLL_INTERVAL_MS = 3000;

const TERMINAL_STATUSES: JobStatus[] = ["succeeded", "failed", "cancelled"];

interface WatchedJobsContextValue {
  watch: (jobId: string) => void;
}

const WatchedJobsContext = createContext<WatchedJobsContextValue | null>(null);

const STATUS_LABEL_TH: Record<JobStatus, string> = {
  pending: "รอคิว",
  running: "กำลังทำงาน",
  succeeded: "สำเร็จ",
  failed: "ล้มเหลว",
  cancelled: "ถูกยกเลิก",
};

// WatchedJobsProvider แก้ปัญหา "อยากรู้ว่า job ที่เพิ่งเปิดดูอยู่เปลี่ยนสถานะไหม แม้จะ
// กดออกไปหน้าอื่นแล้วก็ตาม" — ตั้งใจ**ไม่ใช้ SSE** แม้ backend จะมี
// GET /jobs/{id}/logs/stream อยู่แล้ว เพราะ endpoint นั้น stream แค่ "log ของ job เดียว"
// ไม่ใช่ "สถานะเปลี่ยนของหลาย job พร้อมกัน" และ SSE connection เดิม (useJobLogStream)
// ผูกอายุไว้กับ lifecycle ของหน้า job detail เท่านั้น (ปิดตอน component unmount) —
// ถ้าจะเปิดใหม่ทุกครั้งที่ user เข้าหน้า job detail แล้วให้มันรอดไปหลังออกจากหน้าด้วย
// จะซับซ้อนกว่า poll ธรรมดาเยอะ (ต้อง reparent EventSource ข้าม component tree) จึง
// เลือก**poll `GET /jobs/{id}` ทุก 3 วิ ผ่าน provider ที่ mount อยู่ที่ root** แทน
// (src/app/providers.tsx) — provider ไม่ unmount ตาม route เปลี่ยน จึง poll ต่อเนื่องได้
// จริงแม้ user จะย้ายไปหน้าอื่นแล้วก็ตาม จนกว่า job นั้นจะจบ (ถึงจะเลิก poll เองอัตโนมัติ)
//
// เรียกผ่าน hook useWatchJob(jobId) จากหน้าที่เปิดดู job (เช่น JobLogViewer) แค่ตอน
// mount ครั้งเดียว — "ลงทะเบียน" ให้ provider เริ่ม poll job นั้น ตัว hook เองไม่ต้อง
// เก็บ state หรือ poll เองเลย ปล่อยให้ provider เป็นเจ้าของ lifecycle ทั้งหมด
export function WatchedJobsProvider({ children }: { children: React.ReactNode }) {
  const { showToast } = useToast();
  const [watchedIds, setWatchedIds] = useState<Set<string>>(new Set());
  // lastKnownStatus เก็บสถานะล่าสุดที่รู้ต่อ job — ต้องเป็น ref ไม่ใช่ state เพราะแค่ใช้
  // เทียบค่าภายใน interval callback เท่านั้น ไม่ต้องกระตุ้น re-render เวลาค่าเปลี่ยน
  const lastKnownStatus = useRef<Map<string, JobStatus>>(new Map());

  const watch = useCallback((jobId: string) => {
    setWatchedIds((prev) => (prev.has(jobId) ? prev : new Set(prev).add(jobId)));
  }, []);

  useEffect(() => {
    if (watchedIds.size === 0) return;

    const interval = window.setInterval(async () => {
      for (const jobId of watchedIds) {
        let job: Job;
        try {
          job = await apiClient<Job>(`/jobs/${jobId}`);
        } catch {
          continue; // job หาไม่เจอ/network error ชั่วคราว — ลองใหม่รอบหน้า ไม่ลบออกจากรายการเฝ้าดู
        }

        const previous = lastKnownStatus.current.get(jobId);
        lastKnownStatus.current.set(jobId, job.status);

        // แจ้งเตือนเฉพาะตอน "เปลี่ยนจากสถานะที่ยังไม่จบ ไปเป็นสถานะที่จบแล้ว" เท่านั้น
        // (ไม่แจ้งซ้ำทุกรอบ poll ถ้าสถานะเดิมไม่เปลี่ยน และไม่แจ้งตอนเพิ่งเริ่ม watch
        // ครั้งแรกที่ previous ยังเป็น undefined อยู่ ถ้า job จบไปแล้วตั้งแต่ก่อนเริ่ม watch)
        if (previous && previous !== job.status && TERMINAL_STATUSES.includes(job.status)) {
          showToast({
            message: `Job ${job.bot_name} (${jobId.slice(0, 8)}): ${STATUS_LABEL_TH[job.status]}`,
            tone: job.status === "succeeded" ? "success" : job.status === "failed" ? "error" : "info",
          });
          setWatchedIds((prev) => {
            const next = new Set(prev);
            next.delete(jobId);
            return next;
          });
        }
      }
    }, POLL_INTERVAL_MS);

    return () => window.clearInterval(interval);
  }, [watchedIds, showToast]);

  return <WatchedJobsContext.Provider value={{ watch }}>{children}</WatchedJobsContext.Provider>;
}

// useWatchJob คือ hook ที่หน้า job detail เรียกใช้ (ไม่ใช่ useEffect เอง เพราะ "การเริ่ม
// เฝ้าดู" ต้องเกิดแค่ครั้งเดียวตอน mount ไม่ต้อง cleanup ตอน unmount ด้วยซ้ำ — ตรงข้ามกับ
// SSE connection ทั่วไปที่ปิดตอน unmount เสมอ จุดต่างนี้คือหัวใจของฟีเจอร์นี้เลย)
export function useWatchJob(jobId: string): void {
  const ctx = useContext(WatchedJobsContext);
  useEffect(() => {
    ctx?.watch(jobId);
  }, [ctx, jobId]);
}
