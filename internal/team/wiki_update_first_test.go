package team

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The routing predicate must catch the v3 duplicate shapes and nothing else.
func TestSlugsSimilarForUpdateFirst(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		// v3 disk-truth duplicates ([20:15]) — must fold.
		{"acme-corp-briefing", "acme-corp-brief", true},
		{"playbook-renewal-outreach-playbook", "renewal-outreach-playbook", true},
		{"account-brief-corti-labs", "corti-labs", true},
		{"account-brief-corti-labs", "corti-labs-account-brief", true},
		// Single-token slugs score artificially high on Jaro-Winkler;
		// they are never routed (TestWikiWorkerConcurrentEnqueue shape).
		{"agenta", "agentb", false},
		{"task", "tasks", false},
		// Numeric series are intentional, never duplicates.
		{"investor-update-week-23", "investor-update-week-24", false},
		{"q3-report-2026", "q3-report-2025", false},
		// Unrelated multi-token slugs stay distinct.
		{"acme-corp-brief", "brightline-renewal-plan", false},
	}
	for _, tc := range cases {
		if got := slugsSimilarForUpdateFirst(tc.a, tc.b); got != tc.want {
			t.Errorf("slugsSimilarForUpdateFirst(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
		if got := slugsSimilarForUpdateFirst(tc.b, tc.a); got != tc.want {
			t.Errorf("slugsSimilarForUpdateFirst(%q, %q) = %v, want %v (symmetry)", tc.b, tc.a, got, tc.want)
		}
	}
}

// System and human authors are exempt: sequential decision ids
// (team/decisions/OFFICE-295.md / OFFICE-296.md) are similar BY DESIGN.
func TestRouteAgentCreateToSimilarSlug_AuthorExemptions(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "team", "decisions"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "team", "decisions", "office-295.md"), []byte("# d\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, author := range []string{"system", "human", ""} {
		path, mode := routeBotCreateToSimilarSlug(dir, author, "team/decisions/office-296.md", "create")
		if path != "team/decisions/office-296.md" || mode != "create" {
			t.Errorf("author %q: routed to (%s, %s); exempt authors must pass through", author, path, mode)
		}
	}
	// A bot author in the same situation is ALSO passed through — the
	// numeric-token guard treats sequential ids as a series.
	path, mode := routeBotCreateToSimilarSlug(dir, "eng", "team/decisions/office-296.md", "create")
	if path != "team/decisions/office-296.md" || mode != "create" {
		t.Errorf("numeric series: routed to (%s, %s); want pass-through", path, mode)
	}
}

// A bot create with a similar-slug sibling routes to append on the
// existing article; replace mode and dissimilar slugs pass through.
func TestRouteAgentCreateToSimilarSlug_Routing(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "team", "accounts"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "team", "accounts", "acme-corp-brief.md"), []byte("# Acme\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	path, mode := routeBotCreateToSimilarSlug(dir, "eng", "team/accounts/acme-corp-briefing.md", "create")
	if path != "team/accounts/acme-corp-brief.md" || mode != "append_section" {
		t.Errorf("similar create: got (%s, %s), want append on the existing brief", path, mode)
	}

	path, mode = routeBotCreateToSimilarSlug(dir, "eng", "team/accounts/acme-corp-briefing.md", "replace")
	if path != "team/accounts/acme-corp-briefing.md" || mode != "replace" {
		t.Errorf("replace mode must pass through, got (%s, %s)", path, mode)
	}

	path, mode = routeBotCreateToSimilarSlug(dir, "eng", "team/accounts/brightline-renewal-plan.md", "create")
	if path != "team/accounts/brightline-renewal-plan.md" || mode != "create" {
		t.Errorf("dissimilar create must pass through, got (%s, %s)", path, mode)
	}
}

// End-to-end at the worker boundary: the second similar-slug bot create
// lands as an append on the first article — one file on disk, and the B4
// fold keeps a byte-identical double-write at one commit.
func TestWikiWorkerUpdateFirstAndFold(t *testing.T) {
	worker, repo, _, teardown := newStartedWorker(t)
	defer teardown()

	if _, _, err := worker.Enqueue(context.Background(), "eng", "team/accounts/acme-corp-brief.md",
		"# Acme Corp brief\n", "create", "bot: brief"); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if _, _, err := worker.Enqueue(context.Background(), "eng", "team/accounts/acme-corp-briefing.md",
		"## Addendum\nAPPEND-MARKER\n", "create", "bot: briefing"); err != nil {
		t.Fatalf("similar create: %v", err)
	}
	worker.WaitForIdle()

	if _, err := os.Stat(filepath.Join(repo.Root(), "team", "accounts", "acme-corp-briefing.md")); !os.IsNotExist(err) {
		t.Fatalf("similar-slug create must not mint a second file (stat err=%v)", err)
	}
	body, err := os.ReadFile(filepath.Join(repo.Root(), "team", "accounts", "acme-corp-brief.md"))
	if err != nil {
		t.Fatalf("read existing: %v", err)
	}
	if !strings.Contains(string(body), "APPEND-MARKER") {
		t.Fatalf("append content missing from existing article: %q", body)
	}

	// B4: byte-identical consecutive writes fold into one commit.
	sha1, _, err := worker.Enqueue(context.Background(), "eng", "team/accounts/fold.md",
		"# Fold\n", "replace", "bot: fold")
	if err != nil {
		t.Fatalf("fold write 1: %v", err)
	}
	sha2, _, err := worker.Enqueue(context.Background(), "eng", "team/accounts/fold.md",
		"# Fold\n", "replace", "bot: fold again")
	if err != nil {
		t.Fatalf("fold write 2: %v", err)
	}
	if sha1 != sha2 {
		t.Fatalf("identical double-write must fold: sha1=%s sha2=%s", sha1, sha2)
	}
	refs, err := repo.Log(context.Background(), "team/accounts/fold.md")
	if err != nil {
		t.Fatalf("log: %v", err)
	}
	if len(refs) != 1 {
		t.Fatalf("expected one commit after the fold, got %d", len(refs))
	}
}

// A task's declared artifact is PINNED. The update-first reroute exists to stop
// bots minting near-duplicate articles, but it must never fire on a task-scoped
// write: the broker authorized that one string and the completion gate checks
// that exact file. Before the pin, the write below answered 200 with a byte
// count while appending the bytes to team/reviews/dec-smoke.md and leaving
// dec-smoke-answer.md absent — the lane then refused its own approve ("artifact
// does not exist in the wiki") with nothing else to go on.
func TestTaskArtifactWriteKeepsItsDeclaredPath(t *testing.T) {
	worker, repo, _, teardown := newStartedWorker(t)
	defer teardown()

	if _, _, err := worker.Enqueue(context.Background(), "cos", "team/reviews/dec-smoke.md",
		"# dec smoke\n", "create", "cos: smoke"); err != nil {
		t.Fatalf("seed neighbour: %v", err)
	}
	if _, _, err := worker.EnqueueTaskArtifact(context.Background(), "cos", "team/reviews/dec-smoke-answer.md",
		"# answer smoke\n", "create", "cos: answer smoke"); err != nil {
		t.Fatalf("task artifact write: %v", err)
	}
	worker.WaitForIdle()

	declared := filepath.Join(repo.Root(), "team", "reviews", "dec-smoke-answer.md")
	body, err := os.ReadFile(declared)
	if err != nil {
		t.Fatalf("declared artifact was not written: %v", err)
	}
	if !strings.Contains(string(body), "answer smoke") {
		t.Fatalf("declared artifact holds %q", body)
	}
	neighbour, err := os.ReadFile(filepath.Join(repo.Root(), "team", "reviews", "dec-smoke.md"))
	if err != nil {
		t.Fatalf("read neighbour: %v", err)
	}
	if strings.Contains(string(neighbour), "answer smoke") {
		t.Fatalf("task artifact leaked into the similar-slug neighbour: %q", neighbour)
	}
	// The sibling case still rewrites through the ordinary path, so the pin
	// did not disable update-first for bot writes generally.
	_, mode := resolveWikiWriteTarget(repo.Root(), wikiWriteRequest{
		Slug: "eng", Path: "team/reviews/dec-smoke-followup.md", Mode: "create",
	})
	if mode != "append_section" {
		t.Fatalf("unpinned bot create of a similar slug must still reroute, got mode=%q", mode)
	}
}
