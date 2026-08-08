import type { LogLevel, LogLine } from "../types";

// filterLogLines คือ pure function แยกออกมาจาก job-log-viewer.tsx ตั้งใจ เพื่อให้ test
// ได้โดยไม่ต้อง render component จริง (ไม่ต้อง mock SSE/fetch) — กรอง 2 ชั้น: ตาม level
// (ถ้าเลือก "all" ไม่กรองเลย) แล้วตาม query (case-insensitive substring match บน
// message) ไม่กรอง query ถ้าเป็นช่องว่างล้วนๆ (trim แล้วว่างเปล่า = ไม่กรอง)
export function filterLogLines(lines: LogLine[], level: LogLevel | "all", query: string): LogLine[] {
  const normalizedQuery = query.trim().toLowerCase();
  return lines.filter((line) => {
    if (level !== "all" && line.level !== level) return false;
    if (normalizedQuery && !line.message.toLowerCase().includes(normalizedQuery)) return false;
    return true;
  });
}
