import { useMutation, useQueryClient } from "@tanstack/react-query";

import { apiClient } from "@/lib/api-client";

// CreatedJob เก็บแค่ field เดียวที่ bot-form.tsx ต้องใช้จริง (id เอาไป redirect ไปหน้า
// live monitor ของ job ที่เพิ่งสร้างต่อ) — ตั้งใจไม่ import type Job เต็มๆ จาก
// features/jobs/types เพราะจะกลายเป็น cross-feature import ที่ผิดกติกาที่วางไว้ใน
// PHASE1-DASHBOARD-TODO.md (features/X ห้าม import จาก features/Y อื่นตรงๆ) — backend
// ตอบ field อื่นมาด้วยจริง (bot_name, status, created_at, ...) แต่ฝั่งนี้ไม่ได้ใช้เลย
// จึงไม่ต้องประกาศ type ให้ครบตามที่ backend ส่งมาจริง
interface CreatedJob {
  id: string;
}

// useCreateJob สร้าง mutation hook สำหรับ POST /bots/{name}/jobs — รับ botName ตอนเรียก
// hook (ไม่ใช่ตอน mutate) เพราะ URL ผูกกับ bot ที่เลือกไว้ตั้งแต่ต้น ส่วน mutate เอง
// รับแค่ config (key-value string) ที่ user กรอกจาก dynamic form
//
// invalidateQueries(["jobs"]) หลัง mutation สำเร็จ บอก TanStack Query ว่า cache ของหน้า
// job history (ข้อ 1.4) เก่าแล้ว ต้อง refetch ใหม่รอบหน้าที่มีคน mount หน้านั้น — ไม่ต้อง
// เขียน logic sync ข้อมูลระหว่างสอง feature เอง แค่บอกว่า "key ไหนควรถือว่าเก่าแล้ว"
export function useCreateJob(botName: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (config: Record<string, string>) =>
      apiClient<CreatedJob>(`/bots/${botName}/jobs`, { method: "POST", body: config }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["jobs"] });
    },
  });
}
