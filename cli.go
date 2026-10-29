package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

//go:embed mygo.json
var appConfig []byte

// version is Osprey's, as mygo.json gives it to the app bundle too.
var version = func() string {
	var config struct{ Version string }
	if err := json.Unmarshal(appConfig, &config); err != nil {
		panic(err)
	}
	return config.Version
}()

const usage = `Osprey: review code changes, file by file.

Usage:
  osp                     the changes not committed yet
  osp <base>              what this branch changed since it left <base>,
                          with what is not committed yet, as a pull request
  osp <base> <head>       what <head> changed since it left <base>
                          (or <base>...<head>, as git writes it)
  osp [<base>] <path>     only the changes in a folder or a file
  osp <old> <new>         two files, or two folders

Options:
  -C <dir>        run as if started in <dir>
  --committed     leave out what is not committed yet
  -w, --wait      return only when the window closes (git difftool needs this)
  -h, --help      show this help
  -v, --version   show the version

Examples:
  osp                     watch what an agent writes, as it writes
  osp main                review this branch before its pull request
  osp --committed         this branch's commits since main
  osp src                 only what changed in src
  osp ~/code/app          another repository
  osp HEAD~3              the last 3 commits as they are now, and what's
                          not committed

Commands:
  osp install-cli [<dir>]    put osprey and osp on your PATH, with completions
  osp uninstall-cli          take them off your PATH
  osp completion <shell>     print the completion script of fish, zsh or bash

As git difftool:
  git config --global difftool.osprey.cmd 'osprey --wait "$LOCAL" "$REMOTE"'
  git difftool -d --tool=osprey <anything git diff takes>
`

// options are what the command line asks.
type options struct {
	dir       string
	committed bool
	wait      bool
	args      []string
}

var (
	errHelp    = errors.New("help")
	errVersion = errors.New("version")
)

// parseArgs reads the command line, flags before or after the arguments.
func parseArgs(args []string) (options, error) {
	var o options
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--": // the rest are arguments, the paths among them marked
			o.args = append(o.args, args[i:]...)
			return o, nil
		case a == "-C" || a == "--dir":
			if i+1 >= len(args) {
				return o, fmt.Errorf("%s needs a directory", a)
			}
			i++
			o.dir = args[i]
		case strings.HasPrefix(a, "--dir="):
			o.dir = strings.TrimPrefix(a, "--dir=")
		case a == "--committed":
			o.committed = true
		case a == "--wait" || a == "-w":
			o.wait = true
		case a == "-h" || a == "--help" || a == "help":
			return o, errHelp
		case a == "-v" || a == "--version":
			return o, errVersion
		case strings.HasPrefix(a, "-") && a != "-":
			return o, fmt.Errorf("unknown option %s", a)
		default:
			o.args = append(o.args, a)
		}
	}
	return o, nil
}

// detach starts Osprey again in the background and reports whether it
// did, so that the terminal it was typed in is free at once, as VS Code's
// code command does.
func detach(args []string) bool {
	if os.Getenv("OSPREY_CHILD") != "" || !typedInTerminal() {
		return false
	}
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	cmd := exec.Command(exe, args...)
	cmd.Env = append(os.Environ(), "OSPREY_CHILD=1")
	cmd.SysProcAttr = detachAttr()
	if err := cmd.Start(); err != nil {
		return false
	}
	cmd.Process.Release()
	return true
}

// typedInTerminal reports whether Osprey was started from a terminal.
// On Windows, osp.exe runs it through pipes, and says so with
// OSPREY_TERMINAL.
func typedInTerminal() bool {
	return isTerminal(os.Stdout) || os.Getenv("OSPREY_TERMINAL") != ""
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}
