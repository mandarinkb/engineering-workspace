import { JobLogViewer } from "@/features/jobs/components/job-log-viewer";

interface JobDetailPageProps {
  // Next.js 15 เปลี่ยน params ให้เป็น Promise เสมอ (ไม่ใช่ object ตรงๆ แบบเวอร์ชันก่อน)
  // ต้อง await ก่อนใช้ — เปิดทางให้ Next.js streaming ส่วนอื่นของหน้าได้ก่อนที่ param
  // จะพร้อมในบางสถานการณ์ (ไม่ค่อยมีผลกับหน้านี้เพราะ id มาจาก URL ตรงๆ อยู่แล้ว แต่ก็
  // ต้อง await ตาม API ใหม่เสมอ)
  params: Promise<{ id: string }>;
}

// หน้านี้คือ Live Job Monitor (roadmap ข้อ 1.5) — ตัว page.tsx เองยังเป็น Server
// Component (ไม่มี "use client") ทำหน้าที่แค่ดึง id จาก URL แล้วส่งต่อให้ JobLogViewer
// (Client Component ที่เปิด SSE connection จริง ดู job-log-viewer.tsx) รับช่วงต่อ
export default async function JobDetailPage({ params }: JobDetailPageProps) {
  const { id } = await params;
  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-2xl font-semibold">Job {id}</h1>
      <JobLogViewer jobId={id} />
    </div>
  );
}
