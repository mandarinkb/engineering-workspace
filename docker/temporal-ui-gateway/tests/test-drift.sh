#!/usr/bin/env bash
# ทดสอบ drift.py แบบ black-box ด้วยไฟล์จำลองผลของ Admin API (ไม่ต้องใช้ Docker)
#   ./tests/test-drift.sh
# ผ่านเมื่อพิมพ์ "ผ่านทุกเคส" (exit 0) เคสที่ต้องจับ drift ต้องได้ exit 3 ไม่ใช่แค่ไม่ล้ม
set -uo pipefail
DIR="$(cd "$(dirname "$0")/.." && pwd)"
T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT
fail=0

mkdir -p "$T/routes"
cat > "$T/routes/demo.json" <<'EOF'
{"name":"demo","uri":"/*","plugins":{"serverless-pre-function":{"phase":"access","functions":["return function() end"]}},"upstream":{"type":"roundrobin","nodes":{"temporal-ui:8080":1}}}
EOF

# mk สร้างผลจำลองของ Admin API: ค่าที่ git เขียน + ค่า default ที่เซิร์ฟเวอร์เติมให้ (ต้องไม่นับเป็น drift)
mk() { # mk <ชื่อเคส> <python statement แก้ตัวแปร v และ extra> -> $T/etcd.json
  python3 - "$2" > "$T/etcd.json" <<'EOF'
import json, sys
v = {"id": "demo", "name": "demo", "uri": "/*", "create_time": 1, "update_time": 2, "status": 1,
     "plugins": {"serverless-pre-function": {"phase": "access", "functions": ["return function() end"]}},
     "upstream": {"type": "roundrobin", "pass_host": "pass", "scheme": "http", "hash_on": "vars",
                  "nodes": {"temporal-ui:8080": 1}}}
extra = []
exec(sys.argv[1])
items = [{"key": "/apisix/routes/" + v["id"], "value": v}] + extra
print(json.dumps({"total": len(items), "list": items}))
EOF
}

expect() { # expect <ชื่อเคส> <exit ที่คาด> <ข้อความที่ต้องเห็น (ว่างได้)>
  out="$(python3 "$DIR/drift.py" --etcd "$T/etcd.json" --routes "$T/routes" 2>&1)"
  rc=$?
  if [ "$rc" = "$2" ] && { [ -z "${3:-}" ] || printf '%s' "$out" | grep -qF -- "$3"; }; then
    echo "✓ $1"
  else
    echo "✗ $1 (exit=$rc คาด $2, ข้อความที่ต้องมี: '${3:-}')"; echo "$out" | sed 's/^/    /'; fail=1
  fi
}

mk ok ""
expect "ตรงกับ git (มี default ของเซิร์ฟเวอร์เติม ไม่นับเป็น drift)" 0 "ไม่มี drift"

mk uri 'v["uri"] = "/other"'
expect "แก้ uri" 3 "demo.uri"

mk plug 'v["plugins"]["limit-count"] = {"count": 1}'
expect "เพิ่ม plugin ผ่าน UI" 3 "demo.plugins.limit-count"

mk node 'v["upstream"]["nodes"]["evil:80"] = 1'
expect "เพิ่ม upstream node" 3 "evil:80"

mk lua 'v["plugins"]["serverless-pre-function"]["functions"] = ["return function() return end"]'
expect "แก้ Lua" 3 "functions[0]"

mk extra 'extra = [{"key": "/apisix/routes/x", "value": {"id": "x", "uri": "/x"}}]'
expect "route ใหม่ที่สร้างผ่าน dashboard" 3 "route x อยู่ใน etcd"

mk missing 'v["id"] = "other"'
expect "route ของ git หายจาก etcd" 3 "route demo มีใน git แต่ไม่มีใน etcd"

echo '{"total": 2, "nodes": []}' > "$T/etcd.json"
expect "รูปแบบผลลัพธ์ไม่ตรงที่คาด (total>0 แต่ไม่มี list)" 1 "อ่านรายการ route ไม่ได้"

echo '{"not json' > "$T/etcd.json"
expect "ข้อมูลเสีย" 1 "อ่านข้อมูลไม่ได้"

[ "$fail" = "0" ] && echo "ผ่านทุกเคส" || { echo "มีเคสไม่ผ่าน" >&2; exit 1; }
