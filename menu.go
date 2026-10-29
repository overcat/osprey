package main

import (
	"runtime"

	"github.com/egoist/mygo"
)

// menu is the menu bar, whose keys go to Osprey before the diff's
// editors.
func (a *app) menu() *mygo.Menu {
	do := func(fn func()) func(*mygo.MenuItem, *mygo.Window) {
		return func(*mygo.MenuItem, *mygo.Window) {
			fn()
			if a.win != nil {
				a.win.Invalidate()
			}
		}
	}
	item := func(label, key string, fn func()) *mygo.MenuItem {
		return &mygo.MenuItem{Label: label, Accelerator: key, Click: do(fn)}
	}
	return mygo.NewMenu([]*mygo.MenuItem{
		{Role: mygo.RoleAppMenu},
		{Label: "File", Submenu: []*mygo.MenuItem{
			item("Open Repository…", keyOpen, func() { go a.chooseRepo() }),
			item("Refresh", keyRefresh, a.refresh),
			mygo.Separator(),
			item("Install Command Line Tool…", "", func() { go a.installFromMenu() }),
			mygo.Separator(),
			{Role: mygo.RoleClose},
		}},
		{Role: mygo.RoleEditMenu},
		{Label: "View", Submenu: []*mygo.MenuItem{
			item("Continuous / Single File", "", func() { a.setBrowseMode(flip(a.browseMode)) }),
			item("Show or Hide Sidebar", keySidebar, func() { a.sidebarHidden = !a.sidebarHidden }),
			mygo.Separator(),
			item("Diff", "", func() { a.page = pageDiff }),
			item("Whole File", "", func() { a.page = pageFile }),
			mygo.Separator(),
			item("Side by Side", "", func() { a.layout = layoutSplit }),
			item("Unified", "", func() { a.layout = layoutUnified }),
			mygo.Separator(),
			item("Soft Wrap", "", func() { a.wrap = !a.wrap }),
			item("Files as Tree or List", "", func() { a.flat = !a.flat; a.reveal = true }),
			item("Show Unchanged Lines", "", func() { a.expand = !a.expand }),
			mygo.Separator(),
			{Role: mygo.RoleToggleFullScreen},
		}},
		{Label: "Go", Submenu: []*mygo.MenuItem{
			item("Find File…", keyFind, func() { a.focus = "filter" }),
			item("Find in File…", keyFindInFile, a.findInFile),
			item("History", "", a.showHistory),
			mygo.Separator(),
			item("Next File", keyNextFile, func() { a.step(1, false) }),
			item("Previous File", keyPreviousFile, func() { a.step(-1, false) }),
			item("Next File to Review", "", func() { a.step(1, true) }),
			mygo.Separator(),
			item("Next Change", keyNextChange, func() { a.stepHunk(1) }),
			item("Previous Change", keyPreviousChange, func() { a.stepHunk(-1) }),
		}},
		{Label: "Review", Submenu: []*mygo.MenuItem{
			item("Mark Reviewed and Go to Next", keyReviewed, a.toggleReviewed),
			item("Show Only Files to Review", "", func() { a.setReviewScope(flip(a.reviewScope)) }),
		}},
		{Role: mygo.RoleWindowMenu},
		a.helpMenu(item),
	})
}

// helpMenu is the Help menu, to which macOS adds its search of the menus:
// type what a command does, and it shows where the command is, with its
// key.
func (a *app) helpMenu(item func(label, key string, fn func()) *mygo.MenuItem) *mygo.MenuItem {
	help := &mygo.MenuItem{Label: "Help", Submenu: []*mygo.MenuItem{
		item("Keyboard Shortcuts", keyShortcuts, a.toggleShortcuts),
	}}
	if runtime.GOOS == "darwin" {
		help.Role = mygo.RoleHelp // hidden elsewhere, where the menu is plain
	}
	return help
}

// installCommands installs the command line tool, as osp install-cli
// does, and keeps what it said for the welcome page. It blocks, on a
// goroutine of its own.
func (a *app) installCommands() (string, error) {
	said, err := installCLI("")
	a.win.Update(func() { a.cli = cliState{missing: err != nil, said: said, err: err} })
	return said, err
}

// installFromMenu installs the command line tool, and says where.
func (a *app) installFromMenu() {
	said, err := a.installCommands()
	if err != nil {
		mygo.Dialog.Error("Couldn't install the command line tool", err.Error())
		return
	}
	mygo.Dialog.Message(mygo.MessageOptions{
		Parent:  a.win,
		Message: "Installed osprey and osp",
		Detail:  said + " Run osp in a repository to review its changes.",
	})
}
