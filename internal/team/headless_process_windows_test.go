//go:build windows

package team

import (
	"fmt"
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
	if err := startHeadlessProcess(cmd); err != nil {
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

// A turn ends when its tool exits, and nothing the tool started may outlive it.
// That is not what taskkill buys: it only runs when someone calls it, and a
// shim that exits on its own leaves node/bun holding the turn's stdout pipe,
// its sqlite database, and a CPU. The job object owns that boundary, so the
// leftovers are gone the moment the tool that started them exits.
func TestHeadlessJobKillsLeftoversWhenTheToolExits(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "leftover.pid")
	script := filepath.Join(dir, "spawn-and-exit.cmd")
	content := "@echo off\r\n" +
		"start \"\" /b powershell -NoProfile -Command \"$PID | Set-Content -Encoding ascii '" + pidFile + "'; Start-Sleep -Seconds 120\"\r\n" +
		"for /l %%i in (1,1,200) do (\r\n" +
		"  if exist \"" + pidFile + "\" goto out\r\n" +
		"  ping -n 2 127.0.0.1 >nul\r\n" +
		")\r\n" +
		":out\r\n" +
		"exit /b 0\r\n"
	if err := os.WriteFile(script, []byte(content), 0o600); err != nil {
		t.Fatalf("write shim: %v", err)
	}

	cmd := exec.Command("cmd", "/c", script)
	if err := startHeadlessProcess(cmd); err != nil {
		t.Fatalf("start shim: %v", err)
	}
	// The shim exits by itself: the ordinary end of a turn, no kill involved.
	if err := cmd.Wait(); err != nil {
		t.Fatalf("wait for shim: %v", err)
	}

	raw, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("shim never recorded the leftover pid: %v", err)
	}
	leftover := strings.TrimSpace(string(raw))
	if leftover == "" {
		t.Fatalf("leftover pid file is empty")
	}

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if !processAlive(t, leftover) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("leftover %s outlived the tool that started it: the turn owned no boundary it could not leave", leftover)
}

// A bot turn is a console program. Spawned by a broker that has no console of
// its own — the workspace orchestrator starts brokers DETACHED_PROCESS — it
// opens a window per tool call. The user sees a flash per turn; a recorded demo
// sees one mid-take. This asserts the child really has no console window.
func TestConfigureHeadlessProcessSuppressesConsoleWindow(t *testing.T) {
	if os.Getenv("WUPHF_TEST_CONSOLE_PROBE") == "1" {
		if hwnd, _, _ := procGetConsoleWindow.Call(); hwnd == 0 {
			fmt.Println("NO_CONSOLE_WINDOW")
		} else {
			fmt.Println("HAS_CONSOLE_WINDOW")
		}
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestConfigureHeadlessProcessSuppressesConsoleWindow")
	cmd.Env = append(os.Environ(), "WUPHF_TEST_CONSOLE_PROBE=1")
	configureHeadlessProcess(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("probe child: %v\n%s", err, out)
	}
	if strings.Contains(string(out), "HAS_CONSOLE_WINDOW") {
		t.Fatalf("spawned turn still owns a console window:\n%s", out)
	}
	if !strings.Contains(string(out), "NO_CONSOLE_WINDOW") {
		t.Fatalf("probe produced no verdict:\n%s", out)
	}
}
