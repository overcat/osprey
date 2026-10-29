// Osprey shows the changes of a Git repository, file by file, for a
// person to review what an agent wrote:
//
//	osp              changes not committed yet
//	osp main         what this branch changed since it left main
//	osp a b          two branches, files or folders
package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/overcat/osprey/internal/changes"
)

func main() {
	runFromBundle()
	attachConsole()
	args := os.Args[1:]
	if len(args) > 0 {
		switch args[0] {
		case "install-cli":
			dir := ""
			if len(args) > 1 {
				dir = args[1]
			}
			said, err := installCLI(dir)
			if err != nil {
				fail(err)
			}
			fmt.Println(said)
			return
		case "completion":
			if len(args) < 2 {
				failUsage(errors.New("which shell: osp completion fish, zsh or bash\n" + completionGuide))
			}
			if err := completion(os.Stdout, args[1]); err != nil {
				failUsage(err)
			}
			return
		}
	}
	o, err := parseArgs(args)
	switch {
	case errors.Is(err, errHelp):
		fmt.Print(usage)
		return
	case errors.Is(err, errVersion):
		fmt.Println("osprey", version)
		return
	case err != nil:
		failUsage(err)
	}
	src, err := changes.Open(o.dir, o.args, o.committed)
	var usageErr changes.UsageError
	switch {
	case errors.As(err, &usageErr):
		failUsage(err)
	case errors.Is(err, changes.ErrNoRepository):
		// The window opens to choose one; a terminal is told why.
		if typedInTerminal() {
			fmt.Fprintln(os.Stderr, "osprey: not in a Git repository: choose one in the window")
		}
	case err != nil:
		fail(err)
	}
	if !o.wait && detach(args) {
		return
	}
	releaseConsole()

	a := newApp()
	mygo.App.SetName("Osprey")
	mygo.App.WhenReady(func() {
		setDockIcon()
		a.appMenu = a.menu()
		mygo.App.SetMenu(a.appMenu)
		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title:                "Osprey",
			Width:                1280,
			Height:               820,
			MinWidth:             720,
			MinHeight:            420,
			StateKey:             "main",
			TitleBarStyle:        mygo.TitleBarHidden,
			TrafficLightPosition: &mygo.Point{X: 18, Y: (headerHeight - 14) / 2},
			TitleBarHeight:       headerHeight,
			Vibrancy:             sidebarMaterial(),
			Content:              ui.View(a.view),
		})
		a.win.OnClosed(mygo.App.Quit)
		if src != nil {
			a.open(src)
		}
		a.snapshot()
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

// sidebarMaterial is what shows under the sidebar: macOS's own material.
// Elsewhere the sidebar draws a color: Windows 11's materials would take
// the window's menu bar away.
func sidebarMaterial() mygo.Vibrancy {
	if runtime.GOOS == "darwin" {
		return mygo.VibrancySidebar
	}
	return mygo.VibrancyNone
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "osprey:", err)
	os.Exit(1)
}

// failUsage fails for a command line written wrong, as the help says to
// write it, with exit code 2, as Go's own commands do.
func failUsage(err error) {
	fmt.Fprintf(os.Stderr, "osprey: %v\nRun osp --help for how to write it.\n", err)
	os.Exit(2)
}
