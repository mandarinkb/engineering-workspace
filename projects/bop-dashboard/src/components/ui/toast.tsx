// Toast คือ presentational component ของ toast เดียว — ไม่มี state/timer ของตัวเองเลย
// (auto-dismiss จัดการอยู่ที่ ToastProvider แทน) รับแค่ message/tone/onClose ตาม pattern
// เดียวกับ StatusBadge/StatCard ในโฟลเดอร์เดียวกัน (ห้าม import จาก src/features)
export type ToastTone = "info" | "success" | "error";

const toneClasses: Record<ToastTone, string> = {
  info: "bg-gray-900 text-white",
  success: "bg-green-700 text-white",
  error: "bg-red-700 text-white",
};

interface ToastProps {
  message: string;
  tone: ToastTone;
  onClose: () => void;
}

export function Toast({ message, tone, onClose }: ToastProps) {
  return (
    <div
      role="alert"
      className={`flex items-center gap-3 rounded px-4 py-3 text-sm shadow-lg ${toneClasses[tone]}`}
    >
      <span>{message}</span>
      <button
        type="button"
        onClick={onClose}
        aria-label="ปิดการแจ้งเตือน"
        className="text-white/70 hover:text-white"
      >
        ✕
      </button>
    </div>
  );
}
