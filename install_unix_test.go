//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallCLI(t *testing.T) {
	prefix := t.TempDir()
	bin := filepath.Join(prefix, "bin")
	for _, dir := range []string{"bin", "share/fish/vendor_completions.d", "share/zsh/site-functions", "share/bash-completion/completions"} {
		os.MkdirAll(filepath.Join(prefix, dir), 0o755)
	}
	said, err := installCLI(bin)
	if err != nil {
		t.Fatal(err)
	}
	if want := "osprey and osp are in " + bin + ", with the completions of fish, zsh and bash."; said != want {
		t.Errorf("said %q, want %q", said, want)
	}
	exe, _ := os.Executable()
	exe, _ = filepath.EvalSymlinks(exe)
	for _, name := range names {
		if target, err := os.Readlink(filepath.Join(bin, name)); err != nil || target != exe {
			t.Errorf("%s links to %q, %v", name, target, err)
		}
	}
	for file, want := range map[string]string{
		"share/fish/vendor_completions.d/osp.fish":    "complete -c osp ",
		"share/fish/vendor_completions.d/osprey.fish": "complete -c osprey ",
		"share/zsh/site-functions/_osprey":            "#compdef osprey osp",
		"share/bash-completion/completions/osp":       "complete -F _osprey osprey osp",
	} {
		if b, err := os.ReadFile(filepath.Join(prefix, file)); err != nil || !strings.Contains(string(b), want) {
			t.Errorf("%s: %v, without %q", file, err, want)
		}
	}
}

// TestInstallCLIWithoutShare checks that a folder of no prefix, as ~/bin,
// gets the commands alone.
func TestInstallCLIWithoutShare(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "bin")
	os.Mkdir(bin, 0o755)
	if said, err := installCLI(bin); err != nil || said != "osprey and osp are in "+bin+"." {
		t.Fatalf("said %q, %v", said, err)
	}
}
