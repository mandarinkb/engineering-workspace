import { useQuery } from "@tanstack/react-query";

import { apiClient } from "@/lib/api-client";
import { ApiError } from "@/types/api";
import type { User } from "../types";

// useMe เรียก GET /me เพื่อเช็คว่า cookie (JWT httpOnly) ที่เบราว์เซอร์ถืออยู่ตอนนี้ยัง
// valid อยู่ไหม — เป็น query เดียวที่ทั้งแอปใช้ตัดสินว่า user "login อยู่หรือเปล่า" ไม่มี
// การเก็บ auth state แยกต่างหากใน localStorage/context เอง เพราะ source of truth ที่แท้จริง
// คือ cookie ที่ backend ตรวจสอบเท่านั้น (ดู handleMe ใน internal/api/http/handler.go)
export function useMe() {
  return useQuery<User, ApiError>({
    queryKey: ["me"],
    queryFn: () => apiClient<User>("/me"),
    // 401 (ยังไม่ login) เป็นผลลัพธ์ปกติที่คาดหวังได้ ไม่ใช่ error ที่ retry แล้วจะหาย —
    // retry: false ทับ default retry logic ของ query-client.ts (ที่ปกติก็ไม่ retry 4xx
    // อยู่แล้ว แต่ระบุตรงนี้ชัดเจนอีกชั้นเพราะ 401 ของ query นี้เกิดขึ้น "บ่อยมาก" เป็น
    // สถานการณ์ปกติของ user ที่ยังไม่ login เลย ไม่ใช่กรณีพิเศษ)
    retry: false,
  });
}
