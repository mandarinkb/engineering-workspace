// Catch-up Window คือระยะเวลาย้อนหลังที่ Missed Run ยังถูกรันชดเชยเมื่อระบบกลับมา (หน่วย
// วินาที) — mirror กฎจาก schedule.ParseCatchupWindow ฝั่ง backend: ค่าเริ่มต้น 3600 และ
// ต่ำสุด 10 วินาที (Temporal ไม่ยอมรับค่าที่ต่ำกว่านี้) และสูงสุด 31536000 วินาที (1 ปี)
export const DEFAULT_CATCHUP_WINDOW_SECONDS = 3600;
export const MIN_CATCHUP_WINDOW_SECONDS = 10;
export const MAX_CATCHUP_WINDOW_SECONDS = 31_536_000;

// initialCatchupWindow คืนค่าตั้งต้นของช่องในฟอร์ม — Schedule เดิมที่ไม่เคยตั้ง backend ส่ง
// 0 มา ให้แสดงค่าเริ่มต้นแทน เพราะการบันทึกครั้งนี้จะตั้งค่าเริ่มต้นให้อยู่แล้ว
export function initialCatchupWindow(seconds: number | undefined): string {
  return String(seconds && seconds > 0 ? seconds : DEFAULT_CATCHUP_WINDOW_SECONDS);
}

// validateCatchupWindow คืนข้อความ error หรือ null ถ้าค่าใช้ได้
export function validateCatchupWindow(value: string): string | null {
  const seconds = Number(value);
  const outOfRange = seconds < MIN_CATCHUP_WINDOW_SECONDS || seconds > MAX_CATCHUP_WINDOW_SECONDS;
  if (value.trim() === "" || !Number.isInteger(seconds) || outOfRange) {
    return `Catch-up Window ต้องเป็นจำนวนเต็ม ${MIN_CATCHUP_WINDOW_SECONDS} ถึง ${MAX_CATCHUP_WINDOW_SECONDS} วินาที`;
  }
  return null;
}
