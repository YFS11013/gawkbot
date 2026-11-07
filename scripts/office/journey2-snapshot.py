"""Capture one Journey 2 evidence bundle (tasks, messages, wiki, scheduler, logs).

Usage: python scripts/office/journey2-snapshot.py <label>

Writes docs/office/core-loop-runs/2026-11-08-journey2/snapshots/<label>/ with:
  tasks.json, task-<ID>.json, messages-<channel>.json, wiki-tree.txt,
  scheduler.json, health.json, headless-<bot>.log, broker.log
Everything is read-only against the running office; the token file names the
broker port so the script never guesses.
"""
import json
import os
import re
import sys
import urllib.request
from pathlib import Path

LABEL = sys.argv[1] if len(sys.argv) > 1 else "manual"
SPACE = Path(os.path.expanduser("~/.wuphf-spaces/journey2"))
TEAM = SPACE / ".wuphf" / "team"
LOGS = SPACE / ".wuphf" / "logs"
OUT = Path(__file__).resolve().parents[2] / "docs" / "office" / "core-loop-runs" / "2026-11-08-journey2" / "snapshots" / LABEL
OUT.mkdir(parents=True, exist_ok=True)

PORT = 7910
token_file = Path(os.environ.get("TEMP", "")) / f"wuphf-broker-token-{PORT}"
TOKEN = token_file.read_text(encoding="utf-8").strip() if token_file.exists() else ""


def get(path):
    req = urllib.request.Request(f"http://127.0.0.1:{PORT}{path}", headers={"Authorization": f"Bearer {TOKEN}"})
    with urllib.request.urlopen(req, timeout=10) as r:
        return json.loads(r.read().decode())


def dump(name, obj):
    (OUT / name).write_text(json.dumps(obj, indent=2, ensure_ascii=False), encoding="utf-8")


state = json.loads((TEAM / "broker-state.json").read_text(encoding="utf-8"))
dump("tasks.json", state.get("tasks", []))
dump("messages.json", state.get("messages", []))
for t in state.get("tasks", []):
    dump(f"task-{t['id']}.json", t)

try:
    dump("scheduler.json", get("/scheduler"))
except Exception as exc:  # noqa: BLE001 - evidence capture must not fail the run
    (OUT / "scheduler.error.txt").write_text(str(exc), encoding="utf-8")
try:
    dump("health.json", get("/health"))
except Exception as exc:  # noqa: BLE001
    (OUT / "health.error.txt").write_text(str(exc), encoding="utf-8")

wiki = SPACE / ".wuphf" / "wiki"
rows = []
for p in sorted(wiki.rglob("*")):
    if p.is_file():
        st = p.stat()
        rows.append(f"{p.relative_to(wiki).as_posix()}\t{st.st_size}\t{st.st_mtime:.0f}")
(OUT / "wiki-tree.txt").write_text("\n".join(rows) + "\n", encoding="utf-8")

for log in sorted(LOGS.glob("*.log")):
    (OUT / f"headless-{log.name}").write_text(log.read_text(encoding="utf-8", errors="replace"), encoding="utf-8")

events = SPACE / ".wuphf" / "office" / "events.jsonl"
if events.exists():
    (OUT / "events.jsonl").write_text(events.read_text(encoding="utf-8", errors="replace"), encoding="utf-8")

print(f"snapshot -> {OUT} ({len(rows)} wiki files, {len(state.get('tasks', []))} tasks)")
