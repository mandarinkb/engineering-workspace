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
HELPER_IMAGE="${HELPER_IMAGE:-apache/apisix:3.21.0-debian}" # ต้องมี bash และมีอยู่บนเครื่องแล้ว (ไม่ pull ใหม่)
COMPOSE="${COMPOSE:-/usr/libexec/docker/cli-plugins/docker-compose --env-file $HOME/Workspace/apisix-admin.env -f docker-compose.yml -f ../bop-lan.override.yml --profile apisix-etcd}"
GATEWAY_SERVICE="${GATEWAY_SERVICE:-temporal-ui-gateway-next}"

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
# image ของ etcd ไม่มี shell (ตรวจแล้ว: /bin/sh ไม่มี) จึงแยกเป็นสองขั้น:
#  1) etcdutl ใน image ของ etcd กู้ snapshot ลงโฟลเดอร์ใหม่ restored/ ใน volume
#  2) container ที่มี bash (image ของ APISIX ที่มีอยู่แล้วบนเครื่อง) สลับ member/ เดิมเป็นของที่กู้
if ! docker run --rm -v "${ETCD_VOLUME}:/etcd-data" --entrypoint etcdutl "$ETCD_IMAGE" version >/dev/null 2>&1; then
  echo "เรียก etcdutl จาก ${ETCD_IMAGE} ไม่ได้ ยังไม่แตะข้อมูลใน volume" >&2
  exit 1
fi
if ! docker run --rm --user root -v "${ETCD_VOLUME}:/etcd-data" --entrypoint /bin/bash "$HELPER_IMAGE" \
  -c 'rm -rf /etcd-data/restored'; then
  echo "เตรียมโฟลเดอร์ชั่วคราวใน volume ไม่สำเร็จ ยังไม่แตะข้อมูลเดิม" >&2
  exit 1
fi
if ! docker run --rm -v "${ETCD_VOLUME}:/etcd-data" -v "${abs}:/restore/snapshot.db:ro" --entrypoint etcdutl "$ETCD_IMAGE" \
  snapshot restore /restore/snapshot.db --data-dir /etcd-data/restored; then
  echo "กู้ snapshot ไม่สำเร็จ ข้อมูลเดิมใน volume ยังอยู่ครบ (member/) etcd และ gateway ยังหยุดอยู่" >&2
  exit 1
fi
if ! docker run --rm --user root -v "${ETCD_VOLUME}:/etcd-data" --entrypoint /bin/bash "$HELPER_IMAGE" \
  -c 'set -e; rm -rf /etcd-data/member; mv /etcd-data/restored/member /etcd-data/member; rm -rf /etcd-data/restored'; then
  echo "สลับข้อมูลที่กู้แล้วเข้าที่ไม่สำเร็จ (ข้อมูลเดิมอาจหายแล้ว: ลองรันซ้ำจากไฟล์เดิม)" >&2
  exit 1
fi

$COMPOSE up -d etcd "$GATEWAY_SERVICE" >/dev/null || { echo "เริ่ม service ไม่สำเร็จ" >&2; exit 1; }
echo "กู้คืนเสร็จ: รัน seed.sh หรือ export.sh เพื่อตรวจ route"
