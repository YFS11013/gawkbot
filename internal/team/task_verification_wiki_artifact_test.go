package team

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Journey 14 (core-loop-runs/2026-11-10-journey14): the human's decision
// approve carries no artifact_path - the card names the deliverable the task
// already recorded. The wiki_artifact check read only the request body, so a
// lane that had recorded its artifact still failed its own approve with
// "no artifact_path on the completion". The task's recorded artifact is the
// declaration; the body's value still wins when present.
func TestWikiArtifactApproveUsesTheRecordedArtifact(t *testing.T) {
	home := t.TempDir()
	t.Setenv("WUPHF_RUNTIME_HOME", home)
	rel := "team/accounts/acme-renewal-quote.md"
	path := filepath.Join(home, ".wuphf", "wiki", filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("# Renewal quote\n"), 0o644); err != nil {
		t.Fatalf("write deliverable: %v", err)
	}

	b := newTestBroker(t)
	const taskID = "JOURN-93"
	created := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	now := time.Now().UTC().Format(time.RFC3339)
	b.mu.Lock()
	b.tasks = append(b.tasks, teamTask{
		ID: taskID, Title: "Lane", Channel: testTeamRoom, Owner: "cos",
		status: "review", CreatedBy: "cos", CreatedAt: created, UpdatedAt: now,
		Artifact:     rel,
		Verification: &TaskVerification{Kind: taskVerificationKindWikiArtifact, Spec: rel, Required: true},
	})
	b.mu.Unlock()

	if err := b.gateTaskCompletionVerification(TaskPostRequest{Action: "approve", ID: taskID}); err != nil {
		t.Fatalf("approve must accept the artifact the task already recorded: %v", err)
	}
	task := b.TaskByID(taskID)
	if task.VerificationResult == nil || !task.VerificationResult.Pass {
		t.Fatalf("the check must pass against the recorded artifact, got %+v", task.VerificationResult)
	}
	if !strings.Contains(task.VerificationResult.Detail, filepath.FromSlash(rel)) {
		t.Fatalf("the detail must name the file it checked, got %q", task.VerificationResult.Detail)
	}
}

// A task that never recorded an artifact still fails closed: the card cannot
// vouch for a deliverable the task never claimed.
func TestWikiArtifactApproveWithoutAnyArtifactFailsClosed(t *testing.T) {
	home := t.TempDir()
	t.Setenv("WUPHF_RUNTIME_HOME", home)

	b := newTestBroker(t)
	const taskID = "JOURN-94"
	now := time.Now().UTC().Format(time.RFC3339)
	b.mu.Lock()
	b.tasks = append(b.tasks, teamTask{
		ID: taskID, Title: "Lane", Channel: testTeamRoom, Owner: "cos",
		status: "review", CreatedBy: "cos", CreatedAt: now, UpdatedAt: now,
		Verification: &TaskVerification{Kind: taskVerificationKindWikiArtifact, Spec: "team/accounts/x.md", Required: true},
	})
	b.mu.Unlock()

	err := b.gateTaskCompletionVerification(TaskPostRequest{Action: "approve", ID: taskID})
	if err == nil {
		t.Fatal("approve without any recorded artifact must fail the gate")
	}
	if !strings.Contains(err.Error(), "no artifact_path") {
		t.Fatalf("the error must say the artifact is missing, got %v", err)
	}
}
