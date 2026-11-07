package team

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSeedOpencodeAuthLinksWithoutSecondCopy(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "user", "opencode", "auth.json")
	dst := filepath.Join(dir, "office", "opencode", "auth.json")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte(`{"anthropic":{"key":"x"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := seedOpencodeAuth(src, dst); err != nil {
		t.Fatalf("seed: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read office auth: %v", err)
	}
	if string(got) != `{"anthropic":{"key":"x"}}` {
		t.Fatalf("office auth = %q", got)
	}
	// Rewriting the source must be visible through the destination: a hard
	// link means the office reads the user's live credential instead of a
	// stale copy taken at boot.
	if err := os.WriteFile(src, []byte(`{"anthropic":{"key":"y"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	again, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(again) != `{"anthropic":{"key":"y"}}` {
		t.Fatalf("office auth did not follow the user's file: %q", again)
	}
}

func TestSeedOpencodeAuthKeepsExistingOfficeCopy(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "user", "auth.json")
	dst := filepath.Join(dir, "office", "auth.json")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("user"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("office"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := seedOpencodeAuth(src, dst); err != nil {
		t.Fatalf("seed: %v", err)
	}
	got, _ := os.ReadFile(dst)
	if string(got) != "office" {
		t.Fatalf("existing office auth was overwritten: %q", got)
	}
}

func TestSeedOpencodeAuthWithoutUserFileIsNotAnError(t *testing.T) {
	dir := t.TempDir()
	if err := seedOpencodeAuth(filepath.Join(dir, "missing.json"), filepath.Join(dir, "office", "auth.json")); err != nil {
		t.Fatalf("missing user auth must not fail: %v", err)
	}
}
