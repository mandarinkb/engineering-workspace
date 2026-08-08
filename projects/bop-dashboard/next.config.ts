import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // "standalone" bundle เฉพาะไฟล์ที่ runtime ต้องใช้จริงเข้า .next/standalone/ (รวม
  // node_modules ที่จำเป็นเท่านั้น) production image เลยไม่ต้อง npm install/copy
  // node_modules ทั้งก้อนเข้าไปเลย — ดู KUBERNETES-DEPLOYMENT-TODO.md หัวข้อ 2.3
  output: "standalone",
};

export default nextConfig;
