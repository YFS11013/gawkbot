#!/usr/bin/env python3
"""Per-task token report from the broker's persisted turn manifests.

Events are written by the headless runners to <runtime home>/.wuphf/office/events.jsonl.
Since the review lane creates its own task (source_signal_id=review:<parent>), this
script can join reviewer turns back to the coding task and report the verification
share without guessing.

Usage:
  python scripts/office/task-token-report.py
  python scripts/office/task-token-report.py --runtime-home ~/.wuphf \\
      --broker-url http://127.0.0.1:7892 --token-file /path/to/token
"""
from __future__ import annotations

import argparse
import json
import os
import sys
import urllib.request
from collections import defaultdict
from pathlib import Path


def read_manifests(path: Path) -> list[dict]:
    if not path.exists():
        return []
    rows = []
    with path.open(encoding="utf-8", errors="replace") as fh:
        for line in fh:
            line = line.strip()
            if not line:
                continue
            try:
                row = json.loads(line)
            except json.JSONDecodeError:
                continue
            if isinstance(row, dict):
                rows.append(row)
    return rows

def fetch_usage(broker_url: str | None, token: str | None) -> dict:
    """Return the broker's aggregate token ledger when manifests are absent."""
    if not broker_url or not token:
        return {}
    req = urllib.request.Request(
        f"{broker_url.rstrip('/')}/usage",
        headers={"Authorization": f"Bearer {token}"},
    )
    try:
        with urllib.request.urlopen(req, timeout=15) as resp:
            return json.load(resp)
    except Exception as exc:  # noqa: BLE001
        print(f"warning: cannot read usage ledger: {exc}", file=sys.stderr)
        return {}

def fetch_review_map(broker_url: str | None, token: str | None) -> dict[str, str]:
    """Return {review_task_id: parent_task_id} from the task board."""
    if not broker_url or not token:
        return {}
    req = urllib.request.Request(
        f"{broker_url.rstrip('/')}/tasks?all_channels=true&include_done=true",
        headers={"Authorization": f"Bearer {token}"},

    )
    try:
        with urllib.request.urlopen(req, timeout=15) as resp:
            data = json.load(resp)
    except Exception as exc:  # noqa: BLE001 - telemetry must never be fatal
        print(f"warning: cannot read task board: {exc}", file=sys.stderr)
        return {}
    out: dict[str, str] = {}
    for task in data.get("tasks", []):
        signal = str(task.get("source_signal_id") or "")
        if signal.startswith("review:"):
            out[str(task.get("id") or "")] = signal[len("review:"):]
    return out


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--runtime-home", default=os.path.expanduser("~/.wuphf"))
    ap.add_argument("--broker-url", default=None, help="e.g. http://127.0.0.1:7892")
    ap.add_argument("--token-file", default=None, help="file containing the broker token")
    args = ap.parse_args()

    token = None
    if args.token_file:
        try:
            token = Path(args.token_file).read_text(encoding="utf-8").strip()
        except OSError as exc:
            print(f"warning: cannot read token file: {exc}", file=sys.stderr)

    events = Path(args.runtime_home).expanduser() / ".wuphf" / "office" / "events.jsonl"
    rows = read_manifests(events)
    if not rows:
        print(f"no turn manifests at {events}; falling back to broker /usage aggregate")
        usage = fetch_usage(args.broker_url, token)
        if not usage:
            print("no aggregate available (pass --broker-url and --token-file)")
            return 0
        for scope in ("session", "total"):
            rec = usage.get(scope) or {}
            print(f"{scope}: total_tokens={rec.get('total_tokens', 0)} "
                  f"input={rec.get('input_tokens', 0)} output={rec.get('output_tokens', 0)} "
                  f"cache_read={rec.get('cache_read_tokens', 0)} requests={rec.get('requests', 0)}")
        for bot, rec in (usage.get("agents") or {}).items():
            print(f"agent {bot}: total_tokens={rec.get('total_tokens', 0)} "
                  f"input={rec.get('input_tokens', 0)} output={rec.get('output_tokens', 0)} "
                  f"cache_read={rec.get('cache_read_tokens', 0)}")
        return 0

    review_map = fetch_review_map(args.broker_url, token)
    stats: dict[str, dict] = defaultdict(
        lambda: {"turns": 0, "input": 0, "output": 0, "bot": "", "review_input": 0, "review_output": 0}
    )
    for row in rows:
        task_id = str(row.get("task_id") or "")
        if not task_id:
            continue
        rec = stats[task_id]
        rec["turns"] += 1
        rec["bot"] = str(row.get("agent") or rec["bot"])
        parent = review_map.get(task_id)
        bucket_in = "review_input" if parent else "input"
        bucket_out = "review_output" if parent else "output"
        rec[bucket_in] += int(row.get("input_tokens") or 0)
        rec[bucket_out] += int(row.get("output_tokens") or 0)
        if parent:
            stats[parent]  # ensure the parent row exists even without its own turns

    header = f"{'task':<22} {'bot':<14} {'turns':>5} {'input':>10} {'output':>9} {'total':>10} {'review':>10} {'share':>7}"
    print(header)
    print("-" * len(header))
    for task_id, rec in sorted(stats.items()):
        total = rec["input"] + rec["output"]
        review = rec["review_input"] + rec["review_output"]
        share = (100.0 * review / (total + review)) if (total + review) else 0.0
        print(
            f"{task_id:<22} {rec['bot']:<14} {rec['turns']:>5} "
            f"{rec['input']:>10} {rec['output']:>9} {total:>10} {review:>10} {share:>6.1f}%"
        )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
