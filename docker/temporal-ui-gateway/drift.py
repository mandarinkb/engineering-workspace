#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""เทียบ route ใน etcd (ผลของ GET /apisix/admin/routes) กับ routes/*.json ใน git

ใช้โดย export.sh และ seed.sh (ADR 0003 กลไกข้อ 1) ใช้ได้เฉพาะ python3 มาตรฐาน

กติกา: Admin API เติมค่า default ให้ route (เช่น upstream.pass_host, create_time) จึงเทียบแบบ
"ทุกอย่างที่เขียนไว้ใน git ต้องตรงใน etcd" (subset) ยกเว้นสองจุดที่ต้องตรงทุกคีย์ เพราะการเพิ่มของจาก UI
คือสิ่งที่ต้องการจับ: ชื่อ plugin ใน `plugins` และ node ใน `nodes`

exit: 0 ไม่มี drift, 3 พบ drift, 1 อ่านข้อมูลไม่ได้
"""
import argparse
import glob
import json
import os
import sys

IGNORED = {"id", "create_time", "update_time"}
EXACT_KEYS = {"plugins", "nodes"}


def load_etcd(path):
    with open(path, encoding="utf-8") as f:
        data = json.load(f)
    items = data.get("list") or data.get("items") or []
    if not items and data.get("total"):
        # รูปแบบผลลัพธ์ของ Admin API ต่างจากที่คาด: ห้ามสรุปว่า "ไม่มี drift" จากข้อมูลที่อ่านไม่ออก
        raise ValueError("total > 0 แต่อ่านรายการ route ไม่ได้ (คาดว่ามีคีย์ list)")
    routes = {}
    for it in items:
        v = it.get("value", it)
        rid = v.get("id") or str(it.get("key", "")).rsplit("/", 1)[-1]
        routes[str(rid)] = v
    return routes


def load_git(directory):
    routes = {}
    for p in sorted(glob.glob(os.path.join(directory, "*.json"))):
        with open(p, encoding="utf-8") as f:
            routes[os.path.splitext(os.path.basename(p))[0]] = json.load(f)
    return routes


def diff(expected, actual, path, out):
    """บันทึกความต่างลง out: expected (git) ต้องเป็น subset ของ actual (etcd)"""
    if isinstance(expected, dict):
        if not isinstance(actual, dict):
            out.append(f"{path}: ใน git เป็น object แต่ใน etcd เป็น {type(actual).__name__}")
            return
        key = path.rsplit(".", 1)[-1]
        if key in EXACT_KEYS:
            for extra in sorted(set(actual) - set(expected)):
                out.append(f"{path}.{extra}: มีใน etcd แต่ไม่มีใน git")
        for k, v in expected.items():
            if k not in actual:
                out.append(f"{path}.{k}: มีใน git แต่ไม่มีใน etcd")
            else:
                diff(v, actual[k], f"{path}.{k}", out)
    elif isinstance(expected, list):
        if not isinstance(actual, list) or len(expected) != len(actual):
            out.append(f"{path}: รายการต่างกัน (git {len(expected)} รายการ)")
            return
        for i, (e, a) in enumerate(zip(expected, actual)):
            diff(e, a, f"{path}[{i}]", out)
    elif expected != actual:
        out.append(f"{path}: ค่าต่างกัน")


def normalize(route):
    return {k: v for k, v in route.items() if k not in IGNORED}


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--etcd", required=True, help="ไฟล์ JSON ผลของ GET /apisix/admin/routes")
    ap.add_argument("--routes", required=True, help="โฟลเดอร์ routes/*.json ใน git")
    ap.add_argument("--out", help="เขียน route จาก etcd (จัดรูปแบบคงที่) ลงโฟลเดอร์นี้")
    args = ap.parse_args()
    try:
        etcd = load_etcd(args.etcd)
        git = load_git(args.routes)
    except (OSError, ValueError) as e:
        print(f"อ่านข้อมูลไม่ได้: {e}", file=sys.stderr)
        return 1

    if args.out:
        os.makedirs(args.out, exist_ok=True)
        for rid, r in etcd.items():
            with open(os.path.join(args.out, f"{rid}.json"), "w", encoding="utf-8") as f:
                json.dump(normalize(r), f, ensure_ascii=False, indent=2, sort_keys=True)
                f.write("\n")

    findings = []
    for rid, expected in git.items():
        if rid not in etcd:
            findings.append(f"drift: route {rid} มีใน git แต่ไม่มีใน etcd")
            continue
        sub = []
        diff(expected, etcd[rid], rid, sub)
        findings.extend(f"drift: {s}" for s in sub)
    for rid in sorted(set(etcd) - set(git)):
        findings.append(f"drift: route {rid} อยู่ใน etcd แต่ไม่มีใน routes/ (สร้างผ่าน dashboard?)")

    for line in findings:
        print(line)
    if findings:
        return 3
    print(f"ไม่มี drift ({len(git)} route ตรงกับ etcd)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
