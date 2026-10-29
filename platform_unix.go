//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// detachAttr starts a process in a session of its own, so that closing
// the terminal doesn't close Osprey.
func detachAttr() *syscall.SysProcAttr { return &syscall.SysProcAttr{Setsid: true} }

// attachConsole does nothing: a program started from a terminal writes to
// it already. See platform_windows.go.
func attachConsole() {}

// releaseConsole does nothing, as attachConsole.
func releaseConsole() {}

// runFromBundle runs Osprey again from its app bundle when it started
// through a link to it, as osp is: macOS knows a program's app, and so its
// name in the menu bar and its icon in the Dock, by the path it runs from.
func runFromBundle() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	real, err := filepath.EvalSymlinks(exe)
	if err != nil || real == exe || !strings.Contains(real, ".app/Contents/MacOS/") {
		return
	}
	syscall.Exec(real, append([]string{real}, os.Args[1:]...), os.Environ())
}
