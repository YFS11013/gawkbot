package team

import (
	"strings"
	"testing"
)

// newWriteScopeBroker installs a managed office: cos is the lead (the actor
// checkTaskActionAuthLocked treats as CEO, and the only bot allowed to set a
// definition), eng is a specialist. Both are registered members, which is what
// makes the lead gate and the create-only restriction live at all.
func newWriteScopeBroker(t *testing.T) *Broker {
	t.Helper()
	b := newPlanApprovalBroker(t)
	ensureTestMemberAccess(b, "team", "cos", "Chief of Staff")
	ensureTestMemberAccess(b, "team", "eng", "Eng")
	return b
}

// createApprovedParent files a top-level Issue and clears its plan gate, so
// sub-issue creates are judged on the rule under test instead of the planning
// gate.
func createApprovedParent(t *testing.T, b *Broker) string {
	t.Helper()
	parent, err := b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: "Land the Q4 renewal push",
		Owner: "eng", CreatedBy: "human",
	})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	startPlanning(t, b, parent.Task.ID)
	return parent.Task.ID
}

// J3-3 follow-up (Journey 8): the planner's lanes must carry a write scope in
// the create call. Without one the owner can never publish to the shared wiki —
// /wiki/write-task authorizes exactly definition.artifact_path — so both clean
// runs (J6, J7) ended with zero write-task calls and zero team/** markdown.
// The refusal names the creator, the field to fix, and the escape hatch, and it
// says a plain retry cannot succeed.
func TestLeadSubIssueCreateRequiresWriteScope(t *testing.T) {
	b := newWriteScopeBroker(t)
	parentID := createApprovedParent(t, b)

	_, err := b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: "Draft the renewal one-pager",
		Owner: "eng", CreatedBy: "cos", ParentIssueID: parentID,
	})
	if err == nil {
		t.Fatal("a lead-authored lane with no write scope must be refused")
	}
	for _, want := range []string{"no write scope", "@cos", "definition.artifact_path", "artifact_not_required_reason", "refused again", "team/accounts/"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("refusal should mention %q, got %v", want, err)
		}
	}
	b.mu.Lock()
	before := len(b.tasks)
	b.mu.Unlock()
	if before != 1 {
		t.Fatalf("refused create must not append a lane, have %d tasks", before)
	}
}

// The declared scope arrives in the SAME create call and lands on the lane, so
// the owner can publish without a second define round-trip.
func TestLeadSubIssueCreateAcceptsDeclaredScope(t *testing.T) {
	b := newWriteScopeBroker(t)
	parentID := createApprovedParent(t, b)

	lane, err := b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: "Write the renewal one-pager",
		Owner: "eng", CreatedBy: "cos", ParentIssueID: parentID,
		Definition: &TaskDefinition{Goal: "A one-pager AE can send", ArtifactPath: "team/playbooks/renewal-one-pager.md"},
	})
	if err != nil {
		t.Fatalf("create with a declared scope: %v", err)
	}
	if lane.Task.Definition == nil || lane.Task.Definition.ArtifactPath != "team/playbooks/renewal-one-pager.md" {
		t.Fatalf("definition.artifact_path must ride on the lane, got %+v", lane.Task.Definition)
	}
	if lane.Task.Definition.DefinedAt == "" {
		t.Fatal("definition must be stamped by the broker")
	}
}

// A lane that genuinely ships no file satisfies the requirement by saying so,
// the same escape the completion gate honours.
func TestLeadSubIssueCreateAcceptsNotRequiredReason(t *testing.T) {
	b := newWriteScopeBroker(t)
	parentID := createApprovedParent(t, b)

	lane, err := b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: "Review the renewal one-pager",
		Owner: "eng", CreatedBy: "cos", ParentIssueID: parentID,
		ArtifactNotRequiredReason: "review verdict lives on the task",
	})
	if err != nil {
		t.Fatalf("create with a not-required reason: %v", err)
	}
	if !strings.Contains(lane.Task.ArtifactNotRequired, "review verdict lives on the task") ||
		!strings.Contains(lane.Task.ArtifactNotRequired, "cos") {
		t.Fatalf("not-required declaration must be stamped with reason and actor, got %q", lane.Task.ArtifactNotRequired)
	}
}

// A scope outside the allowed roots is a self-authorization attempt: it must
// fail validation rather than create a lane that can write anywhere.
func TestLeadSubIssueCreateRejectsScopeOutsideRoots(t *testing.T) {
	b := newWriteScopeBroker(t)
	parentID := createApprovedParent(t, b)

	_, err := b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: "Draft the renewal FAQ",
		Owner: "eng", CreatedBy: "cos", ParentIssueID: parentID,
		Definition: &TaskDefinition{Goal: "FAQ", ArtifactPath: "secrets/notes.md"},
	})
	if err == nil || !strings.Contains(err.Error(), "definition artifact_path must live under one of") {
		t.Fatalf("a scope outside the allowed roots must be refused, got %v", err)
	}
}

// A specialist has no authority to pin a shared path (and cannot but its own
// lane), so the scope requirement does not apply to it — refusing there would be
// a dead end, not a fix.
func TestSpecialistSubIssueCreateIsNotBlockedByScopeRule(t *testing.T) {
	b := newWriteScopeBroker(t)
	parentID := createApprovedParent(t, b)

	if _, err := b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: "Collect renewal objections from AE",
		Owner: "eng", CreatedBy: "eng", ParentIssueID: parentID,
	}); err != nil {
		t.Fatalf("specialist lane create must not require a scope: %v", err)
	}
}

// Definition is the lead's contract; a specialist may not set it, and that
// restriction must not be bypassable by moving it into the create call.
func TestSpecialistCreateCannotSetDefinition(t *testing.T) {
	b := newWriteScopeBroker(t)
	parentID := createApprovedParent(t, b)

	_, err := b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: "Collect renewal objections",
		Owner: "eng", CreatedBy: "eng", ParentIssueID: parentID,
		Definition: &TaskDefinition{Goal: "mine now", ArtifactPath: "team/accounts/acme.md"},
	})
	if err == nil || !strings.Contains(err.Error(), "office lead") {
		t.Fatalf("specialist definition on create must be refused, got %v", err)
	}
}

// Journey 8, D2: the planner split one issue into six lanes by firing each lane
// twice, one second apart, re-prefixing the title the second time
// ("JOURN-6-A: Acme 一页报价说明初稿" then "子任务 A — 一页报价说明初稿"). Token
// similarity scored that pair 0.20 — CJK has no spaces, so each clause is one
// token — and the sibling guard let both through. The duplicate then either
// re-did shipped work or, in the reviewer pair, sat running forever with nothing
// left to write.
func TestSiblingLaneWithSharedCJKPhraseIsRefused(t *testing.T) {
	b := newWriteScopeBroker(t)
	parentID := createApprovedParent(t, b)

	if _, err := b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: "JOURN-6-A: Acme 一页报价说明初稿",
		Owner: "eng", CreatedBy: "cos", ParentIssueID: parentID,
		Definition: &TaskDefinition{Goal: "产出一页报价说明初稿", ArtifactPath: "team/reviews/acme-renewal-quote.md"},
	}); err != nil {
		t.Fatalf("first lane: %v", err)
	}
	_, err := b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: "子任务 A — 一页报价说明初稿",
		Owner: "eng", CreatedBy: "cos", ParentIssueID: parentID,
		Definition: &TaskDefinition{Goal: "子任务 A — 一页报价说明初稿", ArtifactPath: "team/research/acme-renewal-quote.md"},
	})
	if err == nil {
		t.Fatal("a re-prefixed restatement of a sibling lane must be refused")
	}
	if !strings.Contains(err.Error(), "duplicates") {
		t.Fatalf("refusal should say the lane duplicates a sibling, got %v", err)
	}
}

// Journey 8, D2 (the reviewer pair): one file is one deliverable. JOURN-23 and
// JOURN-25 both declared team/reviews/acme-renewal-qa.md; the first wrote it and
// closed, the second had nothing left to write and stayed in_progress for the
// rest of the run.
func TestSiblingLaneDeclaringSameWriteScopeIsRefused(t *testing.T) {
	b := newWriteScopeBroker(t)
	parentID := createApprovedParent(t, b)

	if _, err := b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: "质检 Acme 交付物",
		Owner: "eng", CreatedBy: "cos", ParentIssueID: parentID,
		Definition: &TaskDefinition{Goal: "逐项核对两件交付物，给出通过/退回结论", ArtifactPath: "team/reviews/acme-renewal-qa.md"},
	}); err != nil {
		t.Fatalf("first lane: %v", err)
	}
	_, err := b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: "出厂质检",
		Owner: "eng", CreatedBy: "cos", ParentIssueID: parentID,
		Definition: &TaskDefinition{Goal: "子任务 C — 出厂质检", ArtifactPath: "team/reviews/acme-renewal-qa.md"},
	})
	if err == nil {
		t.Fatal("two open lanes under one parent must not declare the same write scope")
	}
	for _, want := range []string{"same write scope", "team/reviews/acme-renewal-qa.md", "allow_duplicate=true"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("refusal should mention %q, got %v", want, err)
		}
	}
}

// The escape has to actually escape, or a deliberate rewrite of one file has no
// way forward at all.
func TestSiblingLaneAllowDuplicateBypassesScopeAndPhraseRules(t *testing.T) {
	b := newWriteScopeBroker(t)
	parentID := createApprovedParent(t, b)

	if _, err := b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: "质检 Acme 交付物",
		Owner: "eng", CreatedBy: "cos", ParentIssueID: parentID,
		Definition: &TaskDefinition{Goal: "逐项核对交付物", ArtifactPath: "team/reviews/acme-renewal-qa.md"},
	}); err != nil {
		t.Fatalf("first lane: %v", err)
	}
	if _, err := b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: "质检 Acme 交付物（返工）",
		Owner: "eng", CreatedBy: "cos", ParentIssueID: parentID, AllowDuplicate: true,
		Definition: &TaskDefinition{Goal: "逐项核对交付物并返工", ArtifactPath: "team/reviews/acme-renewal-qa.md"},
	}); err != nil {
		t.Fatalf("allow_duplicate must opt out of the new sibling rules, got %v", err)
	}
}

// The new rules must not swallow genuinely distinct CJK lanes: a different
// deliverable suffix (初稿/终稿) and a review aimed at a different task share
// atoms but not a deliverable name.
func TestDistinctCJKLanesAreStillAllowed(t *testing.T) {
	b := newWriteScopeBroker(t)
	parentID := createApprovedParent(t, b)

	if _, err := b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: "Acme 续约报价初稿",
		Owner: "eng", CreatedBy: "cos", ParentIssueID: parentID,
		Definition: &TaskDefinition{Goal: "初稿", ArtifactPath: "team/reviews/renewal-draft.md"},
	}); err != nil {
		t.Fatalf("first lane: %v", err)
	}
	if _, err := b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: "Acme 续约报价终稿",
		Owner: "eng", CreatedBy: "cos", ParentIssueID: parentID,
		Definition: &TaskDefinition{Goal: "终稿", ArtifactPath: "team/reviews/renewal-final.md"},
	}); err != nil {
		t.Fatalf("a distinct deliverable must not be refused as a duplicate: %v", err)
	}
	if _, err := b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: "审阅 JOURN-15 的交付",
		Owner: "eng", CreatedBy: "cos", ParentIssueID: parentID,
		Definition: &TaskDefinition{Goal: "审阅 JOURN-15 的交付", ArtifactPath: "team/reviews/JOURN-15.md"},
	}); err != nil {
		t.Fatalf("first review lane: %v", err)
	}
	if _, err := b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: "审阅 JOURN-19 的交付",
		Owner: "eng", CreatedBy: "cos", ParentIssueID: parentID,
		Definition: &TaskDefinition{Goal: "审阅 JOURN-19 的交付", ArtifactPath: "team/reviews/JOURN-19.md"},
	}); err != nil {
		t.Fatalf("a review aimed at a different task must not be refused: %v", err)
	}
}
