#!/usr/bin/env python3
"""Mechanical gate for a meeting-prep pack (team/prep/<slug>-<date>.md).

A prep pack is only worth writing if every claim about the counterparty can
be traced and the pack actually gives the reader something to do. That is
checkable, so it is checked: this script is the `command` verification spec
for a prep task, and it fails closed on the four ways a pack goes hollow.

Rules (all of them are refusals of a specific bad pack):
  1. four required sections exist;
  2. every bullet under the counterparty section carries a date AND a source
     -- a claim about what someone said with no date and no link is exactly
     the fabrication this gate exists to stop;
  3. the "our side" section cites a real prior artifact (a wiki path, a task
     id, or a review), so the pack connects to what we already concluded;
  4. at least three questions and at least one "if he says X then Y" branch.

Usage:
    python check_prep_pack.py <path>              # relative to --wiki, or absolute
    python check_prep_pack.py --self-test         # prove the rules themselves

Exit 0 pass, 1 fail, 2 usage error. Output is ASCII on purpose: the spec
runs under `cmd /C` on Windows, and a gate must never fail on an encoding.
"""

import argparse
import os
import re
import sys

REQUIRED = {
    "counterparty": ("对方", "对象", "对谈", "与会者", "counterparty"),
    "our_side": ("我方", "既有结论", "既有论点", "我们的判断", "our side"),
    "questions": ("问题", "要问", "questions", "question"),
    "branches": ("如果", "分支", "应对", "scenario", "branch"),
}

DATE_RE = re.compile(r"(\d{4}-\d{2}-\d{2}|\d{4}-\d{2}|\d{4}\s*年\s*\d{1,2}\s*月)")
# A source is a link, an explicit marker, or a path into something a reader
# can open. Deliberately generous: the gate is against claims with NO
# provenance, not against a particular citation style.
SOURCE_RE = re.compile(
    r"(\[出处|出处[::]|\[src|来源[::]|\]\(http|https?://|team/|\.md\b|\.json\b|ACMER-\d+|src[::])",
    re.IGNORECASE,
)
PRIOR_ARTIFACT_RE = re.compile(r"(\.md\b|team/|ACMER-\d+|review|任务)", re.IGNORECASE)
# A bullet starting with one of these is a stated absence, not a claim.
PLACEHOLDER_RE = re.compile(r"^\s*(?:[（(]?\s*)?(?:待确认|未收录|未核实|无记录|没有记录|n/?a\b)", re.IGNORECASE)
BRANCH_RE = re.compile(r"(→|->|=>|则|那么|then)")
BULLET_RE = re.compile(r"^\s*(?:[-*+]|\d+[.)])\s+(.*\S)\s*$")


def split_sections(text):
    """Return [(title, [lines])] for every '## ' heading."""
    sections, title, lines = [], None, []
    for line in text.splitlines():
        m = re.match(r"^##\s+(.*\S)\s*$", line)
        if m:
            if title is not None:
                sections.append((title, lines))
            title, lines = m.group(1), []
            continue
        if title is not None:
            lines.append(line)
    if title is not None:
        sections.append((title, lines))
    return sections


def bullets(lines):
    out = []
    for line in lines:
        m = BULLET_RE.match(line)
        if m:
            out.append(m.group(1))
    return out


def check(text):
    """Return a list of failure strings; empty means the pack passes."""
    fails = []
    sections = split_sections(text)
    by_key = {}
    for key, aliases in REQUIRED.items():
        hit = None
        for title, lines in sections:
            low = title.lower()
            if any(a.lower() in low for a in aliases):
                hit = (title, lines)
                break
        if hit is None:
            fails.append(
                "missing section: %s (one of %s)"
                % (key, "/".join(REQUIRED[key][:2]))
            )
        else:
            by_key[key] = hit

    if "counterparty" in by_key:
        items = bullets(by_key["counterparty"][1])
        if not items:
            fails.append("counterparty section has no bullets")
        # A bullet that says "not found" is the honest answer, not a claim
        # about the counterparty: it carries no date or source by nature, and
        # refusing it would push writers toward inventing one. It still may
        # not BE the section — an all-placeholder section is a pack that
        # learned nothing.
        real = 0
        for i, item in enumerate(items, 1):
            if PLACEHOLDER_RE.match(item):
                continue
            real += 1
            if not DATE_RE.search(item):
                fails.append("counterparty bullet %d has no date: %s" % (i, clip(item)))
            if not SOURCE_RE.search(item):
                fails.append("counterparty bullet %d has no source: %s" % (i, clip(item)))
        if items and real == 0:
            fails.append(
                "counterparty section is all placeholders: at least one dated, sourced fact is required"
            )

    if "our_side" in by_key:
        items = bullets(by_key["our_side"][1])
        if not items:
            fails.append("our-side section has no bullets")
        elif not any(PRIOR_ARTIFACT_RE.search(x) for x in items):
            fails.append(
                "our-side section cites no prior artifact (wiki path, task id, or review)"
            )

    if "questions" in by_key:
        items = bullets(by_key["questions"][1])
        if len(items) < 3:
            fails.append("needs at least 3 questions, found %d" % len(items))

    if "branches" in by_key:
        items = bullets(by_key["branches"][1])
        if not items:
            fails.append("branch section has no bullets")
        elif not any(BRANCH_RE.search(x) for x in items):
            fails.append("branch section has no 'if X then Y' mapping (no -> / => / then)")

    return fails


def clip(s, n=90):
    s = " ".join(s.split())
    return s if len(s) <= n else s[: n - 1] + "…"


def resolve(path, wiki):
    if os.path.isabs(path):
        return path
    return os.path.join(wiki, path)


def default_wiki():
    home = os.path.expanduser("~")
    return os.path.join(home, ".wuphf", "wiki")


GOOD = """# 与 X 的准备包

## 对方近况
- 2026-10-28 在会议上说要对冲成本上升（出处: https://example.com/x-1028.html）
- 2026-10 月报显示净流入转负（来源: team/forex-brief-2026-10-31.md）

## 我方既有结论
- 10-31 简报判断美元走弱（team/forex-brief-2026-10-31.md）

## 要问的问题
- 成本上升具体指哪一项？
- 净流入转负是赎回还是估值下跌？
- 是否考虑缩短久期？

## 如果他说需求疲软
- 需求疲软 → 下调明年的量价假设
"""

BAD_NO_DATE = GOOD.replace("2026-10-28 在会议上说要对冲成本上升", "他在会议上说要对冲成本上升")
BAD_FEW_Q = GOOD.replace("- 是否考虑缩短久期？\n", "")
BAD_NO_SRC = GOOD.replace("（出处: https://example.com/x-1028.html）", "")
BAD_NO_BRANCH = GOOD.replace("- 需求疲软 → 下调明年的量价假设", "- 需求疲软的话再看")
# An honest "we do not have this" bullet must not fail the gate...
GOOD_WITH_PLACEHOLDER = GOOD.replace(
    "- 2026-10 月报显示净流入转负（来源: team/forex-brief-2026-10-31.md）",
    "- 2026-10 月报显示净流入转负（来源: team/forex-brief-2026-10-31.md）\n- 未收录：持仓明细与组合净值，不推断",
)
# ...but a section made only of them is a pack that learned nothing.
ALL_PLACEHOLDER = GOOD.replace(
    "- 2026-10-28 在会议上说要对冲成本上升（出处: https://example.com/x-1028.html）\n"
    "- 2026-10 月报显示净流入转负（来源: team/forex-brief-2026-10-31.md）",
    "- 未收录：没有任何关于对方的记录",
)


def self_test():
    cases = [
        ("good", GOOD, []),
        ("no-date", BAD_NO_DATE, ["no date"]),
        ("no-source", BAD_NO_SRC, ["no source"]),
        ("few-questions", BAD_FEW_Q, ["at least 3 questions"]),
        ("no-branch-map", BAD_NO_BRANCH, ["if X then Y"]),
        ("placeholder-ok", GOOD_WITH_PLACEHOLDER, []),
        ("all-placeholder", ALL_PLACEHOLDER, ["all placeholders"]),
        ("empty", "", ["missing section"]),
    ]
    bad = 0
    for name, text, expect in cases:
        got = check(text)
        ok = all(any(e in f for f in got) for e in expect) if expect else not got
        if ok and expect:
            ok = len(got) >= len(expect)
        print("%-14s %s %s" % (name, "PASS" if ok else "FAIL", "; ".join(got) or "clean"))
        if not ok:
            bad += 1
    print("self-test: %d/%d" % (len(cases) - bad, len(cases)))
    return 0 if bad == 0 else 1


def main(argv):
    # The spec runs under `cmd /C`, whose console encoding is cp936 on a
    # zh-CN box. The broker reads the pipe as UTF-8, so Chinese failure text
    # arrived mojibake and the writer could not act on it. Force UTF-8
    # regardless of the console's code page.
    try:
        sys.stdout.reconfigure(encoding="utf-8")
    except Exception:
        pass
    ap = argparse.ArgumentParser(add_help=True)
    ap.add_argument("path", nargs="?")
    ap.add_argument("--wiki", default=default_wiki())
    ap.add_argument("--self-test", action="store_true")
    a = ap.parse_args(argv)
    if a.self_test:
        return self_test()
    if not a.path:
        print("usage: check_prep_pack.py <artifact path under team/prep/>", file=sys.stderr)
        return 2
    full = resolve(a.path.strip().strip('"'), a.wiki)
    if not os.path.isfile(full):
        print("prep check failed: no such file: %s" % full)
        return 1
    with open(full, encoding="utf-8-sig", errors="replace") as f:
        text = f.read()
    if not text.strip():
        print("prep check failed: file is empty")
        return 1
    fails = check(text)
    if fails:
        print("prep check failed (%d):" % len(fails))
        for f_ in fails:
            print("  - " + f_)
        return 1
    secs = sections_by_key(text)
    cp = bullets(secs["counterparty"][1]) if "counterparty" in secs else []
    real = sum(1 for b in cp if not PLACEHOLDER_RE.match(b))
    print(
        "prep ok: 4/4 sections; counterparty %d dated+sourced (+%d stated-absent); questions %d; branches %d"
        % (
            real,
            len(cp) - real,
            len(bullets(secs["questions"][1])) if "questions" in secs else 0,
            len(bullets(secs["branches"][1])) if "branches" in secs else 0,
        )
    )
    return 0


def sections_by_key(text):
    out = {}
    for title, lines in split_sections(text):
        low = title.lower()
        for key, aliases in REQUIRED.items():
            if key not in out and any(a.lower() in low for a in aliases):
                out[key] = (title, lines)
    return out


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
