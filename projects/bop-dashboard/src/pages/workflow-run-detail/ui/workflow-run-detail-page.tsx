import { WorkflowRunDetail } from "@/widgets/workflow-run-detail";

interface WorkflowRunDetailPageProps {
  params: Promise<{ id: string }>; // Next.js 15: params เป็น Promise เสมอ ต้อง await
}

export async function WorkflowRunDetailPage({ params }: WorkflowRunDetailPageProps) {
  const { id } = await params;
  return (
    <div className="flex flex-col gap-6">
      <h1 className="break-all text-2xl font-semibold">{id}</h1>
      <WorkflowRunDetail workflowId={id} />
    </div>
  );
}
