#!/usr/bin/env bash
# สร้าง apisix.yaml (ชุด dev, โหมดไฟล์) จาก routes/*.json
#   ./gen-yaml.sh          เขียน apisix.yaml
#   ./gen-yaml.sh --check  ตรวจว่า apisix.yaml ที่ commit ตรงกับที่สร้างใหม่ (ต่างกันล้ม)
exec python3 "$(cd "$(dirname "$0")" && pwd)/gen_yaml.py" "$@"
