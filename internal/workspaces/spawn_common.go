package workspaces

// spawn_common.go holds the platform-independent half of broker spawning:
// the readiness probe, the environment/argv the child is launched with, the
// log/PID plumbing, and the Spawn entry point. Each platform only supplies
// defaultSpawn, which decides HOW the detached child is created
// (spawn_unix.go: Setpgid, spawn_windows.go: DETACHED_PROCESS).

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/nex-crm/wuphf/internal/runtimebin"
)

const (
	spawnStartTimeout  = 30 * time.Second
	spawnPollInterval  = 200 * time.Millisecond
	spawnBrokerLogName = "broker.log"
	spawnPIDName       = "broker.pid"
)

// spawnFn is the function used to launch a broker subprocess. Overridden in
// tests to record calls without actually spawning.
var spawnFn = defaultSpawn

// Spawn starts a sibling broker process for the workspace identified by name,
// using brokerPort and webPort. The child is detached so it survives its
// parent's exit (POSIX: its own process group; Windows: DETACHED_PROCESS).
// Stdout and stderr are redirected to <runtimeHome>/.wuphf/logs/broker.log.
//
// Spawn returns when the child's broker port accepts HTTP HEAD requests, or
// when spawnStartTimeout elapses (error).
func Spawn(name string, runtimeHome string, brokerPort, webPort int) error {
	return spawnFn(name, runtimeHome, brokerPort, webPort)
}

// spawnArgs is the argv handed to a freshly spawned broker.
func spawnArgs(brokerPort, webPort int) []string {
	return []string{
		"--broker-port", strconv.Itoa(brokerPort),
		"--web-port", strconv.Itoa(webPort),
	}
}

// spawnEnv inherits the current environment but overrides the
// workspace-specific vars. HOME is explicitly preserved so LLM CLI auth
// (codex, claude, opencode) continues to read from the real user home — NOT
// from runtimeHome.
func spawnEnv(runtimeHome string, brokerPort int) []string {
	env := os.Environ()
	env = appendOrReplace(env, "WUPHF_RUNTIME_HOME", runtimeHome)
	env = appendOrReplace(env, "WUPHF_BROKER_PORT", strconv.Itoa(brokerPort))
	return env
}

// openSpawnLog creates the log directory if needed and opens the broker log in
// append mode. The caller owns the returned file and must close it once the
// child has inherited the handle.
func openSpawnLog(runtimeHome string) (*os.File, error) {
	logDir := filepath.Join(runtimeHome, ".wuphf", "logs")
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return nil, fmt.Errorf("mkdir logs: %w", err)
	}
	logPath := filepath.Join(logDir, spawnBrokerLogName)
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open log %s: %w", logPath, err)
	}
	return f, nil
}

// writeSpawnPID records the child's PID so reconciliation and Pause can find a
// broker that outlived the process that started it.
func writeSpawnPID(runtimeHome string, pid int) error {
	pidPath := filepath.Join(runtimeHome, ".wuphf", spawnPIDName)
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(pid)), 0o600); err != nil {
		return fmt.Errorf("write pid %s: %w", pidPath, err)
	}
	return nil
}

// waitForPort polls brokerPort with HTTP HEAD until it responds or ctx expires.
func waitForPort(ctx context.Context, port int) error {
	url := fmt.Sprintf("http://127.0.0.1:%d/", port)
	client := &http.Client{Timeout: spawnPollInterval}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		req, _ := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(spawnPollInterval):
		}
	}
}

// resolveBinary returns the path to the running wuphf binary using
// os.Executable, falling back to runtimebin.LookPath("wuphf").
func resolveBinary() (string, error) {
	if exe, err := os.Executable(); err == nil && exe != "" {
		return exe, nil
	}
	return runtimebin.LookPath("wuphf")
}
