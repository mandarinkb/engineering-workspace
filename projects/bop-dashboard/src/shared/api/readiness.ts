import { API_URL } from "./api-client";

// Readiness คือสถานะของ BOP API จาก GET /readyz:
//   ready        = API ตอบ 200 (ต่อ Temporal ได้)
//   temporal_down = API ตอบแต่ไม่ใช่ 2xx (backend ตอบ 503 "temporal unreachable")
//   api_down     = เรียก API ไม่ติดเลย (container ไม่รัน / network)
export type Readiness = "ready" | "temporal_down" | "api_down";

// checkReadiness ไม่ใช้ apiClient เพราะ /readyz ตอบ 200 แบบ body ว่าง (apiClient จะพัง
// ตอน res.json()) และต้องแยก "API ล่ม" ออกจาก "API ตอบว่า Temporal ล่ม" ซึ่ง apiClient
// รวมเป็น ApiError เหมือนกัน — /readyz ไม่ต้อง login จึงไม่ต้องส่ง cookie
export async function checkReadiness(): Promise<Readiness> {
  try {
    const res = await fetch(`${API_URL}/readyz`, { cache: "no-store" });
    return res.ok ? "ready" : "temporal_down";
  } catch {
    return "api_down";
  }
}
