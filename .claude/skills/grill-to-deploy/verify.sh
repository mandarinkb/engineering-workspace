#!/bin/bash
# verify.sh — ตรวจโค้ด bop บน Mini PC แล้วสรุปสั้นๆ (ผลเต็มลงไฟล์ log)
#
# ใช้:   verify.sh <go|dashboard|all> [--log DIR]
# ตัวแปร: MINIPC (ค่าเริ่มต้น minipc) · MAX_WAIT_SEC (รอ container ว่าง ค่าเริ่มต้น 1800) · RUN_TIMEOUT (เวลาสูงสุดต่อ container ค่าเริ่มต้น 1200 วินาที)
# exit:  0 = ผ่านทุกขั้นที่รัน · 1 = ล้ม หรือหลักฐานไม่ครบ · 2 = ใช้ผิด / ต่อ Mini PC ไม่ได้
#
# หลักการ (traps.md): exit code ของคำสั่งครอบไม่ใช่หลักฐาน สคริปต์นี้ถือว่า "ผ่าน" ก็ต่อเมื่อเห็นบรรทัดยืนยัน
# ของทุกขั้นใน log และไม่เห็นรูปแบบความล้มเหลว
#
# ไม่ครอบคลุม: test ที่ต้องใช้ PostgreSQL จริง (ดู environment.md) · Bruno · Playwright
#
# ข้อควรระวัง: สคริปต์นี้สร้าง container ชั่วคราวบน Mini PC (docker run --rm) ตามกฎส่วนตัว ต้องมีการอนุมัติแผนนี้ก่อนรัน

set -u

MINIPC="${MINIPC:-minipc}"
MAX_WAIT_SEC="${MAX_WAIT_SEC:-1800}"
RUN_TIMEOUT="${RUN_TIMEOUT:-1200}"
GO_DIR="${GO_DIR:-projects/bot-orchestration-platform}"  # override ได้ (ใช้ทดสอบสคริปต์กับสำเนา)
DASH_DIR="projects/bop-dashboard"

# ---------- วิเคราะห์ log (แยกเป็นฟังก์ชันเพื่อทดสอบด้วย log ปลอมได้) ----------

# พิมพ์สรุปของโหมดหนึ่ง คืน 0 ถ้าผ่าน
analyze_go() {
  local log="$1" rc=0 oks fails
  grep -q '^VERIFY_TIMEOUT$' "$log" && { echo "  หมดเวลา (${RUN_TIMEOUT}s) — container ถูกหยุด ผลด้านล่างไม่ครบ"; rc=1; }
  oks=$(grep -cE '^ok[[:space:]]' "$log")
  fails=$(grep -cE '^(--- FAIL|FAIL[[:space:]]|panic:)' "$log")
  grep -q '^VET_OK$' "$log" && echo "  go vet        OK" || { echo "  go vet        FAIL (ไม่เห็น VET_OK)"; rc=1; }
  if [ "$fails" -gt 0 ]; then echo "  go test       FAIL ($fails บรรทัดล้ม)"; rc=1
  elif [ "$oks" -eq 0 ]; then echo "  go test       ไม่พิสูจน์ (ไม่เห็นบรรทัด ok สักแพ็กเกจ)"; rc=1
  else echo "  go test       OK ($oks แพ็กเกจ)"; fi
  grep -q '^GO_LINT_OK$' "$log" && echo "  lint + fmt    OK" || { echo "  lint + fmt    FAIL (ไม่เห็น GO_LINT_OK)"; rc=1; }
  return $rc
}

analyze_dashboard() {
  # คำสั่งฝั่ง container เป็นสาย && (npm ci → vitest → eslint → build) ขั้นหลังจากขั้นที่ล้มจึงไม่ได้ถูกรัน
  local log="$1" rc=0 tests stopped=0
  grep -q '^VERIFY_TIMEOUT$' "$log" && { echo "  หมดเวลา (${RUN_TIMEOUT}s) — container ถูกหยุด ผลด้านล่างไม่ครบ"; rc=1; }
  if [ "$stopped" -eq 0 ]; then
    if grep -q '^VITEST_OK$' "$log"; then
      tests=$(sed $'s/\033\\[[0-9;]*m//g' "$log" | grep -E '^[[:space:]]*Tests[[:space:]]' | tail -1 | sed 's/^[[:space:]]*//')
      echo "  vitest        OK${tests:+ ($tests)}"
    else echo "  vitest        FAIL (ไม่เห็น VITEST_OK — ล้มที่ npm ci / vitest หรือส่งซอร์สไม่สำเร็จ)"; rc=1; stopped=1; fi
  fi
  if [ "$stopped" -eq 1 ]; then echo "  eslint        ไม่ได้รัน (ขั้นก่อนหน้าล้ม)"
  elif grep -q '^ESLINT_OK$' "$log"; then echo "  eslint        OK"
  else echo "  eslint        FAIL (ไม่เห็น ESLINT_OK)"; rc=1; stopped=1; fi
  if [ "$stopped" -eq 1 ]; then echo "  next build    ไม่ได้รัน (ขั้นก่อนหน้าล้ม)"
  elif grep -qE 'Failed to compile|Build error occurred|Type error:' "$log"; then echo "  next build    FAIL (พบข้อความ build ล้ม)"; rc=1
  elif grep -q '^BUILD_OK$' "$log"; then echo "  next build    OK"
  else echo "  next build    FAIL (ไม่เห็น BUILD_OK)"; rc=1; fi
  return $rc
}

# ---------- โหมดทดสอบตัววิเคราะห์ ----------
if [ "${1:-}" = "--analyze" ]; then
  [ $# -eq 3 ] || { echo "ใช้: verify.sh --analyze <go|dashboard> <logfile>" >&2; exit 2; }
  case "$2" in
    go) analyze_go "$3" ;;
    dashboard) analyze_dashboard "$3" ;;
    *) echo "โหมดไม่ถูกต้อง: $2" >&2; exit 2 ;;
  esac
  exit $?
fi

# ---------- อ่านอาร์กิวเมนต์ ----------
MODE="${1:-}"
case "$MODE" in go|dashboard|all) ;; *) echo "ใช้: verify.sh <go|dashboard|all> [--log DIR]" >&2; exit 2 ;; esac
shift
LOG_DIR=""
while [ $# -gt 0 ]; do
  case "$1" in
    --log) LOG_DIR="${2:-}"; [ -n "$LOG_DIR" ] || { echo "--log ต้องมี DIR" >&2; exit 2; }; shift 2 ;;
    *) echo "อาร์กิวเมนต์ไม่รู้จัก: $1" >&2; exit 2 ;;
  esac
done
[ -n "$LOG_DIR" ] || LOG_DIR="$(mktemp -d "${TMPDIR:-/tmp}/verify-bop.XXXXXX")"
mkdir -p "$LOG_DIR" || exit 2

ROOT="$(git rev-parse --show-toplevel 2>/dev/null)" || { echo "ต้องรันใน repo" >&2; exit 2; }
cd "$ROOT" || exit 2

# SSH รับ stdin (ใช้กับ tar ผ่าน pipe) · SSHN ใส่ -n ใช้กับคำสั่งที่ไม่ต้องการ stdin กัน ssh กลืน input ของผู้เรียก
SSH=(ssh -o BatchMode=yes -o ConnectTimeout=10 "$MINIPC")
SSHN=(ssh -n -o BatchMode=yes -o ConnectTimeout=10 "$MINIPC")
"${SSHN[@]}" true 2>/dev/null || { echo "ต่อ $MINIPC ไม่ได้ (ssh BatchMode) — ไม่เดา ไม่หาทางอื่น ให้ถามผู้ใช้" >&2; exit 2; }

CNAME="bop-verify-$$"
cleanup() { "${SSHN[@]}" "docker rm -f $CNAME >/dev/null 2>&1" 2>/dev/null; }
trap 'cleanup; exit 130' INT TERM

# รอจน container ตรวจงานน้อยกว่า 3 ตัว (กัน load ของ Mini PC)
wait_for_slot() {
  local waited=0 n
  while :; do
    n=$("${SSHN[@]}" "docker ps -q --filter ancestor=golang:1.26 --filter ancestor=node:22-alpine --filter ancestor=golangci/golangci-lint:v2.15.0 | wc -l" 2>/dev/null | tr -d ' ')
    [ "${n:-9}" -lt 3 ] 2>/dev/null && return 0
    [ "$waited" -ge "$MAX_WAIT_SEC" ] && { echo "รอ container ว่างเกิน ${MAX_WAIT_SEC}s" >&2; return 1; }
    sleep 20; waited=$((waited + 20))
  done
}

run_go() {
  local log="$LOG_DIR/go.log"
  [ -d "$GO_DIR" ] || { echo "ไม่พบ $GO_DIR" >&2; return 2; }
  wait_for_slot || return 2
  (cd "$GO_DIR" && COPYFILE_DISABLE=1 tar --no-mac-metadata --exclude bruno --exclude docs -cf - .) \
    | "${SSH[@]}" "timeout -k 10 $RUN_TIMEOUT docker run --rm -i --name $CNAME golangci/golangci-lint:v2.15.0 sh -c 'mkdir /src && cd /src && tar xf - 2>/dev/null && go vet ./... && echo VET_OK; go test -p 1 ./... 2>&1 | grep -vE \"no test files|downloading\"; golangci-lint run ./... && golangci-lint fmt --diff ./... && echo GO_LINT_OK'" \
    >"$log" 2>&1
  [ "${PIPESTATUS[1]}" -eq 124 ] && { echo "VERIFY_TIMEOUT" >>"$log"; cleanup; }
  echo "Go ($GO_DIR)"; analyze_go "$log"
}

run_dashboard() {
  local log="$LOG_DIR/dashboard.log"
  [ -d "$DASH_DIR" ] || { echo "ไม่พบ $DASH_DIR" >&2; return 2; }
  wait_for_slot || return 2
  (cd projects && COPYFILE_DISABLE=1 tar --no-mac-metadata --exclude node_modules --exclude .next -cf - bop-dashboard) \
    | "${SSH[@]}" "timeout -k 10 $RUN_TIMEOUT docker run --rm -i --name $CNAME node:22-alpine sh -c 'mkdir /w && tar xf - -C /w 2>/dev/null && cd /w/bop-dashboard && npm ci --silent && npx vitest run && echo VITEST_OK && npx eslint . && echo ESLINT_OK && npx next build && echo BUILD_OK'" \
    >"$log" 2>&1
  [ "${PIPESTATUS[1]}" -eq 124 ] && { echo "VERIFY_TIMEOUT" >>"$log"; cleanup; }
  echo "Dashboard ($DASH_DIR)"; analyze_dashboard "$log"
}

overall=0
report() {  # $1=ชื่อ  $2=rc  $3=log
  if [ "$2" -ne 0 ]; then
    overall=1
    echo "  --- 30 บรรทัดท้ายของ $3 ---"
    tail -30 "$3" 2>/dev/null | sed 's/^/  | /'
  fi
}

case "$MODE" in
  go|all)        run_go;        rc=$?; report go "$rc" "$LOG_DIR/go.log" ;;
esac
case "$MODE" in
  dashboard|all) run_dashboard; rc=$?; report dashboard "$rc" "$LOG_DIR/dashboard.log" ;;
esac

echo
[ "$overall" -eq 0 ] && echo "ผล: ผ่านทุกขั้นที่รัน" || echo "ผล: ไม่ผ่าน"
echo "log เต็ม: $LOG_DIR"
exit "$overall"
