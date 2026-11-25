# Mini PC และเครื่องมือ — เฉพาะที่ค้นหาเองไม่ได้

กฎการเข้าถึง Mini PC (ssh, ห้ามแตะ container ของโปรเจกต์อื่น, ขออนุญาตก่อนสร้างของใหม่) อยู่ใน `~/.claude/CLAUDE.md` ไม่ซ้ำที่นี่

## Mac ไม่มี Go และ Docker

รัน Go, lint และ Node ใน container ชั่วคราวบน Mini PC โดยส่งซอร์สผ่าน tar ทาง stdin ไม่คัดลอกไฟล์ลงดิสก์ของ Mini PC ใช้ `golangci-lint` เวอร์ชันเดียวกับ Jenkins (`v2.15.0`)

**ใช้ `verify.sh` ในโฟลเดอร์ของ skill นี้แทนคำสั่งยาวด้านล่าง:** `.claude/skills/grill-to-deploy/verify.sh <go|dashboard|all> [--log DIR]` รวมการรอ container ว่าง, การรัน, และสรุปผลสั้นๆ (ผลเต็มอยู่ใน log) ถือว่าผ่านก็ต่อเมื่อเห็นบรรทัดยืนยันของทุกขั้นเท่านั้น · คำสั่งด้านล่างเก็บไว้เป็นอ้างอิงว่าสคริปต์ทำอะไร · ทดสอบกับ Mini PC แล้ว 2026-11-23 (รัน `all` บน `main` ผ่านทั้งชุด ~5 นาที; ทำไฟล์ Go format ผิดแล้วสคริปต์จับได้และ `golangci-lint fmt --diff` exit 1 เมื่อมีความต่าง) ยังไม่เคยเห็น timeout เกิดจริง · test ที่ใช้ PostgreSQL จริงไม่รวมในสคริปต์

```bash
# Go: vet + test + lint + format (จาก root ของ repo)
(cd projects/bot-orchestration-platform && COPYFILE_DISABLE=1 tar --no-mac-metadata --exclude bruno --exclude docs -cf - .) \
 | ssh -o BatchMode=yes minipc 'docker run --rm -i golangci/golangci-lint:v2.15.0 sh -c "mkdir /src && cd /src && tar xf - 2>/dev/null && go vet ./... && echo VET_OK; go test -p 1 ./... 2>&1 | grep -vE \"no test files|downloading\"; golangci-lint run ./... && golangci-lint fmt --diff ./... && echo GO_LINT_OK"'

# dashboard
(cd projects && COPYFILE_DISABLE=1 tar --no-mac-metadata --exclude node_modules --exclude .next -cf - bop-dashboard) \
 | ssh -o BatchMode=yes minipc 'docker run --rm -i node:22-alpine sh -c "mkdir /w && tar xf - -C /w 2>/dev/null && cd /w/bop-dashboard && npm ci --silent >/dev/null 2>&1 && npx vitest run && npx eslint . && echo LINT_OK && npx next build"'
```

test ที่ใช้ PostgreSQL จริง: `postgres:16` ชั่วคราวบน docker network ชั่วคราว ใช้ init script ของ repo สร้าง database แล้วลบทั้ง container และ network เมื่อจบ แม้ล้ม

## ภาระเครื่อง

Mini PC ใช้ร่วมกับ stack จริง subagent หลายตัวที่รัน container พร้อมกันเคยทำให้ load average ขึ้นถึง 95 ตั้งกฎนี้ให้ทุก subagent ตั้งแต่ต้น: รันได้ทีละหนึ่ง container ต่อ subagent และรอเมื่อมี container ตรวจงานตั้งแต่ 3 ตัว

```bash
until [ "$(ssh -o BatchMode=yes minipc 'docker ps -q --filter ancestor=golang:1.26 --filter ancestor=node:22-alpine --filter ancestor=golangci/golangci-lint:v2.15.0 | wc -l')" -lt 3 ]; do sleep 20; done
```

เมื่อ usage limit เต็มกลางงาน subagent หยุดพร้อมกัน งานที่ทำไว้ยังอยู่ใน worktree ใช้ `SendMessage` เรียกแต่ละตัวกลับเมื่อ limit รีเซ็ต

## Deploy

- checkout: `~/Workspace/engineering-workspace` (compose project `engineering-workspace`) ใช้ compose ของระบบที่ `/usr/libexec/docker/cli-plugins/docker-compose` ไม่ใช่ `docker compose` (ตัวนั้นเป็นรุ่นเก่า)
- ค่าสำหรับ LAN อยู่ในไฟล์ **นอก repo** `~/Workspace/bop-lan.override.yml` ทุกคำสั่ง compose ต้องใส่ `-f docker-compose.yml -f ../bop-lan.override.yml` รวมถึงคำสั่งใน `RESTART_CMD`
- gateway ของ Temporal UI เป็นโหมด etcd ต้องการ admin key จากไฟล์ **นอก repo** `~/Workspace/apisix-admin.env` (`APISIX_ADMIN_KEY=...` สิทธิ์ 600) ทุกคำสั่ง compose ต้องใส่ `--env-file ~/Workspace/apisix-admin.env` ถ้าไม่ใส่ gateway จะหยุดและ `:8233` เปิดไม่ได้ (fail closed) · route เก็บใน etcd (volume `engineering-workspace_etcd-data`) หลัง deploy ที่แก้ `docker/temporal-ui-gateway/routes/*.json` ให้รัน `docker/temporal-ui-gateway/seed.sh` บน Mini PC (ดู README ในโฟลเดอร์นั้น)
- ถ้าความต่างจากที่ deploy อยู่เป็นเอกสารล้วน compose จะไม่ recreate container (image เหมือนเดิม) บอกผู้ใช้ตามจริงว่าไม่มี restart

```bash
ssh minipc 'cd ~/Workspace/engineering-workspace && git fetch -q origin && git switch -q <branch> && /usr/libexec/docker/cli-plugins/docker-compose --env-file ~/Workspace/apisix-admin.env -f docker-compose.yml -f ../bop-lan.override.yml up --build -d'
# ตรวจ: docker ps, docker logs engineering-workspace-bop-api-1, curl localhost:8080/readyz, ตาราง schema_migrations ใน database bop
```

PostgreSQL ที่มี volume เดิมไม่รัน init script ซ้ำ: ครั้งแรกที่งานเพิ่ม database ต้องรันคำสั่งสร้างด้วยมือ (ดู README ของ backend) และขออนุญาตผู้ใช้ก่อน

สลับกลับ `main` หลังผู้ใช้ merge: เช็คก่อนว่า `git diff --name-only HEAD origin/main` เป็นไฟล์ในโฟลเดอร์ `docs/` และไฟล์คู่มือเท่านั้น แล้ว `git switch main && git pull --ff-only`

## ข้อมูลทดสอบและการลบ

**ใช้ `cleanup-test-data.sh` ในโฟลเดอร์ของ skill นี้แทนคำสั่งมือด้านล่าง:** `./cleanup-test-data.sh` (dry-run นับและแสดงรายการ) แล้ว `./cleanup-test-data.sh --yes` (ลบ Job + log ใน `bop` และ Workflow ใน Temporal ที่ prefix `bruno-`; เพิ่ม prefix ด้วย `--prefix`) มีตัวกันพลาด (ROLLBACK ถ้าจำนวนไม่ตรงที่นับ, ไม่ลบถ้ามี Workflow Running, ผ่านเมื่อนับซ้ำได้ 0) · ทดสอบกับ Mini PC แล้ว 2026-11-25 · **ลบไม่ได้:** Job ที่เคส `01b-get-job-missing-config` สร้าง (config เป็น `{}` ไม่มี prefix ให้จับ) ยังเหลือบน Mini PC ทีละ 1 รายการต่อรอบ · คำสั่งมือด้านล่างเก็บไว้เป็นอ้างอิงว่าสคริปต์ทำอะไร

Bruno สร้างข้อมูลที่ API ลบไม่ได้ (Job, Workflow Run) ลบเอง ใช้ ID ตรงตัวหรือ prefix ที่ตั้งเอง (`bruno-`, `uiwalk-`) และให้คำสั่งตรวจจำนวนก่อนลบ

```bash
# Temporal (CLI อยู่ใน container ของ temporal): นับก่อน ลบ แล้วนับซ้ำ
docker exec engineering-workspace-temporal-1 temporal workflow count --address temporal:7233 --namespace default --query 'WorkflowId STARTS_WITH "bruno-"'
docker exec engineering-workspace-temporal-1 temporal workflow delete --address temporal:7233 --namespace default --query 'WorkflowId STARTS_WITH "bruno-"' --yes
```

ฐานข้อมูล `bop` (user `bop`/`bop`, ใช้ผ่าน container postgres ด้วย `-U temporal -d bop`): ใช้ `DO $$ ... $$` ที่ `RAISE EXCEPTION` ถ้าจำนวนที่ตรงเงื่อนไขไม่ตรงที่คาด หรืออย่างน้อยพิมพ์จำนวนที่ตรง/ที่เก็บไว้ ลบ `log_lines` ก่อน `jobs`

## Git

- commit แบบ Conventional Commits ภาษาอังกฤษ scope `bop` หรือ `dashboard` ท้ายข้อความใส่บรรทัด `Co-Authored-By` ตามที่ระบบกำหนดในรอบนั้น
- commit และ push เมื่อผู้ใช้สั่งหรืออยู่ในแผนที่อนุมัติแล้วเท่านั้น งานแตก branch จาก `main` ไม่ commit ลง `main` ตรง
- ไม่มี `gh`: ให้ลิงก์ `https://github.com/<owner>/<repo>/compare/main...<branch>?expand=1` และเนื้อหา PR แทน
- worktree ของ subagent ถูกล็อกโดย Claude Code: ปลดล็อกด้วย `git worktree unlock` แล้ว `git worktree remove` (ไม่ใช้ `--force`) และ `git branch -d`; ตรวจก่อนว่าไม่มีไฟล์ค้างและ branch ถูก merge แล้ว ไม่ใช้ `rm -rf` กับโฟลเดอร์ worktree

## ที่อยู่

dashboard `:3000` · API `:8080` · Temporal UI `:8233` (บน `192.168.1.46`, ผ่าน gateway ต้องมี session `bop_session`) · ผู้ใช้ dev `admin` / `changeme` (ค่าใน compose ใช้จริงไม่ได้)
