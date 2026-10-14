import type { Bot } from "../types";

interface BotCardProps {
  bot: Bot;
  selected: boolean;
  onSelect: () => void;
}

// BotCard คือ card เดียวในกริดของ bot-list.tsx — ไม่มี "use client" ของตัวเอง เพราะไม่ได้
// เรียก hook หรือ browser API อะไรเลย (ถูก bundle ไปเป็นส่วนหนึ่งของ client component
// tree อัตโนมัติเพราะ parent คือ bot-list.tsx ที่ mark "use client" อยู่แล้ว) แสดงแค่ชื่อ
// bot กับจำนวน config field ให้เห็นคร่าวๆ ก่อนจะกดเลือกไปดู form เต็มๆ
export function BotCard({ bot, selected, onSelect }: BotCardProps) {
  return (
    <button
      type="button"
      onClick={onSelect}
      className={`rounded border p-4 text-left transition ${
        selected ? "border-blue-600 bg-blue-50" : "border-gray-200 hover:border-gray-400"
      }`}
    >
      <p className="font-mono font-medium">{bot.name}</p>
      <p className="mt-1 text-sm text-gray-500">
        {bot.config_schema.length} config field{bot.config_schema.length === 1 ? "" : "s"}
      </p>
    </button>
  );
}
