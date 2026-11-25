#!/usr/bin/env bash
# cleanup-test-data.sh — ลบข้อมูลทดสอบที่ชุด Bruno ทิ้งไว้บน Mini PC (Job + log ในฐานข้อมูล bop และ Workflow
# ใน Temporal) ด้วยตัวกันพลาด ใช้ตามหัวข้อ "ข้อมูลทดสอบและการลบ" ของ environment.md
#
#   ./cleanup-test-data.sh                 # dry-run: นับและแสดงรายการ ไม่ลบอะไร (ค่าเริ่มต้น)
#   ./cleanup-test-data.sh --yes           # ลบจริง
#   ./cleanup-test-data.sh --yes --prefix uiwalk-    # เพิ่ม prefix ได้ (ใช้ซ้ำได้) ค่าเริ่มต้นคือ bruno-
#
# ตัวกันพลาด:
#   - prefix ต้องเป็น a-z A-Z 0-9 - ยาว 5 ตัวขึ้นไปและลงท้ายด้วย - (กัน pattern กว้างและ SQL injection)
#   - Job ที่ลบคือแถวที่ config มี prefix เท่านั้น; Workflow คือ WorkflowId ขึ้นต้นด้วย prefix เท่านั้น
#   - ลบ Job ใน transaction เดียว: ถ้าจำนวนที่ลบจริงไม่ตรงที่นับไว้ก่อนหน้า (มีของใหม่แทรก) ROLLBACK ไม่ลบอะไร
#   - ถ้ามี Workflow ที่ prefix ตรงและยัง Running อยู่ จะไม่ลบอะไรเลย (อาจกำลังมีชุดทดสอบรันอยู่)
#   - Schedule ที่ ID ขึ้นต้นด้วย prefix จะรายงานเท่านั้น ไม่ลบ (Bruno ลบเองตอนจบ)
#   - ผ่านก็ต่อเมื่อหลังลบนับซ้ำได้ 0 ทุกอย่าง (พิมพ์ "ล้างเสร็จ") ไม่ใช่เพราะคำสั่งลบสำเร็จ
#
# ข้อจำกัด: Job ที่ config ไม่มี prefix (เช่นที่ชุด Bruno สร้างก่อนเพิ่ม note bruno- ใน 03-jobs/01-create-job-invalid) ลบด้วยสคริปต์นี้ไม่ได้
#
# ตัวแปร: MINIPC (ค่าเริ่มต้น minipc), PG_CONTAINER, TEMPORAL_CONTAINER
# exit: 0 สำเร็จ (รวม dry-run และไม่มีอะไรต้องลบ), 1 กันพลาดทำงาน/ลบไม่หมด/ต่อเครื่องไม่ได้, 2 ใช้คำสั่งผิด
# CLEANUP_TEST_EXPECT_JOBS ใช้ทดสอบตัวกันพลาดเท่านั้น (บังคับจำนวน Job ที่คาด)
set -uo pipefail

MINIPC="${MINIPC:-minipc}"
PG_CONTAINER="${PG_CONTAINER:-engineering-workspace-postgresql-1}"
TEMPORAL_CONTAINER="${TEMPORAL_CONTAINER:-engineering-workspace-temporal-1}"
MODE="dry"
PREFIXES=()

usage() { sed -n '2,22p' "$0" | sed 's/^# \{0,1\}//'; exit 2; }

while [ $# -gt 0 ]; do
  case "$1" in
    --yes) MODE="delete" ;;
    --dry-run) MODE="dry" ;;
    --prefix) shift; [ $# -gt 0 ] || usage; PREFIXES+=("$1") ;;
    -h|--help) usage ;;
    *) echo "ไม่รู้จักตัวเลือก: $1" >&2; usage ;;
  esac
  shift
done
[ ${#PREFIXES[@]} -gt 0 ] || PREFIXES=("bruno-")
for p in "${PREFIXES[@]}"; do
  if [[ ! "$p" =~ ^[A-Za-z0-9-]{4,}-$ ]]; then
    echo "prefix '$p' ไม่ผ่าน: ต้องเป็น a-z A-Z 0-9 - ยาว 5 ตัวขึ้นไปและลงท้ายด้วย -" >&2
    exit 2
  fi
done

SSH=(ssh -o BatchMode=yes "$MINIPC")

# เงื่อนไข SQL และ query ของ Temporal จาก prefix (ผ่านการตรวจรูปแบบแล้ว จึงฝังตรงๆ ได้)
COND=""; TQ=""
for p in "${PREFIXES[@]}"; do
  COND+="${COND:+ OR }config::text LIKE '%${p}%'"
  TQ+="${TQ:+ OR }WorkflowId STARTS_WITH \"${p}\""
done
[ ${#PREFIXES[@]} -gt 1 ] && TQ="(${TQ})"

# ssh รวมอาร์กิวเมนต์เป็นสตริงเดียวให้ shell ปลายทางตีความใหม่ จึงต้อง quote ทุกอาร์กิวเมนต์ (query มีช่องว่างและเครื่องหมายคำพูด)
rq() { local o="" a; for a in "$@"; do o+=" $(printf '%q' "$a")"; done; printf '%s' "$o"; }
psql() { "${SSH[@]}" "docker exec -i ${PG_CONTAINER} psql -U temporal -d bop -X -q -t -A -F '|' -v ON_ERROR_STOP=1$(rq "$@")"; }
temporal() { "${SSH[@]}" "docker exec ${TEMPORAL_CONTAINER} temporal$(rq "$@") --address temporal:7233 --namespace default"; }
wf_count() { # wf_count [เงื่อนไขเพิ่ม]
  local q="$TQ"; [ -n "${1:-}" ] && q="$TQ AND $1"
  temporal workflow count --query "$q" 2>/dev/null | awk '/Total:/{print $2}'
}
db_counts() { # พิมพ์ "jobs|log_lines"
  psql -c "SELECT (SELECT count(*) FROM jobs WHERE ${COND}), (SELECT count(*) FROM log_lines WHERE job_id IN (SELECT id FROM jobs WHERE ${COND}));"
}

echo "prefix: ${PREFIXES[*]} | โหมด: $([ "$MODE" = delete ] && echo 'ลบจริง (--yes)' || echo 'dry-run (ไม่ลบ)')"

counts="$(db_counts 2>&1)" || { echo "ต่อฐานข้อมูลบน ${MINIPC} ไม่ได้: ${counts}" >&2; exit 1; }
J="${counts%%|*}"; L="${counts##*|}"
W="$(wf_count)"; R="$(wf_count 'ExecutionStatus="Running"')"
if ! [[ "$J" =~ ^[0-9]+$ && "$L" =~ ^[0-9]+$ && "$W" =~ ^[0-9]+$ && "${R:-}" =~ ^[0-9]+$ ]]; then
  echo "อ่านจำนวนไม่ได้ (jobs='$J' log_lines='$L' workflows='$W' running='${R:-}') ไม่ลบอะไร" >&2
  exit 1
fi
SCHED="$(temporal schedule list 2>/dev/null | awk 'NR>1{print $1}' | grep -E "^($(IFS='|'; echo "${PREFIXES[*]}"))" || true)"

echo "พบ: jobs=${J} log_lines=${L} workflows=${W} (Running=${R})"
if [ "$J" -gt 0 ]; then
  echo "Job ที่ตรง (id8 | bot | status | สร้างเมื่อ):"
  psql -c "SELECT left(id,8), bot_name, status, to_char(created_at AT TIME ZONE 'Asia/Bangkok','MM-DD HH24:MI') FROM jobs WHERE ${COND} ORDER BY created_at LIMIT 30;" | sed 's/^/  /'
fi
[ -n "$SCHED" ] && echo "คำเตือน: มี Schedule ที่ ID ขึ้นต้นด้วย prefix (ไม่ลบ ให้ชุดทดสอบลบเอง): $(echo "$SCHED" | tr '\n' ' ')"

if [ "$R" -gt 0 ]; then
  echo "หยุด: มี Workflow ที่ prefix ตรงและยัง Running อยู่ ${R} ตัว (อาจมีชุดทดสอบรันอยู่) ไม่ลบอะไร" >&2
  exit 1
fi
if [ "$J" -eq 0 ] && [ "$L" -eq 0 ] && [ "$W" -eq 0 ]; then
  echo "ไม่มีอะไรต้องลบ"
  exit 0
fi
if [ "$MODE" = dry ]; then
  echo "dry-run: ยังไม่ได้ลบอะไร ใส่ --yes เพื่อลบ"
  exit 0
fi

fail=0
if [ "$J" -gt 0 ] || [ "$L" -gt 0 ]; then
  EXP_J="${CLEANUP_TEST_EXPECT_JOBS:-$J}"
  out="$(psql -f - <<SQL
BEGIN;
WITH d AS (DELETE FROM log_lines WHERE job_id IN (SELECT id FROM jobs WHERE ${COND}) RETURNING 1),
     j AS (DELETE FROM jobs WHERE ${COND} RETURNING 1)
SELECT (SELECT count(*) FROM d) AS dl, (SELECT count(*) FROM j) AS dj \gset
SELECT (:dj = ${EXP_J} AND :dl = ${L}) AS ok \gset
\if :ok
COMMIT;
\echo DELETED|:dj|:dl
\else
ROLLBACK;
\echo ROLLBACK|:dj|:dl
\endif
SQL
  )" || { echo "ลบ Job ล้มเหลว: ${out}" >&2; exit 1; }
  case "$out" in
    DELETED*) echo "ลบ Job แล้ว: jobs/log_lines = ${out#DELETED|}" ;;
    *) echo "ตัวกันพลาดทำงาน: จำนวนที่จะลบ (${out#ROLLBACK|}) ไม่ตรงที่นับไว้ (jobs=${EXP_J} log_lines=${L}) ROLLBACK แล้ว ไม่ลบ Job" >&2; fail=1 ;;
  esac
fi

if [ "$fail" -eq 0 ] && [ "$W" -gt 0 ]; then
  temporal workflow delete --query "$TQ" --yes >/dev/null 2>&1 || echo "คำสั่งลบ Workflow ตอบ error (ตรวจจากการนับซ้ำด้านล่าง)" >&2
  for _ in $(seq 1 30); do [ "$(wf_count)" = "0" ] && break; sleep 2; done
fi

# หลักฐาน: นับซ้ำ (ถ้าไม่ได้ลบ Workflow เพราะตัวกันพลาดของ Job ทำงาน ก็นับให้เห็นสภาพจริง)
counts2="$(db_counts 2>&1)" || { echo "นับซ้ำไม่ได้: ${counts2}" >&2; exit 1; }
J2="${counts2%%|*}"; L2="${counts2##*|}"; W2="$(wf_count)"
echo "หลังลบ: jobs=${J2} log_lines=${L2} workflows=${W2}"
if [ "$fail" -eq 0 ] && [ "$J2" = "0" ] && [ "$L2" = "0" ] && [ "$W2" = "0" ]; then
  echo "ล้างเสร็จ"
  exit 0
fi
echo "ล้างไม่หมด (ดูข้อความด้านบน)" >&2
exit 1
