package main

import "github.com/egoist/mygo/ui"

// shortcutsPanel lists the keyboard shortcuts, by what they are for.
func shortcutsPanel(c *ui.Context) {
	t := c.Theme()
	_, height := c.Size()
	ui.Column(c).Width(320).MaxHeight(height - 48).Padding(4).Gap(16).Label("Keyboard shortcuts").Children(func() {
		ui.Text(c, "Keyboard Shortcuts").FontSize(15).FontWeight(600)
		ui.Scroll(c).Shrink(1).MinHeight(0).Children(func() {
			ui.Column(c).Gap(16).Children(func() {
				for _, g := range shortcuts {
					ui.Column(c).Gap(5).Children(func() {
						ui.Text(c, g.group).FontSize(11).FontWeight(600).TextColor(t.TextMuted)
						for _, s := range g.keys {
							ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
								ui.Text(c, s.label).FontSize(12.5).SingleLine().Grow(1).Shrink(1)
								for _, key := range s.keys {
									keyHint(c, key)
								}
							})
						}
					})
				}
			})
		})
	})
}
