import { StatusBadge } from "@/components/ui/status-badge";

// ScheduleStatusBadge แสดงแค่ 2 สถานะ (ต่างจาก JobStatusBadge/WorkflowStatusBadge ที่มี
// หลายสถานะ) เพราะ schedule มีแค่ paused หรือไม่ paused เท่านั้น — เขียว = active
// (กำลังรันตามตารางปกติ), เทา = paused (หยุดชั่วคราว เทียบ Control-M "Hold")
export function ScheduleStatusBadge({ paused }: { paused: boolean }) {
  return paused ? <StatusBadge label="paused" tone="gray" /> : <StatusBadge label="active" tone="green" />;
}
