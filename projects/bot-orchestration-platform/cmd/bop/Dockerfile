# syntax=docker/dockerfile:1

# Multi-stage build — stage แรก compile binary ด้วย image ที่มี Go toolchain เต็ม
# (~800MB) stage สุดท้ายมีแค่ binary + CA certificates (~20MB) ไม่ลาก toolchain ติดไปด้วย
FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# CGO_ENABLED=0 บังคับ static binary (ไม่ผูกกับ libc ของ image ต้นทาง) จำเป็นเพราะ
# stage สุดท้ายใช้ distroless ที่ไม่มี dynamic linker ให้เลย
RUN CGO_ENABLED=0 GOOS=linux go build -o /bop ./cmd/bop

FROM gcr.io/distroless/static-debian12:nonroot
# distroless "nonroot" variant รันเป็น UID 65532 อัตโนมัติอยู่แล้ว (ไม่ต้องสร้าง user เอง
# เหมือน image ทั่วไป) — ไม่มี shell/package manager ในนี้เลยด้วยซ้ำ (ลด attack surface
# ให้เหลือน้อยที่สุด ตรงกับ Phase 2.7 Security ที่วางแผนไว้)
COPY --from=builder /bop /bop
EXPOSE 8080
ENTRYPOINT ["/bop"]
