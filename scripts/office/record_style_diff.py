#!/usr/bin/env python3
"""Record what a human changed in a bot's draft (the N1 loop).

The mechanism Anthropic's Japan BDR fleet reports as the highest-leverage
part of its setup is not a better prompt: every agent compares the draft it
wrote with the message the human actually sent, and writes the difference
back into a shared knowledge base. A correction made once stops repeating
everywhere. This script is the mechanical half of that loop -- it records the
diff; the bot's own pass turns the diffs into `rule:` entries.

    python record_style_diff.py --bot henji --draft out/draft.md --final sent.md
    python record_style_diff.py --bot cos --draft a.md --final b.md --kind brief --publish

What it writes: one JSON line per (draft, final) pair, appended to
`team/style/diffs/<bot>.jsonl`. The `id` is derived from bot + both content
hashes, so re-running on the same pair is a no-op instead of a second "the
human changed something" signal. An identical pair is still recorded with
`changed: 0`: "the draft needed no edit" is evidence too, and dropping it
would make the digest look like nothing was ever reviewed.

The gate that keeps this loop honest is check_style_digest.py: it refuses a
style file whose entries cite diff ids that do not exist, and refuses a style
file that was never updated after new diffs landed.

Exit codes: 0 recorded/duplicate, 1 unreadable input, 2 usage.
"""

import argparse
import datetime as dt
import difflib
import hashlib
import json
import os
import re
import sys
import urllib.error
import urllib.request

SCHEMA = 1
MAX_HUNKS = 25
MAX_LINE = 400
# A hunk longer than this is a rewrite, not a wording correction. Keeping the
# whole replacement would bury the pattern the bot is supposed to learn.
MAX_HUNK_LINES = 12
DEFAULT_PORT = 7892


def default_root():
    return os.path.join(os.path.expanduser("~"), ".wuphf", "wiki")


def clean_bot(name):
    s = re.sub(r"[^a-z0-9-]+", "-", (name or "").strip().lower()).strip("-")
    return s or "bot"


def read_text(path):
    with open(path, encoding="utf-8-sig", errors="replace") as f:
        return f.read()


def sha(text):
    return hashlib.sha256(text.encode("utf-8")).hexdigest()


def rel(path, root):
    """Wiki-relative when under the root, forward slashes either way."""
    ap = os.path.abspath(path)
    ar = os.path.abspath(root)
    try:
        inside = os.path.commonpath([ap, ar]) == ar
    except ValueError:  # different drives on Windows
        inside = False
    if inside:
        return os.path.relpath(ap, ar).replace(os.sep, "/")
    return ap.replace(os.sep, "/")


def clip_line(s, n=MAX_LINE):
    s = s.rstrip("\r\n")
    return s if len(s) <= n else s[: n - 1] + "…"


def collect_hunks(draft_lines, final_lines):
    sm = difflib.SequenceMatcher(None, draft_lines, final_lines, autojunk=False)
    out = []
    for tag, i1, i2, j1, j2 in sm.get_opcodes():
        if tag == "equal":
            continue
        old = [clip_line(x) for x in draft_lines[i1:i2]][:MAX_HUNK_LINES]
        new = [clip_line(x) for x in final_lines[j1:j2]][:MAX_HUNK_LINES]
        if not old and not new:
            continue
        out.append({"op": tag, "at": i1 + 1, "old": old, "new": new})
    return out


def build_record(bot, kind, draft_path, final_path, draft, final, root, note):
    d_sha, f_sha = sha(draft), sha(final)
    hunks = collect_hunks(draft.splitlines(), final.splitlines())
    truncated = len(hunks) > MAX_HUNKS
    hunks = hunks[:MAX_HUNKS]
    return {
        "schema": SCHEMA,
        "id": "d" + sha("%s|%s|%s" % (bot, d_sha, f_sha))[:12],
        "ts": dt.datetime.now(dt.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "bot": bot,
        "kind": kind,
        "draft": rel(draft_path, root),
        "final": rel(final_path, root),
        "draft_sha256": d_sha,
        "final_sha256": f_sha,
        "changed": len(hunks),
        "added": sum(len(h["new"]) for h in hunks),
        "removed": sum(len(h["old"]) for h in hunks),
        "truncated": truncated,
        "note": note,
        "hunks": hunks,
    }


def existing_ids(path):
    ids = set()
    if not os.path.isfile(path):
        return ids
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
                ids.add(rec["id"])
    return ids


def read_jsonl(path):
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
            if isinstance(rec, dict):
                recs.append(rec)
    return recs


def token():
    t = os.environ.get("WUPHF_TOKEN", "").strip()
    if t:
        return t
    path = os.path.join(
        os.environ.get("TEMP", "/tmp"), "wuphf-broker-token-%d" % DEFAULT_PORT
    )
    with open(path, encoding="utf-8") as f:
        return f.read().strip()


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


def record(bot, draft_path, final_path, root, kind="generic", note=""):
    """Return (status, record, jsonl_path). status: new|duplicate|identical."""
    bot = clean_bot(bot)
    draft, final = read_text(draft_path), read_text(final_path)
    rec = build_record(bot, kind, draft_path, final_path, draft, final, root, note)
    jsonl = os.path.join(root, "team", "style", "diffs", bot + ".jsonl")
    if rec["id"] in existing_ids(jsonl):
        return "duplicate", rec, jsonl
    os.makedirs(os.path.dirname(jsonl), exist_ok=True)
    with open(jsonl, "a", encoding="utf-8", newline="\n") as f:
        f.write(json.dumps(rec, ensure_ascii=False, sort_keys=True) + "\n")
    status = "identical" if rec["changed"] == 0 else "new"
    return status, rec, jsonl


def self_test():
    import tempfile

    bad = 0
    with tempfile.TemporaryDirectory() as tmp:
        draft = os.path.join(tmp, "draft.md")
        final = os.path.join(tmp, "final.md")
        with open(draft, "w", encoding="utf-8", newline="\n") as f:
            f.write("感谢您的来信。\n我们会在 24 小时内回复。\n")
        with open(final, "w", encoding="utf-8", newline="\n") as f:
            f.write("这个问题我来处理。\n我们会在 24 小时内回复。\n")

        status, rec, jsonl = record("henji", draft, final, tmp)
        rows = read_jsonl(jsonl)
        checks = [
            ("first-record", status == "new" and len(rows) == 1),
            ("hunk", rec["changed"] == 1 and rec["hunks"][0]["old"][0].startswith("感谢")),
            ("sha", len(rec["draft_sha256"]) == 64),
            ("rel-path", rec["draft"] == "draft.md"),
        ]

        status2, rec2, _ = record("henji", draft, final, tmp)
        checks.append(("idempotent", status2 == "duplicate" and len(read_jsonl(jsonl)) == 1))
        checks.append(("stable-id", rec2["id"] == rec["id"]))

        same = os.path.join(tmp, "same.md")
        with open(same, "w", encoding="utf-8", newline="\n") as f:
            f.write(read_text(draft))
        status3, rec3, _ = record("henji", draft, same, tmp)
        checks.append(("identical-recorded", status3 == "identical" and rec3["changed"] == 0))
        checks.append(("two-rows", len(read_jsonl(jsonl)) == 2))

        # A rewrite must not smuggle the whole thing into one hunk, and more
        # than MAX_HUNKS separate edits must be capped rather than unbounded.
        base = os.path.join(tmp, "base40.md")
        with open(base, "w", encoding="utf-8", newline="\n") as f:
            for i in range(40):
                f.write("原行 %d\n" % i)
        many = os.path.join(tmp, "many.md")
        with open(many, "w", encoding="utf-8", newline="\n") as f:
            for i in range(40):
                f.write("原行 %d\n新增 %d\n" % (i, i))
        _, rec4, _ = record("cos", base, many, tmp)
        checks.append(
            (
                "hunks-capped",
                len(rec4["hunks"]) == MAX_HUNKS and rec4["truncated"] and rec4["changed"] == MAX_HUNKS,
            )
        )
        checks.append(("hunk-lines-capped", all(len(h["old"]) <= MAX_HUNK_LINES for h in rec4["hunks"])))

        for name, ok in checks:
            print("%-18s %s" % (name, "PASS" if ok else "FAIL"))
            if not ok:
                bad += 1
        print("self-test: %d/%d" % (len(checks) - bad, len(checks)))
    return 0 if bad == 0 else 1


def main(argv):
    try:
        sys.stdout.reconfigure(encoding="utf-8")
    except Exception:
        pass
    ap = argparse.ArgumentParser()
    ap.add_argument("--bot", required=False, help="who wrote the draft (file name of the jsonl)")
    ap.add_argument("--draft", help="the draft the bot produced")
    ap.add_argument("--final", help="what the human actually sent/saved")
    ap.add_argument("--kind", default="generic", help="email|prep|brief|review|generic")
    ap.add_argument("--note", default="")
    ap.add_argument("--root", default=default_root())
    ap.add_argument("--publish", action="store_true", help="commit the jsonl through the broker")
    ap.add_argument("--port", type=int, default=DEFAULT_PORT)
    ap.add_argument("--self-test", action="store_true")
    a = ap.parse_args(argv)

    if a.self_test:
        return self_test()
    if not (a.bot and a.draft and a.final):
        print("usage: record_style_diff.py --bot <b> --draft <d> --final <f>", file=sys.stderr)
        return 2

    try:
        status, rec, jsonl = record(a.bot, a.draft, a.final, a.root, a.kind, a.note)
    except OSError as e:
        print("cannot read input: %s" % e)
        return 1

    rel_jsonl = rel(jsonl, a.root)
    print(
        "%s %s bot=%s changed=%d added=%d removed=%d hunks=%d -> %s"
        % (
            status,
            rec["id"],
            rec["bot"],
            rec["changed"],
            rec["added"],
            rec["removed"],
            len(rec["hunks"]),
            rel_jsonl,
        )
    )
    if a.publish:
        with open(jsonl, encoding="utf-8") as f:
            content = f.read()
        code, resp = publish(
            a.port,
            rel_jsonl,
            content,
            "style-diff: %s %s" % (rec["bot"], rec["ts"]),
        )
        if code != 200:
            print("recorded locally but publish failed: HTTP %s %s" % (code, resp))
            return 1
        print("published %s" % resp.get("commit_sha", ""))
    elif status == "new":
        print("note: written to the working tree only; add --publish to commit it")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
