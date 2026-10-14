import { BotList } from "@/widgets/bot-list";

// page.tsx บาง — แค่ import component จาก features/ มาวาง ไม่มี logic ของตัวเองเลย
// (ตามกติกาที่ PHASE1-DASHBOARD-TODO.md วางไว้: app/ ทำหน้าที่ routing/layout เท่านั้น)
export function BotsPage() {
  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-2xl font-semibold">Bots</h1>
      <BotList />
    </div>
  );
}
