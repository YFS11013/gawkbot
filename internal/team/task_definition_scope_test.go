package team

import (
	"strings"
	"testing"
	"time"
)

// Journey 14 (core-loop-runs/2026-11-10-journey14 §3.2): a re-define replaces
// the whole definition, so a define that only meant to fix the verification
// check wiped the lanes' declared write scope. The scope is an authorization;
// an omitted artifact_path keeps the one already declared.
func TestDefineKeepsAnOmittedArtifactScope(t *testing.T) {
	b := newTestBroker(t)
	const taskID = "JOURN-96"
	now := time.Date(2026, 11, 10, 11, 30, 0, 0, time.UTC).Format(time.RFC3339)
	b.mu.Lock()
	b.tasks = append(b.tasks, teamTask{
		ID: taskID, Title: "Lane", Channel: testTeamRoom, Owner: "cos",
		status: "in_progress", CreatedBy: "cos", CreatedAt: now, UpdatedAt: now,
		Definition: &TaskDefinition{
			Goal:         "Standalone English one-page renewal quote for Acme.",
			ArtifactPath: "team/accounts/acme-renewal-quote.md",
		},
	})
	b.mu.Unlock()

	_, err := b.MutateTask(TaskPostRequest{
		Action: "define", ID: taskID, Channel: testTeamRoom, CreatedBy: "cos",
		Definition: &TaskDefinition{
			Goal:            "Standalone English one-page renewal quote for Acme.",
			SuccessCriteria: []string{"file team/accounts/acme-renewal-quote.md exists"},
		},
	})
	if err != nil {
		t.Fatalf("define must succeed: %v", err)
	}
	task := b.TaskByID(taskID)
	if task.Definition == nil || task.Definition.ArtifactPath != "team/accounts/acme-renewal-quote.md" {
		t.Fatalf("an omitted artifact_path must not clear the write scope, got %+v", task.Definition)
	}
}

// Journey 14 §3.1: the container guard covered the task's artifact but not the
// definition's write scope, so one define authorized the Issue to overwrite the
// very file its lane delivered. Same rule, same refusal.
func TestContainerCannotDeclareALanesFileAsItsWriteScope(t *testing.T) {
	b := newTestBroker(t)
	const parentID, laneID = "JOURN-97", "JOURN-98"
	const lanePath = "team/accounts/acme-renewal-quote.md"
	now := time.Date(2026, 11, 10, 10, 0, 0, 0, time.UTC).Format(time.RFC3339)
	b.mu.Lock()
	b.tasks = append(b.tasks,
		teamTask{
			ID: parentID, Title: "Issue", Channel: testTeamRoom, Owner: "cos",
			status: "in_progress", CreatedBy: "cos", CreatedAt: now, UpdatedAt: now,
			LifecycleState: LifecycleStateRunning,
		},
		teamTask{
			ID: laneID, Title: "Lane", Channel: testTeamRoom, Owner: "cos",
			status: "done", CreatedBy: "cos", CreatedAt: now, UpdatedAt: now,
			ParentIssueID: parentID, Artifact: lanePath,
		})
	b.mu.Unlock()

	_, err := b.MutateTask(TaskPostRequest{
		Action: "define", ID: parentID, Channel: testTeamRoom, CreatedBy: "cos",
		Definition: &TaskDefinition{
			Goal:         "Aggregate the two Acme renewal deliverables.",
			ArtifactPath: lanePath,
		},
	})
	if err == nil {
		t.Fatal("a container must not be authorized to write a lane's file")
	}
	if !strings.Contains(err.Error(), "is a container") || !strings.Contains(err.Error(), laneID) {
		t.Fatalf("the refusal must name the container rule and the lane, got %v", err)
	}
	if task := b.TaskByID(parentID); task.Definition != nil && task.Definition.ArtifactPath != "" {
		t.Fatalf("the refused define must not leave the scope applied, got %+v", task.Definition)
	}
}

// The guard's window: a lane declares its write scope the moment it is
// created, but only records `Artifact` when it completes. Guard exercise
// 2026-11-10 (core-loop-runs/2026-11-10-guards) hit exactly that window — the
// container claimed the pending lane's file and the define was accepted.
func TestContainerCannotClaimAPendingLanesWriteScope(t *testing.T) {
	b := newTestBroker(t)
	const parentID, laneID = "GUARD-9", "GUARD-14"
	const lanePath = "team/emails/guard-email.md"
	now := time.Date(2026, 11, 10, 13, 0, 0, 0, time.UTC).Format(time.RFC3339)
	b.mu.Lock()
	b.tasks = append(b.tasks,
		teamTask{
			ID: parentID, Title: "Issue", Channel: testTeamRoom, Owner: "cos",
			status: "in_progress", CreatedBy: "cos", CreatedAt: now, UpdatedAt: now,
			LifecycleState: LifecycleStateRunning,
		},
		teamTask{
			ID: laneID, Title: "Lane", Channel: testTeamRoom, Owner: "cos",
			status: "in_progress", CreatedBy: "cos", CreatedAt: now, UpdatedAt: now,
			ParentIssueID: parentID, Artifact: "",
			Definition: &TaskDefinition{Goal: "Write the email.", ArtifactPath: lanePath},
		})
	b.mu.Unlock()

	_, err := b.MutateTask(TaskPostRequest{
		Action: "define", ID: parentID, Channel: testTeamRoom, CreatedBy: "cos",
		Definition: &TaskDefinition{Goal: "Aggregate the lanes.", ArtifactPath: lanePath},
	})
	if err == nil {
		t.Fatal("a container must not claim the file a pending lane is about to write")
	}
	if !strings.Contains(err.Error(), "is a container") || !strings.Contains(err.Error(), laneID) {
		t.Fatalf("the refusal must name the container rule and the pending lane, got %v", err)
	}
}

// A canceled lane owns nothing, so the same path is free for the container.
func TestCanceledLaneWriteScopeDoesNotBlockTheContainer(t *testing.T) {
	b := newTestBroker(t)
	const parentID, laneID = "GUARD-9", "GUARD-16"
	const lanePath = "team/emails/dropped.md"
	now := time.Date(2026, 11, 10, 13, 0, 0, 0, time.UTC).Format(time.RFC3339)
	b.mu.Lock()
	b.tasks = append(b.tasks,
		teamTask{
			ID: parentID, Title: "Issue", Channel: testTeamRoom, Owner: "cos",
			status: "in_progress", CreatedBy: "cos", CreatedAt: now, UpdatedAt: now,
			LifecycleState: LifecycleStateRunning,
		},
		teamTask{
			ID: laneID, Title: "Lane", Channel: testTeamRoom, Owner: "cos",
			status: "canceled", CreatedBy: "cos", CreatedAt: now, UpdatedAt: now,
			ParentIssueID: parentID,
			Definition:    &TaskDefinition{Goal: "Write it.", ArtifactPath: lanePath},
		})
	b.mu.Unlock()

	if _, err := b.MutateTask(TaskPostRequest{
		Action: "define", ID: parentID, Channel: testTeamRoom, CreatedBy: "cos",
		Definition: &TaskDefinition{Goal: "Write it myself.", ArtifactPath: lanePath},
	}); err != nil {
		t.Fatalf("a canceled lane must not own its old scope: %v", err)
	}
}

// The same helper backs the artifact_path guard, so a container must not be
// able to restate a pending lane's declared scope as its own artifact either —
// the second write path into the same window.
func TestContainerCannotCompleteWithAPendingLanesDeclaredPath(t *testing.T) {
	b := newTestBroker(t)
	const parentID, laneID = "GUARD-20", "GUARD-21"
	const lanePath = "team/accounts/guard-quote.md"
	now := time.Date(2026, 11, 10, 13, 0, 0, 0, time.UTC).Format(time.RFC3339)
	b.mu.Lock()
	b.tasks = append(b.tasks,
		teamTask{
			ID: parentID, Title: "Issue", Channel: testTeamRoom, Owner: "cos",
			status: "in_progress", CreatedBy: "cos", CreatedAt: now, UpdatedAt: now,
			LifecycleState: LifecycleStateRunning,
		},
		teamTask{
			ID: laneID, Title: "Lane", Channel: testTeamRoom, Owner: "cos",
			status: "in_progress", CreatedBy: "cos", CreatedAt: now, UpdatedAt: now,
			ParentIssueID: parentID,
			Definition:    &TaskDefinition{Goal: "Write it.", ArtifactPath: lanePath},
		})
	b.mu.Unlock()

	_, err := b.MutateTask(TaskPostRequest{
		Action: "complete", ID: parentID, Channel: testTeamRoom, CreatedBy: "cos",
		ArtifactPath: lanePath,
	})
	if err == nil {
		t.Fatal("a container must not adopt the file a pending lane declared")
	}
	if !strings.Contains(err.Error(), "is a container") || !strings.Contains(err.Error(), laneID) {
		t.Fatalf("the refusal must name the container rule and the pending lane, got %v", err)
	}
	if task := b.TaskByID(parentID); strings.TrimSpace(task.Artifact) != "" {
		t.Fatalf("the refused completion must not record an artifact, got %q", task.Artifact)
	}
}

// A canceled lane keeps no claim, so its old scope must not trip the container
// guard on the artifact path either.
func TestCanceledLaneDeclaredPathDoesNotTripTheContainerGuard(t *testing.T) {
	b := newTestBroker(t)
	const parentID, laneID = "GUARD-22", "GUARD-23"
	const lanePath = "team/accounts/dropped.md"
	now := time.Date(2026, 11, 10, 13, 0, 0, 0, time.UTC).Format(time.RFC3339)
	b.mu.Lock()
	b.tasks = append(b.tasks,
		teamTask{
			ID: parentID, Title: "Issue", Channel: testTeamRoom, Owner: "cos",
			status: "in_progress", CreatedBy: "cos", CreatedAt: now, UpdatedAt: now,
			LifecycleState: LifecycleStateRunning,
		},
		teamTask{
			ID: laneID, Title: "Lane", Channel: testTeamRoom, Owner: "cos",
			status: "canceled", CreatedBy: "cos", CreatedAt: now, UpdatedAt: now,
			ParentIssueID: parentID,
			Definition:    &TaskDefinition{Goal: "Write it.", ArtifactPath: lanePath},
		})
	b.mu.Unlock()

	_, err := b.MutateTask(TaskPostRequest{
		Action: "complete", ID: parentID, Channel: testTeamRoom, CreatedBy: "cos",
		ArtifactPath: lanePath,
	})
	if err != nil && strings.Contains(err.Error(), "is a container") {
		t.Fatalf("a canceled lane must not own its old scope: %v", err)
	}
}

// Guard exercise 2026-11-10 fired all three container guards and left
// actions[] and broker.log completely clean, so the only trace of a bypass
// attempt was the operator's own transcript. Each refusal now leaves a
// `guard_refused` row.
func TestContainerGuardRefusalsLeaveALedgerRow(t *testing.T) {
	b := newTestBroker(t)
	const parentID, laneID = "GUARD-30", "GUARD-31"
	const lanePath = "team/accounts/guard-quote.md"
	now := time.Date(2026, 11, 10, 13, 0, 0, 0, time.UTC).Format(time.RFC3339)
	b.mu.Lock()
	b.tasks = append(b.tasks,
		teamTask{
			ID: parentID, Title: "Issue", Channel: testTeamRoom, Owner: "cos",
			status: "in_progress", CreatedBy: "cos", CreatedAt: now, UpdatedAt: now,
			LifecycleState: LifecycleStateRunning,
		},
		teamTask{
			ID: laneID, Title: "Lane", Channel: testTeamRoom, Owner: "cos",
			status: "done", CreatedBy: "cos", CreatedAt: now, UpdatedAt: now,
			ParentIssueID: parentID, Artifact: lanePath,
			LifecycleState: LifecycleStateApproved,
		})
	b.mu.Unlock()

	// 1. define's write scope
	if _, err := b.MutateTask(TaskPostRequest{
		Action: "define", ID: parentID, Channel: testTeamRoom, CreatedBy: "cos",
		Definition: &TaskDefinition{Goal: "Aggregate.", ArtifactPath: lanePath},
	}); err == nil {
		t.Fatal("expected the container scope guard to refuse")
	}
	// 2. the artifact field
	if _, err := b.MutateTask(TaskPostRequest{
		Action: "complete", ID: parentID, Channel: testTeamRoom, CreatedBy: "cos",
		ArtifactPath: lanePath,
	}); err == nil {
		t.Fatal("expected the container artifact guard to refuse")
	}
	// 3. a bot closing a container with delivered lanes
	if _, err := b.MutateTask(TaskPostRequest{
		Action: "complete", ID: parentID, Channel: testTeamRoom, CreatedBy: "cos",
		Verification: verificationNotRequired, VerificationReason: "container",
	}); err == nil {
		t.Fatal("expected the bot-close guard to refuse")
	}

	rows := 0
	for _, action := range b.Actions() {
		if action.Kind != "guard_refused" || action.RelatedID != parentID {
			continue
		}
		rows++
	}
	if rows != 3 {
		t.Fatalf("every refused container mutation must leave one guard_refused row, got %d", rows)
	}
}
