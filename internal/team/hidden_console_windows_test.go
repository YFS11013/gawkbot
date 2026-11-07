//go:build windows

package team

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
)

const detachedProcess = 0x00000008

// Simulates the deployment that flashes windows: a broker started detached has
// no console, so each child it starts allocates one. The test asserts both
// halves — the detached process really has no console without the helper, and
// it ends up with a hidden one after it.
func TestEnsureHiddenConsoleHidesTheWindow(t *testing.T) {
	if os.Getenv("WUPHF_TEST_CONSOLE_MODE") != "" {
		mode := os.Getenv("WUPHF_TEST_CONSOLE_MODE")
		if mode == "hide" {
			EnsureHiddenConsole()
		}
		has, visible := consoleWindowVisible()
		fmt.Printf("VERDICT has_console=%v visible=%v\n", has, visible)
		return
	}

	run := func(mode string) string {
		cmd := exec.Command(os.Args[0], "-test.run=TestEnsureHiddenConsoleHidesTheWindow")
		cmd.Env = append(os.Environ(), "WUPHF_TEST_CONSOLE_MODE="+mode)
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: detachedProcess}
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s: %v\n%s", mode, err, out)
		}
		for _, line := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(line, "VERDICT") {
				return line
			}
		}
		t.Fatalf("%s: no verdict in output:\n%s", mode, out)
		return ""
	}

	// Control: this is the bug's precondition, not the fix's behavior. If the
	// environment ever stops producing a console-less process, the test below
	// would pass for the wrong reason.
	if got := run("none"); !strings.Contains(got, "has_console=false") {
		t.Fatalf("control: detached child unexpectedly has a console: %s", got)
	}
	if got := run("hide"); !strings.Contains(got, "has_console=true visible=false") {
		t.Fatalf("after EnsureHiddenConsole the console must exist and be hidden: %s", got)
	}
}
