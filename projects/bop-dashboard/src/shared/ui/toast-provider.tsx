"use client";

import { createContext, useCallback, useContext, useRef, useState } from "react";

import { Toast, type ToastTone } from "./toast";

interface ToastItem {
  id: number;
  message: string;
  tone: ToastTone;
}

interface ShowToastOptions {
  message: string;
  tone?: ToastTone;
  durationMs?: number;
}

interface ToastContextValue {
  showToast: (options: ShowToastOptions) => void;
}

const ToastContext = createContext<ToastContextValue | null>(null);

const DEFAULT_DURATION_MS = 5000;

// ToastProvider คือ context กลางของทั้งแอป (mount ที่ src/app/providers.tsx เพียงจุด
// เดียว) เก็บ array ของ toast ที่กำลังแสดงอยู่ — ไม่ใช้ library ภายนอก (เช่น sonner) ตั้งใจ
// ให้ hand-roll เอง สอดคล้องกับแนวทางที่โปรเจกต์นี้ hand-roll SSE client/fetch wrapper เอง
// มาตลอด (ดู src/lib/sse-client.ts, src/lib/api-client.ts)
//
// nextId เป็น useRef ธรรมดา (ไม่ใช่ state) เพราะเป็นแค่ตัวนับ id ที่ไม่ต้องกระตุ้น
// re-render เมื่อเปลี่ยนค่า — เพิ่มค่าแล้วอ่านค่าปัจจุบันพอ ไม่ต้องผ่าน setState
export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const nextId = useRef(0);

  const dismiss = useCallback((id: number) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  const showToast = useCallback(
    ({ message, tone = "info", durationMs = DEFAULT_DURATION_MS }: ShowToastOptions) => {
      const id = nextId.current++;
      setToasts((prev) => [...prev, { id, message, tone }]);
      // auto-dismiss ด้วย setTimeout ธรรมดา — ไม่ต้อง clearTimeout ตอน unmount เพราะ
      // dismiss(id) เรียก setToasts ที่ filter หา id ที่ไม่มีอยู่แล้วก็แค่ไม่มีผลอะไร
      // ปลอดภัยเสมอไม่ว่า component จะ unmount ไปแล้วหรือยัง (setState หลัง unmount ของ
      // component ที่เป็นเจ้าของ toast เองไม่เกิดขึ้น เพราะ setToasts อยู่ใน provider
      // ที่ mount อยู่ตลอดอายุแอปอยู่แล้ว ไม่ใช่ component ที่เรียก showToast)
      window.setTimeout(() => dismiss(id), durationMs);
    },
    [dismiss],
  );

  return (
    <ToastContext.Provider value={{ showToast }}>
      {children}
      {/* fixed-position stack มุมขวาล่าง — เรียง toast ใหม่ไว้ล่างสุด (flex-col ปกติ
          ตามลำดับที่ push เข้า array) */}
      <div className="pointer-events-none fixed bottom-4 right-4 z-50 flex flex-col gap-2">
        {toasts.map((toast) => (
          <div key={toast.id} className="pointer-events-auto">
            <Toast message={toast.message} tone={toast.tone} onClose={() => dismiss(toast.id)} />
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}

// useToast ต้องเรียกภายใต้ <ToastProvider> เท่านั้น (ครอบทั้งแอปไว้แล้วที่
// src/app/providers.tsx) — throw ทันทีถ้าเรียกผิดที่ ดีกว่าให้ silently ทำอะไรไม่ได้
export function useToast(): ToastContextValue {
  const ctx = useContext(ToastContext);
  if (!ctx) {
    throw new Error("useToast ต้องเรียกภายใต้ <ToastProvider> เท่านั้น");
  }
  return ctx;
}
