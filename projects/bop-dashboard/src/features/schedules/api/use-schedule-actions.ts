import { useMutation, useQueryClient } from "@tanstack/react-query";

import { apiClient } from "@/lib/api-client";
import type { OverlapPolicy } from "../types";

interface BackfillInput {
  start_time: string;
  end_time: string;
  overlap?: OverlapPolicy;
}

// useScheduleActions รวม mutation ของปุ่ม action ทั้งหมดที่ "สั่งงาน schedule ที่มีอยู่
// แล้ว" ไว้ hook เดียว (pause/unpause/trigger/backfill) — ต่างจาก create/update/delete
// ที่แยกไฟล์ต่างหากเพราะเป็น CRUD หลัก ส่วน 4 ปุ่มนี้เป็น action เสริมที่อยู่ในหน้า
// เดียวกันเสมอ (schedule-detail.tsx) และ invalidate query แบบเดียวกันหมด — trigger
// (สั่งรันทันที นอกตารางเวลาปกติ) คือปุ่ม "Run Now" ที่เทียบเท่า Control-M "Order Now"
export function useScheduleActions(id: string) {
  const queryClient = useQueryClient();

  const invalidate = () => {
    // pause/unpause เปลี่ยน paused state ที่ทั้งหน้า list (GET /schedules) และหน้า
    // detail (GET /schedules/{id}) แสดงอยู่ ต้อง invalidate ทั้งคู่เสมอ ไม่ใช่แค่
    // detail — trigger/backfill เองไม่กระทบ paused state แต่ invalidate เหมือนกันไว้
    // เผื่อ next_action_times/recent_actions เปลี่ยนตาม ไม่ต้องแยก logic ต่อ action
    queryClient.invalidateQueries({ queryKey: ["schedules"] });
    queryClient.invalidateQueries({ queryKey: ["schedules", id] });
  };

  const pause = useMutation({
    mutationFn: (note?: string) =>
      apiClient<void>(`/schedules/${encodeURIComponent(id)}/pause`, { method: "POST", body: { note } }),
    onSuccess: invalidate,
  });

  const unpause = useMutation({
    mutationFn: (note?: string) =>
      apiClient<void>(`/schedules/${encodeURIComponent(id)}/unpause`, { method: "POST", body: { note } }),
    onSuccess: invalidate,
  });

  const trigger = useMutation({
    mutationFn: (overlap?: OverlapPolicy) =>
      apiClient<void>(`/schedules/${encodeURIComponent(id)}/trigger`, { method: "POST", body: { overlap } }),
    onSuccess: invalidate,
  });

  const backfill = useMutation({
    mutationFn: (input: BackfillInput) =>
      apiClient<void>(`/schedules/${encodeURIComponent(id)}/backfill`, { method: "POST", body: input }),
    onSuccess: invalidate,
  });

  return { pause, unpause, trigger, backfill };
}
