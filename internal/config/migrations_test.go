package config

import (
	"os"
	"path/filepath"
	"testing"
)

func withTempHome(t *testing.T) func() {
	t.Helper()
	tmp := t.TempDir()
	// All four, because ConfigPath resolves through WUPHF_RUNTIME_HOME and
	// os.UserHomeDir reads USERPROFILE on Windows: setting HOME alone made
	// these tests save an empty Config over the developer's real
	// ~/.wuphf/config.json.
	for _, key := range []string{"HOME", "USERPROFILE", "WUPHF_RUNTIME_HOME"} {
		key := key
		old := os.Getenv(key)
		t.Cleanup(func() { _ = os.Setenv(key, old) })
	}
	_ = os.Setenv("HOME", tmp)
	_ = os.Setenv("USERPROFILE", tmp)
	_ = os.Setenv("WUPHF_RUNTIME_HOME", tmp)
	t.Setenv("WUPHF_CONFIG_PATH", filepath.Join(tmp, ".wuphf", "config.json"))
	if err := os.MkdirAll(filepath.Join(tmp, ".wuphf"), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	return func() {}
}

func TestMigrateOpenclawBridges_Empty(t *testing.T) {
	defer withTempHome(t)()
	bindings, status, err := MigrateOpenclawBridgesFromConfig()
	if err != nil {
		t.Fatalf("migration: %v", err)
	}
	if len(bindings) != 0 {
		t.Fatalf("expected no bindings, got %d", len(bindings))
	}
	if status.OpenclawBridgesMoved != 0 {
		t.Fatalf("expected 0 moved, got %d", status.OpenclawBridgesMoved)
	}
}

func TestMigrateOpenclawBridges_Happy(t *testing.T) {
	defer withTempHome(t)()
	if err := Save(Config{
		OpenclawGatewayURL: "ws://127.0.0.1:18789",
		OpenclawBridges: []OpenclawBridgeBinding{
			{SessionKey: "agent:a:demo", Slug: "openclaw-a", DisplayName: "Bot A"},
			{SessionKey: "agent:b:demo", Slug: "openclaw-b", DisplayName: "Bot B"},
		},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	bindings, status, err := MigrateOpenclawBridgesFromConfig()
	if err != nil {
		t.Fatalf("migration: %v", err)
	}
	if len(bindings) != 2 {
		t.Fatalf("expected 2 bindings returned, got %d", len(bindings))
	}
	if status.OpenclawBridgesMoved != 2 {
		t.Fatalf("expected OpenclawBridgesMoved=2, got %d", status.OpenclawBridgesMoved)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(cfg.OpenclawBridges) != 0 {
		t.Fatalf("OpenclawBridges should be cleared, got %d", len(cfg.OpenclawBridges))
	}
	// Sibling fields survive the migration untouched.
	if cfg.OpenclawGatewayURL != "ws://127.0.0.1:18789" {
		t.Fatalf("sibling field wiped: %q", cfg.OpenclawGatewayURL)
	}
}

func TestMigrateOpenclawBridges_Idempotent(t *testing.T) {
	defer withTempHome(t)()
	if err := Save(Config{
		OpenclawBridges: []OpenclawBridgeBinding{
			{SessionKey: "k", Slug: "openclaw-a"},
		},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	first, _, err := MigrateOpenclawBridgesFromConfig()
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	if len(first) != 1 {
		t.Fatalf("first call: want 1 binding, got %d", len(first))
	}

	second, status, err := MigrateOpenclawBridgesFromConfig()
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if len(second) != 0 {
		t.Fatalf("second call: want 0 bindings (already migrated), got %d", len(second))
	}
	if status.OpenclawBridgesMoved != 0 {
		t.Fatalf("second call: want 0 moved, got %d", status.OpenclawBridgesMoved)
	}
}
