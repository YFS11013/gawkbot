package team

import (
	"strings"
	"testing"
	"time"
)

// Journey 11 (core-loop-runs/2026-11-10-journey11): cos submitted the parent
// Issue JOURN-2 for review with artifact_path set to JOURN-22's deliverable.
// From then on the Issue carried one of its two files as its own artifact, the
// decision card named only that file, and the promotion described a two-file
// Issue by one of them. The lanes already record what they shipped; the
// container may only record files it wrote itself.
func TestContainerRefusesALanePathAsItsOwnArtifact(t *testing.T) {
	b := newTestBroker(t)
	const issueID = "JOURN-2"
	now := time.Date(2026, 11, 10, 4, 24, 27, 0, time.UTC).Format(time.RFC3339)
	b.mu.Lock()
	b.tasks = append(b.tasks,
		teamTask{ID: issueID, Title: "Journey 11: Acme renewal", Channel: "team", Owner: "cos",
			TaskType: "issue", status: "in_progress", CreatedBy: "cos", CreatedAt: now, UpdatedAt: now},
		teamTask{ID: "JOURN-22", Title: "Quote one-pager", Channel: "team", Owner: "analyst",
			ParentIssueID: issueID, status: "done", Artifact: "team/accounts/acme-renewal-quote.md",
			CreatedAt: now, UpdatedAt: now},
		teamTask{ID: "JOURN-24", Title: "Follow-up email", Channel: "team", Owner: "ae",
			ParentIssueID: issueID, status: "done", Artifact: "team/emails/acme-renewal-followup.md",
			CreatedAt: now, UpdatedAt: now},
	)
	b.mu.Unlock()

	_, err := b.MutateTask(TaskPostRequest{
		Action: "submit_for_review", ID: issueID, Channel: "team", CreatedBy: "cos",
		ArtifactPath: "team/accounts/acme-renewal-quote.md",
	})
	if err == nil {
		t.Fatal("an Issue must not adopt one of its lanes' files as its own artifact")
	}
	msg := err.Error()
	for _, want := range []string{"container", "JOURN-22", "artifact_path"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("the refusal must name %q, got %q", want, msg)
		}
	}
	if task := b.TaskByID(issueID); task == nil || strings.TrimSpace(task.Artifact) != "" {
		t.Fatalf("the refused write must not land: artifact=%q", task.Artifact)
	}

	// A file the container wrote itself is a legitimate record, and so is no
	// artifact at all (the card aggregates the lanes).
	if _, err := b.MutateTask(TaskPostRequest{
		Action: "submit_for_review", ID: issueID, Channel: "team", CreatedBy: "cos",
		ArtifactPath: "team/accounts/acme-renewal-summary.md",
	}); err != nil {
		t.Fatalf("the Issue may record a file of its own: %v", err)
	}
	if task := b.TaskByID(issueID); strings.TrimSpace(task.Artifact) != "team/accounts/acme-renewal-summary.md" {
		t.Fatalf("the Issue's own artifact must stick, got %q", task.Artifact)
	}
}

// A lane keeps full control of its own path: the container rule must not leak
// into ordinary child tasks.
func TestLaneStillRecordsItsOwnArtifact(t *testing.T) {
	b := newTestBroker(t)
	ensureTestMemberAccess(b, "team", "analyst", "Analyst")
	const issueID = "JOURN-3"
	now := time.Date(2026, 11, 10, 4, 24, 27, 0, time.UTC).Format(time.RFC3339)
	b.mu.Lock()
	b.tasks = append(b.tasks,
		teamTask{ID: issueID, Title: "Issue", Channel: "team", Owner: "cos",
			TaskType: "issue", status: "in_progress", CreatedBy: "cos", CreatedAt: now, UpdatedAt: now},
		teamTask{ID: "JOURN-30", Title: "Lane", Channel: "team", Owner: "analyst",
			ParentIssueID: issueID, status: "in_progress", CreatedAt: now, UpdatedAt: now},
	)
	b.mu.Unlock()

	if _, err := b.MutateTask(TaskPostRequest{
		Action: "submit_for_review", ID: "JOURN-30", Channel: "team", CreatedBy: "analyst",
		ArtifactPath: "team/accounts/acme-renewal-quote.md",
	}); err != nil {
		t.Fatalf("a lane must record its own deliverable: %v", err)
	}
	if task := b.TaskByID("JOURN-30"); strings.TrimSpace(task.Artifact) != "team/accounts/acme-renewal-quote.md" {
		t.Fatalf("lane artifact not recorded, got %q", task.Artifact)
	}
}

// The approve button posts to /tasks/{id}/decision, a second write path for the
// same field. It must refuse the same lie, or the container rule is enforced
// only where nobody approves.
func TestDecisionApproveRefusesALanePathAsTheIssueArtifact(t *testing.T) {
	b := newTestBroker(t)
	const issueID = "JOURN-4"
	now := time.Date(2026, 11, 10, 4, 24, 51, 0, time.UTC).Format(time.RFC3339)
	b.mu.Lock()
	b.tasks = append(b.tasks,
		teamTask{ID: issueID, Title: "Issue", Channel: "team", Owner: "cos",
			TaskType: "issue", status: "review", CreatedBy: "cos", CreatedAt: now, UpdatedAt: now},
		teamTask{ID: "JOURN-40", Title: "Lane", Channel: "team", Owner: "analyst",
			ParentIssueID: issueID, status: "done", Artifact: "team/accounts/acme-renewal-quote.md",
			CreatedAt: now, UpdatedAt: now},
	)
	parent := b.taskByIDLocked(issueID)
	parent.LifecycleState = LifecycleStateDecision
	parent.pipelineStage = "decision"
	b.mu.Unlock()

	err := b.recordTaskDecisionInternal(issueID, "approve", decisionActor{slug: "you", isHuman: true}, "",
		DecisionDelivery{ArtifactPath: "team/accounts/acme-renewal-quote.md"})
	if err == nil {
		t.Fatal("approving an Issue with one of its lanes' files must be refused")
	}
	for _, want := range []string{"container", "JOURN-40"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("the refusal must name %q, got %q", want, err.Error())
		}
	}
	if task := b.TaskByID(issueID); task.LifecycleState == LifecycleStateApproved {
		t.Fatal("a refused approve must not terminalize the Issue")
	}
	// The Approve button is a second door onto the same container rule, and an
	// attempt is the event an operator needs to count. Until this row existed
	// the decision path refused silently: guard exercise 2026-11-10 left three
	// refusals with no trace on the task-endpoint side, the fix covered those
	// three call sites, and this one — the one a human clicks — stayed dark
	// (noted in that run's observations §7).
	rows := 0
	for _, action := range b.Actions() {
		if action.Kind != "guard_refused" || action.RelatedID != issueID {
			continue
		}
		rows++
		if !strings.Contains(action.Summary, "team/accounts/acme-renewal-quote.md") ||
			!strings.Contains(action.Summary, "JOURN-40") {
			t.Fatalf("the row must name the path and the lane that owns it, got %q", action.Summary)
		}
	}
	if rows != 1 {
		t.Fatalf("a refused approve must leave exactly one guard_refused row, got %d", rows)
	}
}

// The row is for refusals, not for every approve: the same button on a
// container that names no lane file must stay quiet, or the counter stops
// meaning anything.
func TestDecisionApproveWithoutALanePathLeavesNoGuardRow(t *testing.T) {
	b := newTestBroker(t)
	const issueID = "JOURN-6"
	now := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC).Format(time.RFC3339)
	b.mu.Lock()
	b.tasks = append(b.tasks,
		teamTask{ID: issueID, Title: "Issue", Channel: "team", Owner: "cos",
			TaskType: "issue", status: "review", CreatedBy: "cos", CreatedAt: now, UpdatedAt: now},
		teamTask{ID: "JOURN-60", Title: "Lane", Channel: "team", Owner: "analyst",
			ParentIssueID: issueID, status: "done", Artifact: "team/accounts/acme-renewal-quote.md",
			CreatedAt: now, UpdatedAt: now},
	)
	parent := b.taskByIDLocked(issueID)
	parent.LifecycleState = LifecycleStateDecision
	parent.pipelineStage = "decision"
	b.mu.Unlock()

	// No delivery: the Issue ships no file of its own, its lane already did.
	if err := b.recordTaskDecisionInternal(issueID, "approve", decisionActor{slug: "you", isHuman: true}, "",
		DecisionDelivery{}); err != nil {
		t.Fatalf("approving a container on its lane's delivery must work: %v", err)
	}
	for _, action := range b.Actions() {
		if action.Kind == "guard_refused" && action.RelatedID == issueID {
			t.Fatalf("a clean approve must not leave a guard_refused row: %+v", action)
		}
	}
	if task := b.TaskByID(issueID); task.LifecycleState != LifecycleStateApproved {
		t.Fatalf("the approve must terminalize the Issue, got %s", task.LifecycleState)
	}
}

// Journey 12 (core-loop-runs/2026-11-10-journey12): refused the lane path, cos
// answered "a container cannot cite a lane's deliverable" and then closed the
// Issue from the task endpoint with verification=not_required. No decision card
// was raised, the human was never asked, and the delivered Issue was stamped
// hollow_completion=true. A container with delivered lanes has exactly one
// closing route: submit_for_review → decision card → the human.
func TestContainerWithDeliveredLanesCannotCloseFromTheTaskEndpoint(t *testing.T) {
	b := newTestBroker(t)
	const issueID = "JOURN-5"
	now := time.Date(2026, 11, 10, 8, 0, 39, 0, time.UTC).Format(time.RFC3339)
	b.mu.Lock()
	b.tasks = append(b.tasks,
		teamTask{ID: issueID, Title: "Issue", Channel: "team", Owner: "cos",
			TaskType: "issue", status: "in_progress", CreatedBy: "cos", CreatedAt: now, UpdatedAt: now},
		teamTask{ID: "JOURN-50", Title: "Lane A", Channel: "team", Owner: "analyst",
			ParentIssueID: issueID, status: "done", Artifact: "team/accounts/acme-renewal-quote.md",
			CreatedAt: now, UpdatedAt: now},
		teamTask{ID: "JOURN-51", Title: "Lane B", Channel: "team", Owner: "ae",
			ParentIssueID: issueID, status: "done", Artifact: "team/emails/acme-renewal-followup.md",
			CreatedAt: now, UpdatedAt: now},
	)
	b.mu.Unlock()

	_, err := b.MutateTask(TaskPostRequest{
		Action: "complete", ID: issueID, Channel: "team", CreatedBy: "cos",
		Verification: "not_required", VerificationReason: "Container issue: its two lanes already hold the deliverables.",
	})
	if err == nil {
		t.Fatal("an Issue whose lanes delivered must not close from the task endpoint")
	}
	for _, want := range []string{"container", "submit_for_review", "may not close it from here"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("the refusal must name %q, got %q", want, err.Error())
		}
	}
	task := b.TaskByID(issueID)
	if strings.EqualFold(task.status, "done") {
		t.Fatalf("the Issue must not be closed: status=%q lifecycle=%q", task.status, task.LifecycleState)
	}
	if task.HollowCompletion {
		t.Fatal("a delivered container must never be recorded hollow")
	}
}

// A container whose lanes shipped nothing is still closable: there is nothing to
// hand the human. Only delivered lanes route through the decision card.
func TestEmptyContainerStillClosesFromTheTaskEndpoint(t *testing.T) {
	b := newTestBroker(t)
	const issueID = "JOURN-6"
	now := time.Date(2026, 11, 10, 8, 0, 39, 0, time.UTC).Format(time.RFC3339)
	b.mu.Lock()
	b.tasks = append(b.tasks,
		teamTask{ID: issueID, Title: "Issue", Channel: "team", Owner: "cos",
			TaskType: "issue", status: "in_progress", CreatedBy: "cos", CreatedAt: now, UpdatedAt: now},
		teamTask{ID: "JOURN-60", Title: "Lane", Channel: "team", Owner: "analyst",
			ParentIssueID: issueID, status: "canceled", CreatedAt: now, UpdatedAt: now},
	)
	b.mu.Unlock()

	if _, err := b.MutateTask(TaskPostRequest{
		Action: "complete", ID: issueID, Channel: "team", CreatedBy: "cos",
		Verification: "not_required", VerificationReason: "both lanes were canceled; nothing shipped",
	}); err != nil {
		t.Fatalf("a container whose lanes shipped nothing must still close: %v", err)
	}
	task := b.TaskByID(issueID)
	if !strings.EqualFold(task.status, "done") {
		t.Fatalf("expected done, got %q", task.status)
	}
	if !task.HollowCompletion {
		t.Fatal("a container that shipped nothing IS hollow: the flag has to say so")
	}
}

// The self-heal path is the third writer of a parent's artifact: a finished
// self-heal lane's file used to be stamped onto the stalled Issue. That is the
// same lie J11 caught on the submit path — a container carrying one lane's path
// as its own. The lane keeps the artifact; the Issue's stays empty.
func TestSelfHealLaneDoesNotStampItsArtifactOnTheContainer(t *testing.T) {
	b := newTestBroker(t)
	const issueID = "JOURN-7"
	now := time.Date(2026, 11, 10, 9, 20, 0, 0, time.UTC).Format(time.RFC3339)
	b.mu.Lock()
	b.tasks = append(b.tasks,
		teamTask{ID: issueID, Title: "Issue", Channel: "team", Owner: "cos",
			TaskType: "issue", status: "in_progress", LifecycleState: "running",
			CreatedBy: "cos", CreatedAt: now, UpdatedAt: now},
		teamTask{ID: "JOURN-70", Title: "Self-heal missing capability", Channel: "team", Owner: "cos",
			ParentIssueID: issueID, PipelineID: "incident", status: "done",
			Artifact:  "team/accounts/acme-renewal-quote.md",
			CreatedBy: "cos", CreatedAt: now, UpdatedAt: now},
	)
	b.mu.Unlock()

	b.attachSelfHealCompletionToParentLocked(&b.tasks[1])

	if got := strings.TrimSpace(b.tasks[0].Artifact); got != "" {
		t.Fatalf("a lane's file must not become the container's own artifact, got %q", got)
	}
	if !strings.Contains(b.tasks[0].Details, "lane JOURN-70") {
		t.Fatalf("the parent note must say the deliverable belongs to the lane: %q", b.tasks[0].Details)
	}
	if got := strings.TrimSpace(b.tasks[1].Artifact); got != "team/accounts/acme-renewal-quote.md" {
		t.Fatalf("the lane keeps its own artifact, got %q", got)
	}
}
