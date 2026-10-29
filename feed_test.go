package main

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/overcat/osprey/internal/changes"
)

func TestContinuousNavigation(t *testing.T) {
	a := reviewFixture()
	h := ui.NewTester(a.view, 1280, 820)
	if !h.HasText("File header src/review.go") || !h.HasText("File header src/source.go") {
		t.Fatal("continuous mode should show more than one file")
	}
	click(t, h, "README.md")
	if a.selected != "README.md" {
		t.Fatalf("sidebar navigation selected %s", a.selected)
	}
	list, _ := h.Find("Continuous changes")
	header, _ := h.Find("File header README.md")
	if math.Abs(float64(header.Y-list.Y)) > 1 {
		t.Fatalf("sidebar did not align the file header: %+v / %+v", header, list)
	}
	// Scroll over the editor, not the outer scrollbar: the whole document moves.
	h.Scroll(list.X+100, list.Y+150, 0, -200)
	h.Frame()
	if a.selected == "README.md" {
		t.Fatal("sidebar did not follow scrolling")
	}
	click(t, h, "src/review.go")
	click(t, h, "Collapse src/review.go")
	if !a.files[0].folded {
		t.Fatal("collapse button did not fold the file")
	}
	click(t, h, "Expand src/review.go")
	if a.files[0].folded {
		t.Fatal("expand button did not unfold the file")
	}
}

func TestContinuousReviewKeepsPlace(t *testing.T) {
	a := reviewFixture()
	h := ui.NewTester(a.view, 1280, 820)
	click(t, h, "To Review")
	before, _ := h.Find("File header src/source.go")
	click(t, h, "Reviewed src/review.go")
	after, _ := h.Find("File header src/source.go")
	if before != after || len(a.visible()) != 3 || !a.reviewed(a.files[0]) {
		t.Fatalf("marking a file moved the feed: %+v -> %+v", before, after)
	}
	click(t, h, "src/source.go")
	a.toggleReviewed() // the existing menu command still advances
	h.Frame()
	if a.selected != "README.md" || !a.reviewed(a.files[1]) {
		t.Fatal("mark-and-next did not advance")
	}
	click(t, h, "Reviewed README.md")
	if !h.HasText("All files reviewed") {
		t.Fatal("missing end-of-review feedback")
	}
	click(t, h, "Hide Reviewed")
	if a.current() != nil || !h.HasText("You're all caught up.") {
		t.Fatal("explicit cleanup did not finish pending review")
	}
}

func TestContinuousFocusRestoresPosition(t *testing.T) {
	a := reviewFixture()
	long := newFile(changes.Change{Path: a.files[0].Path, Status: changes.StatusAdded, New: strings.Repeat("line of code\n", 100)})
	a.files[0] = long
	h := ui.NewTester(a.view, 1280, 820)
	list, _ := h.Find("Continuous changes")
	h.Scroll(list.X+100, list.Y+150, 0, 300)
	h.Frame()
	// The editor of the long file's text moves with the feed.
	before, ok := h.Find("New text")
	if !ok {
		t.Fatal("no editor in the feed")
	}
	header, _ := h.Find("File header " + long.Path)
	if math.Abs(float64(header.Y-list.Y)) > 1 {
		t.Fatal("file heading did not stay pinned")
	}
	click(t, h, "Focus on "+long.Path)
	if a.browseMode != modeSingle {
		t.Fatal("focus action did not enter single-file mode")
	}
	click(t, h, "Continuous")
	after, ok := h.Find("New text")
	if !ok {
		t.Fatal("no editor in the feed after coming back")
	}
	if math.Abs(float64(before.Y-after.Y)) > 1 {
		t.Fatalf("focus mode lost the feed position: %v -> %v", before, after)
	}
	if v := a.files[0].views; v[modeSingle].diff == v[modeContinuous].diff {
		t.Fatal("reading modes must keep independent editor positions")
	}
}

func TestContinuousHunkNavigation(t *testing.T) {
	for _, layout := range []int{layoutSplit, layoutUnified} {
		a := reviewFixture()
		a.layout = layout
		// Separated edits, including a pure deletion in the last hunk.
		old := "old\n" + strings.Repeat("context\n", 80) + "deleted\n"
		new := "new\n" + strings.Repeat("context\n", 80)
		a.files[0] = newFile(changes.Change{Path: a.files[0].Path, Status: changes.StatusModified, Old: old, New: new})
		a.expand = true
		h := ui.NewTester(a.view, 1280, 820)
		a.stepHunk(1)
		h.Frame()
		a.stepHunk(1)
		h.Frame()
		anchor, ok := h.Find("Change anchor " + a.selected)
		list, _ := h.Find("Continuous changes")
		if !ok || anchor.Y < list.Y-1 || anchor.Y+anchor.H > list.Y+list.H+1 {
			t.Fatalf("layout %d: hunk navigation did not reveal the change: %+v, list %+v", layout, anchor, list)
		}
		a.stepHunk(-1)
		h.Frame()
		anchor, ok = h.Find("Change anchor " + a.selected)
		if !ok || anchor.Y+anchor.H < list.Y+fileHeadingHeight || anchor.Y > list.Y+list.H {
			t.Fatal("previous change was not revealed below the pinned header")
		}
	}
}

func TestContinuousCompactControls(t *testing.T) {
	a := reviewFixture()
	h := ui.NewTester(a.view, 720, 420)
	for _, label := range []string{"Reviewed src/review.go", "Focus on src/review.go", "Collapse src/review.go", "Single File", "Next change"} {
		r, ok := h.Find(label)
		if !ok || r.X < 0 || r.X+r.W > 720 || r.Y < 0 || r.Y+r.H > 420 {
			t.Fatalf("%s outside window: %+v", label, r)
		}
	}
}

func TestContinuousEditorFocus(t *testing.T) {
	a := reviewFixture()
	h := ui.NewTester(a.view, 1280, 820)
	second, _ := h.Find("File header src/source.go")
	h.ClickAt(second.X+140, second.Y+fileHeadingHeight+55)
	h.Frame()
	if a.selected != "src/source.go" {
		t.Fatalf("clicking the second editor left %s active", a.selected)
	}
	a.toggleReviewed()
	h.Frame()
	if !a.reviewed(a.files[1]) || a.reviewed(a.files[0]) {
		t.Fatal("keyboard review marked a different file from the focused editor")
	}
}

func TestContinuousRefreshKeepsAnchor(t *testing.T) {
	a := reviewFixture()
	h := ui.NewTester(a.view, 1280, 820)
	click(t, h, "src/source.go")
	before, _ := h.Find("File header src/source.go")
	inserted := newFile(changes.Change{Path: "src/added.go", Status: changes.StatusAdded, New: "new file\n"})
	changed := newFile(changes.Change{Path: a.files[0].Path, Status: changes.StatusAdded, New: strings.Repeat("new code\n", 100)})
	a.apply([]*file{inserted, changed, a.files[1], a.files[2]}, a.branch, nil)
	h.Frame()
	after, _ := h.Find("File header src/source.go")
	if a.selected != "src/source.go" || math.Abs(float64(after.Y-before.Y)) > 1 {
		t.Fatalf("refresh moved the reading anchor: %s, %v -> %v", a.selected, before, after)
	}
}

func TestContinuousVirtualizesFiles(t *testing.T) {
	a := reviewFixture()
	for i := range 300 {
		a.files = append(a.files, newFile(changes.Change{Path: fmt.Sprintf("file-%03d.go", i), Status: changes.StatusAdded, New: "package app\n"}))
	}
	h := ui.NewTester(a.view, 1280, 820)
	built := 0
	for _, f := range a.files {
		if f.views[modeContinuous].diff != nil {
			built++
		}
	}
	// The initial layout estimates short rows before measuring them, so
	// allow its overscan while ensuring the other hundreds stay unbuilt.
	if built > 20 {
		t.Fatalf("built %d editors for one viewport", built)
	}
	a.show(a.files[len(a.files)-1])
	h.Frame()
	if a.selected != "file-299.go" || !h.HasText("File header file-299.go") {
		t.Fatal("could not navigate to an unbuilt file")
	}
}
