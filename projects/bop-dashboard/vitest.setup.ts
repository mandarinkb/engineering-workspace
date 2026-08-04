// import ครั้งเดียวที่นี่พอ — เพิ่ม custom matcher ของ @testing-library/jest-dom
// (เช่น .toBeInTheDocument(), .toHaveTextContent()) เข้าไปให้ expect() ของ Vitest ใช้ได้
// ทุก test file โดยไม่ต้อง import ซ้ำเองทุกไฟล์ (ตั้งค่าผ่าน setupFiles ใน vitest.config.ts)
import "@testing-library/jest-dom/vitest";
