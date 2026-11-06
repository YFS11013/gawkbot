package team

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

// countingProvider records how many articles reached the LLM. The point of the
// on-disk verdict cache is that a restarted process asks about an unchanged
// article zero times.
type countingProvider struct {
	calls atomic.Int64
}

func (p *countingProvider) AskIsSkill(_ context.Context, _, _, _ string) (bool, SkillFrontmatter, string, string, error) {
	p.calls.Add(1)
	return false, SkillFrontmatter{}, "", "", nil
}

func newScannerCacheWiki(t *testing.T, articles map[string]string) (*Broker, string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "wiki")
	repo := NewRepoAt(root, filepath.Join(t.TempDir(), "wiki.bak"))
	if err := repo.Init(context.Background()); err != nil {
		t.Fatalf("init wiki repo: %v", err)
	}
	for rel, body := range articles {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", abs, err)
		}
		if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	b := newTestBroker(t)
	worker := NewWikiWorker(repo, b)
	ctx, cancel := context.WithCancel(context.Background())
	worker.Start(ctx)
	t.Cleanup(func() {
		cancel()
		worker.Stop()
		<-worker.Done()
	})
	b.mu.Lock()
	b.wikiWorker = worker
	b.mu.Unlock()
	return b, root
}

func runScanPass(t *testing.T, b *Broker, p llmProvider) {
	t.Helper()
	scanner := NewSkillScanner(b, p, 100)
	b.SetSkillScanner(scanner)
	if _, err := b.compileWikiSkills(context.Background(), "", true, "manual"); err != nil {
		t.Fatalf("compileWikiSkills: %v", err)
	}
}

func TestSkillScanner_VerdictCacheSurvivesRestart(t *testing.T) {
	t.Setenv("WUPHF_SKILL_COMPILE_COOLDOWN", "0s")
	b, root := newScannerCacheWiki(t, map[string]string{
		"team/alpha.md": "alpha procedure",
		"team/beta.md":  "beta procedure",
	})

	first := &countingProvider{}
	runScanPass(t, b, first)
	if got := first.calls.Load(); got != 2 {
		t.Fatalf("first pass: want 2 LLM calls, got %d", got)
	}

	cachePath := filepath.Join(root, ".git", "skills-scan-cache.json")
	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf("verdict cache not written to %s: %v", cachePath, err)
	}

	// A second scanner over the same wiki stands in for a restarted broker:
	// fresh process, empty in-memory map.
	second := &countingProvider{}
	runScanPass(t, b, second)
	if got := second.calls.Load(); got != 0 {
		t.Fatalf("after restart: want 0 LLM calls (cache hit), got %d", got)
	}

	// Editing one article must still re-judge exactly that one.
	if err := os.WriteFile(filepath.Join(root, "team", "beta.md"), []byte("beta procedure, revised"), 0o644); err != nil {
		t.Fatalf("edit article: %v", err)
	}
	third := &countingProvider{}
	runScanPass(t, b, third)
	if got := third.calls.Load(); got != 1 {
		t.Fatalf("after edit: want 1 LLM call, got %d", got)
	}
}

func TestSkillScanner_ScopedPassKeepsFullTreeCache(t *testing.T) {
	t.Setenv("WUPHF_SKILL_COMPILE_COOLDOWN", "0s")
	b, _ := newScannerCacheWiki(t, map[string]string{
		"team/alpha.md":       "alpha procedure",
		"team/customers/x.md": "customer procedure",
	})

	runScanPass(t, b, &countingProvider{})

	// Scoped pass over team/customers must not evict team/alpha.md.
	scoped := &countingProvider{}
	scanner := NewSkillScanner(b, scoped, 100)
	b.SetSkillScanner(scanner)
	if _, err := b.compileWikiSkills(context.Background(), "team/customers", true, "manual"); err != nil {
		t.Fatalf("scoped compile: %v", err)
	}
	if got := scoped.calls.Load(); got != 0 {
		t.Fatalf("scoped pass: want 0 LLM calls (already cached), got %d", got)
	}

	full := &countingProvider{}
	runScanPass(t, b, full)
	if got := full.calls.Load(); got != 0 {
		t.Fatalf("full pass after scoped: want 0 LLM calls, got %d", got)
	}
}

func TestSkillScanner_VerdictCacheRejectsWrongVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")
	if err := os.WriteFile(path, []byte(`{"version":999,"entries":{"team/a.md":"deadbeef"}}`), 0o644); err != nil {
		t.Fatalf("seed cache: %v", err)
	}
	b := newTestBroker(t)
	s := NewSkillScanner(b, &countingProvider{}, 10)
	s.loadSkillScanCache(path)
	if len(s.mtimeCache) != 0 {
		t.Fatalf("wrong-version cache must be ignored, got %d entries", len(s.mtimeCache))
	}
}

// failingProvider stands in for a provider whose response cannot be read, e.g.
// the `--format json` event stream that used to defeat the verdict decoder, or
// a call that timed out.
type failingProvider struct {
	calls atomic.Int64
}

func (p *failingProvider) AskIsSkill(_ context.Context, _, _, _ string) (bool, SkillFrontmatter, string, string, error) {
	p.calls.Add(1)
	return false, SkillFrontmatter{}, "", "", errors.New("skill_scanner: parse llm json: invalid character '{' after top-level value")
}

// A failed judgement must not be stored as a "not a skill" verdict. Caching it
// would freeze the article's classification until its content changed, which is
// exactly what happened on 2026-11-06 when every response was an event stream.
func TestSkillScanner_UnreadableVerdictIsNotCached(t *testing.T) {
	t.Setenv("WUPHF_SKILL_COMPILE_COOLDOWN", "0s")
	b, root := newScannerCacheWiki(t, map[string]string{
		"team/alpha.md": "alpha procedure",
	})

	first := &failingProvider{}
	runScanPass(t, b, first)
	if got := first.calls.Load(); got != 1 {
		t.Fatalf("first pass: want 1 attempt, got %d", got)
	}

	cachePath := filepath.Join(root, ".git", "skills-scan-cache.json")
	data, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("read cache: %v", err)
	}
	var cache struct {
		Entries map[string]string `json:"entries"`
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		t.Fatalf("decode cache: %v", err)
	}
	if _, ok := cache.Entries["team/alpha.md"]; ok {
		t.Fatalf("failed verdict must not be cached, got %v", cache.Entries)
	}

	second := &failingProvider{}
	runScanPass(t, b, second)
	if got := second.calls.Load(); got != 1 {
		t.Fatalf("second pass: want the article retried (1 attempt), got %d", got)
	}
}
