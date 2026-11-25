#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""สร้าง apisix.yaml (โหมดไฟล์ ใช้กับชุด dev) จาก routes/*.json (แหล่งจริง)

ใช้ผ่าน gen-yaml.sh: spec apisix-admin-dashboard หัวข้อ Decisions ข้อ 1
  gen_yaml.py            เขียน apisix.yaml
  gen_yaml.py --check    ตรวจว่า apisix.yaml ที่ commit ตรงกับที่สร้างใหม่ (ต่างกัน exit 1 พร้อม diff)
ใช้เฉพาะ python3 มาตรฐาน
"""
import argparse
import difflib
import glob
import json
import os
import re
import sys

HEADER = """\
# สร้างจาก routes/*.json ด้วย gen-yaml.sh — ห้ามแก้ไฟล์นี้ด้วยมือ (แก้ JSON แล้วรัน ./gen-yaml.sh)
# ไฟล์นี้ใช้กับ gateway โหมดไฟล์ของชุด dev (projects/bot-orchestration-platform/docker-compose.yml)
# route จริงบน Mini PC อยู่ใน etcd และถูก seed จากไฟล์ JSON ชุดเดียวกัน (ADR 0003)
#
# route นี้ให้เฉพาะผู้ที่ login BOP เข้า Temporal UI ได้ (ADR 0002):
# - ตรวจ session ด้วย Lua เองแทน plugin forward-auth เพราะ forward-auth ตอบได้แค่สถานะของ
#   บริการตรวจสิทธิ์ (401) ทำ redirect ไปหน้า login ของ dashboard ไม่ได้
# - ตัด Cookie ด้วย serverless-post-function ที่รันหลัง pre-function ห้ามใช้ proxy-rewrite
#   เพราะรันก่อนขั้นตรวจจึงลบ cookie ทิ้งก่อนถูกส่งไปถาม /me ทำให้ทุกคนไม่ผ่าน
"""

PLAIN_KEY = re.compile(r"^[A-Za-z_][A-Za-z0-9_-]*$")


def key(k):
    return k if PLAIN_KEY.match(k) else json.dumps(k, ensure_ascii=False)


def scalar(v):
    if v is None:
        return "null"
    if v is True:
        return "true"
    if v is False:
        return "false"
    if isinstance(v, (int, float)):
        return json.dumps(v)
    return json.dumps(v, ensure_ascii=False)  # สตริงขึ้นบรรทัดเดียว: ใช้รูปแบบ double-quoted เสมอ ไม่ให้ถูกตีเป็น bool/null


def block(s, ind):
    """สตริงหลายบรรทัดเป็น block scalar"""
    lines = s.split("\n")
    if s.endswith("\n\n"):
        raise ValueError("สตริงลงท้ายด้วยบรรทัดว่างหลายบรรทัดไม่รองรับ")
    if lines[0][:1] in (" ", "\t"):
        raise ValueError("บรรทัดแรกของสตริงหลายบรรทัดขึ้นต้นด้วยช่องว่าง ไม่รองรับ")
    style = "|" if s.endswith("\n") else "|-"
    body = lines[:-1] if s.endswith("\n") else lines
    return style, [(" " * ind + ln) if ln else "" for ln in body]


def emit(v, ind):
    pad = " " * ind
    out = []
    if isinstance(v, dict):
        for k, val in v.items():
            if isinstance(val, dict) and val:
                out.append(f"{pad}{key(k)}:")
                out += emit(val, ind + 2)
            elif isinstance(val, list) and val:
                out.append(f"{pad}{key(k)}:")
                out += emit(val, ind + 2)
            elif isinstance(val, str) and "\n" in val:
                style, body = block(val, ind + 2)
                out.append(f"{pad}{key(k)}: {style}")
                out += body
            elif isinstance(val, dict):
                out.append(f"{pad}{key(k)}: {{}}")
            elif isinstance(val, list):
                out.append(f"{pad}{key(k)}: []")
            else:
                out.append(f"{pad}{key(k)}: {scalar(val)}")
    elif isinstance(v, list):
        for item in v:
            if isinstance(item, str) and "\n" in item:
                style, body = block(item, ind + 2)
                out.append(f"{pad}- {style}")
                out += body
            elif isinstance(item, (dict, list)):
                inner = emit(item, ind + 2)
                inner[0] = pad + "- " + inner[0][ind + 2:]
                out += inner
            else:
                out.append(f"{pad}- {scalar(item)}")
    return out


def build(routes_dir):
    files = sorted(glob.glob(os.path.join(routes_dir, "*.json")))
    if not files:
        raise ValueError(f"ไม่พบ {routes_dir}/*.json")
    routes = []
    for p in files:
        with open(p, encoding="utf-8") as f:
            r = json.load(f)
        rid = os.path.splitext(os.path.basename(p))[0]
        routes.append({"id": rid, **r})
    return HEADER + "\n".join(["routes:"] + emit(routes, 2)) + "\n#END\n"


def main():
    here = os.path.dirname(os.path.abspath(__file__))
    ap = argparse.ArgumentParser()
    ap.add_argument("--routes", default=os.path.join(here, "routes"))
    ap.add_argument("--file", default=os.path.join(here, "apisix.yaml"))
    ap.add_argument("--check", action="store_true")
    args = ap.parse_args()
    try:
        text = build(args.routes)
    except (OSError, ValueError) as e:
        print(f"สร้างไม่ได้: {e}", file=sys.stderr)
        return 2
    if args.check:
        try:
            current = open(args.file, encoding="utf-8").read()
        except OSError as e:
            print(f"อ่าน {args.file} ไม่ได้: {e}", file=sys.stderr)
            return 2
        if current == text:
            print("apisix.yaml ตรงกับที่สร้างจาก routes/*.json")
            return 0
        sys.stdout.writelines(
            difflib.unified_diff(current.splitlines(True), text.splitlines(True), "apisix.yaml (ที่ commit)", "ที่สร้างใหม่")
        )
        print("\napisix.yaml ไม่ตรงกับ routes/*.json — รัน ./gen-yaml.sh แล้ว commit", file=sys.stderr)
        return 1
    with open(args.file, "w", encoding="utf-8") as f:
        f.write(text)
    print(f"เขียน {args.file}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
