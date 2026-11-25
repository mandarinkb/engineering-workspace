#!/usr/bin/env bash
# ดึง route ปัจจุบันจาก etcd ผ่าน Admin API แล้วเทียบกับ routes/*.json ใน git (ADR 0003 กลไกข้อ 1)
# ไม่แก้ไฟล์ใน repo เอง: เขียนผลที่จัดรูปแบบคงที่ลงโฟลเดอร์ชั่วคราวให้ผู้ใช้เปิดดู/คัดลอกกลับไป commit เอง
#
#   APISIX_ADMIN_KEY=... ./export.sh [โฟลเดอร์ปลายทาง]
#
# exit: 0 ไม่มี drift, 3 พบ drift (ไม่ใช่ error), 1 เรียก Admin API ไม่ได้, 2 ตั้งค่าไม่ครบ
set -uo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
ADMIN_URL="${ADMIN_URL:-http://127.0.0.1:9180}"
OUT="${1:-$(mktemp -d "${TMPDIR:-/tmp}/apisix-export.XXXXXX")}"

if [ -z "${APISIX_ADMIN_KEY:-}" ]; then
  echo "ต้องตั้ง APISIX_ADMIN_KEY (ค่า key ของ Admin API)" >&2
  exit 2
fi

resp="$(mktemp)"
trap 'rm -f "$resp"' EXIT
code="$(curl -s -m 10 -o "$resp" -w '%{http_code}' -H "X-API-KEY: ${APISIX_ADMIN_KEY}" "${ADMIN_URL}/apisix/admin/routes")"
if [ "$code" != "200" ]; then
  echo "เรียก Admin API ไม่สำเร็จ: HTTP ${code}" >&2
  exit 1
fi

python3 "$DIR/drift.py" --etcd "$resp" --routes "$DIR/routes" --out "$OUT"
rc=$?
echo "route จาก etcd (จัดรูปแบบคงที่) อยู่ที่: ${OUT}"
exit "$rc"
