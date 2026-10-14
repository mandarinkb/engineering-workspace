import { redirect } from "next/navigation";

// หน้า root ("/") ไม่มี UI ของตัวเองเลย แค่ redirect ไปที่ /jobs เสมอ (middleware.ts จะ
// เด้งไป /login ให้เองถ้ายังไม่ login) — redirect() ของ next/navigation ทำงานฝั่ง server
// (Server Component ธรรมดา ไม่ต้อง "use client") เร็วกว่าการ redirect ฝั่ง client ผ่าน
// useEffect + router.push ที่ต้อง render หน้าเปล่าๆ ขึ้นมาก่อนแล้วค่อย redirect ทีหลัง
export default function RootPage() {
  redirect("/jobs");
}
