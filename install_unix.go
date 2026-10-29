//go:build !windows

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// binDirs are the usual directories of the PATH, where installCLI links
// osprey and osp in the first it can write.
func binDirs() []string {
	home, _ := os.UserHomeDir()
	return []string{"/opt/homebrew/bin", "/usr/local/bin", filepath.Join(home, ".local", "bin"), filepath.Join(home, "bin")}
}

// cliMissing reports whether osprey and osp don't run this program yet
// from one of binDirs, or from /usr/bin, where the Debian package links
// them.
func cliMissing() bool {
	exe, err := os.Executable()
	if err == nil {
		exe, err = filepath.EvalSymlinks(exe)
	}
	if err != nil {
		return false
	}
	for _, d := range append(binDirs(), "/usr/bin") {
		if linksTo(d, exe) {
			return false
		}
	}
	return true
}

// linksTo reports whether osprey and osp in dir both run exe.
func linksTo(dir, exe string) bool {
	for _, name := range names {
		if target, err := filepath.EvalSymlinks(filepath.Join(dir, name)); err != nil || target != exe {
			return false
		}
	}
	return true
}

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
	if dir != "" {
		if !isDir(dir) {
			return "", fmt.Errorf("%s is not a directory", dir)
		}
		return linkIn(dir, exe)
	}
	for _, d := range binDirs() {
		if !isDir(d) {
			continue
		}
		if said, err := linkIn(d, exe); err == nil {
			return said, nil
		}
	}
	return installFallback(exe)
}

// linkIn links osprey and osp to exe in dir, over an earlier Osprey's,
// with the shells' completions beside them, and says so.
func linkIn(dir, exe string) (string, error) {
	for _, name := range names {
		link := filepath.Join(dir, name)
		if target, err := os.Readlink(link); err == nil && target == exe {
			continue
		}
		if _, err := os.Lstat(link); err == nil {
			if target, _ := os.Readlink(link); !strings.Contains(strings.ToLower(target), "osprey") {
				return "", fmt.Errorf("%s already exists", link)
			}
			os.Remove(link)
		}
		if err := os.Symlink(exe, link); err != nil {
			return "", err
		}
	}
	said := "osprey and osp are in " + dir
	if shells := installCompletions(dir); len(shells) > 0 {
		said += ", with the completions of " + andList(shells)
	}
	return said + ".", nil
}

// installInHome links osprey and osp in ~/.local/bin, the place of a
// user's own commands, which it makes if need be, and says when the PATH
// lacks it, as until the next login where it was just made.
func installInHome(exe string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	said, err := linkIn(dir, exe)
	if err == nil && !slices.Contains(filepath.SplitList(os.Getenv("PATH")), dir) {
		said += " It isn't on your PATH yet: log in again, or add it, for terminals to find them."
	}
	return said, err
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
	for _, s := range completionFiles {
		dir := filepath.Join(share, s.dir)
		if !isDir(dir) && (s.shell != "fish" || fishErr != nil || os.MkdirAll(dir, 0o755) != nil) {
			continue
		}
		ok := true
		for file, commands := range s.files {
			var script bytes.Buffer
			completion(&script, s.shell, commands...)
			if os.WriteFile(filepath.Join(dir, file), script.Bytes(), 0o644) != nil {
				ok = false
			}
		}
		if ok {
			shells = append(shells, s.shell)
		}
	}
	return shells
}

// completionFiles are the scripts installCompletions writes for each
// shell, in its folder under share, with the commands each completes.
var completionFiles = []struct {
	shell, dir string
	files      map[string][]string
}{
	{"fish", "fish/vendor_completions.d", map[string][]string{"osprey.fish": {"osprey"}, "osp.fish": {"osp"}}},
	{"zsh", "zsh/site-functions", map[string][]string{"_osprey": names}},
	{"bash", "bash-completion/completions", map[string][]string{"osprey": names, "osp": names}},
}

// uninstallCLI removes the osprey and osp of Osprey from the usual
// directories of the PATH, with their completions, as install-cli put
// them, and says where from.
func uninstallCLI() (string, error) {
	var from []string
	for _, d := range binDirs() {
		took, err := unlinkIn(d)
		if errors.Is(err, fs.ErrPermission) {
			exe, _ := os.Executable()
			return "", fmt.Errorf("%w: run sudo %q uninstall-cli", err, exe)
		} else if err != nil {
			return "", err
		}
		if took {
			from = append(from, d)
		}
	}
	if len(from) == 0 {
		return "osprey and osp weren't on your PATH.", nil
	}
	return "osprey and osp are out of " + andList(from) + ".", nil
}

// unlinkIn removes from dir the links osprey and osp to Osprey, as
// install-cli makes them, with the completions beside them, and reports
// whether there were any.
func unlinkIn(dir string) (bool, error) {
	took := false
	for _, name := range names {
		link := filepath.Join(dir, name)
		if target, err := os.Readlink(link); err != nil || !strings.Contains(strings.ToLower(target), "osprey") {
			continue
		}
		if err := os.Remove(link); err != nil {
			return took, err
		}
		took = true
	}
	if took && filepath.Base(dir) == "bin" {
		share := filepath.Join(filepath.Dir(dir), "share")
		for _, s := range completionFiles {
			for file := range s.files {
				os.Remove(filepath.Join(share, s.dir, file))
			}
		}
	}
	return took, nil
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
