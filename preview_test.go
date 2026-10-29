package main

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/overcat/osprey/internal/changes"
	"github.com/overcat/osprey/internal/review"
)

// TestPreview renders the real UI without opening a window. Set
// OSPREY_PREVIEW_DIR to write light, dark and compact previews.
func TestPreview(t *testing.T) {
	out := os.Getenv("OSPREY_PREVIEW_DIR")
	if out == "" {
		t.Skip("set OSPREY_PREVIEW_DIR to render previews")
	}
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	a := newApp()
	a.src = &changes.Git{Root: root}
	a.reviews = review.New()
	files, err := (&loader{src: a.src}).load()
	if err != nil {
		t.Fatal(err)
	}
	a.apply(files, "main", nil)
	a.selected = "view.go"
	a.show(a.current())
	h := ui.NewTester(a.view, 1280, 820)
	h.SetTitleBar(ui.TitleBar{Left: 80, Height: headerHeight})
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	save := func(name string) {
		f, err := os.Create(filepath.Join(out, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, h.Image()); err != nil {
			t.Fatal(err)
		}
	}
	save("light")
	h.SetDark(true)
	save("dark")
	h.SetDark(false)
	h.SetSize(800, 600)
	save("compact")
	h.SetSize(720, 420)
	save("minimum")
	h.SetSize(1280, 820)
	a.layout = layoutUnified
	h.Frame()
	save("unified")
	a.page = pageFile
	h.Frame()
	save("full-file")
	for _, f := range a.files {
		a.setReviewed(f, true)
	}
	a.setReviewScope(scopeToReview)
	h.Frame()
	save("complete")
	a.filter = "no-such-file"
	a.setReviewScope(scopeAll)
	h.Frame()
	save("no-matches")
	a.src = nil
	h.Frame()
	save("welcome")
	a = reviewFixture()
	h = ui.NewTester(a.view, 1280, 820)
	h.SetTitleBar(ui.TitleBar{Left: 80, Height: headerHeight})
	save("continuous")
	h.SetDark(true)
	save("continuous-dark")
	h.SetDark(false)
	h.SetSize(720, 420)
	save("continuous-compact")
}
