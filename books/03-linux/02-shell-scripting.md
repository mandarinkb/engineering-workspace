# บทที่ 2: Shell Scripting

## 2.1 bash

### โครงสร้างพื้นฐาน

```bash
#!/bin/bash
# shebang บอกว่าให้รันไฟล์นี้ด้วย bash

name="Somchai"                    # กำหนดตัวแปร ห้ามมีช่องว่างรอบ =
echo "Hello, $name"               # เรียกใช้ตัวแปรด้วย $

if [ "$name" = "Somchai" ]; then  # เว้นวรรครอบ [ ] เสมอ
    echo "Match"
elif [ -z "$name" ]; then         # -z = string ว่าง
    echo "Empty"
else
    echo "No match"
fi

for i in 1 2 3; do
    echo "Number: $i"
done

while read -r line; do
    echo "Line: $line"
done < input.txt

my_function() {
    local result=$1               # $1 คือ argument ตัวแรก, local จำกัด scope ในฟังก์ชัน
    echo "Got: $result"
}
my_function "hello"
```

### Exit Code

ทุกคำสั่งใน shell คืนค่า **exit code** (0-255) เก็บไว้ในตัวแปรพิเศษ `$?` — `0` แปลว่าสำเร็จ ค่าอื่นแปลว่ามีปัญหา (ความหมายขึ้นกับแต่ละโปรแกรม):

```bash
grep "error" app.log
if [ $? -eq 0 ]; then
    echo "พบคำว่า error ใน log"
fi

# แบบสั้น ใช้ && (ทำต่อถ้าสำเร็จ) และ || (ทำต่อถ้าล้มเหลว)
grep "error" app.log && echo "พบ error" || echo "ไม่พบ error"
```

exit code สำคัญมากในบริบท CI/CD ([Volume 16](../16-cicd/README.md)) — pipeline จะหยุดและ mark ว่า fail ทันทีที่คำสั่งใดคืน exit code ไม่เป็น 0 (ถ้า pipeline ตั้งค่าแบบ fail-fast)

### Pipe และ Redirection

```bash
command1 | command2       # ส่ง stdout ของ command1 เป็น stdin ของ command2

command > file.txt        # เขียน stdout ลงไฟล์ (ทับของเดิม)
command >> file.txt       # เขียน stdout ต่อท้ายไฟล์ (append)
command 2> error.log      # เขียน stderr ลงไฟล์แยก
command > out.log 2>&1    # รวม stdout และ stderr ลงไฟล์เดียวกัน (ลำดับสำคัญ!)
command > /dev/null 2>&1  # ทิ้งทั้ง output และ error ไปเลย
```

## 2.2 grep

ค้นหา pattern ในข้อความ:

```bash
grep "error" app.log           # หาบรรทัดที่มีคำว่า error
grep -i "error" app.log        # ไม่สนตัวพิมพ์เล็ก/ใหญ่ (case-insensitive)
grep -v "debug" app.log        # แสดงบรรทัดที่ "ไม่มี" คำว่า debug (invert match)
grep -r "TODO" ./src           # ค้นแบบ recursive ในทุกไฟล์ใต้ directory
grep -n "error" app.log        # แสดงเลขบรรทัดด้วย
grep -c "error" app.log        # นับจำนวนบรรทัดที่ match
grep -E "error|fatal" app.log  # extended regex — ใช้ | สำหรับ "หรือ" ได้โดยไม่ต้อง escape
```

## 2.3 awk

ประมวลผลข้อมูลแบบแบ่งเป็น field (column) ตาม delimiter (default คือ whitespace):

```bash
# log format: 2026-08-01 10:00:00 ERROR service-a "connection timeout"
awk '{print $3}' app.log            # พิมพ์ field ที่ 3 (ERROR) ของทุกบรรทัด
awk '{print $1, $2}' app.log        # พิมพ์ field 1 และ 2 (วันที่ + เวลา)
awk -F',' '{print $2}' data.csv     # ใช้ comma เป็น delimiter แทน whitespace
awk '$3 == "ERROR" {print}' app.log # แสดงเฉพาะบรรทัดที่ field 3 เท่ากับ ERROR
awk '{sum += $5} END {print sum}' access.log   # รวมค่าคอลัมน์ 5 ทั้งไฟล์ (เช่น response size)
awk 'END {print NR}' app.log        # NR = จำนวนบรรทัดทั้งหมด
```

`awk` เหมาะกับข้อมูลที่มีโครงสร้างเป็น column ชัดเจน เช่น สรุปสถิติจาก access log ที่ `grep`/`sed` ทำได้ไม่สะดวก

## 2.4 sed

**sed (Stream EDitor)** แก้ไขข้อความแบบ non-interactive ทีละบรรทัด เหมาะกับ automation:

```bash
sed 's/error/ERROR/' app.log         # แทนที่คำแรกที่เจอในแต่ละบรรทัด (substitute)
sed 's/error/ERROR/g' app.log        # แทนที่ทุกจุดที่เจอในบรรทัด (global)
sed -i 's/foo/bar/g' config.yaml     # แก้ไฟล์ตรงๆ ในที่ (in-place) — ใช้อย่างระวัง เพราะเขียนทับไฟล์จริง
sed '/^#/d' config.conf              # ลบบรรทัดที่ขึ้นต้นด้วย # (comment) ออก
sed -n '10,20p' app.log              # แสดงเฉพาะบรรทัด 10 ถึง 20
```

> คำแนะนำ: ก่อนใช้ `-i` (in-place) กับไฟล์สำคัญ ควรทดสอบคำสั่งโดยไม่ใส่ `-i` ก่อนเสมอ เพื่อดู output ว่าตรงตามที่ต้องการก่อนจะเขียนทับไฟล์จริง

## 2.5 curl

เครื่องมือทดสอบและ debug HTTP API ที่สำคัญที่สุดตัวหนึ่งของ backend engineer (เชื่อมกับ [Volume 2 — HTTP](../02-network/03-http.md)):

```bash
curl https://api.example.com/users              # GET request พื้นฐาน
curl -X POST https://api.example.com/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Somchai"}'                        # POST พร้อม JSON body

curl -H "Authorization: Bearer $TOKEN" ...       # ใส่ header เอง (เช่น JWT — Volume 9)

curl -I https://api.example.com                  # ดูแค่ response header (HEAD request)
curl -v https://api.example.com                  # verbose — เห็นทั้ง TLS handshake, request/response header ทั้งหมด (ดี๊บักตอน connection มีปัญหา)
curl -o result.json https://api.example.com/data # บันทึก response ลงไฟล์
curl -w "%{http_code} %{time_total}s\n" -o /dev/null -s https://api.example.com  # วัด status code + เวลาที่ใช้
```

`curl -v` คือเครื่องมือแรกที่ควรใช้เวลาสงสัยว่า "ทำไม request ไม่สำเร็จ" เพราะเห็นได้ทั้ง DNS resolution, TCP connect, TLS handshake, และ HTTP header แบบเต็ม — ครอบคลุมทุก layer ที่เรียนมาใน [Volume 2](../02-network/README.md)

## 2.6 jq

Parse และ filter ข้อมูล JSON จาก command line — ใช้คู่กับ `curl` บ่อยมากเวลา debug REST API:

```bash
curl -s https://api.example.com/users | jq '.'              # pretty-print JSON
curl -s https://api.example.com/users | jq '.[0].name'      # ดึงค่า field name ของ item แรก
curl -s https://api.example.com/users | jq '.[] | .id'      # ดึง id ของทุก item ใน array
curl -s https://api.example.com/users | jq '.[] | select(.active == true)'  # กรองเฉพาะที่ active
curl -s https://api.example.com/users | jq 'length'         # นับจำนวน item ใน array
curl -s https://api.example.com/users | jq -r '.[0].email'  # -r = raw output (ไม่มี quote ครอบ)
```

## สรุปย่อ

| เครื่องมือ | ใช้ทำอะไร |
|---|---|
| bash | ตรรกะควบคุมโปรแกรม (condition, loop, function), exit code, pipe/redirect |
| grep | ค้นหา pattern ในข้อความ |
| awk | ประมวลผลข้อมูลแบบ column/field-based |
| sed | แก้ไขข้อความแบบ non-interactive, in-place edit |
| curl | ยิง HTTP request ทดสอบ/debug API |
| jq | parse/filter JSON จาก command line |

## คำถามทบทวน

1. `command > out.log 2>&1` กับ `command 2>&1 > out.log` ให้ผลต่างกันอย่างไร? (คำใบ้: ลำดับของ redirection มีผล)
2. เขียนคำสั่ง `awk` ที่รวมยอดคอลัมน์ที่ 5 ของไฟล์ CSV โดยใช้ comma เป็น delimiter
3. ทำไม `curl -v` ถึงเป็นเครื่องมือแรกที่ควรใช้เวลา debug ปัญหา "เชื่อมต่อ API ไม่ได้"?
4. ใช้ `jq` ดึง field `email` ของ item ที่ `active == true` เท่านั้นจาก array JSON เขียนคำสั่งเต็ม

---

ก่อนหน้า: [บทที่ 1 — Linux Fundamentals](01-linux-fundamentals.md) | ถัดไป: [บทที่ 3 — Git](03-git.md)
