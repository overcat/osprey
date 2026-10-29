package main

import (
	"cmp"

	"github.com/overcat/osprey/internal/changes"
)

// What of a repository's history the files compare.
const (
	compareBranch      = iota // the branch's changes since it left its base
	compareUncommitted        // the changes not committed yet
	compareCommit             // one commit's
)

// comparison is what of a repository's history the files show.
type comparison struct {
	kind   int
	commit string // the hash of compareCommit's commit
}

// repo is the Git repository the comparisons are of. It lasts as the user
// picks one comparison after another.
type repo struct {
	root string
	// base and head are as the command line gave them: the ref the
	// branch's changes count from, "" to find it, and the ref compared, ""
	// for what is checked out.
	base, head string
	// uncommitted makes the branch's changes include those not committed
	// yet, when what is checked out is compared.
	uncommitted bool
	// paths, when set, keep every comparison to the files under them.
	paths   []string
	cmp     comparison
	history changes.History
	cache   *changes.HistoryCache
}

// newRepo starts from the comparison the command line chose.
func newRepo(src *changes.Git) *repo {
	r := &repo{root: src.Root, base: src.Base, head: src.Head, paths: src.Paths, uncommitted: true, cache: &changes.HistoryCache{}}
	switch {
	case src.Base == "":
		r.cmp.kind = compareUncommitted
	case src.Head == "HEAD": // --committed
		r.cmp.kind = compareBranch
		r.head, r.uncommitted = "", false
	default:
		r.cmp.kind = compareBranch
	}
	return r
}

// branchBase is what the branch's changes count from, "" when there is
// nothing to count from.
func (r *repo) branchBase() string { return cmp.Or(r.base, r.history.Base) }

// checkedOut reports whether the comparisons end at what is checked out,
// so that the changes not committed yet show.
func (r *repo) checkedOut() bool { return r.head == "" }

// source returns the source of the comparison chosen.
func (r *repo) source() *changes.Git {
	switch r.cmp.kind {
	case compareUncommitted:
		return &changes.Git{Root: r.root, Paths: r.paths}
	case compareCommit:
		return &changes.Git{Root: r.root, Commit: r.cmp.commit, Paths: r.paths}
	}
	head := r.head
	if head == "" && !r.uncommitted {
		head = "HEAD"
	}
	return &changes.Git{Root: r.root, Base: r.branchBase(), Head: head, Paths: r.paths}
}

// label names a comparison, as the history shows the one chosen.
func (r *repo) label(c comparison) string {
	switch c.kind {
	case compareUncommitted:
		return "Uncommitted changes"
	case compareCommit:
		if cm, ok := r.history.Find(c.commit); ok {
			return cm.Subject
		}
		return "Commit " + c.commit[:min(7, len(c.commit))]
	}
	if r.head != "" {
		return r.branchBase() + " → " + r.head
	}
	return "All changes since " + r.branchBase()
}

// historyLoader returns what loads the history, on a loader's goroutine.
func (r *repo) historyLoader() func() (changes.History, error) {
	root, base, head, paths, cache := r.root, r.base, r.head, r.paths, r.cache
	return func() (changes.History, error) { return cache.Load(root, base, head, paths) }
}
