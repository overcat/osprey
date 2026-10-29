// Package gittest makes Git repositories for tests.
package gittest

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Run runs git in dir, as an author of its own and without the user's or
// the system's configuration, failing the test when it fails.
func Run(t testing.TB, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
		"GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// Write writes text to the file name of dir, a path with slashes, making
// its directories.
func Write(t testing.TB, dir, name, text string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Repo makes a repository on main with one commit, "first", of four files:
// README.md, src/app.go, src/old.go, long enough for a rename of it to be
// found, and gone.txt.
func Repo(t testing.TB) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	Run(t, dir, "init", "-q", "-b", "main")
	Write(t, dir, "README.md", "hello\n")
	Write(t, dir, "src/app.go", "package app\n\nfunc A() {}\n")
	Write(t, dir, "src/old.go", "package app\n\n// Old is long enough to be found as renamed.\nfunc Old() int { return 42 }\n")
	Write(t, dir, "gone.txt", "bye\n")
	Run(t, dir, "add", ".")
	Run(t, dir, "commit", "-q", "-m", "first")
	return dir
}

// Branch makes Repo, then a branch, feature, two commits from main, which
// add and then change src/login.go, and a change of README.md not
// committed yet.
func Branch(t testing.TB) string {
	t.Helper()
	dir := Repo(t)
	Run(t, dir, "checkout", "-q", "-b", "feature")
	Write(t, dir, "src/login.go", "package app\n\nfunc Login() {}\n")
	Run(t, dir, "add", ".")
	Run(t, dir, "commit", "-q", "-m", "Add login")
	Write(t, dir, "src/login.go", "package app\n\nfunc Login() error { return nil }\n")
	Run(t, dir, "commit", "-q", "-am", "Return errors from login")
	Write(t, dir, "README.md", "hello, uncommitted\n")
	return dir
}
