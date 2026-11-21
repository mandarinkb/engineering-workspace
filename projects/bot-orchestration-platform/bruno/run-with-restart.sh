#!/usr/bin/env bash
# run-with-restart.sh — พิสูจน์ว่า Job และ log ของ Job อยู่รอดข้าม restart ของ bop-api และ Job ที่ถูก
# ขัดจังหวะโดย restart ไม่ค้างเป็น running (Bruno สั่ง restart เองไม่ได้):
#   1. รันชุดหลักทั้งหมด (สร้าง Job ที่ติด runMarker ของรอบนี้)
#   2. สร้าง Job ที่ค้างนาน (http-status-checker ชี้ไป HANG_URL ที่ไม่ตอบ pool ตัดจบเองที่ 30 วินาที)
#      แล้ว poll จนเห็นว่าเป็น running จริงก่อน restart — ไม่เคยเป็น running = พิมพ์ SKIPPED แล้ว
#      exit 2 (ห้ามผ่านเงียบ)
#   3. recreate container bop-api ด้วยคำสั่งใน RESTART_CMD แล้วรอ /healthz
#   4. รันเฉพาะโฟลเดอร์ after-restart ด้วย runMarker เดียวกัน (Job + log แบบ REST)
#   5. เปิด SSE stream ของ Job เดียวกันด้วย curl แล้วตรวจว่า backfill มี log จากรอบก่อน (Bruno CLI
#      อ่าน stream ที่เปิดค้างไม่ได้)
#   6. ตรวจว่า Job ค้างนานจากข้อ 2 เป็น failed พร้อม error "interrupted" (ticket 14: bop-api
#      กวาด Job ที่ค้างตอน start)
#
# ใช้งาน (ในโฟลเดอร์ bruno/):
#   RESTART_CMD='<คำสั่ง recreate bop-api>' ./run-with-restart.sh [env]   # env ค่าเริ่มต้น minipc
#
# exit code: 0 ผ่านทั้งหมด, 1 ไม่ผ่าน, 2 SKIPPED (สร้าง Job ที่ running ก่อน restart ไม่ได้ จึงไม่ได้
# พิสูจน์ข้อ 6), 3 NOT PROVEN (Job เป็น cancelled แทน failed ดูด้านล่าง)
#
# HANG_URL (ค่าเริ่มต้น http://10.255.255.1:81/ — address ใน private range ที่ไม่มีเครื่องจริง packet
# ถูกทิ้ง connect เลยค้าง) ตั้งเองได้ถ้า network ของ container ตอบ "no route" ทันทีจน Job จบก่อน
# ต้องเป็น address ที่ container bop-api เห็นแล้วไม่ตอบ
#
# ข้อควรรู้: ถ้า grace period ของ container ยาวกว่า 30 วินาที bop-api ตัวเก่าจะปิดแบบปกติและยกเลิก
# Job เป็น cancelled เอง การกวาดตอน start ไม่ได้ถูกใช้ ข้อ 6 จะแจ้งชัดว่าไม่ได้พิสูจน์อะไร
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

JAR="$(mktemp)"
trap 'rm -f "$JAR"' EXIT
login() {
  curl -fsS -c "$JAR" -H 'Content-Type: application/json' \
    -d "{\"username\": \"$(env_var username)\", \"password\": \"$(env_var password)\"}" \
    "${BASE_URL}/login" >/dev/null
}
# อ่าน field ระดับบนของ JSON ที่ stdin: json_field <ชื่อ field>
json_field() {
  node -e 'let s = "";
    process.stdin.on("data", (c) => (s += c)).on("end", () => {
      try {
        const v = JSON.parse(s)[process.argv[1]];
        process.stdout.write(v == null ? "" : String(v));
      } catch {}
    });' "$1"
}

echo "== 1/6 ชุดหลัก (runMarker=${RUN_MARKER})"
bru run --env "$ENV_NAME" --env-var "runMarker=${RUN_MARKER}"

HANG_URL="${HANG_URL:-http://10.255.255.1:81/}"
echo "== 2/6 สร้าง Job ที่ค้างนานแล้วรอให้เป็น running (HANG_URL=${HANG_URL})"
login
HANG_JOB_ID="$(curl -fsS -b "$JAR" -H 'Content-Type: application/json' \
  -d "{\"url\": \"${HANG_URL}?bruno-hang=${RUN_MARKER}\"}" \
  "${BASE_URL}/bots/http-status-checker/jobs" | json_field id)"
if [ -z "$HANG_JOB_ID" ]; then
  echo "SKIPPED: สร้าง Job ค้างนานไม่สำเร็จ จึงไม่ได้พิสูจน์การกวาด Job ที่ถูกขัดจังหวะ" >&2
  exit 2
fi
HANG_STATUS=""
for _ in $(seq 1 20); do
  HANG_STATUS="$(curl -fsS -b "$JAR" "${BASE_URL}/jobs/${HANG_JOB_ID}" | json_field status)"
  [ "$HANG_STATUS" = "running" ] && break
  sleep 0.5
done
if [ "$HANG_STATUS" != "running" ]; then
  echo "SKIPPED: Job ${HANG_JOB_ID} ไม่เคยเป็น running (สถานะล่าสุด=${HANG_STATUS:-?}) — HANG_URL ตอบเร็วเกินไปหรือ pool เต็ม จึงไม่ได้พิสูจน์การกวาด Job ที่ถูกขัดจังหวะ ตั้ง HANG_URL ใหม่แล้วรันซ้ำ" >&2
  exit 2
fi
echo "Job ${HANG_JOB_ID} กำลัง running"

echo "== 3/6 recreate bop-api"
bash -c "$RESTART_CMD"
for _ in $(seq 1 60); do
  if curl -fsS "${BASE_URL}/healthz" >/dev/null 2>&1; then
    break
  fi
  sleep 2
done
curl -fsS "${BASE_URL}/healthz" >/dev/null

echo "== 4/6 after-restart"
bru run after-restart --env "$ENV_NAME" --env-var "runMarker=${RUN_MARKER}"

echo "== 5/6 stream ของ log จากรอบก่อน"
login
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
if ! grep -q "\"job_id\":\"${JOB_ID}\"" <<<"$STREAM" || ! grep -q 'job succeeded' <<<"$STREAM"; then
  curl -fsS -b "$JAR" -X POST "${BASE_URL}/logout" >/dev/null || true
  echo "FAIL: stream ของ Job ${JOB_ID} ไม่มี log จากรอบก่อน restart" >&2
  printf '%s\n' "$STREAM" >&2
  exit 1
fi
echo "stream ของ Job ${JOB_ID} มี log จากรอบก่อน restart ครบ"

echo "== 6/6 Job ค้างนาน ${HANG_JOB_ID} ต้องเป็น failed (interrupted)"
HANG_JSON="$(curl -fsS -b "$JAR" "${BASE_URL}/jobs/${HANG_JOB_ID}")"
curl -fsS -b "$JAR" -X POST "${BASE_URL}/logout" >/dev/null || true
FINAL_STATUS="$(json_field status <<<"$HANG_JSON")"
FINAL_ERROR="$(json_field error <<<"$HANG_JSON")"
FINAL_ENDED="$(json_field ended_at <<<"$HANG_JSON")"
if [ "$FINAL_STATUS" = "cancelled" ]; then
  echo "NOT PROVEN: Job เป็น cancelled — bop-api ตัวเก่าปิดแบบปกติเองก่อนถูก kill การกวาดตอน start ไม่ได้ถูกใช้ (ลด grace period ของ container ให้น้อยกว่า 30 วินาทีแล้วรันซ้ำ)" >&2
  exit 3
fi
if [ "$FINAL_STATUS" != "failed" ] || [[ "$FINAL_ERROR" != *interrupted* ]] || [ -z "$FINAL_ENDED" ]; then
  echo "FAIL: Job ${HANG_JOB_ID} หลัง restart: status=${FINAL_STATUS} error=${FINAL_ERROR} ended_at=${FINAL_ENDED} (ต้องเป็น failed + interrupted + มี ended_at)" >&2
  exit 1
fi
echo "Job ${HANG_JOB_ID} เป็น failed: ${FINAL_ERROR}"
