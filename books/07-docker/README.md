# Volume 7 — Docker

จุดเริ่มต้นของ containerization เป็นพื้นฐานที่ต้องแน่นก่อนไปเรียน [Kubernetes](../08-kubernetes/README.md) เพราะ K8s ก็คือระบบที่ orchestrate container จำนวนมากนั่นเอง

## เป้าหมายการเรียนรู้

- เขียน Dockerfile ที่ build image ได้เล็ก ปลอดภัย และ build เร็ว (layer caching)
- ใช้ Docker Compose จัดการ multi-container application สำหรับ local development
- เข้าใจ networking และ volume ของ Docker เพียงพอที่จะ debug ปัญหา container สื่อสารกันไม่ได้ หรือข้อมูลหายเมื่อ container restart

## สารบัญ

### [บทที่ 1 — Images และ Dockerfile](01-images-and-dockerfile.md)

- [ ] Image layer, union filesystem, image caching
- [ ] Image tag, digest, registry (Docker Hub, Harbor — เชื่อมกับ [Volume 16 — CI/CD](../16-cicd/README.md))
- [ ] Multi-stage build เพื่อลดขนาด image
- [ ] Instruction หลัก: `FROM`, `RUN`, `COPY`, `ADD`, `CMD`, `ENTRYPOINT`, `WORKDIR`, `ENV`, `ARG`
- [ ] Layer caching — เรียงลำดับ instruction อย่างไรให้ build เร็ว
- [ ] Best practice: non-root user, `.dockerignore`, image เล็ก (alpine/distroless)

### [บทที่ 2 — Compose](02-compose.md)

- [ ] `docker-compose.yml` structure: services, networks, volumes
- [ ] `depends_on`, healthcheck, environment variable / `.env` file
- [ ] ใช้ Compose สำหรับ local dev environment ที่จำลอง production (Postgres + Redis + service)

### [บทที่ 3 — Networks และ Volumes](03-networks-and-volumes.md)

- [ ] Network driver: bridge, host, none, overlay
- [ ] Container-to-container communication ผ่านชื่อ service (DNS ภายใน Docker network)
- [ ] Port mapping (`-p`) vs internal network
- [ ] Named volume vs bind mount vs tmpfs
- [ ] Data persistence — ทำไม container ที่ไม่มี volume แล้ว restart ข้อมูลหาย
- [ ] Volume driver, backup strategy เบื้องต้น

## Lab ที่เกี่ยวข้อง

- [labs/docker/](../../labs/docker/README.md)

## แหล่งอ้างอิงแนะนำ

ดู [resources/](../../resources/README.md)
