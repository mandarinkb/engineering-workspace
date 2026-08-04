import { useMutation, useQueryClient } from "@tanstack/react-query";

import { apiClient } from "@/lib/api-client";

interface LoginInput {
  username: string;
  password: string;
}

// useLogin เรียก POST /login — เมื่อสำเร็จ backend จะฝัง JWT ใน httpOnly cookie ให้เอง
// ผ่าน Set-Cookie header (ดู handleLogin ใน internal/api/http/handler.go) โค้ดฝั่งนี้จึง
// ไม่ต้องรับ/เก็บ token อะไรเองเลย แค่รอ mutation สำเร็จแล้วสั่ง invalidate query "me"
// (useMe) ให้ refetch ทันที — refetch ครั้งนี้จะเห็นว่า login แล้วเพราะ cookie ใหม่ถูก
// แนบไปกับ request อัตโนมัติ (credentials: "include" ใน api-client.ts)
export function useLogin() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: LoginInput) => apiClient<void>("/login", { method: "POST", body: input }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["me"] });
    },
  });
}
