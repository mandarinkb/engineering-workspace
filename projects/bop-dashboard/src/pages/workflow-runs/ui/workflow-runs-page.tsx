import Link from "next/link";

import { WorkflowRunTable } from "@/widgets/workflow-run-table";

export function WorkflowRunsPage() {
  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl">Workflow Runs</h1>
        <Link
          href="/workflow-runs/new"
          className="btn-primary px-4 py-2 text-sm"
        >
          + Run workflow
        </Link>
      </div>
      <WorkflowRunTable />
    </div>
  );
}
