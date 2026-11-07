//go:build windows

package workspaces

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// defaultSpawn launches the broker as a detached process so it survives its
// parent's exit, the Windows equivalent of POSIX Setpgid:
//
//   - DETACHED_PROCESS (0x8) gives the child no console at all, so closing the
//     parent's terminal or Ctrl+C on it cannot reach the broker. Stdout and
//     stderr are explicit *os.File handles, which keeps working without a
//     console.
//   - CREATE_NEW_PROCESS_GROUP (0x200) makes the child the root of its own
//     process group, mirroring the POSIX process-group semantics and letting a
//     supervisor deliver CTRL_BREAK to the broker alone.
//
// Everything else — argv, env, log redirect, PID file, readiness probe —
// lives in spawn_common.go.
//
// Job objects: a job is deliberately NOT used here. Joining the child to a job
// with JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE would terminate it the moment this
// process exits (that flag fires when the last handle closes, and the last
// handle belongs to the parent), which is exactly the opposite of surviving.
// Without that flag the job would be destroyed with the parent's handle and
// buy us nothing. A kill-the-whole-tree supervisor needs a long-lived owner
// (the broker itself or a separate daemon), so for now the trade-off is:
// survival first, and pause/kill only terminates the direct child — a
// grandchild broker tree would not be reaped.
func defaultSpawn(name string, runtimeHome string, brokerPort, webPort int) error {
	binary, err := resolveBinary()
	if err != nil {
		return fmt.Errorf("workspaces: spawn %q: resolve binary: %w", name, err)
	}

	logFile, err := openSpawnLog(runtimeHome)
	if err != nil {
		return fmt.Errorf("workspaces: spawn %q: %w", name, err)
	}
	defer func() { _ = logFile.Close() }()

	// No context watcher: the child is meant to outlive us, so there is nothing
	// that should be able to cancel it.
	newCmd := func(flags uint32) *exec.Cmd {
		c := exec.Command(binary, spawnArgs(brokerPort, webPort)...)
		c.SysProcAttr = &syscall.SysProcAttr{CreationFlags: flags, HideWindow: true}
		c.Stdout = logFile
		c.Stderr = logFile
		c.Env = spawnEnv(runtimeHome, brokerPort)
		return c
	}

	const baseFlags = windows.CREATE_NEW_PROCESS_GROUP | windows.DETACHED_PROCESS
	// CREATE_BREAKAWAY_FROM_JOB: DETACHED_PROCESS alone keeps the child inside
	// whatever job object the parent belongs to. Terminals and task runners
	// commonly put their command tree in a job that is terminated when the
	// command finishes, so the "detached" broker died seconds after the
	// resume/create command that started it returned. Breakaway is refused
	// when the job forbids it (ERROR_ACCESS_DENIED); fall back to the old flags
	// so a restrictive job degrades to the previous behavior instead of
	// failing to start the office at all.
	cmd := newCmd(baseFlags | windows.CREATE_BREAKAWAY_FROM_JOB)
	err = cmd.Start()
	if err != nil && isJobBreakawayDenied(err) {
		cmd = newCmd(baseFlags)
		err = cmd.Start()
	}
	if err != nil {
		return fmt.Errorf("workspaces: spawn %q: start process: %w", name, err)
	}

	if err := writeSpawnPID(runtimeHome, cmd.Process.Pid); err != nil {
		terminateSpawned(cmd)
		return fmt.Errorf("workspaces: spawn %q: %w", name, err)
	}

	// Wait for the broker to bind its port.
	ctx, cancel := context.WithTimeout(context.Background(), spawnStartTimeout)
	defer cancel()
	if err := waitForPort(ctx, brokerPort); err != nil {
		// Clean up the detached child so a stale broker isn't left running on
		// the workspace's port pair after a failed readiness probe.
		terminateSpawned(cmd)
		return fmt.Errorf("workspaces: spawn %q: broker did not bind port %d within %s: %w",
			name, brokerPort, spawnStartTimeout, err)
	}

	// Drop our process handle: on Windows that releases the last reference this
	// process holds, so the broker is fully independent of us.
	_ = cmd.Process.Release()
	return nil
}

// isJobBreakawayDenied reports whether CreateProcess refused the breakaway flag
// because the parent's job object does not allow it.
func isJobBreakawayDenied(err error) bool {
	return errors.Is(err, syscall.ERROR_ACCESS_DENIED)
}

// terminateSpawned kills a child that failed to become ready and waits for it
// so the process handle is closed deterministically. TerminateProcess reaches
// the direct child only — see the job-object note on defaultSpawn.
func terminateSpawned(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
	_, _ = cmd.Process.Wait()
}
