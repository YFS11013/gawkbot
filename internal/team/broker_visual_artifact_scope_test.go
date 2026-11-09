package team

import (
	"errors"
	"strings"
	"testing"
)

// createScopedTask files a top-level lane that declared the one wiki file it may
// write, then clears the plan gate so the completion gate is what gets tested.
func createScopedTask(t *testing.T, b *Broker, scope string) string {
	t.Helper()
	created, err := b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: "Write the renewal quote",
		Owner: "eng", CreatedBy: "cos",
		Definition: &TaskDefinition{Goal: "One-page renewal quote", ArtifactPath: scope},
	})
	if err != nil {
		t.Fatalf("create scoped lane: %v", err)
	}
	startPlanning(t, b, created.Task.ID)
	return created.Task.ID
}

// Journey 8, D3: JOURN-17 declared team/research/acme-renewal-quote.md at
// create, then closed with the visual artifact ra_4904a7a50a981735. The gate saw
// a non-empty Artifact and passed, team/research/ stayed empty, and the run's
// "the shared wiki really gets the files" claim rested on five lanes instead of
// six. A declared scope is a contract for a file, and a visual artifact lives
// under wiki/visual-artifacts/ — outside team/**, outside the archivist, outside
// everything the scope was pinned for.
func TestVisualArtifactCannotCloseADeclaredScope(t *testing.T) {
	b := newCompletionHookBroker(t)
	const scope = "team/research/acme-renewal-quote.md"
	taskID := createScopedTask(t, b, scope)

	err := finishTask(t, b, taskID, "ra_4904a7a50a981735")
	var mutationErr *TaskMutationError
	if !errors.As(err, &mutationErr) || mutationErr.Kind != TaskMutationArtifactRequired {
		t.Fatalf("a visual artifact must not close a lane that declared a wiki path, got %v", err)
	}
	for _, want := range []string{scope, "visual artifact", "wiki/write-task", "verification=not_required"} {
		if !strings.Contains(mutationErr.Message, want) {
			t.Fatalf("refusal should mention %q, got %q", want, mutationErr.Message)
		}
	}
	if task := b.TaskByID(taskID); task == nil || strings.EqualFold(strings.TrimSpace(task.status), "done") {
		t.Fatalf("the refusal must roll the done transition back, got %+v", task)
	}
}

// The declared-scope rule keeps the same escape the artifact gate already has:
// a lane that turns out to ship no file says so, with a reason, and the
// declaration is recorded instead of silently dropped.
func TestVisualArtifactAllowedWithNotRequiredDeclaration(t *testing.T) {
	b := newCompletionHookBroker(t)
	taskID := createScopedTask(t, b, "team/research/acme-renewal-quote.md")

	if _, err := b.MutateTask(TaskPostRequest{
		Action: "complete", ID: taskID, Channel: "team", CreatedBy: "eng",
		ArtifactPath: "ra_4904a7a50a981735",
		Verification: verificationNotRequired, VerificationReason: "quote went out as a visual one-pager",
	}); err != nil {
		t.Fatalf("not_required + reason must clear the gate, got %v", err)
	}
	if cur := b.TaskByID(taskID); cur != nil && !strings.EqualFold(strings.TrimSpace(cur.status), "done") {
		if _, err := b.MutateTask(TaskPostRequest{
			Action: "approve", ID: taskID, Channel: "team", CreatedBy: "cos",
			ArtifactPath: "ra_4904a7a50a981735",
			Verification: verificationNotRequired, VerificationReason: "quote went out as a visual one-pager",
		}); err != nil {
			t.Fatalf("approve with not_required + reason: %v", err)
		}
	}
	task := b.TaskByID(taskID)
	if task == nil {
		t.Fatal("task vanished")
	}
	if strings.EqualFold(strings.TrimSpace(task.status), "done") && !strings.Contains(task.ArtifactNotRequired, "visual one-pager") {
		t.Fatalf("the not-required reason must be recorded, got %q", task.ArtifactNotRequired)
	}
}

// Lanes without a declared scope keep the old behaviour: a visual artifact is a
// legitimate delivery there (that was the whole J6/J7 path), so the new rule
// must not start refusing work it has no contract to hold anyone to.
func TestVisualArtifactStillClosesUnscopedLane(t *testing.T) {
	b := newCompletionHookBroker(t)
	created, err := b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: "Sketch the renewal one-pager",
		Owner: "eng", CreatedBy: "cos",
	})
	if err != nil {
		t.Fatalf("create unscoped task: %v", err)
	}
	startPlanning(t, b, created.Task.ID)

	err = finishTask(t, b, created.Task.ID, "ra_4904a7a50a981735")
	if err != nil && strings.Contains(err.Error(), "declared the write scope") {
		t.Fatalf("an unscoped lane must not be held to a scope it never declared: %v", err)
	}
}
