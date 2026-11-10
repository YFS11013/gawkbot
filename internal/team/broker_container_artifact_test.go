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
