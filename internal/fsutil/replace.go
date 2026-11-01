// Package fsutil holds file-system helpers whose behaviour has to be the same
// on every platform the product ships on.
package fsutil

import (
	"os"
	"path/filepath"
	"time"
)

const (
	// A rename that fails because someone else holds the destination open is
	// worth retrying for about a tenth of a second: that is how long an
	// indexer, a virus scanner or another process reading the same file keeps
	// its handle. Anything longer is a real failure and should surface.
	renameAttempts = 5
	renameBackoff  = 10 * time.Millisecond
)

// ReplaceFile moves src over dst, retrying the brief window in which another
// process still holds dst open.
//
// It exists because Windows refuses to replace a destination that any handle
// has open without FILE_SHARE_DELETE, and on a real machine that is routine
// rather than exceptional: the search indexer and the anti-virus scanner both
// open files as they are written. Unix rename has no such window, so
// renameRetryable reports false there and this is one call.
//
// The retry only covers "the destination was busy" — not a missing directory,
// a read-only volume or a permissions problem, which no amount of waiting
// fixes. When the attempts run out the original error is returned so callers
// keep reporting the real cause.
func ReplaceFile(src, dst string) error {
	var err error
	for attempt := range renameAttempts {
		if err = os.Rename(src, dst); err == nil || !renameRetryable(err) {
			return err
		}
		time.Sleep(renameBackoff * time.Duration(attempt+1))
	}
	return err
}

// WriteFileAtomic writes data to path via a uniquely named sibling and a
// ReplaceFile, so a reader either sees the old bytes or the new ones. The
// caller's directory must already exist.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	cleanup := func() { _ = os.Remove(name) }
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	if err := ReplaceFile(name, path); err != nil {
		cleanup()
		return err
	}
	return nil
}
