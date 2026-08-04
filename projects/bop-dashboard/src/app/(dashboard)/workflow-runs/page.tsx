import { WorkflowRunTable } from "@/features/workflow-runs/components/workflow-run-table";

export default function WorkflowRunsPage() {
  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-2xl font-semibold">Workflow Runs</h1>
      <WorkflowRunTable />
    </div>
  );
}
