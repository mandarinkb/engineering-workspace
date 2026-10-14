import { useMutation, useQueryClient } from "@tanstack/react-query";

import { apiClient } from "@/lib/api-client";

// useCancelJob เรียก DELETE /jobs/{id} — สั่งยกเลิก job ที่กำลังรันอยู่ (backend คืน 204
// ถ้าสำเร็จ, 409 Conflict ถ้า job นั้นไม่ได้กำลังรันอยู่แล้ว ดู handleCancelJob ใน
// internal/api/http/handler.go) ใช้ทั้ง cancel เดี่ยวและ bulk cancel (เรียกหลายครั้งพร้อม
// กันผ่าน mutateAsync วน id ที่เลือกไว้ ดู job-table.tsx — ไม่มี bulk endpoint ฝั่ง
// backend ให้เรียกทีเดียว)
export function useCancelJob() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => apiClient<void>(`/jobs/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["jobs"] });
    },
  });
}
