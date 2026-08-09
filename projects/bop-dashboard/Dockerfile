# syntax=docker/dockerfile:1

FROM node:22-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

FROM node:22-alpine AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
# NEXT_PUBLIC_* ต้อง inline เข้า JS bundle ตอน build เวลานี้เท่านั้น (Next.js อ่านค่าตอน
# `next build` แล้วฝังลงไฟล์ static เลย) ตั้งเป็น ENV ตอน container รันจริงทีหลัง**ไม่มีผล
# อะไรเลย** — ต้องส่งผ่าน --build-arg ตอน `docker build` หรือ `build.args` ใน compose เท่านั้น
ARG NEXT_PUBLIC_API_URL=http://localhost:8080
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL
RUN npm run build

FROM node:22-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
RUN addgroup --system --gid 1001 nodejs && adduser --system --uid 1001 nextjs
COPY --from=builder /app/public ./public
# output: "standalone" (next.config.ts) bundle เฉพาะไฟล์ที่ runtime ต้องใช้จริงมาไว้ที่
# .next/standalone/ — image นี้เลยไม่ต้อง npm install หรือ copy node_modules ทั้งก้อนเข้ามาเลย
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static
USER nextjs
EXPOSE 3000
ENV PORT=3000
CMD ["node", "server.js"]
