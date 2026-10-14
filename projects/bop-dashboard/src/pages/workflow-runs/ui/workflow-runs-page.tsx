import { WorkflowRunTable } from "@/widgets/workflow-run-table";

export function WorkflowRunsPage() {
  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-2xl font-semibold">Workflow Runs</h1>
      <WorkflowRunTable />
    </div>
  );
}
