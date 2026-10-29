package main

import "github.com/egoist/mygo/ui"

// commands are the command lines the welcome page teaches, run in a
// repository's folder.
var commands = []struct{ line, does string }{
	{"osp", "Changes not committed yet"},
	{"osp main", "This branch since main, uncommitted too"},
	{"osp --committed", "This branch's commits since main"},
	{"osp src", "Only the changes in src"},
	{"osp HEAD~3", "The last 3 commits, uncommitted too"},
	{"osp old new", "Two files, or two folders"},
}

func (a *app) welcome(c *ui.Context) {
	t := c.Theme()
	height := contentHeight(c)
	roomy := height >= 700 // room for the picture and the keys
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		// The room of macOS's window controls, as the toolbar's; the
		// window's own title bar holds them elsewhere.
		top := float32(0)
		if !ownTitleBar {
			top = headerHeight
			ui.Row(c).Height(top).Shrink(0).FillWidth().DragWindow()
		}
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).FillWidth().MinHeight(height-top).Center().Gap(6).Padding(0, 32, 32, 32).Children(func() {
				if roomy {
					ui.Image(c, brandImage).Size(96, 96).Margin(0, 0, 12, 0)
				}
				ui.Text(c, "Welcome to Osprey").FontSize(26).FontWeight(600).TextAlign(ui.Center)
				ui.Text(c, "Review the changes of a repository, file by file.").FontSize(14).TextColor(t.TextMuted).TextAlign(ui.Center)
				if ui.PrimaryButton(c, "").Margin(18, 0, 4, 0).Children(func() {
					ui.Icon(c, iconFolderOpen)
					ui.Text(c, "Open Repository…").SingleLine()
				}).Clicked() {
					go a.chooseRepo()
				}
				a.commandCard(c)
				if roomy {
					ui.Row(c).Margin(24, 0, 0, 0).Gap(24).Children(func() {
						for _, hint := range []struct{ key, label string }{{keyFind, "Find a file"}, {keyNextChange, "Next change"}, {keyReviewed, "Mark reviewed"}, {keyShortcuts, "All shortcuts"}} {
							ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
								keyHint(c, hint.key)
								ui.Text(c, hint.label).FontSize(12).TextColor(t.TextMuted)
							})
						}
					})
				}
			})
		})
	})
}

// commandCard lists the commands, to copy into a terminal.
func (a *app) commandCard(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Margin(18, 0, 0, 0).Width(500).Padding(12, 16).Gap(4).Radius(10).
		Background(t.Surface).Border(1, t.Border).Selectable().Label("Commands").Children(func() {
		ui.Text(c, "Or, in a repository's folder in a terminal:").FontSize(12).TextColor(t.TextMuted).Margin(0, 0, 6, 0)
		for _, cmd := range commands {
			ui.Row(c).Gap(16).AlignItems(ui.Center).Children(func() {
				ui.Text(c, cmd.line).Font("monospace").FontSize(12).Width(170).Shrink(0).SingleLine()
				ui.Text(c, cmd.does).FontSize(12).TextColor(t.TextMuted).SingleLine().Shrink(1)
			})
		}
	})
}
