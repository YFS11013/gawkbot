//go:build !windows

package team

// EnsureHiddenConsole is a no-op outside Windows: processes there either have
// a controlling terminal or run without one by design, and neither allocates a
// visible window for children.
func EnsureHiddenConsole() {}
