# บทที่ 2: Compose

## 2.1 ทำไมต้องมี Compose

แอปพลิเคชันจริงแทบไม่เคยรันเป็น container เดียวโดดๆ — ต้องมี database, cache, และอาจมีอีกหลาย service ประกอบกัน การรัน `docker run` ทีละคำสั่งยาวๆ ซ้ำๆ ทุกครั้งที่เริ่มงานเป็นเรื่องน่าเบื่อและผิดพลาดง่าย **Docker Compose** แก้ปัญหานี้ด้วยการอธิบาย multi-container application ทั้งชุดไว้ในไฟล์ YAML ไฟล์เดียว แล้วสั่ง `docker compose up` ครั้งเดียวเพื่อสร้างทุกอย่างพร้อมกัน — เครือข่าย, volume, และ container ทุกตัว — ตามที่กำหนดไว้

## 2.2 โครงสร้างของ `docker-compose.yml`

ไฟล์ Compose แบ่งเป็น 3 ส่วนหลัก:

- **`services`** — แต่ละ service คือ container หนึ่งประเภท (image ที่ใช้, port, environment, dependency)
- **`networks`** — เครือข่ายที่ให้ service คุยกันได้ (ดูรายละเอียดกลไก DNS ใน [บทที่ 3](03-networks-and-volumes.md)) — ถ้าไม่ประกาศเอง Compose จะสร้าง default bridge network ให้อัตโนมัติ ทุก service ในไฟล์เดียวกันจะอยู่ใน network นั้นและคุยกันด้วยชื่อ service ได้ทันที
- **`volumes`** — พื้นที่เก็บข้อมูลถาวรที่ไม่หายไปพร้อม container (ดูรายละเอียดใน [บทที่ 3](03-networks-and-volumes.md))

## 2.3 ตัวอย่างเต็ม: Go API + PostgreSQL + Redis

```yaml
# docker-compose.yml
services:
  api:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    environment:
      DB_HOST: postgres
      DB_PORT: "5432"
      DB_USER: ${POSTGRES_USER}
      DB_PASSWORD: ${POSTGRES_PASSWORD}
      DB_NAME: ${POSTGRES_DB}
      REDIS_ADDR: redis:6379
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    networks:
      - backend

  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_DB: ${POSTGRES_DB}
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER}"]
      interval: 5s
      timeout: 3s
      retries: 5
    networks:
      - backend

  redis:
    image: redis:7-alpine
    volumes:
      - redisdata:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5
    networks:
      - backend

networks:
  backend:
    driver: bridge

volumes:
  pgdata:
  redisdata:
```

จุดที่ควรสังเกต: `api` ไม่ได้เชื่อมต่อ PostgreSQL ด้วย IP address ใดๆ เลย แต่ใช้ `DB_HOST: postgres` — ตรงนี้คือ service discovery ผ่าน DNS ภายใน Docker network ซึ่งอธิบายกลไกละเอียดใน [บทที่ 3](03-networks-and-volumes.md)

## 2.4 `depends_on` และข้อจำกัดสำคัญ

`depends_on` ควบคุม**ลำดับการ start** container — บอก Compose ว่า service `api` ต้องรอให้ `postgres` และ `redis` ถูกสร้างขึ้นก่อน แต่มีข้อจำกัดที่ทำให้ engineer จำนวนมากเจอบั๊กที่ debug ยาก:

**`depends_on` แบบพื้นฐาน (ไม่มี `condition`) รอแค่ "container เริ่มทำงาน (started)" ไม่ได้รอว่า service ข้างในนั้น "พร้อมรับ connection จริง (ready)"**

PostgreSQL container อาจ start แล้วในสถานะ "running" แต่กระบวนการ initialize database ภายใน (สร้าง data directory, run migration ของตัว PostgreSQL เอง) ยังไม่เสร็จ ทำให้ยัง accept connection ไม่ได้จริงๆ อีกหลายวินาที ถ้า `api` container เชื่อมต่อทันทีที่ start ตาม `depends_on` ธรรมดา จะได้ connection refused แม้ Compose จะรายงานว่าลำดับ start ถูกต้องแล้วก็ตาม

### Healthcheck คือทางแก้ที่ถูกต้อง

จากตัวอย่างข้างบน `depends_on` ใช้รูปแบบ:

```yaml
depends_on:
  postgres:
    condition: service_healthy
```

ควบคู่กับการประกาศ `healthcheck` ใน service `postgres`/`redis` เอง — Compose จะรัน `test` command ที่กำหนดไว้ซ้ำๆ ตาม `interval` จนกว่าจะสำเร็จติดต่อกันตามเงื่อนไข แล้วถึงจะ mark container ว่า `healthy` **`api` จะไม่ start จนกว่า `postgres`/`redis` รายงานสถานะ `healthy` จริงๆ ไม่ใช่แค่ `started`** ซึ่งแก้ปัญหา race condition ข้างต้นได้ตรงจุด

## 2.5 Environment Variables และ `.env` File

ค่าที่ใช้ `${VARIABLE}` ใน `docker-compose.yml` (เช่น `${POSTGRES_USER}`) ถูกดึงมาจากไฟล์ **`.env`** ที่วางอยู่ใน directory เดียวกับ `docker-compose.yml` โดยอัตโนมัติ:

```bash
# .env
POSTGRES_USER=appuser
POSTGRES_PASSWORD=supersecret
POSTGRES_DB=appdb
```

การแยกค่าที่เปลี่ยนตาม environment (username, password, connection string) ออกจากตัว compose file ทำให้:

- ไฟล์ `docker-compose.yml` เดียวใช้ได้กับหลาย environment (local, staging) แค่เปลี่ยนไฟล์ `.env` ที่ใช้
- `.env` ควรอยู่ใน `.gitignore` เสมอ (ไม่ commit ค่า secret ลง repository) — โดยทั่วไปจะ commit ไฟล์ตัวอย่าง `.env.example` ที่มีแค่ชื่อ key ไว้แทน ให้คนอื่น copy ไปกรอกค่าเอง

## 2.6 Compose สำหรับจำลอง Production-like Local Dev Environment

ประโยชน์ที่สำคัญที่สุดของ Compose ไม่ใช่แค่ "รันหลาย container พร้อมกัน" แต่คือการทำให้ทุกคนในทีมมี environment สำหรับพัฒนาที่**ใกล้เคียง production มากที่สุดเท่าที่จะทำได้** โดยไม่ต้องติดตั้ง PostgreSQL/Redis ลงเครื่องตัวเองโดยตรง (ซึ่งมักนำไปสู่ปัญหา "ที่เครื่องผมรันได้" เพราะ version ของ dependency ต่างจาก production)

แนวทางที่แนะนำ:

- ใช้ **image tag เดียวกับที่ production ใช้จริง** (เช่น `postgres:16-alpine` ตัวเดียวกับที่ deploy จริงใน Kubernetes) เพื่อลดโอกาสที่พฤติกรรมจะต่างกันระหว่าง local กับ production
- แยก `docker-compose.override.yml` สำหรับค่าที่ใช้เฉพาะ local (เช่น mount source code เป็น bind mount เพื่อ hot-reload) โดยไม่ปนกับ `docker-compose.yml` หลักที่สะท้อนโครงสร้างจริง — Compose จะ merge สองไฟล์นี้ให้อัตโนมัติเมื่อสั่ง `docker compose up`
- ใช้ healthcheck ทุก service ที่มี dependency กัน (ดังตัวอย่างข้างบน) เพื่อให้ startup ของ local environment เสถียร ไม่ flaky เวลาสั่ง `up` รอบใหม่ทุกเช้า

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| `docker-compose.yml` | รวม services/networks/volumes ไว้ไฟล์เดียว, `up` ครั้งเดียวสร้างทุกอย่าง |
| Default network | service ในไฟล์เดียวกันอยู่ network เดียวกันอัตโนมัติ คุยกันด้วยชื่อ service ได้ |
| `depends_on` (ธรรมดา) | รอแค่ container "started" ไม่ได้รอว่า service "ready" รับ connection ได้จริง |
| Healthcheck + `condition: service_healthy` | ทางแก้ที่ถูกต้อง — รอจนกว่า service จะพร้อมจริงตาม test ที่กำหนด |
| `.env` | แยก secret/config ออกจาก compose file, ไม่ commit ลง git, ใช้ `.env.example` แทน |
| Production-like local dev | ใช้ image tag เดียวกับ production, แยก override file สำหรับของเฉพาะ local |

## คำถามทบทวน

1. อธิบายว่าทำไม `depends_on` แบบไม่มี `condition` ถึงไม่รับประกันว่า service ที่ต้องพึ่งพาจะพร้อมรับ connection
2. Healthcheck ของ PostgreSQL ในตัวอย่างใช้คำสั่งอะไร ทำงานอย่างไร และ Compose ใช้ผลลัพธ์นั้นตัดสินใจอะไรต่อ
3. ทำไมไฟล์ `.env` ถึงไม่ควรถูก commit ลง git แต่ `.env.example` กลับควร commit
4. ถ้าไม่ประกาศ `networks` เองใน compose file เลย service ต่างๆ ยังคุยกันได้ไหม เพราะเหตุใด
5. การใช้ image tag เดียวกับ production ใน local Compose environment ช่วยลดปัญหาอะไรที่พบบ่อยในการพัฒนาซอฟต์แวร์เป็นทีม

---

ก่อนหน้า: [บทที่ 1 — Images และ Dockerfile](01-images-and-dockerfile.md) | ถัดไป: [บทที่ 3 — Networks และ Volumes](03-networks-and-volumes.md)
