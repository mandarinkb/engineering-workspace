# บทที่ 1: Images และ Dockerfile

## 1.1 Image คืออะไร

**Docker image** คือ template แบบ read-only ที่ประกอบด้วยทุกอย่างที่โปรแกรมต้องใช้ในการรัน — binary, library, dependency, configuration, และคำสั่งเริ่มต้น (entrypoint) เมื่อสั่ง `docker run` บน image หนึ่งตัว Docker จะสร้าง **container** ขึ้นมาเป็น instance ที่รันจริง โดยเพิ่ม writable layer บางๆ ทับบน image นั้น (ดูหัวข้อ 1.2)

จุดสำคัญที่ต้องเข้าใจตั้งแต่ต้น: **image ไม่ใช่ก้อนไฟล์เดียว** แต่เป็นการซ้อนกันของ **layer** หลายชั้น แต่ละ layer คือ diff ของ filesystem จาก layer ก่อนหน้า — นี่คือกลไกที่ทำให้ Docker build เร็ว, แชร์ layer ข้าม image ได้, และ push/pull เฉพาะส่วนที่เปลี่ยนแทนที่จะโอนทั้ง image ใหม่ทุกครั้ง

## 1.2 Layer และ Union Filesystem

แต่ละ instruction ใน Dockerfile ที่แก้ filesystem (`RUN`, `COPY`, `ADD`) จะสร้าง layer ใหม่หนึ่งชั้น โดย Docker ใช้ **union filesystem** (โดยทั่วไปคือ **OverlayFS** บน Linux) เพื่อรวมหลาย layer ที่แยกเก็บจริงบน disk ให้ process มองเห็นเป็น filesystem เดียวที่สมบูรณ์

```
Container ที่กำลังรัน
┌─────────────────────────────────────┐
│   Writable Layer (container layer)   │  ← เขียนได้ ทุกการเปลี่ยนแปลงขณะรันอยู่ที่นี่
├─────────────────────────────────────┤
│   Layer 4: COPY . .                  │  ← read-only จาก image
├─────────────────────────────────────┤
│   Layer 3: RUN npm install           │  ← read-only
├─────────────────────────────────────┤
│   Layer 2: COPY package.json .       │  ← read-only
├─────────────────────────────────────┤
│   Layer 1: FROM node:20-alpine       │  ← read-only (base image)
└─────────────────────────────────────┘
        ▲ มองจาก process ข้างในเห็นเป็น filesystem เดียว (OverlayFS merged view)
```

หลักการทำงานที่สำคัญคือ **copy-on-write (CoW)**: เมื่อ process ใน container แก้ไขไฟล์ที่มาจาก layer ล่างๆ (read-only) ระบบจะ copy ไฟล์นั้นขึ้นมาที่ writable layer ก่อน แล้วแก้ที่สำเนานั้นแทน — layer เดิมไม่ถูกแตะต้อง ทำให้ layer เดียวกันแชร์ใช้ร่วมกันได้ระหว่างหลาย container/image พร้อมกันโดยไม่ชนกัน

**ผลลัพธ์เชิงปฏิบัติที่ตามมา 2 อย่าง:**

1. **Layer ที่เหมือนกันแชร์กันได้** — ถ้า image สองตัวใช้ `FROM node:20-alpine` เหมือนกัน layer นั้นถูก pull/เก็บบน disk แค่ครั้งเดียว ไม่ว่าจะมีกี่ image ที่อ้างถึง
2. **Writable layer เป็นของ container นั้นๆ เพียงตัวเดียว และหายไปเมื่อ container ถูกลบ** — เป็นเหตุผลว่าทำไมข้อมูลที่เขียนลง container โดยไม่ผ่าน volume ถึงหายเมื่อ `docker rm` (รายละเอียดเต็มอยู่ใน [บทที่ 3 — Volumes](03-networks-and-volumes.md))

## 1.3 Image Tag vs Digest

Image หนึ่งตัวถูกอ้างอิงได้สองแบบ:

- **Tag** — ชื่อที่มนุษย์อ่านง่าย เช่น `myapp:1.4.0`, `myapp:latest` — **tag เป็น mutable pointer** เปลี่ยนให้ชี้ไป image ตัวใหม่ได้ตลอดเวลา (เช่น push image ใหม่ทับ tag เดิม) นี่คือกับดักคลาสสิก: `latest` วันนี้กับ `latest` เมื่อวานอาจเป็นคนละ image กันโดยสิ้นเชิง
- **Digest** — hash แบบ SHA-256 ที่คำนวณจากเนื้อหา manifest ของ image เช่น `myapp@sha256:a1b2c3...` — **immutable** เนื้อหาเปลี่ยนแม้นิดเดียว digest เปลี่ยนทันที เหมือนกับ commit hash ใน Git

```
myapp:latest              → tag, ชี้ image ตัวไหนก็ได้ เปลี่ยนได้ตลอด
myapp@sha256:a1b2c3...    → digest, ระบุ image ตัวเดียวเป๊ะๆ เปลี่ยนไม่ได้
```

**Best practice สำหรับ production**: pin ด้วย digest (หรืออย่างน้อย tag แบบ semantic version ที่ตายตัว เช่น `1.4.0` ไม่ใช่ `latest`) เพื่อรับประกันว่า deployment ทุกครั้ง — ทั้งใน CI/CD และเวลา scale pod ใหม่ใน Kubernetes — ได้ image เนื้อหาเดียวกันเป๊ะๆ ไม่ใช่ "image ล่าสุด ณ ตอนที่บังเอิญ pull"

## 1.4 Registry

**Registry** คือที่เก็บและกระจาย image ผ่าน `docker push`/`docker pull` โดย public registry ที่นิยมที่สุดคือ **Docker Hub** ซึ่งเหมาะกับ base image สาธารณะ (เช่น `node`, `postgres`, `alpine`) แต่สำหรับ image ของบริษัทเอง แนวทางที่ปลอดภัยและควบคุมได้กว่าคือ private registry ภายในองค์กร

(เชื่อมกับ [Volume 16 — CI/CD](../16-cicd/README.md) ที่ใช้ **Harbor** เป็น private container registry — นอกจากเก็บ image แล้ว Harbor ยังทำ vulnerability scanning และ image signing ให้อัตโนมัติ ซึ่งเป็นขั้นตอนมาตรฐานก่อน image จะถูกอนุญาตให้ deploy ขึ้น production)

## 1.5 Dockerfile: Instruction หลัก

| Instruction | หน้าที่ |
|---|---|
| `FROM` | กำหนด base image เริ่มต้น |
| `RUN` | รันคำสั่งตอน build (สร้าง layer ใหม่) เช่น install package |
| `COPY` | คัดลอกไฟล์จาก build context เข้า image |
| `ADD` | เหมือน `COPY` แต่เพิ่มความสามารถพิเศษ (แตกไฟล์ archive อัตโนมัติ, ดึงจาก URL) — **แนะนำให้ใช้ `COPY` เป็นค่าเริ่มต้นเสมอ** เพราะพฤติกรรมชัดเจนคาดเดาได้กว่า ใช้ `ADD` เฉพาะตอนต้องการความสามารถพิเศษนั้นจริงๆ |
| `CMD` | คำสั่ง default ที่รันเมื่อ container start — ผู้ใช้ override ได้ง่ายตอน `docker run` |
| `ENTRYPOINT` | คำสั่งหลักที่รันเสมอ ไม่ถูก override ง่ายๆ (มักใช้คู่กับ `CMD` เป็น default argument) |
| `WORKDIR` | กำหนด working directory สำหรับ instruction ถัดไป |
| `ENV` | ตั้ง environment variable ที่ฝังอยู่ใน image ถาวร |
| `ARG` | ตัวแปรที่ใช้ได้เฉพาะตอน build เท่านั้น (ผ่าน `--build-arg`) ไม่ถูกฝังอยู่ใน image ที่รันจริง |

ความต่างระหว่าง `CMD` กับ `ENTRYPOINT` ที่มักสับสน: ถ้าตั้งทั้งคู่ `ENTRYPOINT` จะเป็นโปรแกรมหลักที่รันแน่นอน ส่วน `CMD` กลายเป็น default argument ที่ส่งให้ `ENTRYPOINT` (และผู้ใช้ override argument นี้ได้ตอน `docker run image arg-ใหม่`) รูปแบบนี้เหมาะกับ image ที่ทำหน้าที่เหมือน CLI tool ตัวหนึ่ง

## 1.6 Layer Caching และทำไมลำดับ Instruction ถึงสำคัญ

เวลา build image, Docker จะพิจารณาแต่ละ instruction ทีละบรรทัด **ถ้า instruction นั้นกับ context ของมันไม่เปลี่ยนจากครั้งก่อน Docker จะใช้ layer เดิมจาก cache ทันที** โดยไม่รันคำสั่งซ้ำ — แต่**ทันทีที่ instruction ใดถูก cache miss (invalidate) ทุก instruction ที่ตามมาหลังจากนั้นจะถูก build ใหม่หมด ต่อให้เนื้อหาของมันเองไม่ได้เปลี่ยนเลยก็ตาม** เพราะ layer ถัดไปสร้างต่อจาก layer ก่อนหน้าเสมอ

นี่คือเหตุผลว่าทำไม**ลำดับ instruction** ใน Dockerfile ถึงกระทบความเร็วในการ build โดยตรง หลักการคือ: **วางสิ่งที่เปลี่ยนน้อย/ไม่บ่อยไว้บนสุด วางสิ่งที่เปลี่ยนบ่อยที่สุด (เช่นซอร์สโค้ด) ไว้ล่างสุด**

### ตัวอย่างที่ผิด — ลำดับทำให้ cache แทบไม่มีประโยชน์

```dockerfile
FROM golang:1.22-alpine

WORKDIR /app

COPY . .                    # copy ทั้ง repo — เปลี่ยนแม้แค่ comment ใน .go ไฟล์เดียว ก็ cache miss ที่นี่
RUN go mod download         # ต้องรันใหม่ทุกครั้ง แม้ go.mod/go.sum ไม่ได้เปลี่ยนเลย!
RUN go build -o server .

CMD ["./server"]
```

ปัญหา: `COPY . .` รวมทั้งซอร์สโค้ดที่แก้ทุกวันเข้าไปในบรรทัดเดียวกับ dependency ทำให้แม้แก้แค่ 1 บรรทัดในโค้ด ก็ทำให้ `go mod download` (ซึ่งอาจใช้เวลาหลายสิบวินาทีถึงหลายนาทีถ้า dependency เยอะ) ต้องรันใหม่ทุกครั้งโดยไม่จำเป็น

### ตัวอย่างที่ถูก — แยก dependency ออกจาก source code

```dockerfile
FROM golang:1.22-alpine

WORKDIR /app

COPY go.mod go.sum ./       # copy แค่ dependency manifest ก่อน
RUN go mod download         # layer นี้ cache ได้ ตราบใดที่ go.mod/go.sum ไม่เปลี่ยน

COPY . .                    # ค่อย copy source code ทีหลัง — เปลี่ยนบ่อยสุด อยู่ล่างสุด
RUN go build -o server .

CMD ["./server"]
```

ตอนนี้ถ้าแก้แค่ไฟล์ `.go` โดยไม่แตะ `go.mod`/`go.sum` เลย `RUN go mod download` จะยังคง cache hit อยู่เสมอ — build ถัดไปเร็วขึ้นมากเพราะข้าม step ที่หนักที่สุด (ดาวน์โหลด dependency ผ่าน network) ไปได้ทันที

## 1.7 Multi-stage Build

ปัญหาของการ build ตรงๆ ใน image เดียว: image สุดท้ายจะพ่วง build tool, compiler, source code ดิบ, และ intermediate artifact ทั้งหมดที่ใช้แค่ตอน build แต่ไม่จำเป็นตอนรันจริง — ทำให้ image ใหญ่และมี attack surface สูงกว่าที่ควร (ยิ่งมี tool เยอะ ยิ่งมีช่องโหว่ที่เป็นไปได้เยอะ)

**Multi-stage build** แก้ปัญหานี้ด้วยการมีหลาย `FROM` ใน Dockerfile เดียว โดย stage หลังๆ copy เฉพาะ **ผลลัพธ์สุดท้าย** ที่ต้องการจาก stage ก่อนหน้า ทิ้งทุกอย่างที่เหลือไว้ (compiler, cache, source code) ไม่ให้ตามไปอยู่ใน image สุดท้าย

### ตัวอย่างเต็ม: Go application

```dockerfile
# ---------- Stage 1: build ----------
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/api

# ---------- Stage 2: runtime ----------
FROM alpine:3.19

RUN apk add --no-cache ca-certificates \
    && addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app
COPY --from=builder /server /app/server

USER appuser
EXPOSE 8080
ENTRYPOINT ["/app/server"]
```

สิ่งที่เกิดขึ้น:

- **Stage `builder`** ใช้ image `golang:1.22-alpine` ที่มี Go toolchain เต็ม (ใหญ่ ~300-400MB) เพื่อ compile binary ตัวเดียว — `CGO_ENABLED=0` ทำให้ได้ static binary ที่ไม่ผูกกับ C library ของระบบ รันบน base image เปล่าๆ ได้โดยตรง
- **Stage สุดท้าย** เริ่มจาก `alpine` เปล่าๆ ใหม่ทั้งหมด แล้ว `COPY --from=builder` ดึงมาแค่ไฟล์ binary `/server` เพียงไฟล์เดียว — Go toolchain, source code, module cache ทั้งหมดจาก stage แรกถูกทิ้งไว้ไม่ตามมา
- ผลลัพธ์: image สุดท้ายมีขนาดหลักสิบ MB (เทียบกับหลักร้อย MB ถ้า build ใน image เดียว) และไม่มี compiler/shell tool ที่ไม่จำเป็นให้ผู้บุกรุกใช้ประโยชน์ต่อได้หากเจาะเข้ามาได้

## 1.8 Best Practices

### รันด้วย Non-root User

Container ที่รันเป็น `root` โดย default (ถ้าไม่ระบุ `USER`) — ถ้ามีช่องโหว่ใน application ที่ทำให้ผู้บุกรุก execute คำสั่งได้ ผู้บุกรุกจะมีสิทธิ์ root ภายใน container ทันที ซึ่งเพิ่มความเสี่ยงของ **container breakout** (การหลุดออกจาก container ไปมีสิทธิ์บน host) ถ้ามีช่องโหว่อื่นประกอบเพิ่ม การสร้าง user ธรรมดาแล้วสั่ง `USER appuser` ก่อนจบ Dockerfile (ดังตัวอย่างข้างบน) ลด blast radius ของช่องโหว่ประเภทนี้ได้อย่างมีนัยสำคัญ ตาม principle of least privilege

### `.dockerignore`

ไฟล์ `.dockerignore` (syntax คล้าย `.gitignore`) กำหนดว่าไฟล์/โฟลเดอร์ไหน**ไม่ต้อง**ส่งเข้าไปเป็นส่วนหนึ่งของ **build context** (ข้อมูลทั้งหมดที่ถูกส่งให้ Docker daemon ตอน `docker build .`) เหตุผลที่ต้องมี:

```
.git
node_modules
*.log
.env
tmp/
```

- **ความเร็ว** — build context ที่เล็กลง ส่งเข้า daemon เร็วขึ้น (สำคัญมากถ้า `.git` history ใหญ่ หรือมี `node_modules` ค้างอยู่)
- **ความปลอดภัย** — ป้องกันไม่ให้ไฟล์ sensitive อย่าง `.env`, private key, หรือ `.git` (ซึ่งอาจมี credential เก่าฝังอยู่ใน history) หลุดเข้าไปอยู่ใน layer ของ image โดยไม่ตั้งใจ — เมื่อไฟล์เข้าไปอยู่ใน layer แล้ว การลบทิ้งด้วย `RUN rm` ใน instruction ถัดไป**ไม่ได้ลบออกจาก layer เดิม** (layer ก่อนหน้ายังอยู่ครบใน image, ดูหัวข้อ 1.2) ข้อมูลนั้นยัง extract ออกมาดูได้เสมอถ้ามี image อยู่ในมือ

### เลือก Base Image ให้เล็ก

| Base image | ขนาดโดยประมาณ | ข้อควรระวัง |
|---|---|---|
| `ubuntu`/`debian` | ~70-120MB | ครบเครื่องมือ debug ง่าย แต่ใหญ่และมี package เยอะเกินจำเป็น |
| `alpine` | ~5-8MB | เล็กมาก ใช้ musl libc (ไม่ใช่ glibc) — บาง binary ที่ compile ผูกกับ glibc โดยเฉพาะอาจรันไม่ได้ตรงๆ ต้องเช็คก่อนใช้ |
| `distroless` (Google) | ~2-20MB ขึ้นกับ variant | ไม่มี shell, ไม่มี package manager เลย — เล็กและ attack surface ต่ำสุด แต่ debug ยาก (`docker exec sh` ใช้ไม่ได้) |

หลักการเลือก: image เล็กลง = attack surface เล็กลง (ไม่มี package/tool ที่ไม่ได้ใช้ให้เป็นช่องโหว่) และ pull/deploy เร็วขึ้น (สำคัญมากเวลา scale pod จำนวนมากใน Kubernetes) — แต่ต้องชั่งน้ำหนักกับความสะดวกในการ debug production issue เสมอ ทีมจำนวนมากเลือก `alpine` เป็นจุดกึ่งกลางที่ยังพอมี shell ให้ `exec` เข้าไปดูได้ยามฉุกเฉิน ส่วน `distroless` เหมาะกับ workload ที่ mature แล้วและมี observability (ดู [Volume 14 — Observability](../14-observability/README.md)) ที่ดีพอจนไม่ต้อง `exec` เข้า container บ่อยๆ

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| Layer | image คือ layer ที่ซ้อนกันแบบ read-only, container เพิ่ม writable layer ทับบนสุด (ephemeral) |
| Union Filesystem | OverlayFS รวมหลาย layer เป็น view เดียว ใช้ copy-on-write ตอนแก้ไฟล์จาก layer ล่าง |
| Tag vs Digest | tag เปลี่ยนได้ (mutable), digest ตายตัว (immutable) — production ควร pin ด้วย digest/version คงที่ |
| Registry | Docker Hub สำหรับ public image, Harbor สำหรับ private + scanning ในองค์กร |
| Layer Caching | วาง instruction ที่เปลี่ยนน้อยไว้บน, เปลี่ยนบ่อย (source code) ไว้ล่าง เพื่อ maximize cache hit |
| Multi-stage Build | แยก build stage (มี toolchain) ออกจาก runtime stage (มีแค่ binary) ลดขนาดและ attack surface |
| Best Practices | non-root user ลด blast radius, `.dockerignore` กัน secret หลุดและลด context, base image เล็กลด attack surface |

## คำถามทบทวน

1. อธิบายว่าทำไม `docker rm` container ถึงไม่กระทบ image เดิม แต่ทำไมข้อมูลที่เขียนขณะ container รันอยู่ถึงหายไปพร้อมกับ container นั้น
2. ทำไม pin image ด้วย `latest` tag ถึงเสี่ยงกว่าการ pin ด้วย digest หรือ version number ที่ตายตัว ยกตัวอย่างสถานการณ์ที่เกิดปัญหาจริง
3. ถ้าสลับลำดับใน Dockerfile ตัวอย่าง "ที่ผิด" ในหัวข้อ 1.6 ให้ `COPY . .` มาอยู่หลัง `RUN go mod download` ผลลัพธ์ด้าน caching จะเปลี่ยนไปอย่างไร เพราะเหตุใด
4. Multi-stage build ช่วยลดขนาด image และ attack surface ได้อย่างไร ยกตัวอย่างสิ่งที่ "หายไป" จาก image สุดท้าย
5. ทำไมการลบไฟล์ secret ด้วย `RUN rm` ใน instruction ถัดไปของ Dockerfile ถึงไม่ได้แปลว่าไฟล์นั้นหายไปจาก image จริงๆ

---

ถัดไป: [บทที่ 2 — Compose](02-compose.md)
