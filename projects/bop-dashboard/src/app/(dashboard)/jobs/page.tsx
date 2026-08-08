import { Suspense } from "react";

import { JobFilter } from "@/features/jobs/components/job-filter";
import { JobStats } from "@/features/jobs/components/job-stats";
import { JobTable } from "@/features/jobs/components/job-table";

// ต้องห่อ <Suspense> รอบ component ที่เรียก useSearchParams() เสมอ (JobFilter/JobTable
// ทั้งคู่) — Next.js App Router bail out ไปเป็น client-side rendering ทันทีที่เจอ
// useSearchParams() เพราะค่านี้อ่านได้แค่ตอนรันบนเบราว์เซอร์จริงเท่านั้น (query string
// ของ URL ไม่รู้ค่าล่วงหน้าตอน build/prerender) ถ้าไม่ห่อ Suspense ไว้ Next.js จะ error
// ตอน build ("useSearchParams() should be wrapped in a suspense boundary") เพราะไม่รู้ว่า
// จะแสดงอะไรให้ user เห็นระหว่างรอ client-side hydration
export default function JobsPage() {
  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-2xl font-semibold">Job History</h1>
      <JobStats />
      <Suspense fallback={<p className="text-gray-500">กำลังโหลด...</p>}>
        <JobFilter />
        <JobTable />
      </Suspense>
    </div>
  );
}
