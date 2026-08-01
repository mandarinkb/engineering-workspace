# บทที่ 3: Networks และ Volumes

## 3.1 Network Drivers

Docker รองรับ network driver หลายแบบ แต่ละแบบเหมาะกับสถานการณ์ต่างกัน:

| Driver | ลักษณะการทำงาน | ใช้เมื่อไหร่ |
|---|---|---|
| **bridge** | สร้าง virtual network ภายใน host เดียว container ที่อยู่ network เดียวกันคุยกันได้ผ่าน internal IP/DNS, ออกสู่ภายนอกผ่าน NAT | ค่า default และที่ใช้บ่อยที่สุดสำหรับรัน container บนเครื่องเดียว (local dev, single-host deployment) |
| **host** | container ใช้ network namespace เดียวกับ host โดยตรง ไม่มีการแยก/NAT เลย | ต้องการ performance สูงสุด (ไม่มี overhead ของ NAT) หรือ container ต้องเข้าถึง network interface ของ host ตรงๆ — แลกมาด้วยการเสีย network isolation ทั้งหมด |
| **none** | container ไม่มี network interface ใดๆ เลยนอกจาก loopback | งานที่ไม่ต้องการ network เลยด้วยเหตุผลด้าน security (เช่น batch job ประมวลผลไฟล์ล้วนๆ) |
| **overlay** | สร้าง virtual network ที่ครอบคลุมหลาย host พร้อมกัน (multi-host networking) | Docker Swarm หรือ container orchestration ข้ามหลายเครื่อง — แนวคิดเดียวกับที่ Kubernetes ใช้ CNI plugin ทำ pod network ข้าม node (ดู [Volume 8 — Kubernetes](../08-kubernetes/README.md)) |

สำหรับงานส่วนใหญ่ในหลักสูตรนี้ (local dev ด้วย Compose) **bridge** คือ driver ที่ใช้งานจริงเกือบตลอด

## 3.2 Container-to-Container Communication ผ่านชื่อ Service

เมื่อ container หลายตัวอยู่ใน **custom bridge network เดียวกัน** (ไม่ใช่ default bridge ที่ Docker สร้างให้อัตโนมัติตอนไม่ประกาศ network เอง) Docker จะรัน embedded DNS server ให้ภายใน network นั้น ทำให้ container คุยกันด้วย**ชื่อ service/container name** แทนที่จะต้องรู้ IP address ตายตัว — เหมือนกับที่ [DNS](../02-network/02-dns.md) ทำหน้าที่แปลงชื่อเป็น IP ในระดับ internet แต่ที่นี่เป็นระดับ network ของ Docker เอง

```
custom bridge network "backend"
┌─────────────────────────────────────────────────┐
│                                                   │
│   api ──── DNS query "postgres" ────► ตอบ IP     │
│    │                                    ภายใน     │
│    └──── connect ด้วย IP ที่ได้ ──────► postgres  │
│                                                   │
│   redis (อีก container ใน network เดียวกัน)      │
└─────────────────────────────────────────────────┘
```

ตัวอย่างจาก [บทที่ 2](02-compose.md): service `api` เชื่อมต่อ PostgreSQL ผ่าน `DB_HOST: postgres` โดยไม่รู้ IP จริงเลย — Docker resolve ชื่อ `postgres` เป็น IP ของ container นั้นให้อัตโนมัติ ข้อดีที่ตามมาคือถ้า container `postgres` ถูก restart แล้วได้ IP ใหม่ (ซึ่งเกิดขึ้นได้เสมอ) `api` ก็ยังเชื่อมต่อได้ถูกต้องโดยไม่ต้องแก้ configuration ใดๆ เพราะอ้างอิงผ่านชื่อเสมอ ไม่ใช่ IP ตายตัว

**ข้อควรระวัง**: กลไกนี้ทำงานเฉพาะใน**custom network** เท่านั้น container ที่อยู่ใน default bridge network (ที่เกิดจากไม่ประกาศ `networks:` ใน Compose แล้ว Docker ใช้ default ให้) จะ**ไม่มี** DNS resolution ด้วยชื่อ container ให้ใช้ ต้องพึ่ง `--link` แบบเก่า (deprecated) หรือ IP ตรงๆ เท่านั้น — นี่คือเหตุผลหนึ่งที่ Compose แนะนำให้ประกาศ custom network เสมอ

## 3.3 Port Mapping (`-p`) vs Internal-only Communication

ต้องแยกให้ชัดระหว่างสองแนวคิด:

- **Internal communication** — container คุยกันเองภายใน network เดียวกัน (เช่น `api` → `postgres` port 5432) **ไม่ต้อง** publish port ออกสู่ host เลย เพราะทั้งคู่อยู่ใน network เดียวกันอยู่แล้ว มองเห็นกันโดยตรง
- **Port mapping (`-p host_port:container_port`)** — เปิด port ของ container ให้เข้าถึงได้จาก**ภายนอก host** (เช่นจาก browser บนเครื่อง developer) ผ่าน `localhost:host_port`

ในตัวอย่าง Compose บทที่แล้ว:

```yaml
api:
  ports:
    - "8080:8080"   # เปิดให้เรียกจากนอก container ได้ (curl localhost:8080)

postgres:
  # ไม่มี ports: เลย — เข้าถึงได้เฉพาะจาก container อื่นใน network "backend" เท่านั้น
```

`postgres` ไม่ได้ประกาศ `ports:` เลย เพราะไม่มีเหตุผลให้เครื่องนอกเข้าถึง database ตรงๆ — นี่คือหลัก **least privilege** ระดับ network: เปิดเฉพาะ port ที่จำเป็นต้องเข้าถึงจากภายนอกจริงๆ เท่านั้น ส่วนที่เหลือให้คุยกันแบบ internal-only ลด attack surface ของระบบทั้งหมด

## 3.4 Volumes: ทำไมต้องมี

### Writable Layer เป็นของชั่วคราว

ย้อนกลับไปที่กลไก layer ใน [บทที่ 1](01-images-and-dockerfile.md#12-layer-และ-union-filesystem): container ที่กำลังรันมี **writable layer** ของตัวเองอยู่บนสุด ทุกไฟล์ที่เขียนขณะ container รันอยู่ (เช่น data ของ database, upload file) ถูกเก็บไว้ที่ layer นี้ **เมื่อ container ถูกลบ (`docker rm`) writable layer นั้นถูกลบไปด้วย — ข้อมูลทั้งหมดที่ไม่ได้อยู่ใน volume จะหายทันที** แม้จะสั่งแค่ `docker stop`/`docker start` ใหม่ก็ยังปลอดภัยอยู่ (writable layer ยังอยู่) แต่ `docker rm` หรือ `docker compose down` (ที่ลบ container จริง) จะทำให้ข้อมูลหายอย่างถาวร

**Volume** คือกลไกที่ "ดึง" ตำแหน่งเก็บข้อมูลนั้นออกมาจาก writable layer ของ container ไปเก็บไว้ในที่ที่**อยู่นอกวงจรชีวิตของ container** แทน — container จะถูกสร้างใหม่/ลบไปกี่ครั้งก็ตาม ข้อมูลใน volume ยังอยู่เสมอ

### เปรียบเทียบ Named Volume, Bind Mount, tmpfs

| | Named Volume | Bind Mount | tmpfs |
|---|---|---|---|
| ที่เก็บจริง | จัดการโดย Docker เอง (มักอยู่ที่ `/var/lib/docker/volumes/`) | path ใดก็ได้บน host ที่ผู้ใช้ระบุเอง | RAM ของ host เท่านั้น ไม่แตะ disk |
| ใครควบคุม path | Docker | ผู้ใช้ (รู้ path เต็มบน host) | ไม่มี path บน host เลย |
| Persistence | อยู่ถาวรจนกว่าจะสั่งลบ volume เอง | อยู่ถาวรตราบใดที่ path บน host ยังอยู่ | หายทันทีที่ container หยุด |
| Portability | ย้ายข้าม host ยากกว่า (ผูกกับ Docker บนเครื่องนั้น) แต่จัดการง่ายผ่าน `docker volume` command | ผูกกับโครงสร้าง path ของ host เครื่องนั้นเป๊ะๆ ย้ายเครื่องแล้ว path อาจไม่ตรง | ไม่เกี่ยวกับ persistence เลย |
| Use case ทั่วไป | ข้อมูล database ใน production, ข้อมูลที่ Docker ควรเป็นคนจัดการเต็มรูปแบบ | mount source code จาก host เข้า container เพื่อ hot-reload ตอน dev | เก็บข้อมูล sensitive ชั่วคราวที่ไม่ต้องการให้ตกค้างบน disk เลย (เช่น temp key) |

```yaml
services:
  api:
    volumes:
      - ./src:/app/src              # bind mount — dev only, hot-reload จากเครื่อง host
  postgres:
    volumes:
      - pgdata:/var/lib/postgresql/data   # named volume — production-grade persistence
    tmpfs:
      - /tmp                         # tmpfs — ข้อมูลชั่วคราวใน RAM ล้วนๆ

volumes:
  pgdata:
```

**หลักการเลือก**: ใช้ **named volume** สำหรับข้อมูลที่ต้องคงอยู่จริง (database, message queue state) เพราะ Docker จัดการ lifecycle ให้เอง; ใช้ **bind mount** เฉพาะตอน dev ที่ต้องการให้การแก้โค้ดบนเครื่อง host สะท้อนเข้า container ทันที; ใช้ **tmpfs** เมื่อไม่ต้องการให้ข้อมูลตกค้างบน disk เลยด้วยเหตุผลความปลอดภัยหรือ performance (RAM เร็วกว่า disk มาก)

## 3.5 Volume Backup Strategy เบื้องต้น

แม้ named volume จะอยู่ถาวรกว่า writable layer แต่ก็ยังอยู่บน disk ของ host เครื่องเดียว — ถ้า host เครื่องนั้นพัง ข้อมูลก็หายไปด้วย แนวทาง backup พื้นฐานที่ใช้กันทั่วไป:

- **Snapshot ผ่าน container ชั่วคราว** — รัน container เสริมที่ mount volume เดียวกันแบบ read-only แล้ว `tar` เนื้อหาออกมาเป็นไฟล์ archive เก็บไว้ที่อื่น (เช่น object storage)
- **Database-native backup tool** — สำหรับ database โดยเฉพาะ ควรใช้เครื่องมือของ database เอง (เช่น `pg_dump` สำหรับ PostgreSQL — ดู [Volume 5 — PostgreSQL](../05-postgresql/README.md)) แทนการ copy ไฟล์ดิบ เพราะรับประกัน consistency ของข้อมูลระหว่าง backup ได้ดีกว่าการ copy ไฟล์ตรงๆ ขณะ database กำลังเขียนข้อมูลอยู่
- ใน production ระดับ Kubernetes แนวคิดนี้ขยายไปเป็นการ backup ผ่าน StorageClass/CSI snapshot ที่ผูกกับ infrastructure จริง (ดู [Volume 8 — Kubernetes](../08-kubernetes/README.md))

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| Network Driver | bridge = ค่า default บน host เดียว, host = ไม่มี isolation, none = ไม่มี network, overlay = ข้ามหลาย host |
| DNS ภายใน Docker network | container ใน custom network คุยกันด้วยชื่อ service ได้เลย ไม่ต้องรู้ IP (ใช้ได้เฉพาะ custom network) |
| Port mapping vs internal | `-p`/`ports:` เปิดสู่ภายนอก host, ไม่ประกาศ = เข้าถึงได้เฉพาะภายใน network เดียวกัน (least privilege) |
| Writable layer ephemeral | ข้อมูลที่เขียนโดยไม่ผ่าน volume หายไปพร้อม container เมื่อถูกลบ |
| Named Volume / Bind Mount / tmpfs | named volume = Docker จัดการ เหมาะ production, bind mount = ผูก path host เหมาะ dev, tmpfs = RAM ล้วน ไม่ persist |
| Backup | ใช้เครื่องมือ native ของ database เพื่อความ consistency แทนการ copy ไฟล์ดิบ |

## คำถามทบทวน

1. ทำไม container สองตัวที่อยู่ใน default bridge network ของ Docker ถึงคุยกันด้วยชื่อ container ไม่ได้ ต้องทำอย่างไรถึงจะใช้ได้
2. อธิบายว่าทำไม `postgres` ใน docker-compose.yml ตัวอย่างไม่จำเป็นต้องประกาศ `ports:` ทั้งที่ `api` ยังเชื่อมต่อ `postgres` ได้ปกติ
3. ถ้ารัน PostgreSQL ใน container โดยไม่ mount volume เลย แล้วสั่ง `docker compose down` จะเกิดอะไรขึ้นกับข้อมูลในนั้น เพราะเหตุใด
4. เลือกระหว่าง named volume, bind mount, tmpfs ให้เหมาะกับสถานการณ์ต่อไปนี้: (ก) เก็บข้อมูล production database (ข) mount source code เพื่อ hot-reload ตอน dev (ค) เก็บ temporary decryption key ที่ไม่ต้องการให้ตกค้างบน disk
5. ทำไมการ backup database ด้วยเครื่องมือ native อย่าง `pg_dump` ถึงปลอดภัยกว่าการ copy ไฟล์ดิบจาก volume ตรงๆ ขณะ database กำลังทำงานอยู่

---

ก่อนหน้า: [บทที่ 2 — Compose](02-compose.md) | กลับสู่ [สารบัญ Volume 7](README.md)
