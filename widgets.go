package main

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"github.com/overcat/osprey/internal/changes"
)

// Small parts the views share, drawn as macOS draws its own.

// statusSquare shows how a file changed: its letter on the system's color.
func statusSquare(c *ui.Context, s changes.Status) {
	ui.Box(c).Size(16, 16).Radius(4).Background(colors(c.Theme()).of(s)).Center().Shrink(0).Tooltip(s.String()).Children(func() {
		ui.Text(c, string(s)).FontSize(10).Bold().TextColor(ui.Hex("#FFFFFF"))
	})
}

// lineStats shows the lines added and deleted; onAccent shows them on the
// accent color of a row chosen.
func lineStats(c *ui.Context, added, deleted int, size float32, onAccent bool) {
	t := c.Theme()
	p := colors(t)
	plus, minus := p.addedText, p.deletedText
	if onAccent {
		plus, minus = t.AccentText.Alpha(0.85), t.AccentText.Alpha(0.85)
	}
	ui.Row(c).Gap(5).Shrink(0).FontSize(size).FontFeatures("tnum").Children(func() {
		if added > 0 {
			ui.Text(c, fmt.Sprintf("+%d", added)).TextColor(plus)
		}
		if deleted > 0 {
			ui.Text(c, fmt.Sprintf("−%d", deleted)).TextColor(minus)
		}
	})
}

// keyHint shows a key, as a hint of what it does.
func keyHint(c *ui.Context, key string) {
	t := c.Theme()
	ui.Text(c, keyLabel(key)).FontSize(11).TextColor(t.TextMuted).Padding(2, 5).Radius(4).Background(fieldFill(t)).Shrink(0)
}

// iconButton is a toolbar's button showing an icon: label names what it
// does, and its tooltip shows key too, when not "".
func iconButton(c *ui.Context, icon *ui.SVG, label, key string) ui.Element {
	t := c.Theme()
	b := ui.ButtonBase(c.Key(icon)).Size(28, 26).Center().Radius(6).TextColor(t.TextMuted).Label(label).Tooltip(withKey(label, key)).Shrink(0)
	switch {
	case b.Pressed():
		b.Background(t.SurfacePressed).TextColor(t.Text)
	case b.Hovered():
		b.Background(t.SurfaceHover).TextColor(t.Text)
	}
	b.Children(func() { ui.Icon(c, icon).FontSize(15) })
	return b
}

// toggleIcon is an icon button that stays pressed while on.
func toggleIcon(c *ui.Context, icon *ui.SVG, label, key string, on bool) ui.Element {
	b := iconButton(c, icon, label, key)
	if on {
		b.Background(c.Theme().SurfacePressed).TextColor(c.Theme().Text)
	}
	return b
}

// segmented is a segmented control as AppKit draws it: the segment chosen
// raised in a sunken track, each as wide as its label.
func segmented(c *ui.Context, key string, chosen *int, labels ...string) ui.Element {
	return segmentedControl(c, key, chosen, false, labels)
}

// segmentedFill is a segmented control as wide as it is given, its
// segments sharing the width, as a sidebar's filter.
func segmentedFill(c *ui.Context, key string, chosen *int, labels ...string) ui.Element {
	return segmentedControl(c, key, chosen, true, labels)
}

func segmentedControl(c *ui.Context, key string, chosen *int, fill bool, labels []string) ui.Element {
	t := c.Theme()
	seg := ui.SegmentedBase(c.Key(key), chosen, len(labels))
	seg.Track.Padding(1).Radius(7).Background(fieldFill(t)).Shrink(0).Children(func() {
		for i, label := range labels {
			s := seg.Segment(i).Padding(3, 10).Radius(6).Label(label)
			if fill {
				s.Grow(1).Basis(0).Center()
			}
			if i == *chosen {
				if t.Dark {
					s.Background(ui.RGBA(255, 255, 255, 0.2))
				} else {
					s.Background(ui.Hex("#FFFFFF")).Shadow(0, 0, 0, 0.5, ui.RGBA(0, 0, 0, 0.12)).Shadow(0, 1, 2, 0, ui.RGBA(0, 0, 0, 0.12))
				}
			}
			s.Children(func() { ui.Text(c, label).FontSize(12).SingleLine() })
		}
	})
	return seg.Track
}

// toolbarSeparator divides a toolbar's groups.
func toolbarSeparator(c *ui.Context) {
	ui.Box(c).Size(1, 18).Margin(0, 4).Background(c.Theme().Border).Shrink(0)
}

// message fills the pane with a short message, and what more builds
// under it.
func message(c *ui.Context, icon *ui.SVG, title, detail string, more func()) {
	t := c.Theme()
	ui.Column(c).Fill().Center().Gap(6).Padding(32).Children(func() {
		ui.Icon(c, icon).FontSize(36).TextColor(t.TextMuted.Alpha(0.7)).Margin(0, 0, 10, 0)
		ui.Text(c, title).FontSize(17).FontWeight(600).TextAlign(ui.Center)
		ui.Text(c, detail).FontSize(13).LineHeight(1.5).TextColor(t.TextMuted).TextAlign(ui.Center).MaxWidth(360).Selectable()
		if more != nil {
			ui.Row(c).Margin(12, 0, 0, 0).Gap(8).Children(more)
		}
	})
}
