//go:build !windows

package fsutil

// renameRetryable is always false off Windows: a rename either succeeds or
// fails for a reason waiting cannot fix, so ReplaceFile makes exactly one
// attempt there.
func renameRetryable(error) bool {
	return false
}
