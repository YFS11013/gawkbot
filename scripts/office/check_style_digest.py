#!/usr/bin/env python3
"""Gate for the style knowledge base (the other half of the N1 loop).

record_style_diff.py records what a human changed. That alone learns nothing:
a pile of diffs that nobody turns into a rule is just storage. This gate is
the part that refuses the empty ceremony, on four specific failures:

  1. the style file does not exist, or has no `## rule:` entry;
  2. an entry has no `evidence:` line -- a rule with no diff behind it is an
     opinion, and the next bot will delete it;
  3. an `evidence:` id does not appear in the diffs jsonl -- fabricated or
     stale provenance;
  4. the same rule is written twice (normalised, so punctuation and spacing
     do not hide a duplicate) -- duplicates mean the bot appended instead of
     merging, and the file stops being readable within weeks.

Plus one freshness rule: if diffs landed inside the window, the style file's
frontmatter `updated:` must be at least as new as the last of them. Without
it, the loop can quietly stop at "we recorded the edits".

    python check_style_digest.py --bot henji --since 7d
    python check_style_digest.py --bot henji --self-test

Exit 0 pass, 1 fail, 2 usage. ASCII output on purpose: this runs under
`cmd /C` on Windows and must never fail on an encoding.
"""

import argparse
import datetime as dt
import json
import os
import re
import sys

DEFAULT_ROOT = os.path.join(os.path.expanduser("~"), ".wuphf", "wiki")
DATE_RE = re.compile(r"^\s*(\d{4}-\d{2}-\d{2})\s*$")
TS_RE = re.compile(r"^(\d{4}-\d{2}-\d{2})T")
ENTRY_RE = re.compile(r"^##\s+(.*\S)\s*$")
RULE_RE = re.compile(r"^rule\s*[:：]\s*(.+)$", re.IGNORECASE)
EVIDENCE_RE = re.compile(r"^\s*evidence\s*[:：]\s*(.+)$", re.IGNORECASE)
TOKEN_SPLIT_RE = re.compile(r"[,，、\s]+")
# Everything that is not a letter/digit (Latin or CJK) is dropped before
# comparing rules, so "不回「感谢来信」" and "不回 感谢来信" collide on purpose.
NORM_RE = re.compile(r"[^0-9a-z\u4e00-\u9fff]+")


def parse_frontmatter(text):
    """Return (frontmatter dict, body). Missing frontmatter yields ({}, text)."""
    lines = text.splitlines()
    if not lines or lines[0].strip() != "---":
        return {}, text
    data = {}
    for i in range(1, len(lines)):
        line = lines[i]
        if line.strip() == "---":
            return data, "\n".join(lines[i + 1 :])
        if ":" in line:
            k, v = line.split(":", 1)
            data[k.strip().lower()] = v.strip()
    return {}, text


def parse_entries(body):
    """Return [(rule_text, evidence_tokens)] in file order."""
    out, rule, ev = [], None, []
    for line in body.splitlines():
        m = ENTRY_RE.match(line)
        if m:
            if rule is not None:
                out.append((rule, ev))
            title = m.group(1)
            rm = RULE_RE.match(title)
            rule = rm.group(1) if rm else None
            raw_title = title if rm is None else None
            if raw_title is not None:
                # Not a rule entry: keep it so the caller can report it, with
                # an impossible rule text marker instead of silently dropping.
                rule = "\x00" + raw_title
            ev = []
            continue
        if rule is not None:
            em = EVIDENCE_RE.match(line)
            if em:
                ev.extend(t for t in TOKEN_SPLIT_RE.split(em.group(1)) if t)
    if rule is not None:
        out.append((rule, ev))
    return out


def norm(s):
    return NORM_RE.sub("", s.lower())


def load_diffs(path):
    recs = []
    if not os.path.isfile(path):
        return recs
    with open(path, encoding="utf-8", errors="replace") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            try:
                rec = json.loads(line)
            except ValueError:
                continue
            if isinstance(rec, dict) and rec.get("id"):
                recs.append(rec)
    return recs


def parse_since(spec, now):
    spec = (spec or "7d").strip()
    m = re.fullmatch(r"(\d+)([dh])", spec)
    if m:
        n = int(m.group(1))
        delta = dt.timedelta(hours=n) if m.group(2) == "h" else dt.timedelta(days=n)
        return now - delta
    day = dt.date.fromisoformat(spec)
    return dt.datetime(day.year, day.month, day.day, tzinfo=dt.timezone.utc)


def check(bot, style_text, diffs, since, min_diffs=1):
    """Return a list of failure strings; empty means the style base passes."""
    fails = []
    fm, body = parse_frontmatter(style_text)
    if fm.get("bot") and fm["bot"] != bot:
        fails.append("frontmatter bot is %r, expected %r" % (fm["bot"], bot))
    updated = fm.get("updated", "")
    if not DATE_RE.match(updated):
        fails.append("frontmatter updated: is missing or not YYYY-MM-DD (got %r)" % updated)

    entries = parse_entries(body)
    if not entries:
        fails.append("no '## rule:' entry")
    known = {r["id"] for r in diffs}
    seen = {}
    for i, (rule, ev) in enumerate(entries, 1):
        if rule.startswith("\x00"):
            fails.append("heading %d is not a 'rule:' entry: %s" % (i, rule[1:]))
            continue
        if not ev:
            fails.append("rule %d has no evidence line: %s" % (i, rule[:60]))
        missing = [t for t in ev if t not in known]
        if missing:
            fails.append(
                "rule %d cites unknown diff id(s): %s" % (i, ", ".join(missing[:4]))
            )
        key = norm(rule)
        if key in seen:
            fails.append("rule %d duplicates rule %d: %s" % (i, seen[key], rule[:60]))
        else:
            seen[key] = i

    window = [r for r in diffs if in_window(r, since)]
    changed = [r for r in window if int(r.get("changed") or 0) > 0]
    if len(changed) >= min_diffs and DATE_RE.match(updated):
        last = max((ts_day(r) for r in changed), default="")
        if last and updated < last:
            fails.append(
                "style file is stale: updated %s but a diff landed on %s" % (updated, last)
            )
    return fails


def in_window(rec, since):
    day = ts_day(rec)
    if not day:
        return False
    d = dt.date.fromisoformat(day)
    return dt.datetime(d.year, d.month, d.day, tzinfo=dt.timezone.utc) >= since


def ts_day(rec):
    m = TS_RE.match(str(rec.get("ts", "")))
    return m.group(1) if m else ""


GOOD = """---
bot: henji
updated: 2026-11-02
---

# henji 的措辞库

## rule: 回信开头不写「感谢您的来信」
evidence: d1a2b3c4d5e6

为什么：读起来像自动回执。
改法：直接回应对方问的那件事。

## rule: 报价只说区间，不给单点
evidence: d0f1e2d3c4b5, d9a8b7c6d5e4

为什么：单点价格在谈判里先失一步。
"""
DIFFS = [
    {"id": "d1a2b3c4d5e6", "ts": "2026-11-01T03:00:00Z", "changed": 2},
    {"id": "d0f1e2d3c4b5", "ts": "2026-11-01T04:00:00Z", "changed": 1},
    {"id": "d9a8b7c6d5e4", "ts": "2026-11-02T01:00:00Z", "changed": 3},
]


def self_test():
    now = dt.datetime(2026, 11, 2, 12, tzinfo=dt.timezone.utc)
    since = parse_since("7d", now)
    cases = [
        ("good", GOOD, DIFFS, 1, []),
        (
            "no-rule",
            GOOD.replace("## rule: 回信开头不写「感谢您的来信」", "## 回信开头不写「感谢您的来信」"),
            DIFFS,
            1,
            ["not a 'rule:' entry"],
        ),
        (
            "no-evidence",
            GOOD.replace("evidence: d1a2b3c4d5e6\n", ""),
            DIFFS,
            1,
            ["no evidence line"],
        ),
        (
            "unknown-id",
            GOOD.replace("d1a2b3c4d5e6", "dfffffffffff"),
            DIFFS,
            1,
            ["unknown diff id"],
        ),
        (
            "duplicate-rule",
            GOOD.replace("报价只说区间，不给单点", "回信开头不写「感谢您的来信」"),
            DIFFS,
            1,
            ["duplicates rule"],
        ),
        (
            "stale",
            GOOD.replace("updated: 2026-11-02", "updated: 2026-10-20"),
            DIFFS,
            1,
            ["stale"],
        ),
        (
            "no-frontmatter",
            GOOD.split("---\n", 2)[2],
            DIFFS,
            1,
            ["updated: is missing"],
        ),
        ("empty", "", DIFFS, 1, ["no '## rule:' entry"]),
        # A window with no changed diff must not demand a rewrite: the loop is
        # idle, not stuck. The evidence still has to exist -- old is fine.
        (
            "idle-window",
            GOOD.replace("updated: 2026-11-02", "updated: 2026-09-01"),
            [dict(r, ts="2026-09-01T00:00:00Z") for r in DIFFS],
            1,
            [],
        ),
    ]
    bad = 0
    for name, text, diffs, min_d, expect in cases:
        got = check("henji", text, diffs, since, min_d)
        ok = all(any(e in f for f in got) for e in expect) if expect else not got
        print("%-14s %s %s" % (name, "PASS" if ok else "FAIL", "; ".join(got) or "clean"))
        if not ok:
            bad += 1
    # Bot mismatch must fire without touching the rest.
    got = check("cos", GOOD, DIFFS, since, 1)
    ok = any("frontmatter bot" in f for f in got)
    print("%-14s %s %s" % ("wrong-bot", "PASS" if ok else "FAIL", "; ".join(got)))
    if not ok:
        bad += 1
    total = len(cases) + 1
    print("self-test: %d/%d" % (total - bad, total))
    return 0 if bad == 0 else 1


def main(argv):
    try:
        sys.stdout.reconfigure(encoding="utf-8")
    except Exception:
        pass
    ap = argparse.ArgumentParser()
    ap.add_argument("--bot")
    ap.add_argument("--root", default=DEFAULT_ROOT)
    ap.add_argument("--style", default=None)
    ap.add_argument("--diffs", default=None)
    ap.add_argument("--since", default="7d", help="7d | 24h | 2026-10-01")
    ap.add_argument("--min-diffs", type=int, default=1)
    ap.add_argument("--self-test", action="store_true")
    a = ap.parse_args(argv)
    if a.self_test:
        return self_test()
    if not a.bot:
        print("usage: check_style_digest.py --bot <b> [--since 7d]", file=sys.stderr)
        return 2

    bot = re.sub(r"[^a-z0-9-]+", "-", a.bot.strip().lower()).strip("-") or "bot"
    style = a.style or os.path.join(a.root, "team", "style", bot + ".md")
    diffs_path = a.diffs or os.path.join(a.root, "team", "style", "diffs", bot + ".jsonl")
    if not os.path.isfile(style):
        print("style check failed: no such file: %s" % style)
        return 1
    with open(style, encoding="utf-8-sig", errors="replace") as f:
        text = f.read()
    if not text.strip():
        print("style check failed: file is empty")
        return 1

    diffs = load_diffs(diffs_path)
    try:
        since = parse_since(a.since, dt.datetime.now(dt.timezone.utc))
    except ValueError:
        print("style check failed: bad --since %r (use 7d / 24h / YYYY-MM-DD)" % a.since)
        return 2

    fails = check(bot, text, diffs, since, a.min_diffs)
    if fails:
        print("style check failed (%d):" % len(fails))
        for f_ in fails:
            print("  - " + f_)
        return 1
    fm, body = parse_frontmatter(text)
    entries = parse_entries(body)
    window = [r for r in diffs if in_window(r, since)]
    print(
        "style ok: bot=%s entries=%d evidence=%d diffs(total=%d window=%d) updated=%s"
        % (
            bot,
            len(entries),
            sum(len(ev) for _, ev in entries),
            len(diffs),
            len(window),
            fm.get("updated", "?"),
        )
    )
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
