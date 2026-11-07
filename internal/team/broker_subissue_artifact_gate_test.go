package team

import (
	"errors"
	"strings"
	"testing"
)

// TestArtifactGate_BindsOnSubIssueOfDefinedParent pins the Journey 1 hole:
// only the parent Issue carried a Definition, so JOURN-24 closed done with the
// three renewal emails still in a notebook. A sub-issue inherits the parent's
// contract, so its done transition must carry a delivered artifact too.
func TestArtifactGate_BindsOnSubIssueOfDefinedParent(t *testing.T) {
	b := newCompletionHookBroker(t)
	parentID := completionHookCreateDefinedTask(t, b)

	child, err := b.MutateTask(TaskPostRequest{
		Action:        "create",
		Channel:       "team",
		Title:         "Draft the renewal emails",
		Owner:         "eng",
		CreatedBy:     "cos",
		ParentIssueID: parentID,
	})
	if err != nil {
		t.Fatalf("create sub-issue: %v", err)
	}
	startPlanning(t, b, child.Task.ID)

	err = finishTask(t, b, child.Task.ID, "")
	var mutationErr *TaskMutationError
	if !errors.As(err, &mutationErr) || mutationErr.Kind != TaskMutationArtifactRequired {
		t.Fatalf("expected TaskMutationArtifactRequired for the sub-issue, got %v", err)
	}
	if !strings.Contains(mutationErr.Message, "sub-issue") {
		t.Fatalf("gate error must say why it bound: %q", mutationErr.Message)
	}
	if task := b.TaskByID(child.Task.ID); strings.EqualFold(strings.TrimSpace(task.status), "done") {
		t.Fatalf("gate must roll back the sub-issue's done transition, got %q", task.status)
	}
}

// TestScheduleNoTraceRetry_ArmsShortRecheck checks the short re-arm: a task
// whose turn left no durable trace gets a recheck minutes away, not the hourly
// follow-up. A task with an open blocking request is left alone.
func TestScheduleNoTraceRetry_ArmsShortRecheck(t *testing.T) {
	b := newCompletionHookBroker(t)
	b.mu.Lock()
	b.tasks = []teamTask{{
		ID:           "JOURN-91",
		Channel:      "team",
		Title:        "Draft renewal emails",
		Owner:        "eng",
		status:       "in_progress",
		StalledTurns: 1,
		Definition:   &TaskDefinition{Goal: "draft"},
	}}
	b.mu.Unlock()

	if err := b.ScheduleNoTraceRetry("JOURN-91"); err != nil {
		t.Fatalf("ScheduleNoTraceRetry: %v", err)
	}
	b.mu.Lock()
	var found *schedulerJob
	for i := range b.scheduler {
		if b.scheduler[i].Kind == "recheck" && b.scheduler[i].TargetID == "JOURN-91" {
			found = &b.scheduler[i]
		}
	}
	b.mu.Unlock()
	if found == nil {
		t.Fatal("no recheck job armed for the no-trace turn")
	}

	// A blocking human request means the owner is not the blocker.
	b.mu.Lock()
	b.requests = append(b.requests, humanInterview{
		ID:       "req-1",
		Kind:     "approval",
		Status:   "pending",
		Channel:  "team",
		From:     "eng",
		Question: "Approve the send?",
		Blocking: true,
	})
	b.scheduler = nil
	b.mu.Unlock()
	if err := b.ScheduleNoTraceRetry("JOURN-91"); err != nil {
		t.Fatalf("ScheduleNoTraceRetry with request: %v", err)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, job := range b.scheduler {
		if job.Kind == "recheck" && job.TargetID == "JOURN-91" {
			t.Fatal("armed a retry while a blocking request waits on the human")
		}
	}
}
