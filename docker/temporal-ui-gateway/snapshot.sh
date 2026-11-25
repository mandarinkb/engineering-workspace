#!/usr/bin/env bash
# สำรองข้อมูล etcd (route ทั้งหมดของ gateway โหมด etcd) ด้วย etcdctl snapshot (ADR 0003 กลไกข้อ 4)
# เก็บไว้ที่ ~/Workspace/apisix-snapshots/ (นอก repo) 5 ไฟล์ล่าสุด seed.sh เรียกสคริปต์นี้ก่อนเขียนทุกครั้ง
#
#   ./snapshot.sh
# ผ่านเมื่อพิมพ์ "snapshot: <ไฟล์> (<ขนาด> ไบต์)" และ exit 0 ถ้าล้ม seed ต้องไม่เขียนต่อ
set -uo pipefail

ETCD_CONTAINER="${ETCD_CONTAINER:-engineering-workspace-etcd-1}"
SNAPSHOT_DIR="${SNAPSHOT_DIR:-$HOME/Workspace/apisix-snapshots}"
KEEP="${KEEP:-5}"

mkdir -p "$SNAPSHOT_DIR" || { echo "สร้าง $SNAPSHOT_DIR ไม่ได้" >&2; exit 1; }
name="etcd-$(date +%Y%m%d-%H%M%S).db"
tmp="/tmp/$name"

if ! docker exec "$ETCD_CONTAINER" etcdctl --endpoints=http://127.0.0.1:2379 snapshot save "$tmp" >/dev/null 2>&1; then
  echo "สั่ง etcdctl snapshot ใน $ETCD_CONTAINER ไม่สำเร็จ" >&2
  exit 1
fi
if ! docker cp "$ETCD_CONTAINER:$tmp" "$SNAPSHOT_DIR/$name" >/dev/null 2>&1; then
  echo "คัดลอก snapshot ออกจาก container ไม่สำเร็จ" >&2
  docker exec "$ETCD_CONTAINER" rm -f "$tmp" >/dev/null 2>&1
  exit 1
fi
docker exec "$ETCD_CONTAINER" rm -f "$tmp" >/dev/null 2>&1
chmod 600 "$SNAPSHOT_DIR/$name"

size="$(wc -c < "$SNAPSHOT_DIR/$name" | tr -d ' ')"
if [ "${size:-0}" -lt 1024 ]; then
  echo "snapshot เล็กผิดปกติ (${size} ไบต์): $SNAPSHOT_DIR/$name" >&2
  exit 1
fi
echo "snapshot: $SNAPSHOT_DIR/$name (${size} ไบต์)"

# เก็บแค่ KEEP ไฟล์ล่าสุด ลบเฉพาะไฟล์ที่ชื่อตรงรูปแบบของสคริปต์นี้ในโฟลเดอร์นี้
ls -1 "$SNAPSHOT_DIR"/etcd-????????-??????.db 2>/dev/null | sort | head -n -"$KEEP" | while read -r old; do
  rm -f -- "$old" && echo "ลบ snapshot เก่า: $old"
done
exit 0
