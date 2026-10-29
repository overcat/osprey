package main

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/overcat/osprey/internal/changes"
)

// The sidebar's tabs.
const (
	tabFiles = iota
	tabHistory
)

// showHistory shows the history in the sidebar, or the files again.
func (a *app) showHistory() {
	if a.repo != nil {
		a.sideTab = flip(a.sideTab)
		a.sidebarHidden = false
	}
}

// historyTab is the sidebar's tab of the history.
func (a *app) historyTab(c *ui.Context) {
	ui.Scroll(c).Grow(1).Padding(4, 0).Label("History list").Children(func() { a.historyEntries(c) })
}

// historyEntries lists what to compare: the branch's changes, those not
// committed yet, and the commits.
func (a *app) historyEntries(c *ui.Context) {
	t := c.Theme()
	r := a.repo
	h := &r.history
	if base := r.branchBase(); base != "" {
		detail := fmt.Sprintf("%s since %s", plural(len(h.Branch), "commit"), base)
		if r.head != "" {
			detail = plural(len(h.Branch), "commit")
		}
		a.historyRow(c, comparison{kind: compareBranch}, "All changes", false, func(muted ui.Color) {
			ui.Text(c, detail).FontSize(11).TextColor(muted).SingleLine()
		})
	}
	if r.checkedOut() {
		a.historyRow(c, comparison{kind: compareUncommitted}, "Uncommitted changes", false, func(muted ui.Color) {
			ui.Text(c, "Not committed yet").FontSize(11).TextColor(muted).SingleLine()
		})
	}
	if len(h.Branch) > 0 {
		title := "On this branch"
		switch {
		case r.base != "":
			title = "Since " + r.base
		case h.Upstream:
			title = "Not pushed yet"
		}
		historySection(c, title, len(h.Branch))
		for _, cm := range h.Branch {
			a.commitRow(c, cm)
		}
	}
	if len(h.Earlier) > 0 {
		title := "Earlier"
		if r.branchBase() == "" {
			title = "Recent commits"
		}
		historySection(c, title, -1)
		for _, cm := range h.Earlier {
			a.commitRow(c, cm)
		}
	}
	if len(h.Branch)+len(h.Earlier) == 0 && a.loaded {
		ui.Text(c, "No commits yet").FontSize(12).TextColor(t.TextMuted).Padding(10, 18)
	}
}

// historySection heads a section of the history, with its count when not
// negative.
func historySection(c *ui.Context, title string, count int) {
	t := c.Theme()
	ui.Row(c).Gap(6).Padding(10, 18, 2, 18).AlignItems(ui.Center).Children(func() {
		ui.Text(c, title).FontSize(11).FontWeight(600).TextColor(t.TextMuted).SingleLine().Shrink(1)
		if count >= 0 {
			ui.Text(c, fmt.Sprint(count)).FontSize(11).TextColor(t.TextMuted.Alpha(0.7)).FontFeatures("tnum")
		}
	})
}

func (a *app) commitRow(c *ui.Context, cm changes.Commit) {
	cmp := comparison{kind: compareCommit, commit: cm.Hash}
	row := a.historyRow(c, cmp, cm.Subject, a.reviews.Done(cm.Hash), func(muted ui.Color) {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, cm.Short).Font("monospace").FontSize(11).TextColor(muted)
			ui.Text(c, ago(cm.Time, time.Now())).FontSize(11).TextColor(muted).SingleLine().Shrink(1)
			ui.Box(c).Grow(1)
			lineStats(c, cm.Added, cm.Deleted, 11, a.repo.cmp == cmp)
		})
	})
	row.Tooltip(fmt.Sprintf("%s\n%s, %s", cm.Subject, cm.Author, cm.Time.Format("Jan 2, 2006 at 15:04")))
}

// historyRow is a row of the history, choosing cmp, in two lines: title,
// and what detail builds in the muted color it is given; done checks it.
func (a *app) historyRow(c *ui.Context, cmp comparison, title string, done bool, detail func(muted ui.Color)) ui.Element {
	t := c.Theme()
	chosen := a.repo.cmp == cmp
	b := ui.ButtonBase(c.Key(cmp)).Margin(0, 10).Padding(4, 8).Radius(6).Label(title)
	muted := t.TextMuted
	switch {
	case chosen:
		b.Background(t.Accent).TextColor(t.AccentText)
		muted = t.AccentText.Alpha(0.75)
	case b.Hovered():
		b.Background(t.SurfaceHover)
	}
	if b.Clicked() {
		a.compare(cmp)
	}
	b.Children(func() {
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(1).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Text(c, title).FontSize(13).SingleLine().Grow(1).Shrink(1)
				if done {
					check := ui.Icon(c, iconCheck).FontSize(13).TextColor(t.Accent).Tooltip("Every change reviewed")
					if chosen {
						check.TextColor(t.AccentText)
					}
				}
			})
			detail(muted)
		})
	})
	return b
}

func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

// ago says how long before now t was, as a history lists commits.
func ago(t, now time.Time) string {
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%d min ago", int(d.Minutes()))
	case d < 2*time.Hour:
		return "1 hour ago"
	case d < 24*time.Hour:
		return fmt.Sprintf("%d hours ago", int(d.Hours()))
	case d < 48*time.Hour:
		return "yesterday"
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%d days ago", int(d.Hours()/24))
	case t.Year() == now.Year():
		return t.Format("Jan 2")
	}
	return t.Format("Jan 2, 2006")
}
