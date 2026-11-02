#!/usr/bin/env python3
"""Routine scorecard: what is supposed to run, and what actually ran.

Every account of a long-running agent setup that includes a failure includes
the same failure: it kept looking busy while it was broken. One reported
"analytics cron failed silently for three days"; another listed "12 items
waiting on Robert" next to its wins; a third reported 28/29 published runs and
counted the one timeout. None of them had a number for it until afterwards.

This script produces that number. It reads two sources and writes one page:

  broker  -- GET /scheduler (job list) + GET /scheduler/{slug}/runs (ring
             buffer, 20 entries per slug) + GET /tasks (routine-originated
             tasks, for the pending queue)
  Windows -- `schtasks /query /fo CSV /v`, which is the only durable record
             for jobs that are actually driven by Task Scheduler rather than
             by the broker loop (see RUNTIME-API.md §5)

Outputs, all under the wiki root:
  team/routines/scorecard-<YYYY-MM-DD>.json   full snapshot (machine)
  team/routines/scorecard-<YYYY-MM-DD>.md     one page (human)
  team/routines/history.jsonl                one line per observation, appended

The JSONL append is the point. The broker's per-slug run history is a 20-entry
ring buffer, so it cannot answer "did this run all week". The snapshot can, as
long as something takes one regularly.

Exit codes: 0 wrote, 1 broker unreachable when it was required, 2 usage.
"""

import argparse
import csv
import datetime as dt
import io
import json
import os
import re
import shutil
import subprocess
import sys
import urllib.error
import urllib.request

SCHEMA = 1
DEFAULT_PORT = 7892
DEFAULT_MATCH = r"wuphf|gawkbot|forex"
# Windows Task Scheduler result codes that are not failures.
WIN_NEVER_RUN = 267011  # 0x41303 "task has not yet run"
WIN_RUNNING = 267009  # 0x41301 "task is currently running"
WIN_OK = 0
TASK_CLOSED = {"done", "failed", "cancelled", "canceled", "archived", "completed"}
# `routine:<slug>:<next_run>`. The timestamp itself contains colons, and a
# slug may too (`one-workflow:cos__human:…`), so a rsplit on ':' cuts the
# timestamp instead of the separator. Anchor on the timestamp shape instead.
ROUTINE_SID_RE = re.compile(r"^routine:(.+):(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z?)$")


def default_root():
    return os.path.join(os.path.expanduser("~"), ".wuphf", "wiki")


def default_data_dir():
    return os.path.join(os.path.expanduser("~"), ".wuphf", "data", "routines")


def now_utc():
    """Every comparison in this file is UTC-naive.

    Mixing the two sources without this is how a 6-hour job looks 8 hours
    late on a UTC+8 box: broker timestamps carry Z, Windows ones are local.
    """
    return dt.datetime.now(dt.timezone.utc).replace(tzinfo=None)


def local_offset():
    return dt.datetime.now() - now_utc()


def now_local():
    return dt.datetime.now()


def iso(d):
    return d.strftime("%Y-%m-%dT%H:%M:%S") if d else ""


def token():
    t = os.environ.get("WUPHF_TOKEN", "").strip()
    if t:
        return t
    path = os.path.join(
        os.environ.get("TEMP", "/tmp"), "wuphf-broker-token-%d" % DEFAULT_PORT
    )
    with open(path, encoding="utf-8") as f:
        return f.read().strip()


def get_json(url, tok, timeout=20):
    req = urllib.request.Request(url, headers={"Authorization": "Bearer " + tok})
    with urllib.request.urlopen(req, timeout=timeout) as f:
        return json.load(f)


def fetch_broker(port, tok):
    base = "http://127.0.0.1:%d" % port
    payload = get_json(base + "/scheduler", tok)
    jobs = payload.get("jobs", payload if isinstance(payload, list) else [])
    runs = {}
    for j in jobs:
        slug = j.get("slug", "")
        if not slug:
            continue
        try:
            r = get_json(base + "/scheduler/%s/runs" % slug, tok)
            runs[slug] = r.get("runs", r if isinstance(r, list) else [])
        except (urllib.error.URLError, ValueError, OSError):
            runs[slug] = []
    tasks = []
    try:
        t = get_json(base + "/tasks?all_channels=true&include_done=true", tok)
        tasks = t.get("tasks", t if isinstance(t, list) else [])
    except (urllib.error.URLError, ValueError, OSError):
        tasks = []
    return jobs, runs, tasks


def pending_by_slug(tasks):
    """Routine-originated tasks that are not closed, grouped by job slug.

    The dedup key written by broker_routine_tasks.go is
    `routine:<slug>:<next_run>`; splitting on ':' keeps a slug that itself
    contains ':' intact up to the last field.
    """
    out = {}
    for t in tasks:
        sid = str(t.get("source_signal_id") or "")
        if not sid.startswith("routine:"):
            continue
        rest = sid[len("routine:") :]
        m = ROUTINE_SID_RE.match(sid)
        slug = m.group(1) if m else rest
        if str(t.get("status") or "").lower() in TASK_CLOSED:
            continue
        out.setdefault(slug, []).append(t)
    return out


def run_status(runs):
    ok = failed = other = 0
    last_ok = None
    for r in runs:
        st = str(r.get("status") or "").lower()
        when = r.get("finished_at") or r.get("started_at") or ""
        if st in ("ok", "success", "succeeded", "done"):
            ok += 1
            if when and (last_ok is None or when > last_ok):
                last_ok = when
        elif st in ("failed", "error", "fail"):
            failed += 1
        else:
            other += 1
    return ok, failed, other, last_ok


def broker_obs(job, runs, pend, now):
    slug = job.get("slug", "")
    ok, failed, other, last_ok = run_status(runs)
    interval = int(job.get("interval_minutes") or 0)
    last_run = job.get("last_run") or ""
    pend_list = pend.get(slug, [])
    oldest = min((t.get("created_at") or "" for t in pend_list), default="")
    return {
        "source": "broker",
        "slug": slug,
        "label": job.get("label", ""),
        "enabled": bool(job.get("enabled")),
        "state": job.get("status", ""),
        "interval_minutes": interval,
        "last_run": last_run,
        "last_result": job.get("last_run_status", ""),
        "next_run": job.get("next_run", ""),
        "runs_seen": len(runs),
        "ok_seen": ok,
        "failed_seen": failed,
        "other_seen": other,
        "last_success_at": last_ok or "",
        "pending": len(pend_list),
        "oldest_pending_at": oldest,
    }


def parse_win_time(s):
    s = (s or "").strip()
    if not s or s.upper() == "N/A":
        return None
    for fmt in ("%Y/%m/%d %H:%M:%S", "%Y-%m-%d %H:%M:%S", "%Y/%m/%d %H:%M"):
        try:
            return dt.datetime.strptime(s, fmt)
        except ValueError:
            continue
    return None


def win_result(s):
    """Return 'ok' | 'never_run' | 'running' | 'fail' | 'unknown'."""
    s = (s or "").strip()
    if not s:
        return "unknown"
    try:
        n = int(s)
    except ValueError:
        return "unknown"
    if n == WIN_OK:
        return "ok"
    if n == WIN_NEVER_RUN:
        return "never_run"
    if n == WIN_RUNNING:
        return "running"
    return "fail"


def parse_repeat_minutes(s):
    """`重复: 每` cell -> minutes.

    The CSV header is localised (zh-CN prints 主机名/任务名/…), so columns are
    read by position. The repeat cell is the reliable period: it is what the
    schedule says, unlike NextRun-LastRun which is one sample of it.
    Formats seen: "0 小时， 15 分钟", "1 天", "0 hours, 15 minutes".
    """
    s = (s or "").strip()
    if not s or s.upper() == "N/A":
        return 0
    total = 0
    for num, unit in re.findall(r"(\d+)\s*([^\d,\s，、]+)", s):
        n = int(num)
        u = unit.lower()
        if u.startswith("天") or u.startswith("day"):
            total += n * 1440
        elif u.startswith("小时") or u.startswith("hour"):
            total += n * 60
        elif u.startswith("分钟") or u.startswith("minute") or u.startswith("min"):
            total += n
    return total


# Column positions in `schtasks /query /fo CSV /v`. The header row is
# translated by the OS locale, so matching on names silently yields zero rows
# on a zh-CN box -- which is exactly how the first run reported "23 jobs" with
# no Windows tasks and no error.
WIN_COL = {
    "TaskName": 1,
    "Next Run Time": 2,
    "Status": 3,
    "Last Run Time": 5,
    "Last Result": 6,
    "Scheduled Task State": 11,
    "Repeat": 24,
}


def parse_schtasks_csv(text):
    rows = list(csv.reader(io.StringIO(text)))
    if not rows:
        return []
    width = max(WIN_COL.values()) + 1
    out = []
    for r in rows[1:]:
        if len(r) < width:
            continue
        out.append({key: r[idx] for key, idx in WIN_COL.items()})
    return out


def collect_schtasks(match):
    """Return raw rows for tasks whose name matches, or [] off Windows."""
    if not shutil.which("schtasks"):
        return []
    # /v is verbose: it carries Last Result and Next Run Time, which the
    # non-verbose listing omits. The console code page is cp936 on a zh-CN
    # box, so decode by hand rather than trusting locale.
    p = subprocess.run(
        ["schtasks", "/query", "/fo", "CSV", "/v"],
        capture_output=True,
        timeout=120,
    )
    if p.returncode != 0:
        raise OSError("schtasks exited %d: %s" % (p.returncode, p.stderr[:200]))
    text = p.stdout.decode("gbk", "replace")
    rx = re.compile(match, re.IGNORECASE)
    return [r for r in parse_schtasks_csv(text) if rx.search(r.get("TaskName", ""))]


def win_obs(row, now, offset=None):
    name = row.get("TaskName", "").lstrip("\\")
    last = parse_win_time(row.get("Last Run Time"))
    nxt = parse_win_time(row.get("Next Run Time"))
    res = win_result(row.get("Last Result"))
    state = (row.get("Status") or "").strip()
    task_state = (row.get("Scheduled Task State") or "").strip()
    enabled = not ("disabled" in task_state.lower() or "禁" in task_state)
    if offset is None:
        offset = local_offset()
    # A task that has never run reports Last Run Time = 1999/11/30, which is a
    # placeholder, not a date. Keeping it would compute a 14-million-minute
    # period from NextRun-LastRun and, on the page, claim a task last ran in
    # 1999. Result 267011 is the authoritative signal; the year check catches
    # the case where the result was already lost.
    if res == "never_run" or (last and last.year <= 2000):
        last = None
    last_utc = last - offset if last else None
    last_run = iso(last_utc)
    # Period, in order of trustworthiness: the schedule's own repeat cell,
    # then the observed NextRun-LastRun gap, then nothing (no rule, rather
    # than a guessed one -- a one-shot task has no Next Run at all).
    interval = parse_repeat_minutes(row.get("Repeat"))
    if interval <= 0 and last and nxt and nxt > last:
        interval = int(round((nxt - last).total_seconds() / 60.0))
    return {
        "source": "schtasks",
        "slug": row.get("TaskName", ""),
        "label": name,
        "enabled": enabled,
        "state": state,
        "interval_minutes": interval,
        "last_run": last_run,
        "last_result": res,
        "next_run": iso(nxt - offset if nxt else None),
        "runs_seen": 0,
        "ok_seen": 0,
        "failed_seen": 0,
        "other_seen": 0,
        "last_success_at": last_run if res == "ok" else "",
        "pending": 0,
        "oldest_pending_at": "",
        "raw_last_result": (row.get("Last Result") or "").strip(),
    }


def parse_any_time(s):
    """Broker timestamps: UTC, `YYYY-MM-DDTHH:MM:SS[Z]`."""
    s = (s or "").strip().rstrip("Z")
    if not s:
        return None
    for fmt in ("%Y-%m-%dT%H:%M:%S", "%Y/%m/%d %H:%M:%S"):
        try:
            return dt.datetime.strptime(s, fmt)
        except ValueError:
            continue
    return None


def scheduling_alert(o, now, grace_min=15, dormant_days=7):
    """Return (kind, reason) or ('', '').

    Three separate facts, deliberately not merged:

      overdue  -- next_run is in the past: the scheduler itself stopped
                  advancing this job. Does not need a period to detect.
      stale    -- the job ran, but not within 2x its interval. Only applies
                  when the interval is known (broker system crons, or the
                  Windows Last Run -> Next Run delta).
      dormant  -- disabled on purpose or by accident; report when its last
                  run is old enough that "on purpose" deserves a second look.

    A job with no last_run and no runs is `data missing`, not `broken`: the
    broker keeps run history in memory, so a restart erases it. Reporting
    that as silence was the first version's only false alarm.
    """
    if not o.get("enabled"):
        base = o.get("last_run") or ""
        when = parse_any_time(base)
        if when is not None and (now - when).days >= dormant_days:
            return "dormant", "disabled, last ran %d days ago (%s)" % (
                (now - when).days,
                base[:16],
            )
        return "", ""

    nxt = parse_any_time(o.get("next_run") or "")
    if nxt is not None and nxt < now - dt.timedelta(minutes=grace_min):
        return "overdue", "next_run %s is in the past" % (o.get("next_run") or "")[:16]

    interval = int(o.get("interval_minutes") or 0)
    if interval <= 0:
        return "", ""
    base = o.get("last_run") or o.get("last_success_at") or ""
    when = parse_any_time(base)
    if when is None:
        return "", ""
    age = (now - when).total_seconds() / 60.0
    if age > 2 * interval:
        return "stale", "silent for %dm (interval %dm)" % (int(age), interval)
    return "", ""


def stale_reason(o, now):
    """Back-compat wrapper used by the self-test fixtures."""
    return scheduling_alert(o, now)[1]


def analyse(observations, now, max_pending=10):
    """Return (alerts, totals). Alerts are (kind, who, detail) triples.

    `dormant` is deliberately NOT an alert. A disabled task that last ran 115
    days ago is a fact worth showing, but calling it an alert means every
    sweep on this machine starts with seven lines nobody intends to act on,
    and the two real ones stop being read.
    """
    alerts = []
    dormant = []
    pending_total = 0
    for o in observations:
        who = "%s/%s" % (o.get("source"), o.get("label") or o.get("slug"))
        if o.get("source") == "schtasks" and o.get("enabled"):
            if o.get("last_result") == "fail":
                alerts.append(("failed", who, "Last Result %s" % o.get("raw_last_result", "")))
            elif o.get("last_result") == "unknown":
                alerts.append(("unknown", who, "no Last Result reported"))
        kind, reason = scheduling_alert(o, now)
        if kind == "dormant":
            dormant.append((who, reason))
        elif kind:
            alerts.append((kind, who, reason))
        pending_total += int(o.get("pending") or 0)
    # Per-job pending is in the table on purpose: a queue of one is work, not
    # an incident. Only an oversized queue is an alert, because that is when
    # the human starts skimming and the approval gate turns into a rubber stamp.
    if pending_total > max_pending:
        alerts.append(
            ("queue", "all routines", "%d pending > --max-pending %d" % (pending_total, max_pending))
        )
    return alerts, {"pending_total": pending_total, "jobs": len(observations), "dormant": dormant}


def render_md(snap):
    lines = [
        "# Routine 记分卡：%s" % snap["generated_at"][:10],
        "",
        "> 生成：`python scripts/office/routine_scorecard.py`。判据：`check_routine_scorecard.py`。",
        "> 生成时间 %s（UTC）；broker 端口 %s；任务匹配 `%s`。"
        % (snap["generated_at"], snap.get("port", "-"), snap.get("match", "-")),
        "",
        "## 结论",
        "",
    ]
    alerts = snap["alerts"]
    if alerts:
        lines.append("**%d 条告警。**" % len(alerts))
        for kind, who, detail in alerts:
            lines.append("- `%s` %s —— %s" % (kind, who, detail))
    else:
        lines.append("无告警：启用的任务都在周期内跑过，且没有失败码。")
    dormant = snap["totals"].get("dormant") or []
    if dormant:
        lines += ["", "## 已停用（不计入告警）", ""]
        for who, reason in dormant:
            lines.append("- %s —— %s" % (who, reason))
    lines += ["", "## broker routines", ""]
    b = [o for o in snap["observations"] if o["source"] == "broker"]
    if b:
        lines.append("| job | enabled | 周期 | last_run | next_run | runs | failed | pending |")
        lines.append("|---|---|---|---|---|---|---|---|")
        for o in b:
            last = o["last_run"] or ("*no data*" if o["runs_seen"] == 0 else "-")
            lines.append(
                "| `%s` | %s | %s | %s | %s | %d | %d | %d |"
                % (
                    o["slug"],
                    "yes" if o["enabled"] else "**no**",
                    ("%dm" % o["interval_minutes"]) if o["interval_minutes"] else "-",
                    last,
                    o["next_run"] or "-",
                    o["runs_seen"],
                    o["failed_seen"],
                    o["pending"],
                )
            )
        lines.append("")
        lines.append(
            "`*no data*` = broker 的 `last_run` 与 run 环形缓冲都是空的（重启会清空）；"
            "这类 job 本卡只记事实，不判定为失败。"
        )
    else:
        lines.append("未取到（broker 不可达或 `--source schtasks`）。")
    lines += ["", "## Windows 计划任务", ""]
    w = [o for o in snap["observations"] if o["source"] == "schtasks"]
    if w:
        lines.append("| 任务 | 状态 | last_run | Last Result | next_run | 推算周期 |")
        lines.append("|---|---|---|---|---|---|")
        for o in w:
            lines.append(
                "| `%s` | %s | %s | %s | %s | %s |"
                % (
                    o["label"],
                    o["state"] or ("enabled" if o["enabled"] else "disabled"),
                    o["last_run"] or "-",
                    o.get("last_result", "-"),
                    o["next_run"] or "-",
                    ("%dm" % o["interval_minutes"]) if o["interval_minutes"] else "-",
                )
            )
    else:
        lines.append("未取到（非 Windows 或匹配不到任务）。")
    lines += [
        "",
        "## 这套数字的口径",
        "",
        "- `runs_seen` 是 broker 每 slug 的环形缓冲（上限 20 条），**不是窗口内次数**；"
        "高频 job 会一直显示 20，不能据此算成功率。",
        "- `last_run` 来自 job 字段；broker 重启后可能为空，此时本卡只报事实，不判定 stale。",
        "- Windows 任务的 `Last Result`：`0` 成功、`267011` 从未运行、`267009` 正在运行，其余按失败处理。",
        "- 推算周期 = `Next Run Time − Last Run Time`，只用于 staleness；一次性任务没有 Next Run，因此不判。",
        "- `pending` 只统计 `source_signal_id` 以 `routine:` 开头且未关闭的任务。",
        "",
    ]
    return "\n".join(lines)


def publish(port, rel_path, content, message):
    req = urllib.request.Request(
        "http://127.0.0.1:%d/wiki/write" % port,
        data=json.dumps(
            {
                "slug": "office",
                "path": rel_path,
                "content": content,
                "mode": "replace",
                "commit_message": message,
            },
            ensure_ascii=False,
        ).encode("utf-8"),
        headers={"Authorization": "Bearer " + token(), "Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=60) as f:
            return f.status, json.load(f)
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode("utf-8", "replace")


def self_test():
    bad = 0
    checks = []

    def eq(name, got, want):
        checks.append((name, got == want, got, want))

    eq("win-result-ok", win_result("0"), "ok")
    eq("win-result-never", win_result("267011"), "never_run")
    eq("win-result-running", win_result("267009"), "running")
    eq("win-result-fail", win_result("1"), "fail")
    eq("win-result-blank", win_result(""), "unknown")

    eq("parse-win-time", parse_win_time("2026/11/2 11:21:01"), dt.datetime(2026, 11, 2, 11, 21, 1))
    eq("parse-win-na", parse_win_time("N/A"), None)

    text = (
        'HostName,"TaskName","Next Run Time","Status","Logon Mode","Last Run Time","Last Result",'
        '"Author","Task To Run","Start In","Comment","Scheduled Task State",'
        '"Idle Time","Power Management","Run As User","Delete If Not Scheduled","Idle","Power",'
        '"Schedule Type","Start Time","Start Date","End Date","Days","Months",'
        '"Repeat: Every","Repeat: Until: Time","Repeat: Until: Duration","Repeat: If Running"\n'
        'PC,"\\Gawkbot\\ForexBridge","2026/11/2 16:19:00","Ready","Interactive",'
        '"2026/11/2 10:19:01","0","PC","cmd","N/A","note","Enabled",'
        '"Disabled","","ireke","Disabled","","","One Time, Minutes","10:19:00","2026/10/30","N/A","N/A","N/A",'
        '"0 hours, 360 minutes","None","Disabled","Disabled"\n'
    )
    rows = parse_schtasks_csv(text)
    eq("csv-rows", len(rows), 1)
    eq("repeat-en", parse_repeat_minutes("0 hours, 15 minutes"), 15)
    eq("repeat-zh", parse_repeat_minutes("0 小时， 15 分钟"), 15)
    eq("repeat-days", parse_repeat_minutes("1 天"), 1440)
    eq("repeat-empty", parse_repeat_minutes("N/A"), 0)
    o = win_obs(rows[0], dt.datetime(2026, 11, 2, 12, 0), offset=dt.timedelta(0))
    eq("csv-label", o["label"], "Gawkbot\\ForexBridge")
    eq("csv-interval", o["interval_minutes"], 360)
    eq("csv-result", o["last_result"], "ok")
    eq("csv-stale", stale_reason(o, dt.datetime(2026, 11, 2, 12, 0)), "")

    # A task that never ran shows 1999/11/30 as its Last Run Time. Treated as
    # a date it yields a 14-million-minute period and a page claiming 1999.
    never = dict(
        rows[0],
        **{
            "Last Run Time": "1999/11/30 0:00:00",
            "Last Result": "267011",
            "Next Run Time": "2026/11/3 9:05:00",
            "Repeat": "已禁用",
        }
    )
    n = win_obs(never, dt.datetime(2026, 11, 2, 12, 0), offset=dt.timedelta(0))
    eq("never-run-last", n["last_run"], "")
    eq("never-run-interval", n["interval_minutes"], 0)
    eq("never-run-result", n["last_result"], "never_run")
    eq("never-run-not-alert", scheduling_alert(n, dt.datetime(2026, 11, 2, 12, 0)), ("", ""))

    # 6h interval, now is 1h after the next_run -> the scheduler stopped
    # advancing it. Overdue needs no period to detect, which is why it is a
    # separate kind from "silent for 2x interval".
    eq(
        "overdue-kind",
        scheduling_alert(o, dt.datetime(2026, 11, 2, 18, 0))[0],
        "overdue",
    )
    # disabled on purpose: silent until the last run is old enough to ask
    eq("disabled-fresh", stale_reason(dict(o, enabled=False), dt.datetime(2026, 11, 2, 12, 0)), "")
    eq(
        "disabled-dormant",
        stale_reason(dict(o, enabled=False), dt.datetime(2026, 12, 1)).startswith("disabled"),
        True,
    )
    # no interval -> no staleness rule rather than a guessed one, and the
    # fixture's next_run is still in the future at this `now`.
    o3 = dict(o, interval_minutes=0)
    eq("no-interval-not-stale", stale_reason(o3, dt.datetime(2026, 11, 2, 12, 0)), "")
    # No last_run, no runs, no next_run: that is missing data (the broker
    # keeps run history in memory), not a broken job. This was the first
    # version's only false alarm -- six system crons reported "never ran".
    eq(
        "no-data-not-alerted",
        stale_reason(
            {"enabled": True, "interval_minutes": 1, "last_run": "", "next_run": "", "runs_seen": 0},
            dt.datetime(2026, 11, 2, 12, 0),
        ),
        "",
    )

    tasks = [
        {"status": "done", "source_signal_id": "routine:5:2026-11-01T03:08:00Z"},
        {"status": "pending", "source_signal_id": "routine:5:2026-11-02T03:08:00Z"},
        {"status": "pending", "source_signal_id": "routine:a:b:2026-11-02T03:08:00Z"},
        {"status": "pending", "source_signal_id": "review:ACMER-1"},
        {"status": "pending"},
    ]
    pend = pending_by_slug(tasks)
    eq("pending-slug", sorted(pend), ["5", "a:b"])
    eq("pending-count", len(pend["5"]), 1)

    runs = [
        {"status": "ok", "finished_at": "2026-11-02T01:00:00"},
        {"status": "failed", "started_at": "2026-11-02T02:00:00"},
        {"status": "ok", "finished_at": "2026-11-02T03:00:00"},
    ]
    ok, failed, other, last_ok = run_status(runs)
    eq("run-status", (ok, failed, other), (2, 1, 0))
    eq("run-last-ok", last_ok, "2026-11-02T03:00:00")

    alerts, totals = analyse([dict(o, pending=3)], dt.datetime(2026, 11, 2, 12, 0), max_pending=10)
    eq("no-queue-alert", [a[0] for a in alerts], [])
    alerts, totals = analyse([dict(o, pending=12)], dt.datetime(2026, 11, 2, 12, 0), max_pending=10)
    eq("queue-alert", any(a[0] == "queue" for a in alerts), True)

    for name, ok_, got, want in checks:
        print("%-20s %s %s" % (name, "PASS" if ok_ else "FAIL", "" if ok_ else "got %r want %r" % (got, want)))
        if not ok_:
            bad += 1
    print("self-test: %d/%d" % (len(checks) - bad, len(checks)))
    return 0 if bad == 0 else 1


def main(argv):
    try:
        sys.stdout.reconfigure(encoding="utf-8")
    except Exception:
        pass
    ap = argparse.ArgumentParser()
    ap.add_argument("--root", default=default_root())
    ap.add_argument("--data-dir", default=default_data_dir())
    ap.add_argument("--port", type=int, default=DEFAULT_PORT)
    ap.add_argument("--source", default="all", choices=("all", "broker", "schtasks"))
    ap.add_argument("--match", default=DEFAULT_MATCH, help="regex for Windows task names")
    ap.add_argument("--max-pending", type=int, default=10)
    ap.add_argument("--publish", action="store_true", help="commit the snapshot through the broker")
    ap.add_argument("--date", default=None, help="snapshot date (default today)")
    ap.add_argument("--self-test", action="store_true")
    a = ap.parse_args(argv)
    if a.self_test:
        return self_test()

    now = now_utc()
    day = a.date or now_local().strftime("%Y-%m-%d")
    observations = []
    errors = []

    if a.source in ("all", "broker"):
        try:
            jobs, runs, tasks = fetch_broker(a.port, token())
            pend = pending_by_slug(tasks)
            for j in jobs:
                observations.append(broker_obs(j, runs.get(j.get("slug", ""), []), pend, now))
        except (urllib.error.URLError, OSError, ValueError) as e:
            errors.append("broker: %s" % e)

    if a.source in ("all", "schtasks"):
        try:
            for row in collect_schtasks(a.match):
                observations.append(win_obs(row, now))
        except (OSError, subprocess.SubprocessError) as e:
            errors.append("schtasks: %s" % e)

    if errors and not observations:
        print("scorecard failed: nothing collected")
        for e in errors:
            print("  - " + e)
        return 1

    alerts, totals = analyse(observations, now, a.max_pending)
    snap = {
        "schema": SCHEMA,
        "generated_at": now.strftime("%Y-%m-%dT%H:%M:%S"),
        "date": day,
        "port": a.port,
        "match": a.match,
        "observations": observations,
        "alerts": alerts,
        "totals": totals,
        "errors": errors,
    }

    wiki_dir = os.path.join(a.root, "team", "routines")
    data_dir = a.data_dir
    os.makedirs(wiki_dir, exist_ok=True)
    os.makedirs(data_dir, exist_ok=True)
    json_path = os.path.join(data_dir, "scorecard-%s.json" % day)
    md_path = os.path.join(wiki_dir, "scorecard-%s.md" % day)
    with open(json_path, "w", encoding="utf-8", newline="\n") as f:
        json.dump(snap, f, ensure_ascii=False, indent=1, sort_keys=True)
        f.write("\n")
    with open(md_path, "w", encoding="utf-8", newline="\n") as f:
        f.write(render_md(snap))

    with open(os.path.join(data_dir, "history.jsonl"), "a", encoding="utf-8", newline="\n") as f:
        for o in observations:
            row = dict(o)
            row["ts"] = snap["generated_at"]
            f.write(json.dumps(row, ensure_ascii=False, sort_keys=True) + "\n")

    print(
        "scorecard: %d jobs, %d alerts, pending=%d -> %s"
        % (len(observations), len(alerts), totals["pending_total"], os.path.relpath(md_path, a.root).replace(os.sep, "/"))
    )
    for kind, who, detail in alerts:
        print("  %-8s %s: %s" % (kind, who, detail))
    for e in errors:
        print("  warn    " + e)

    if a.publish:
        # The wiki accepts markdown only (validateArticlePath ends in .md), so
        # the JSON snapshot and the JSONL history stay in ~/.wuphf/data: they
        # are runtime observations, not articles. The page carries the verdict.
        rel = "team/routines/scorecard-%s.md" % day
        with open(md_path, encoding="utf-8") as f:
            body = f.read()
        code, resp = publish(a.port, rel, body, "routine scorecard: %s" % day)
        if code != 200:
            print("written locally but publish failed: HTTP %s %s" % (code, resp))
            return 1
        print("published %s %s" % (rel, resp.get("commit_sha", "")))
    else:
        print("note: written to the working tree only; add --publish to commit it")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
