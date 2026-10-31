//go:build windows

package team

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// processAlive reports whether a pid still exists. tasklist exits 0 whether or
// not it matched, so the pid string has to be found in the output.
func processAlive(t *testing.T, pid string) bool {
	t.Helper()
	out, err := exec.Command("tasklist", "/FI", "PID eq "+pid, "/NH").Output()
	if err != nil {
		t.Fatalf("tasklist: %v", err)
	}
	return strings.Contains(string(out), pid)
}

// Killing the pid Go spawned must take the whole tree with it. The bots launch
// opencode through the npm .cmd shim, so a plain TerminateProcess left the real
// Bun child running and holding the turn's stdout pipe; cmd.Wait then blocked
// until that orphan died on its own. On 2026-10-30 that turned a 25-minute turn
// timeout into an 8.5-hour stall (three cos turns, EPIPE at 08:00).
func TestTerminateHeadlessProcessPIDKillsGrandchild(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "child.pid")
	script := filepath.Join(dir, "spawn.cmd")
	content := "@echo off\r\n" +
		"start \"\" /b powershell -NoProfile -Command \"$PID | Set-Content -Encoding ascii '" + pidFile + "'; Start-Sleep -Seconds 120\"\r\n" +
		"ping -n 120 127.0.0.1 >nul\r\n"
	if err := os.WriteFile(script, []byte(content), 0o600); err != nil {
		t.Fatalf("write shim: %v", err)
	}

	cmd := exec.Command("cmd", "/c", script)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start shim: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	var grandchild string
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if raw, err := os.ReadFile(pidFile); err == nil {
			grandchild = strings.TrimSpace(string(raw))
			if grandchild != "" {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	if grandchild == "" {
		t.Fatalf("shim never recorded the grandchild pid")
	}
	if !processAlive(t, grandchild) {
		t.Fatalf("grandchild %s not running before the kill", grandchild)
	}

	terminateHeadlessProcess(cmd)

	deadline = time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if !processAlive(t, grandchild) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("grandchild %s survived terminateHeadlessProcess: the shim was killed but the tree was not", grandchild)
}
