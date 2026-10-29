package main

import (
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func TestShortcutsPanel(t *testing.T) {
	for _, size := range [][2]int{{1280, 820}, {720, 420}} {
		a := reviewFixture()
		h := ui.NewTester(a.view, size[0], size[1])
		a.toggleShortcuts()
		h.Frame()
		r, ok := h.Find("Keyboard shortcuts")
		if !ok || r.X < 0 || r.Y < 0 || r.X+r.W > float32(size[0]) || r.Y+r.H > float32(size[1]) {
			t.Fatalf("%dx%d: the panel doesn't fit the window: %+v", size[0], size[1], r)
		}
		if !h.HasText("Mark reviewed, go to the next") || !h.HasText(keyLabel(keyReviewed)) {
			t.Fatalf("%dx%d: the panel doesn't list the review's keys", size[0], size[1])
		}
		h.Key(0, ui.KeyEscape)
		h.Frame()
		if a.shortcutsOpen || h.HasText("Keyboard Shortcuts") {
			t.Fatalf("%dx%d: Escape didn't close the panel", size[0], size[1])
		}
	}
}

// TestShortcutsListTheMenu checks that the panel lists every key of the
// menu, so that they can't drift apart.
func TestShortcutsListTheMenu(t *testing.T) {
	listed := map[string]bool{}
	for _, g := range shortcuts {
		for _, s := range g.keys {
			for _, key := range s.keys {
				listed[key] = true
			}
		}
	}
	checked := 0
	var walk func(items []*mygo.MenuItem)
	walk = func(items []*mygo.MenuItem) {
		for _, it := range items {
			if it.Accelerator != "" && it.Role == "" {
				checked++
				if !listed[it.Accelerator] {
					t.Errorf("%s (%s) isn't in the shortcuts panel", it.Label, it.Accelerator)
				}
			}
			walk(it.Submenu)
		}
	}
	walk(newApp().menu().Items())
	if checked != len(listed) {
		t.Fatalf("the menu binds %d keys, and the panel lists %d", checked, len(listed))
	}
}

func TestFindInFile(t *testing.T) {
	for _, page := range []int{pageDiff, pageFile} {
		a := reviewFixture()
		a.setBrowseMode(modeSingle)
		a.page = page
		h := ui.NewTester(a.view, 1280, 820)
		if _, open := h.Find("Find"); open {
			t.Fatal("the find bar is open already")
		}
		a.findInFile()
		h.Frame()
		if _, open := h.Find("Find"); !open {
			t.Errorf("page %d: ⌘F didn't open the find bar", page)
		}
	}
}
