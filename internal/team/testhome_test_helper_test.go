package team

import (
	"os"
	"os/exec"
	"runtime"
	"testing"
)

// setTestHome points every home lookup at dir.
//
// HOME is what Unix reads; Windows ignores it and reads USERPROFILE. A test
// that sets only HOME therefore passes on Unix while silently reading the
// developer's real ~/.config, ~/.codex and ~/.wuphf on Windows — which is how
// a machine with no seeded opencode.json made the per-bot config merge lose
// the base config, and how a real codex credential nearly got printed into a
// failure message. Setting both keeps the isolation the test meant to have.
func setTestHome(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
}

// linkDirForTest creates link pointing at target, as a symbolic link where the
// OS allows one and a junction on Windows where it does not.
//
// os.Symlink on Windows needs developer mode or an elevated process, so a test
// that builds a "the path we were handed is not the real one" fixture failed on
// every normal desktop. A junction needs no privilege and is resolved by the
// same filepath.EvalSymlinks call the production code makes.
func linkDirForTest(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err == nil {
		return
	} else if runtime.GOOS != "windows" {
		t.Fatalf("symlink %s -> %s: %v", link, target, err)
	}
	if out, jerr := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput(); jerr != nil {
		t.Skipf("no directory link available here: symlink and junction both failed (%v, %s)", jerr, out)
	}
}
