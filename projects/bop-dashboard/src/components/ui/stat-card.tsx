import type { BadgeTone } from "./status-badge";

// StatCard คือ card ตัวเลขสรุปสั้นๆ (เช่น "success rate 92%") — เหมือน StatusBadge
// (ไฟล์เดียวกันในโฟลเดอร์นี้) ไม่รู้จัก concept ของ job/bot/workflow เลย รับแค่
// label/value/tone (สีเน้น, optional) ที่แต่ละ feature คำนวณมาเองแล้วส่งเข้ามา ตาม
// กติกาเดียวกัน: src/components/ui ห้าม import จาก src/features
const toneTextClasses: Record<BadgeTone, string> = {
  gray: "text-gray-900 dark:text-gray-100",
  yellow: "text-yellow-700 dark:text-yellow-400",
  green: "text-green-700 dark:text-green-400",
  red: "text-red-700 dark:text-red-400",
};

interface StatCardProps {
  label: string;
  value: string | number;
  tone?: BadgeTone;
}

export function StatCard({ label, value, tone = "gray" }: StatCardProps) {
  return (
    <div className="rounded border border-gray-200 p-4 dark:border-gray-700">
      <p className="text-xs text-gray-500 dark:text-gray-400">{label}</p>
      <p className={`mt-1 text-2xl font-semibold ${toneTextClasses[tone]}`}>{value}</p>
    </div>
  );
}
