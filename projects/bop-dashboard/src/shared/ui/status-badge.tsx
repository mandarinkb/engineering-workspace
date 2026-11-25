// BadgeTone คือชุดสีที่ StatusBadge รองรับ — เป็น component กลางที่ไม่รู้จัก concept ของ
// "JobStatus" หรือ "workflow status" เลย (ไม่ผูก business logic ของ feature ไหนเลยตาม
// กติกาที่ PHASE1-DASHBOARD-TODO.md วางไว้: src/shared/ui ห้าม import อะไรจาก
// src/features) รับแค่ tone (สี) กับ label (ข้อความ) ที่แต่ละ feature คำนวณมาเองแล้ว
export type BadgeTone = "gray" | "yellow" | "green" | "red";

// พื้นอ่อน + ขอบสีเข้ม + ตัวพิมพ์ใหญ่ แบบป้าย status ของ Temporal UI (สีมาจากพาเล็ตต์ของ
// Temporal: slate/green/amber/red ขั้น 3-5 สำหรับพื้น-ขอบ และขั้น 12 สำหรับตัวอักษร)
const toneClasses: Record<BadgeTone, string> = {
  gray: "border-[#d4d3cd] bg-[#f0efec] text-[#1e1d1b] dark:border-gray-600 dark:bg-white/10 dark:text-gray-300",
  yellow: "border-[#f3d673] bg-[#fff7c2] text-[#4f3422] dark:border-yellow-500/40 dark:bg-yellow-500/15 dark:text-yellow-300",
  green: "border-[#adddc0] bg-[#e6f6eb] text-[#193b2d] dark:border-green-500/40 dark:bg-green-500/15 dark:text-green-300",
  red: "border-[#fdbdbe] bg-[#feebec] text-[#641723] dark:border-red-500/40 dark:bg-red-500/15 dark:text-red-300",
};

interface StatusBadgeProps {
  label: string;
  tone: BadgeTone;
}

// StatusBadge คือ badge สีตาม roadmap ข้อ 1.7 (running=เหลือง, succeeded=เขียว,
// failed=แดง) — ใช้ร่วมกันได้ทั้งหน้า job history/live monitor (ข้อ 1.4/1.5) และหน้า
// workflow run (ดู entities/job/ui/job-status-badge.tsx และ
// entities/workflow-run/ui/workflow-status-badge.tsx ที่ map status string
// ของแต่ละ feature มาเป็น tone/label แล้วส่งเข้ามาที่นี่)
export function StatusBadge({ label, tone }: StatusBadgeProps) {
  return (
    <span className={`inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-medium uppercase ${toneClasses[tone]}`}>
      {label}
    </span>
  );
}
