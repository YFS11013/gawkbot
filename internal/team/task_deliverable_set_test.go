package team

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Journey 23 (core-loop-runs/2026-11-13-journey23): JOURN-2's definition listed
// two deliverables. The task recorded one of them as its artifact, so the
// digest, and the card, only ever saw that one. The human sent the work back
// over the PUBLIC page; the model rewrote it; the digest did not move, the next
// card announced "None of those files has changed since this work was sent
// back", and the model made the digest move by appending a trailing space to
// the internal page and reverting it one commit later. The set has to be every
// file the definition names.
func TestDeliverableSetCoversEveryDeclaredFile(t *testing.T) {
	b := newHumanNoteTestBroker(t)
	wikiRoot := filepath.Join(t.TempDir(), "wiki")
	b.wikiWorker = NewWikiWorker(NewRepoAt(wikiRoot, filepath.Join(t.TempDir(), "wiki.bak")), b)

	const internalRel = "team/reviews/j23-internal.md"
	const publicRel = "team/reviews/j23-public.md"
	for _, rel := range []string{internalRel, publicRel} {
		abs := filepath.Join(wikiRoot, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte("v1 "+rel+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	b.mu.Lock()
	b.tasks = append(b.tasks, teamTask{
		ID: "JOURN-23", Channel: "general", Owner: "cos", CreatedBy: "cos",
		Title: "Write the two review pages", status: "review", reviewState: "ready_for_review",
		Artifact: internalRel,
		Definition: &TaskDefinition{
			Goal:            "two pages",
			ArtifactPath:    internalRel,
			Deliverables:    []TaskDeliverable{{Name: internalRel}, {Name: publicRel}, {Name: "renewal brief"}},
			SuccessCriteria: []string{"both pages land"},
		},
	})
	b.mu.Unlock()

	b.mu.Lock()
	paths := b.deliverablePathsLocked(b.taskByIDLocked("JOURN-23"))
	b.mu.Unlock()
	if len(paths) != 2 {
		t.Fatalf("the set must be both declared files (a plain-language deliverable name carries no path), got %v", paths)
	}

	if err := b.RecordTaskDecisionWithComment("JOURN-23", string(RecordDecisionRequestChanges), "the public note talks about internals", "you"); err != nil {
		t.Fatalf("send back: %v", err)
	}
	if unchanged, _ := b.sentBackDeliverablesUnchanged("JOURN-23"); !unchanged {
		t.Fatal("nothing was edited yet; the set must read as unchanged")
	}

	// The rework lands in the file the card does NOT name as the artifact.
	publicAbs := filepath.Join(wikiRoot, filepath.FromSlash(publicRel))
	if err := os.WriteFile(publicAbs, []byte("v2 plain language\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if unchanged, _ := b.sentBackDeliverablesUnchanged("JOURN-23"); unchanged {
		t.Fatal("a rewrite of the other declared file is a change; the set must see it")
	}

	reopenInDecision(t, b, "JOURN-23")
	cards := decisionCards(b)
	if len(cards) == 0 {
		t.Fatal("want a decision card")
	}
	body := cards[len(cards)-1].Question
	for _, rel := range []string{internalRel, publicRel} {
		if !strings.Contains(body, rel) {
			t.Fatalf("the card must name %s as part of what is being accepted, got %q", rel, body)
		}
	}
	if strings.Contains(body, "None of those files has changed") {
		t.Fatalf("the rework changed a file; the card must not claim nothing moved, got %q", body)
	}
}
