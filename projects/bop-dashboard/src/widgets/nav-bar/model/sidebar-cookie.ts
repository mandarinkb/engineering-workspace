// SIDEBAR_COOKIE เก็บสถานะพับ/กางของ sidebar ไว้ใน cookie (ไม่ใช่ localStorage) เพราะ
// app/(dashboard)/layout.tsx ซึ่งเป็น Server Component อ่าน cookie ได้ตอน render ฝั่ง server
// HTML แรกที่ส่งไปจึงพับ/กางถูกตั้งแต่ต้น ไม่มีจังหวะที่ sidebar กางวาบขึ้นมาก่อนแล้วค่อยพับ
// (ถ้าใช้ localStorage ต้องรอ JS ฝั่ง client อ่านค่าหลัง hydrate) — cookie นี้เป็นแค่ UI
// preference ไม่มีข้อมูลสำคัญ เลยเขียนจาก JS ได้ตรงๆ ไม่ต้อง httpOnly
//
// ไฟล์นี้ตั้งใจไม่มี "use client" — layout (server) ต้องเรียก isSidebarCollapsed() ได้
// ถ้าอยู่ในไฟล์ "use client" (nav-bar.tsx) Next.js จะ throw ตอน render ฝั่ง server
export const SIDEBAR_COOKIE = "bop_sidebar";

const COLLAPSED = "collapsed";
const EXPANDED = "expanded";
const ONE_YEAR_SECONDS = 60 * 60 * 24 * 365;

export function isSidebarCollapsed(cookieValue: string | undefined): boolean {
  return cookieValue === COLLAPSED;
}

// sidebarCookie คืนสตริงสำหรับ document.cookie ฝั่ง client
export function sidebarCookie(collapsed: boolean): string {
  return `${SIDEBAR_COOKIE}=${collapsed ? COLLAPSED : EXPANDED}; path=/; max-age=${ONE_YEAR_SECONDS}; samesite=lax`;
}
