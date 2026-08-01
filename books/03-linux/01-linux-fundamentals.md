# บทที่ 1: Linux Fundamentals

## 1.1 Filesystem

### FHS (Filesystem Hierarchy Standard)

Linux จัดวางไฟล์ตามมาตรฐาน FHS เพื่อให้ทุก distro มีโครงสร้างคล้ายกัน คาดเดาได้:

```
/
├── bin, sbin   → โปรแกรมพื้นฐานที่จำเป็นตอน boot (มักลิงก์ไป /usr/bin)
├── etc         → ไฟล์ configuration ของระบบ
├── home        → home directory ของ user แต่ละคน
├── var         → ข้อมูลที่เปลี่ยนแปลงบ่อย (log, cache, spool) → /var/log สำคัญมากสำหรับ debug
├── tmp         → ไฟล์ชั่วคราว ถูกล้างเมื่อ reboot (บางระบบ)
├── usr         → โปรแกรมและไลบรารีของ user-space ส่วนใหญ่
├── proc        → virtual filesystem แสดงข้อมูล process/kernel แบบ real-time (ไม่ใช่ไฟล์จริงบน disk)
└── sys         → virtual filesystem แสดงข้อมูล device/kernel (คล้าย /proc แต่เน้น hardware)
```

`/proc` และ `/sys` สำคัญกับ engineer มาก เพราะเป็นหน้าต่างมองเข้าไปใน kernel โดยตรง เช่น `/proc/<pid>/status` ดู memory ของ process หนึ่ง, `/proc/meminfo` ดู memory รวมทั้งระบบ, `/proc/cpuinfo` ดูข้อมูล CPU — เชื่อมกับความรู้ process/memory จาก [Volume 1](../01-computer-foundation/README.md)

### Mount Point

Filesystem อื่น (partition, disk อีกลูก, network share) ถูกนำมา "ต่อ" เข้ากับ tree เดียวของ `/` ที่ตำแหน่งที่เรียกว่า **mount point** ผ่านคำสั่ง `mount` — ต่างจาก Windows ที่มี drive letter แยก (C:, D:) Linux มองทุกอย่างเป็น tree เดียว ไม่ว่าข้อมูลจริงจะอยู่ disk ไหนก็ตาม ตรวจสอบ mount ปัจจุบันด้วย `df -h` หรือ `mount`

### Hard Link vs Symbolic Link

- **Hard link** — ชื่อไฟล์อีกชื่อหนึ่งที่ชี้ไปยัง **inode เดียวกัน** (ดูหัวข้อ 1.2) เหมือนไฟล์เดียวกันมีสองชื่อ ลบชื่อไหนก็ไม่กระทบอีกชื่อ (ตราบใดที่ยังมีชื่ออื่นเหลืออยู่) สร้างด้วย `ln target linkname`
- **Symbolic link (symlink)** — ไฟล์แยกต่างหากที่เก็บ "path" ไปยังไฟล์เป้าหมาย คล้าย shortcut ถ้าไฟล์ต้นทางถูกลบ symlink จะกลายเป็น "broken link" ทันที สร้างด้วย `ln -s target linkname`

## 1.2 inode

**inode** คือโครงสร้างข้อมูลที่เก็บ **metadata** ของไฟล์หนึ่งไฟล์ — permission, owner, timestamp (created/modified/accessed), ขนาดไฟล์, และ pointer ไปยัง data block ที่เก็บเนื้อหาไฟล์จริงบน disk

### สิ่งสำคัญ: inode ไม่เก็บชื่อไฟล์

ชื่อไฟล์ถูกเก็บแยกต่างหากใน **directory entry** ซึ่งเป็นแค่ mapping ระหว่าง "ชื่อ" กับ "inode number":

```
Directory entry (ใน /home/user/)        inode table
┌─────────────────┬──────────┐         ┌────────┬─────────────────────┐
│ report.txt       │  12345   │────────►│ 12345  │ permission, owner,  │
│ report_backup.txt│  12345   │────────►│        │ timestamp, size,    │
│                   │          │         │        │ → pointer to data   │
└─────────────────┴──────────┘         └────────┴─────────────────────┘
```

ดู inode number ของไฟล์ด้วย `ls -i` หรือดู metadata เต็มด้วย `stat filename`

### ทำไม `rm` ไฟล์ที่ process เปิดอยู่ แล้ว disk ยังไม่ว่าง

นี่คือคำถามคลาสสิกที่ทุก backend/infra engineer ต้องเจอสักวัน เหตุผลคือ inode มี **link count** (จำนวนชื่อที่ชี้มาที่ inode นี้) และแยกกันกับ**จำนวน process ที่เปิดไฟล์นี้ค้างอยู่ (open file descriptor)**

`rm` ไม่ได้ "ลบข้อมูล" ทันที — มันแค่**ลบ directory entry** (unlink) ทำให้ link count ลดลง 1 **data block จะถูกคืนกลับให้ระบบก็ต่อเมื่อ link count = 0 และไม่มี process ไหนเปิดไฟล์นั้นค้างอยู่แล้ว**

สถานการณ์จริง: service เขียน log ไปที่ `/var/log/app.log` ค้างไว้ (มี file descriptor เปิดอยู่) ถ้าไปสั่ง `rm /var/log/app.log` ตรงๆ โดยไม่ restart service ไฟล์จะหายจาก `ls` ทันที แต่ **disk space ไม่ว่างขึ้น** เพราะ service ยังเขียนข้อมูลลง data block เดิมต่อไปเรื่อยๆ ผ่าน file descriptor ที่ยังเปิดอยู่ (ถึงแม้ inode จะไม่มีชื่อผูกอยู่แล้วก็ตาม) วิธีแก้ที่ถูกต้องคือ truncate ไฟล์ (`> /var/log/app.log` หรือใช้ logrotate) หรือ restart process ให้เปิด file descriptor ใหม่

ตรวจสอบไฟล์ที่ถูกลบแต่ยังถูกเปิดค้างได้ด้วย `lsof +L1`

## 1.3 systemd

**systemd** คือ init system มาตรฐานของ distro หลักในปัจจุบัน (Ubuntu, RHEL/CentOS, Debian) ทำหน้าที่เป็น process แรกที่ OS รัน (PID 1) และจัดการ service ทั้งหมดในระบบ

### Unit File

การกำหนดค่า service หนึ่งตัวเรียกว่า **unit** เขียนเป็นไฟล์ `.service` (มี unit type อื่นด้วย เช่น `.socket`, `.target`, `.timer`):

```ini
[Unit]
Description=My Backend API Service
After=network.target postgresql.service

[Service]
ExecStart=/usr/bin/my-api-server
Restart=on-failure
User=appuser

[Install]
WantedBy=multi-user.target
```

- `[Unit]` — metadata และ dependency (`After=` แค่กำหนดลำดับ start ไม่ได้บังคับว่าต้องมี service นั้นอยู่จริง; `Requires=` บังคับว่าต้องมี dependency นั้นทำงานอยู่ด้วย)
- `[Service]` — วิธีรัน process จริง, restart policy
- `[Install]` — ผูกกับ target ไหนตอน enable (คล้าย runlevel สมัยก่อน เช่น `multi-user.target` = โหมดทำงานปกติแบบไม่มี GUI)

### คำสั่งที่ใช้บ่อย

```bash
systemctl start my-api        # เริ่ม service ตอนนี้
systemctl stop my-api         # หยุด service
systemctl restart my-api      # หยุดแล้วเริ่มใหม่
systemctl status my-api       # ดูสถานะปัจจุบัน + log ล่าสุด
systemctl enable my-api       # ตั้งให้ start อัตโนมัติตอน boot
systemctl daemon-reload       # บอก systemd ให้อ่าน unit file ที่แก้ไขใหม่
```

## 1.4 journalctl

**journalctl** คือเครื่องมืออ่าน log จาก **systemd journal** ซึ่งเก็บ log แบบ binary structured format (ต่างจาก syslog แบบข้อความล้วนสมัยก่อน) รวม log จากทุก service ที่รันผ่าน systemd ไว้ที่เดียว

```bash
journalctl -u my-api              # ดู log เฉพาะ service นี้
journalctl -u my-api -f           # follow log แบบ real-time (เหมือน tail -f)
journalctl -u my-api --since "1 hour ago"
journalctl -u my-api -p err       # กรองเฉพาะระดับ error ขึ้นไป
journalctl -b                     # log ตั้งแต่ boot ครั้งล่าสุด
journalctl -b -1                  # log ของ boot ครั้งก่อนหน้า (เช่น ดูว่าทำไม server ล่ม/reboot)
```

## 1.5 ps

**ps** แสดง snapshot ของ process ที่กำลังรันอยู่ ณ ขณะนั้น (เชื่อมกับแนวคิด process จาก [Volume 1](../01-computer-foundation/README.md))

```bash
ps aux       # แสดงทุก process ของทุก user, format BSD-style
ps -ef       # แสดงทุก process, format UNIX-style, เน้นแสดง PPID ชัดเจน
ps --forest  # แสดงเป็น tree ดูความสัมพันธ์ parent-child ได้ง่าย
```

### อ่านคอลัมน์สำคัญ

| คอลัมน์ | ความหมาย |
|---|---|
| PID / PPID | process ID / parent process ID |
| %CPU / %MEM | สัดส่วนการใช้ CPU/memory |
| VSZ | virtual memory size (รวม memory ที่ map ไว้แต่อาจยังไม่ได้ใช้จริง) |
| RSS | Resident Set Size — memory จริงที่อยู่ใน RAM ตอนนี้ (ตัวเลขที่มักสนใจมากกว่า VSZ) |
| STAT | สถานะ process: `R` running, `S` sleeping (interruptible), `D` uninterruptible sleep (มักรอ I/O — ถ้าค้างนานผิดปกติคือสัญญาณปัญหา disk/network), `Z` zombie (จบแล้วแต่ parent ยังไม่ `wait()` เก็บ), `T` stopped |

## 1.6 top

**top** แสดงข้อมูล process แบบ real-time พร้อม refresh อัตโนมัติ (ต่างจาก `ps` ที่เป็น snapshot ครั้งเดียว)

### Load Average

ตัวเลข 3 ค่าที่มุมบนของ `top` (เช่น `load average: 2.5, 1.8, 0.9`) คือค่าเฉลี่ยจำนวน process ที่ "รันอยู่หรือรอคิว CPU" ในช่วง 1, 5, 15 นาทีที่ผ่านมา

**จุดสำคัญที่ต้องตีความให้ถูก**: load average ต้องเทียบกับ**จำนวน CPU core** เสมอ ไม่ใช่ดูตัวเลขลอยๆ

- Server มี 4 core, load average = 4.0 → CPU ถูกใช้เต็มพอดี (ไม่มีคิวค้าง)
- Server มี 4 core, load average = 8.0 → มี process รอคิวเฉลี่ย 4 ตัวตลอดเวลา ระบบเริ่มโอเวอร์โหลด
- Server มี 4 core, load average = 1.0 → ใช้ CPU แค่ 25% ของกำลังทั้งหมด

## 1.7 ss

**ss (socket statistics)** คือเครื่องมือรุ่นใหม่ที่แทนที่ `netstat` (เร็วกว่าเพราะอ่านข้อมูลตรงจาก kernel ผ่าน netlink แทนที่จะ parse `/proc`) ใช้ดู socket/connection ทั้งหมดในระบบ (เชื่อมกับ [Volume 2 — Network](../02-network/README.md))

```bash
ss -tuln     # t=tcp, u=udp, l=listening only, n=numeric (ไม่ resolve เป็นชื่อ) — ดูว่า port ไหนกำลัง listen
ss -tnp      # แสดง process (PID/program name) ที่ผูกกับแต่ละ connection
ss -tn state established   # ดูเฉพาะ connection ที่ established อยู่
```

ตัวอย่างการ debug จริง: service รายงานว่า "connect refused" ไปยัง port 5432 (PostgreSQL) — เช็คด้วย `ss -tuln | grep 5432` ว่ามีอะไร listen อยู่บน port นั้นจริงไหม ถ้าไม่มีเลยแปลว่า database service ไม่ได้รันอยู่หรือ bind ผิด interface

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| Filesystem | ทุกอย่างเป็น tree เดียวจาก `/`, `/proc` `/sys` คือหน้าต่างมองเข้า kernel |
| inode | เก็บ metadata + pointer ข้อมูล, **ไม่เก็บชื่อไฟล์** — ชื่ออยู่ที่ directory entry |
| rm ไม่คืน disk ทันที | ต้องทั้ง link count = 0 **และ** ไม่มี open file descriptor ค้าง |
| systemd | จัดการ service ผ่าน unit file, `systemctl` เป็นเครื่องมือหลัก |
| journalctl | อ่าน structured log จาก systemd journal, กรองตาม unit/เวลา/level ได้ |
| ps / top | `ps` = snapshot, `top` = real-time; load average ต้องเทียบกับจำนวน core |
| ss | ดู socket/connection แทน netstat รุ่นเก่า |

## คำถามทบทวน

1. อธิบายว่าทำไม `rm` ไฟล์ log ที่ service เปิดค้างอยู่ ถึงไม่ทำให้ disk space ว่างขึ้นทันที และวิธีแก้ที่ถูกต้องคืออะไร
2. hard link กับ symbolic link ต่างกันอย่างไรในแง่ inode?
3. `VSZ` กับ `RSS` ใน `ps` ต่างกันอย่างไร ตัวไหนสะท้อน memory ที่ "ใช้จริง" มากกว่า?
4. Server มี 8 core, load average อยู่ที่ 12.0 บอกอะไรเกี่ยวกับสถานะระบบ?
5. `journalctl -u my-service -p err --since "1 hour ago"` ทำอะไร แยกส่วนความหมายทีละ flag

---

ถัดไป: [บทที่ 2 — Shell Scripting](02-shell-scripting.md)
