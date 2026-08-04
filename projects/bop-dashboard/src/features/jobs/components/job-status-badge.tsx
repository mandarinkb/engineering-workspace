import { StatusBadge, type BadgeTone } from "@/components/ui/status-badge";
import type { JobStatus } from "../types";

// toneByStatus แปลง JobStatus (จาก Go: pending/running/succeeded/failed/cancelled) เป็นสี
// ตามที่ roadmap ข้อ 1.7 กำหนดไว้: running=เหลือง, succeeded=เขียว, failed=แดง — pending
// กับ cancelled ยังไม่ได้ระบุสีไว้ใน roadmap ตรงๆ เลือกให้เป็น gray เพราะเป็นสถานะ
// "ไม่ active" ทั้งคู่ (ยังไม่เริ่ม / เลิกไปแล้ว) ต่างจาก running ที่ active อยู่จริง
const toneByStatus: Record<JobStatus, BadgeTone> = {
  pending: "gray",
  running: "yellow",
  succeeded: "green",
  failed: "red",
  cancelled: "gray",
};

export function JobStatusBadge({ status }: { status: JobStatus }) {
  return <StatusBadge label={status} tone={toneByStatus[status]} />;
}
