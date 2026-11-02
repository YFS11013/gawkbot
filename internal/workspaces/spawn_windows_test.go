//go:build windows

package workspaces

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

// spawnHelperEnv turns the test binary into a fake broker. defaultSpawn
// re-executes os.Executable(), which under `go test` is this binary, so the
// child needs a way to recognise itself before the testing package parses its
// flags.
const spawnHelperEnv = "WUPHF_TEST_SPAWN_HELPER"

// spawnHelperReadyMarker is written to stderr (i.e. into broker.log) once the
// fake broker is listening. Its presence proves the log redirect works.
const spawnHelperReadyMarker = "spawn-helper: listening"

// spawnHelperListenFailed marks the log line emitted when the fake broker
// cannot bind its port (used by the port-collision test).
const spawnHelperListenFailed = "spawn-helper: listen failed"

// TestMain intercepts the helper invocation before flag.Parse: defaultSpawn
// passes only --broker-port/--web-port, which the testing package does not know
// and would reject with "flag provided but not defined".
func TestMain(m *testing.M) {
	if os.Getenv(spawnHelperEnv) == "1" {
		runSpawnHelper()
		return
	}
	os.Exit(m.Run())
}

// runSpawnHelper behaves like a broker: bind the requested port, answer HTTP
// HEAD, then block forever (the parent test terminates it by PID). It never
// returns.
func runSpawnHelper() {
	brokerPort, err := spawnHelperFlag(os.Args[1:], "--broker-port")
	if err != nil {
		fmt.Fprintf(os.Stderr, "spawn-helper: %v\n", err)
		os.Exit(2)
	}
	webPort, _ := spawnHelperFlag(os.Args[1:], "--web-port")

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", brokerPort))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s port=%d: %v\n", spawnHelperListenFailed, brokerPort, err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "%s broker=%d web=%d pid=%d\n",
		spawnHelperReadyMarker, brokerPort, webPort, os.Getpid())

	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})}
	if err := srv.Serve(ln); err != nil {
		fmt.Fprintf(os.Stderr, "spawn-helper: serve: %v\n", err)
		os.Exit(1)
	}
}

// spawnHelperFlag returns the value following name in args.
func spawnHelperFlag(args []string, name string) (int, error) {
	for i := 0; i+1 < len(args); i++ {
		if args[i] != name {
			continue
		}
		v, err := strconv.Atoi(args[i+1])
		if err != nil {
			return 0, fmt.Errorf("%s: %w", name, err)
		}
		return v, nil
	}
	return 0, fmt.Errorf("missing %s in %v", name, args)
}

// processIsAlive reports whether pid names a live process without reaping it.
// It opens the process with SYNCHRONIZE only, so it works on any live child
// regardless of its exit code.
func processIsAlive(t *testing.T, pid int) bool {
	t.Helper()
	h, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	event, err := windows.WaitForSingleObject(h, 0)
	if err != nil {
		return false
	}
	return event == uint32(windows.WAIT_TIMEOUT)
}

// waitForProcessExit polls until pid is gone, so teardown happens before
// t.TempDir's RemoveAll runs (the child holds the log file open).
func waitForProcessExit(t *testing.T, pid int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !processIsAlive(t, pid) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("pid %d still alive after %s", pid, timeout)
}

// killSpawnedBroker terminates a helper started by Spawn and waits for it to
// die, keeping the test's temp dirs removable.
func killSpawnedBroker(t *testing.T, pid int) {
	t.Helper()
	if proc, err := os.FindProcess(pid); err == nil {
		_ = proc.Kill()
	}
	waitForProcessExit(t, pid, 10*time.Second)
}

// waitForFileContains polls path until its contents contain substr.
func waitForFileContains(t *testing.T, path, substr string, timeout time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var last string
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil {
			last = string(data)
			if strings.Contains(last, substr) {
				return last
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("%s never contained %q; last contents:\n%s", path, substr, last)
	return ""
}

// readSpawnedPID reads the PID file Spawn wrote.
func readSpawnedPID(t *testing.T, runtimeHome string) int {
	t.Helper()
	pidPath := filepath.Join(runtimeHome, ".wuphf", spawnPIDName)
	raw, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatalf("read pid file %s: %v", pidPath, err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatalf("pid file %s holds %q: %v", pidPath, raw, err)
	}
	return pid
}

// TestSpawnWindowsDetachedHelper exercises the real defaultSpawn path (no
// spawnFn substitution): the child must be a live, detached process with a
// redirected log and a readiness-probed port.
func TestSpawnWindowsDetachedHelper(t *testing.T) {
	// The child inherits this env var via os.Environ() in defaultSpawn; the
	// parent already passed its TestMain helper check, so setting it here only
	// affects processes spawned from now on.
	t.Setenv(spawnHelperEnv, "1")

	runtimeHome := t.TempDir()
	brokerPort := freePort(t)
	webPort := freePort(t)

	if err := Spawn("win-detached", runtimeHome, brokerPort, webPort); err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	pid := readSpawnedPID(t, runtimeHome)
	t.Cleanup(func() { killSpawnedBroker(t, pid) })

	if pid == os.Getpid() {
		t.Fatalf("pid file holds the test process pid %d, expected the child's", pid)
	}
	if !processIsAlive(t, pid) {
		t.Fatalf("spawned child pid %d is not alive", pid)
	}

	logPath := filepath.Join(runtimeHome, ".wuphf", "logs", spawnBrokerLogName)
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("broker log %s: %v", logPath, err)
	}
	logBody := waitForFileContains(t, logPath, spawnHelperReadyMarker, 10*time.Second)
	if !strings.Contains(logBody, fmt.Sprintf("pid=%d", pid)) {
		t.Fatalf("log does not belong to pid %d:\n%s", pid, logBody)
	}
	if !strings.Contains(logBody, fmt.Sprintf("web=%d", webPort)) {
		t.Fatalf("log does not show the --web-port=%d argv:\n%s", webPort, logBody)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := waitForPort(ctx, brokerPort); err != nil {
		t.Fatalf("spawned child is not answering HEAD on %d: %v", brokerPort, err)
	}
}

// TestSpawnWindowsPortAlreadyInUse spawns a second broker onto a port the first
// one already owns. waitForPort only needs a responder, so Spawn returns
// without error — and the collision must not disturb the live first child. If a
// future change makes Spawn fail here instead, the error has to name the port.
func TestSpawnWindowsPortAlreadyInUse(t *testing.T) {
	t.Setenv(spawnHelperEnv, "1")

	runtimeHome := t.TempDir()
	brokerPort := freePort(t)
	webPort := freePort(t)

	if err := Spawn("win-first", runtimeHome, brokerPort, webPort); err != nil {
		t.Fatalf("first Spawn: %v", err)
	}
	firstPID := readSpawnedPID(t, runtimeHome)
	t.Cleanup(func() { killSpawnedBroker(t, firstPID) })

	err := Spawn("win-second", runtimeHome, brokerPort, freePort(t))
	if err != nil {
		if !strings.Contains(err.Error(), strconv.Itoa(brokerPort)) {
			t.Fatalf("second Spawn error does not name port %d: %v", brokerPort, err)
		}
		t.Logf("second Spawn rejected the occupied port: %v", err)
	}

	if !processIsAlive(t, firstPID) {
		t.Fatalf("first broker (pid %d) died after the colliding spawn", firstPID)
	}

	logPath := filepath.Join(runtimeHome, ".wuphf", "logs", spawnBrokerLogName)
	logBody := waitForFileContains(t, logPath, spawnHelperListenFailed, 10*time.Second)
	if !strings.Contains(logBody, fmt.Sprintf("port=%d", brokerPort)) {
		t.Fatalf("collision log does not name port %d:\n%s", brokerPort, logBody)
	}
}
