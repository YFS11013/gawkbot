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
