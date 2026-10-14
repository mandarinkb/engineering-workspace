import { useQuery } from "@tanstack/react-query";

import { apiClient } from "@/lib/api-client";
import type { Bot } from "../types";

// useBots เรียก GET /bots — เป็น query เดียวที่ทั้งหน้า bot list (roadmap ข้อ 1.2) และ
// bot-form.tsx (ข้อ 1.3) ใช้ร่วมกัน (bot-form อ่าน config_schema ของ bot ที่ user เลือก
// จาก response เดียวกันนี้ ไม่ได้ยิง request แยกต่างหาก)
export function useBots() {
  return useQuery({
    queryKey: ["bots"],
    queryFn: () => apiClient<Bot[]>("/bots"),
  });
}
