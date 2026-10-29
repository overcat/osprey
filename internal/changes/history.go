package changes

import (
	"cmp"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// History is what there is to pick from in a repository: the commits on
// its branch since it left its base, and the commits before them.
type History struct {
	// Base is what the branch's changes count from: the base given, or
	// the default branch, "" when there is none.
	Base string
	// Upstream reports that Base is the branch's upstream, as on main:
	// the branch's commits are those not pushed yet.
	Upstream bool
	Branch   []Commit // since Base, newest first
	Earlier  []Commit // before them, newest first
}

// Find returns the commit of the history with the hash.
func (h *History) Find(hash string) (Commit, bool) {
	for _, list := range [][]Commit{h.Branch, h.Earlier} {
		for _, c := range list {
			if c.Hash == hash {
				return c, true
			}
		}
	}
	return Commit{}, false
}

// Commit is a commit of a history.
type Commit struct {
	Hash, Short, Subject, Author string
	Time                         time.Time
	// Files counts the files the commit changed; Added and Deleted their
	// lines.
	Files, Added, Deleted int
}

// HistoryCache keeps a repository's history between loads, which load it
// again only once a commit moved it: as files change, and as the user
// picks another comparison. Its zero value is ready to use.
type HistoryCache struct {
	mu      sync.Mutex
	key     string // what the tip and the base were
	history History
}

// branchCommits and earlierCommits are how many commits a history lists,
// of the branch and before it.
const (
	branchCommits  = 200
	earlierCommits = 30
)

// Load returns the history of the repository at root: its branch counted
// from base, or from the default branch when "", up to head, or what is
// checked out when "", and the commits that touched paths, when given. A
// repository without commits has none.
func (h *HistoryCache) Load(root, base, head string, paths []string) (History, error) {
	upstream := false
	if base == "" {
		base, upstream = defaultBase(root)
	}
	tip := cmp.Or(head, "HEAD")
	revs := []string{tip}
	if base != "" {
		revs = append(revs, base)
	}
	key := Revisions(root, revs...)
	if key == "" {
		return History{}, nil // no commits yet
	}
	key += strings.Join(paths, "\x00")
	limit := append([]string{"--"}, paths...)
	h.mu.Lock()
	defer h.mu.Unlock()
	if key == h.key {
		return h.history, nil
	}
	hist := History{Base: base, Upstream: upstream}
	from := tip
	var err error
	if base != "" {
		if hist.Branch, err = logCommits(root, append([]string{"-n", fmt.Sprint(branchCommits), base + ".." + tip}, limit...)...); err != nil {
			return History{}, err
		}
		from = mergeBase(root, base, tip)
	}
	if from != "" {
		if hist.Earlier, err = logCommits(root, append([]string{"-n", fmt.Sprint(earlierCommits), from}, limit...)...); err != nil {
			return History{}, err
		}
	}
	h.key, h.history = key, hist
	return hist, nil
}

// logCommits lists the commits git log lists for args, newest first, each
// with the count of the files and lines it changed.
func logCommits(dir string, args ...string) ([]Commit, error) {
	out, err := run(dir, append([]string{"-c", "log.showSignature=false", "log", "--no-color",
		"--format=%x1e%H%x1f%h%x1f%an%x1f%at%x1f%s", "--shortstat"}, args...)...)
	if err != nil {
		return nil, err
	}
	var list []Commit
	for _, rec := range strings.Split(string(out), "\x1e")[1:] {
		head, stat, _ := strings.Cut(rec, "\n")
		f := strings.Split(head, "\x1f")
		if len(f) != 5 {
			return nil, fmt.Errorf("git log: unexpected record %q", head)
		}
		sec, _ := strconv.ParseInt(f[3], 10, 64)
		c := Commit{Hash: f[0], Short: f[1], Author: f[2], Time: time.Unix(sec, 0), Subject: f[4]}
		c.Files, c.Added, c.Deleted = parseShortstat(stat)
		list = append(list, c)
	}
	return list, nil
}

var shortstat = regexp.MustCompile(`(\d+) files? changed(?:, (\d+) insertions?\(\+\))?(?:, (\d+) deletions?\(-\))?`)

// parseShortstat reads git's "3 files changed, 18 insertions(+), 4
// deletions(-)", whose counts of none are left out.
func parseShortstat(s string) (files, added, deleted int) {
	m := shortstat.FindStringSubmatch(s)
	if m == nil {
		return 0, 0, 0
	}
	files, _ = strconv.Atoi(m[1])
	added, _ = strconv.Atoi(m[2])
	deleted, _ = strconv.Atoi(m[3])
	return files, added, deleted
}

// defaultBase returns the branch the one checked out most likely left:
// the remote's default branch, else main or master, the local branch
// when there is one. On that branch itself it returns its upstream, as
// origin/main, so that its changes are those not pushed yet; "" when it
// has none, or there is no such branch.
func defaultBase(dir string) (base string, upstream bool) {
	var names []string
	if remote, err := line(dir, "symbolic-ref", "--short", "-q", "refs/remotes/origin/HEAD"); err == nil && remote != "" {
		names = append(names, strings.TrimPrefix(remote, "origin/"), remote)
	}
	names = append(names, "main", "master")
	current := currentBranch(dir)
	for _, name := range names {
		if _, ok := resolveCommit(dir, name); !ok {
			continue
		}
		if name != current {
			return name, false
		}
		up, err := line(dir, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
		if err != nil || up == "" {
			return "", false
		}
		return up, true
	}
	return "", false
}
