package team

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Journey 22 (core-loop-runs/2026-11-13-journey22): JOURN-2 was an office task
// with two real wiki pages and review_state=not_required — the work is writing,
// not code, so taskNeedsStructuredReview said false. The model called complete
// after the plan was approved and the Issue went straight to done: no decision
// card, no human. Its sibling in the same run raised a card for the same kind of
// work, only because it happened to say submit_for_review. The guard makes the
// verb irrelevant: a bot that shipped a file asks the human.
func TestOfficeDeliveryCannotBeClosedWithoutTheHuman(t *testing.T) {
	b := newHumanNoteTestBroker(t)
	wikiRoot := filepath.Join(t.TempDir(), "wiki")
	b.wikiWorker = NewWikiWorker(NewRepoAt(wikiRoot, filepath.Join(t.TempDir(), "wiki.bak")), b)

	const artifactRel = "team/reviews/j22-public.md"
	abs := filepath.Join(wikiRoot, filepath.FromSlash(artifactRel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte("1. Faster reviews\n2. Clearer errors\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	b.tasks = append(b.tasks, teamTask{
		ID:            "task-office-1",
		Channel:       "general",
		Title:         "Write the public note",
		Owner:         "eng",
		TaskType:      "issue",
		ExecutionMode: "office",
		status:        "in_progress",
	})

	// The bot ships a real file and closes the task itself.
	_, err := b.MutateTask(TaskPostRequest{
		Action: "complete", ID: "task-office-1", Channel: "general",
		CreatedBy: "eng", ArtifactPath: artifactRel, Details: "both pages are up",
	})
	var mutationErr *TaskMutationError
	if !errors.As(err, &mutationErr) || mutationErr.Kind != TaskMutationConflict {
		t.Fatalf("a bot closing its own delivered office task must be refused, got %v", err)
	}
	if !strings.Contains(mutationErr.Message, "submit_for_review") {
		t.Fatalf("the refusal must point at the review verb, got %q", mutationErr.Message)
	}
	if got := b.TaskByID("task-office-1"); got == nil || got.status == "done" {
		t.Fatalf("the refusal must roll the transition back, got %+v", got)
	}

	// The route the refusal names works, and it leaves the task in review for
	// the card instead of closing it.
	if _, err := b.MutateTask(TaskPostRequest{
		Action: "submit_for_review", ID: "task-office-1", Channel: "general",
		CreatedBy: "eng", ArtifactPath: artifactRel, Details: "both pages are up",
	}); err != nil {
		t.Fatalf("submit_for_review: %v", err)
	}
	if got := b.TaskByID("task-office-1"); got == nil || got.status != "review" {
		t.Fatalf("submit_for_review must park the task in review, got %+v", got)
	}

	// The human still closes it, from the same endpoint.
	if _, err := b.MutateTask(TaskPostRequest{
		Action: "approve", ID: "task-office-1", Channel: "general",
		CreatedBy: "you", Details: "looks right",
	}); err != nil {
		t.Fatalf("a human must still be able to close it, got %v", err)
	}
	if got := b.TaskByID("task-office-1"); got == nil || got.status != "done" {
		t.Fatalf("human approve must land done, got %+v", got)
	}
}
