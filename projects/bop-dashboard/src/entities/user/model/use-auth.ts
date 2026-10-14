import { useMe } from "../api/use-me";

// useAuth ห่อ useMe ให้เหลือแค่ shape ที่ component ทั่วไปอยากได้จริงๆ (isAuthenticated
// ตรงๆ แทนที่จะให้ทุกที่ต้องเช็ค query.isSuccess/query.data เอง) — ใช้ใน nav-bar.tsx
// เพื่อโชว์ชื่อ user ที่ login อยู่ และปุ่ม logout
export function useAuth() {
  const { data: user, isLoading, isError } = useMe();

  return {
    user: user ?? null,
    isLoading,
    isAuthenticated: !isError && user !== undefined,
  };
}
