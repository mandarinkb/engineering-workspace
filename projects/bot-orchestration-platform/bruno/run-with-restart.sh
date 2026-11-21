#!/usr/bin/env bash
# run-with-restart.sh — พิสูจน์ว่า Job และ log ของ Job อยู่รอดข้าม restart ของ bop-api (Bruno สั่ง
# restart เองไม่ได้):
#   1. รันชุดหลักทั้งหมด (สร้าง Job ที่ติด runMarker ของรอบนี้)
#   2. recreate container bop-api ด้วยคำสั่งใน RESTART_CMD แล้วรอ /healthz
#   3. รันเฉพาะโฟลเดอร์ after-restart ด้วย runMarker เดียวกัน (Job + log แบบ REST)
#   4. เปิด SSE stream ของ Job เดียวกันด้วย curl แล้วตรวจว่า backfill มี log จากรอบก่อน (Bruno CLI
#      อ่าน stream ที่เปิดค้างไม่ได้)
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
env_var() { sed -n "s/^ *$1: *//p" "environments/${ENV_NAME}.bru"; }
BASE_URL="$(env_var baseUrl)"
RUN_MARKER="bruno-$(date +%Y%m%d%H%M%S)"

echo "== 1/4 ชุดหลัก (runMarker=${RUN_MARKER})"
bru run --env "$ENV_NAME" --env-var "runMarker=${RUN_MARKER}"

echo "== 2/4 recreate bop-api"
bash -c "$RESTART_CMD"
for _ in $(seq 1 60); do
  if curl -fsS "${BASE_URL}/healthz" >/dev/null 2>&1; then
    break
  fi
  sleep 2
done
curl -fsS "${BASE_URL}/healthz" >/dev/null

echo "== 3/4 after-restart"
bru run after-restart --env "$ENV_NAME" --env-var "runMarker=${RUN_MARKER}"

echo "== 4/4 stream ของ log จากรอบก่อน"
JAR="$(mktemp)"
trap 'rm -f "$JAR"' EXIT
curl -fsS -c "$JAR" -H 'Content-Type: application/json' \
  -d "{\"username\": \"$(env_var username)\", \"password\": \"$(env_var password)\"}" \
  "${BASE_URL}/login" >/dev/null
JOB_ID="$(curl -fsS -b "$JAR" "${BASE_URL}/jobs" | RUN_MARKER="$RUN_MARKER" node -e '
  let s = "";
  process.stdin.on("data", (c) => (s += c)).on("end", () => {
    const marker = "bruno-run=" + process.env.RUN_MARKER;
    const j = JSON.parse(s).find((j) => j.config && String(j.config.url).endsWith(marker));
    process.stdout.write(j ? j.id : "");
  });')"
if [ -z "$JOB_ID" ]; then
  echo "FAIL: ไม่พบ Job ของ runMarker=${RUN_MARKER}" >&2
  exit 1
fi
# stream เปิดค้างตลอด: อ่าน backfill 3 วินาทีแล้วตัด (curl exit 28 ตอนหมดเวลาเป็นเรื่องปกติ)
STREAM="$(curl -s -N -b "$JAR" --max-time 3 "${BASE_URL}/jobs/${JOB_ID}/logs/stream" || true)"
curl -fsS -b "$JAR" -X POST "${BASE_URL}/logout" >/dev/null || true
if ! grep -q "\"job_id\":\"${JOB_ID}\"" <<<"$STREAM" || ! grep -q 'job succeeded' <<<"$STREAM"; then
  echo "FAIL: stream ของ Job ${JOB_ID} ไม่มี log จากรอบก่อน restart" >&2
  printf '%s\n' "$STREAM" >&2
  exit 1
fi
echo "stream ของ Job ${JOB_ID} มี log จากรอบก่อน restart ครบ"
