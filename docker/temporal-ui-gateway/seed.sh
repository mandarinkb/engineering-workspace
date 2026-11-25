#!/usr/bin/env bash
# เขียน route ใน routes/*.json ลง etcd ของ gateway โหมด etcd ผ่าน Admin API แล้วตรวจว่า gateway
# ยังไม่ปล่อยผ่านคนที่ไม่ login (ADR 0003 กลไกข้อ 1, 3 และ 4)
#
# ใช้บน Mini PC (Admin API เข้าได้จาก 127.0.0.1 เท่านั้น):
#   APISIX_ADMIN_KEY=... ./seed.sh                      # gateway ที่ :18233
#   APISIX_ADMIN_KEY=... GATEWAY_URL=http://127.0.0.1:8233 ./seed.sh
# ตัวแปรเสริม: ADMIN_URL, GATEWAY_URL, SKIP_SNAPSHOT=1 (ข้ามการสำรอง ใช้เฉพาะตอนทดสอบสคริปต์)
#
# ลำดับ: snapshot ก่อน (ล้ม = ไม่เขียน) → เขียน route → รายงาน drift (ไม่ลบ) → ตรวจ 302
# ผ่านก็ต่อเมื่อเห็นบรรทัด "ยืนยัน: 302" (exit 0) ไม่ใช่เพราะ curl สำเร็จ
# exit: 0 ผ่าน, 1 ตรวจแล้วไม่ผ่านหรือเขียนไม่สำเร็จ, 2 ตั้งค่าไม่ครบ
set -uo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
ADMIN_URL="${ADMIN_URL:-http://127.0.0.1:9180}"
GATEWAY_URL="${GATEWAY_URL:-http://127.0.0.1:18233}"
ROUTES_DIR="$DIR/routes"

if [ -z "${APISIX_ADMIN_KEY:-}" ]; then
  echo "ต้องตั้ง APISIX_ADMIN_KEY (ค่า key ของ Admin API)" >&2
  exit 2
fi
shopt -s nullglob
files=("$ROUTES_DIR"/*.json)
if [ ${#files[@]} -eq 0 ]; then
  echo "ไม่พบ routes/*.json" >&2
  exit 2
fi

# สำรอง etcd ก่อนเขียนทุกครั้ง: ถ้าสำรองไม่ได้ ไม่ seed (กู้คืนไม่ได้ = ไม่ควรแก้)
if [ "${SKIP_SNAPSHOT:-}" != "1" ]; then
  if ! "$DIR/snapshot.sh"; then
    echo "snapshot ไม่สำเร็จ จึงไม่ seed" >&2
    exit 1
  fi
fi

api() { # api METHOD PATH [ไฟล์ body] — พิมพ์ body ตามด้วยบรรทัดใหม่และ status
  local method="$1" path="$2" body="${3:-}"
  if [ -n "$body" ]; then
    curl -s -m 10 -w '\n%{http_code}' -X "$method" -H "X-API-KEY: ${APISIX_ADMIN_KEY}" \
      -H 'Content-Type: application/json' --data-binary "@$body" "${ADMIN_URL}${path}"
  else
    curl -s -m 10 -w '\n%{http_code}' -X "$method" -H "X-API-KEY: ${APISIX_ADMIN_KEY}" "${ADMIN_URL}${path}"
  fi
}

fail=0
for f in "${files[@]}"; do
  id="$(basename "$f" .json)"
  out="$(api PUT "/apisix/admin/routes/${id}" "$f")"
  code="${out##*$'\n'}"
  if [ "$code" = "200" ] || [ "$code" = "201" ]; then
    echo "เขียน route ${id}: HTTP ${code}"
  else
    echo "เขียน route ${id} ไม่สำเร็จ: HTTP ${code}" >&2
    echo "${out%$'\n'*}" | head -c 400 >&2
    echo >&2
    fail=1
  fi
done

# drift: route ใน etcd ที่ไม่ตรงกับ git (เช่นสร้าง/แก้ผ่าน dashboard) — แค่รายงาน ไม่ลบ ไม่นับเป็นล้ม
resp="$(mktemp)"
trap 'rm -f "$resp"' EXIT
code="$(curl -s -m 10 -o "$resp" -w '%{http_code}' -H "X-API-KEY: ${APISIX_ADMIN_KEY}" "${ADMIN_URL}/apisix/admin/routes")"
if [ "$code" = "200" ]; then
  python3 "$DIR/drift.py" --etcd "$resp" --routes "$ROUTES_DIR"
  [ "$?" = "1" ] && { echo "เปรียบเทียบ drift ไม่ได้ (รูปแบบผลลัพธ์ของ Admin API ไม่ตรงที่คาด)" >&2; fail=1; }
else
  echo "อ่านรายการ route ไม่สำเร็จ: HTTP ${code}" >&2
  fail=1
fi

# ตรวจหลักฐานว่าไม่ปล่อยผ่าน: ไม่มี cookie ต้องได้ 302 ไป /login (รอให้ gateway โหลด route ได้ไม่กี่วินาที)
status="" loc=""
for _ in 1 2 3 4 5 6 7 8 9 10; do
  hdr="$(curl -s -m 5 -o /dev/null -D - "${GATEWAY_URL}/" || true)"
  status="$(printf '%s' "$hdr" | head -1 | awk '{print $2}')"
  loc="$(printf '%s' "$hdr" | grep -i '^location:' | tr -d '\r' | awk '{print $2}')"
  [ "$status" = "302" ] && break
  sleep 1
done
if [ "$status" = "302" ] && [[ "$loc" == */login ]]; then
  echo "ยืนยัน: 302 → ${loc}"
else
  echo "ไม่ผ่าน: ไม่มี cookie แต่ได้ status='${status}' location='${loc}' (ต้องเป็น 302 ไป /login)" >&2
  fail=1
fi

[ "$fail" = "0" ] && echo "seed ผ่าน" || echo "seed ไม่ผ่าน" >&2
exit "$fail"
