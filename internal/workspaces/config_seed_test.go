package workspaces

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// The broker resolves its runtime once, at process start. Seeding the config
// after Spawn is therefore the same as not seeding it at all — the first
// session runs on defaults. This pins the pre-spawn contract.
func TestSeedWorkspaceConfigCopiesProviderAndMemory(t *testing.T) {
	srcHome := t.TempDir()
	t.Setenv("WUPHF_RUNTIME_HOME", srcHome)
	if err := os.MkdirAll(filepath.Join(srcHome, ".wuphf"), 0o700); err != nil {
		t.Fatal(err)
	}
	source := []byte(`{"llm_provider":"opencode","llm_provider_priority":["opencode"],"memory_backend":"markdown","company_name":"Nex"}`)
	if err := os.WriteFile(filepath.Join(srcHome, ".wuphf", "config.json"), source, 0o600); err != nil {
		t.Fatal(err)
	}

	dstHome := t.TempDir()
	if err := seedWorkspaceConfig(dstHome); err != nil {
		t.Fatalf("seed: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(dstHome, ".wuphf", "config.json"))
	if err != nil {
		t.Fatalf("read seeded config: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("seeded config is not JSON: %v", err)
	}
	if got["llm_provider"] != "opencode" {
		t.Fatalf("llm_provider = %v, want opencode", got["llm_provider"])
	}
	if got["memory_backend"] != "markdown" {
		t.Fatalf("memory_backend = %v, want markdown", got["memory_backend"])
	}
	if _, ok := got["company_name"]; ok {
		t.Fatalf("company identity must not be inherited: %v", got)
	}
}

func TestSeedWorkspaceConfigLeavesAnExistingFileAlone(t *testing.T) {
	srcHome := t.TempDir()
	t.Setenv("WUPHF_RUNTIME_HOME", srcHome)
	if err := os.MkdirAll(filepath.Join(srcHome, ".wuphf"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcHome, ".wuphf", "config.json"), []byte(`{"llm_provider":"codex"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	dstHome := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dstHome, ".wuphf"), 0o700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dstHome, ".wuphf", "config.json")
	if err := os.WriteFile(target, []byte(`{"llm_provider":"custom"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := seedWorkspaceConfig(dstHome); err != nil {
		t.Fatalf("seed: %v", err)
	}
	raw, _ := os.ReadFile(target)
	if string(raw) != `{"llm_provider":"custom"}` {
		t.Fatalf("existing config was overwritten: %s", raw)
	}
}

// The registry lives under the operator's home by default. A test (or an
// embedded run) that must not touch it sets WUPHF_SPACES_DIR — os.UserHomeDir
// ignores HOME on Windows, so the Unix idiom alone is not enough.
func TestSpacesDirHonorsExplicitOverride(t *testing.T) {
	want := t.TempDir()
	t.Setenv("WUPHF_SPACES_DIR", want)
	got, err := spacesDir()
	if err != nil {
		t.Fatalf("spacesDir: %v", err)
	}
	if got != want {
		t.Fatalf("spacesDir = %q, want %q", got, want)
	}
}
