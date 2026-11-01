//go:build windows

package team

import (
	"context"
	"os/exec"
	"syscall"
)

// verificationShellCommand builds the platform shell invocation for a
// command-kind definition-of-done check (task_verification.go).
//
// The spec is passed as a literal command line, not as one argv element.
// exec.Command would re-quote a spec that already contains quotes (a Windows
// path with spaces, or any spec that quotes a path), and cmd /C then hands
// the re-quoted text to the child: observed with
//
//	python "C:\repo\scripts\check.py" "team/prep/x.md"
//
// which ran as `python <cwd>\"C:\repo\...\check.py"` and failed with
// "Invalid argument" (exit 2) — the script never executed, so a correct
// deliverable could not complete. CmdLine keeps the text cmd parses
// byte-identical to the spec the task declared.
func verificationShellCommand(ctx context.Context, spec string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "cmd")
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: "cmd /C " + spec}
	return cmd
}
