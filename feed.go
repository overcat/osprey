package main

import (
	"slices"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// followFeed reads the last layout before building the sidebar. Explicit
// navigation takes precedence; following a scroll must never scroll back.
func (a *app) followFeed() {
	if a.browseMode != modeContinuous || a.feed.target != "" {
		return
	}
	first, last := a.feed.list.Visible()
	if last < first || first/2 >= len(a.feed.rows) {
		return
	}
	f := a.feed.rows[first/2]
	if f.Path == a.feed.top {
		return
	}
	a.feed.top = f.Path
	if a.selected != f.Path && slices.Contains(a.visible(), f) {
		a.selectFile(f)
	}
}

// feedView shows the files one after another. Each has two virtual rows:
// its heading, which stays pinned while its diff scrolls, and the diff, as
// high as its lines. The list keeps its place by path as files are loaded
// again.
func (a *app) feedView(c *ui.Context, files []*file) {
	t := c.Theme()
	a.feed.list.Key = func(i int) any {
		return struct {
			path string
			part int
		}{files[i/2].Path, i % 2}
	}
	a.feed.list.Header = func(i int) bool { return i%2 == 0 }
	if a.feed.target != "" {
		if i := slices.IndexFunc(files, func(f *file) bool { return f.Path == a.feed.target }); i >= 0 {
			a.feed.list.ScrollTo(2*i, ui.Start)
		}
		a.feed.target = ""
	}
	a.feed.rows = files
	height := contentHeight(c)
	ui.List(c, &a.feed.list, len(files)*2, func(i int) {
		f := files[i/2]
		if i%2 == 0 {
			a.fileHeader(c, f, true)
			return
		}
		body := ui.Column(c)
		last := i == len(files)*2-1
		// Let the last heading reach the top even when its diff is short.
		// Earlier short files flow directly into one another.
		if last {
			body.MinHeight(max(80, height-headerHeight-fileHeadingHeight))
		}
		body.Children(func() {
			if !f.folded {
				a.feedBody(c, f)
			}
			if last {
				reviewed, _, _ := a.counts()
				label := "End of changes"
				if reviewed == len(a.files) {
					label = "All files reviewed"
				}
				ui.Text(c, label).FontSize(11).TextColor(t.TextMuted).Padding(20).TextAlign(ui.Center).FillWidth()
			}
		})
	}).Grow(1).MinHeight(0).Label("Continuous changes")
}

func (a *app) feedBody(c *ui.Context, f *file) {
	t := c.Theme()
	if !f.hasText() {
		ui.Text(c, f.Note+": no preview").FontSize(12).TextColor(t.TextMuted).Padding(16, 20)
		return
	}
	if a.page == pageFile {
		a.followEditorFocus(editor.View(c.Key(f), a.fullFor(f, t)).FillWidth(), f)
		return
	}
	d := a.diffFor(f, t)
	if len(d.Hunks()) == 0 {
		ui.Text(c, "No changes in the text").FontSize(12).TextColor(t.TextMuted).Padding(16, 20)
		return
	}
	d.RenderAnnotation = func(c *ui.Context, _ editor.DiffAnnotation) {
		// Include room above the changed line for the pinned file heading.
		anchor := ui.Box(c).Absolute().Top(-fileHeadingHeight - 44).Height(fileHeadingHeight + 66).Width(1).Label("Change anchor " + f.Path)
		if a.feed.hunk == f {
			anchor.ScrollIntoView()
			a.feed.hunk = nil
		}
	}
	a.followEditorFocus(editor.DiffView(c.Key(f), d).FillWidth(), f)
}

// Clicking into another visible file makes keyboard review commands act
// on that file. Scrolling to the next heading takes over again.
func (a *app) followEditorFocus(e ui.Element, f *file) {
	focused := e.FocusWithin()
	if focused && !f.focused {
		a.selectFile(f)
	}
	f.focused = focused
}

// revealHunk scrolls the feed to change i of f. A zero-height annotation
// at the change anchors it exactly, even for wrapped lines and unified
// deletions, without estimating the diff's geometry.
func (a *app) revealHunk(f *file, i int) {
	d := f.diffView(modeContinuous)
	h := d.Hunks()[i]
	side, line := editor.DiffNew, h.NewStart
	if h.NewStart == h.NewEnd {
		side, line = editor.DiffOld, h.OldStart
	}
	d.Annotations = []editor.DiffAnnotation{{Side: side, Line: line, Key: "navigation"}}
	f.folded = false
	a.feed.hunk = f
}
