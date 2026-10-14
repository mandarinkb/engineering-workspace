import { dirname } from "path";
import { fileURLToPath } from "url";
import { FlatCompat } from "@eslint/eslintrc";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const compat = new FlatCompat({
  baseDirectory: __dirname,
});


// กฎ import ของ Feature-Sliced Design (ดู ARCHITECTURE.md) — ใช้ no-restricted-imports ที่มี
// ใน ESLint อยู่แล้ว ไม่ต้องเพิ่ม plugin: ชั้นล่างห้าม import ชั้นบน, slice ในชั้นเดียวกันห้าม
// import กันเอง, และนอก slice ต้อง import ผ่าน public API (index.ts) เท่านั้น ห้าม import ลึก
const deepImports = ["@/pages/*/*", "@/widgets/*/*", "@/features/*/*", "@/entities/*/*"];

function fsdLayer(files, forbidden) {
  return {
    files,
    rules: {
      "no-restricted-imports": [
        "error",
        {
          patterns: [
            {
              group: deepImports,
              message: "import ผ่าน public API ของ slice (เช่น @/entities/job) ห้าม import ไฟล์ข้างในตรงๆ",
            },
            ...(forbidden.length
              ? [{ group: forbidden, message: "ผิดกฎชั้นของ FSD: ชั้นล่างห้าม import ชั้นบน และ slice ในชั้นเดียวกันห้าม import กันเอง" }]
              : []),
          ],
        },
      ],
    },
  };
}

const fsdRules = [
  fsdLayer(["app/**/*.{ts,tsx}", "middleware.ts"], []),
  fsdLayer(["src/app/**/*.{ts,tsx}"], []),
  fsdLayer(["src/pages/**/*.{ts,tsx}"], ["@/app/*", "@/pages/*"]),
  fsdLayer(["src/widgets/**/*.{ts,tsx}"], ["@/app/*", "@/pages/*", "@/widgets/*"]),
  fsdLayer(["src/features/**/*.{ts,tsx}"], ["@/app/*", "@/pages/*", "@/widgets/*", "@/features/*"]),
  fsdLayer(["src/entities/**/*.{ts,tsx}"], ["@/app/*", "@/pages/*", "@/widgets/*", "@/features/*", "@/entities/*"]),
  fsdLayer(["src/shared/**/*.{ts,tsx}"], ["@/app/*", "@/pages/*", "@/widgets/*", "@/features/*", "@/entities/*"]),
];

const eslintConfig = [
  ...compat.extends("next/core-web-vitals", "next/typescript"),
  {
    ignores: [
      "node_modules/**",
      ".next/**",
      "out/**",
      "build/**",
      "next-env.d.ts",
    ],
  },
  ...fsdRules,
];

export default eslintConfig;
