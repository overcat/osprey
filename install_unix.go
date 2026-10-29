//go:build !windows

package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// installCLI links osprey and osp to this program in dir, or the first
// directory of the usual ones on the PATH it can write, with the shells'
// completions beside them, and says what it did.
func installCLI(dir string) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if exe, err = filepath.EvalSymlinks(exe); err != nil {
		return "", err
	}
	dirs := []string{dir}
	if dir == "" {
		home, _ := os.UserHomeDir()
		dirs = []string{"/opt/homebrew/bin", "/usr/local/bin", filepath.Join(home, ".local", "bin"), filepath.Join(home, "bin")}
	} else if !isDir(dir) {
		return "", fmt.Errorf("%s is not a directory", dir)
	}
	var lastErr error
	for _, d := range dirs {
		if !isDir(d) {
			continue
		}
		ok := true
		for _, name := range names {
			link := filepath.Join(d, name)
			if target, err := os.Readlink(link); err == nil && target == exe {
				continue
			}
			if _, err := os.Lstat(link); err == nil {
				if target, _ := os.Readlink(link); !strings.Contains(strings.ToLower(target), "osprey") {
					lastErr = fmt.Errorf("%s already exists", link)
					ok = false
					break
				}
				os.Remove(link)
			}
			if err := os.Symlink(exe, link); err != nil {
				lastErr = err
				ok = false
				break
			}
		}
		if ok {
			said := "osprey and osp are in " + d
			if shells := installCompletions(d); len(shells) > 0 {
				said += ", with the completions of " + andList(shells)
			}
			return said + ".", nil
		}
	}
	if lastErr == nil {
		lastErr = errors.New("no directory to install into: give one, as osp install-cli ~/bin")
	}
	return "", lastErr
}

// installCompletions puts the shells' completion scripts beside the
// commands in bin, as a package manager does: in the share folder of the
// same prefix, where the shells look for them, without touching their
// configuration. fish reads its folder there whenever fish is installed;
// zsh and bash read theirs where it exists. It returns the shells whose
// scripts it installed.
func installCompletions(bin string) []string {
	share := filepath.Join(filepath.Dir(bin), "share")
	if filepath.Base(bin) != "bin" || !isDir(share) {
		return nil
	}
	_, fishErr := exec.LookPath("fish")
	var shells []string
	for _, s := range []struct {
		shell, dir string
		create     bool
		files      map[string][]string // the commands each file completes
	}{
		{"fish", filepath.Join(share, "fish", "vendor_completions.d"), fishErr == nil, map[string][]string{"osprey.fish": {"osprey"}, "osp.fish": {"osp"}}},
		{"zsh", filepath.Join(share, "zsh", "site-functions"), false, map[string][]string{"_osprey": names}},
		{"bash", filepath.Join(share, "bash-completion", "completions"), false, map[string][]string{"osprey": names, "osp": names}},
	} {
		if !isDir(s.dir) && (!s.create || os.MkdirAll(s.dir, 0o755) != nil) {
			continue
		}
		ok := true
		for file, commands := range s.files {
			var script bytes.Buffer
			completion(&script, s.shell, commands...)
			if os.WriteFile(filepath.Join(s.dir, file), script.Bytes(), 0o644) != nil {
				ok = false
			}
		}
		if ok {
			shells = append(shells, s.shell)
		}
	}
	return shells
}

// andList writes words as a list: "a, b and c".
func andList(words []string) string {
	if len(words) < 2 {
		return strings.Join(words, "")
	}
	return strings.Join(words[:len(words)-1], ", ") + " and " + words[len(words)-1]
}

func isDir(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}
