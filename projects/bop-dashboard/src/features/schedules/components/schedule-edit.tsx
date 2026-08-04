"use client";

import { useSchedule } from "../api/use-schedule";
import { ScheduleForm } from "./schedule-form";

interface ScheduleEditViewProps {
  id: string;
}

// ScheduleEditView โหลด schedule ปัจจุบันก่อน (ผ่าน useSchedule ตัวเดียวกับหน้า detail
// — ถ้า cache ยังสดจาก TanStack Query ก็ไม่ต้องรอ loading เลยด้วยซ้ำ) แล้วส่งต่อเป็น
// initial prop ให้ ScheduleForm — ต้องรอให้โหลดเสร็จก่อนค่อย mount ScheduleForm เพราะ
// useForm ของ react-hook-form อ่าน defaultValues แค่ตอน mount ครั้งแรกเท่านั้น (ถ้า
// mount ฟอร์มไปก่อนข้อมูลมาถึง แล้วค่อยเปลี่ยน initial ทีหลัง ฟอร์มจะไม่ update
// defaultValues ให้เองอัตโนมัติ)
export function ScheduleEditView({ id }: ScheduleEditViewProps) {
  const { data: schedule, isLoading, isError, error } = useSchedule(id);

  if (isLoading) {
    return <p className="text-gray-500">กำลังโหลด...</p>;
  }
  if (isError) {
    return <p className="text-red-600">โหลด schedule ไม่สำเร็จ: {error.message}</p>;
  }
  if (!schedule) {
    return null;
  }

  return <ScheduleForm initial={schedule} />;
}
