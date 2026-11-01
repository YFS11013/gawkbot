package workspaces

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// requireDirectorySymlinks skips the test where the OS will not let an ordinary
// user create a directory link.
//
// The multi-workspace layout is "~/.wuphf is a link to the active workspace".
// That is free on Unix; on Windows os.Symlink needs developer mode or an
// elevated process, and the product's own doctor reports the layout as
// unsupported there. Checking the capability rather than the OS keeps the
// coverage on machines that can do it.
func requireDirectorySymlinks(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "windows" {
		return
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("directory symlinks are unavailable here (developer mode or admin required): %v", err)
	}
}
