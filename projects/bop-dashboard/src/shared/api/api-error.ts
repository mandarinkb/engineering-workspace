// Type กลางที่ใช้ข้าม slice ได้ทั้งหมด (ไม่ผูกกับ entity หรือ feature ไหนโดยเฉพาะ)
// อยู่ที่ชั้น shared ของ Feature-Sliced Design เพราะทุก slice ต้องใช้ร่วมกัน — ชั้น shared
// ห้าม import อะไรจากชั้นที่สูงกว่าเลย (ทิศทางเดียว ดู ARCHITECTURE.md)

// ApiError คือรูปแบบ error ที่ src/shared/api/api-client.ts throw ออกมาเวลา request ไม่สำเร็จ
// (ไม่ว่าจะ network error หรือ backend ตอบ status code ที่ไม่ใช่ 2xx) — ทุก feature/api/*.ts
// จับ error นี้ได้ type เดียวกันหมด ไม่ต้องเดารูปแบบ error เองทีละที่
export class ApiError extends Error {
  // status เป็น 0 ถ้า request ไม่ถึง server เลย (เช่น network ล่ม, CORS block) — ต่างจาก
  // status ที่ backend ตอบมาจริง (4xx/5xx) ซึ่งจะมีค่าตรงกับ HTTP status code จริง
  readonly status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}
