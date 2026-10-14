import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

const COOKIE_NAME = "bop_session"; // ต้องตรงกับ cookieName ใน internal/api/http/handler.go เป๊ะ
const PUBLIC_PATHS = ["/login"];

// middleware ทำงานที่ edge ก่อนจะ render หน้าไหนก็ตาม (เร็วกว่าเช็คใน component เพราะไม่
// ต้องโหลด React tree ขึ้นมาก่อนแล้วค่อยรู้ว่าต้อง redirect) — roadmap ข้อ 1.8 ("Protected
// route — ต้อง login ก่อนเห็น dashboard")
//
// ข้อจำกัดที่ต้องรู้ (สำคัญ อย่าเข้าใจผิดว่าที่นี่คือจุดตรวจสอบความปลอดภัยตัวจริง):
// middleware เช็คแค่ "มี cookie ชื่อ bop_session อยู่ไหม" เท่านั้น ไม่ได้ verify signature
// หรือเช็ควันหมดอายุของ JWT เลย เพราะ middleware รันบน Edge Runtime ที่ไม่มี JWT secret
// ของ backend อยู่ในมือ (secret อยู่ฝั่ง Go เท่านั้น — ดู internal/auth/jwt.go) การตรวจสอบ
// ที่แท้จริงว่า token valid จริงไหมเกิดขึ้นตอน component เรียก GET /me ผ่าน useAuth()
// อีกที (ถ้า cookie ปลอมหรือหมดอายุแล้ว backend จะตอบ 401 กลับมาเอง) — middleware ตัวนี้
// จึงเป็นแค่ "UX shortcut" กัน user เห็นหน้า dashboard วาบขึ้นมาก่อนโดน redirect ไม่ใช่
// security boundary ตัวจริง (security boundary ตัวจริงอยู่ที่ backend ทั้งหมด)
export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;
  const hasSession = request.cookies.has(COOKIE_NAME);

  if (PUBLIC_PATHS.includes(pathname)) {
    // login อยู่แล้วแต่พยายามเข้าหน้า /login ซ้ำ พาไปหน้า /jobs แทน — ไม่มีเหตุผลให้เห็น
    // form login อีกครั้งทั้งที่ login ค้างอยู่แล้ว
    if (hasSession) {
      return NextResponse.redirect(new URL("/jobs", request.url));
    }
    return NextResponse.next();
  }

  if (!hasSession) {
    return NextResponse.redirect(new URL("/login", request.url));
  }

  return NextResponse.next();
}

export const config = {
  // จับทุก path ยกเว้น static asset ของ Next.js เอง (_next, favicon ฯลฯ) — ไม่งั้น
  // middleware จะรันทุกครั้งที่โหลดไฟล์ static โดยไม่จำเป็นเลย
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],
};
