//go:build !windows

package team

// renameRetryable is always false off Windows: a rename either succeeds or
// fails for a reason waiting cannot fix, so replaceFile makes exactly one
// attempt there.
func renameRetryable(error) bool {
	return false
}
