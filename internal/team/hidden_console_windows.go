//go:build windows

package team

import (
	"sync"
	"syscall"
)

var (
	hiddenConsoleOnce sync.Once

	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	procAllocConsole     = kernel32.NewProc("AllocConsole")
	procGetConsoleWindow = kernel32.NewProc("GetConsoleWindow")

	user32              = syscall.NewLazyDLL("user32.dll")
	procShowWindow      = user32.NewProc("ShowWindow")
	procIsWindowVisible = user32.NewProc("IsWindowVisible")
)

const swHide = 0

// EnsureHiddenConsole gives the process a console when it has none, then hides
// that console's window.
//
// Why this exists: a process started detached (the workspace orchestrator
// starts brokers with DETACHED_PROCESS) has no console, so every console
// program it starts gets a brand-new console window. Per-spawn flags like
// CREATE_NO_WINDOW cover the spawns someone remembered to wrap — bot turns and
// `command` verification — but not git, wiki writes, or the next spawn path
// added later. Allocating one hidden console at startup flips the default: a
// child inherits it and never needs a window, whatever code spawns it.
//
// A process that already has a console (a CLI run from a terminal) is left
// alone apart from hiding the window when the console has none visible. This
// is a no-op on every other platform.
func EnsureHiddenConsole() {
	hiddenConsoleOnce.Do(func() {
		hwnd, _, _ := procGetConsoleWindow.Call()
		if hwnd != 0 {
			// A console already exists. It may be the terminal the human is
			// typing in (classic conhost returns that window here), and hiding
			// it would make their terminal vanish. Leave it alone: children
			// inherit it either way, so no child allocates one.
			return
		}
		procAllocConsole.Call()
		if hwnd, _, _ = procGetConsoleWindow.Call(); hwnd != 0 {
			procShowWindow.Call(hwnd, uintptr(swHide))
		}
	})
}

// consoleWindowVisible reports whether the current process owns a console
// window that is on screen. Used by tests; harmless in production.
func consoleWindowVisible() (bool, bool) {
	hwnd, _, _ := procGetConsoleWindow.Call()
	if hwnd == 0 {
		return false, false // no console at all
	}
	vis, _, _ := procIsWindowVisible.Call(hwnd)
	return true, vis != 0
}
