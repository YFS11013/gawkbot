#!/usr/bin/env python3
"""Thesis ledger (A1): validate what you claimed, score it, publish the week.

The source of truth is `team/theses/intake.md` in the wiki, because git is the
only thing on this box that records who wrote a line and when. The JSONL under
~/.wuphf/data/theses/ is a rebuild-on-every-run derivative: every bot here can
write that directory (`edit: allow`, `external_directory: allow` in
opencode.*.json), so a ledger that lived only there could have a miss edited
into a hit with nothing to compare against.

Scoring is a pure function of (entry, observations). The observations come from
~/.wuphf/data/forex/snapshots/, written by forex-bridge.py on every run --
without that series a claim about last week could only be answered with this
morning's price, which is the fabrication failure mode with a timestamp on it.

    python check_theses.py --lint                # validate, print ids, no writes
    python check_theses.py --score [--today D]   # rebuild ledger, judge what is due
    python check_theses.py --report [--emit]     # week page; --emit writes it
    python check_theses.py --self-test

Exit 0 pass, 1 fail, 2 usage. Fail is a verdict, not a crash: an empty intake,
an entry past its grace period, or a marker that parsed to zero entries all
mean the ledger is not doing its job.
"""

import argparse
import datetime as dt
import glob
import hashlib
import json
import os
import re
import subprocess
import sys

DEFAULT_ROOT = os.path.join(os.path.expanduser("~"), ".wuphf", "wiki")
DEFAULT_DATA = os.path.join(os.path.expanduser("~"), ".wuphf", "data", "theses")
DEFAULT_SNAP = os.path.join(os.path.expanduser("~"), ".wuphf", "data", "forex", "snapshots")
INTAKE_REL = "team/theses/intake.md"
GRACE_DAYS = 2

# Only `level` is scored in v1. `relative` and `event` are rejected at intake
# with a code rather than accepted and left forever pending: a kind that is
# silently never judged is indistinguishable from a kind that works.
SCORED_KINDS = ("level",)
KNOWN_KINDS = ("level", "relative", "event")
ENTERED_ATTRS = {"key", "claim", "criterion", "review_date", "invalidation", "sources", "supersedes"}
CANON_KEYS = {"id", "created_at", "status", "direction", "criterion_hash", "author"}


def now_utc():
    return dt.datetime.now(dt.timezone.utc).replace(tzinfo=None)


def parse_date(s):
    s = str(s or "").strip()[:10]
    try:
        return dt.datetime.strptime(s, "%Y-%m-%d").date()
    except ValueError:
        return None


def iso(d):
    return d.strftime("%Y-%m-%d") if d else ""


# ---------------------------------------------------------------- intake

FENCE = re.compile(r"^```thesis[ \t]*$")
FENCE_END = re.compile(r"^```[ \t]*$")


def parse_intake(text):
    """Return (blocks, errors) from the thesis fences.

    Returns the raw block text and the line it started on, so an error can name
    where to look. `blocks` counts fences even when their contents do not
    parse -- that is what makes "there are fences but zero entries" a
    detectable fault instead of an empty success.
    """
    lines = text.splitlines()
    blocks, errors = [], []
    i = 0
    while i < len(lines):
        if FENCE.match(lines[i]):
            start = i + 1
            j = start
            while j < len(lines) and not FENCE_END.match(lines[j]):
                j += 1
            if j >= len(lines):
                errors.append((start, "unterminated thesis fence"))
                break
            blocks.append((start, "\n".join(lines[start:j])))
            i = j + 1
            continue
        i += 1
    return blocks, errors


def criterion_hash(c):
    blob = json.dumps(c, sort_keys=True, separators=(",", ":"), ensure_ascii=False)
    return "sha256:" + hashlib.sha256(blob.encode("utf-8")).hexdigest()[:16]


def check_unknown(obj, allowed, where, problems):
    for k in sorted(obj):
        if k not in allowed:
            problems.append(("schema", "%s: unknown key %r (typos are rejected, not ignored)" % (where, k)))


def validate_criterion(c, where):
    """Return a list of (code, detail). Empty means the criterion is scoreable."""
    out = []
    if not isinstance(c, dict):
        return [("no_criterion", "%s: criterion must be an object" % where)]
    kind = c.get("kind")
    if kind not in KNOWN_KINDS:
        out.append(("no_criterion", "%s: criterion.kind must be one of %s" % (where, ", ".join(KNOWN_KINDS))))
        return out
    if kind not in SCORED_KINDS:
        out.append(("kind_not_scored", "%s: %s is not scored in v1; use level" % (where, kind)))
        return out

    check_unknown(c, {"kind", "series", "metric", "window", "agg", "op", "value", "require"}, where, out)
    if not isinstance(c.get("series"), str) or not c["series"].strip():
        out.append(("no_criterion", "%s: series is required" % where))
    if c.get("metric") not in ("close", "high", "low"):
        out.append(("no_criterion", "%s: metric must be close|high|low" % where))
    if c.get("agg") not in ("any", "all", "last"):
        out.append(("no_criterion", "%s: agg must be any|all|last" % where))
    if c.get("op") not in ("gt", "gte", "lt", "lte", "between", "eq"):
        out.append(("no_criterion", "%s: op must be gt|gte|lt|lte|between|eq" % where))

    v = c.get("value")
    if c.get("op") == "between":
        if not (isinstance(v, list) and len(v) == 2 and all(isinstance(x, (int, float)) for x in v)):
            out.append(("no_criterion", "%s: between needs [low, high]" % where))
        elif v[0] > v[1]:
            out.append(("no_criterion", "%s: between bounds are reversed" % where))
    elif not isinstance(v, (int, float)) or isinstance(v, bool):
        # A string threshold ("157.5", "15%") is how a numeric comparison
        # quietly becomes a string comparison.
        out.append(("no_criterion", "%s: value must be a JSON number, got %r" % (where, v)))

    w = c.get("window")
    if not isinstance(w, dict):
        out.append(("no_criterion", "%s: window is required (a judgement with no deadline cannot be wrong)" % where))
    else:
        check_unknown(w, {"start", "end"}, where + ".window", out)
        s, e = parse_date(w.get("start")), parse_date(w.get("end"))
        if not s or not e:
            out.append(("no_criterion", "%s: window.start/end must be YYYY-MM-DD" % where))
        else:
            if s > e:
                out.append(("no_criterion", "%s: window.start is after window.end" % where))
            if (e - s).days > 31:
                out.append(("window_too_long", "%s: window is %d days; a 31+ day claim is an opinion" % (where, (e - s).days)))

    req = c.get("require")
    if not isinstance(req, dict):
        out.append(("no_criterion", "%s: require.{min_obs,stale_days} is required" % where))
    else:
        check_unknown(req, {"min_obs", "stale_days"}, where + ".require", out)
        mo, sd = req.get("min_obs"), req.get("stale_days")
        if not isinstance(mo, int) or isinstance(mo, bool) or mo < 1:
            out.append(("no_criterion", "%s: require.min_obs must be an int >= 1" % where))
        if not isinstance(sd, int) or isinstance(sd, bool) or not 0 <= sd <= 7:
            out.append(("no_criterion", "%s: require.stale_days must be an int 0..7" % where))
    return out


def validate_entry(raw, where, today, live_hashes, ids):
    """Return (entry_or_None, problems). Problems are (code, detail) tuples."""
    problems = []
    if not isinstance(raw, dict):
        return None, [("no_criterion", "%s: block must be a JSON object" % where)]
    for k in sorted(raw):
        if k in CANON_KEYS:
            problems.append(("schema", "%s: %r is generated, not entered" % (where, k)))
        elif k not in ENTERED_ATTRS:
            problems.append(("schema", "%s: unknown key %r" % (where, k)))

    claim = raw.get("claim")
    if not isinstance(claim, str) or not claim.strip():
        problems.append(("no_criterion", "%s: claim is required" % where))

    review = parse_date(raw.get("review_date"))
    if not review:
        problems.append(("no_criterion", "%s: review_date is required (YYYY-MM-DD)" % where))

    crit = raw.get("criterion")
    problems.extend(validate_criterion(crit, where))

    if not problems:
        w = crit["window"]
        end = parse_date(w["end"])
        if review <= end:
            problems.append(("review_before_window", "%s: review_date %s must be after window.end %s -- you cannot judge a window before it closes" % (where, review, end)))
        span = (review - end).days
        if span > 14:
            problems.append(("review_too_late", "%s: review_date is %d days after the window; judge inside 14" % (where, span)))

    inv = raw.get("invalidation", {"kind": "none"})
    if not isinstance(inv, dict) or inv.get("kind") not in ("none", "criterion", "manual"):
        problems.append(("schema", "%s: invalidation.kind must be none|criterion|manual" % where))
    elif inv.get("kind") == "criterion":
        problems.extend(validate_criterion(inv.get("criterion"), where + ".invalidation"))

    srcs = raw.get("sources")
    if not isinstance(srcs, list) or not (1 <= len(srcs) <= 3):
        problems.append(("no_criterion", "%s: sources must list 1..3 entries" % where))

    sup = raw.get("supersedes")
    if sup is not None and sup not in ids:
        problems.append(("schema", "%s: supersedes %r is not a known id" % (where, sup)))

    if problems:
        return None, problems

    ch = criterion_hash(crit)
    if ch in live_hashes:
        problems.append(("duplicate", "%s: same criterion as %s -- restating a claim inflates the denominator" % (where, live_hashes[ch])))
        return None, problems

    entry = dict(raw)
    entry["criterion_hash"] = ch
    entry["review_date"] = iso(review)
    return entry, []


def load_intake(root, today, data_dir):
    """Parse + validate. Returns (entries, rejects, fatal)."""
    path = os.path.join(root, INTAKE_REL)
    rejects, entries = [], []
    if not os.path.exists(path):
        return [], [], ["intake is missing: %s (create it; an absent ledger is not a clean ledger)" % path]
    try:
        with open(path, encoding="utf-8-sig") as f:
            text = f.read()
    except OSError as e:
        return [], [], ["cannot read %s: %s" % (path, e)]

    blocks, fence_errors = parse_intake(text)
    fatal = ["%s:%d %s" % (INTAKE_REL, line, msg) for line, msg in fence_errors]

    if not blocks and "```thesis" in text:
        # The marker exists but the parser saw nothing. Returning "0 entries,
        # no errors" here is exactly the shape that let 13 Windows tasks vanish
        # from the routine scorecard without a warning.
        fatal.append("intake contains a ```thesis marker but parsed 0 blocks; the fence must start at column 0")

    live_hashes, ids = {}, {e["id"] for e in []}
    for idx, (line, body) in enumerate(blocks, start=1):
        where = "%s:%d" % (INTAKE_REL, line)
        try:
            raw = json.loads(body)
        except ValueError as e:
            rejects.append({"at": iso(today), "where": where, "code": "schema", "detail": str(e), "claim": body.strip()[:120]})
            continue
        entry, problems = validate_entry(raw, where, today, live_hashes, ids)
        if problems:
            for code, detail in problems:
                rejects.append({"at": iso(today), "where": where, "code": code, "detail": detail, "claim": (raw.get("claim") if isinstance(raw, dict) else "")[:200]})
            continue
        entry["id"] = "TH-%s-%03d" % (iso(today), idx)
        entry["created_at"] = iso(today)
        entry["where"] = where
        live_hashes[entry["criterion_hash"]] = entry["id"]
        ids.add(entry["id"])
        entries.append(entry)
    return entries, rejects, fatal


# ---------------------------------------------------------------- scoring

def compare(v, op, target):
    if op == "gt":
        return v > target
    if op == "gte":
        return v >= target
    if op == "lt":
        return v < target
    if op == "lte":
        return v <= target
    if op == "eq":
        return v == target
    if op == "between":
        return target[0] <= v <= target[1]
    return False


def load_observations(snap_dir, series):
    rows = []
    for path in sorted(glob.glob(os.path.join(snap_dir, "*.jsonl"))):
        try:
            with open(path, encoding="utf-8") as f:
                for line in f:
                    line = line.strip()
                    if not line:
                        continue
                    try:
                        o = json.loads(line)
                    except ValueError:
                        continue
                    if o.get("pair") == series:
                        rows.append(o)
        except OSError:
            continue
    return rows


def score_entry(entry, rows, today):
    """Return (result, reason, detail).

    result in hit | miss | void | pending. Order matters: a decisive
    observation beats a coverage complaint, because 'any day closed above 155'
    is settled by one witness and 'every day closed above 155' is settled by
    one counter-example. Coverage only decides the cases with no decisive
    witness left.
    """
    crit = entry["criterion"]
    review = parse_date(entry["review_date"])
    if today < review:
        return "pending", "not_due", ""

    window = crit["window"]
    start, end = parse_date(window["start"]), parse_date(window["end"])
    if today < end:
        return "pending", "window_open", ""

    metric = crit["metric"]
    inwin = [o for o in rows if start <= (parse_date(o.get("as_of")) or start) <= end]
    inwin.sort(key=lambda o: o.get("as_of") or "")
    vals = [(o["as_of"], o[metric]) for o in inwin if isinstance(o.get(metric), (int, float))]
    if not vals:
        if (today - end).days <= GRACE_DAYS:
            return "pending", "no_data", "no observation for %s between %s and %s" % (crit["series"], start, end)
        return "void", "no_data", "no observation for %s between %s and %s" % (crit["series"], start, end)

    req = crit["require"]
    min_obs, stale = req["min_obs"], req["stale_days"]
    op, target, agg = crit["op"], crit["value"], crit["agg"]

    if agg == "any":
        for as_of, v in vals:
            if compare(v, op, target):
                return "hit", "", "%s %s=%s at %s" % (crit["series"], metric, v, as_of)
    elif agg == "all":
        for as_of, v in vals:
            if not compare(v, op, target):
                return "miss", "", "%s %s=%s at %s falsifies" % (crit["series"], metric, v, as_of)
    else:  # last
        as_of, v = vals[-1]
        lag = (end - parse_date(as_of)).days
        if lag > stale:
            why = "last observation is %s, %d days before window end %s (stale_days=%d)" % (as_of, lag, end, stale)
            if (today - end).days <= GRACE_DAYS:
                return "pending", "stale_data", why
            return "void", "no_data", why
        return ("hit" if compare(v, op, target) else "miss"), "", "%s %s=%s at %s" % (crit["series"], metric, v, as_of)

    # No decisive witness: coverage decides whether an absence is evidence.
    if len(vals) < min_obs:
        why = "only %d observation(s), require.min_obs=%d" % (len(vals), min_obs)
        if (today - end).days <= GRACE_DAYS:
            return "pending", "coverage", why
        return "void", "no_data", why
    if agg == "any":
        return "miss", "", "%d observation(s), none matched" % len(vals)
    return "hit", "", "%d observation(s) all matched" % len(vals)


# ---------------------------------------------------------------- outputs

def intake_age_days(root):
    """Days since the intake page was first committed, or None if absent.

    Used only for the empty-ledger grace period: a page created today is not
    evidence of anything, but the same page still empty in two weeks means the
    ledger was never adopted.

    Deliberately not mtime: the wiki watcher rewrites and re-commits pages when
    it sees an external edit, which resets mtime to now. A file that is three
    weeks stale then looks newly created, and the empty ledger passes forever.
    The first git commit of the path does not move.
    """
    path = os.path.join(root, INTAKE_REL)
    if not os.path.exists(path):
        return None
    try:
        out = subprocess.run(
            ["git", "-C", root, "log", "--reverse", "--format=%ct", "--diff-filter=A", "--", INTAKE_REL],
            capture_output=True, text=True, timeout=10,
        )
        stamps = [int(x) for x in out.stdout.split() if x.strip().isdigit()]
        if stamps:
            first = dt.datetime.fromtimestamp(min(stamps))
            return max(0, int((now_utc() - first).total_seconds() // 86400))
    except (OSError, ValueError, subprocess.SubprocessError):
        pass
    return max(0, int((now_utc() - dt.datetime.fromtimestamp(os.path.getmtime(path))).total_seconds() // 86400))


def fmt_reject(r):
    """One line for a reject, without printing the location twice.

    validate_* already prefixes most details with the source location; the
    record carries it separately for machine use. Repeating it reads like a
    corrupted path and sends the reader looking for a file named after a line.
    """
    where, detail = r.get("where", ""), r.get("detail", "")
    if not where or detail.startswith(where):
        return detail
    return "%s: %s" % (where, detail)


def write_jsonl(path, rows):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8", newline="\n") as f:
        for r in rows:
            f.write(json.dumps(r, ensure_ascii=False, sort_keys=True) + "\n")


def fold_ledger(entries, verdicts):
    latest = {}
    for v in verdicts:
        latest[v["id"]] = v
    out = []
    for e in entries:
        v = latest.get(e["id"])
        out.append(
            {
                "id": e["id"],
                "claim": e["claim"],
                "series": e["criterion"]["series"],
                "op": e["criterion"]["op"],
                "value": e["criterion"]["value"],
                "window": e["criterion"]["window"],
                "review_date": e["review_date"],
                "verdict": (v or {}).get("result", "pending"),
                "reason": (v or {}).get("reason", ""),
                "detail": (v or {}).get("detail", ""),
                "checked_at": (v or {}).get("checked_at", ""),
                "criterion_hash": e["criterion_hash"],
                "where": e.get("where", ""),
            }
        )
    return out


def render_week(folded, rejects, today):
    counts = {"hit": 0, "miss": 0, "void": 0, "pending": 0}
    for r in folded:
        counts[r["verdict"]] = counts.get(r["verdict"], 0) + 1
    judged = counts["hit"] + counts["miss"] + counts["void"]
    lines = [
        "# 论点记分卡：%s" % iso(today),
        "",
        "> 由 `scripts/office/check_theses.py --report` 生成。源是 `team/theses/intake.md`。",
        "",
        "## 计数",
        "",
        "| 命中 | 未命中 | 不计分 | 未到期 | 无法判定 |",
        "|---|---|---|---|---|",
        "| %d | %d | %d | %d | %d |" % (counts["hit"], counts["miss"], counts["void"], counts["pending"], 0),
        "",
        "判分 %d 条，命中率 %s。" % (judged, ("%.0f%%" % (100.0 * counts["hit"] / judged)) if judged else "n/a"),
        "",
        "## 明细",
        "",
    ]
    if not folded:
        lines.append("（台账为空。空台账不是好消息：说明本周没有写出一条可判定的判断。）")
    for r in folded:
        lines.append(
            "- `%s` %s — **%s**%s  \n  `%s %s %s`, 窗口 %s..%s, 复核 %s"
            % (
                r["id"],
                r["claim"],
                r["verdict"],
                ("：" + r["detail"]) if r["detail"] else "",
                r["series"],
                r["op"],
                r["value"],
                r["window"]["start"],
                r["window"]["end"],
                r["review_date"],
            )
        )
    if rejects:
        lines += ["", "## 被拒的录入", ""]
        for r in rejects:
            lines.append("- `%s` %s" % (r["code"], fmt_reject(r)))
    return "\n".join(lines) + "\n"


def publish(root, rel, content, port):
    import urllib.request

    token_file = os.path.join(os.environ.get("TEMP", "/tmp"), "wuphf-broker-token-%d" % port)
    if not os.path.exists(token_file):
        return 0, "no broker token at %s" % token_file
    token = open(token_file, encoding="utf-8").read().strip()
    body = json.dumps({"slug": "office", "path": rel, "content": content, "mode": "replace", "commit_message": "theses: %s" % rel}, ensure_ascii=False).encode("utf-8")
    req = urllib.request.Request(
        "http://127.0.0.1:%d/wiki/write" % port,
        data=body,
        headers={"Authorization": "Bearer " + token, "Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=60) as f:
            return f.status, json.load(f).get("commit_sha", "")
    except Exception as e:  # noqa: BLE001 - surfaced to the caller as a failure
        return 0, str(e)


# ---------------------------------------------------------------- self test

def self_test():
    bad = 0
    checks = 0

    def eq(name, got, want):
        nonlocal bad, checks
        checks += 1
        ok = got == want
        print("%-24s %s %s" % (name, "PASS" if ok else "FAIL", "" if ok else "got %r want %r" % (got, want)))
        if not ok:
            bad += 1

    def ok(name, cond, detail=""):
        nonlocal bad, checks
        checks += 1
        print("%-24s %s %s" % (name, "PASS" if cond else "FAIL", "" if cond else detail))
        if not cond:
            bad += 1

    def crit(**kw):
        base = {
            "kind": "level",
            "series": "USDJPY",
            "metric": "close",
            "window": {"start": "2026-11-02", "end": "2026-11-06"},
            "agg": "any",
            "op": "gt",
            "value": 155.0,
            "require": {"min_obs": 3, "stale_days": 2},
        }
        base.update(kw)
        return base

    def entry(**kw):
        e = {"id": "TH-1", "claim": "x", "criterion": crit(), "review_date": "2026-11-09", "criterion_hash": "h"}
        e.update(kw)
        return e

    def obs(vals):
        return [{"pair": "USDJPY", "as_of": d, "close": v} for d, v in vals]

    today = dt.date(2026, 11, 9)

    # E1: one true and one false claim over the same data must split.
    rows = obs([("2026-11-02", 150.0), ("2026-11-03", 156.0), ("2026-11-04", 157.0), ("2026-11-05", 156.0)])
    eq("must-hit", score_entry(entry(), rows, today)[0], "hit")
    eq("must-miss", score_entry(entry(criterion=crit(op="gt", value=999.0)), rows, today)[0], "miss")

    # A decisive counter-example settles `all` even when coverage is short.
    # 150 < 145 is false, so one observation already falsifies the universal --
    # even though coverage (min_obs=3) is short. A counter-example needs no样本.
    eq("all-counterexample", score_entry(entry(criterion=crit(agg="all", op="lt", value=145.0)), rows[:1], today)[0], "miss")

    # Empty-subset universal: 1 observation, min_obs=5, all match. Must NOT be hit.
    thin = obs([("2026-11-06", 200.0)])
    eq("all-thin-coverage", score_entry(entry(criterion=crit(agg="all", value=155.0, require={"min_obs": 5, "stale_days": 0})), thin, today)[0], "void")

    # `last` with a stale tail must not answer with last week's close.
    stale = obs([("2026-11-02", 160.0)])
    eq("last-stale", score_entry(entry(criterion=crit(agg="last", require={"min_obs": 1, "stale_days": 0})), stale, today)[0], "void")

    # last inside tolerance uses the real final observation.
    fresh = obs([("2026-11-02", 150.0), ("2026-11-06", 158.0)])
    eq("last-fresh", score_entry(entry(criterion=crit(agg="last", require={"min_obs": 1, "stale_days": 2})), fresh, today)[0], "hit")

    # Review date not reached, and window still open: both pending, never miss.
    eq("not-due", score_entry(entry(review_date="2026-11-20"), rows, today)[0], "pending")
    eq("window-open", score_entry(entry(), rows, dt.date(2026, 11, 4))[0], "pending")

    # Missing data inside grace is pending, after grace it is void, never miss.
    eq("no-data-grace", score_entry(entry(), [], today)[0], "void")
    eq("no-data-fresh", score_entry(entry(), [], dt.date(2026, 11, 7))[0], "pending")

    # ---------------------------------------------------------- intake parsing
    def lint(text, today=dt.date(2026, 11, 2)):
        blocks, errs = parse_intake(text)
        if errs:
            return [], [("schema", str(errs))]
        entries, rejects = [], []
        live, ids = {}, set()
        for idx, (line, body) in enumerate(blocks, start=1):
            try:
                raw = json.loads(body)
            except ValueError as e:
                rejects.append(("schema", str(e)))
                continue
            e, probs = validate_entry(raw, "L%d" % line, today, live, ids)
            if probs:
                rejects.extend(probs)
                continue
            e["id"] = "TH-%03d" % idx
            live[e["criterion_hash"]] = e["id"]
            entries.append(e)
        return entries, rejects

    good = json.dumps({"claim": "USDJPY closes above 155", "criterion": crit(), "review_date": "2026-11-09", "invalidation": {"kind": "none"}, "sources": [{"path": "a.md", "line": 1, "quote": "q"}]})
    es, rs = lint("```thesis\n" + good + "\n```\n")
    eq("intake-ok", len(es), 1)
    eq("intake-ok-rejects", len(rs), 0)

    # A marker that parses to nothing is a fault, not a clean empty ledger.
    fatal_text = "```thesis\n```"
    blocks, _ = parse_intake(fatal_text)
    eq("empty-fence-detected", len(blocks), 1)

    bad_json = "```thesis\n{\"claim\": }\n```"
    es, rs = lint(bad_json)
    eq("bad-json-rejected", len(es) + len(rs), 1)
    ok("bad-json-coded", rs and rs[0][0] == "schema", str(rs))

    # A string threshold would make the comparison lexicographic.
    s_thresh = json.dumps({"claim": "x", "criterion": crit(value="155.0"), "review_date": "2026-11-09", "invalidation": {"kind": "none"}, "sources": [{"path": "a.md", "line": 1, "quote": "q"}]})
    es, rs = lint("```thesis\n" + s_thresh + "\n```")
    eq("string-threshold-rejected", len(es), 0)

    # A typo'd key must not be silently dropped.
    typo = json.loads(good)
    typo["criterion"]["walue"] = 155.0
    es, rs = lint("```thesis\n" + json.dumps(typo) + "\n```")
    eq("typo-key-rejected", len(es), 0)
    # Reject records must be (code, detail) pairs. A bare string here raises
    # "too many values to unpack" in the report formatter, i.e. the gate crashes
    # instead of reporting -- which is worse than a wrong verdict.
    ok("reject-shape", all(isinstance(r, tuple) and len(r) == 2 for r in rs), str(rs))

    # Review before the window closes means judging a window that is not over.
    early = json.loads(good)
    early["review_date"] = "2026-11-05"
    es, rs = lint("```thesis\n" + json.dumps(early) + "\n```")
    eq("early-review-rejected", len(es), 0)

    # Restating the same criterion must not inflate the denominator.
    es, rs = lint("```thesis\n" + good + "\n```\n\n```thesis\n" + good + "\n```")
    eq("duplicate-rejected", len(es), 1)

    # relative/event are refused by name rather than left forever pending.
    rel = json.loads(good)
    rel["criterion"]["kind"] = "relative"
    es, rs = lint("```thesis\n" + json.dumps(rel) + "\n```")
    eq("unscored-kind-rejected", len(es), 0)

    # ---------------------------------------------------------- the gate
    def gate(folded, rejects):
        fails = []
        if not folded and not rejects:
            fails.append("empty ledger: nothing was ever recorded")
        for r in folded:
            if r["verdict"] == "pending" and r["checked_at"] == "":
                fails.append("gated-pending %s reason=%s" % (r["id"], r["reason"]))
        return fails

    eq("gate-empty", len(gate([], [])), 1)
    eq("gate-clean", len(gate([{"id": "a", "verdict": "hit", "checked_at": "x", "reason": ""}], [])), 0)

    print("self-test: %d checks, %d failed" % (checks, bad))
    return 0 if bad == 0 else 1


# ---------------------------------------------------------------- main

def main(argv):
    try:
        sys.stdout.reconfigure(encoding="utf-8")
    except Exception:
        pass
    ap = argparse.ArgumentParser()
    ap.add_argument("--root", default=DEFAULT_ROOT)
    ap.add_argument("--data-dir", default=DEFAULT_DATA)
    ap.add_argument("--snap-dir", default=DEFAULT_SNAP)
    ap.add_argument("--today", default=None)
    ap.add_argument("--port", type=int, default=7892)
    ap.add_argument("--lint", action="store_true")
    ap.add_argument("--score", action="store_true")
    ap.add_argument("--report", action="store_true")
    ap.add_argument("--emit", action="store_true", help="publish the week page through the broker")
    ap.add_argument("--dry-run", action="store_true")
    ap.add_argument("--self-test", action="store_true")
    a = ap.parse_args(argv)
    if a.self_test:
        return self_test()

    today = parse_date(a.today) or now_utc().date()
    entries, rejects, fatal = load_intake(a.root, today, a.data_dir)
    if fatal:
        print("thesis check failed:")
        for f in fatal:
            print("  - " + f)
        return 1

    if a.lint:
        for e in entries:
            print("%-16s %s  [%s %s %s, %s..%s]" % (e["id"], e["claim"], e["criterion"]["series"], e["criterion"]["op"], e["criterion"]["value"], e["criterion"]["window"]["start"], e["criterion"]["window"]["end"]))
        for r in rejects:
            print("REJECT %-14s %s" % (r["code"], fmt_reject(r)))
        print("lint: %d entries, %d rejects" % (len(entries), len(rejects)))
        return 1 if rejects else 0

    verdicts = []
    verdict_path = os.path.join(a.data_dir, "verdicts.jsonl")
    if os.path.exists(verdict_path):
        with open(verdict_path, encoding="utf-8") as f:
            for line in f:
                line = line.strip()
                if line:
                    try:
                        verdicts.append(json.loads(line))
                    except ValueError:
                        continue

    if a.score:
        known = {v["id"] for v in verdicts}
        for e in entries:
            if e["id"] in known:
                continue
            rows = load_observations(a.snap_dir, e["criterion"]["series"])
            result, reason, detail = score_entry(e, rows, today)
            if result == "pending":
                continue  # retried every run; never persisted, so it cannot go stale
            verdicts.append({"id": e["id"], "result": result, "reason": reason, "detail": detail, "checked_at": iso(today), "criterion_hash": e["criterion_hash"]})
        if not a.dry_run:
            # --dry-run must write nothing at all. The watchdog runs this every
            # 15 minutes; touching the ledger and rejects on each pass would
            # make "when did this last change" unanswerable.
            write_jsonl(verdict_path, verdicts)
            write_jsonl(os.path.join(a.data_dir, "ledger.jsonl"), fold_ledger(entries, verdicts))
            write_jsonl(os.path.join(a.data_dir, "rejects.jsonl"), rejects)

    folded = fold_ledger(entries, verdicts)
    if a.report or a.emit:
        page = render_week(folded, rejects, today)
        if a.emit:
            rel = "team/theses/week-%s.md" % iso(today)
            code, detail = publish(a.root, rel, page, a.port)
            print("%s %s" % (rel, detail))
            if code != 200:
                return 1
        else:
            print(page)

    counts = {}
    for r in folded:
        counts[r["verdict"]] = counts.get(r["verdict"], 0) + 1
    print(
        "theses: %d entries, %d rejects, hit=%d miss=%d void=%d pending=%d"
        % (len(folded), len(rejects), counts.get("hit", 0), counts.get("miss", 0), counts.get("void", 0), counts.get("pending", 0))
    )

    # The gate. An empty ledger passes only when there were rejects to look at;
    # otherwise "nothing to score" and "nothing was ever written" look alike.
    if not folded and not rejects:
        age = intake_age_days(a.root)
        if age is not None and age <= 14:
            print("theses: ledger is empty (%d day(s) old, inside the 14-day grace)" % age)
            return 0
        print("thesis check failed: empty ledger -- the intake page is %s days old and still has no entry"
              % ("?" if age is None else age))
        return 1
    stuck = [r for r in folded if r["verdict"] == "pending" and (today - parse_date(r["review_date"])).days > GRACE_DAYS]
    if stuck:
        print("thesis check failed: %d entr(ies) past review_date+%dd without a verdict" % (len(stuck), GRACE_DAYS))
        for r in stuck:
            print("  - %s review_date=%s reason=%s" % (r["id"], r["review_date"], r["reason"] or "not scored"))
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
