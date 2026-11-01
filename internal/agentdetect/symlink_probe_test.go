package agentdetect

import (
	"os"
	"path/filepath"
	"testing"
)

// requireFileSymlinks skips a test where the OS will not let an ordinary user
// create a file link. Windows needs developer mode or an elevated process for
// os.Symlink; the behaviour under test (never following a link out of a
// project folder) cannot be exercised without one.
func requireFileSymlinks(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("x"), 0o600); err != nil {
		t.Fatalf("write target: %v", err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "link")); err != nil {
		t.Skipf("file symlinks are unavailable here (developer mode or admin required): %v", err)
	}
}
