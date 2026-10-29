package main

import (
	"context"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/overcat/osprey/internal/changes"
	"github.com/overcat/osprey/internal/review"
)

// The choices the user switches between, ints as ui.Segmented keeps them.
const (
	modeContinuous = iota // every file in one feed
	modeSingle            // one file at a time
)

const (
	pageDiff = iota
	pageFile // the whole file
)

const (
	layoutSplit = iota
	layoutUnified
)

const (
	scopeAll      = iota
	scopeToReview // only the files not reviewed yet
)

// flip switches a choice between its two values.
func flip(v int) int { return 1 - v }

// app is Osprey's state, which its window's view shows: the session of
// what it compares, and how the user chose to see it.
type app struct {
	win *mygo.Window
	// appMenu is the menu bar's, which menuButton opens where the window
	// shows no menu bar.
	appMenu *mygo.Menu
	// repo is the Git repository the session compares, nil for folders
	// and files; it lasts as the user picks another comparison of it.
	repo *repo
	session

	focus         string // the control asked to take the focus
	sideTab       int    // tabFiles or tabHistory
	sidebarHidden bool
	browseMode    int
	page          int
	layout        int
	wrap          bool
	expand        bool // show unchanged lines in the diff
	flat          bool // the files as a list rather than a tree
	sidebarWidth  float32
	shortcutsOpen bool     // the panel of the keyboard shortcuts
	cli           cliState // the command line tool, as the welcome page tells of it
}

// session is what Osprey shows of one source, and how far its review
// is; opening another source, or another comparison, starts a new one.
type session struct {
	src     changes.Source
	stop    context.CancelFunc // the loads and watches of src
	reload  chan struct{}
	loaded  bool
	loadErr error
	branch  string
	files   []*file
	reviews *review.Store

	selected    string // the path of the file shown
	filter      string
	reviewScope int
	// retained are the files reviewed in the feed while it shows those to
	// review, by path and hash: they keep their place until the list is
	// filtered again, so that marking one doesn't move the text read.
	retained   map[string]string
	collapsed  map[string]bool // the sidebar's directories closed
	list       ui.ListState    // the sidebar's
	reveal     bool            // scroll the selected row into view
	feed       feed
	focusEntry string // the file selected as single-file mode began
}

// feed is the continuous feed's state.
type feed struct {
	list ui.ListState
	rows []*file // the files the last frame listed
	// target is a file to scroll to, as the user chose; following the
	// scrolling never sets it.
	target string
	top    string // the file last seen at the top
	hunk   *file  // a file to scroll to the anchor of its change
}

func newApp() *app {
	return &app{session: session{collapsed: map[string]bool{}}, sidebarWidth: 260}
}

// open starts showing src, as the command line or the Open dialog chose
// it.
func (a *app) open(src changes.Source) {
	a.repo = nil
	if g, ok := src.(*changes.Git); ok {
		a.repo = newRepo(g)
		src = a.repo.source()
	}
	a.start(src, false)
}

// chooseRepo asks for a repository and opens it. It blocks, on a
// goroutine of its own.
func (a *app) chooseRepo() {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: a.win, Title: "Open Repository", Directory: true})
	if err != nil || len(paths) == 0 {
		return
	}
	root, err := changes.RepoRoot(paths[0])
	if err != nil {
		mygo.Dialog.Error("Couldn't open the folder", err.Error())
		return
	}
	a.win.Update(func() { a.open(&changes.Git{Root: root}) })
}

// compare shows another comparison of the repository, keeping the
// reviews and the filters.
func (a *app) compare(c comparison) {
	if a.repo == nil {
		return
	}
	a.repo.cmp = c
	a.start(a.repo.source(), true)
}

// setUncommitted chooses whether the branch's changes include those not
// committed yet.
func (a *app) setUncommitted(on bool) {
	if a.repo == nil || a.repo.uncommitted == on {
		return
	}
	a.repo.uncommitted = on
	if a.repo.cmp.kind == compareBranch {
		a.compare(a.repo.cmp)
	}
}

// start shows src in a session of its own: loading it, then again
// whenever its files change. Another comparison of the same repository
// keeps the session's reviews, filters and branch.
func (a *app) start(src changes.Source, sameRepo bool) {
	if a.stop != nil {
		a.stop()
	}
	ctx, stop := context.WithCancel(context.Background())
	last := a.session
	a.session = session{
		src:       src,
		stop:      stop,
		reload:    make(chan struct{}, 1),
		collapsed: map[string]bool{},
	}
	if sameRepo {
		a.reviews, a.filter, a.reviewScope, a.branch = last.reviews, last.filter, last.reviewScope, last.branch
	} else {
		a.reviews = review.Load(src.Key())
	}
	if a.win == nil {
		stop() // no window to show it in, as in tests
		return
	}
	a.win.SetTitle("Osprey — " + src.Title())
	ld := &loader{src: src}
	if a.repo != nil {
		ld.history = a.repo.historyLoader()
	}
	go a.loadLoop(ctx, ld, a.reload)
	g, isGit := src.(*changes.Git)
	if dir := src.WatchDir(); dir != "" {
		go watch(ctx, dir, isGit, a.reload)
	}
	if isGit {
		go watchHead(ctx, g, a.reload)
	}
	a.refresh()
}
