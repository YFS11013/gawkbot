package commands

import (
	"path/filepath"
	"testing"

	"github.com/nex-crm/wuphf/internal/config"
)

func withTempConfigHome(t *testing.T, f func()) {
	t.Helper()
	// Windows resolves the home directory from USERPROFILE, and ConfigPath
	// reads WUPHF_RUNTIME_HOME / WUPHF_CONFIG_PATH first: setting HOME alone
	// let /config set write the developer's real ~/.wuphf/config.json.
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("WUPHF_RUNTIME_HOME", dir)
	t.Setenv("WUPHF_CONFIG_PATH", filepath.Join(dir, ".wuphf", "config.json"))
	f()
}

func TestConfigSetActionProviderPersists(t *testing.T) {
	withTempConfigHome(t, func() {
		result := Dispatch("/config set action_provider composio", "text", 0)
		if result.ExitCode != 0 {
			t.Fatalf("expected success, got exit=%d err=%q output=%q", result.ExitCode, result.Error, result.Output)
		}
		if result.Output != "Set action_provider = composio" {
			t.Fatalf("unexpected output %q", result.Output)
		}

		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("load config: %v", err)
		}
		if cfg.ActionProvider != "composio" {
			t.Fatalf("expected action_provider composio, got %q", cfg.ActionProvider)
		}
		if got := config.ResolveActionProvider(); got != "composio" {
			t.Fatalf("expected resolved action provider composio, got %q", got)
		}
	})
}

func TestConfigSetBlueprintPersists(t *testing.T) {
	withTempConfigHome(t, func() {
		result := Dispatch("/config set blueprint niche-crm", "text", 0)
		if result.ExitCode != 0 {
			t.Fatalf("expected success, got exit=%d err=%q output=%q", result.ExitCode, result.Error, result.Output)
		}
		if result.Output != "Set blueprint = niche-crm" {
			t.Fatalf("unexpected output %q", result.Output)
		}

		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("load config: %v", err)
		}
		if got := cfg.ActiveBlueprint(); got != "niche-crm" {
			t.Fatalf("expected active blueprint niche-crm, got %q", got)
		}
		if cfg.Pack != "" {
			t.Fatalf("expected legacy pack alias to remain unset on new writes, got %q", cfg.Pack)
		}
	})
}

func TestConfigShowIncludesResolvedActionProvider(t *testing.T) {
	withTempConfigHome(t, func() {
		if err := config.Save(config.Config{ActionProvider: "composio"}); err != nil {
			t.Fatalf("save config: %v", err)
		}

		result := Dispatch("/config show", "text", 0)
		if result.ExitCode != 0 {
			t.Fatalf("expected success, got exit=%d err=%q output=%q", result.ExitCode, result.Error, result.Output)
		}
		if want := "Action provider: composio"; !contains(result.Output, want) {
			t.Fatalf("expected %q in output %q", want, result.Output)
		}
		if want := "Blueprint:"; !contains(result.Output, want) {
			t.Fatalf("expected %q in output %q", want, result.Output)
		}
	})
}
