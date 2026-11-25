#!/usr/bin/env bash
# กู้ etcd ของ gateway โหมด etcd จากไฟล์ snapshot (ADR 0003 กลไกข้อ 4) — ทำลายข้อมูลปัจจุบันใน volume
#
#   ./restore.sh <snapshot.db> --yes
#
# ขั้นตอน: หยุด gateway และ etcd → ล้าง volume ของ etcd แล้วกู้จากไฟล์ด้วย container ชั่วคราว
# (image เดียวกับ etcd) → เริ่ม etcd และ gateway ใหม่ ต้องรันในโฟลเดอร์ checkout ที่มี docker-compose.yml
# ตัวแปร: COMPOSE (คำสั่ง compose รวม -f/--env-file/--profile), ETCD_VOLUME (ค่าเริ่มต้น engineering-workspace_etcd-data)
# ผ่านเมื่อพิมพ์ "กู้คืนเสร็จ" (exit 0) แล้วควรรัน seed.sh หรือ export.sh เพื่อตรวจ route
set -uo pipefail

SNAP="${1:-}"
CONFIRM="${2:-}"
ETCD_VOLUME="${ETCD_VOLUME:-engineering-workspace_etcd-data}"
ETCD_IMAGE="${ETCD_IMAGE:-quay.io/coreos/etcd:v3.5.21}"
COMPOSE="${COMPOSE:-/usr/libexec/docker/cli-plugins/docker-compose --env-file $HOME/Workspace/apisix-admin.env -f docker-compose.yml -f ../bop-lan.override.yml}"
GATEWAY_SERVICE="${GATEWAY_SERVICE:-temporal-ui-gateway}"

if [ -z "$SNAP" ] || [ ! -f "$SNAP" ]; then
  echo "ใช้: $0 <snapshot.db> --yes (ไม่พบไฟล์ '${SNAP}')" >&2
  exit 2
fi
if [ "$CONFIRM" != "--yes" ]; then
  echo "สคริปต์นี้ลบข้อมูล etcd ปัจจุบันใน volume ${ETCD_VOLUME} ถ้าแน่ใจให้ใส่ --yes" >&2
  exit 2
fi

abs="$(cd "$(dirname "$SNAP")" && pwd)/$(basename "$SNAP")"
echo "หยุด ${GATEWAY_SERVICE} และ etcd"
$COMPOSE stop "$GATEWAY_SERVICE" etcd >/dev/null || { echo "หยุด service ไม่สำเร็จ" >&2; exit 1; }

echo "กู้จาก ${abs} ลง volume ${ETCD_VOLUME}"
if ! docker run --rm -v "${ETCD_VOLUME}:/etcd-data" -v "${abs}:/restore/snapshot.db:ro" --entrypoint /bin/sh "$ETCD_IMAGE" \
  -c 'rm -rf /etcd-data/* /etcd-data/.[!.]* 2>/dev/null; etcdutl snapshot restore /restore/snapshot.db --data-dir /etcd-data/restored && cp -a /etcd-data/restored/member /etcd-data/member && rm -rf /etcd-data/restored'; then
  echo "กู้ไม่สำเร็จ (etcd และ gateway ยังหยุดอยู่ ข้อมูลใน volume อาจถูกล้างแล้ว: ลองกู้อีกครั้งจากไฟล์เดิม)" >&2
  exit 1
fi

$COMPOSE up -d etcd "$GATEWAY_SERVICE" >/dev/null || { echo "เริ่ม service ไม่สำเร็จ" >&2; exit 1; }
echo "กู้คืนเสร็จ: รัน seed.sh หรือ export.sh เพื่อตรวจ route"
