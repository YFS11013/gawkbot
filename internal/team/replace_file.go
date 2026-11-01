package team

import "github.com/nex-crm/wuphf/internal/fsutil"

// replaceFile moves src over dst through the shared retrying implementation.
// It stays a package-local name because every write in this package goes
// through it and the call sites read better without the qualifier.
func replaceFile(src, dst string) error {
	return fsutil.ReplaceFile(src, dst)
}
