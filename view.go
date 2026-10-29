package main

import (
	"fmt"
	"runtime"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	"github.com/overcat/osprey/internal/changes"
)

// headerHeight is the height of the toolbar along the top, under the
// window controls, which line up with its own: mygo.WindowOptions'
// TrafficLightPosition centers them in it on macOS, and TitleBarHeight
// makes Windows' as tall as it, and centers Linux's.
const headerHeight = 52

// fileHeadingHeight is the height of the bar over each file.
const fileHeadingHeight = 38

func (a *app) view(c *ui.Context) {
	c.SetTheme(nativeTheme(c.Theme()))
	// The window's material shows wherever the view draws no background:
	// under the sidebar.
	if c.Vibrancy() {
		c.Root().Background(ui.Transparent)
	}
	w, _ := c.Size()
	a.sidebarWidth = min(max(a.sidebarWidth, 220), min(400, max(220, w-440)))
	a.followFeed()
	a.syncSelection()
	switch {
	case a.src == nil:
		a.welcome(c)
	case a.sidebarHidden:
		a.main(c)
	default:
		ui.Split(c, &a.sidebarWidth, func() { a.sidebar(c) }, func() { a.main(c) }).Fill()
	}
	ui.Modal(c, &a.shortcutsOpen, func() { shortcutsPanel(c) })
}

// main is the pane of the files: its toolbar, then the feed or the file
// shown.
func (a *app) main(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		a.toolbar(c)
		if a.loadErr != nil && len(a.files) > 0 {
			ui.Row(c).Padding(6, 16).Gap(8).AlignItems(ui.Center).Shrink(0).Background(t.Danger.Alpha(0.1)).Children(func() {
				ui.Icon(c, iconWarning).FontSize(13).TextColor(t.Danger)
				ui.Text(c, "Couldn't refresh: "+a.loadErr.Error()).FontSize(12).SingleLine().Shrink(1)
			})
		}
		if a.browseMode == modeContinuous && a.loaded && a.current() != nil {
			a.feedView(c.Key("continuous"), a.visible())
			return
		}
		if f := a.current(); f != nil && a.loaded {
			a.fileHeader(c.Key("single"), f, false)
		}
		a.body(c)
	})
}

func (a *app) paneWidth(c *ui.Context) float32 {
	w, _ := c.Size()
	if a.sidebarHidden {
		return w
	}
	return w - a.sidebarWidth - 1
}

// toolbar is the window's title and its controls, in the title bar.
func (a *app) toolbar(c *ui.Context) {
	t := c.Theme()
	bar := c.TitleBar()
	f := a.current()
	// room is the pane's width but for the window's buttons, which take
	// the right end of the title bar off macOS.
	room := a.paneWidth(c) - bar.Right
	left := float32(16)
	if a.sidebarHidden {
		left = bar.Left + 8
	}
	ui.Row(c).Height(headerHeight).Shrink(0).Padding(0, bar.Right+10, 0, left).Gap(4).AlignItems(ui.Center).
		BorderWidth(0, 0, 1, 0).BorderColor(t.Border).DragWindow().Children(func() {
		if a.sidebarHidden && iconButton(c, iconSidebar, "Show Sidebar", keySidebar).Clicked() {
			a.sidebarHidden = false
		}
		a.windowTitle(c)
		if f != nil && a.loaded {
			a.fileOptions(c, f, room)
			toolbarSeparator(c)
		}
		mode := a.browseMode
		if room >= 820 {
			segmented(c, "mode", &mode, "Continuous", "Single File").Tooltip("Continuous or single file")
		} else if toggleIcon(c, iconFocus, "Single File", "", mode == modeSingle).Clicked() {
			mode = flip(mode)
		}
		if mode != a.browseMode {
			a.setBrowseMode(mode)
		}
		if room >= 600 {
			if iconButton(c, iconRefresh, "Refresh", keyRefresh).Clicked() {
				a.refresh()
			}
			if iconButton(c, iconFolderOpen, "Open Repository", keyOpen).Clicked() {
				go a.chooseRepo()
			}
		}
		a.menuButton(c)
	})
}

// menuInWindow is whether the window shows a button for the menus: off
// macOS, menu bars belong to windows, and one without its title bar, as
// Osprey's, shows none; only Alt or F10 would open its menus.
var menuInWindow = runtime.GOOS != "darwin"

// menuButton opens the menu bar's menus under it, where the window shows
// no menu bar.
func (a *app) menuButton(c *ui.Context) {
	if !menuInWindow {
		return
	}
	b := iconButton(c, iconMenu, "Menu", "")
	if b.Clicked() && a.win != nil {
		r := b.Bounds()
		go a.appMenu.PopupAt(a.win, int(r.X), int(r.Y+r.H))
	}
}

// windowTitle names the repository and its branch, over what is compared,
// as a window's title and subtitle.
func (a *app) windowTitle(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(1).Margin(0, 8, 0, 0).Children(func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Text(c, a.src.Title()).FontSize(13).FontWeight(600).SingleLine().Shrink(1)
			if a.branch != "" {
				ui.Icon(c, iconBranch).FontSize(11).TextColor(t.TextMuted)
				ui.Text(c, a.branch).FontSize(11).TextColor(t.TextMuted).SingleLine().Shrink(1)
			}
		})
		label := a.comparisonLabel()
		ui.Text(c, label).FontSize(11).TextColor(t.TextMuted).SingleLine().Tooltip(label)
	})
}

// comparisonLabel says what is compared.
func (a *app) comparisonLabel() string {
	if a.repo == nil {
		return a.src.Subtitle()
	}
	label := a.repo.label(a.repo.cmp)
	if a.repo.cmp.kind == compareBranch && a.repo.checkedOut() && a.repo.uncommitted {
		label += ", with uncommitted"
	}
	return label + changes.PathsLabel(a.repo.paths)
}

// fileOptions are the toolbar's controls of how the file shows. Where
// there is no room for the toggles of the view, the View menu has them.
func (a *app) fileOptions(c *ui.Context, f *file, room float32) {
	segmented(c, "page", &a.page, "Diff", "Full File").Tooltip("Diff or full file")
	diff := a.page == pageDiff && f.hasText()
	if room >= 420 {
		if diff && !f.oneSided() {
			if room >= 900 {
				segmented(c, "layout", &a.layout, "Split", "Unified").Tooltip("Split or unified")
			} else if toggleIcon(c, iconSplit, "Unified", "", a.layout == layoutUnified).Clicked() {
				a.layout = flip(a.layout)
			}
		}
		if diff && toggleIcon(c, iconExpand, "Show unchanged lines", "", a.expand).Clicked() {
			a.expand = !a.expand
		}
		if toggleIcon(c, iconWrap, "Wrap lines", "", a.wrap).Clicked() {
			a.wrap = !a.wrap
		}
	}
	if diff {
		toolbarSeparator(c)
		a.hunkNav(c, f, room)
	}
}

// hunkNav goes to the changes of f, and counts them where there is room.
func (a *app) hunkNav(c *ui.Context, f *file, room float32) {
	t := c.Theme()
	d := f.diffView(a.browseMode)
	n := len(d.Hunks())
	if n == 0 {
		return
	}
	if iconButton(c, iconChevronUp, "Previous change", keyPreviousChange).Clicked() {
		a.stepHunk(-1)
	}
	if room >= 900 {
		label := plural(n, "change")
		if i, inside := d.CurrentHunk(); inside {
			label = fmt.Sprintf("%d of %d", i+1, n)
		}
		ui.Text(c, label).FontSize(11).TextColor(t.TextMuted).FontFeatures("tnum").SingleLine().Shrink(0)
	}
	if iconButton(c, iconChevronDn, "Next change", keyNextChange).Clicked() {
		a.stepHunk(1)
	}
}

// codeLineHeight spaces the lines of code, in lines of text.
const codeLineHeight = 1.6

// diffFor returns f's diff in the reading mode, with the view's options.
func (a *app) diffFor(f *file, t *ui.Theme) *editor.DiffState {
	d := f.diffView(a.browseMode)
	d.Layout = editor.DiffSplit
	if a.layout == layoutUnified || f.oneSided() {
		d.Layout = editor.DiffUnified
	}
	d.Wrap, d.ExpandUnchanged = a.wrap, a.expand
	d.Theme, d.LineHeight = codeTheme(t), codeLineHeight
	return d
}

// fullFor returns the editor of the whole of f in the reading mode, with
// the view's options.
func (a *app) fullFor(f *file, t *ui.Theme) *editor.State {
	s := f.fullView(a.browseMode)
	s.SoftWrap = a.wrap
	s.Theme, s.LineHeight = codeTheme(t), codeLineHeight
	return s
}

// fileHeader is the bar over a file: what changed, and whether it is
// reviewed. In the feed it folds the file and focuses on it too.
func (a *app) fileHeader(c *ui.Context, f *file, inFeed bool) {
	t := c.Theme()
	ui.Row(c).Height(fileHeadingHeight).Shrink(0).Padding(0, 12, 0, 8).Gap(8).AlignItems(ui.Center).
		Background(t.Surface).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Label("File header " + f.Path).Children(func() {
		if inFeed {
			icon, action := iconChevronDn, "Collapse "
			if f.folded {
				icon, action = iconChevron, "Expand "
			}
			if iconButton(c, icon, action+f.Path, "").Size(22, 22).Clicked() {
				f.folded = !f.folded
				a.selectFile(f)
			}
		}
		statusSquare(c, f.Status)
		dir, name := splitPath(f.Path)
		ui.Row(c).Grow(1).Shrink(1).MinWidth(0).AlignItems(ui.Center).Children(func() {
			if dir != "" {
				ui.Text(c, dir).FontSize(12.5).TextColor(t.TextMuted).SingleLine().Shrink(100)
			}
			ui.Text(c, name).FontSize(12.5).FontWeight(600).SingleLine().Shrink(1).Tooltip(f.Path)
			if f.OldPath != "" {
				ui.Text(c, "from "+f.OldPath).FontSize(11.5).TextColor(t.TextMuted).SingleLine().Shrink(100).Margin(0, 0, 0, 8)
			}
		})
		lineStats(c, f.added, f.deleted, 12, false)
		if inFeed && iconButton(c, iconFocus, "Focus on "+f.Path, "").Clicked() {
			a.selectFile(f)
			a.setBrowseMode(modeSingle)
		}
		on := a.reviewed(f)
		if ui.Checkbox(c.Key("review"), &on, "Reviewed").FontSize(12.5).Label("Reviewed " + f.Path).
			Tooltip("Reviewed: " + keyLabel(keyReviewed) + " marks it and goes to the next file").Changed() {
			a.markInPlace(f)
		}
	})
}

// body is the file shown alone, or why there is none.
func (a *app) body(c *ui.Context) {
	t := c.Theme()
	f := a.current()
	switch {
	case !a.loaded:
		ui.Column(c).Fill().Center().Children(func() { ui.Spinner(c) })
		return
	case a.loadErr != nil && len(a.files) == 0:
		message(c, iconWarning, "Couldn't load the changes", a.loadErr.Error(), func() {
			if ui.Button(c, "Try Again").Clicked() {
				a.refresh()
			}
		})
		return
	case len(a.files) == 0:
		title := "No changes"
		if g, ok := a.src.(*changes.Git); ok && g.Uncommitted() {
			title = "Nothing to review"
		}
		detail := a.src.Subtitle() + ": no files changed."
		if a.src.WatchDir() != "" {
			detail += " Osprey refreshes as files change."
		}
		message(c, iconCheckDone, title, detail, nil)
		return
	case f == nil:
		reviewed, _, _ := a.counts()
		if a.reviewScope == scopeToReview && reviewed == len(a.files) {
			message(c, iconCheckDone, "You're all caught up.", "Every changed file has been reviewed. New edits will appear here automatically.", func() {
				if ui.Button(c, "Show All Files").Clicked() {
					a.showAll()
				}
			})
		} else {
			message(c, iconSearch, "No matching files", "Try another name, or clear the filter.", func() {
				if ui.Button(c, "Clear Filter").Clicked() {
					a.showAll()
				}
			})
		}
		return
	}
	switch {
	case !f.hasText():
		message(c, iconFile, f.Note, "Osprey shows text files only.", nil)
	case a.page == pageFile:
		editor.View(c.Key(f), a.fullFor(f, t)).Grow(1).FillWidth()
	default:
		d := a.diffFor(f, t)
		if len(d.Hunks()) == 0 {
			detail := "The text of this file is the same."
			if f.OldPath != "" {
				detail = "Renamed from " + f.OldPath + ", with the same text."
			}
			message(c, iconFile, "No changes in the text", detail, nil)
			return
		}
		editor.DiffView(c.Key(f), d).Grow(1).FillWidth()
	}
}
