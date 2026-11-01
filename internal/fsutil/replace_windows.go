//go:build windows

package fsutil

import (
	"errors"
	"os"
	"syscall"
)

// Windows error codes that mean "someone else has the destination open right
// now". Go's syscall package exports only the first of them.
const (
	errSharingViolation syscall.Errno = 32
	errLockViolation    syscall.Errno = 33
)

// renameRetryable reports whether a failed rename is the "destination is open
// somewhere else" window rather than a real error. Windows reports it three
// ways depending on who holds the handle: ACCESS_DENIED when the replacement
// needs delete access the holder did not share, SHARING_VIOLATION for the
// classic open-handle case, and LOCK_VIOLATION when a byte-range lock covers
// the file.
func renameRetryable(err error) bool {
	var linkErr *os.LinkError
	if !errors.As(err, &linkErr) {
		return false
	}
	return errors.Is(linkErr.Err, syscall.ERROR_ACCESS_DENIED) ||
		errors.Is(linkErr.Err, errSharingViolation) ||
		errors.Is(linkErr.Err, errLockViolation)
}
