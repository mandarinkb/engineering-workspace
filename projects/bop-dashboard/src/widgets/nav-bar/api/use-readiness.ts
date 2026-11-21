import { useQuery } from "@tanstack/react-query";

import { checkReadiness } from "@/shared/api";

const READINESS_REFETCH_MS = 15_000;

// useReadiness poll GET /readyz ให้ไฟสถานะใน sidebar — checkReadiness ไม่ throw (คืนสถานะ
// เป็นค่าแทน) จึงไม่มี retry ของ TanStack Query มาหน่วงตอน API ล่ม ไฟแดงขึ้นตั้งแต่รอบแรก
export function useReadiness() {
  return useQuery({
    queryKey: ["readiness"],
    queryFn: checkReadiness,
    refetchInterval: READINESS_REFETCH_MS,
    staleTime: 0,
  });
}
