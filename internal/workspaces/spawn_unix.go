//go:build darwin || linux

package workspaces

import (
	"context"
	"fmt"
	"os/exec"
	"syscall"
)

// defaultSpawn launches the broker in its own process group (Setpgid: true) so
// it survives its parent's exit. Everything else — argv, env, log redirect,
// PID file, readiness probe — lives in spawn_common.go.
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

	cmd := exec.CommandContext(context.Background(), binary, spawnArgs(brokerPort, webPort)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Env = spawnEnv(runtimeHome, brokerPort)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("workspaces: spawn %q: start process: %w", name, err)
	}

	if err := writeSpawnPID(runtimeHome, cmd.Process.Pid); err != nil {
		_ = cmd.Process.Kill()
		return fmt.Errorf("workspaces: spawn %q: %w", name, err)
	}

	// Wait for the broker to bind its port.
	ctx, cancel := context.WithTimeout(context.Background(), spawnStartTimeout)
	defer cancel()
	if err := waitForPort(ctx, brokerPort); err != nil {
		// Clean up the detached child so a stale broker isn't left running on
		// the workspace's port pair after a failed readiness probe.
		_ = cmd.Process.Kill()
		return fmt.Errorf("workspaces: spawn %q: broker did not bind port %d within %s: %w",
			name, brokerPort, spawnStartTimeout, err)
	}
	return nil
}
