#!/usr/bin/env python3
"""Gate for the routine scorecard (N9).

A scorecard that is never refused is decoration. This checks four things, all
of them facts about the snapshot rather than opinions about the office:

  1. the newest snapshot exists and was generated within --max-age-hours;
  2. the collector completed: `errors` is empty (a broker that was down when
     the card was taken is not a healthy office);
  3. `alerts` is empty -- every alert kind in routine_scorecard.py is already
     "something needs a human", so any entry fails;
  4. `observations` is non-empty (an empty sweep would otherwise pass all of
     the above vacuously).

`dormant` entries are not alerts: a task disabled 115 days ago is a fact, not
an incident. They live in the snapshot under totals.dormant and never fail the
gate.

    python check_routine_scorecard.py --data-dir ~/.wuphf/data/routines
    python check_routine_scorecard.py --self-test

Exit 0 pass, 1 fail, 2 usage.
"""

import argparse
import datetime as dt
import glob
import json
import os
import sys

DEFAULT_DATA_DIR = os.path.join(os.path.expanduser("~"), ".wuphf", "data", "routines")


def parse_ts(s):
    s = (s or "").strip().rstrip("Z")
    if not s:
        return None
    for fmt in ("%Y-%m-%dT%H:%M:%S", "%Y-%m-%dT%H:%M"):
        try:
            return dt.datetime.strptime(s, fmt)
        except ValueError:
            continue
    return None


def latest_snapshot(data_dir):
    files = sorted(glob.glob(os.path.join(data_dir, "scorecard-*.json")))
    return files[-1] if files else ""


def check(snap, now, max_age_hours=26, allow_partial=False):
    fails = []
    gen = parse_ts(snap.get("generated_at"))
    if gen is None:
        fails.append("generated_at is missing or unparseable")
    else:
        age = (now - gen).total_seconds() / 3600.0
        if age < -0.25:
            fails.append("generated_at is %0.1fh in the future" % -age)
        elif age > max_age_hours:
            fails.append("snapshot is %.1fh old (limit %dh)" % (age, max_age_hours))

    obs = snap.get("observations") or []
    if not obs:
        fails.append("no observations: an empty sweep proves nothing")

    if not allow_partial:
        for e in snap.get("errors") or []:
            fails.append("collector error: %s" % e)

    for kind, who, detail in snap.get("alerts") or []:
        fails.append("[%s] %s: %s" % (kind, who, detail))
    return fails


def self_test():
    import tempfile

    now = dt.datetime(2026, 11, 2, 12, 0)
    good = {
        "generated_at": "2026-11-02T11:30:00",
        "observations": [{"slug": "x", "source": "broker"}],
        "alerts": [],
        "errors": [],
        "totals": {"dormant": [("schtasks/old", "disabled, last ran 115 days ago")]},
    }
    cases = [
        ("good", good, {}, []),
        (
            "stale-snapshot",
            dict(good, generated_at="2026-11-01T00:00:00"),
            {},
            ["snapshot is"],
        ),
        ("future", dict(good, generated_at="2026-11-05T00:00:00"), {}, ["in the future"]),
        ("empty", dict(good, observations=[]), {}, ["no observations"]),
        ("missing-ts", dict(good, generated_at=""), {}, ["generated_at"]),
        (
            "has-alert",
            dict(good, alerts=[["overdue", "broker/x", "next_run in the past"]]),
            {},
            ["[overdue]"],
        ),
        (
            "collector-error",
            dict(good, errors=["broker: connection refused"]),
            {},
            ["collector error"],
        ),
        ("partial-allowed", dict(good, errors=["schtasks: not found"]), {"allow_partial": True}, []),
        # dormant is never a failure
        ("dormant-only", dict(good), {}, []),
    ]
    bad = 0
    for name, snap, kw, expect in cases:
        got = check(snap, now, **kw)
        ok = all(any(e in f for f in got) for e in expect) if expect else not got
        print("%-16s %s %s" % (name, "PASS" if ok else "FAIL", "; ".join(got) or "clean"))
        if not ok:
            bad += 1

    with tempfile.TemporaryDirectory() as tmp:
        d = tmp
        os.makedirs(d, exist_ok=True)
        with open(os.path.join(d, "scorecard-2026-11-01.json"), "w", encoding="utf-8") as f:
            json.dump(dict(good, generated_at="2026-11-01T09:00:00"), f)
        with open(os.path.join(d, "scorecard-2026-11-02.json"), "w", encoding="utf-8") as f:
            json.dump(good, f)
        newest = latest_snapshot(d)
        ok = os.path.basename(newest) == "scorecard-2026-11-02.json"
        print("%-16s %s %s" % ("latest-by-date", "PASS" if ok else "FAIL", os.path.basename(newest)))
        if not ok:
            bad += 1
        # an unreadable newest file must not silently fall back to an older one
        with open(newest, "w", encoding="utf-8") as f:
            f.write("{not json")
        try:
            json.load(open(newest, encoding="utf-8"))
            ok = False
        except ValueError:
            ok = True
        print("%-16s %s" % ("corrupt-detected", "PASS" if ok else "FAIL"))
        if not ok:
            bad += 1

    total = len(cases) + 2
    print("self-test: %d/%d" % (total - bad, total))
    return 0 if bad == 0 else 1


def main(argv):
    try:
        sys.stdout.reconfigure(encoding="utf-8")
    except Exception:
        pass
    ap = argparse.ArgumentParser()
    ap.add_argument("--data-dir", default=DEFAULT_DATA_DIR)
    ap.add_argument("--max-age-hours", type=float, default=26.0)
    ap.add_argument("--allow-partial", action="store_true")
    ap.add_argument("--self-test", action="store_true")
    a = ap.parse_args(argv)
    if a.self_test:
        return self_test()

    path = latest_snapshot(a.data_dir)
    if not path:
        print("routine check failed: no scorecard-*.json under %s" % a.data_dir)
        return 1
    try:
        with open(path, encoding="utf-8-sig") as f:
            snap = json.load(f)
    except (OSError, ValueError) as e:
        print("routine check failed: cannot read %s: %s" % (path, e))
        return 1

    now = dt.datetime.now(dt.timezone.utc).replace(tzinfo=None)
    fails = check(snap, now, a.max_age_hours, a.allow_partial)
    if fails:
        print("routine check failed (%d):" % len(fails))
        for f_ in fails:
            print("  - " + f_)
        return 1
    gen = snap.get("generated_at", "")
    print(
        "routine ok: %d jobs, 0 alerts, snapshot %sZ"
        % (len(snap.get("observations") or []), gen)
    )
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
