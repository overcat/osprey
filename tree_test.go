package main

import (
	"slices"
	"strings"
	"testing"

	"github.com/overcat/osprey/internal/changes"
)

func filesAt(paths ...string) []*file {
	var files []*file
	for _, p := range paths {
		files = append(files, &file{Change: changes.Change{Path: p}})
	}
	slices.SortFunc(files, func(a, b *file) int { return comparePaths(a.Path, b.Path) })
	return files
}

func TestTreeRows(t *testing.T) {
	files := filesAt("z.go", "a/b/c/one.go", "a/b/c/two.go", "a/top.go", "lib/x.go")
	none := func(*file) bool { return false }
	var got []string
	for _, r := range treeRows(files, nil, none) {
		got = append(got, strings.Repeat("  ", r.depth)+r.name)
	}
	want := []string{"a", "  b/c", "    one.go", "    two.go", "  top.go", "lib", "  x.go", "z.go"}
	if !slices.Equal(got, want) {
		t.Fatalf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if rows := treeRows(files, map[string]bool{"a/": true}, none); len(rows) != 4 || rows[0].files != 3 {
		t.Fatalf("collapsed: %+v", rows)
	}
}

func TestFilterFiles(t *testing.T) {
	files := filesAt("src/Login.go", "src/logout.go", "docs/login.md")
	var got []string
	for _, f := range filterFiles(files, "LOGIN src") {
		got = append(got, f.Path)
	}
	if !slices.Equal(got, []string{"src/Login.go"}) {
		t.Fatalf("got %q", got)
	}
	if len(filterFiles(files, "  ")) != 3 {
		t.Fatal("an empty filter should keep every file")
	}
}
