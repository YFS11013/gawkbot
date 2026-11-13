package team

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nex-crm/wuphf/internal/config"
)

// Journey 28 (core-loop-runs/2026-11-13-journey28): JOURN-5 shipped two wiki
// pages and closed itself in one call. It had declared
// verification_kind=wiki_artifact pointed at ONE of them; the check ran, found
// the file the bot had just written, passed, and the J22 leaf guard excused
// the close — done with no card, review_state=not_required, nobody looking at
// the bytes. That is the exact Journey 22 outcome, reached through the
// machine-proof exception.
func TestAWikiPageIsNotItsOwnAcceptance(t *testing.T) {
	b, _, stop := newKnowledgeBroker(t)
	defer stop()
	const page = "team/reviews/j28-internal.md"
	now := time.Now().UTC().Format(time.RFC3339)

	// Two different resolvers, two roots: the done gate stats the broker's
	// worker root while the wiki_artifact check resolves under the runtime
	// home. The page has to be where each one looks.
	b.mu.Lock()
	workerRoot := b.wikiRootLocked()
	b.mu.Unlock()
	roots := []string{workerRoot, filepath.Join(config.RuntimeHomeDir(), ".wuphf", "wiki")}
	full := filepath.Join(roots[0], filepath.FromSlash(page))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, root := range roots {
		if root == "" {
			continue
		}
		seed := filepath.Join(root, filepath.FromSlash(page))
		if err := os.MkdirAll(filepath.Dir(seed), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(seed, []byte("v1\n"), 0o644); err != nil {
			t.Fatalf("seed the page: %v", err)
		}
	}
	if _, err := os.Stat(full); err != nil {
		t.Fatalf("the page must exist for the done gate: %v", err)
	}
	b.mu.Lock()
	b.tasks = append(b.tasks, teamTask{ID: "JOURN-5", Title: "Two review pages", Channel: "team",
		Owner: "cos", CreatedBy: "cos", status: "in_progress", Artifact: page,
		Verification:       &TaskVerification{Kind: taskVerificationKindWikiArtifact, Spec: page, Required: true},
		VerificationResult: &TaskVerificationResult{Pass: true, Kind: taskVerificationKindWikiArtifact, Detail: "exists", CheckedAt: now},
		CreatedAt:          now, UpdatedAt: now})
	b.mu.Unlock()

	_, err := b.MutateTask(TaskPostRequest{Action: "complete", ID: "JOURN-5", Channel: "team",
		CreatedBy: "cos", ArtifactPath: page})
	if err == nil {
		t.Fatal("a bot must not close its own wiki page: existence is not acceptance")
	}
	for _, want := range []string{"submit_for_review", "may not close it from here"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("the refusal must say %q, got %q", want, err.Error())
		}
	}
	if task := b.TaskByID("JOURN-5"); task == nil || strings.EqualFold(task.status, "done") {
		t.Fatalf("the refused close must not land: %+v", task)
	}

	// The human owns acceptance: the same close from them is the approval.
	if _, err := b.MutateTask(TaskPostRequest{Action: "complete", ID: "JOURN-5", Channel: "team",
		CreatedBy: "you", ArtifactPath: page}); err != nil {
		t.Fatalf("a human closing the wiki deliverable is the approval: %v", err)
	}
}

// The exception survives for checks that are not the bot's own writing: a
// passing command against a file outside the wiki is evidence about the world,
// so that task still closes without a card.
func TestCommandVerifiedWorkStillClosesWithoutACard(t *testing.T) {
	b, _, stop := newKnowledgeBroker(t)
	defer stop()
	now := time.Now().UTC().Format(time.RFC3339)
	worktree := t.TempDir()
	report := filepath.Join(worktree, "out", "report.txt")
	if err := os.MkdirAll(filepath.Dir(report), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(report, []byte("ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	b.mu.Lock()
	b.tasks = append(b.tasks, teamTask{ID: "JOB-9", Title: "Build the report", Channel: "team",
		Owner: "cos", CreatedBy: "cos", status: "in_progress", Artifact: "out/report.txt", WorktreePath: worktree,
		Verification:       &TaskVerification{Kind: taskVerificationKindCommand, Spec: "true", Required: true},
		VerificationResult: &TaskVerificationResult{Pass: true, Kind: taskVerificationKindCommand, Detail: "exit 0", CheckedAt: now},
		CreatedAt:          now, UpdatedAt: now})
	b.mu.Unlock()

	if _, err := b.MutateTask(TaskPostRequest{Action: "complete", ID: "JOB-9", Channel: "team",
		CreatedBy: "cos", ArtifactPath: "out/report.txt"}); err != nil {
		t.Fatalf("machine-checked work outside the wiki still closes directly: %v", err)
	}
}
