package main

import (
	"context"
	"path"
	"slices"
	"time"

	"github.com/egoist/mygo"

	"github.com/overcat/osprey/internal/changes"
)

// Loading a session's files on goroutines of their own, and again as an
// agent changes them.

// loader loads a source's files, remembering the counts of lines of the
// changes it saw, so that a reload diffs only the files that changed.
type loader struct {
	src     changes.Source
	stats   map[string][2]int
	history func() (changes.History, error) // of a Git repository's source
}

func (l *loader) load() ([]*file, error) {
	list, err := l.src.Load()
	if err != nil {
		return nil, err
	}
	stats := make(map[string][2]int, len(list))
	files := make([]*file, len(list))
	for i, ch := range list {
		f := newFile(ch)
		if f.hasText() {
			if s, ok := l.stats[f.hash]; ok {
				f.added, f.deleted = s[0], s[1]
			} else {
				f.added, f.deleted = f.diffView(modeSingle).Stats()
			}
		}
		stats[f.hash] = [2]int{f.added, f.deleted}
		files[i] = f
	}
	l.stats = stats
	return files, nil
}

// refresh loads the files again.
func (a *app) refresh() {
	if a.reload != nil {
		poke(a.reload)
	}
}

// poke asks for a load, unless one is asked for already.
func poke(reload chan<- struct{}) {
	select {
	case reload <- struct{}{}:
	default:
	}
}

// loadLoop loads the files each time reload asks, until ctx is done, and
// shows them.
func (a *app) loadLoop(ctx context.Context, ld *loader, reload <-chan struct{}) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-reload:
		}
		files, err := ld.load()
		branch := ""
		if g, ok := ld.src.(*changes.Git); ok && err == nil {
			branch = g.Branch()
		}
		// The history is extra: a load without it still shows the files.
		var hist *changes.History
		if ld.history != nil && err == nil {
			if h, err := ld.history(); err == nil {
				hist = &h
			}
		}
		if ctx.Err() != nil {
			return
		}
		a.win.Update(func() {
			if ctx.Err() == nil {
				if hist != nil && a.repo != nil {
					a.repo.history = *hist
				}
				a.apply(files, branch, err)
			}
		})
	}
}

// apply shows the files loaded, keeping the views of those still there.
func (a *app) apply(files []*file, branch string, err error) {
	a.loaded = true
	a.loadErr = err
	if err != nil {
		return
	}
	a.branch = branch
	old := make(map[string]*file, len(a.files))
	for _, f := range a.files {
		old[f.Path] = f
	}
	for i, f := range files {
		if p, ok := old[f.Path]; ok {
			if p.hash != f.hash {
				p.update(f)
			}
			files[i] = p
		}
	}
	// The sidebar's tree takes them in its order.
	slices.SortFunc(files, func(a, b *file) int { return comparePaths(a.Path, b.Path) })
	a.files = files
	a.syncSelection()
	a.syncDone()
}

// unwatched are the usual names of directories of dependencies and build
// output, whose changes are not the review's.
var unwatched = []string{"node_modules", ".next", ".turbo", ".cache", "__pycache__", ".venv", ".gradle", ".idea", "target", "dist", "build"}

// watchLimit is how many directories a watch may enroll: Linux watches
// each directory with a handle of its own, up to 8192 unless told more.
const watchLimit = 1 << 17

// pollEvery is how often a load is asked for where files can't be watched.
var pollEvery = 5 * time.Second

// watch asks for a load as files change on disk, as an agent writes them.
// Where the system can't watch dir, as a tree past the system's limit of
// watches, it asks every pollEvery instead.
func watch(ctx context.Context, dir string, isGit bool, reload chan<- struct{}) {
	w, err := mygo.WatchFiles(ctx, dir, mygo.FileWatchOptions{
		Recursive:  true,
		HintsOnly:  true,
		MaxEntries: watchLimit,
		Ignore:     watchIgnore(dir, isGit),
	})
	if err == nil {
		for err == nil {
			if _, err = w.Next(ctx); err == nil {
				poke(reload)
			}
		}
		w.Close()
	}
	t := time.NewTicker(pollEvery)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			poke(reload)
		}
	}
}

// watchIgnore returns what watch leaves out of the directory dir: Git's
// own, those Git ignores in a repository, and those of the usual names of
// dependencies and build output that it tracks nothing in, as build output
// made after the watch began. A directory of those names Git tracks files
// in is watched, as source code.
func watchIgnore(dir string, isGit bool) func(p string, isDir bool) bool {
	var ignored, tracked map[string]bool
	if isGit {
		ignored, tracked = changes.IgnoredDirs(dir), changes.TrackedDirs(dir)
	}
	return func(p string, isDir bool) bool {
		if !isDir {
			return false
		}
		name := path.Base(p)
		return name == ".git" || ignored[p] || slices.Contains(unwatched, name) && !tracked[p]
	}
}

// watchHead asks for a load when a commit moves what the files are
// compared with, as when an agent commits; watch leaves out .git.
func watchHead(ctx context.Context, src *changes.Git, reload chan<- struct{}) {
	revs := []string{"HEAD"}
	if src.Base != "" {
		revs = append(revs, src.Base)
	}
	last := ""
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		if now := changes.Revisions(src.Root, revs...); now != last {
			if last != "" {
				poke(reload)
			}
			last = now
		}
	}
}
