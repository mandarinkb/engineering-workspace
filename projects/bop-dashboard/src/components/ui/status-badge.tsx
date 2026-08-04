// BadgeTone คือชุดสีที่ StatusBadge รองรับ — เป็น component กลางที่ไม่รู้จัก concept ของ
// "JobStatus" หรือ "workflow status" เลย (ไม่ผูก business logic ของ feature ไหนเลยตาม
// กติกาที่ PHASE1-DASHBOARD-TODO.md วางไว้: src/components/ui ห้าม import อะไรจาก
// src/features) รับแค่ tone (สี) กับ label (ข้อความ) ที่แต่ละ feature คำนวณมาเองแล้ว
export type BadgeTone = "gray" | "yellow" | "green" | "red";

const toneClasses: Record<BadgeTone, string> = {
  gray: "bg-gray-100 text-gray-700",
  yellow: "bg-yellow-100 text-yellow-800",
  green: "bg-green-100 text-green-800",
  red: "bg-red-100 text-red-800",
};

interface StatusBadgeProps {
  label: string;
  tone: BadgeTone;
}

// StatusBadge คือ badge สีตาม roadmap ข้อ 1.7 (running=เหลือง, succeeded=เขียว,
// failed=แดง) — ใช้ร่วมกันได้ทั้งหน้า job history/live monitor (ข้อ 1.4/1.5) และหน้า
// workflow run (ดู features/jobs/components/job-status-badge.tsx และ
// features/workflow-runs/components/workflow-status-badge.tsx ที่ map status string
// ของแต่ละ feature มาเป็น tone/label แล้วส่งเข้ามาที่นี่)
export function StatusBadge({ label, tone }: StatusBadgeProps) {
  return (
    <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${toneClasses[tone]}`}>
      {label}
    </span>
  );
}
