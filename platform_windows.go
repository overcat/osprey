package main

import (
	"io"
	"log"
	"os"
	"syscall"
)

// detachAttr starts a process without the console it was typed in.
func detachAttr() *syscall.SysProcAttr {
	const detachedProcess = 0x00000008
	return &syscall.SysProcAttr{CreationFlags: detachedProcess | syscall.CREATE_NEW_PROCESS_GROUP}
}

// runFromBundle does nothing: Windows has no app bundles.
func runFromBundle() {}

var (
	kernel32          = syscall.NewLazyDLL("kernel32.dll")
	procAttachConsole = kernel32.NewProc("AttachConsole")
	procFreeConsole   = kernel32.NewProc("FreeConsole")
)

// console is what attachConsole attached: whether it did, and whether the
// log writes to the console.
var console struct{ attached, log bool }

// attachConsole gives what the command line prints the console of the
// terminal it was typed in. mygo builds Osprey as a GUI program, which
// Windows starts without a console, so what it prints would go nowhere;
// output redirected to a file or a pipe stays where it goes. A terminal
// doesn't wait for a GUI program, though: its next prompt may come before
// the output, and its exit code is not the program's.
func attachConsole() {
	if os.Getenv("OSPREY_CHILD") != "" {
		return // detach gave it somewhere to write, and nothing to say
	}
	stdout, stderr := usable(os.Stdout), usable(os.Stderr)
	if stdout && stderr {
		return
	}
	const attachParentProcess = uintptr(^uint32(0)) // ATTACH_PARENT_PROCESS, (DWORD)-1
	if ok, _, _ := procAttachConsole.Call(attachParentProcess); ok == 0 {
		return // not started from a terminal
	}
	console.attached = true
	out, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0)
	if err != nil {
		return
	}
	if !stdout {
		os.Stdout = out
	}
	if !stderr {
		os.Stderr = out
		log.SetOutput(out)
		console.log = true
	}
}

// releaseConsole lets go of the console attachConsole attached, before the
// window opens: a program attached to a console closes with it.
func releaseConsole() {
	if !console.attached {
		return
	}
	procFreeConsole.Call()
	if console.log {
		log.SetOutput(io.Discard)
	}
	console.attached, console.log = false, false
}

// usable reports whether f can be written to: a GUI program's standard
// output and error can't, unless they were redirected.
func usable(f *os.File) bool {
	_, err := f.Stat()
	return err == nil
}
