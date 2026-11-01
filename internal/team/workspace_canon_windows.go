//go:build windows

package team

import (
	"os"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

// fileNameNormalized asks GetFinalPathNameByHandle for the normalized (fully
// resolved) form; x/sys does not export the constant.
const fileNameNormalized = 0x0

// finalPath returns the path a handle on abs actually reaches, falling back to
// abs when the directory is gone or unreadable. GetFinalPathNameByHandle is the
// only way to see through a junction: filepath.EvalSymlinks does not follow
// them, so without this a workspace whose runtime home is a junction is never
// marked active.
func finalPath(abs string) string {
	f, err := os.Open(abs)
	if err != nil {
		return abs
	}
	defer f.Close()
	buf := make([]uint16, syscall.MAX_LONG_PATH)
	n, err := windows.GetFinalPathNameByHandle(
		windows.Handle(f.Fd()),
		&buf[0],
		uint32(len(buf)),
		fileNameNormalized,
	)
	if err != nil || n == 0 || int(n) > len(buf) {
		return abs
	}
	return trimExtendedPrefix(syscall.UTF16ToString(buf[:n]))
}

// trimExtendedPrefix turns `\\?\C:\dir` into `C:\dir` and `\\?\UNC\srv\share`
// into `\\srv\share`, so the value compares equal to the paths callers supply.
func trimExtendedPrefix(p string) string {
	const (
		extended    = `\\?\`
		extendedUNC = `\\?\UNC\`
	)
	switch {
	case strings.HasPrefix(p, extendedUNC):
		return `\\` + p[len(extendedUNC):]
	case strings.HasPrefix(p, extended):
		return p[len(extended):]
	}
	return p
}
