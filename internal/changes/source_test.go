package changes

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/overcat/osprey/internal/gittest"
)

func TestWorkTree(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Write(t, dir, "src/app.go", "package app\n\nfunc A() { println() }\n")
	os.Remove(filepath.Join(dir, "gone.txt"))
	gittest.Run(t, dir, "mv", "src/old.go", "src/renamed.go")
	gittest.Write(t, dir, "new/file.txt", "new\n")
	gittest.Write(t, dir, "logo.bin", "\x00\x01\x02")

	list, err := (&Git{Root: dir}).Load()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"A logo.bin (Binary file)", "A new/file.txt", "D gone.txt", "M src/app.go", "R src/renamed.go <- src/old.go"}
	if got := summary(list); !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
	for _, ch := range list {
		switch ch.Path {
		case "src/app.go":
			if ch.Old != "package app\n\nfunc A() {}\n" || !strings.Contains(ch.New, "println") {
				t.Errorf("app.go texts: %q → %q", ch.Old, ch.New)
			}
		case "gone.txt":
			if ch.Old != "bye\n" || ch.New != "" {
				t.Errorf("gone.txt texts: %q → %q", ch.Old, ch.New)
			}
		case "new/file.txt":
			if ch.Old != "" || ch.New != "new\n" {
				t.Errorf("file.txt texts: %q → %q", ch.Old, ch.New)
			}
		}
	}
}

func TestBranches(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Run(t, dir, "checkout", "-q", "-b", "feature")
	gittest.Write(t, dir, "src/feature.go", "package app\n")
	gittest.Run(t, dir, "add", ".")
	gittest.Run(t, dir, "commit", "-q", "-m", "feature")
	// main moves on after the branch left it: not the branch's change.
	gittest.Run(t, dir, "checkout", "-q", "main")
	gittest.Write(t, dir, "main-only.txt", "main\n")
	gittest.Run(t, dir, "add", ".")
	gittest.Run(t, dir, "commit", "-q", "-m", "main moves")
	gittest.Run(t, dir, "checkout", "-q", "feature")
	gittest.Write(t, dir, "README.md", "hello, uncommitted\n")

	for _, tc := range []struct {
		src  Git
		want []string
	}{
		{Git{Base: "main"}, []string{"A src/feature.go", "M README.md"}},
		{Git{Base: "main", Head: "HEAD"}, []string{"A src/feature.go"}},
		{Git{Base: "main", Head: "feature"}, []string{"A src/feature.go"}},
		{Git{Base: "feature", Head: "main"}, []string{"A main-only.txt"}},
		{Git{}, []string{"M README.md"}},
	} {
		tc.src.Root = dir
		list, err := tc.src.Load()
		if err != nil {
			t.Fatal(err)
		}
		if got := summary(list); !slices.Equal(got, tc.want) {
			t.Errorf("%s: got %q, want %q", tc.src.Subtitle(), got, tc.want)
		}
	}
}

func TestCommit(t *testing.T) {
	dir := gittest.Branch(t)
	head, _ := line(dir, "rev-parse", "HEAD")
	list, err := (&Git{Root: dir, Commit: head}).Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := summary(list); !slices.Equal(got, []string{"M src/login.go"}) {
		t.Errorf("the last commit: %q", got)
	}
	first, _ := line(dir, "rev-list", "--max-parents=0", "HEAD")
	list, err = (&Git{Root: dir, Commit: first}).Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := summary(list); !slices.Equal(got, []string{"A README.md", "A gone.txt", "A src/app.go", "A src/old.go"}) {
		t.Errorf("the first commit, without a parent: %q", got)
	}
}

func TestBeforeFirstCommit(t *testing.T) {
	dir, _ := filepath.EvalSymlinks(t.TempDir())
	gittest.Run(t, dir, "init", "-q")
	gittest.Write(t, dir, "a.txt", "a\n")
	list, err := (&Git{Root: dir}).Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := summary(list); !slices.Equal(got, []string{"A a.txt"}) {
		t.Fatalf("got %q", got)
	}
}

// TestLargeWorkFile checks that a file too large to show is not taken for
// a binary one, and that its change follows it.
func TestLargeWorkFile(t *testing.T) {
	dir := gittest.Repo(t)
	big := strings.Repeat("text\n", maxText/5+1)
	gittest.Write(t, dir, "big.txt", big)
	load := func() Change {
		t.Helper()
		list, err := (&Git{Root: dir}).Load()
		if err != nil || len(list) != 1 {
			t.Fatal(list, err)
		}
		return list[0]
	}
	first := load()
	if first.Note != tooLarge {
		t.Fatalf("note %q", first.Note)
	}
	gittest.Write(t, dir, "big.txt", big+"more\n")
	if again := load(); again.Hash() == first.Hash() {
		t.Fatal("the change of a large file didn't follow it")
	}
}

func TestDirs(t *testing.T) {
	root := t.TempDir()
	a, b := filepath.Join(root, "a"), filepath.Join(root, "b")
	gittest.Write(t, a, "same.txt", "s\n")
	gittest.Write(t, b, "same.txt", "s\n")
	gittest.Write(t, a, "sub/changed.txt", "1\n")
	gittest.Write(t, b, "sub/changed.txt", "2\n")
	gittest.Write(t, a, "removed.txt", "r\n")
	gittest.Write(t, b, "added.txt", "n\n")
	list, err := (&Dirs{Old: a, New: b}).Load()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"A added.txt", "D removed.txt", "M sub/changed.txt"}
	if got := summary(list); !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestFiles(t *testing.T) {
	dir := t.TempDir()
	gittest.Write(t, dir, "old.go", "package a\n")
	gittest.Write(t, dir, "new.go", "package b\n")
	gittest.Write(t, dir, "same.go", "package a\n")
	list, err := (&Files{Old: filepath.Join(dir, "old.go"), New: filepath.Join(dir, "new.go")}).Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := summary(list); !slices.Equal(got, []string{"M new.go <- old.go"}) || list[0].Old != "package a\n" {
		t.Fatalf("got %q", got)
	}
	if list, _ := (&Files{Old: filepath.Join(dir, "old.go"), New: filepath.Join(dir, "same.go")}).Load(); len(list) != 0 {
		t.Fatalf("files the same have no changes: %q", summary(list))
	}
}

// TestPaths checks that paths keep the changes to the files under them,
// taken as they are written: "[odd].txt" is a name, not a pattern.
func TestPaths(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Write(t, dir, "src/app.go", "package app\n\nfunc A() { println() }\n")
	gittest.Write(t, dir, "src/new.go", "package app\n")
	gittest.Write(t, dir, "README.md", "hello, edited\n")
	gittest.Write(t, dir, "[odd].txt", "odd\n")
	gittest.Write(t, dir, "o.txt", "a name the pattern [odd].txt would match\n")
	for _, tc := range []struct {
		paths []string
		want  []string
	}{
		{[]string{"src"}, []string{"A src/new.go", "M src/app.go"}},
		{[]string{"README.md", "src/new.go"}, []string{"A src/new.go", "M README.md"}},
		{[]string{"[odd].txt"}, []string{"A [odd].txt"}},
	} {
		list, err := (&Git{Root: dir, Paths: tc.paths}).Load()
		if err != nil {
			t.Fatal(err)
		}
		if got := summary(list); !slices.Equal(got, tc.want) {
			t.Errorf("%q: got %q, want %q", tc.paths, got, tc.want)
		}
	}
}
