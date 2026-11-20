import { WorkflowRunDetail } from "@/widgets/workflow-run-detail";

interface WorkflowRunDetailPageProps {
  params: Promise<{ id: string }>; // Next.js 15: params เป็น Promise เสมอ ต้อง await
}

export async function WorkflowRunDetailPage({ params }: WorkflowRunDetailPageProps) {
  // params ของ dynamic segment ยังเป็นรูป URL-encoded (":" มาเป็น "%3A") ต้อง decode ก่อน
  // ไม่งั้น hook ที่ encodeURIComponent ซ้ำจะส่ง "%253A" ไป backend แล้วได้ 404 — เจอกับ
  // workflow ID ของ schedule เช่น check-example-com-2026-11-20T16:52:00Z
  const id = decodeURIComponent((await params).id);
  return (
    <div className="flex flex-col gap-6">
      <h1 className="break-all text-2xl font-semibold">{id}</h1>
      <WorkflowRunDetail workflowId={id} />
    </div>
  );
}
