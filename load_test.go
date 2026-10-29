package main

import (
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/overcat/osprey/internal/changes"
	"github.com/overcat/osprey/internal/gittest"
	"github.com/overcat/osprey/internal/review"
)

func TestReloadKeepsViewsAndReviews(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Write(t, dir, "src/app.go", "package app\n\nfunc A() { println(1) }\n")
	gittest.Write(t, dir, "README.md", "hello, edited\n")
	// The reviews are saved in a home of the test's own.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("AppData", filepath.Join(home, "AppData"))

	src := &changes.Git{Root: dir}
	a := newApp()
	a.src, a.reviews = src, review.Load(src.Key())
	ld := &loader{src: src}
	load := func() {
		t.Helper()
		files, err := ld.load()
		if err != nil {
			t.Fatal(err)
		}
		a.apply(files, "", nil)
	}
	byPath := func(p string) *file {
		for _, f := range a.files {
			if f.Path == p {
				return f
			}
		}
		t.Fatalf("no %s", p)
		return nil
	}

	load()
	if len(a.files) != 2 || a.selected != "src/app.go" {
		t.Fatalf("files %d, selected %q", len(a.files), a.selected)
	}
	app, readme := byPath("src/app.go"), byPath("README.md")
	if app.added != 1 || app.deleted != 1 {
		t.Errorf("app.go: +%d -%d", app.added, app.deleted)
	}
	a.toggleReviewed() // app.go, then on to README.md
	if !a.reviewed(app) || a.selected != "README.md" {
		t.Fatalf("reviewed %v, selected %q", a.reviewed(app), a.selected)
	}
	a.toggleReviewed()
	buf := app.new

	// An agent edits app.go again: its review goes, its buffer stays.
	gittest.Write(t, dir, "src/app.go", "package app\n\nfunc A() { println(2) }\n")
	load()
	if byPath("src/app.go") != app || app.new != buf {
		t.Fatal("reload replaced the file's views")
	}
	if got := buf.String(); got != "package app\n\nfunc A() { println(2) }\n" {
		t.Fatalf("buffer not updated: %q", got)
	}
	if a.reviewed(app) {
		t.Error("a file changed since its review is still reviewed")
	}
	if !a.reviewed(byPath("README.md")) || byPath("README.md") != readme {
		t.Error("an unchanged file lost its review or its views")
	}
	// Reviews last between launches.
	if !review.Load(src.Key()).Reviewed(readme.Path, readme.hash) {
		t.Error("reviews weren't saved")
	}
}

func TestWatchIgnore(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Write(t, dir, ".gitignore", "node_modules/\n")
	gittest.Write(t, dir, "build/make.go", "package build\n") // source code, in a directory named as build output
	gittest.Run(t, dir, "add", ".")
	gittest.Run(t, dir, "commit", "-q", "-m", "build")
	gittest.Write(t, dir, "node_modules/x/index.js", "x\n")
	gittest.Write(t, dir, "web/node_modules/y/index.js", "y\n")
	gittest.Write(t, dir, "dist/out.js", "out\n") // build output Git doesn't ignore
	ignore := watchIgnore(dir, true)
	for p, want := range map[string]bool{
		".git": true, "node_modules": true, "web/node_modules": true, "dist": true,
		"build": false, "src": false, "web": false,
	} {
		if got := ignore(p, true); got != want {
			t.Errorf("%s: ignored %v, want %v", p, got, want)
		}
	}
	if ignore("dist", false) {
		t.Error("files are not left out, their directories are")
	}
	if outside := watchIgnore(t.TempDir(), false); !outside("build", true) || !outside(".git", true) {
		t.Error("outside Git, directories of the usual names are left out")
	}
}

// TestWatchFallsBackToPolling checks that files are loaded again even
// where they can't be watched.
func TestWatchFallsBackToPolling(t *testing.T) {
	defer func(every time.Duration) { pollEvery = every }(pollEvery)
	pollEvery = 10 * time.Millisecond
	reload := make(chan struct{}, 1)
	go watch(t.Context(), filepath.Join(t.TempDir(), "missing"), false, reload)
	select {
	case <-reload:
	case <-time.After(5 * time.Second):
		t.Fatal("no load was asked for")
	}
}

func TestApplySorts(t *testing.T) {
	a := newApp()
	a.reviews = review.New()
	var files []*file
	for _, p := range []string{"z.go", "src/b.go", "README.md", "src/a/x.go"} {
		files = append(files, newFile(changes.Change{Path: p, Status: changes.StatusAdded, New: "x\n"}))
	}
	a.apply(files, "", nil)
	var got []string
	for _, f := range a.files {
		got = append(got, f.Path)
	}
	if want := []string{"src/a/x.go", "src/b.go", "README.md", "z.go"}; !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}
