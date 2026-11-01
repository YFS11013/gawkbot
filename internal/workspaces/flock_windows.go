//go:build windows

package workspaces

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// errFileLockUnavailable is returned by lockFileExclusiveNonBlocking when the
// lock is held elsewhere. Unix reports EWOULDBLOCK; Windows reports a lock
// violation, so it is mapped onto the same signal here.
var errFileLockUnavailable = errors.New("workspaces: file lock is held by another process")

// lockFileExclusive takes an exclusive byte-range lock on the file, blocking
// until it is available.
//
// This used to be a no-op with a note that multi-broker concurrency was out of
// scope on Windows. It was not: Registry writes are read-modify-write, and two
// workspaces saving at once renamed over each other, failing with "the process
// cannot access the file" and losing one of the two updates.
func lockFileExclusive(f *os.File) error {
	return windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0, new(windows.Overlapped))
}

// lockFileExclusiveNonBlocking takes the same lock but reports
// errFileLockUnavailable instead of waiting.
func lockFileExclusiveNonBlocking(f *os.File) error {
	flags := uint32(windows.LOCKFILE_EXCLUSIVE_LOCK | windows.LOCKFILE_FAIL_IMMEDIATELY)
	err := windows.LockFileEx(windows.Handle(f.Fd()), flags, 0, 1, 0, new(windows.Overlapped))
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		return errFileLockUnavailable
	}
	return err
}

// unlockFile releases the lock taken by either acquire path.
func unlockFile(f *os.File) error {
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, new(windows.Overlapped))
}
