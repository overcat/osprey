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

// The welcome page offers to install the commands unless osprey and osp
// both run the program from a directory of the PATH.
func TestLinksTo(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	exe, other, bin := filepath.Join(dir, "Osprey"), filepath.Join(dir, "other"), filepath.Join(dir, "bin")
	for _, p := range []string{exe, other} {
		if err := os.WriteFile(p, nil, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	os.Mkdir(bin, 0o755)
	link := func(name, target string) {
		os.Remove(filepath.Join(bin, name))
		if err := os.Symlink(target, filepath.Join(bin, name)); err != nil {
			t.Fatal(err)
		}
	}
	if linksTo(bin, exe) {
		t.Error("an empty directory runs the program")
	}
	link("osprey", exe)
	if linksTo(bin, exe) {
		t.Error("osprey alone runs the program, without osp")
	}
	link("osp", other)
	if linksTo(bin, exe) {
		t.Error("osp runs another program, yet counts")
	}
	link("osp", exe)
	if !linksTo(bin, exe) {
		t.Error("osprey and osp run the program, yet don't count")
	}
}

// Where no usual directory of the PATH takes them, as on a fresh Linux,
// the commands go to ~/.local/bin, which install-cli makes, saying while
// the PATH lacks it.
func TestInstallInHome(t *testing.T) {
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	exe, bin := filepath.Join(home, "Osprey"), filepath.Join(home, ".local", "bin")
	if err := os.WriteFile(exe, nil, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", "/usr/bin:/bin")
	if said, err := installInHome(exe); err != nil || !linksTo(bin, exe) || !strings.Contains(said, "isn't on your PATH yet") {
		t.Fatalf("said %q, %v", said, err)
	}
	t.Setenv("PATH", bin+":/usr/bin:/bin")
	if said, err := installInHome(exe); err != nil || said != "osprey and osp are in "+bin+"." {
		t.Fatalf("with it on the PATH, said %q, %v", said, err)
	}
}
