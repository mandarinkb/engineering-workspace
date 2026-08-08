# syntax=docker/dockerfile:1

# โครงเดียวกับ cmd/bop/Dockerfile เป๊ะ แค่ build target ต่างกัน (./cmd/bop-worker) — ไม่มี
# EXPOSE port หลักเพราะ process นี้ไม่รับ HTTP traffic จากภายนอก มีแค่ health server เล็กๆ
# บน :9090 ไว้ตอบ liveness probe (ดู cmd/bop-worker/main.go) จึง EXPOSE เฉพาะ port นั้น
FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bop-worker ./cmd/bop-worker

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /bop-worker /bop-worker
EXPOSE 9090
ENTRYPOINT ["/bop-worker"]
