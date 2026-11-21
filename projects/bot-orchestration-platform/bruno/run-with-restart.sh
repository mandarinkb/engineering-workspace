#!/usr/bin/env bash
# run-with-restart.sh — พิสูจน์ว่า Job อยู่รอดข้าม restart ของ bop-api (Bruno สั่ง restart เองไม่ได้):
#   1. รันชุดหลักทั้งหมด (สร้าง Job ที่ติด runMarker ของรอบนี้)
#   2. recreate container bop-api ด้วยคำสั่งใน RESTART_CMD แล้วรอ /healthz
#   3. รันเฉพาะโฟลเดอร์ after-restart ด้วย runMarker เดียวกัน
#
# ใช้งาน (ในโฟลเดอร์ bruno/):
#   RESTART_CMD='<คำสั่ง recreate bop-api>' ./run-with-restart.sh [env]   # env ค่าเริ่มต้น minipc
#
# ตัวอย่าง RESTART_CMD:
#   stack บนเครื่องตัวเอง (รันจาก root ของ repo):
#     RESTART_CMD='docker compose -f ../../../docker-compose.yml up -d --force-recreate --no-deps bop-api'
#   stack บน Mini PC (compose ของระบบ ไม่ใช่ตัวเก่าที่ shadow อยู่):
#     RESTART_CMD="ssh minipc 'cd <โฟลเดอร์ที่ deploy> && /usr/libexec/docker/cli-plugins/docker-compose up -d --force-recreate --no-deps bop-api'"
set -euo pipefail

ENV_NAME="${1:-minipc}"
: "${RESTART_CMD:?ตั้ง RESTART_CMD เป็นคำสั่ง recreate container bop-api ก่อน (ดูหัวไฟล์)}"

cd "$(dirname "$0")"
BASE_URL="$(sed -n 's/^ *baseUrl: *//p' "environments/${ENV_NAME}.bru")"
RUN_MARKER="bruno-$(date +%Y%m%d%H%M%S)"

echo "== 1/3 ชุดหลัก (runMarker=${RUN_MARKER})"
bru run --env "$ENV_NAME" --env-var "runMarker=${RUN_MARKER}"

echo "== 2/3 recreate bop-api"
bash -c "$RESTART_CMD"
for _ in $(seq 1 60); do
  if curl -fsS "${BASE_URL}/healthz" >/dev/null 2>&1; then
    break
  fi
  sleep 2
done
curl -fsS "${BASE_URL}/healthz" >/dev/null

echo "== 3/3 after-restart"
bru run after-restart --env "$ENV_NAME" --env-var "runMarker=${RUN_MARKER}"
