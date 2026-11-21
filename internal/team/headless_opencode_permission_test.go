package team

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestHeadlessOpencodeNonCodingDeniesNetworkAndSubagents locks the §9.1
// permission hardening for non-coding bots: a role such as cos must not be able
// to fetch the web (invent numbers the digest cannot cite) or spawn subagents
// (which have no wuphf routing and stall the turn). Both must stay denied at the
// layer opencode actually consults, not just in the prompt.
func TestHeadlessOpencodeNonCodingDeniesNetworkAndSubagents(t *testing.T) {
	home := t.TempDir()
	setTestHome(t, home)
	t.Setenv("WUPHF_RUNTIME_HOME", home)

	configDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}

	orig := headlessOpencodeExecutablePath
	headlessOpencodeExecutablePath = func() (string, error) { return "/usr/local/bin/wuphf", nil }
	defer func() { headlessOpencodeExecutablePath = orig }()

	l := &Launcher{}
	path, err := l.writeHeadlessOpencodeMCPConfig("cos")
	if err != nil {
		t.Fatalf("writeHeadlessOpencodeMCPConfig(cos): %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated config: %v", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("generated config is not JSON: %v\n%s", err, raw)
	}
	agents, _ := cfg["agent"].(map[string]any)
	headless, _ := agents["wuphf-headless"].(map[string]any)
	perm, _ := headless["permission"].(map[string]any)
	if perm == nil {
		t.Fatalf("agent.wuphf-headless.permission missing from cos config")
	}
	for _, key := range []string{"webfetch", "task"} {
		if got := perm[key]; got != "deny" {
			t.Fatalf("permission.%s = %#v, want deny", key, got)
		}
	}
}
