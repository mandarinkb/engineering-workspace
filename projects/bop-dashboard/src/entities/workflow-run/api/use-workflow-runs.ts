import { useInfiniteQuery, useQuery } from "@tanstack/react-query";

import { apiClient, LIST_REFETCH_INTERVAL_MS } from "@/shared/api";
import { nextRunPageToken, runCountPath, runListPath } from "../lib/run-list";
import type { RunStatusFilter, WorkflowRunCount, WorkflowRunListParams, WorkflowRunPage } from "../model/types";

// GET /workflow-runs query ตรงไปที่ Temporal Visibility API ทีละหน้า (ไม่ได้อ่านจาก database
// ของ BOP เอง) และกรองสถานะฝั่ง server — query key ทุกตัวขึ้นต้นด้วย "workflow-runs" เพื่อให้
// invalidateQueries({ queryKey: ["workflow-runs"] }) หลังสั่งรัน workflow ล้างได้ครบทุกตัว
// ส่วนตัวที่ 2 ("pages" / "page") แยก cache ของ infinite query ออกจาก query หน้าเดียว
// เพราะรูปข้อมูลต่างกัน และไม่ชนกับ ["workflow-runs", id] ของหน้า detail (ความยาว key ต่างกัน)

// useWorkflowRunPages โหลดทีละหน้าสำหรับปุ่ม "โหลดเพิ่ม" — ตอน auto-refresh ทุก 10 วินาที
// TanStack Query refetch ทุกหน้าที่โหลดไว้ใหม่ไล่จากหน้าแรกด้วย token ชุดใหม่ จำนวนหน้าเท่าเดิม
// แถวที่โหลดเพิ่มไว้จึงไม่หาย และ run ใหม่ขึ้นมาที่หน้าแรก
export function useWorkflowRunPages(params: WorkflowRunListParams = {}) {
  return useInfiniteQuery({
    queryKey: ["workflow-runs", "pages", params],
    queryFn: ({ pageParam }) => apiClient<WorkflowRunPage>(runListPath(params, pageParam)),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: nextRunPageToken,
    refetchInterval: LIST_REFETCH_INTERVAL_MS,
  });
}

// useWorkflowRunPage โหลดหน้าแรกหน้าเดียว (เช่นสรุปในหน้า Overview)
export function useWorkflowRunPage(params: WorkflowRunListParams = {}) {
  return useQuery({
    queryKey: ["workflow-runs", "page", params],
    queryFn: () => apiClient<WorkflowRunPage>(runListPath(params)),
    refetchInterval: LIST_REFETCH_INTERVAL_MS,
  });
}

// useWorkflowRunCount นับ Workflow Run ตามสถานะฝั่ง server (GET /workflow-runs/count) — ถูกต้อง
// ไม่ว่าจะมี run กี่หน้า ต่างจากการนับจากหน้าแรกของรายการ
export function useWorkflowRunCount(status: RunStatusFilter = "") {
  return useQuery({
    queryKey: ["workflow-runs", "count", status],
    queryFn: () => apiClient<WorkflowRunCount>(runCountPath(status)),
    refetchInterval: LIST_REFETCH_INTERVAL_MS,
  });
}
