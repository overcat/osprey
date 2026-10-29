package main

import (
	"github.com/egoist/mygo/ui/editor"

	"github.com/overcat/osprey/internal/changes"
)

// file is a changed file as the window shows it: its texts, and the views
// of them that each reading mode keeps across frames and reloads.
type file struct {
	changes.Change
	hash           string
	added, deleted int
	// old and new are the texts compared, nil when they don't show (Note).
	old, new *editor.Buffer
	views    [2]fileViews // by browse mode
	folded   bool         // collapsed in the continuous feed
	focused  bool         // one of its editors in the feed had the keyboard
}

// fileViews are a file's diff and whole-file editor in one reading mode.
// The modes share the texts, but each keeps its own carets, folds and
// scrolling, so that focusing on one file doesn't move the feed.
type fileViews struct {
	diff *editor.DiffState
	full *editor.State
}

// newFile makes the buffers of a change. Their texts are the buffers'
// from then on, so the change lets go of its own.
func newFile(ch changes.Change) *file {
	f := &file{Change: ch, hash: ch.Hash()}
	if ch.Note == "" {
		lang := editor.LanguageFor(ch.Path)
		f.old, f.new = editor.NewBuffer(ch.Old), editor.NewBuffer(ch.New)
		f.old.SetLanguage(lang)
		f.new.SetLanguage(lang)
	}
	f.Old, f.New = "", ""
	return f
}

// hasText reports whether the file's texts show, unlike a binary file's.
func (f *file) hasText() bool { return f.old != nil }

// oneSided reports whether one side of the file is empty, as an added or
// deleted file's, which shows as a unified diff.
func (f *file) oneSided() bool {
	return f.Status == changes.StatusAdded || f.Status == changes.StatusDeleted
}

// diffView returns the file's diff in a reading mode: in the feed, as
// high as its lines. It is nil for a file without text.
func (f *file) diffView(mode int) *editor.DiffState {
	v := &f.views[mode]
	if v.diff == nil && f.hasText() {
		v.diff = &editor.DiffState{Old: f.old, New: f.new, FitContent: mode == modeContinuous}
	}
	return v.diff
}

// fullView returns the editor showing the whole file in a reading mode:
// the new text with its changes marked in the gutter, or the old one of a
// deleted file.
func (f *file) fullView(mode int) *editor.State {
	v := &f.views[mode]
	if v.full == nil {
		v.full = &editor.State{ReadOnly: true, FitContent: mode == modeContinuous}
	}
	if f.Status == changes.StatusDeleted {
		v.full.Buffer, v.full.DiffBase = f.old, nil
	} else {
		v.full.Buffer, v.full.DiffBase = f.new, f.old
	}
	return v.full
}

// update takes the texts of g, the file loaded again, keeping f's
// buffers, so that its views keep their place as an agent edits it.
func (f *file) update(g *file) {
	f.folded = false
	if f.hasText() && g.hasText() {
		resetText(f.old, g.old)
		resetText(f.new, g.new)
		if d := f.views[modeContinuous].diff; d != nil {
			d.Annotations = nil // the anchor of a change it went to
		}
	} else {
		f.old, f.new, f.views = g.old, g.new, g.views
	}
	f.Change, f.hash, f.added, f.deleted = g.Change, g.hash, g.added, g.deleted
}

func resetText(b, to *editor.Buffer) {
	if s := to.String(); s != b.String() {
		b.Reset(s)
	}
}
