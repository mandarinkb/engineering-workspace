"use client";

import { useRouter, useSearchParams } from "next/navigation";

const STATUS_OPTIONS = ["", "pending", "running", "succeeded", "failed", "cancelled"] as const;

// JobFilter ไม่มี state ของตัวเองเลย (ไม่มี useState สักตัว) — อ่าน/เขียนค่า filter ผ่าน
// URL query param โดยตรงเสมอ (useSearchParams อ่านค่าปัจจุบัน, router.push เขียนค่าใหม่)
// ตามที่ roadmap ข้อ 1.4 ต้องการ ("ฝึก sync state ระหว่าง URL query param กับ UI filter")
// ข้อดีเทียบกับเก็บใน useState ธรรมดา: filter ที่ user ตั้งไว้รอดจากการ refresh หน้า และ
// copy URL ไปแชร์ให้คนอื่นเห็น filter เดียวกันได้ทันที — job-table.tsx (คนละ component)
// อ่าน URL เดียวกันนี้เองผ่าน useSearchParams เหมือนกัน ทั้งสองไฟล์จึง "sync" กันผ่าน URL
// โดยไม่ต้องมี state ที่แชร์กันตรงๆ เลย (ไม่ต้องยก state ขึ้นไปไว้ที่ parent component)
export function JobFilter() {
  const router = useRouter();
  const searchParams = useSearchParams();

  const bot = searchParams.get("bot") ?? "";
  const status = searchParams.get("status") ?? "";
  const from = searchParams.get("from") ?? "";
  const to = searchParams.get("to") ?? "";

  function updateParam(key: string, value: string) {
    const params = new URLSearchParams(searchParams.toString());
    if (value) {
      params.set(key, value);
    } else {
      params.delete(key); // ค่าว่างเปล่า = ไม่กรอง ลบ query param ออกไปเลยดีกว่าเก็บ "" ไว้เฉยๆ
    }
    router.push(`?${params.toString()}`);
  }

  return (
    <div className="flex flex-wrap gap-4">
      <input
        type="text"
        placeholder="กรองตามชื่อ bot..."
        value={bot}
        onChange={(event) => updateParam("bot", event.target.value)}
        className="rounded border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none"
      />
      <select
        value={status}
        onChange={(event) => updateParam("status", event.target.value)}
        className="rounded border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none"
      >
        {STATUS_OPTIONS.map((s) => (
          <option key={s || "all"} value={s}>
            {s === "" ? "ทุกสถานะ" : s}
          </option>
        ))}
      </select>
      {/* date-range filter — ผูกกับ URL param "from"/"to" ตาม pattern เดียวกับ
          bot/status ด้านบนเป๊ะ เทียบ job.created_at (ISO timestamp string) กับค่าที่
          กรอกใน job-table.tsx */}
      <label className="flex items-center gap-2 text-sm text-gray-600">
        จาก
        <input
          type="date"
          value={from}
          onChange={(event) => updateParam("from", event.target.value)}
          className="rounded border border-gray-300 px-2 py-2 focus:border-blue-500 focus:outline-none"
        />
      </label>
      <label className="flex items-center gap-2 text-sm text-gray-600">
        ถึง
        <input
          type="date"
          value={to}
          onChange={(event) => updateParam("to", event.target.value)}
          className="rounded border border-gray-300 px-2 py-2 focus:border-blue-500 focus:outline-none"
        />
      </label>
    </div>
  );
}
