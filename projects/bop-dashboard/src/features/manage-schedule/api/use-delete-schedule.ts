import { useMutation, useQueryClient } from "@tanstack/react-query";

import { apiClient } from "@/lib/api-client";

// useDeleteSchedule เรียก DELETE /schedules/{id} — ลบถาวรทั้งฝั่ง Temporal และ
// definition ของ backend เอง (ดู handleDeleteSchedule) ไม่มีทาง undo
export function useDeleteSchedule() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => apiClient<void>(`/schedules/${encodeURIComponent(id)}`, { method: "DELETE" }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["schedules"] });
    },
  });
}
