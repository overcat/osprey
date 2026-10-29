package main

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/overcat/osprey/internal/changes"
)

// historyFixture is reviewFixture with a repository's history.
func historyFixture() *app {
	a := reviewFixture()
	a.repo = &repo{root: "/workspace/osprey", uncommitted: true, cmp: comparison{kind: compareBranch}, cache: &changes.HistoryCache{},
		history: changes.History{Base: "main",
			Branch: []changes.Commit{
				{Hash: "2222222aaaa", Short: "2222222", Subject: "Return errors from login", Time: time.Now().Add(-2 * time.Hour), Files: 1, Added: 2, Deleted: 1},
				{Hash: "1111111aaaa", Short: "1111111", Subject: "Add login", Time: time.Now().Add(-26 * time.Hour), Files: 1, Added: 3},
			},
			Earlier: []changes.Commit{{Hash: "0000000aaaa", Short: "0000000", Subject: "First commit", Time: time.Now().AddDate(0, -2, 0)}},
		}}
	return a
}

func TestHistoryInSidebar(t *testing.T) {
	a := historyFixture()
	h := ui.NewTester(a.view, 1280, 820)
	if !h.HasText("All changes since main, with uncommitted") || !h.HasText("Include uncommitted changes") {
		t.Fatal("the window should name the comparison, with the switch")
	}
	click(t, h, "Include uncommitted changes")
	if a.repo.uncommitted || a.repo.source().Head != "HEAD" || !h.HasText("All changes since main") {
		t.Fatal("the switch didn't leave out the uncommitted changes")
	}
	click(t, h, "History")
	if a.sideTab != tabHistory || !h.HasText("On this branch") || !h.HasText("Earlier") {
		t.Fatal("the history tab didn't show the commits")
	}
	a.reviews.SetDone("1111111aaaa", true)
	click(t, h, "Add login")
	if a.repo.cmp != (comparison{kind: compareCommit, commit: "1111111aaaa"}) || a.sideTab != tabHistory {
		t.Fatalf("choosing a commit compared %+v", a.repo.cmp)
	}
	if h.HasText("Include uncommitted changes") {
		t.Fatal("a commit has no changes not committed yet")
	}
	click(t, h, "Uncommitted changes")
	if a.repo.cmp.kind != compareUncommitted || a.repo.source().Base != "" {
		t.Fatal("couldn't go back to the changes not committed yet")
	}
}

func TestHistoryCompactWindow(t *testing.T) {
	a := historyFixture()
	h := ui.NewTester(a.view, 720, 420)
	a.showHistory()
	h.Frame()
	r, ok := h.Find("Add login")
	if !ok || r.Y+r.H > 420 || r.X+r.W > 720 {
		t.Fatalf("the history doesn't fit a small window: %+v", r)
	}
}

func TestDoneFollowsReviews(t *testing.T) {
	a := historyFixture()
	a.repo.cmp = comparison{kind: compareCommit, commit: "1111111aaaa"}
	for _, f := range a.files[:2] {
		a.setReviewed(f, true)
	}
	a.selected = a.files[2].Path
	a.toggleReviewed()
	if !a.reviews.Done("1111111aaaa") {
		t.Fatal("reviewing the commit's last file should check it")
	}
	a.selected = a.files[0].Path
	a.toggleReviewed()
	if a.reviews.Done("1111111aaaa") {
		t.Fatal("unmarking a file should uncheck the commit")
	}
}

func TestAgo(t *testing.T) {
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	for d, want := range map[time.Duration]string{
		10 * time.Second: "just now", 5 * time.Minute: "5 min ago", 90 * time.Minute: "1 hour ago",
		5 * time.Hour: "5 hours ago", 30 * time.Hour: "yesterday", 72 * time.Hour: "3 days ago",
		30 * 24 * time.Hour: "Sep 28", 400 * 24 * time.Hour: "Sep 23, 2025",
	} {
		if got := ago(now.Add(-d), now); got != want {
			t.Errorf("%v: %q, want %q", d, got, want)
		}
	}
}

// Off macOS, where the window's own title bar holds the window's controls,
// the sidebar's tabs take its top row, beside the toolbar, and end on the
// line under it; on macOS they go under that row, which the controls take.
func TestSidebarTabsInTopRow(t *testing.T) {
	defer func(own bool) { ownTitleBar = own }(ownTitleBar)
	for _, own := range []bool{true, false} {
		ownTitleBar = own
		a := historyFixture()
		h := ui.NewTester(a.view, 1280, 820)
		top := float32(0)
		if own {
			top = titleBarHeight
		}
		tab, ok := h.Find("History")
		if !ok {
			t.Fatalf("own title bar %v: no History tab", own)
		}
		if d := tab.Y + tab.H - (top + headerHeight); own && (d < -1 || d > 1) {
			t.Errorf("own title bar: the tab at %+v doesn't end on the toolbar's line, at %v", tab, top+headerHeight)
		}
		if !own && tab.Y < top+headerHeight {
			t.Errorf("macOS: the tab at %+v is in the row of the window's controls", tab)
		}
		click(t, h, "History")
		if a.sideTab != tabHistory {
			t.Errorf("own title bar %v: the History tab didn't open the history", own)
		}
	}
}
