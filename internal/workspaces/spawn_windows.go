//go:build windows

package workspaces

import "errors"

// errSpawnNotSupportedOnWindows is returned by Spawn on Windows. Multi-
// workspace broker spawning relies on POSIX detached process groups
// (Setpgid + flock-coordinated PID files); the Windows port is tracked as
// a follow-up. Returning a typed error lets the orchestrator surface a
// clean message instead of a Windows cross-build failure.
var errSpawnNotSupportedOnWindows = errors.New(
	"workspaces: multi-workspace broker spawning is not yet supported on Windows",
)

// spawnFn is the function used to launch a broker subprocess. It exists on
// every platform so the orchestrator tests compile and can exercise the
// Create/Resume paths by substituting their own launcher — only the default
// differs, because Windows has nothing to launch yet.
var spawnFn = defaultSpawn

// Spawn starts a sibling broker process for the workspace identified by name.
// On Windows the default returns errSpawnNotSupportedOnWindows; the other
// orchestrator paths (List, Switch, Pause, Doctor) stay available.
func Spawn(name string, runtimeHome string, brokerPort, webPort int) error {
	return spawnFn(name, runtimeHome, brokerPort, webPort)
}

func defaultSpawn(string, string, int, int) error {
	return errSpawnNotSupportedOnWindows
}
