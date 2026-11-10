import json, os, time, urllib.request, urllib.error

PORT = 7910
TOK = open(os.path.join(os.environ["TEMP"], "wuphf-broker-token-7910"), encoding="utf-8").read().strip()


def call(method, path, body=None):
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request("http://127.0.0.1:%d%s" % (PORT, path), data=data, method=method,
                                 headers={"Content-Type": "application/json", "Authorization": "Bearer " + TOK})
    try:
        with urllib.request.urlopen(req, timeout=120) as r:
            return r.status, r.read().decode()
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode()


def jget(path):
    s, b = call("GET", path)
    return json.loads(b)


def task(tid):
    d = jget("/tasks/" + tid)
    return d.get("task") or d


def cards():
    return [r for r in jget("/requests").get("requests") or [] if r.get("status") == "pending"]


def show(label, res):
    s, b = res
    try:
        d = json.loads(b)
        if isinstance(d, dict) and "task" in d:
            t = d["task"]
            b = json.dumps({k: t.get(k) for k in ("id", "status", "lifecycle_state", "artifact", "parent_issue_id")}, ensure_ascii=False)
    except Exception:
        pass
    print("%-42s %s %s" % (label, s, str(b)[:300].replace("\n", " ")))
    return s, b


def create(body):
    s, b = call("POST", "/tasks", body)
    show("create " + body.get("title", "")[:28], (s, b))
    if s == 200:
        return (json.loads(b).get("task") or {}).get("id")
    return None


issue = create({
    "action": "create", "channel": "cos__human", "created_by": "you", "owner": "cos",
    "title": "Guard exercise 2: two-lane pack",
    "details": "Two independent files, one per lane. Deliver them as wiki articles.",
})
print("ISSUE", issue, task(issue).get("status"), task(issue).get("lifecycle_state"))

for r in cards():
    key = r.get("dedupe_key") or ""
    if key.startswith("plan-approval:" + str(issue)):
        show("approve plan", call("POST", "/requests/answer", {"id": r["id"], "choice_id": "approve"}))
if task(issue).get("lifecycle_state") == "planning":
    show("start issue (human approve)", call("POST", "/tasks", {"action": "approve", "id": issue, "channel": "cos__human", "created_by": "you"}))
print("ISSUE after start", task(issue).get("status"), task(issue).get("lifecycle_state"))

LANES = {"quote": "team/accounts/guard-quote.md", "email": "team/emails/guard-email.md"}
lane_ids = {}
for name, path in LANES.items():
    lane_ids[name] = create({
        "action": "create", "channel": "cos__human", "created_by": "cos", "owner": "cos",
        "title": "Guard lane: " + name,
        "details": "Deliver %s as a wiki article." % path,
        "parent_issue_id": issue,
        "definition": {
            "goal": "Produce %s as a standalone English article." % path,
            "deliverables": [{"name": name, "format": path + " (wiki)"}],
            "success_criteria": ["file " + path + " exists"],
            "artifact_path": path,
        },
        "verification_kind": "wiki_artifact", "verification_spec": path, "verification_required": True,
    })
print("LANES", lane_ids)

for name, tid in lane_ids.items():
    print("   lane state", name, task(tid).get("status"), task(tid).get("lifecycle_state"))

for name, tid in lane_ids.items():
    show("write " + name, call("POST", "/wiki/write-task", {
        "slug": "cos", "task_id": tid, "path": LANES[name],
        "content": "# Guard " + name + "\n\nLane deliverable.\n",
        "mode": "create", "commit_message": "guard lane " + name,
    }))

for name, tid in lane_ids.items():
    show("complete lane " + name, call("POST", "/tasks", {
        "action": "complete", "id": tid, "channel": "cos__human", "created_by": "cos",
        "artifact_path": LANES[name],
    }))

time.sleep(2)
for name, tid in lane_ids.items():
    t = task(tid)
    print("LANE", name, t.get("status"), t.get("lifecycle_state"), "art", t.get("artifact"),
          "vres", (t.get("verification_result") or {}).get("pass"))
    if t.get("status") == "review":
        show("approve lane " + name, call("POST", "/tasks", {"action": "approve", "id": tid, "channel": "cos__human", "created_by": "cos"}))

time.sleep(1)
for name, tid in lane_ids.items():
    t = task(tid)
    print("LANE2", name, t.get("status"), t.get("lifecycle_state"), "art", t.get("artifact"))

print("=== guard probes")
show("V1 artifact_path = lane path", call("POST", "/tasks", {
    "action": "complete", "id": issue, "channel": "cos__human", "created_by": "cos",
    "artifact_path": LANES["quote"],
}))
show("V1b definition scope = lane path", call("POST", "/tasks", {
    "action": "define", "id": issue, "channel": "cos__human", "created_by": "cos",
    "definition": {"goal": "Aggregate the two lane files.", "artifact_path": LANES["email"]},
}))
show("V7 complete + not_required", call("POST", "/tasks", {
    "action": "complete", "id": issue, "channel": "cos__human", "created_by": "cos",
    "verification": "not_required", "verification_reason": "container",
}))
print("container after refusals:", task(issue).get("status"), task(issue).get("lifecycle_state"), task(issue).get("artifact"))
show("legit submit_for_review", call("POST", "/tasks", {
    "action": "submit_for_review", "id": issue, "channel": "cos__human", "created_by": "cos",
}))
time.sleep(2)
for r in cards():
    print("CARD", r.get("id"), r.get("dedupe_key"), "|", (r.get("question") or "")[:220].replace("\n", " "))
t = task(issue)
print("CONTAINER", t.get("status"), t.get("lifecycle_state"), "art", t.get("artifact"))
print("STATE", json.dumps({k: t.get(k) for k in ("artifact", "artifact_not_required", "hollow_completion")}, ensure_ascii=False))
