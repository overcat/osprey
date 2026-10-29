package main

import (
	"slices"
	"strings"

	"github.com/egoist/mygo/ui/editor"
)

// What the user does with the files: choosing which show, marking them
// reviewed, and going from one change to the next.

// reviewed reports whether f is reviewed, as it is now.
func (a *app) reviewed(f *file) bool { return a.reviews.Reviewed(f.Path, f.hash) }

// setReviewed marks f reviewed, or not.
func (a *app) setReviewed(f *file, on bool) {
	a.reviews.Set(f.Path, f.hash, on)
	a.syncDone()
}

// syncDone records whether every change of the commit shown is reviewed,
// which the history marks.
func (a *app) syncDone() {
	if a.repo == nil || a.repo.cmp.kind != compareCommit || !a.loaded || a.loadErr != nil {
		return
	}
	reviewed, _, _ := a.counts()
	a.reviews.SetDone(a.repo.cmp.commit, len(a.files) > 0 && reviewed == len(a.files))
}

// counts returns how many files are reviewed, and the lines added and
// deleted in all.
func (a *app) counts() (reviewed, added, deleted int) {
	for _, f := range a.files {
		if a.reviewed(f) {
			reviewed++
		}
		added += f.added
		deleted += f.deleted
	}
	return
}

// visible returns the files the filter and the review scope keep, in the
// sidebar's order.
func (a *app) visible() []*file {
	files := filterFiles(a.files, a.filter)
	if a.reviewScope == scopeAll {
		return files
	}
	out := make([]*file, 0, len(files))
	for _, f := range files {
		if !a.reviewed(f) || a.retained[f.Path] == f.hash {
			out = append(out, f)
		}
	}
	return out
}

func (a *app) current() *file {
	for _, f := range a.files {
		if f.Path == a.selected {
			return f
		}
	}
	return nil
}

// syncSelection keeps the file shown among the visible ones.
func (a *app) syncSelection() {
	files := a.visible()
	if slices.ContainsFunc(files, func(f *file) bool { return f.Path == a.selected }) {
		return
	}
	a.selected = ""
	if len(files) > 0 {
		a.show(files[0])
	}
}

// filtered follows a change of the filter or the review scope, which
// lets go of the files retained.
func (a *app) filtered() {
	a.retained = nil
	a.syncSelection()
	a.feed.target = a.selected
}

// showAll clears the filter and shows every file, reviewed or not.
func (a *app) showAll() {
	a.filter = ""
	a.setReviewScope(scopeAll)
}

func (a *app) setReviewScope(scope int) {
	a.reviewScope = scope
	a.filtered()
	a.feed.hunk = nil
	a.reveal = true
}

func (a *app) setBrowseMode(mode int) {
	if a.browseMode == mode {
		return
	}
	a.feed.hunk = nil
	if mode == modeSingle {
		a.focusEntry = a.selected
	} else if a.selected != a.focusEntry {
		a.feed.target = a.selected
	}
	a.browseMode = mode
}

// show goes to f: in the feed, scrolling to it.
func (a *app) show(f *file) {
	if f == nil {
		return
	}
	if a.browseMode == modeContinuous {
		f.folded = false
		a.feed.target = f.Path
	}
	a.selectFile(f)
}

// selectFile selects f, opening the directories around it, without
// scrolling the feed: as the feed's scrolling selects the file on top.
func (a *app) selectFile(f *file) {
	a.selected = f.Path
	for dir := range a.collapsed {
		if strings.HasPrefix(f.Path, dir) {
			delete(a.collapsed, dir)
		}
	}
	a.reveal = true
}

// step selects the file n after the one shown among the visible ones,
// those not reviewed only with unreviewed.
func (a *app) step(n int, unreviewed bool) {
	files := a.visible()
	if len(files) == 0 {
		return
	}
	i := slices.IndexFunc(files, func(f *file) bool { return f.Path == a.selected })
	for range files {
		switch {
		case i < 0 && n > 0:
			i = 0
		case i < 0:
			i = len(files) - 1
		default:
			i = (i + n + len(files)) % len(files)
		}
		if !unreviewed || !a.reviewed(files[i]) {
			a.show(files[i])
			return
		}
	}
}

// markInPlace marks f reviewed, or not, keeping it in the feed of files to
// review, so that marking it cannot move the text under the reader's eyes.
func (a *app) markInPlace(f *file) {
	on := !a.reviewed(f)
	if on && a.reviewScope == scopeToReview {
		if a.retained == nil {
			a.retained = map[string]string{}
		}
		a.retained[f.Path] = f.hash
	} else {
		delete(a.retained, f.Path)
	}
	a.setReviewed(f, on)
}

// toggleReviewed marks the file shown reviewed, and goes on to the next
// one to review, or unmarks it.
func (a *app) toggleReviewed() {
	f := a.current()
	if f == nil {
		return
	}
	on := !a.reviewed(f)
	// Find the next file before the current one leaves the pending list.
	if on {
		a.step(1, true)
	}
	a.setReviewed(f, on)
	a.syncSelection()
}

// findInFile opens the find bar of the file shown, in the diff or the
// whole file, as its editor's own key would.
func (a *app) findInFile() {
	f := a.current()
	if f == nil || !f.hasText() {
		return
	}
	if a.page == pageFile {
		f.fullView(a.browseMode).Run(editor.Find)
	} else {
		f.diffView(a.browseMode).Run(editor.Find)
	}
}

// toggleShortcuts shows the panel of the keyboard shortcuts, or hides it.
func (a *app) toggleShortcuts() { a.shortcutsOpen = !a.shortcutsOpen }

// stepHunk moves to the change n after the caret's in the diff.
func (a *app) stepHunk(n int) {
	f := a.current()
	if f == nil || !f.hasText() {
		return
	}
	a.page = pageDiff
	d := f.diffView(a.browseMode)
	hunks := d.Hunks()
	if len(hunks) == 0 {
		return
	}
	i, inside := d.CurrentHunk()
	switch {
	case n > 0:
		i++
	case inside || i < 0:
		i--
	}
	i = min(max(i, 0), len(hunks)-1)
	d.GoToHunk(i)
	if a.browseMode == modeContinuous {
		a.revealHunk(f, i)
	}
}
