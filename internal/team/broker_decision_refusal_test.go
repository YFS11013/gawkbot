package team

import (
	"strings"
	"testing"
)

// Journey 14 (core-loop-runs/2026-11-10-journey14), live: clicking
// approve_decision on a lane whose definition-of-done check fails returned 409,
// but the answer had already been persisted — the card was consumed, the
// "Approved" line was posted, and the lane had no card left to retry. A refusal
// means the decision did not land, so the card must stay open.
func TestRefusedDecisionCardAnswerStaysOpen(t *testing.T) {
	b := newTestBroker(t)
	home := t.TempDir()
	t.Setenv("WUPHF_RUNTIME_HOME", home)

	const taskID = "JOURN-95"
	// Record the deliverable, then point the gate at a file that is NOT in the
	// wiki: the approve must refuse, exactly as it did on the live run.
	b.mu.Lock()
	b.tasks = append(b.tasks, teamTask{
		ID: taskID, Title: "Lane", Channel: testTeamRoom, Owner: "cos",
		status: "review", CreatedBy: "cos", CreatedAt: "2026-11-10T10:00:00Z", UpdatedAt: "2026-11-10T10:00:00Z",
		Artifact:     "team/accounts/missing.md",
		Verification: &TaskVerification{Kind: taskVerificationKindWikiArtifact, Spec: "team/accounts/missing.md", Required: true},
	})
	b.requests = append(b.requests, humanInterview{
		ID: "request-1", Kind: "decision", Status: "pending", From: "cos",
		Channel: testTeamRoom, Question: "Approve?", IssueID: taskID,
		DedupeKey: "decision:" + taskID,
		Options: []interviewOption{
			{ID: decisionChoiceApprove, Label: "Approve"},
			{ID: decisionChoiceChanges, Label: "Send back"},
		},
	})
	b.mu.Unlock()

	status, msg := b.answerRequestFromActor("you", "request-1", decisionChoiceApprove, "", "")
	if status != 409 {
		t.Fatalf("a failing gate must refuse the approve with 409, got %d (%q)", status, msg)
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	req := findRequestLocked(b.requests, "request-1")
	if !requestIsActive(req) {
		t.Fatal("a refused decision must leave the card open so the human can retry")
	}
	if req.Answered != nil {
		t.Fatalf("a refused decision must not record an answer, got %+v", req.Answered)
	}
	for _, m := range b.messages {
		if strings.Contains(m.Content, "Approved") {
			t.Fatalf("a refused decision must not announce an approval, got %q", m.Content)
		}
	}
	found := false
	for _, a := range b.actions {
		if a.Kind == "request_reopened" && a.RelatedID == "request-1" {
			found = true
		}
	}
	if !found {
		t.Fatal("the refusal must be recorded in the action log")
	}
	if task := b.taskByIDLocked(taskID); task == nil || task.Status() == "done" {
		t.Fatal("a refused approve must leave the lane where it was")
	}
	_ = home
}
