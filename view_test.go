package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/overcat/osprey/internal/changes"
	"github.com/overcat/osprey/internal/review"
)

func reviewFixture() *app {
	a := newApp()
	a.src = &changes.Git{Root: "/workspace/osprey"}
	a.reviews = review.New()
	for _, path := range []string{"src/review.go", "src/source.go", "README.md"} {
		f := newFile(changes.Change{Path: path, Status: changes.StatusModified, Old: "package review\n\nconst ready = false\n", New: "package review\n\nconst ready = true\n"})
		f.added, f.deleted = f.diffView(modeSingle).Stats()
		a.files = append(a.files, f)
	}
	a.loaded, a.branch = true, "feature/review"
	a.syncSelection()
	return a
}

func click(t *testing.T, h *ui.Tester, label string) {
	t.Helper()
	h.Advance(500 * time.Millisecond)
	if err := h.Click(label); err != nil {
		t.Fatal(err)
	}
}

func TestReviewWorkflow(t *testing.T) {
	a := reviewFixture()
	a.setBrowseMode(modeSingle)
	h := ui.NewTester(a.view, 1280, 820)
	click(t, h, "To Review")
	// ⌘↩ marks the file shown, and goes on to the next one to review.
	for i, f := range a.files {
		if a.current() != f {
			t.Fatalf("review %d: selected %q, want %q", i, a.selected, f.Path)
		}
		a.toggleReviewed()
		h.Frame()
		if !a.reviewed(f) {
			t.Fatalf("%s was not marked reviewed", f.Path)
		}
		if len(a.visible()) != 2-i {
			t.Fatal("reviewed file stayed in pending list")
		}
	}
	if a.current() != nil || !h.HasText("You're all caught up.") {
		t.Fatal("missing completion state")
	}
	click(t, h, "Show All Files")
	if a.reviewScope != scopeAll || len(a.visible()) != 3 {
		t.Fatal("could not return to all files")
	}
	// The check box unmarks the file shown, where it is.
	f := a.current()
	click(t, h, "Reviewed "+f.Path)
	if a.reviewed(f) || a.current() != f {
		t.Fatal("could not undo review in place")
	}

	click(t, h, "Filter files")
	h.Type("source")
	if a.selected != "src/source.go" {
		t.Fatalf("search left an unrelated file selected: %s", a.selected)
	}
	h.Type("-missing")
	if a.current() != nil || !h.HasText("No matching files") {
		t.Fatal("search should show the empty state")
	}
	click(t, h, "Clear Filter")
	if a.filter != "" || a.current() == nil {
		t.Fatal("clearing filters didn't restore selection")
	}
}

func TestReviewScopeAfterReload(t *testing.T) {
	a := reviewFixture()
	for _, f := range a.files {
		a.setReviewed(f, true)
	}
	a.setReviewScope(scopeToReview)
	if a.current() != nil {
		t.Fatal("completed scope should have no selection")
	}
	changed := newFile(changes.Change{Path: "src/source.go", Status: changes.StatusModified, Old: "before\n", New: "after\n"})
	a.apply([]*file{a.files[0], changed, a.files[2]}, a.branch, nil)
	if a.current() == nil || a.selected != changed.Path || len(a.visible()) != 1 {
		t.Fatal("new edits did not reappear in the pending review list")
	}
	a.filter = "README"
	a.syncSelection()
	if a.current() != nil {
		t.Fatal("search and review scope should both apply")
	}
	a.setReviewScope(scopeAll)
	if a.selected != "README.md" {
		t.Fatal("all files scope should preserve the search")
	}
}

func TestCompactReviewControls(t *testing.T) {
	for _, size := range [][2]int{{720, 420}, {800, 600}, {1280, 820}} {
		t.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(t *testing.T) {
			a := reviewFixture()
			a.setBrowseMode(modeSingle)
			h := ui.NewTester(a.view, size[0], size[1])
			for _, dark := range []bool{false, true} {
				h.SetDark(dark)
				for _, label := range []string{"Reviewed " + a.selected, "Full File", "Next change", "Wrap lines"} {
					r, ok := h.Find(label)
					if !ok || r.X < a.sidebarWidth || r.X+r.W > float32(size[0]) || r.Y+r.H > float32(size[1]) {
						t.Fatalf("%q outside the detail pane: %+v", label, r)
					}
				}
				click(t, h, "Full File")
				if a.page != pageFile {
					t.Fatal("full-file control did not switch views")
				}
				click(t, h, "Diff")
				click(t, h, "Show unchanged lines")
				if a.current().diffView(modeSingle).ExpandUnchanged != a.expand {
					t.Fatal("context expansion did not reach the diff")
				}
				click(t, h, "Wrap lines")
				if a.current().diffView(modeSingle).Wrap != a.wrap {
					t.Fatal("wrap control did not reach the diff")
				}
			}
		})
	}
}

// Off macOS the window shows no menu bar: a button opens the menus, left of
// the window's buttons, on the welcome page too, however narrow the window,
// and no control of the toolbar goes under the window's buttons.
func TestMenuButton(t *testing.T) {
	defer func(shown bool) { menuInWindow = shown }(menuInWindow)
	bar := ui.TitleBar{Height: headerHeight, Right: 138} // Windows' three buttons
	for _, shown := range []bool{true, false} {
		menuInWindow = shown
		for _, size := range [][2]int{{1280, 820}, {720, 420}} {
			for name, a := range map[string]*app{"welcome page": newApp(), "files": reviewFixture()} {
				h := ui.NewTester(a.view, size[0], size[1])
				h.SetTitleBar(bar)
				r, ok := h.Find("Menu")
				if ok != shown {
					t.Fatalf("%s, %dx%d: a menu button is %v, want %v", name, size[0], size[1], ok, shown)
				}
				if ok && r.Y+r.H > headerHeight {
					t.Errorf("%s, %dx%d: the menu button at %+v is not in the title bar", name, size[0], size[1], r)
				}
				for _, label := range []string{"Menu", "Open Repository", "Refresh", "Single File", "Next change", "Wrap lines"} {
					if r, ok := h.Find(label); ok && r.Y < headerHeight && r.X+r.W > float32(size[0])-bar.Right {
						t.Errorf("%s, %dx%d: %q at %+v goes under the window's buttons", name, size[0], size[1], label, r)
					}
				}
				if shown {
					click(t, h, "Menu") // with no window to open the menus over, nothing
				}
			}
		}
	}
}

func TestWelcomeCommands(t *testing.T) {
	for _, size := range [][2]int{{1280, 820}, {720, 420}} {
		h := ui.NewTester(newApp().view, size[0], size[1])
		for _, cmd := range commands {
			if !h.HasText(cmd.line) || !h.HasText(cmd.does) {
				t.Fatalf("%dx%d: the welcome page doesn't show %q", size[0], size[1], cmd.line)
			}
		}
		r, ok := h.Find("Commands")
		if !ok || r.Y < 0 || r.Y+r.H > float32(size[1]) || r.X < 0 || r.X+r.W > float32(size[0]) {
			t.Fatalf("%dx%d: commands outside the window: %+v", size[0], size[1], r)
		}
	}
}
