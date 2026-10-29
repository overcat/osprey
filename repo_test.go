package main

import (
	"slices"
	"testing"

	"github.com/overcat/osprey/internal/changes"
	"github.com/overcat/osprey/internal/gittest"
)

func TestComparisons(t *testing.T) {
	dir := gittest.Branch(t)
	r := newRepo(&changes.Git{Root: dir})
	h, err := r.historyLoader()()
	if err != nil {
		t.Fatal(err)
	}
	r.history = h
	if r.cmp.kind != compareUncommitted || r.label(r.cmp) != "Uncommitted changes" {
		t.Fatalf("osp starts at %+v", r.cmp)
	}
	load := func() []string {
		t.Helper()
		list, err := r.source().Load()
		if err != nil {
			t.Fatal(err)
		}
		var paths []string
		for _, ch := range list {
			paths = append(paths, string(ch.Status)+" "+ch.Path)
		}
		slices.Sort(paths)
		return paths
	}
	if got := load(); !slices.Equal(got, []string{"M README.md"}) {
		t.Errorf("uncommitted: %q", got)
	}
	r.cmp = comparison{kind: compareBranch}
	if got := load(); !slices.Equal(got, []string{"A src/login.go", "M README.md"}) {
		t.Errorf("branch with uncommitted: %q", got)
	}
	if r.label(r.cmp) != "All changes since main" {
		t.Errorf("label %q", r.label(r.cmp))
	}
	r.uncommitted = false
	if got := load(); !slices.Equal(got, []string{"A src/login.go"}) {
		t.Errorf("branch, committed only: %q", got)
	}
	r.cmp = comparison{kind: compareCommit, commit: h.Branch[0].Hash}
	if got := load(); !slices.Equal(got, []string{"M src/login.go"}) || r.label(r.cmp) != "Return errors from login" {
		t.Errorf("commit: %q, %q", got, r.label(r.cmp))
	}
}

func TestRepoFromCommandLine(t *testing.T) {
	for _, tc := range []struct {
		src         changes.Git
		kind        int
		uncommitted bool
		label       string
	}{
		{changes.Git{}, compareUncommitted, true, "Uncommitted changes"},
		{changes.Git{Base: "main"}, compareBranch, true, "All changes since main"},
		{changes.Git{Base: "main", Head: "HEAD"}, compareBranch, false, "All changes since main"}, // --committed
		{changes.Git{Base: "main", Head: "feature"}, compareBranch, true, "main → feature"},
	} {
		r := newRepo(&tc.src)
		if r.cmp.kind != tc.kind || r.uncommitted != tc.uncommitted || r.label(r.cmp) != tc.label {
			t.Errorf("%+v: %+v, %q", tc.src, r.cmp, r.label(r.cmp))
		}
	}
}

func TestRepoKeepsPaths(t *testing.T) {
	r := newRepo(&changes.Git{Root: "/repo", Base: "main", Paths: []string{"src"}})
	for _, c := range []comparison{{kind: compareBranch}, {kind: compareUncommitted}, {kind: compareCommit, commit: "abc"}} {
		r.cmp = c
		if got := r.source().Paths; !slices.Equal(got, []string{"src"}) {
			t.Errorf("%+v kept to %q", c, got)
		}
	}
	a := newApp()
	a.repo, a.src = r, r.source()
	if got := a.comparisonLabel(); got != "Commit abc · in src" {
		t.Errorf("label %q", got)
	}
}
