import { useMutation, useQueryClient } from "@tanstack/react-query";

import { apiClient } from "@/lib/api-client";

// useLogout เรียก POST /logout (backend ล้าง cookie ทิ้งให้) แล้ว invalidate query "me"
// เหมือนกับ useLogin — ต่างกันแค่ทิศทางผลลัพธ์ (refetch แล้วจะเจอ 401 แทนที่จะเจอ user)
export function useLogout() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => apiClient<void>("/logout", { method: "POST" }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["me"] });
    },
  });
}
