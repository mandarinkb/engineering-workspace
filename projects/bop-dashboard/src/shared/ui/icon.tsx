// ไอคอนเป็น inline SVG (stroke ตาม currentColor) แทนการเพิ่ม icon library ทั้งก้อนเข้ามา
// เพื่อใช้แค่ไม่กี่ตัว — สีเปลี่ยนตาม text color ของ element แม่ (active/hover) เอง
export function Icon({ d }: { d: string }) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.8}
      strokeLinecap="round"
      strokeLinejoin="round"
      className="h-5 w-5 shrink-0"
      aria-hidden="true"
    >
      <path d={d} />
    </svg>
  );
}
