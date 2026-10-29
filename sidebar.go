package main

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"github.com/overcat/osprey/internal/changes"
)

// The sidebar, over the window's material: the files or the history, and
// how far the review is.

// sidebarRowHeight is the height of the sidebar's rows, as Finder's.
const sidebarRowHeight = 26

func (a *app) sidebar(c *ui.Context) {
	side := ui.Column(c).Fill()
	if !c.Vibrancy() {
		side.Background(sidebarFill(c.Theme()))
	}
	side.Children(func() {
		a.sidebarTop(c)
		if a.repo != nil {
			a.sidebarTabs(c)
		}
		if a.repo != nil && a.sideTab == tabHistory {
			a.historyTab(c)
		} else {
			a.filterField(c)
			a.fileScope(c)
			a.fileList(c)
		}
		a.sidebarFooter(c)
	})
}

// sidebarTop is the sidebar's row beside the toolbar, with the button
// hiding the sidebar, and room for macOS's window controls.
func (a *app) sidebarTop(c *ui.Context) {
	bar := windowControls(c)
	ui.Row(c).Height(headerHeight).Shrink(0).Padding(0, 10, 0, bar.Left+8).AlignItems(ui.Center).Justify(ui.End).DragWindow().Children(func() {
		if iconButton(c, iconSidebar, "Hide Sidebar", keySidebar).Clicked() {
			a.sidebarHidden = true
		}
	})
}

// sidebarTabs switch the sidebar between the files and the history: words,
// the chosen one underlined in the accent color.
func (a *app) sidebarTabs(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Gap(18).Padding(0, 18).Shrink(0).BorderWidth(0, 0, 1, 0).BorderColor(t.Border.Alpha(0.6)).Children(func() {
		for _, tab := range []struct {
			tab   int
			label string
		}{{tabFiles, "Files"}, {tabHistory, "History"}} {
			chosen := a.sideTab == tab.tab
			b := ui.ButtonBase(c.Key(tab.label)).Padding(4, 0, 8, 0).Label(tab.label)
			if b.Clicked() {
				a.sideTab = tab.tab
			}
			text := t.TextMuted
			switch {
			case chosen:
				text = t.Text
				b.BorderWidth(0, 0, 2, 0).BorderColor(t.Accent)
			case b.Hovered():
				text = t.Text
			}
			// Bold either way, so that choosing doesn't move the words.
			b.Children(func() { ui.Text(c, tab.label).FontSize(13).FontWeight(600).TextColor(text) })
		}
	})
}

// filterField filters the files by their paths, as a search field.
func (a *app) filterField(c *ui.Context) {
	t := c.Theme()
	field := ui.Row(c).Height(28).Margin(10, 10, 6, 10).Padding(0, 4, 0, 8).Gap(6).AlignItems(ui.Center).Shrink(0).
		Radius(7).Background(fieldFill(t)).Cursor(ui.CursorText)
	field.Children(func() {
		ui.Icon(c, iconSearch).FontSize(13).TextColor(t.TextMuted)
		previous := a.filter
		in := ui.TextInputBase(c.Key("filter"), &a.filter).Placeholder("Filter").Label("Filter files").
			Grow(1).FontSize(13).FocusBind(&a.focus, "filter")
		if a.filter != "" {
			clear := iconButton(c, iconClear, "Clear", "").Size(20, 20)
			if clear.Clicked() || in.Shortcut(0, ui.KeyEscape) {
				a.filter = ""
			}
		}
		if previous != a.filter {
			a.filtered()
		}
	})
}

// fileScope chooses all the files or those to review, over the section of
// the files.
func (a *app) fileScope(c *ui.Context) {
	t := c.Theme()
	reviewed, _, _ := a.counts()
	scope := a.reviewScope
	ui.Row(c).Padding(2, 10, 4, 10).Shrink(0).Children(func() {
		segmentedFill(c, "scope", &scope, "All", "To Review").Grow(1)
	})
	if scope != a.reviewScope {
		a.setReviewScope(scope)
	}
	ui.Row(c).Height(26).Padding(0, 6, 0, 18).Gap(6).AlignItems(ui.Center).Shrink(0).Children(func() {
		title := "Changed files"
		count := len(a.files)
		if a.reviewScope == scopeToReview {
			title, count = "To review", len(a.files)-reviewed
		}
		ui.Text(c, title).FontSize(11).FontWeight(600).TextColor(t.TextMuted)
		ui.Text(c, fmt.Sprint(count)).FontSize(11).TextColor(t.TextMuted.Alpha(0.7)).FontFeatures("tnum")
		ui.Box(c).Grow(1)
		if a.reviewScope == scopeToReview && len(a.retained) > 0 {
			hide := ui.ButtonBase(c.Key("hide-reviewed")).Padding(2, 6).Radius(5).TextColor(t.Accent).Label("Hide Reviewed").Tooltip("Remove the files just reviewed from this list")
			if hide.Hovered() {
				hide.Background(t.SurfaceHover)
			}
			if hide.Clicked() {
				a.setReviewScope(scopeToReview)
			}
			hide.Children(func() { ui.Text(c, "Hide Reviewed").FontSize(11).FontWeight(500) })
		}
		icon, label := iconList, "Show as List"
		if a.flat {
			icon, label = iconTree, "Show as Tree"
		}
		if iconButton(c, icon, label, "").Size(24, 22).Clicked() {
			a.flat = !a.flat
			a.reveal = true
		}
	})
}

func (a *app) fileList(c *ui.Context) {
	t := c.Theme()
	files := a.visible()
	var rows []row
	if a.flat {
		rows = listRows(files)
	} else {
		collapsed := a.collapsed
		if a.filter != "" {
			collapsed = nil // show every file found
		}
		rows = treeRows(files, collapsed, a.reviewed)
	}
	if a.reveal {
		for i, r := range rows {
			if r.f != nil && r.f.Path == a.selected {
				a.list.ScrollIntoView(i)
				break
			}
		}
		a.reveal = false
	}
	ui.List(c, &a.list, len(rows), func(i int) {
		if r := rows[i]; r.f != nil {
			a.fileRow(c, r)
		} else {
			a.dirRow(c, r)
		}
	}).Grow(1).PaddingY(2).Children(func() {
		if len(rows) == 0 && a.loaded && len(a.files) > 0 {
			label := "No files match"
			if a.reviewScope == scopeToReview && a.filter == "" {
				label = "You're all caught up."
			}
			ui.Text(c, label).FontSize(12).TextColor(t.TextMuted).Padding(10, 18)
		}
	})
}

// rowBase is a row of the sidebar's list: chosen, it fills with the accent
// color, as a source list's.
func rowBase(c *ui.Context, key any, depth int, selected bool) ui.Element {
	t := c.Theme()
	b := ui.ButtonBase(c.Key(key)).Height(sidebarRowHeight).Margin(0, 10).Padding(0, 8, 0, 6+float32(depth)*14).
		Gap(6).AlignItems(ui.Center).Radius(6)
	switch {
	case selected:
		b.Background(t.Accent).TextColor(t.AccentText)
	case b.Hovered():
		b.Background(t.SurfaceHover)
	}
	return b
}

func (a *app) dirRow(c *ui.Context, r row) {
	t := c.Theme()
	open := !a.collapsed[r.dir] || a.filter != ""
	b := rowBase(c, "dir:"+r.dir, r.depth, false).Label(r.name)
	if b.Clicked() {
		if a.collapsed[r.dir] {
			delete(a.collapsed, r.dir)
		} else {
			a.collapsed[r.dir] = true
		}
	}
	b.Children(func() {
		chevron := iconChevron
		if open {
			chevron = iconChevronDn
		}
		ui.Icon(c, chevron).FontSize(11).TextColor(t.TextMuted.Alpha(0.8))
		ui.Text(c, r.name).SingleLine().Grow(1).Shrink(1).FontSize(13).FontWeight(500)
		label := fmt.Sprint(r.files)
		if r.reviewed > 0 {
			label = fmt.Sprintf("%d/%d", r.reviewed, r.files)
		}
		ui.Text(c, label).FontSize(11).TextColor(t.TextMuted.Alpha(0.8)).FontFeatures("tnum")
	})
}

func (a *app) fileRow(c *ui.Context, r row) {
	t := c.Theme()
	f := r.f
	reviewed := a.reviewed(f)
	selected := f.Path == a.selected
	depth := r.depth
	if !a.flat {
		depth++ // past the chevrons of directories
	}
	b := rowBase(c, f, depth, selected).Label(f.Path)
	if b.Clicked() {
		a.show(f)
	}
	b.Children(func() {
		statusSquare(c, f.Status)
		name, dir := r.name, ""
		if a.flat {
			dir, name = splitPath(f.Path)
		}
		text := ui.Text(c, name).SingleLine().Shrink(1).FontSize(13)
		if reviewed && !selected {
			text.TextColor(t.TextMuted)
		}
		if f.Status == changes.StatusDeleted {
			text.Strikethrough()
		}
		if dir != "" {
			muted := t.TextMuted
			if selected {
				muted = t.AccentText.Alpha(0.75)
			}
			ui.Text(c, dir).SingleLine().Shrink(100).FontSize(11).TextColor(muted)
		}
		ui.Box(c).Grow(1)
		if reviewed {
			mark := ui.Icon(c, iconCheck).FontSize(13).TextColor(t.Accent)
			if selected {
				mark.TextColor(t.AccentText)
			}
		} else {
			lineStats(c, f.added, f.deleted, 11, selected)
		}
	})
}

// sidebarFooter shows how far the review is, with the switch of the
// changes not committed yet.
func (a *app) sidebarFooter(c *ui.Context) {
	t := c.Theme()
	toggle := a.repo != nil && a.repo.cmp.kind == compareBranch && a.repo.checkedOut()
	if !toggle && len(a.files) == 0 {
		return
	}
	ui.Column(c).Padding(10, 14, 12, 14).Gap(8).Shrink(0).BorderWidth(1, 0, 0, 0).BorderColor(t.Border.Alpha(0.6)).Children(func() {
		if toggle {
			on := a.repo.uncommitted
			if ui.Checkbox(c, &on, "Include uncommitted changes").FontSize(12).Changed() {
				a.setUncommitted(on)
			}
		}
		if len(a.files) == 0 {
			return
		}
		reviewed, _, _ := a.counts()
		ui.Column(c).Gap(6).Children(func() {
			label := fmt.Sprintf("%d of %d reviewed", reviewed, len(a.files))
			if reviewed == len(a.files) {
				label = "All files reviewed"
			}
			ui.Text(c, label).FontSize(11).TextColor(t.TextMuted).FontFeatures("tnum")
			track := ui.RGBA(0, 0, 0, 0.1)
			if t.Dark {
				track = ui.RGBA(255, 255, 255, 0.12)
			}
			ui.Box(c).Height(4).FillWidth().Radius(2).Background(track).Children(func() {
				if reviewed > 0 {
					ui.Box(c).Height(4).WidthPercent(100 * float32(reviewed) / float32(len(a.files))).Radius(2).Background(t.Accent)
				}
			})
		})
	})
}
