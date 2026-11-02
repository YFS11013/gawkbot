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
# The spec resolves `python` through PATH, where a Windows box often has the
# WindowsApps stub ahead of the real interpreter; the stub answers with a
# store prompt (or exit 9009) and the gate reads that as a failed deliverable.
# Pin the interpreter that is running this script instead.
INTERPRETER = sys.executable or "python"

# Both kinds run the same gate: the four-section shape is what makes a pack
# usable, and it does not change when the counterparty is a candidate instead
# of a customer. Only the directory, the section titles and the wording move.
# Keeping one gate is the point -- a second checker would drift from this one.
KINDS = {
    "meeting": {
        "dir": "team/prep",
        "playbook": "meeting-prep",
        "pack": "准备包",
        "noun": "会面",
        "sections": "## 对方近况 / ## 我方既有结论 / ## 要问的问题 / ## 如果他说…",
        "title": "准备包：与 %s 的会面（%s）",
        "title_rewrite": "重写准备包：与 %s 的会面（%s）",
        "goal": "为 %s 与 %s 的会面产出一页纸准备包（%s）",
        "criteria": (
            "「对方近况」每条带日期与出处（wiki 路径/链接/任务号）",
            "「我方既有结论」引用至少一个既有产物（wiki 路径或任务号）",
            "问题至少 3 条，分别问事实、问判断、问下一步",
            "至少 1 条 如果 X → 说明 Y → 我们改 Z 的分支",
        ),
        "details": (
            "%s 要与 %s 会面。请按 meeting-prep playbook 产出一页纸准备包，路径 %s。"
            "重点：对方最近说了什么（带日期与出处）、我们上次的结论是否仍成立、"
            "三个该问的问题，以及至少一条「如果他说 X → 说明 Y → 我们改 Z」的分支。"
        ),
    },
    "candidate": {
        "dir": "team/candidates",
        "playbook": "candidate-prep",
        "pack": "候选人简报",
        "noun": "面试",
        "sections": "## 候选人背景 / ## 与岗位的匹配 / ## 要问的问题 / ## 如果他说…",
        "title": "候选人简报：%s（%s）",
        "title_rewrite": "重写候选人简报：%s（%s）",
        "goal": "为 %s 与候选人 %s 的面试产出一页纸简报（%s）",
        "criteria": (
            "「候选人背景」每条带日期与出处（公开产出/简历/wiki 路径/链接）",
            "「与岗位的匹配」引用岗位要求或既有面试记录（wiki 路径或任务号）",
            "问题至少 3 条，分别问事实、问判断、问下一步",
            "至少 1 条 如果他说 X → 说明 Y → 下一步 Z 的分支",
        ),
        "details": (
            "%s 要面试 %s。请按 candidate-prep playbook 产出一页纸候选人简报，路径 %s。"
            "重点：候选人公开可查的产出（带日期与出处）、与岗位要求逐条对得上/对不上、"
            "三个该问的问题，以及至少一条「如果他答 X → 说明 Y → 下一步 Z」的分支。"
            "简历上的自我陈述不算出处，必须指到能打开的材料。"
        ),
    },
}


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
    ap.add_argument(
        "--kind",
        default="meeting",
        choices=sorted(KINDS),
        help="what kind of pack: a meeting with someone, or a candidate interview",
    )
    ap.add_argument("--date", default=None, help="meeting date YYYY-MM-DD (default tomorrow)")
    ap.add_argument("--owner", default=DEFAULT_OWNER)
    ap.add_argument("--topic", default="", help="optional theme")
    ap.add_argument("--port", type=int, default=DEFAULT_PORT)
    ap.add_argument("--approve", action="store_true", help="approve straight away")
    ap.add_argument(
        "--forbid",
        action="append",
        default=[],
        help="literal phrase the pack must not contain (repeatable); for rewrites",
    )
    ap.add_argument(
        "--details-file",
        default=None,
        help="read task details from this file (rewrite instructions live in a file, not argv)",
    )
    ap.add_argument(
        "--rewrite",
        action="store_true",
        help="rewrite in place: same artifact path, title says so",
    )
    ap.add_argument("--dry-run", action="store_true", help="print the payload only")
    a = ap.parse_args(argv)

    kind = KINDS[a.kind]
    date = a.date or (dt.date.today() + dt.timedelta(days=1)).isoformat()
    dt.date.fromisoformat(date)  # fail early on a malformed date
    path = "%s/%s-%s.md" % (kind["dir"], slugify(a.who), date)

    criteria = list(kind["criteria"])
    goal = kind["goal"] % (date, a.who, path)
    if a.topic:
        goal += "；主题：%s" % a.topic
    details = kind["details"] % (date, a.who, path)
    if a.details_file:
        with open(a.details_file, encoding="utf-8") as f:
            details = f.read()
    criteria = list(criteria)
    if a.forbid:
        criteria.append("不得出现这些字面串：" + "、".join(a.forbid))
    definition = {
        "goal": goal,
        "deliverables": [
            {
                "name": kind["pack"],
                "format": "markdown，四个小节：%s，路径 %s" % (kind["sections"], path),
            }
        ],
        "success_criteria": criteria,
        "artifact_path": path,
    }
    create = {
        "action": "create",
        "title": (kind["title_rewrite"] if a.rewrite else kind["title"]) % (a.who, date),
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
        "verification_spec": '"%s" "%s" "%s"%s'
        % (
            INTERPRETER,
            GATE,
            path,
            "".join(' --forbid "%s"' % f for f in a.forbid),
        ),
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
