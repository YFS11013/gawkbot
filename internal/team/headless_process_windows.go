//go:build windows

package team

import (
	"os"
	"os/exec"
	"strconv"
)

func configureHeadlessProcess(cmd *exec.Cmd) {}

func terminateHeadlessProcess(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	terminateHeadlessProcessPID(cmd.Process.Pid)
}

// terminateHeadlessProcessPID kills the process AND its tree.
//
// The bots run `opencode` through the npm shim (opencode.cmd -> node/bun), and
// every other headless tool has a similar shim. TerminateProcess on the pid Go
// spawned only kills the shim: the real CLI keeps running, keeps the stdout
// pipe open, and the worker's cmd.Wait blocks until that survivor happens to
// exit. That is what turned a 25-minute turn timeout into an 8.5-hour stall on
// 2026-10-30 (three cos turns: started 23:31, logged "timed out after 25m0s" at
// 08:00 the next morning when the orphaned Bun process finally died of EPIPE).
//
// taskkill /T walks the child tree. Falls back to killing just the pid when
// taskkill is unavailable (locked-down PATH).
func terminateHeadlessProcessPID(pid int) {
	if pid <= 0 {
		return
	}
	if err := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid)).Run(); err == nil {
		return
	}
	if proc, err := os.FindProcess(pid); err == nil {
		_ = proc.Kill()
	}
}
