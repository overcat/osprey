package main

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
)

// fallbackScript runs install-cli in /usr/local/bin as an administrator,
// with the program it is given, once macOS asked for the password.
var fallbackScript = []string{
	`on run argv`,
	`do shell script "mkdir -p /usr/local/bin && " & quoted form of (item 1 of argv) & " install-cli /usr/local/bin" with prompt "Osprey wants to put the osprey and osp commands in /usr/local/bin." with administrator privileges`,
	`end run`,
}

// installFallback installs osprey and osp where this user can write in
// none of the usual directories of the PATH, as on a Mac without
// Homebrew: in /usr/local/bin, which the PATH of every macOS shell holds,
// with an administrator's password.
func installFallback(exe string) (string, error) {
	var args []string
	for _, line := range fallbackScript {
		args = append(args, "-e", line)
	}
	out, err := exec.Command("osascript", append(args, exe)...).CombinedOutput()
	out = bytes.TrimSpace(out)
	switch {
	case err == nil:
		return string(out), nil // what install-cli said
	case bytes.Contains(out, []byte("(-128)")): // the password wasn't given
		return "", errors.New("installing them in /usr/local/bin takes an administrator's password")
	}
	return "", fmt.Errorf("installing them in /usr/local/bin: %s", out)
}
