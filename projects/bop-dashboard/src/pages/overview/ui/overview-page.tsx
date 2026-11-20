import { Overview } from "@/widgets/overview";

export function OverviewPage() {
  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-2xl font-semibold">Overview</h1>
      <Overview />
    </div>
  );
}
