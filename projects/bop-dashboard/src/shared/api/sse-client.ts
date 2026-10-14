// SSEStatus คือสถานะของ SSE connection ตอนนี้ — ใช้ให้ UI (เช่น job-log-viewer.tsx)
// แสดง badge บอก user ว่ากำลังเชื่อมต่ออยู่/เชื่อมสำเร็จแล้ว/หลุดการเชื่อมต่อ
export type SSEStatus = "connecting" | "connected" | "disconnected";

interface ConnectSSEOptions<T> {
  onMessage: (data: T) => void;
  onStatusChange?: (status: SSEStatus) => void;
}

// connectSSE เปิด Server-Sent Events connection ไปยัง backend แล้วคืน "cleanup function"
// ที่ต้องเรียกตอน component unmount เสมอ (เช่นใน useEffect cleanup) เพื่อปิด connection
// ไม่ให้ค้างอยู่เบื้องหลังทั้งที่ user ออกจากหน้านั้นไปแล้ว (memory leak ทั้งฝั่ง browser
// และฝั่ง backend ที่ยังคง subscriber ของ jobID นั้นค้างไว้ในหน่วยความจำ)
//
// จุดสำคัญเรื่อง reconnect: EventSource (Web API มาตรฐานของเบราว์เซอร์) มี auto-reconnect
// ในตัวอยู่แล้ว — ถ้า connection หลุดกลางทาง (network สะดุด, server restart) เบราว์เซอร์
// จะพยายามเชื่อมต่อใหม่ให้เองอัตโนมัติโดยไม่ต้องเขียน logic เพิ่มเลยสักบรรทัด (ต่างจาก
// WebSocket ที่ต้องเขียน reconnect loop เอง) — ฟังก์ชันนี้แค่ "ฟัง" event ที่ EventSource
// ยิงออกมาแล้วแปลงเป็น SSEStatus ให้ UI ใช้แสดงผลเท่านั้น ไม่ได้ควบคุมการ reconnect เอง
//
// withCredentials: true จำเป็นมาก — เหมือนกับ credentials: "include" ใน api-client.ts
// เพราะ backend อยู่คนละ origin (localhost:8080 vs dashboard ที่ localhost:3000) ถ้าไม่ตั้ง
// ค่านี้ เบราว์เซอร์จะไม่แนบ cookie (JWT httpOnly) ไปกับ SSE request เลย
export function connectSSE<T>(path: string, options: ConnectSSEOptions<T>): () => void {
  const apiUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

  options.onStatusChange?.("connecting");
  const source = new EventSource(`${apiUrl}${path}`, { withCredentials: true });

  source.onopen = () => {
    options.onStatusChange?.("connected");
  };

  source.onmessage = (event: MessageEvent<string>) => {
    try {
      options.onMessage(JSON.parse(event.data) as T);
    } catch {
      // ข้าม message ที่ parse เป็น JSON ไม่ได้ (ไม่ควรเกิดจริงเพราะ backend เข้ารหัส
      // logger.Line เป็น JSON เสมอ ดู writeSSELine ใน internal/api/http/handler.go)
    }
  };

  source.onerror = () => {
    // เข้า case นี้ทั้งตอน connection หลุดชั่วคราว (กำลังจะ auto-reconnect) และตอนปิด
    // ถาวร (เช่น server ปิดตัว) — เบราว์เซอร์ไม่ได้แยกสองกรณีนี้ให้ผ่าน event ต่างกัน
    // จึงรายงานเป็น "disconnected" เหมือนกันไปก่อน ถ้า reconnect สำเร็จจะได้ event
    // "onopen" กลับมาเป็น "connected" อีกครั้งเอง
    options.onStatusChange?.("disconnected");
  };

  return () => {
    source.close();
  };
}
