"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

import { useBots } from "../api/use-bots";
import { BotCard } from "./bot-card";
import { BotForm } from "./bot-form";

// BotList คือหน้าหลักของ roadmap ข้อ 1.2 — โหลดรายชื่อ bot ผ่าน useBots (useEffect ซ่อน
// อยู่ภายใน TanStack Query แล้ว ไม่ต้องเขียน useEffect ยิง fetch เองตรงๆ) แสดงเป็นกริดของ
// BotCard พร้อม search filter (useState เก็บคำค้นหา, กรองฝั่ง client ล้วนๆ เพราะจำนวน bot
// ทั้งระบบไม่เยอะ ไม่คุ้มที่จะทำ server-side search) และเก็บ "bot ที่กำลังเลือกอยู่" อีก
// useState หนึ่งตัว — เลือก bot ไหนแล้ว BotForm (dynamic form ตามข้อ 1.3) จะโผล่ขึ้นมา
// ให้กรอก config ของ bot ตัวนั้นทันที
export function BotList() {
  const { data: bots, isLoading, isError, error } = useBots();
  const [search, setSearch] = useState("");
  const [selectedName, setSelectedName] = useState<string | null>(null);
  const router = useRouter();

  if (isLoading) {
    return <p className="text-gray-500">กำลังโหลดรายการ bot...</p>;
  }
  if (isError) {
    return <p className="text-red-600">โหลดรายการ bot ไม่สำเร็จ: {error.message}</p>;
  }

  const filtered = (bots ?? []).filter((b) => b.name.toLowerCase().includes(search.toLowerCase()));
  const selected = filtered.find((b) => b.name === selectedName) ?? null;

  return (
    <div className="flex flex-col gap-6">
      <input
        type="search"
        placeholder="ค้นหา bot..."
        value={search}
        onChange={(event) => setSearch(event.target.value)}
        className="w-full max-w-sm rounded border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none"
      />

      {/* empty state — ยังไม่มี bot ที่ตรงกับคำค้นหา (หรือระบบยังไม่มี bot เลยตั้งแต่แรก) */}
      {filtered.length === 0 ? (
        <p className="text-gray-500">ยังไม่มี bot ที่ตรงกับคำค้นหา</p>
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {/* key={bot.name} ตรงนี้คือ key ปกติของ list rendering (บอก React ว่า card ไหน
              เป็นตัวไหนข้ามรอบ re-render) คนละเรื่องกับ key={selected.name} ที่ BotForm
              ด้านล่างซึ่งใช้บังคับให้ remount ทั้ง component (ดูคำอธิบายที่ bot-form.tsx) */}
          {filtered.map((bot) => (
            <BotCard
              key={bot.name}
              bot={bot}
              selected={bot.name === selectedName}
              onSelect={() => setSelectedName(bot.name)}
            />
          ))}
        </div>
      )}

      {selected && (
        <div className="max-w-sm rounded border border-gray-200 p-4">
          <h2 className="mb-4 text-lg font-semibold">รัน {selected.name}</h2>
          <BotForm key={selected.name} bot={selected} onCreated={(jobId) => router.push(`/jobs/${jobId}`)} />
        </div>
      )}
    </div>
  );
}
