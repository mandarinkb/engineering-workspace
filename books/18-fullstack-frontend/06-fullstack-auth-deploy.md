# บทที่ 6: Fullstack — Go + Next.js, Auth, Deploy

บทนี้เป็นจุดที่ความรู้ backend ของคุณกลายเป็นข้อได้เปรียบ developer ที่เป็น frontend อย่างเดียวมักติดเรื่อง CORS, cookie และ deploy แต่สำหรับคุณ ส่วนที่ใหม่มีแค่ "สิ่งที่ **browser** บังคับใช้" ซึ่ง curl หรือ Postman ไม่ได้บังคับ

## 6.1 สถาปัตยกรรม: Frontend แยกจาก Backend

```text
                    ┌──────────────── Browser ────────────────┐
                    │  HTML + JS (React)                       │
                    └──────┬─────────────────────────┬─────────┘
            โหลดหน้า (HTML) │                         │ fetch API (JSON + cookie)
                           ▼                         ▼
                ┌────────────────────┐     ┌────────────────────┐     ┌──────────┐
                │ Next.js :3000      │────►│ Go API :8080       │────►│ Postgres │
                │ (SSR, static, BFF) │ SSR │ (business logic,   │     └──────────┘
                └────────────────────┘fetch│  auth, validation) │
                                           └────────────────────┘
```

มี 2 เส้นทางที่ request วิ่งไป Go:

1. **Browser → Go โดยตรง** (client component + TanStack Query) — เป็น cross-origin (`:3000` → `:8080`) **ติด CORS** ต้องส่ง cookie ข้าม origin
2. **Next server → Go** (server component, route handler) — server คุยกับ server **ไม่มี CORS** แต่ cookie ของ user ไม่ได้ติดไปเอง ต้องส่งต่อเอง

ทางเลือกที่ทำให้ง่ายขึ้นมาก: ให้ทั้งสองอยู่หลัง **reverse proxy ตัวเดียว** (nginx/APISIX/Ingress) — `example.com/` → Next, `example.com/api/` → Go แบบนี้กลายเป็น same-origin ไม่มี CORS และ cookie ทำงานตรงไปตรงมา (เชื่อมกับ [Volume 10 — APISIX](../10-apisix/README.md))

## 6.2 CORS — สิ่งที่ Browser บังคับ

**Origin** = scheme + host + port → `http://localhost:3000` กับ `http://localhost:8080` คือคนละ origin

**Same-Origin Policy:** JS ที่โหลดจาก origin A อ่าน response จาก origin B **ไม่ได้** เว้นแต่ B บอกว่าอนุญาตผ่าน header CORS — นี่คือการป้องกันฝั่ง browser ไม่ให้เว็บอื่นแอบใช้ cookie ของคุณอ่านข้อมูลจาก API ของคุณ

สิ่งที่ backend engineer มักเข้าใจผิด:

- request **ถึง** server แล้ว (ดูใน log ของ Go ได้) แต่ browser ไม่ยอมให้ JS อ่าน response
- curl/Postman ไม่ติด CORS เพราะไม่ใช่ browser — CORS ไม่ใช่ security ของ server
- ต้องแก้ที่ **server** (ใส่ header) แก้ที่ frontend ไม่ได้ (นอกจากทำ proxy)

### Preflight

ถ้า request ไม่ "ธรรมดา" (method เป็น PUT/PATCH/DELETE, หรือมี `Content-Type: application/json`, หรือมี header เอง) browser จะส่ง `OPTIONS` ไปถามก่อน:

```text
OPTIONS /tasks/1
Origin: http://localhost:3000
Access-Control-Request-Method: PATCH
Access-Control-Request-Headers: content-type

← 204
Access-Control-Allow-Origin: http://localhost:3000
Access-Control-Allow-Methods: GET, POST, PATCH, DELETE
Access-Control-Allow-Headers: Content-Type
Access-Control-Allow-Credentials: true
Access-Control-Max-Age: 600          ← cache ผล preflight 10 นาที
```

### CORS middleware ใน Go ที่ถูกต้องสำหรับ cookie

```go
func CORS(allowedOrigins map[string]bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if allowedOrigins[origin] {
				// ส่ง cookie (credentials) ด้วย → ห้ามใช้ "*" ต้องระบุ origin ตรงๆ
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Add("Vary", "Origin") // บอก cache ว่า response ต่างกันตาม Origin
			}
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
				w.Header().Set("Access-Control-Max-Age", "600")
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
```

ฝั่ง frontend ต้องใส่ `credentials: "include"` ทุก request ไม่งั้น browser จะไม่ส่ง cookie ข้าม origin (และไม่ยอมรับ `Set-Cookie` ที่ตอบกลับมา)

> **ดูของจริง:** `projects/bot-orchestration-platform/internal/adapter/http/handler.go` ช่วงบรรทัด ~60–70 และ `projects/bop-dashboard/src/shared/api/api-client.ts` (`credentials: "include"`)

## 6.3 Auth: JWT ใน httpOnly Cookie (ฝั่ง Go)

### เก็บ token ไว้ที่ไหน?

| ที่เก็บ | XSS (script แปลกปลอมในหน้า) | CSRF (เว็บอื่นยิง request แทนเรา) | หมายเหตุ |
|---|---|---|---|
| `localStorage` | ❌ **ขโมย token ได้ทันที** | ✅ ปลอดภัย (ไม่ส่งอัตโนมัติ) | ต้องแนบ `Authorization` เองทุก request |
| **httpOnly cookie** | ✅ JS อ่านไม่ได้ | ⚠️ ต้องป้องกันด้วย `SameSite` | browser แนบให้อัตโนมัติ |

สำหรับ web app แนะนำ **httpOnly cookie + `SameSite=Lax`** (หรือ `Strict`) — XSS ยังทำอันตรายได้ (ยิง request ในชื่อ user ขณะเปิดหน้าอยู่) แต่ขโมย token ออกไปใช้ภายหลังไม่ได้ ส่วน `Authorization: Bearer` เหมาะกับ mobile app / service-to-service

### Endpoint

```go
const cookieName = "session"

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct{ Username, Password string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	user, err := h.users.Authenticate(r.Context(), req.Username, req.Password) // bcrypt compare
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	token, _ := h.jwt.Sign(user.ID, 24*time.Hour)

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,                 // JS อ่านไม่ได้
		Secure:   h.isProduction,       // production ต้อง HTTPS
		SameSite: http.SameSiteLaxMode, // ไม่ส่ง cookie กับ POST ที่มาจากเว็บอื่น
		MaxAge:   int((24 * time.Hour).Seconds()),
	})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
	w.WriteHeader(http.StatusNoContent)
}

// GET /me — frontend ใช้เช็คว่า login อยู่ไหมและเป็นใคร
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context()) // ใส่ไว้โดย auth middleware
	json.NewEncoder(w).Encode(user)
}

func (h *Handler) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(cookieName)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		claims, err := h.jwt.Verify(c.Value)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(WithUserID(r.Context(), claims.UserID)))
	})
}
```

### เรื่อง SameSite กับ localhost

`localhost:3000` กับ `localhost:8080` เป็น **cross-origin แต่ same-site** (site ดูที่ domain ไม่ดู port) `SameSite=Lax` จึงใช้ได้ตอน dev ถ้า production อยู่คนละ domain จริง (เช่น `app.com` กับ `api.other.com`) ต้องใช้ `SameSite=None; Secure` — หรือดีกว่านั้น ใช้ subdomain ของ domain เดียวกัน (`app.example.com` / `api.example.com`) หรือ reverse proxy ให้เป็น origin เดียว

> **ดูของจริง:** `projects/bot-orchestration-platform/internal/adapter/http/handler.go` ช่วงบรรทัด ~305–370 (คอมเมนต์อธิบายเรื่อง SameSite/Secure ไว้ละเอียด) และ `internal/pkg/jwt/jwt.go`

## 6.4 Auth ฝั่ง Next.js

### Hook ข้อมูล user

```ts
// entities/user/api.ts
export function useMe() {
  return useQuery({
    queryKey: ["me"],
    queryFn: () => api<User>("/me"),
    retry: false,                 // 401 = ยังไม่ login ไม่ต้อง retry
    staleTime: 5 * 60_000,
  });
}

export function useLogin() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { username: string; password: string }) =>
      api<void>("/login", { method: "POST", body }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["me"] }),
  });
}

export function useLogout() {
  const qc = useQueryClient();
  const router = useRouter();
  return useMutation({
    mutationFn: () => api<void>("/logout", { method: "POST" }),
    onSuccess: () => {
      qc.clear();                 // ล้าง cache ทั้งหมด — ข้อมูลของ user เดิมต้องไม่ค้าง
      router.push("/login");
    },
  });
}
```

### จัดการ 401 กลางที่เดียว

เมื่อ cookie หมดอายุกลางคัน API ตัวไหนก็อาจตอบ 401 ดักไว้ที่ `QueryCache`/`MutationCache` ตัวเดียวแทนการเช็คทุกหน้า:

```ts
new QueryClient({
  queryCache: new QueryCache({
    onError: (error) => {
      if (error instanceof ApiError && error.status === 401) window.location.href = "/login";
    },
  }),
});
```

### ชั้นของการป้องกัน

```text
1. middleware/proxy.ts   มี cookie ไหม? ไม่มี → redirect /login        (UX: ไม่เห็นหน้าวาบ)
2. useMe() ใน layout     GET /me ผ่านไหม? 401 → redirect /login        (UX: รู้ว่าเป็นใคร)
3. Go RequireAuth        verify JWT ทุก request                          (✅ security จริง)
```

ถ้าลบชั้น 1–2 ทิ้ง ระบบยังปลอดภัย ถ้าลบชั้น 3 ทิ้ง ระบบไม่ปลอดภัยทันที

> **ดูของจริง:** `projects/bop-dashboard/middleware.ts`, `src/features/auth/`, `src/entities/user/model/use-auth.ts`, `src/entities/user/api/use-me.ts`

## 6.5 Environment Variable: ฝั่ง Server vs ฝั่ง Client

```bash
# .env.local (ห้าม commit) — ทำ .env.example ไว้ให้คนอื่นรู้ว่าต้องตั้งอะไร
NEXT_PUBLIC_API_URL=http://localhost:8080      # ← browser ใช้ (client component)
API_INTERNAL_URL=http://localhost:8080         # ← server ใช้ (server component, route handler)
```

| | `NEXT_PUBLIC_*` | ไม่มี prefix |
|---|---|---|
| อ่านได้ที่ | server + **browser** | server เท่านั้น |
| ถูกฝังเมื่อไหร่ | **ตอน `next build`** (แทนค่าลงไปใน JS bundle) | อ่านตอน runtime |
| ใส่ secret ได้ไหม | ❌ **ใครก็เปิดดูใน DevTools ได้** | ✅ |

กับดักที่เจอบ่อยตอนทำ Docker: `NEXT_PUBLIC_*` ต้องส่งเป็น **build arg** ตอน `docker build` — ตั้งเป็น env ตอน `docker run` **ไม่มีผล** เพราะค่าถูกฝังลงไฟล์ JS ไปแล้ว

ทำไมต้องมี 2 URL: ใน Docker Compose, browser เรียก Go ผ่าน `http://localhost:8080` (port ที่ publish ออกมาบนเครื่อง) แต่ container ของ Next เรียก Go ผ่านชื่อ service `http://api:8080` ในเครือข่ายของ compose

## 6.6 Docker Compose และ Deploy

### Dockerfile ของ Next.js (multi-stage + standalone)

```ts
// next.config.ts
const nextConfig = { output: "standalone" };   // bundle เฉพาะไฟล์ที่ runtime ต้องใช้
export default nextConfig;
```

```dockerfile
FROM node:22-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

FROM node:22-alpine AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
ARG NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL
RUN npm run build

FROM node:22-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
RUN addgroup -S nodejs && adduser -S nextjs -G nodejs
COPY --from=builder /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static
USER nextjs
EXPOSE 3000
CMD ["node", "server.js"]
```

แนวคิดเหมือน multi-stage ของ Go (build ใน image ใหญ่ ส่งแค่ผลลัพธ์ไป image เล็ก) ต่างกันที่ runtime ยังต้องมี Node

### docker-compose.yml

```yaml
services:
  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: tasks
      POSTGRES_PASSWORD: tasks
      POSTGRES_DB: tasks
    volumes: [dbdata:/var/lib/postgresql/data]
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U tasks"]
      interval: 5s

  api:
    build: ./tasks-api
    environment:
      DATABASE_URL: postgres://tasks:tasks@db:5432/tasks?sslmode=disable
      JWT_SECRET: change-me
      ALLOWED_ORIGINS: http://localhost:3000
    ports: ["8080:8080"]
    depends_on:
      db: { condition: service_healthy }

  web:
    build:
      context: ./next-tasks
      args:
        NEXT_PUBLIC_API_URL: http://localhost:8080   # browser เรียกผ่าน port ที่ publish
    environment:
      API_INTERNAL_URL: http://api:8080              # server ของ Next เรียกผ่านเครือข่าย compose
    ports: ["3000:3000"]
    depends_on: [api]

volumes:
  dbdata:
```

> **รันที่ไหน:** Mac เครื่องนี้ไม่มี Docker — ใช้ Mini PC (`ssh minipc`) เรียก compose ตัวระบบ `/usr/libexec/docker/cli-plugins/docker-compose` และเช็คพอร์ตว่างด้วย `ss -ltn` ก่อน (มีโปรเจกต์อื่นรันอยู่) หรือถ้า browser อยู่บน Mac ให้เปลี่ยน `localhost` ในค่าข้างบนเป็น IP ของ Mini PC

### ตัวเลือก deploy สำหรับ portfolio

| ส่วน | ตัวเลือกง่าย |
|---|---|
| Next.js | **Vercel** (ฟรี, push GitHub แล้ว deploy เอง) |
| Go API + Postgres | Render / Fly.io / Railway (มี free/cheap tier), หรือ VPS + compose |
| ทั้งหมดที่เดียว | VPS + Docker Compose + Caddy (HTTPS อัตโนมัติ) — โชว์ทักษะ infra ได้ด้วย |

ถ้า frontend กับ backend อยู่คนละ domain ให้กลับไปอ่านเรื่อง `SameSite=None; Secure` ในหัวข้อ 6.3

## สรุปบท

- browser → Go = cross-origin ต้องมี CORS ที่ระบุ origin + `Allow-Credentials` และ frontend ใส่ `credentials: "include"`
- Next server → Go ไม่มี CORS แต่ต้องใช้ URL ภายใน
- JWT ใน httpOnly cookie + SameSite สำหรับ web app; security จริงอยู่ที่ Go เท่านั้น
- `NEXT_PUBLIC_*` ถูกฝังตอน build และทุกคนเห็น — ห้ามใส่ secret
- reverse proxy ตัวเดียวทำให้ปัญหา CORS/cookie หายไปเกือบหมด
