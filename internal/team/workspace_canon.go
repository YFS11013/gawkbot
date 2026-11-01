package team

import (
	"path/filepath"
	"strings"
)

// canonicalWorkspaceRuntimeHome resolves a runtime home to the path the OS
// actually reaches, so two spellings of the same directory compare equal.
func canonicalWorkspaceRuntimeHome(raw string) (string, bool) {
	path := strings.TrimSpace(raw)
	if path == "" {
		return "", false
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path), true
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err == nil {
		abs = resolved
	} else {
		abs = filepath.Clean(abs)
	}
	return canonicalizeFinalPath(abs), true
}

// canonicalizeFinalPath lets each platform finish the job where EvalSymlinks
// stops short: on Windows a directory junction is a reparse point that
// EvalSymlinks returns unchanged, so two workspaces pointing at the same
// directory through a junction look different without this.
func canonicalizeFinalPath(abs string) string {
	return finalPath(abs)
}
