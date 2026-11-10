package team

// The guard exercise (core-loop-runs/2026-11-10-guards) found GUARD-4 replacing
// GUARD-12's delivered team/accounts/guard-quote.md (33 → 1642 bytes) with a
// mode=create write, and nothing anywhere said so. Refusing the write would
// break the legitimate shape — a routine that rewrites the same page every run —
// so the decision is to record the replacement instead: an audit row naming who
// overwrote whom.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestOverwritingAnotherTasksDeliverableLeavesARow(t *testing.T) {
	b, _, stop := newKnowledgeBroker(t)
	defer stop()
	now := time.Now().UTC().Format(time.RFC3339)
	const shared = "team/reports/shared-ledger.md"

	b.mu.Lock()
	b.tasks = append(b.tasks,
		teamTask{ID: "GUARD-12", Channel: "team", Title: "Lane A", Owner: "dwight",
			CreatedBy: "cos", status: "done", Artifact: shared, LifecycleState: LifecycleStateApproved,
			CreatedAt: now, UpdatedAt: now},
		teamTask{ID: "GUARD-4", Channel: "team", Title: "Lane B", Owner: "kevin",
			CreatedBy: "cos", status: "in_progress", LifecycleState: LifecycleStateRunning,
			Definition: &TaskDefinition{Goal: "Rewrite the ledger", ArtifactPath: shared,
				DefinedAt: now},
			CreatedAt: now, UpdatedAt: now},
	)
	b.mu.Unlock()

	// The page has to exist before the second write so the handler sees an
	// overwrite rather than a creation.
	if _, _, err := b.WikiWorker().EnqueueTaskArtifact(t.Context(), "dwight", shared, "v1\n", "create", ""); err != nil {
		t.Fatalf("seed the shared page: %v", err)
	}

	body := strings.NewReader(`{"slug":"kevin","task_id":"GUARD-4","path":"` + shared + `","content":"v2 — rewritten\n","mode":"replace"}`)
	rec := httptest.NewRecorder()
	b.handleWikiWriteTask(rec, httptest.NewRequest(http.MethodPost, "/wiki/write-task", body))
	if rec.Code != http.StatusOK {
		t.Fatalf("a write inside the declared scope must land, got %d: %s", rec.Code, rec.Body.String())
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	rows := 0
	for _, action := range b.actions {
		if action.Kind != "artifact_overwritten" || action.RelatedID != "GUARD-4" {
			continue
		}
		rows++
		for _, want := range []string{shared, "GUARD-12"} {
			if !strings.Contains(action.Summary, want) {
				t.Fatalf("the row must name %q, got %q", want, action.Summary)
			}
		}
	}
	if rows != 1 {
		t.Fatalf("exactly one overwrite row expected, got %d", rows)
	}
}

// A task writing its own page, or a page nobody delivered, is not an overwrite:
// the row would fire on every second save and stop meaning anything.
func TestRewritingAnUnownedPageLeavesNoRow(t *testing.T) {
	b, _, stop := newKnowledgeBroker(t)
	defer stop()
	now := time.Now().UTC().Format(time.RFC3339)
	const own = "team/reports/solo.md"

	b.mu.Lock()
	b.tasks = append(b.tasks, teamTask{ID: "SOLO-1", Channel: "team", Title: "Solo", Owner: "kevin",
		CreatedBy: "cos", status: "in_progress", LifecycleState: LifecycleStateRunning,
		Definition: &TaskDefinition{Goal: "Write the report", ArtifactPath: own, DefinedAt: now},
		CreatedAt:  now, UpdatedAt: now})
	b.mu.Unlock()

	if _, _, err := b.WikiWorker().EnqueueTaskArtifact(t.Context(), "kevin", own, "draft\n", "create", ""); err != nil {
		t.Fatalf("first write: %v", err)
	}
	body := strings.NewReader(`{"slug":"kevin","task_id":"SOLO-1","path":"` + own + `","content":"final\n","mode":"replace"}`)
	rec := httptest.NewRecorder()
	b.handleWikiWriteTask(rec, httptest.NewRequest(http.MethodPost, "/wiki/write-task", body))
	if rec.Code != http.StatusOK {
		t.Fatalf("second write: %d %s", rec.Code, rec.Body.String())
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	for _, action := range b.actions {
		if action.Kind == "artifact_overwritten" {
			t.Fatalf("rewriting an undelivered page is not an overwrite: %+v", action)
		}
	}
}
