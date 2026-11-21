import { API_URL } from "./api-client";

const LOGIN_PATH = "/login";

let redirecting = false;

// redirectToLogin พาไปหน้า login หลังได้ 401 ระหว่างใช้งาน (เช่น session หมดอายุ) — ใช้เป็น
// onUnauthorized ของ createQueryClient
//
// ต้องเรียก POST /logout ก่อนเสมอ: cookie ที่หมดอายุหรือ signature ผิดยังค้างอยู่ในเบราว์เซอร์
// ถ้าไม่ล้าง middleware.ts (ที่เช็คแค่ว่ามี cookie ไหม) จะเด้งจาก /login กลับไป /overview
// วนไม่จบ — /logout เรียกได้โดยไม่ต้องมี session ที่ valid
//
// ใช้ window.location แทน router.push เพื่อโหลดหน้าใหม่ทั้งหน้า cache ของ query ที่ได้มาด้วย
// session เดิมจึงหายไปด้วย ไม่ทำอะไรถ้าอยู่หน้า login อยู่แล้ว (401 ของ credential ผิดเป็น
// เรื่องปกติของหน้านั้น) และไม่ทำซ้ำระหว่างกำลังพาไป เพราะหลาย query มักได้ 401 พร้อมกัน
export function redirectToLogin(): void {
  if (typeof window === "undefined" || redirecting || window.location.pathname === LOGIN_PATH) {
    return;
  }
  redirecting = true;
  void fetch(`${API_URL}/logout`, { method: "POST", credentials: "include" })
    .catch(() => undefined)
    .finally(() => {
      window.location.assign(LOGIN_PATH);
    });
}
