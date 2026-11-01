#!/usr/bin/env python3
"""Open a meeting-prep task (the A4 use case) against the running broker.

Two ways to get a prep pack, both of them a single command:

    python new_prep_task.py --who fund-invest --date 2026-11-02
    python new_prep_task.py --who "Acme 的采购负责人" --date 2026-11-03 --owner cos --topic 续约

Why a script and not a curl snippet: the task needs BOTH a structured
definition and a machine gate whose spec embeds a literal artifact path. Hand
writing that twice is how the path and the spec drift apart, and a drifted
spec either passes everything or blocks honest work. One source of truth here:
the path is computed once and used for `artifact_path`, for the success
criteria text, and for the verification spec.

The gate is scripts/office/check_prep_pack.py. It runs on `cmd /C` under
Windows and `sh -c` elsewhere, so the spec quotes both paths and stays free
of shell metacharacters beyond quoting.

Exit codes: 0 created, 1 broker refused, 2 usage.
"""

import argparse
import datetime as dt
import json
import os
import re
import sys
import urllib.error
import urllib.request

DEFAULT_PORT = 7892
DEFAULT_OWNER = "data-keeper"
GATE = os.path.join(
    os.path.dirname(os.path.abspath(__file__)), "check_prep_pack.py"
)


def slugify(who):
    """ASCII slug for the file name; non-ASCII names get a stable hash."""
    s = re.sub(r"[^a-z0-9]+", "-", who.lower()).strip("-")
    if s:
        return s[:40]
    # Chinese (or any non-latin) name: keep a short fingerprint so two
    # different counterparties never collide on the same prep file.
    import hashlib

    return "entity-" + hashlib.sha1(who.encode("utf-8")).hexdigest()[:8]


def token():
    t = os.environ.get("WUPHF_TOKEN", "").strip()
    if t:
        return t
    path = os.path.join(os.environ.get("TEMP", "/tmp"), "wuphf-broker-token-%d" % DEFAULT_PORT)
    with open(path, encoding="utf-8") as f:
        return f.read().strip()


def post(port, tok, body):
    req = urllib.request.Request(
        "http://127.0.0.1:%d/tasks" % port,
        data=json.dumps(body, ensure_ascii=False).encode("utf-8"),
        headers={"Authorization": "Bearer " + tok, "Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=60) as f:
            return f.status, json.load(f)
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode("utf-8", "replace")


def main(argv):
    ap = argparse.ArgumentParser()
    ap.add_argument("--who", required=True, help="counterparty name")
    ap.add_argument("--date", default=None, help="meeting date YYYY-MM-DD (default tomorrow)")
    ap.add_argument("--owner", default=DEFAULT_OWNER)
    ap.add_argument("--topic", default="", help="optional theme")
    ap.add_argument("--port", type=int, default=DEFAULT_PORT)
    ap.add_argument("--approve", action="store_true", help="approve straight away")
    ap.add_argument("--dry-run", action="store_true", help="print the payload only")
    a = ap.parse_args(argv)

    date = a.date or (dt.date.today() + dt.timedelta(days=1)).isoformat()
    dt.date.fromisoformat(date)  # fail early on a malformed date
    path = "team/prep/%s-%s.md" % (slugify(a.who), date)

    criteria = [
        "「对方近况」每条带日期与出处（wiki 路径/链接/任务号）",
        "「我方既有结论」引用至少一个既有产物（wiki 路径或任务号）",
        "问题至少 3 条，分别问事实、问判断、问下一步",
        "至少 1 条 如果 X → 说明 Y → 我们改 Z 的分支",
    ]
    goal = "为 %s 与 %s 的会面产出一页纸准备包（%s）" % (date, a.who, path)
    if a.topic:
        goal += "；主题：%s" % a.topic
    details = (
        "%s 要与 %s 会面。请按 meeting-prep playbook 产出一页纸准备包，路径 %s。"
        "重点：对方最近说了什么（带日期与出处）、我们上次的结论是否仍成立、"
        "三个该问的问题，以及至少一条「如果他说 X → 说明 Y → 我们改 Z」的分支。"
        % (date, a.who, path)
    )
    definition = {
        "goal": goal,
        "deliverables": [
            {
                "name": "准备包",
                "format": "markdown，四个小节：## 对方近况 / ## 我方既有结论 / ## 要问的问题 / ## 如果他说…，路径 " + path,
            }
        ],
        "success_criteria": criteria,
        "artifact_path": path,
    }
    create = {
        "action": "create",
        "title": "准备包：与 %s 的会面（%s）" % (a.who, date),
        "details": details,
        "created_by": "human",
        "owner": a.owner,
        "channel": "%s__human" % a.owner,
        "task_type": "issue",
        "execution_mode": "office",
    }
    define = {
        "action": "define",
        "created_by": "human",
        "definition": definition,
        "verification_kind": "command",
        "verification_spec": 'python "%s" "%s"' % (GATE, path),
        "verification_required": True,
    }

    if a.dry_run:
        print(json.dumps({"create": create, "define": define}, ensure_ascii=False, indent=1))
        return 0

    tok = token()
    status, resp = post(a.port, tok, create)
    if status != 200:
        print("create failed: HTTP %s %s" % (status, resp))
        return 1
    task = resp.get("task", resp)
    tid = task.get("id", "")
    if not tid:
        print("create returned no task id: %s" % resp)
        return 1

    define["id"] = tid
    status, resp = post(a.port, tok, define)
    if status != 200:
        print("created %s but define failed: HTTP %s %s" % (tid, status, resp))
        return 1

    if a.approve:
        status, resp = post(
            a.port, tok, {"action": "approve", "id": tid, "created_by": "human", "channel": task.get("channel")}
        )
        if status != 200:
            print("%s defined but approve failed: HTTP %s %s" % (tid, status, resp))
            return 1

    print("%s  %s  owner=%s  gate=%s" % (tid, path, a.owner, os.path.basename(GATE)))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
