#!/usr/bin/env bash
# ทดสอบ gen_yaml.py: apisix.yaml ที่ commit ต้องตรงกับ routes/*.json และการแก้ JSON ต้องถูกจับ
#   ./tests/test-gen-yaml.sh
# ผ่านเมื่อพิมพ์ "ผ่านทุกเคส" (exit 0)
set -uo pipefail
DIR="$(cd "$(dirname "$0")/.." && pwd)"
T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT
fail=0
ok() { echo "✓ $1"; }
ng() { echo "✗ $1"; fail=1; }

if python3 "$DIR/gen_yaml.py" --check >/dev/null; then ok "apisix.yaml ที่ commit ตรงกับ routes/*.json"; else ng "apisix.yaml ที่ commit ไม่ตรงกับ routes/*.json"; fi

# แก้ Lua ใน JSON 1 อักขระ (สำเนาใน $T) → --check ต้องล้ม
cp -R "$DIR/routes" "$T/routes"
python3 - "$T/routes/temporal-ui.json" <<'EOF'
import json, sys
p = sys.argv[1]
r = json.load(open(p, encoding="utf-8"))
f = r["plugins"]["serverless-pre-function"]["functions"]
f[0] = f[0].replace("2000", "2001", 1)
json.dump(r, open(p, "w", encoding="utf-8"), ensure_ascii=False, indent=2)
EOF
out="$(python3 "$DIR/gen_yaml.py" --routes "$T/routes" --file "$DIR/apisix.yaml" --check 2>&1)"; rc=$?
if [ "$rc" = "1" ] && printf '%s' "$out" | grep -q "2001"; then ok "แก้ Lua 1 อักขระ → --check ล้ม (exit 1) และแสดงผลต่าง"; else ng "แก้ Lua แล้ว --check ไม่ล้มตามคาด (exit=$rc)"; fi

# คืนค่า (สร้างใหม่จากสำเนาที่แก้แล้วลงไฟล์ชั่วคราว) แล้ว --check กับไฟล์นั้นต้องผ่าน = ตัวสร้างกับตัวตรวจสอดคล้องกัน
python3 "$DIR/gen_yaml.py" --routes "$T/routes" --file "$T/out.yaml" >/dev/null
if python3 "$DIR/gen_yaml.py" --routes "$T/routes" --file "$T/out.yaml" --check >/dev/null; then ok "สร้างแล้วตรวจผ่าน"; else ng "สร้างแล้วตรวจไม่ผ่าน"; fi

# YAML ที่ได้ต้องมีความหมายเดียวกับ JSON (Lua ตรงทุกตัวอักษร) — ต้องมี ruby
if command -v ruby >/dev/null; then
  if ruby -ryaml -rjson -e '
      d = ARGV[0]
      y = YAML.load_file(File.join(d, "apisix.yaml"))["routes"]
      j = Dir[File.join(d, "routes", "*.json")].sort.map { |f| JSON.parse(File.read(f)).merge("id" => File.basename(f, ".json")) }
      exit(y.sort_by { |r| r["id"] } == j.sort_by { |r| r["id"] } ? 0 : 1)' "$DIR"; then
    ok "YAML ที่ commit มีความหมายเดียวกับ JSON (ตรวจด้วย ruby)"
  else ng "YAML กับ JSON ความหมายไม่ตรงกัน"; fi
else
  echo "ข้าม: ไม่มี ruby ไม่ได้ตรวจความหมาย (ยังไม่พิสูจน์)"
fi

[ "$fail" = "0" ] && echo "ผ่านทุกเคส" || { echo "มีเคสไม่ผ่าน" >&2; exit 1; }
