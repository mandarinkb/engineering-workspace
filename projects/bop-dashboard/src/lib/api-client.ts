import { ApiError } from "@/types/api";

// API_URL อ่านจาก environment variable ที่มี prefix NEXT_PUBLIC_ (ต้องมี prefix นี้เท่านั้น
// โค้ดฝั่ง client/เบราว์เซอร์ถึงจะอ่านค่านี้ได้ — ดู .env.example) ถ้าไม่ได้ตั้งไว้เลย
// fallback ไปที่ localhost:8080 เพื่อให้ `npm run dev` ทำงานได้ทันทีโดยไม่ต้อง config
// อะไรก่อน ตราบใดที่ backend (cmd/bop) รันอยู่ที่ port default ของมันเอง
const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

interface RequestOptions {
  method?: "GET" | "POST" | "DELETE";
  // body เป็น unknown ไม่ใช่ any โดยตั้งใจ — บังคับให้ผู้เรียกต้องระบุ type จริงตอนใช้งาน
  // ผ่าน generic <T> ของฟังก์ชันนี้เอง ไม่ปล่อยให้ TypeScript เงียบผ่านค่าอะไรก็ได้
  body?: unknown;
}

// apiClient คือ fetch wrapper กลางตัวเดียวที่ทุก feature/*/api/*.ts ต้องเรียกผ่านนี้เสมอ
// (ห้ามยิง fetch() ตรงๆ กระจายอยู่หลายที่) รวม concern 3 อย่างไว้ที่เดียว:
//   1. แนบ base URL ให้อัตโนมัติ (เรียกแค่ path เช่น "/jobs" ไม่ต้องพิมพ์ host เต็ม)
//   2. แนบ credentials: "include" เสมอ — จำเป็นมากเพราะ dashboard (localhost:3000) กับ
//      backend (localhost:8080) เป็นคนละ origin การจะให้เบราว์เซอร์แนบ cookie (JWT
//      httpOnly ที่ POST /login ตั้งไว้) ไปด้วยตอนยิง cross-origin request ต้องระบุ
//      credentials: "include" ชัดเจนเสมอ ไม่งั้น cookie จะไม่ถูกส่งไปเลยแม้จะมีอยู่จริง
//   3. แปลง error ทุกรูปแบบ (network ล่ม, backend ตอบ non-2xx) ให้เป็น ApiError รูปแบบ
//      เดียวกันหมด — โค้ดที่เรียกใช้ (TanStack Query hook ใน feature ต่างๆ) เช็ค error
//      ได้แบบเดียวกันทุกที่ ไม่ต้องเขียน error handling ซ้ำๆ ต่างกันไปตาม feature
export async function apiClient<T>(path: string, options: RequestOptions = {}): Promise<T> {
  let res: Response;
  try {
    res = await fetch(`${API_URL}${path}`, {
      method: options.method ?? "GET",
      headers: options.body !== undefined ? { "Content-Type": "application/json" } : undefined,
      body: options.body !== undefined ? JSON.stringify(options.body) : undefined,
      credentials: "include",
    });
  } catch {
    // fetch() throw ตรงนี้แปลว่า request ไม่ถึง server เลย (network ล่ม, DNS ผิด, CORS
    // block ตั้งแต่ต้น) ไม่ใช่กรณี server ตอบกลับมาแต่เป็น error status — แยกออกจากกัน
    // ชัดเจนด้วย status: 0 (ไม่มี HTTP status code จริงให้ใช้ในกรณีนี้)
    throw new ApiError("เรียก BOP API ไม่ติด (backend รันอยู่ไหม?)", 0);
  }

  if (!res.ok) {
    // อ่าน body เป็น text (ไม่ใช่ JSON) เพราะ backend ตอบ error ด้วย http.Error ธรรมดา
    // (plain text) ไม่ใช่ JSON object เสมอไป (ดู internal/api/http/handler.go)
    const message = (await res.text().catch(() => "")) || res.statusText;
    throw new ApiError(message, res.status);
  }

  // 204 No Content (เช่น POST /login, POST /logout, DELETE /jobs/{id}) ไม่มี body ให้ parse
  // เป็น JSON เลย — ต้องเช็คแยกก่อน ไม่งั้น res.json() จะ throw เพราะ body ว่างเปล่า
  if (res.status === 204) {
    return undefined as T;
  }

  return (await res.json()) as T;
}
