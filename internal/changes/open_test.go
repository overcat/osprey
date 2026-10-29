package changes

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/overcat/osprey/internal/gittest"
)

func TestOpen(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Run(t, dir, "branch", "dev")
	other := t.TempDir()
	gittest.Write(t, other, "x/a.txt", "1\n")
	gittest.Write(t, other, "y/a.txt", "2\n")
	gittest.Write(t, other, "a.txt", "1\n")
	head, _ := line(dir, "rev-parse", "--short", "HEAD")
	x, y := filepath.Join(other, "x"), filepath.Join(other, "y")

	for _, tc := range []struct {
		args      []string
		committed bool
		want      string
	}{
		{nil, false, "Uncommitted changes"},
		{[]string{"dev"}, false, "Since dev · with uncommitted"},
		{[]string{"dev"}, true, "Since dev"},
		{[]string{"HEAD", "dev"}, false, head + " → dev"}, // HEAD pinned, dev a branch
		{[]string{"dev", "HEAD"}, false, "Since dev"},     // what is committed on the branch
		{[]string{x, y}, false, x + " → " + y},
		{[]string{filepath.Join(x, "a.txt"), filepath.Join(y, "a.txt")}, false, filepath.Join(x, "a.txt") + " → " + filepath.Join(y, "a.txt")},
	} {
		src, err := Open(dir, tc.args, tc.committed)
		if err != nil {
			t.Fatalf("%v: %v", tc.args, err)
		}
		if got := src.Subtitle(); got != tc.want {
			t.Errorf("%v: got %q, want %q", tc.args, got, tc.want)
		}
	}
	for _, args := range [][]string{{"nope"}, {"dev", "nope"}, {x, filepath.Join(other, "a.txt")}, {"a", "b", "c"}} {
		if _, err := Open(dir, args, false); err == nil {
			t.Errorf("%v should fail", args)
		}
	}
}

func TestOpenOutsideRepository(t *testing.T) {
	outside := t.TempDir()
	t.Chdir(outside)
	if _, err := Open("", nil, false); !errors.Is(err, ErrNoRepository) {
		t.Fatalf("launched outside a repository: %v", err)
	}
	if _, err := Open(outside, nil, false); err == nil || errors.Is(err, ErrNoRepository) {
		t.Fatalf("a directory given outside a repository: %v", err)
	}
}

// TestPinnedBase checks that a base named by where it is, as HEAD~1, stays
// where it was as commits are made.
func TestPinnedBase(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Write(t, dir, "a.txt", "a\n")
	gittest.Run(t, dir, "add", ".")
	gittest.Run(t, dir, "commit", "-q", "-m", "a")
	src, err := Open(dir, []string{"HEAD~1"}, false)
	if err != nil {
		t.Fatal(err)
	}
	gittest.Write(t, dir, "b.txt", "b\n")
	gittest.Run(t, dir, "add", ".")
	gittest.Run(t, dir, "commit", "-q", "-m", "b")
	list, err := src.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := summary(list); !slices.Equal(got, []string{"A a.txt", "A b.txt"}) {
		t.Fatalf("the base moved with the new commit: %q", got)
	}
}

func TestOpenPaths(t *testing.T) {
	dir := gittest.Repo(t) // README.md, gone.txt, src/app.go, src/old.go
	gittest.Run(t, dir, "branch", "dev")
	gittest.Run(t, dir, "branch", "src") // a branch named as a folder
	other := gittest.Repo(t)
	src := filepath.Join(dir, "src")

	for _, tc := range []struct {
		dir   string
		args  []string
		root  string
		title string
	}{
		{dir, []string{"src"}, dir, "Since src · with uncommitted"}, // the branch, before the folder
		{dir, []string{"--", "src"}, dir, "Uncommitted changes · in src"},
		{dir, []string{"README.md"}, dir, "Uncommitted changes · in README.md"},
		{dir, []string{"."}, dir, "Uncommitted changes"},
		{src, []string{"."}, dir, "Uncommitted changes · in src"}, // found from -C
		{src, []string{"app.go"}, dir, "Uncommitted changes · in src/app.go"},
		{dir, []string{other}, other, "Uncommitted changes"},
		{dir, []string{filepath.Join(other, "src")}, other, "Uncommitted changes · in src"},
		{dir, []string{"dev", "src/app.go"}, dir, "Since dev · with uncommitted · in src/app.go"},
		{dir, []string{"dev", "--", "gone.txt", "src"}, dir, "Since dev · with uncommitted · in gone.txt, src"},
		{dir, []string{"dev...HEAD"}, dir, "Since dev"},
		{dir, []string{"dev..."}, dir, "Since dev"},
	} {
		src, err := Open(tc.dir, tc.args, false)
		if err != nil {
			t.Errorf("%v: %v", tc.args, err)
			continue
		}
		g, ok := src.(*Git)
		if !ok || g.Root != tc.root || g.Subtitle() != tc.title {
			t.Errorf("%v from %s: %T at %s, %q; want %q", tc.args, tc.dir, src, g.Root, src.Subtitle(), tc.title)
		}
	}

	// Two files or folders on their own are compared, in a repository too,
	// and found from -C.
	if f, err := Open(src, []string{"app.go", "old.go"}, false); err != nil || f.Subtitle() != filepath.Join(src, "app.go")+" → "+filepath.Join(src, "old.go") {
		t.Errorf("two files: %v, %v", f, err)
	}

	for args, want := range map[string]string{
		"dev..HEAD":       "three dots",
		"src/app.go dev":  "before the files",
		"dev " + other:    "not in the repository",
		"README.md src":   "osp -- README.md src",
		"dev HEAD dev":    "too many",
		"nothing-like-it": "not a branch, a commit, a file or a folder",
		"dev...nothing":   "nothing is not a branch or a commit",
	} {
		_, err := Open(dir, strings.Fields(args), false)
		var usageErr UsageError
		if !errors.As(err, &usageErr) || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v, want a usage error saying %q", args, err, want)
		}
	}
}

func TestOpenCommitted(t *testing.T) {
	dir := gittest.Branch(t) // feature, from main
	src, err := Open(dir, nil, true)
	if err != nil || src.Subtitle() != "Since main" {
		t.Fatalf("osp --committed on a branch: %v, %v", src, err)
	}
	gittest.Run(t, dir, "checkout", "-q", "main")
	var usageErr UsageError
	if _, err := Open(dir, nil, true); !errors.As(err, &usageErr) {
		t.Fatalf("osp --committed on main, which has nothing to count from: %v", err)
	}
}
