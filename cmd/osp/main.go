// Command osp runs Osprey from a terminal on Windows.
//
// Windows starts Osprey.exe, a GUI program, without a console: a terminal
// doesn't wait for it, doesn't show what it prints, and gets no exit code
// from it. This console program ships beside Osprey.exe as osp.exe, and as
// osprey.com, which a terminal runs before Osprey.exe of the same name. It
// runs Osprey.exe with its arguments, passes on what Osprey prints, and
// exits as Osprey does. Osprey still goes to the background when it opens
// a window, as from a terminal anywhere: OSPREY_TERMINAL tells it that it
// was typed in one.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func main() {
	self, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "osprey:", err)
		os.Exit(1)
	}
	app := filepath.Join(filepath.Dir(self), "Osprey")
	if runtime.GOOS == "windows" {
		app += ".exe"
	}
	os.Exit(run(app, os.Args[1:], isTerminal(os.Stdout), os.Stdout, os.Stderr))
}

// run runs app with args and returns its exit code. With terminal, it
// tells app that it was typed in a terminal.
func run(app string, args []string, terminal bool, stdout, stderr io.Writer) int {
	cmd := exec.Command(app, args...)
	if terminal {
		cmd.Env = append(os.Environ(), "OSPREY_TERMINAL=1")
	}
	// Through pipes, rather than this program's own handles, so that what
	// Osprey prints reaches the console as this console program writes it,
	// Unicode included, whatever the console's code page.
	cmd.Stdout, cmd.Stderr = struct{ io.Writer }{stdout}, struct{ io.Writer }{stderr}
	err := cmd.Run()
	var exit *exec.ExitError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &exit):
		return exit.ExitCode()
	}
	fmt.Fprintln(stderr, "osprey:", err)
	return 1
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}
