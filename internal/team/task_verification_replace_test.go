package team

import (
	"testing"
	"time"
)

// Journey 14 (core-loop-runs/2026-11-10-journey14): a lane was created with a
// required artifact gate whose spec was a wiki-relative path. The artifact check
// globs the task worktree, so the lane's own wiki deliverable could never pass:
// the human's approve on the lane's decision card 409'd, the model diagnosed it
// exactly ("校验卡在 artifact（工作区文件）上，而交付物是 wiki 文章") and tried to
// fix it with action=define — which was silently ignored, because a re-define
// could not touch an established gate. A gate that has already FAILED is not
// worth protecting: the re-define must be able to replace it, and the swap must
// show up in the action log.
func TestDefineReplacesAFailedVerificationGate(t *testing.T) {
	b := newTestBroker(t)
	const taskID = "JOURN-90"
	now := time.Date(2026, 11, 10, 10, 30, 0, 0, time.UTC).Format(time.RFC3339)
	b.mu.Lock()
	b.tasks = append(b.tasks, teamTask{
		ID: taskID, Title: "Lane", Channel: testTeamRoom, Owner: "cos",
		status: "in_progress", CreatedBy: "cos", CreatedAt: now, UpdatedAt: now,
		Verification:       &TaskVerification{Kind: "artifact", Spec: "team/accounts/acme-renewal-quote.md", Required: true},
		VerificationResult: &TaskVerificationResult{Pass: false, Kind: "artifact", Detail: "no non-empty artifact matches"},
	})
	b.mu.Unlock()

	_, err := b.MutateTask(TaskPostRequest{
		Action: "define", ID: taskID, Channel: testTeamRoom, CreatedBy: "cos",
		VerificationKind: "none",
		Definition: &TaskDefinition{
			Goal:            "Standalone English one-page renewal quote for Acme, published to the team wiki.",
			Deliverables:    []TaskDeliverable{{Name: "One-page renewal quote", Format: "markdown article in the wiki"}},
			SuccessCriteria: []string{"Article exists at team/accounts/acme-renewal-quote.md and is non-empty"},
			AccessNeeded:    []string{"wiki write access"},
		},
	})
	if err != nil {
		t.Fatalf("a failed gate must be replaceable by define: %v", err)
	}

	task := b.TaskByID(taskID)
	if task.Verification != nil {
		t.Fatalf("the failed artifact gate must be gone, got %+v", task.Verification)
	}
	b.mu.Lock()
	found := false
	for i := range b.actions {
		if b.actions[i].Kind == "verification_replaced" {
			found = true
		}
	}
	b.mu.Unlock()

	if !found {
		t.Fatal("the swap must be recorded in the action log")
	}
}

// A gate that has not failed stays put: the model cannot weaken its own DoD by
// re-defining. This is the behaviour the Journey 14 change deliberately kept.
func TestDefineDoesNotReplaceAPassingVerificationGate(t *testing.T) {
	b := newTestBroker(t)
	const taskID = "JOURN-91"
	now := time.Date(2026, 11, 10, 10, 30, 0, 0, time.UTC).Format(time.RFC3339)
	b.mu.Lock()
	b.tasks = append(b.tasks, teamTask{
		ID: taskID, Title: "Lane", Channel: testTeamRoom, Owner: "cos",
		status: "in_progress", CreatedBy: "cos", CreatedAt: now, UpdatedAt: now,
		Verification:       &TaskVerification{Kind: "artifact", Spec: "team/accounts/acme-renewal-quote.md", Required: true},
		VerificationResult: &TaskVerificationResult{Pass: true, Kind: "artifact", Detail: "found"},
	})
	b.mu.Unlock()

	_, err := b.MutateTask(TaskPostRequest{
		Action: "define", ID: taskID, Channel: testTeamRoom, CreatedBy: "cos",
		VerificationKind: "none",
		Definition: &TaskDefinition{
			Goal:            "Standalone English one-page renewal quote for Acme.",
			Deliverables:    []TaskDeliverable{{Name: "One-page renewal quote", Format: "markdown"}},
			SuccessCriteria: []string{"Article exists and is non-empty"},
		},
	})
	if err != nil {
		t.Fatalf("define must still succeed: %v", err)
	}
	task := b.TaskByID(taskID)
	if task.Verification == nil || task.Verification.Kind != "artifact" {
		t.Fatalf("a passing gate must survive a re-define, got %+v", task.Verification)
	}
}
