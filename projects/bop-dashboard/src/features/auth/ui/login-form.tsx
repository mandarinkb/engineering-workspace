"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { useLogin } from "../api/use-login";

// loginSchema คือ zod schema ที่ประกาศ "รูปร่างและกฎ" ของข้อมูล form นี้ไว้ที่เดียว —
// react-hook-form ใช้ schema เดียวกันนี้ validate ทุก field ให้อัตโนมัติผ่าน zodResolver
// (ไม่ต้องเขียน if/else validate เองทีละ field) ส่วน z.infer<typeof loginSchema> ด้านล่าง
// คือการให้ TypeScript "อ่าน" schema นี้แล้วสร้าง type ให้อัตโนมัติ (LoginFormValues จะมี
// field username/password เป็น string ตรงตาม schema เป๊ะ) แนวคิดนี้คล้าย struct tag
// validation ฝั่ง Go ที่คุ้นเคยอยู่แล้ว (ประกาศกฎไว้ที่ type เดียว ไม่ต้องเขียนซ้ำ)
const loginSchema = z.object({
  username: z.string().min(1, "กรอก username"),
  password: z.string().min(1, "กรอก password"),
});

type LoginFormValues = z.infer<typeof loginSchema>;

// LoginForm คือ controlled form เต็มรูปแบบ (react-hook-form register ทุก input ให้เอง
// ผ่าน {...register("fieldName")} — "controlled" หมายถึงค่าของ input ถูกควบคุมโดย form
// state เสมอ ไม่ใช่ DOM ควบคุมตัวเอง) ใช้ pattern เดียวกับที่ bot-form.tsx (roadmap ข้อ
// 1.3) จะใช้กับ dynamic config field ทีหลัง — ต่างกันแค่ schema ของ form นี้ตายตัว
// (username/password คงที่) แต่ของ bot-form.tsx จะสร้าง schema แบบ dynamic ตาม
// config_schema ที่ได้จาก GET /bots
export function LoginForm() {
  const router = useRouter();
  const login = useLogin();
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginFormValues>({ resolver: zodResolver(loginSchema) });

  // handleSubmit ของ react-hook-form จะ validate ตาม schema ก่อนเสมอ ถ้าผ่านถึงจะเรียก
  // callback ของเรา (ค่าที่ผ่านมาถึงตรงนี้จึงรับประกันว่าตรงตาม LoginFormValues แน่นอน
  // ไม่ต้องเช็คซ้ำอีก) — ถ้า validate ไม่ผ่าน handleSubmit จะไม่เรียก callback เลย แค่
  // อัปเดต errors ให้ input ที่ผิดแสดง error message ใต้ field นั้นแทน
  const onSubmit = handleSubmit((values) => {
    login.mutate(values, {
      onSuccess: () => {
        router.push("/jobs"); // login สำเร็จ พาไปหน้าแรกของ dashboard ที่ต้อง login ก่อน
      },
    });
  });

  return (
    <form onSubmit={onSubmit} className="flex w-full max-w-sm flex-col gap-4">
      <div>
        <label htmlFor="username" className="block text-sm font-medium">
          Username
        </label>
        <input
          id="username"
          {...register("username")}
          className="mt-1 w-full rounded border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none"
          autoComplete="username"
        />
        {errors.username && <p className="mt-1 text-sm text-red-600">{errors.username.message}</p>}
      </div>

      <div>
        <label htmlFor="password" className="block text-sm font-medium">
          Password
        </label>
        <input
          id="password"
          type="password"
          {...register("password")}
          className="mt-1 w-full rounded border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none"
          autoComplete="current-password"
        />
        {errors.password && <p className="mt-1 text-sm text-red-600">{errors.password.message}</p>}
      </div>

      {/* login.error มาจาก ApiError ที่ api-client.ts throw ออกมา (ดูตัวอย่าง 401
          "invalid credentials" ตรงจาก backend) — แสดงตรงๆ ให้ user เห็นว่าผิดตรงไหน */}
      {login.isError && <p className="text-sm text-red-600">{login.error.message}</p>}

      <button
        type="submit"
        disabled={login.isPending}
        className="rounded bg-blue-600 px-4 py-2 font-medium text-white hover:bg-blue-700 disabled:opacity-50"
      >
        {login.isPending ? "กำลังเข้าสู่ระบบ..." : "เข้าสู่ระบบ"}
      </button>
    </form>
  );
}
