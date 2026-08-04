import react from "@vitejs/plugin-react";
import path from "node:path";
import { defineConfig } from "vitest/config";

// vitest.config.ts แยกจาก next.config.ts เพราะ Vitest ไม่ได้รันผ่าน Next.js dev/build
// pipeline เลย (เร็วกว่ามากสำหรับ unit/component test — ไม่ต้อง compile ทั้งแอปทุกครั้ง)
// ต้องตั้ง alias "@/*" ให้ตรงกับ tsconfig.json เอง เพราะ Vitest ใช้ Vite ข้างใต้ ไม่ได้
// อ่าน path mapping จาก tsconfig ให้อัตโนมัติ
export default defineConfig({
  plugins: [react()],
  test: {
    environment: "jsdom", // จำลอง DOM ในสภาพแวดล้อม Node.js ให้ React Testing Library render ได้
    setupFiles: ["./vitest.setup.ts"],
    globals: true, // ใช้ describe/it/expect ได้โดยไม่ต้อง import เอง (เหมือน Jest)
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
});
